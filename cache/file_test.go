package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFile_SetAndGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	f := NewFile(path)

	err := f.Set("k", "v", time.Minute)
	assert.NoError(t, err)

	assert.Equal(t, "v", f.Get("k"))
}

func TestFile_IsExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	f := NewFile(path)

	assert.False(t, f.IsExist("k"))
	f.Set("k", "v", time.Minute)
	assert.True(t, f.IsExist("k"))
}

func TestFile_Delete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	f := NewFile(path)

	f.Set("k", "v", time.Minute)
	assert.True(t, f.IsExist("k"))

	f.Delete("k")
	assert.False(t, f.IsExist("k"))
	assert.Nil(t, f.Get("k"))
}

func TestFile_Expired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	f := NewFile(path)

	f.Set("k", "v", 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	assert.Nil(t, f.Get("k"))
	assert.False(t, f.IsExist("k"))
}

func TestFile_PersistAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")

	// 模拟进程 A 写入
	f1 := NewFile(path)
	f1.Set("token", "abc123", time.Hour)

	// 模拟进程 B 读取
	f2 := NewFile(path)
	assert.Equal(t, "abc123", f2.Get("token"))
}

func TestFile_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	os.WriteFile(path, []byte{}, 0600)

	f := NewFile(path)
	assert.Nil(t, f.Get("k"))
	assert.False(t, f.IsExist("k"))
	// Set 应能正常覆盖空文件
	assert.NoError(t, f.Set("k", "v", time.Minute))
	assert.Equal(t, "v", f.Get("k"))
}
