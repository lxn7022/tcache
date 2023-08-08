package cache

import (
	"context"
	"sync/atomic"

	"github.com/coocood/freecache"
)

type freecacheImpl struct {
	cache *freecache.Cache
	loads int64
	cfg   *Config
}

func init() {
	Register(EngineFreeCache, newFreeCache)
}

func newFreeCache(cfg *Config) (API, error) {
	return &freecacheImpl{cache: freecache.NewCache(cfg.MaxSizeInBytes()), cfg: cfg}, nil
}

// Get 获取key, 不存在返回ErrEntryNotFound, 通过输入序列化方式自动解析数据结构
func (s *freecacheImpl) Get(key string, value interface{}) error {
	data, err := s.cache.Get(str2bytes(key))
	if len(data) == 0 || err != nil {
		return ErrNotFound
	}
	return s.cfg.Serializer.Unmarshal(data, value)
}

// GetWithLoad 返回key对应的value, 如果key不存在，使用load函数加载返回，并缓存ttl秒
func (s *freecacheImpl) GetWithLoad(ctx context.Context, key string, value interface{}, load LoadFunc) error {
	hit, err := getWithLoad(ctx, s, key, value, load)
	if err != nil {
		return err
	}
	if !hit {
		atomic.AddInt64(&s.loads, 1)
	}
	return nil
}

// Set 保存一对<key, value>，可能因value格式不支持而保存失败, 通过输入序列化方式自动打包数据
func (s *freecacheImpl) Set(key string, value interface{}) error {
	return s.SetWithExpire(key, value, s.cfg.DefaultTTL)
}

// SetWithExpire 设置key value，并制定过期时间
func (s *freecacheImpl) SetWithExpire(key string, value interface{}, ttl int64) error {
	data, err := s.cfg.Serializer.Marshal(value)
	if err != nil {
		return err
	}
	return s.cache.Set(str2bytes(key), data, int(ttl))
}

// Delete 删除一个key
func (s *freecacheImpl) Delete(key string) error {
	s.cache.Del(str2bytes(key))
	return nil
}

// Clear 清空所有元素
func (s *freecacheImpl) Clear() error {
	s.loads = 0
	s.cache.Clear()
	return nil
}

// GetStats 获取统计数据
func (s *freecacheImpl) GetStats() Stats {
	return Stats{
		Hits:    s.cache.HitCount(),
		Misses:  s.cache.MissCount(),
		HitRate: s.cache.HitRate(),
		Loads:   s.loads,
	}
}
