package captcha

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// OCRResponse ddddocr-fastapi 标准 JSON 响应结构。
type OCRResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// RemoteRecognizer 通过 HTTP 调用 ddddocr-fastapi 进行验证码识别。
// 适用于不希望本地部署 ONNX 模型的场景。
//
// 远程 OCR 服务协议（兼容 ddddocr-fastapi）：
//   - POST /ocr（默认），multipart/form-data 上传 file 字段
//   - 响应 JSON：{"code": 200, "message": "success", "data": "识别结果"}
//
// 使用示例：
//
//	// 默认：http://localhost:8000/ocr，multipart 文件上传
//	r := captcha.NewRemoteRecognizer()
//
//	// 自定义地址
//	r := captcha.NewRemoteRecognizer(captcha.WithRemoteEndpoint("http://10.0.0.5:8000/ocr"))
type RemoteRecognizer struct {
	endpoint   string
	httpClient *http.Client
	useBase64  bool // true 时使用 base64 image 字段代替 file 上传
}

// RemoteOption 函数式选项类型。
type RemoteOption func(*RemoteRecognizer)

// WithRemoteEndpoint 设置远程 OCR 服务的完整 URL（含路径）。
// 默认: "http://localhost:8000/ocr"。
func WithRemoteEndpoint(url string) RemoteOption {
	return func(r *RemoteRecognizer) {
		r.endpoint = url
	}
}

// WithRemoteHTTPClient 注入自定义 HTTP 客户端。
func WithRemoteHTTPClient(hc *http.Client) RemoteOption {
	return func(r *RemoteRecognizer) {
		r.httpClient = hc
	}
}

// WithRemoteTimeout 设置请求超时时间。
func WithRemoteTimeout(d time.Duration) RemoteOption {
	return func(r *RemoteRecognizer) {
		r.httpClient.Timeout = d
	}
}

// WithRemoteBase64 使用 base64 image 字段代替默认的 multipart file 上传。
// 默认使用 multipart/form-data 方式（更高效）。
func WithRemoteBase64() RemoteOption {
	return func(r *RemoteRecognizer) {
		r.useBase64 = true
	}
}

// NewRemoteRecognizer 创建适配 ddddocr-fastapi 的远程 OCR 识别器。
//
// 默认连接 http://localhost:8000/ocr，使用 multipart/form-data 文件上传。
// 通过 RemoteOption 可自定义地址、超时、传输方式。
func NewRemoteRecognizer(opts ...RemoteOption) *RemoteRecognizer {
	r := &RemoteRecognizer{
		endpoint: "http://localhost:8000/ocr",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Recognize 将验证码图片发送到 ddddocr-fastapi 进行识别。
func (r *RemoteRecognizer) Recognize(img image.Image) (string, error) {
	if r.useBase64 {
		return r.recognizeB64(img)
	}
	return r.recognizeFile(img)
}

// recognizeFile 以 multipart/form-data 文件上传方式发送图片。
func (r *RemoteRecognizer) recognizeFile(img image.Image) (string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// 创建 file 字段
	part, err := writer.CreateFormFile("file", "captcha.png")
	if err != nil {
		return "", fmt.Errorf("创建 multipart file 字段失败: %w", err)
	}

	if err := png.Encode(part, img); err != nil {
		return "", fmt.Errorf("编码验证码图片为 PNG 失败: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("关闭 multipart writer 失败: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, r.endpoint, &body)
	if err != nil {
		return "", fmt.Errorf("创建 OCR 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("远程 OCR 请求失败: %w", err)
	}
	defer resp.Body.Close()

	return r.parseResponse(resp)
}

// recognizeB64 以 base64 image 字段方式发送图片（application/x-www-form-urlencoded）。
func (r *RemoteRecognizer) recognizeB64(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("编码验证码图片为 PNG 失败: %w", err)
	}

	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	// 构造 application/x-www-form-urlencoded 请求体
	formBody := fmt.Sprintf("image=%s", b64)
	req, err := http.NewRequest(http.MethodPost, r.endpoint, bytes.NewBufferString(formBody))
	if err != nil {
		return "", fmt.Errorf("创建 OCR 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("远程 OCR 请求失败: %w", err)
	}
	defer resp.Body.Close()

	return r.parseResponse(resp)
}

// parseResponse 解析 ddddocr-fastapi 的 JSON 响应。
func (r *RemoteRecognizer) parseResponse(resp *http.Response) (string, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取远程 OCR 响应失败: %w", err)
	}

	var ocrResp OCRResponse
	if err := json.Unmarshal(body, &ocrResp); err != nil {
		return "", fmt.Errorf("解析远程 OCR JSON 响应失败 (body=%s): %w", string(body), err)
	}

	if ocrResp.Code != 200 {
		return "", fmt.Errorf("远程 OCR 返回错误 (code=%d, message=%s)", ocrResp.Code, ocrResp.Message)
	}

	return ocrResp.Data, nil
}

// Close 释放资源。远程识别器无需释放底层资源。
func (r *RemoteRecognizer) Close() error {
	return nil
}
