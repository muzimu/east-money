package captcha

import (
	"image"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/up-zero/gotool/imageutil"
)

// =============================================================================
// 测试常量 — 模型及图片路径
// =============================================================================

const (
	testImagePath      = "testdata/captcha.jpeg"
	expectedResult     = "1242"
	onnxRuntimeLibPath = "/opt/homebrew/Cellar/onnxruntime/1.27.0/lib/libonnxruntime.dylib"
	modelPath          = "/Library/workspace/ddddocr/benchmark/common.onnx"
	dictPath           = "/Library/workspace/ddddocr/dict.txt"
)

// =============================================================================
// 辅助函数
// =============================================================================

// loadTestImage 加载测试用验证码图片。
func loadTestImage(tb testing.TB) image.Image {
	tb.Helper()
	img, err := imageutil.Open(testImagePath)
	require.NoError(tb, err, "加载测试图片失败，请确认 testdata/captcha.jpeg 存在")
	return img
}

// newTestRecognizer 创建测试用识别器，若模型不存在则跳过测试。
func newTestRecognizer(tb testing.TB) *DefaultRecognizer {
	tb.Helper()
	if testing.Short() {
		tb.Skip("跳过需 ONNX 模型的测试（-short 模式）")
	}
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		tb.Skipf("模型文件不存在: %s", modelPath)
	}
	if _, err := os.Stat(onnxRuntimeLibPath); os.IsNotExist(err) {
		tb.Skipf("ONNX Runtime 库不存在: %s", onnxRuntimeLibPath)
	}

	r, err := NewDefaultRecognizer(modelPath, dictPath, onnxRuntimeLibPath)
	require.NoError(tb, err, "创建识别器失败")
	return r
}

// =============================================================================
// 创建测试
// =============================================================================

func TestNewDefaultRecognizer_Success(t *testing.T) {
	r := newTestRecognizer(t)
	defer r.Close()
	assert.NotNil(t, r)
	assert.NotNil(t, r.engine, "引擎不应为 nil")
}

func TestNewDefaultRecognizer_InvalidModelPath(t *testing.T) {
	r, err := NewDefaultRecognizer("/nonexistent/model.onnx", dictPath, onnxRuntimeLibPath)
	assert.Error(t, err, "无效模型路径应返回错误")
	assert.Nil(t, r, "创建失败时识别器应为 nil")
	assert.Contains(t, err.Error(), "创建 ddddocr 引擎失败")
}

func TestNewDefaultRecognizer_InvalidDictPath(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需 ONNX 模型的测试（-short 模式）")
	}
	r, err := NewDefaultRecognizer(modelPath, "/nonexistent/dict.txt", onnxRuntimeLibPath)
	assert.Error(t, err, "无效字典路径应返回错误")
	assert.Nil(t, r)
}

func TestNewDefaultRecognizer_EmptyONNXLib(t *testing.T) {
	// 验证空路径被显式拒绝。
	r, err := NewDefaultRecognizer(modelPath, dictPath, "")
	assert.Error(t, err, "空 ONNX 库路径应被拒绝")
	assert.Nil(t, r)
	assert.Contains(t, err.Error(), "不能为空")
}

func TestNewDefaultRecognizer_InvalidONNXLibIgnored(t *testing.T) {
	// ddddocr 使用 purego 方式加载 ONNX Runtime，不验证 .dylib 文件是否存在，
	// 因此传入无效（但非空）路径时仍可正常创建引擎。
	if testing.Short() {
		t.Skip("跳过需 ONNX 模型的测试（-short 模式）")
	}
	r, err := NewDefaultRecognizer(modelPath, dictPath, "/nonexistent/libonnxruntime.dylib")
	require.NoError(t, err, "purego 模式下无效路径不影响引擎创建")
	require.NotNil(t, r)
	defer r.Close()

	// 验证可正常识别
	img := loadTestImage(t)
	result, err := r.Recognize(img)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

// =============================================================================
// OCR 识别测试
// =============================================================================

func TestRecognize_Success(t *testing.T) {
	r := newTestRecognizer(t)
	defer r.Close()

	img := loadTestImage(t)
	result, err := r.Recognize(img)

	require.NoError(t, err, "OCR 识别不应出错")
	assert.Equal(t, expectedResult, result, "识别结果应为 %s", expectedResult)
}

func TestRecognize_NilEngine(t *testing.T) {
	r := &DefaultRecognizer{engine: nil}

	img := loadTestImage(t)
	result, err := r.Recognize(img)

	assert.Error(t, err, "nil 引擎时应返回错误")
	assert.Empty(t, result, "nil 引擎时结果应为空")
	assert.Contains(t, err.Error(), "OCR 引擎未初始化")
}

func TestRecognize_AfterClose(t *testing.T) {
	r := newTestRecognizer(t)
	err := r.Close()
	require.NoError(t, err, "关闭不应出错")

	img := loadTestImage(t)
	result, err := r.Recognize(img)

	assert.Error(t, err, "关闭后识别应返回错误")
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "OCR 引擎未初始化")
}

// =============================================================================
// Close 测试
// =============================================================================

func TestClose_Success(t *testing.T) {
	r := newTestRecognizer(t)

	err := r.Close()
	assert.NoError(t, err, "首次关闭不应出错")
	assert.Nil(t, r.engine, "关闭后引擎应为 nil")
}

func TestClose_DoubleClose(t *testing.T) {
	r := newTestRecognizer(t)

	err := r.Close()
	require.NoError(t, err, "首次关闭不应出错")

	// 第二次关闭应安全（幂等）
	err = r.Close()
	assert.NoError(t, err, "重复关闭不应出错")
	assert.Nil(t, r.engine)
}

// =============================================================================
// 并发安全测试
// =============================================================================

func TestRecognize_Concurrent(t *testing.T) {
	r := newTestRecognizer(t)
	defer r.Close()

	img := loadTestImage(t)

	const n = 10
	results := make([]string, n)
	errs := make([]error, n)

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			result, e := r.Recognize(img)
			results[idx] = result
			errs[idx] = e
		}(i)
	}
	wg.Wait()

	// 所有并发调用应返回一致结果
	for i := range n {
		assert.NoError(t, errs[i], "并发调用 %d 不应出错", i)
		assert.Equal(t, expectedResult, results[i], "并发调用 %d 结果应一致", i)
	}
}

// =============================================================================
// Recognizer 接口验证
// =============================================================================

func TestDefaultRecognizer_ImplementsRecognizer(t *testing.T) {
	// 编译期验证 DefaultRecognizer 实现了 Recognizer 接口
	var _ Recognizer = (*DefaultRecognizer)(nil)

	r := newTestRecognizer(t)
	defer r.Close()

	var iface Recognizer = r
	assert.NotNil(t, iface)
}
