// Package bitmap 封装常用bitmap库，提供统一访问接口。
package bitmap

import (
	"fmt"
	"io"
	"sync"
)

// bitmap引擎类型定义
const (
	EngineBitset  = "bitset"
	EngineRoaring = "roaring"
)

// Constructor bitmap实例构造函数
type Constructor func() (API, error)

// API 对外接口
type API interface {
	// Add 将整数x添加到位图
	Add(x uint64)
	// Remove 将整数x从位图移除
	Remove(x uint64)
	// Contains 整数x是否包含在位图
	Contains(x uint64) bool
	// IsEmpty 判断位图是否为空
	IsEmpty() bool
	// Len 返回位图中存储的元素个数
	Len() uint64
	// WriteTo 将位图写到一个流里面
	WriteTo(stream io.Writer) (int64, error)
	// ReadFrom 从流里面加载一个位图
	ReadFrom(stream io.Reader) (int64, error)
	// Clear 清空位图
	Clear()
}

var (
	lock          = sync.RWMutex{}
	bitmapEngines = make(map[string]Constructor)
)

// New 构造函数
func New(engine string) (API, error) {
	if f := getConstructor(engine); f != nil {
		return f()
	}
	return nil, fmt.Errorf("no such engine:%v", engine)
}

// Register 自定义注册一个新的bitmap构造函数，允许在bitset/roaring之外自定义扩展
func Register(engine string, newFunc Constructor) {
	lock.Lock()
	defer lock.Unlock()
	bitmapEngines[engine] = newFunc
}

// getConstructor 根据bitmap引擎名返回bitmap构造函数
func getConstructor(engine string) Constructor {
	lock.RLock()
	f := bitmapEngines[engine]
	lock.RUnlock()
	return f
}
