package cache

import (
	"context"
	"reflect"

	"git.code.oa.com/trpc-go/trpc-database/localcache"
)

const tenYearsInSecond = 10 * 365 * 24 * 60 * 60

type localcacheImpl struct {
	cache localcache.Cache
	cfg   *Config
	stats *Stats
}

func init() {
	Register(EngineLocalCache, newLocalCache)
}

func newLocalCache(cfg *Config) (API, error) {
	if cfg.DefaultTTL == 0 {
		// DefaultTTL为0表示不过期，但localcache又必须设置过期时间，因此取一个较大的过期时间
		cfg.DefaultTTL = tenYearsInSecond
	}
	return &localcacheImpl{
		cache: localcache.New(localcache.WithCapacity(cfg.sizeToCapacity()), localcache.WithExpiration(cfg.DefaultTTL)),
		cfg:   cfg,
		stats: &Stats{},
	}, nil
}

// Get 定制化Get方法，获取key, 不存在返回ErrEntryNotFound。
func (s *localcacheImpl) Get(key string, value interface{}) error {
	data, ok := s.cache.Get(key)
	if !ok {
		s.stats.atomicAddMissCount()
		return ErrNotFound
	}
	s.stats.atomicAddHitCount()
	return safeSet(value, data)
}

// GetWithLoad 返回key对应的value, 如果key不存在，使用load函数加载返回，并缓存ttl秒。
func (s *localcacheImpl) GetWithLoad(ctx context.Context, key string, value interface{}, load LoadFunc) error {
	hit, err := getWithLoad(ctx, s, key, value, load)
	if err != nil {
		return err
	}
	if !hit {
		s.stats.atomicAddLoadCount()
	}
	return nil
}

// Set 保存一对<key, value>。
func (s *localcacheImpl) Set(key string, value interface{}) error {
	return s.SetWithExpire(key, value, s.cfg.DefaultTTL)
}

// SetWithExpire 设置key value，并制定过期时间。
func (s *localcacheImpl) SetWithExpire(key string, value interface{}, ttl int64) error {
	s.cache.SetWithExpire(key, value, ttl)
	return nil
}

// Delete 删除一个key。
func (s *localcacheImpl) Delete(key string) error {
	s.cache.Del(key)
	return nil
}

// Clear 清空所有元素。
func (s *localcacheImpl) Clear() error {
	s.cache.Clear()
	return nil
}

// GetStats 获取统计数据。
func (s *localcacheImpl) GetStats() Stats {
	s.stats.HitRate = s.stats.calcHitRate() // calculate on read
	return *s.stats
}

// safeSet 将source的值赋值给target
func safeSet(target, source interface{}) error {
	value := reflect.ValueOf(target)
	kind := value.Kind()
	if kind != reflect.Ptr && kind != reflect.Interface {
		return ErrCannotSet
	}
	elem := value.Elem()
	if !elem.CanSet() {
		return ErrCannotSet
	}
	elem.Set(reflect.ValueOf(source))
	return nil
}
