package roaring

import (
	"context"
	"embed"
	"fmt"

	"git.code.oa.com/trpc-go/trpc-database/redis"
)

// operation
type operation string

// redis操作枚举
const (
	Add      operation = "Add"
	Remove   operation = "Remove"
	Contains operation = "Contains"
	IsEmpty  operation = "IsEmpty"
	Len      operation = "Len"
	Clear    operation = "Clear"
)

// 无意义，目的为了能导入'embed'包
// 关于embed用法，参考https://pkg.go.dev/embed
var _ embed.FS

//go:embed roaring64.lua
var roaring64Script string

type roaring64Impl struct {
	name   string
	proxy  redis.Client
	script *redis.Script
}

// NewRoaring64 新建使用lua的redis bitmap实现。
// 注意：name用于底层redis存储对应数据，不同bitmap使用不同的name，name相同认为是同一个bitmap。
func NewRoaring64(name string, proxy redis.Client) RB64API {
	return &roaring64Impl{
		name:   name,
		proxy:  proxy,
		script: redis.NewScript(1, roaring64Script),
	}
}

// Add 将整数x添加到位图
func (r *roaring64Impl) Add(ctx context.Context, x uint64) error {
	_, err := r.script.Do(ctx, r.proxy, r.getKeyName(), Add, x)
	return err
}

// Remove 将整数x从位图移除
func (r *roaring64Impl) Remove(ctx context.Context, x uint64) error {
	_, err := r.script.Do(ctx, r.proxy, r.getKeyName(), Remove, x)
	return err
}

// Contains 整数x是否包含在位图
func (r *roaring64Impl) Contains(ctx context.Context, x uint64) (bool, error) {
	return redis.Bool(r.script.Do(ctx, r.proxy, r.getKeyName(), Contains, x))
}

// IsEmpty 判断位图是否为空
func (r *roaring64Impl) IsEmpty(ctx context.Context) (bool, error) {
	return redis.Bool(r.script.Do(ctx, r.proxy, r.getKeyName(), IsEmpty))
}

// Len 返回位图中存储的元素个数
func (r *roaring64Impl) Len(ctx context.Context) (uint64, error) {
	return redis.Uint64(r.script.Do(ctx, r.proxy, r.getKeyName(), Len))
}

// Clear 清空位图
func (r *roaring64Impl) Clear(ctx context.Context) error {
	_, err := r.script.Do(ctx, r.proxy, r.getKeyName(), Clear)
	return err
}

func (r *roaring64Impl) getKeyName() string {
	return fmt.Sprintf("{%s}", r.name)
}
