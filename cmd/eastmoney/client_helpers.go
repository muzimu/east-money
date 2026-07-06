package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/muzimu/east-money/cache"
	"github.com/muzimu/east-money/captcha"
	"github.com/muzimu/east-money/client"
)

// createClient 创建交易客户端，使用文件缓存持久化登录状态。
// skipCookies=true 时跳过 Cookie 加载，用于 login 命令强制重新认证。
func createClient(skipCookies bool) (*client.Client, error) {
	u, p := resolveCredentials()
	if u == "" || p == "" {
		return nil, fmt.Errorf("请提供用户名和密码（-u/-p、环境变量或配置文件）")
	}

	var recognizer captcha.Recognizer
	var err error
	if cfg.OCR.Remote != "" {
		recognizer = captcha.NewRemoteRecognizer(captcha.WithRemoteEndpoint(cfg.OCR.Remote))
	} else {
		recognizer, err = captcha.NewDefaultRecognizer(cfg.OCR.Model, cfg.OCR.Dict, cfg.OCR.ONNXLib)
		if err != nil {
			return nil, fmt.Errorf("创建 OCR 引擎失败: %w", err)
		}
	}

	c, err := client.NewClient(u, p, recognizer, client.WithLogger(&zerologAdapter{cmdLogger}))
	if err != nil {
		recognizer.Close()
		return nil, fmt.Errorf("创建客户端失败: %w", err)
	}

	if dir := filepath.Dir(flagSession); dir != "." {
		os.MkdirAll(dir, 0700)
	}
	c.SetCache(cache.NewFile(flagSession))

	if !skipCookies {
		loadCookies(c)
	}

	currentClient = c
	return c, nil
}

// cookiePath 根据 session 路径推导 Cookie 文件路径。
func cookiePath() string {
	dir := filepath.Dir(flagSession)
	return filepath.Join(dir, "cookies.json")
}

// saveCookies 持久化 Cookie 到文件。
func saveCookies() {
	if currentClient == nil {
		return
	}
	cookies := currentClient.ExportCookies()
	data, err := json.Marshal(cookies)
	if err != nil {
		cmdLogger.Error().Err(err).Msg("序列化 Cookie 失败")
		return
	}
	if err := os.WriteFile(cookiePath(), data, 0600); err != nil {
		cmdLogger.Error().Err(err).Msg("写入 Cookie 文件失败")
	}
}

// loadCookies 从文件加载 Cookie。
func loadCookies(c *client.Client) {
	data, err := os.ReadFile(cookiePath())
	if err != nil {
		return
	}
	var cookies []*http.Cookie
	if json.Unmarshal(data, &cookies) != nil {
		return
	}
	c.ImportCookies(cookies)
}
