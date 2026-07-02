// Package credential 提供东方财富会话凭证管理。
// 核心模式：缓存优先 → 双检锁 → 自动重登 → TTL 过期。
package credential

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	"github.com/muzimu/east-money/cache"
)

const cacheKeyPrefix = "eastmoney_validatekey_"

// LoginFunc 执行登录并返回 validateKey。
// 由 client 层以闭包形式注入，避免循环依赖。
type LoginFunc func() (validateKey string, err error)

// SessionHandle 管理东方财富会话凭证。
type SessionHandle interface {
	// GetValidateKey 返回有效的 validateKey。若缓存失效则自动重登。
	GetValidateKey() (string, error)

	// ForceReLogin 强制重新登录，丢弃当前缓存。
	ForceReLogin() error

	// CookieJar 返回会话的 cookie 管理器。
	CookieJar() http.CookieJar
}

// DefaultSession 实现缓存优先 + 双检锁的会话管理。
type DefaultSession struct {
	username    string
	cacheKey    string
	cache       cache.Cache
	cookieJar   *cookiejar.Jar
	loginFunc   LoginFunc
	sessionLock sync.Mutex
	duration    time.Duration // 登录会话时长
	ttlBuffer   time.Duration // TTL 缓冲时间
}

// NewDefaultSession 创建会话管理器。
//
// 参数：
//   - username: 账户用户名（用于缓存 key 隔离）
//   - cache: 缓存后端（必填，若为 nil 则 panic）
//   - loginFunc: 登录回调，在缓存失效时被调用
//   - duration: 登录会话时长
//   - ttlBuffer: TTL 缓冲时间（提前多久刷新）
func NewDefaultSession(
	username string,
	cache cache.Cache,
	loginFunc LoginFunc,
	duration time.Duration,
	ttlBuffer time.Duration,
) *DefaultSession {
	if cache == nil {
		panic("credential: cache is required")
	}
	if ttlBuffer <= 0 {
		ttlBuffer = 2 * time.Minute
	}
	if duration <= 0 {
		duration = 30 * time.Minute
	}

	jar, _ := cookiejar.New(nil)

	return &DefaultSession{
		username:  username,
		cacheKey:  cacheKeyPrefix + username,
		cache:     cache,
		cookieJar: jar,
		loginFunc: loginFunc,
		duration:  duration,
		ttlBuffer: ttlBuffer,
	}
}

// GetValidateKey 返回有效的 validateKey。
// 快路径：缓存命中直接返回。
// 慢路径：缓存未命中/过期 → 双检锁 → 调用 LoginFunc → 写入缓存。
func (s *DefaultSession) GetValidateKey() (string, error) {
	// 快路径：从缓存读取
	if val := s.cache.Get(s.cacheKey); val != nil {
		if key, ok := val.(string); ok && key != "" {
			return key, nil
		}
	}

	// 慢路径：加锁 + 双检
	s.sessionLock.Lock()
	defer s.sessionLock.Unlock()

	// 双检：可能其他 goroutine 已经完成登录
	if val := s.cache.Get(s.cacheKey); val != nil {
		if key, ok := val.(string); ok && key != "" {
			return key, nil
		}
	}

	// 缓存失效，执行登录
	validateKey, err := s.loginFunc()
	if err != nil {
		return "", fmt.Errorf("登录失败: %w", err)
	}

	if validateKey == "" {
		return "", fmt.Errorf("登录返回空的 validateKey")
	}

	// 计算 TTL 并写入缓存
	ttl := s.duration - s.ttlBuffer
	if ttl <= 0 {
		ttl = time.Minute
	}
	if setErr := s.cache.Set(s.cacheKey, validateKey, ttl); setErr != nil {
		return "", fmt.Errorf("缓存 validateKey 失败: %w", setErr)
	}

	return validateKey, nil
}

// ForceReLogin 清除缓存中的 validateKey 并触发重新登录。
func (s *DefaultSession) ForceReLogin() error {
	// 清除缓存中的旧 key
	_ = s.cache.Delete(s.cacheKey)

	// 触发重新登录
	_, err := s.GetValidateKey()
	return err
}

// CookieJar 返回会话的 cookie 管理器。
func (s *DefaultSession) CookieJar() http.CookieJar {
	return s.cookieJar
}
