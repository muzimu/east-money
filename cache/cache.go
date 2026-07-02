// Package cache 提供缓存接口及默认内存实现。
// 参考 github.com/silenceper/wechat/v2/cache 的设计模式。
package cache

import "time"

// Cache 缓存接口，支持 TTL 过期。
type Cache interface {
	Get(key string) any
	Set(key string, val any, timeout time.Duration) error
	IsExist(key string) bool
	Delete(key string) error
}
