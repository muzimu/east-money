package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemory_SetAndGet(t *testing.T) {
	mem := NewMemory()
	key := "test_key"
	val := "test_value"

	err := mem.Set(key, val, time.Minute)
	assert.NoError(t, err)

	result := mem.Get(key)
	assert.Equal(t, val, result)
}

func TestMemory_GetExpired(t *testing.T) {
	mem := NewMemory()
	err := mem.Set("k", "v", 1*time.Millisecond)
	assert.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	result := mem.Get("k")
	assert.Nil(t, result)
}

func TestMemory_IsExist(t *testing.T) {
	mem := NewMemory()

	// 不存在
	assert.False(t, mem.IsExist("missing"))

	// 存在
	mem.Set("exists", "value", time.Minute)
	assert.True(t, mem.IsExist("exists"))

	// 过期后不存在
	mem.Set("expires", "x", 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	assert.False(t, mem.IsExist("expires"))
}

func TestMemory_Delete(t *testing.T) {
	mem := NewMemory()
	mem.Set("k", "v", time.Minute)
	assert.True(t, mem.IsExist("k"))

	err := mem.Delete("k")
	assert.NoError(t, err)
	assert.False(t, mem.IsExist("k"))
	assert.Nil(t, mem.Get("k"))
}

func TestMemory_Concurrent(t *testing.T) {
	mem := NewMemory()
	var wg sync.WaitGroup
	n := 100

	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			mem.Set("key", idx, time.Minute)
			_ = mem.Get("key")
		}(i)
	}
	wg.Wait()
	assert.True(t, mem.IsExist("key"))
}
