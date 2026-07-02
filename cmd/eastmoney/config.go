package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

// Config YAML 配置文件结构。
type Config struct {
	User     string    `yaml:"user"`
	Password string    `yaml:"password"`
	Log      string    `yaml:"log"`
	OCR      OCRConfig `yaml:"ocr"`
}

// OCRConfig OCR 相关配置。
type OCRConfig struct {
	Model   string `yaml:"model"`
	Dict    string `yaml:"dict"`
	ONNXLib string `yaml:"onnx_lib"`
}

// mergeConfig 按优先级合并配置：CLI flags > 环境变量 > YAML 配置 > 默认值。
func mergeConfig() error {
	// 1. 加载 YAML 配置文件
	configPath := resolveConfigPath()
	if data, err := os.ReadFile(configPath); err == nil {
		var fileCfg Config
		if err := yaml.Unmarshal(data, &fileCfg); err != nil {
			return fmt.Errorf("解析配置文件 %s 失败: %w", configPath, err)
		}
		applyFileConfig(&fileCfg)
	}

	// 2. 环境变量覆盖
	if v := os.Getenv("EM_USERNAME"); v != "" {
		cfg.User = v
	}
	if v := os.Getenv("EM_PASSWORD"); v != "" {
		cfg.Password = v
	}

	if flagLog != "" {
		cfg.Log = flagLog
	}

	// 3. CLI flags 覆盖（最高优先级）
	if flagUser != "" {
		cfg.User = flagUser
	}
	if flagPassword != "" {
		cfg.Password = flagPassword
	}
	if flagModel != "" {
		cfg.OCR.Model = flagModel
	}
	if flagDict != "" {
		cfg.OCR.Dict = flagDict
	}
	if flagONNXLib != "" {
		cfg.OCR.ONNXLib = flagONNXLib
	}

	// 4. ONNX 库自动检测（仅当未显式指定时）
	if cfg.OCR.ONNXLib == "" {
		cfg.OCR.ONNXLib = autoDetectONNXLib()
	}

	return nil
}

// applyFileConfig 将配置文件中的非零值写入全局配置。
func applyFileConfig(fc *Config) {
	if fc.User != "" {
		cfg.User = fc.User
	}
	if fc.Password != "" {
		cfg.Password = fc.Password
	}
	if fc.Log != "" {
		cfg.Log = fc.Log
	}
	if fc.OCR.Model != "" {
		cfg.OCR.Model = fc.OCR.Model
	}
	if fc.OCR.Dict != "" {
		cfg.OCR.Dict = fc.OCR.Dict
	}
	if fc.OCR.ONNXLib != "" {
		cfg.OCR.ONNXLib = fc.OCR.ONNXLib
	}
}

// resolveConfigPath 确定配置文件路径。
func resolveConfigPath() string {
	if flagConfig != "" {
		return flagConfig
	}
	return "config.yaml"
}

// autoDetectONNXLib 自动检测 ONNX Runtime 库路径。
func autoDetectONNXLib() string {
	var libName string
	var systemPaths []string
	switch runtime.GOOS {
	case "darwin":
		libName = fmt.Sprintf("onnxruntime_%s.dylib", runtime.GOARCH)
		systemPaths = []string{
			"/opt/homebrew/lib/libonnxruntime.dylib",
			"/usr/local/lib/libonnxruntime.dylib",
		}
	case "linux":
		libName = fmt.Sprintf("onnxruntime_%s.so", runtime.GOARCH)
		systemPaths = []string{
			"/usr/lib/libonnxruntime.so",
			"/usr/local/lib/libonnxruntime.so",
		}
	case "windows":
		libName = "onnxruntime.dll"
	default:
		return ""
	}

	searchPaths := systemPaths
	searchPaths = append(searchPaths,
		filepath.Join("go-ocr", "lib", libName),
		filepath.Join("..", "go-ocr", "lib", libName),
	)

	for _, p := range searchPaths {
		if info, err := os.Stat(p); err == nil && info.Size() > 1024 {
			return p
		}
	}

	return ""
}

// zerologAdapter 将 zerolog.Logger 适配为 client.Logger 接口。
type zerologAdapter struct{ log zerolog.Logger }

func (z *zerologAdapter) Info(args ...any)             { z.log.Info().Msg(fmt.Sprint(args...)) }
func (z *zerologAdapter) Infof(f string, args ...any)  { z.log.Info().Msgf(f, args...) }
func (z *zerologAdapter) Debug(args ...any)            { z.log.Debug().Msg(fmt.Sprint(args...)) }
func (z *zerologAdapter) Debugf(f string, args ...any) { z.log.Debug().Msgf(f, args...) }
func (z *zerologAdapter) Error(args ...any)            { z.log.Error().Msg(fmt.Sprint(args...)) }
func (z *zerologAdapter) Errorf(f string, args ...any) { z.log.Error().Msgf(f, args...) }

// initLogger 初始化文件日志。
func initLogger() {
	if dir := filepath.Dir(cfg.Log); dir != "." {
		os.MkdirAll(dir, 0700)
	}
	logFile, err := os.OpenFile(cfg.Log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建日志文件失败: %v\n", err)
		cmdLogger = zerolog.New(os.Stderr).With().Timestamp().Logger()
		return
	}
	cmdLogger = zerolog.New(logFile).With().Timestamp().Logger()
}

// resolveCredentials 从合并后的配置获取凭据。
func resolveCredentials() (string, string) {
	return cfg.User, cfg.Password
}
