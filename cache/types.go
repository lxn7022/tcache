package cache

import (
	"errors"
	"sync/atomic"

	jsoniter "github.com/json-iterator/go"
)

// cache引擎类型定义
const (
	EngineBigCache   = "bigcache"
	EngineFreeCache  = "freecache"
	EngineFastCache  = "fastcache"
	EngineLocalCache = "localcache"
)

// 错误码定义
var (
	ErrNotFound     = errors.New("not found")      // key不存在
	ErrNoSuchEngine = errors.New("no such engine") // 没有该cache引擎
	ErrNotSupported = errors.New("not supported")  // 不支持的特性
	ErrCannotSet    = errors.New("cannot set")     // Get接口传入的值不支持赋值
)

// Stats 统计数据
type Stats struct {
	Hits    int64   `json:"hits"`       // 命中次数
	Misses  int64   `json:"misses"`     // 未命中次数
	HitRate float64 `json:"hit_rate"`   // 命中率,0~1
	Loads   int64   `json:"auto_loads"` // 自动加载次数
}

func (s *Stats) atomicAddHitCount() {
	atomic.AddInt64(&s.Hits, 1)
}

func (s *Stats) atomicAddMissCount() {
	atomic.AddInt64(&s.Misses, 1)
}

func (s *Stats) atomicAddLoadCount() {
	atomic.AddInt64(&s.Loads, 1)
}

func (s *Stats) calcHitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

// Config 参数选项
type Config struct {
	MaxSizeInMB int        // cache占用的最大内存，单位MB
	DefaultTTL  int64      // 未单独设置key过期时，默认过期时间，单位秒，0代表不限制
	Serializer  Serializer // 序列化/反序列化实例
}

// MaxSizeInBytes cache占用的最大内存，单位字节
func (c *Config) MaxSizeInBytes() int {
	return c.MaxSizeInMB * 1024 * 1024
}

func (c *Config) fixEmpty() {
	if c.Serializer == nil {
		c.Serializer = jsoniter.ConfigCompatibleWithStandardLibrary
	}
}

// sizeToCapacity 初始化时指定的是缓存的大小。有些存储引擎使用的是key数量，因此按照一个key
// 100字节来估算，将字节大小转化为key容量
func (c *Config) sizeToCapacity() int {
	return c.MaxSizeInBytes() / 100
}

// Option 设置参数选项
type Option func(*Config)

// WithDefaultTTL 指定key默认过期时间，单位秒，0代表不限制
func WithDefaultTTL(ttl int64) Option {
	return func(c *Config) {
		c.DefaultTTL = ttl
	}
}

// WithSerializer 指定序列化/反序列化实例
func WithSerializer(s Serializer) Option {
	return func(c *Config) {
		c.Serializer = s
	}
}
