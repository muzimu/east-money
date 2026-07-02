package cache

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// fileEntry 文件缓存中的条目。
type fileEntry struct {
	Data    any `json:"data"`
	Expired time.Time   `json:"expired"`
}

// File 基于 JSON 文件的持久化缓存实现。
// 适用于 CLI 等短生命周期进程，跨进程共享登录状态。
type File struct {
	mu   sync.RWMutex
	path string
}

// NewFile 创建文件缓存实例。
// path 为缓存文件路径，文件不存在时会自动创建。
func NewFile(path string) *File {
	return &File{path: path}
}

// load 从文件读取所有缓存数据。
func (f *File) load() map[string]*fileEntry {
	f.mu.RLock()
	defer f.mu.RUnlock()

	data, err := os.ReadFile(f.path)
	if err != nil {
		return nil
	}

	var entries map[string]*fileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}
	return entries
}

// save 将所有缓存数据写入文件。
func (f *File) save(entries map[string]*fileEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	return os.WriteFile(f.path, data, 0600)
}

// Get 返回 key 对应的缓存值。
func (f *File) Get(key string) any {
	entries := f.load()
	if entries == nil {
		return nil
	}
	entry, ok := entries[key]
	if !ok {
		return nil
	}
	if time.Now().After(entry.Expired) {
		return nil
	}
	return entry.Data
}

// IsExist 检查 key 是否存在且未过期。
func (f *File) IsExist(key string) bool {
	return f.Get(key) != nil
}

// Set 设置缓存值并指定过期时间。
func (f *File) Set(key string, val any, timeout time.Duration) error {
	entries := f.load()
	if entries == nil {
		entries = make(map[string]*fileEntry)
	}
	entries[key] = &fileEntry{
		Data:    val,
		Expired: time.Now().Add(timeout),
	}
	return f.save(entries)
}

// Delete 删除缓存中的一个 key。
func (f *File) Delete(key string) error {
	entries := f.load()
	if entries == nil {
		return nil
	}
	delete(entries, key)
	return f.save(entries)
}
