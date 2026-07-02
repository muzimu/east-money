package captcha

import (
	"encoding/json"
	"image"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// 辅助函数
// =============================================================================

// newOCRTestServer 创建模拟 ddddocr-fastapi 服务器。
// 返回 server URL 和接收到的文件数据指针。
func newOCRTestServer(tb testing.TB, result string, statusCode int) (*httptest.Server, *[]byte) {
	tb.Helper()
	var receivedFile []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(tb, http.MethodPost, r.Method)

		// 解析 multipart form
		err := r.ParseMultipartForm(10 << 20) // 10MB
		require.NoError(tb, err, "解析 multipart form 失败")

		file, _, err := r.FormFile("file")
		require.NoError(tb, err, "缺少 file 字段")
		receivedFile, _ = io.ReadAll(file)

		// 返回 ddddocr-fastapi 标准 JSON 响应
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OCRResponse{
			Code:    statusCode,
			Message: "success",
			Data:    result,
		})
	}))

	tb.Cleanup(srv.Close)
	return srv, &receivedFile
}

// newOCRTestServerB64 创建模拟 ddddocr-fastapi 服务器（base64 模式）。
func newOCRTestServerB64(tb testing.TB, result string, statusCode int) (*httptest.Server, *string) {
	tb.Helper()
	var receivedB64 string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(tb, http.MethodPost, r.Method)
		assert.Equal(tb, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		body, _ := io.ReadAll(r.Body)
		// 提取 image= 后的 base64 值
		receivedB64 = strings.TrimPrefix(string(body), "image=")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(OCRResponse{
			Code:    statusCode,
			Message: "success",
			Data:    result,
		})
	}))

	tb.Cleanup(srv.Close)
	return srv, &receivedB64
}

// =============================================================================
// 创建测试
// =============================================================================

func TestNewRemoteRecognizer_Default(t *testing.T) {
	r := NewRemoteRecognizer()
	require.NotNil(t, r)
	assert.Equal(t, "http://localhost:8000/ocr", r.endpoint)
	assert.NotNil(t, r.httpClient)
	assert.Equal(t, 10*time.Second, r.httpClient.Timeout)
	assert.False(t, r.useBase64, "默认应为 multipart file 模式")
}

func TestNewRemoteRecognizer_WithOptions(t *testing.T) {
	customClient := &http.Client{Timeout: 5 * time.Second}
	r := NewRemoteRecognizer(
		WithRemoteEndpoint("http://10.0.0.5:8000/ocr"),
		WithRemoteHTTPClient(customClient),
		WithRemoteTimeout(15*time.Second),
		WithRemoteBase64(),
	)
	require.NotNil(t, r)
	assert.Equal(t, "http://10.0.0.5:8000/ocr", r.endpoint)
	assert.True(t, r.useBase64)
}

// =============================================================================
// OCR 识别测试 — multipart file 模式
// =============================================================================

func TestRemoteRecognizer_Recognize_Success(t *testing.T) {
	srv, received := newOCRTestServer(t, "4829", 200)
	img := loadTestImage(t)

	r := NewRemoteRecognizer(WithRemoteEndpoint(srv.URL))
	code, err := r.Recognize(img)

	require.NoError(t, err)
	assert.Equal(t, "4829", code)
	assert.NotEmpty(t, *received, "应发送了图片数据")
	assert.True(t, len(*received) > 100, "PNG 数据应大于 100 字节")
}

func TestRemoteRecognizer_Recognize_Non200Code(t *testing.T) {
	srv, _ := newOCRTestServer(t, "", 500)
	img := loadTestImage(t)

	r := NewRemoteRecognizer(WithRemoteEndpoint(srv.URL))
	code, err := r.Recognize(img)

	assert.Error(t, err)
	assert.Empty(t, code)
	assert.Contains(t, err.Error(), "远程 OCR 返回错误")
	assert.Contains(t, err.Error(), "code=500")
}

func TestRemoteRecognizer_Recognize_ConnectionRefused(t *testing.T) {
	img := loadTestImage(t)
	r := NewRemoteRecognizer(
		WithRemoteEndpoint("http://127.0.0.1:19999/ocr"),
		WithRemoteTimeout(100*time.Millisecond),
	)
	code, err := r.Recognize(img)

	assert.Error(t, err)
	assert.Empty(t, code)
	assert.Contains(t, err.Error(), "远程 OCR 请求失败")
}

func TestRemoteRecognizer_Recognize_EmptyDataField(t *testing.T) {
	srv, _ := newOCRTestServer(t, "", 200)
	img := loadTestImage(t)

	r := NewRemoteRecognizer(WithRemoteEndpoint(srv.URL))
	code, err := r.Recognize(img)

	require.NoError(t, err)
	assert.Empty(t, code)
}

// =============================================================================
// OCR 识别测试 — base64 模式
// =============================================================================

func TestRemoteRecognizer_RecognizeB64_Success(t *testing.T) {
	srv, receivedB64 := newOCRTestServerB64(t, "5678", 200)
	img := loadTestImage(t)

	r := NewRemoteRecognizer(
		WithRemoteEndpoint(srv.URL),
		WithRemoteBase64(),
	)
	code, err := r.Recognize(img)

	require.NoError(t, err)
	assert.Equal(t, "5678", code)
	assert.NotEmpty(t, *receivedB64, "应发送了 base64 数据")
}

func TestRemoteRecognizer_RecognizeB64_Non200Code(t *testing.T) {
	srv, _ := newOCRTestServerB64(t, "", 400)
	img := loadTestImage(t)

	r := NewRemoteRecognizer(
		WithRemoteEndpoint(srv.URL),
		WithRemoteBase64(),
	)
	code, err := r.Recognize(img)

	assert.Error(t, err)
	assert.Empty(t, code)
	assert.Contains(t, err.Error(), "code=400")
}

// =============================================================================
// Close 测试
// =============================================================================

func TestRemoteRecognizer_Close(t *testing.T) {
	r := NewRemoteRecognizer()
	err := r.Close()
	assert.NoError(t, err)

	// 重复关闭应安全（幂等）
	err = r.Close()
	assert.NoError(t, err)
}

// =============================================================================
// Recognizer 接口验证
// =============================================================================

func TestRemoteRecognizer_ImplementsRecognizer(t *testing.T) {
	// 编译期验证 RemoteRecognizer 实现了 Recognizer 接口
	var _ Recognizer = (*RemoteRecognizer)(nil)

	r := NewRemoteRecognizer()
	var iface Recognizer = r
	assert.NotNil(t, iface)
}

// =============================================================================
// PNG 编码兼容性测试
// =============================================================================

func TestRemoteRecognizer_Recognize_ValidPNGInMultipart(t *testing.T) {
	// 验证发送到 ddddocr-fastapi 的 multipart file 包含有效的 PNG
	srv, received := newOCRTestServer(t, "0000", 200)
	img := loadTestImage(t)

	r := NewRemoteRecognizer(WithRemoteEndpoint(srv.URL))
	_, err := r.Recognize(img)
	require.NoError(t, err)

	// 验证 file 字段中的数据是有效的 PNG
	_, format, err := image.Decode(strings.NewReader(string(*received)))
	require.NoError(t, err)
	assert.Equal(t, "png", format)
}
