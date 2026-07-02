package cache

import (
	"sync"
	"time"
)

// Memory 并发安全的内存缓存实现，基于 map 和 sync.RWMutex。
type Memory struct {
	mu   sync.RWMutex
	data map[string]*item
}

// item 缓存数据项，包含值和过期时间。
type item struct {
	Data    any
	Expired time.Time
}

// NewMemory 创建一个可用的内存缓存实例。
func NewMemory() *Memory {
	return &Memory{
		data: map[string]*item{},
	}
}

// Get 返回 key 对应的缓存值，若不存在或已过期则返回 nil。
func (m *Memory) Get(key string) any {
	m.mu.RLock()
	it, ok := m.data[key]
	expired := ok && it.Expired.Before(time.Now())
	m.mu.RUnlock()

	if !ok {
		return nil
	}
	if expired {
		m.deleteExpiredKey(key)
		return nil
	}
	return it.Data
}

// IsExist 检查 key 是否存在且未过期。
func (m *Memory) IsExist(key string) bool {
	m.mu.RLock()
	it, ok := m.data[key]
	expired := ok && it.Expired.Before(time.Now())
	m.mu.RUnlock()

	if !ok {
		return false
	}
	if expired {
		m.deleteExpiredKey(key)
		return false
	}
	return true
}

// Set 设置缓存值并指定过期时间。
func (m *Memory) Set(key string, val any, timeout time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = &item{
		Data:    val,
		Expired: time.Now().Add(timeout),
	}
	return nil
}

// Delete 删除缓存中的一个 key。
func (m *Memory) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

// deleteExpiredKey 仅在 key 已过期时删除它。
// 调用方不得持有任何锁。
func (m *Memory) deleteExpiredKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if it, ok := m.data[key]; ok && it.Expired.Before(time.Now()) {
		delete(m.data, key)
	}
}
