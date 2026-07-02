// Package captcha 提供验证码识别能力。
// 默认实现使用 ddddocr ONNX 模型进行本地 OCR 识别。
package captcha

import (
	"fmt"
	"image"
	"sync"

	ddddocr "github.com/getcharzp/go-ocr/ddddocr"
)

// Recognizer 定义验证码识别接口。
type Recognizer interface {
	// Recognize 对验证码图片进行 OCR 识别，返回识别出的文本。
	Recognize(img image.Image) (string, error)
	// Close 释放底层资源（ONNX sessions）。
	Close() error
}

// DefaultRecognizer 使用单引擎 + Mutex 的 ddddocr 验证码识别器。
// 单引擎模式适用于交易场景的低并发需求（1-5 QPS），
// 单次识别约 9.8ms，进程内存约 3MB。
type DefaultRecognizer struct {
	mu     sync.Mutex
	engine *ddddocr.Engine
}

// NewDefaultRecognizer 创建默认验证码识别器。
//
// 参数：
//   - modelPath: common.onnx 模型文件路径
//   - dictPath: dict.txt 字符集文件路径
//   - onnxRuntimeLibPath: onnxruntime 共享库路径
func NewDefaultRecognizer(modelPath, dictPath, onnxRuntimeLibPath string) (*DefaultRecognizer, error) {
	cfg := ddddocr.Config{
		ModelPath:          modelPath,
		DictPath:           dictPath,
		OnnxRuntimeLibPath: onnxRuntimeLibPath,
		UseCustomModel:     false, // 使用官方模型（beta 模式）
	}

	engine, err := ddddocr.NewEngine(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建 ddddocr 引擎失败: %w", err)
	}

	return &DefaultRecognizer{
		engine: engine,
	}, nil
}

// Recognize 对验证码图片进行 OCR 识别。
// 使用 Mutex 保护 ONNX session，因为 ONNX session 不保证线程安全。
func (r *DefaultRecognizer) Recognize(img image.Image) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.engine == nil {
		return "", fmt.Errorf("OCR 引擎未初始化")
	}

	return r.engine.Classification(img)
}

// Close 释放 ONNX 引擎资源。
func (r *DefaultRecognizer) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.engine != nil {
		r.engine.Destroy()
		r.engine = nil
	}
	return nil
}
