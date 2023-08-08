package roaring

import (
	"context"
)

// RB64API 64位roaring对外接口
type RB64API interface {
	// Add 将整数x添加到位图
	Add(ctx context.Context, x uint64) error
	// Remove 将整数x从位图移除
	Remove(ctx context.Context, x uint64) error
	// Contains 整数x是否包含在位图
	Contains(ctx context.Context, x uint64) (bool, error)
	// IsEmpty 判断位图是否为空
	IsEmpty(ctx context.Context) (bool, error)
	// Len 返回位图中存储的元素个数
	Len(ctx context.Context) (uint64, error)
	// Clear 清空位图
	Clear(ctx context.Context) error
}

// RB32API 32位roaring对外接口
type RB32API interface {
	// Add 将整数x添加到位图
	Add(ctx context.Context, x uint32) error
	// Remove 将整数x从位图移除
	Remove(ctx context.Context, x uint32) error
	// Contains 整数x是否包含在位图
	Contains(ctx context.Context, x uint32) (bool, error)
	// IsEmpty 判断位图是否为空
	IsEmpty(ctx context.Context) (bool, error)
	// Len 返回位图中存储的元素个数
	Len(ctx context.Context) (uint32, error)
	// Clear 清空位图
	Clear(ctx context.Context) error
}
