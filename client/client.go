package client

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	eastmoney "github.com/muzimu/east-money"
	"github.com/muzimu/east-money/cache"
	"github.com/muzimu/east-money/captcha"
	"github.com/muzimu/east-money/credential"
)

// Logger 日志接口，兼容标准库和第三方日志库。
type Logger interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

// LoginFailureCallback 登录失败回调，用于外部监控和告警。
type LoginFailureCallback func(err error)

// Client 东方财富交易客户端。一个实例对应一个账户。
type Client struct {
	username string
	password string

	httpClient *http.Client
	session    credential.SessionHandle
	captcha    captcha.Recognizer
	logger     Logger

	// 可配置选项
	retryMax    int
	retryWait   time.Duration
	duration    int // 登录会话时长（分钟）
	onLoginFail LoginFailureCallback
}

// Option 函数式选项类型。
type Option func(*Client)

// WithRetry 设置重试次数和基础等待时间。
func WithRetry(max int, wait time.Duration) Option {
	return func(c *Client) {
		c.retryMax = max
		c.retryWait = wait
	}
}

// WithDuration 设置登录会话时长（分钟）。
func WithDuration(minutes int) Option {
	return func(c *Client) {
		if minutes > 0 {
			c.duration = minutes
		}
	}
}

// WithLogger 注入日志记录器。
func WithLogger(l Logger) Option {
	return func(c *Client) { c.logger = l }
}

// WithLoginFailureCallback 设置登录失败回调。
func WithLoginFailureCallback(cb LoginFailureCallback) Option {
	return func(c *Client) { c.onLoginFail = cb }
}

// WithCaptchaRecognizer 注入自定义验证码识别器。
func WithCaptchaRecognizer(r captcha.Recognizer) Option {
	return func(c *Client) { c.captcha = r }
}

// WithHTTPClient 注入自定义 HTTP 客户端（用于测试 mock）。
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.httpClient = hc }
}

// NewClient 创建一个东方财富客户端。
//
// 参数：
//   - username: 账户用户名
//   - password: 账户密码（明文）
//   - captchaRecognizer: 验证码识别器（必填）
//   - opts: 可选配置项
func NewClient(username, password string, captchaRecognizer captcha.Recognizer, opts ...Option) (*Client, error) {
	if captchaRecognizer == nil {
		return nil, fmt.Errorf("captcha recognizer is required")
	}

	// 默认值
	jar, _ := cookiejar.New(nil)
	c := &Client{
		username: username,
		password: password,
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		captcha:   captchaRecognizer,
		logger:    &noopLogger{},
		retryMax:  3,
		retryWait: 500 * time.Millisecond,
		duration:  30,
	}

	// 应用选项
	for _, o := range opts {
		o(c)
	}

	// 如果用户提供了自定义 HTTP Client，确保使用其 CookieJar
	if c.httpClient.Jar == nil {
		c.httpClient.Jar = jar
	}

	// 创建会话管理器，注入登录回调
	memCache := cache.NewMemory()
	loginFunc := c.createLoginFunc()
	dur := time.Duration(c.duration) * time.Minute
	c.session = credential.NewDefaultSession(username, memCache, loginFunc, dur, 2*time.Minute)
	c.httpClient.Jar = c.session.CookieJar()

	return c, nil
}

// SetCache 替换默认的内存缓存为自定义实现（如 Redis）。
// 须在首次 API 调用前设置。
func (c *Client) SetCache(cache cache.Cache) {
	loginFunc := c.createLoginFunc()
	dur := time.Duration(c.duration) * time.Minute
	c.session = credential.NewDefaultSession(c.username, cache, loginFunc, dur, 2*time.Minute)
	c.httpClient.Jar = c.session.CookieJar()
}

// GetValidateKey 返回当前有效的 validateKey，供外部使用。
func (c *Client) GetValidateKey() (string, error) {
	return c.session.GetValidateKey()
}

// ForceReLogin 强制重新登录。
func (c *Client) ForceReLogin() error {
	return c.session.ForceReLogin()
}

// ExportCookies 导出当前会话的所有 cookie，用于跨进程持久化。
func (c *Client) ExportCookies() []*http.Cookie {
	return c.httpClient.Jar.Cookies(c.baseURL())
}

// ImportCookies 导入 cookie 到当前会话，用于跨进程恢复登录状态。
func (c *Client) ImportCookies(cookies []*http.Cookie) {
	c.httpClient.Jar.SetCookies(c.baseURL(), cookies)
}

func (c *Client) baseURL() *url.URL {
	u, _ := url.Parse(eastmoney.BaseURL)
	return u
}

func (c *Client) newRequest(method, targetURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, targetURL, body)
	if err != nil {
		return nil, err
	}
	applyBaseHeaders(req)
	return req, nil
}

func (c *Client) newFormRequest(targetURL string, form url.Values) (*http.Request, error) {
	req, err := c.newRequest(http.MethodPost, targetURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

func applyBaseHeaders(req *http.Request) {
	for k, v := range eastmoney.BaseHeaders() {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}
}
