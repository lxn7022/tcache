package cache

import (
	"context"
	"sync"
)

const defaultSize = 500 // 500MB

var (
	defaultCache API // 默认cache实例，方便使用
	once         sync.Once
)

func getDefault() API {
	once.Do(func() {
		defaultCache, _ = New(EngineFreeCache, defaultSize)
	})
	return defaultCache
}

// Get 获取key, 不存在返回ErrEntryNotFound
func Get(key string, value interface{}) error {
	return getDefault().Get(key, value)
}

// GetWithLoad 返回key对应的value, 如果key不存在，使用load函数加载返回，并缓存ttl秒
func GetWithLoad(ctx context.Context, key string, value interface{}, load LoadFunc) error {
	return getDefault().GetWithLoad(ctx, key, value, load)
}

// Set 保存一对<key, value>，可能因value格式不支持序列化而保存失败
func Set(key string, value interface{}) error {
	return getDefault().Set(key, value)
}

// SetWithExpire 设置key value，并制定过期时间
func SetWithExpire(key string, value interface{}, ttl int64) error {
	return getDefault().SetWithExpire(key, value, ttl)
}

// Delete 删除一个key
func Delete(key string) error {
	return getDefault().Delete(key)
}

// Clear 清空所有元素
func Clear() error {
	return getDefault().Clear()
}

// GetStats 获取统计数据
func GetStats() Stats {
	return getDefault().GetStats()
}
