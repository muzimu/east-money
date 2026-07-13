package credential

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/muzimu/east-money/cache"
	"github.com/stretchr/testify/assert"
)

func TestGetValidateKey_CacheHit(t *testing.T) {
	mem := cache.NewMemory()
	_ = mem.Set(cacheKeyPrefix+"testuser", "cached-key", 5*time.Minute)

	var loginCalled int
	loginFunc := func() (string, error) {
		loginCalled++
		return "new-key", nil
	}

	sess := NewDefaultSession("testuser", mem, loginFunc, 30*time.Minute, 2*time.Minute)
	key, err := sess.GetValidateKey()

	assert.NoError(t, err)
	assert.Equal(t, "cached-key", key)
	assert.Equal(t, 0, loginCalled, "login should not be called on cache hit")
}

func TestGetValidateKey_CacheExpired(t *testing.T) {
	mem := cache.NewMemory()
	_ = mem.Set(cacheKeyPrefix+"testuser", "expired-key", 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	var loginCalled bool
	loginFunc := func() (string, error) {
		loginCalled = true
		return "fresh-key", nil
	}

	sess := NewDefaultSession("testuser", mem, loginFunc, 30*time.Minute, 2*time.Minute)
	key, err := sess.GetValidateKey()

	assert.NoError(t, err)
	assert.Equal(t, "fresh-key", key)
	assert.True(t, loginCalled)
}

func TestGetValidateKey_CacheEmpty(t *testing.T) {
	mem := cache.NewMemory()

	var loginCalled bool
	loginFunc := func() (string, error) {
		loginCalled = true
		return "new-key", nil
	}

	sess := NewDefaultSession("testuser", mem, loginFunc, 30*time.Minute, 2*time.Minute)
	key, err := sess.GetValidateKey()

	assert.NoError(t, err)
	assert.Equal(t, "new-key", key)
	assert.True(t, loginCalled)
}

func TestGetValidateKey_LoginFails(t *testing.T) {
	mem := cache.NewMemory()

	loginFunc := func() (string, error) {
		return "", fmt.Errorf("login error")
	}

	sess := NewDefaultSession("testuser", mem, loginFunc, 30*time.Minute, 2*time.Minute)
	key, err := sess.GetValidateKey()

	assert.Error(t, err)
	assert.Empty(t, key)
	assert.Contains(t, err.Error(), "登录失败")
}

func TestGetValidateKey_EmptyReturn(t *testing.T) {
	mem := cache.NewMemory()

	loginFunc := func() (string, error) {
		return "", nil
	}

	sess := NewDefaultSession("testuser", mem, loginFunc, 30*time.Minute, 2*time.Minute)
	key, err := sess.GetValidateKey()

	assert.Error(t, err)
	assert.Empty(t, key)
	assert.Contains(t, err.Error(), "空的 validateKey")
}

func TestGetValidateKey_ConcurrentDoubleCheck(t *testing.T) {
	mem := cache.NewMemory()

	var mu sync.Mutex
	callCount := 0

	loginFunc := func() (string, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		// 模拟登录延迟
		time.Sleep(50 * time.Millisecond)
		return "concurrent-key", nil
	}

	sess := NewDefaultSession("testuser", mem, loginFunc, 30*time.Minute, 2*time.Minute)

	var wg sync.WaitGroup
	n := 20
	keys := make([]string, n)

	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			k, err := sess.GetValidateKey()
			if err == nil {
				keys[idx] = k
			}
		}(i)
	}
	wg.Wait()

	// 所有 goroutine 应获得相同的 key
	for i := range n {
		if keys[i] != "" {
			assert.Equal(t, "concurrent-key", keys[i])
		}
	}

	mu.Lock()
	assert.Equal(t, 1, callCount, "loginFunc should be called exactly once")
	mu.Unlock()
}

func TestForceReLogin(t *testing.T) {
	mem := cache.NewMemory()
	_ = mem.Set(cacheKeyPrefix+"testuser", "old-key", 5*time.Minute)

	loginCount := 0
	loginFunc := func() (string, error) {
		loginCount++
		return fmt.Sprintf("new-key-%d", loginCount), nil
	}

	sess := NewDefaultSession("testuser", mem, loginFunc, 30*time.Minute, 2*time.Minute)

	// 强制重新登录
	err := sess.ForceReLogin()
	assert.NoError(t, err)

	// 验证已获得新 key
	key, err := sess.GetValidateKey()
	assert.NoError(t, err)
	assert.Equal(t, "new-key-1", key)
}

func TestCookieJar(t *testing.T) {
	mem := cache.NewMemory()
	loginFunc := func() (string, error) {
		return "key", nil
	}

	sess := NewDefaultSession("testuser", mem, loginFunc, 30*time.Minute, 2*time.Minute)
	jar := sess.CookieJar()
	assert.NotNil(t, jar)
}

func TestNewDefaultSession_NilCache(t *testing.T) {
	assert.Panics(t, func() {
		NewDefaultSession("user", nil, nil, 0, 0)
	})
}
