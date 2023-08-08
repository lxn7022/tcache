package bitmap

import (
	"io"
)

var defaultBitmap API

func init() {
	defaultBitmap, _ = newRoaring()
}

// Add 将整数x添加到位图
func Add(x uint64) {
	defaultBitmap.Add(x)
}

// Remove 将整数x从位图移除
func Remove(x uint64) {
	defaultBitmap.Remove(x)
}

// Contains 整数x是否包含在位图
func Contains(x uint64) bool {
	return defaultBitmap.Contains(x)
}

// IsEmpty 判断位图是否为空
func IsEmpty() bool {
	return defaultBitmap.IsEmpty()
}

// Len 返回位图中存储的元素个数
func Len() uint64 {
	return defaultBitmap.Len()
}

// WriteTo 将位图写到一个流里面
func WriteTo(stream io.Writer) (int64, error) {
	return defaultBitmap.WriteTo(stream)
}

// ReadFrom 从流里面加载一个位图
func ReadFrom(stream io.Reader) (int64, error) {
	return defaultBitmap.ReadFrom(stream)
}
