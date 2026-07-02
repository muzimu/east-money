package client

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"image"
	_ "image/jpeg" // JPEG 解码器
	_ "image/png"  // PNG 解码器
	"io"
	"math/big"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	eastmoney "github.com/muzimu/east-money"
)

// createLoginFunc 创建登录回调闭包。
// 将 loginFunc 注入 credential.DefaultSession，实现依赖反转。
func (c *Client) createLoginFunc() func() (string, error) {
	return func() (string, error) {
		return c.doLogin()
	}
}

// doLogin 执行完整的登录流程：
//  1. 获取验证码图片并 OCR
//  2. RSA 加密密码
//  3. POST 登录表单
//  4. GET Trade/Buy 页面提取 em_validatekey
func (c *Client) doLogin() (string, error) {
	c.logger.Debug("开始登录...")

	// 1. 获取验证码
	randNum, captchaCode, err := c.getCaptcha()
	if err != nil {
		return "", fmt.Errorf("获取验证码失败: %w", err)
	}
	c.logger.Debugf("验证码: %s (rand=%s)", captchaCode, randNum)

	// 2. RSA 加密密码
	encryptedPass, err := rsaEncryptPassword(c.password)
	if err != nil {
		return "", fmt.Errorf("密码加密失败: %w", err)
	}

	// 3. 构造并发送登录请求
	loginURL := eastmoney.BaseURL + eastmoney.LoginPath
	form := url.Values{
		"userId":       {c.username},
		"password":     {encryptedPass},
		"randNumber":   {randNum},
		"identifyCode": {captchaCode},
		"duration":     {strconv.Itoa(c.duration)},
		"authCode":     {""},
		"type":         {"Z"},
		"secInfo":      {""},
	}

	c.logger.Debugf("发送登录请求: %s", loginURL)
	resp, err := c.httpClient.PostForm(loginURL, form)
	if err != nil {
		return "", fmt.Errorf("登录 POST 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("登录返回 HTTP %d: %s", resp.StatusCode, string(body))
	}

	// 检查登录响应
	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err == nil {
		c.logger.Infof("登录响应: Status=%s, Message=%s", loginResp.Status, loginResp.Message)
		if !loginResp.IsSuccess() {
			return "", fmt.Errorf("登录失败: %s", loginResp.Message)
		}
	} else {
		c.logger.Debugf("登录响应（非JSON）: %s", string(body))
	}

	// 提取 validateKey
	validateKey, err := c.extractValidateKey()
	if err != nil {
		return "", fmt.Errorf("登录失败: %w", err)
	}

	c.logger.Infof("登录成功, validateKey=%s", validateKey)
	return validateKey, nil
}

// getCaptcha 获取验证码图片并进行 OCR 识别。
// 失败时最多重试 MaxCaptchaRetry 次（匹配 Python 递归重试行为）。
func (c *Client) getCaptcha() (randNum string, code string, err error) {
	for attempt := 0; attempt < eastmoney.MaxCaptchaRetry; attempt++ {
		randNum, code, err = c.tryGetCaptcha()
		if err != nil {
			c.logger.Debugf("验证码尝试 %d/%d 失败: %v", attempt+1, eastmoney.MaxCaptchaRetry, err)
			continue
		}

		// 验证 OCR 结果可解析为数字
		if _, parseErr := strconv.Atoi(code); parseErr != nil {
			c.logger.Debugf("验证码 OCR 返回非数字 '%s', 重试...", code)
			continue
		}

		return randNum, code, nil
	}
	return "", "", fmt.Errorf("验证码识别失败，已重试 %d 次: %w", eastmoney.MaxCaptchaRetry, err)
}

// tryGetCaptcha 单次验证码获取尝试。
func (c *Client) tryGetCaptcha() (string, string, error) {
	// 生成随机数
	r, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return "", "", fmt.Errorf("生成随机数失败: %w", err)
	}
	randNum := fmt.Sprintf("0.%d", r)

	// 获取验证码图片
	captchaURL := eastmoney.BaseURL + eastmoney.CaptchaPath + randNum
	resp, err := c.httpClient.Get(captchaURL)
	if err != nil {
		return "", "", fmt.Errorf("获取验证码图片失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("验证码接口返回 HTTP %d: %s", resp.StatusCode, string(body))
	}

	// 解码图片
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("解码验证码图片失败: %w", err)
	}

	// OCR 识别
	code, err := c.captcha.Recognize(img)
	if err != nil {
		return "", "", fmt.Errorf("OCR 识别失败: %w", err)
	}

	code = strings.TrimSpace(code)
	return randNum, code, nil
}

// extractValidateKey 访问 Trade/Buy 页面，从 HTML 中提取 em_validatekey。
func (c *Client) extractValidateKey() (string, error) {
	pageURL := eastmoney.BaseURL + eastmoney.TradeBuyPage
	resp, err := c.httpClient.Get(pageURL)
	if err != nil {
		return "", fmt.Errorf("获取交易页面失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("交易页面返回 HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取交易页面失败: %w", err)
	}

	re := regexp.MustCompile(`id="em_validatekey" type="hidden" value="([^"]*)"`)
	matches := re.FindSubmatch(body)
	if len(matches) < 2 {
		return "", fmt.Errorf("在 HTML 中未找到 em_validatekey")
	}

	return string(matches[1]), nil
}

// rsaEncryptPassword 使用东方财富公钥 RSA PKCS1v15 加密密码。
func rsaEncryptPassword(password string) (string, error) {
	block, _ := pem.Decode([]byte(eastmoney.RSAPublicKeyPEM))
	if block == nil {
		return "", fmt.Errorf("PEM 解码失败")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("解析公钥失败: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("公钥类型不是 RSA")
	}

	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPub, []byte(password))
	if err != nil {
		return "", fmt.Errorf("RSA 加密失败: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}
