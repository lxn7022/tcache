package bitmap

import (
	"io"

	"github.com/RoaringBitmap/roaring/roaring64"
)

func init() {
	Register(EngineRoaring, newRoaring)
}

type roaring struct {
	rb64 *roaring64.Bitmap
}

func newRoaring() (API, error) {
	return &roaring{roaring64.New()}, nil
}

// Add 将整数x添加到位图
func (r *roaring) Add(x uint64) {
	r.rb64.Add(x)
}

// Remove 将整数x从位图移除
func (r *roaring) Remove(x uint64) {
	r.rb64.Remove(x)
}

// Contains 整数x是否包含在位图
func (r *roaring) Contains(x uint64) bool {
	return r.rb64.Contains(x)
}

// IsEmpty 判断位图是否为空
func (r *roaring) IsEmpty() bool {
	return r.rb64.IsEmpty()
}

// Len 返回位图中存储的元素个数
func (r *roaring) Len() uint64 {
	return r.rb64.GetCardinality()
}

// WriteTo 将位图写到一个流里面
func (r *roaring) WriteTo(stream io.Writer) (int64, error) {
	return r.rb64.WriteTo(stream)
}

// ReadFrom 从流里面加载一个位图
func (r *roaring) ReadFrom(stream io.Reader) (int64, error) {
	return r.rb64.ReadFrom(stream)
}

// Clear 清空位图
func (r *roaring) Clear() {
	r.rb64.Clear()
}
