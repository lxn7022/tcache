package cache

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/allegro/bigcache/v3"
)

const defaultEviction = 7 * 24 * time.Hour // 7天

// DefaultConfig 默认配置，导出，外部可以覆盖
var DefaultConfig = bigcache.DefaultConfig(defaultEviction)

type bigcacheImpl struct {
	cache *bigcache.BigCache
	loads int64
	cfg   *Config
}

func init() {
	Register(EngineBigCache, newBigCache)
}

func newBigCache(cfg *Config) (API, error) {
	ctx := context.Background()
	config := DefaultConfig
	config.LifeWindow = time.Duration(cfg.DefaultTTL) * time.Second
	config.HardMaxCacheSize = cfg.MaxSizeInMB
	bc, err := bigcache.New(ctx, config)
	if err != nil {
		return nil, err
	}
	return &bigcacheImpl{cache: bc, cfg: cfg}, nil
}

// Get 获取key, 不存在返回ErrEntryNotFound, 通过输入序列化方式自动解析数据结构
func (s *bigcacheImpl) Get(key string, value interface{}) error {
	data, err := s.cache.Get(key)
	if len(data) == 0 || err != nil {
		return ErrNotFound
	}
	return s.cfg.Serializer.Unmarshal(data, value)
}

// GetWithLoad 返回key对应的value, 如果key不存在，使用load函数加载返回，并缓存ttl秒
func (s *bigcacheImpl) GetWithLoad(ctx context.Context, key string, value interface{}, load LoadFunc) error {
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
func (s *bigcacheImpl) Set(key string, value interface{}) error {
	data, err := s.cfg.Serializer.Marshal(value)
	if err != nil {
		return err
	}
	return s.cache.Set(key, data)
}

// SetWithExpire 设置key value，并制定过期时间
func (s *bigcacheImpl) SetWithExpire(key string, value interface{}, ttl int64) error {
	return ErrNotSupported
}

// Delete 删除一个key
func (s *bigcacheImpl) Delete(key string) error {
	return s.cache.Delete(key)
}

// Clear 清空所有元素
func (s *bigcacheImpl) Clear() error {
	s.loads = 0
	return s.cache.Reset()
}

// GetStats 获取统计数据
func (s *bigcacheImpl) GetStats() Stats {
	stats := s.cache.Stats()
	return Stats{
		Hits:    stats.Hits,
		Misses:  stats.Misses,
		HitRate: float64(stats.Hits) / float64(stats.Hits+stats.Misses),
		Loads:   s.loads,
	}
}
