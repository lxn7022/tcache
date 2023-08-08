// Package cache localcache库接口封装，对外提供统一接口
package cache

import (
	"context"
	"reflect"
	"sync"
	"unsafe"

	"git.code.oa.com/video_pay_root/pay-go-comm/utils/parallel"
	"golang.org/x/sync/singleflight"
)

// Constructor cache实例构造函数
type Constructor func(cfg *Config) (API, error)

// LoadFunc 加载key对应的value数据，用于填充cache
type LoadFunc func(ctx context.Context, key string, value interface{}) (ttl int64, err error)

// API 接口抽象
type API interface {
	// Get 获取key, 不存在返回ErrEntryNotFound
	Get(key string, value interface{}) error
	// GetWithLoad 返回key对应的value, 如果key不存在，使用load函数加载返回，并缓存ttl秒
	GetWithLoad(ctx context.Context, key string, value interface{}, load LoadFunc) error
	// Set 保存一对<key, value>
	Set(key string, value interface{}) error
	// SetWithExpire 设置key value，并指定过期时间
	SetWithExpire(key string, value interface{}, ttl int64) error
	// Delete 删除一个key
	Delete(key string) error
	// Clear 清空所有元素
	Clear() error
	// GetStats 获取统计数据
	GetStats() Stats
}

// Serializer 用于value序列化的接口
type Serializer interface {
	Unmarshal(in []byte, body interface{}) error
	Marshal(body interface{}) (out []byte, err error)
}

var (
	lock         = sync.RWMutex{}
	cacheEngines = make(map[string]Constructor)
	group        = &singleflight.Group{}
)

// New 构造函数
func New(engine string, maxSizeInMB int, opts ...Option) (API, error) {
	parallel.GoRun(func() {
		report(engine)
	})
	cfg := &Config{MaxSizeInMB: maxSizeInMB}
	for _, opt := range opts {
		opt(cfg)
	}
	cfg.fixEmpty()

	if f := GetConstructor(engine); f != nil {
		return f(cfg)
	}
	return nil, ErrNoSuchEngine
}

// Register 自定义注册一个新的cache构造函数，允许在bigcache/freecache/fastcache之外自定义扩展
func Register(engine string, newFunc Constructor) {
	lock.Lock()
	defer lock.Unlock()
	cacheEngines[engine] = newFunc
}

// GetConstructor 根据cache引擎名返回cache构造函数
func GetConstructor(engine string) Constructor {
	lock.RLock()
	f := cacheEngines[engine]
	lock.RUnlock()
	return f
}

func str2bytes(s string) (b []byte) {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bh.Data, bh.Len, bh.Cap = sh.Data, sh.Len, sh.Len
	return b
}

// getWithLoad key如果不存在，实时加载
func getWithLoad(ctx context.Context, cache API, key string, value interface{}, load LoadFunc) (hit bool, err error) {
	if err := cache.Get(key, value); err == nil {
		return true, nil
	}
	// 不存在，则重新获取；使用singleflight防止并发获取
	if _, err, _ := group.Do(key, func() (interface{}, error) {
		ttl, err := load(ctx, key, value)
		if err != nil {
			return nil, err
		}
		if ttl > 0 {
			cache.SetWithExpire(key, value, ttl)
		} else {
			cache.Set(key, value)
		}
		return nil, err
	}); err != nil {
		return false, err
	}
	return false, nil
}
