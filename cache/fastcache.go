package cache

import (
	"context"
	"encoding/binary"
	"sync/atomic"
	"time"

	"github.com/VictoriaMetrics/fastcache"
)

type fastcacheImpl struct {
	cache  *fastcache.Cache
	loads  int64
	misses int64
	cfg    *Config
}

const (
	timestampSizeInBytes = 8 // 用于保存时戳的字节数
)

func init() {
	Register(EngineFastCache, newFastCache)
}

func newFastCache(cfg *Config) (API, error) {
	return &fastcacheImpl{cache: fastcache.New(cfg.MaxSizeInBytes()), cfg: cfg}, nil
}

// Get 获取key, 不存在返回ErrEntryNotFound, 通过输入序列化方式自动解析数据结构
func (s *fastcacheImpl) Get(key string, value interface{}) error {
	var entry []byte
	if entry = s.cache.Get(entry, str2bytes(key)); len(entry) == 0 {
		return ErrNotFound
	}
	expire, data := readEntry(entry)
	if expire > 0 && expire < uint64(time.Now().Unix()) {
		s.Delete(key)
		atomic.AddInt64(&s.misses, 1)
		return ErrNotFound
	}
	return s.cfg.Serializer.Unmarshal(data, value)
}

// GetWithLoad 返回key对应的value, 如果key不存在，使用load函数加载返回，并缓存ttl秒
func (s *fastcacheImpl) GetWithLoad(ctx context.Context, key string, value interface{}, load LoadFunc) error {
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
func (s *fastcacheImpl) Set(key string, value interface{}) error {
	return s.SetWithExpire(key, value, s.cfg.DefaultTTL)
}

// SetWithExpire 设置key value，并制定过期时间
func (s *fastcacheImpl) SetWithExpire(key string, value interface{}, ttl int64) error {
	var expire int64
	if ttl > 0 {
		expire = time.Now().Unix() + ttl
	}
	data, err := s.cfg.Serializer.Marshal(value)
	if err != nil {
		return err
	}
	s.cache.Set(str2bytes(key), wrapEntry(uint64(expire), data))
	return nil
}

// Delete 删除一个key
func (s *fastcacheImpl) Delete(key string) error {
	s.cache.Del(str2bytes(key))
	return nil
}

// Clear 清空所有元素
func (s *fastcacheImpl) Clear() error {
	s.loads = 0
	s.cache.Reset()
	return nil
}

// GetStats 获取统计数据
func (s *fastcacheImpl) GetStats() Stats {
	var stats fastcache.Stats
	s.cache.UpdateStats(&stats)
	misses := int64(stats.Misses) + s.misses
	hits := int64(stats.GetCalls) - misses
	return Stats{
		Hits:    hits,
		Misses:  misses,
		HitRate: float64(hits) / float64(stats.GetCalls),
		Loads:   s.loads,
	}
}

func wrapEntry(timestamp uint64, value []byte) []byte {
	entry := make([]byte, timestampSizeInBytes+len(value))
	binary.LittleEndian.PutUint64(entry, timestamp)
	copy(entry[timestampSizeInBytes:], value)
	return entry
}

func readEntry(entry []byte) (timestamp uint64, value []byte) {
	timestamp = binary.LittleEndian.Uint64(entry)
	// copy on read
	value = make([]byte, len(entry)-timestampSizeInBytes)
	copy(value, entry[timestampSizeInBytes:])
	return timestamp, value
}
