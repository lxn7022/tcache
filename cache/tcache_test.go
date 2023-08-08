package cache

import (
	"context"
	"testing"
	"time"

	"git.code.oa.com/trpc-go/trpc-go"
	"github.com/stretchr/testify/assert"
)

const (
	keyNumber    = "NUMBER"
	keyString    = "STRING"
	keyBytes     = "BYTES"
	keyStruct    = "STRUCT"
	keyLoadError = "LOAD_ERROR"
)

func getLoadFunc(ttl int64) LoadFunc {
	return func(ctx context.Context, key string, value interface{}) (int64, error) {
		switch key {
		case keyNumber:
			if v, ok := value.(*int); ok {
				*v = 1
			}
		case keyString:
			if v, ok := value.(*string); ok {
				*v = "ok"
			}
		case keyBytes:
			if v, ok := value.(*[]byte); ok {
				*v = []byte("xxx")
			}
		case keyStruct:
			if v, ok := value.(*Stats); ok {
				v.Hits = 100
			}
		}
		return ttl, nil
	}
}

func testFunc(t *testing.T, cache API, ttl int64) {
	// int值不存在
	var numVal int
	assert.Equal(t, ErrNotFound, cache.Get(keyNumber, &numVal))
	assert.Equal(t, 0, numVal)

	// int值存在
	assert.Nil(t, cache.Set(keyNumber, 1))
	time.Sleep(60 * time.Millisecond) // 如果是异步set的，可能会有延迟，set完立马取取不到，延迟一段时间取
	assert.Nil(t, cache.Get(keyNumber, &numVal))
	assert.Equal(t, 1, numVal)

	// bytes值存在
	var bytesVal []byte
	assert.Nil(t, cache.Set(keyBytes, []byte("xxx")))
	time.Sleep(60 * time.Millisecond)
	assert.Nil(t, cache.Get(keyBytes, &bytesVal))
	assert.Equal(t, []byte("xxx"), bytesVal)

	// string值不存在,自动加载
	var strVal string
	ctx := trpc.BackgroundContext()
	assert.Nil(t, cache.GetWithLoad(ctx, keyString, &strVal, getLoadFunc(ttl)))
	assert.Equal(t, "ok", strVal)
	time.Sleep(3 * time.Second)
	assert.Equal(t, ErrNotFound, cache.Get(keyString, &strVal))

	// struct值不存在,自动加载
	var structVal Stats
	assert.Nil(t, cache.GetWithLoad(ctx, keyStruct, &structVal, getLoadFunc(ttl)))
	assert.Equal(t, Stats{Hits: 100}, structVal)
	// 删除
	assert.Nil(t, cache.Delete(keyStruct))
	assert.Equal(t, ErrNotFound, cache.Get(keyStruct, &structVal))

	// 获取统计数据
	stats := cache.GetStats()
	assert.Equal(t, int64(2), stats.Loads)
	assert.Equal(t, int64(2), stats.Hits)
	assert.Equal(t, int64(5), stats.Misses)

	// clear
	assert.Nil(t, cache.Set(keyNumber, 1))
	assert.Nil(t, cache.Clear())
	assert.Equal(t, ErrNotFound, cache.Get(keyNumber, &numVal))
}

func TestNewBigCache(t *testing.T) {
	cache, err := New(EngineBigCache, 500, WithDefaultTTL(1))
	assert.Nil(t, err)

	testFunc(t, cache, 0)
}

func TestNewFreeCache(t *testing.T) {
	cache, err := New(EngineFreeCache, 500)
	assert.Nil(t, err)

	testFunc(t, cache, 1)
}

func TestNewFastCache(t *testing.T) {
	cache, err := New(EngineFastCache, 500)
	assert.Nil(t, err)

	testFunc(t, cache, 1)
}

func TestNewLocalCache(t *testing.T) {
	cache, err := New(EngineLocalCache, 500)
	assert.Nil(t, err)

	testFunc(t, cache, 3)
}
