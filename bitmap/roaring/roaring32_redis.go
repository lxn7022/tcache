package roaring

import (
	"context"
	"embed"
	"fmt"

	"git.code.oa.com/trpc-go/trpc-database/redis"
)

var _ embed.FS

//go:embed roaring32.lua
var roaring32Script string

type roaring32Impl struct {
	name   string
	proxy  redis.Client
	script *redis.Script
}

// NewRoaring32 新建使用lua的redis bitmap实现。
// 注意：name用于底层redis存储对应数据，不同bitmap使用不同的name，name相同认为是同一个bitmap。
func NewRoaring32(name string, proxy redis.Client) RB32API {
	return &roaring32Impl{
		name:   name,
		proxy:  proxy,
		script: redis.NewScript(1, roaring32Script),
	}
}

// Add 将整数x添加到位图
func (r *roaring32Impl) Add(ctx context.Context, x uint32) error {
	_, err := r.script.Do(ctx, r.proxy, r.getKeyName(), Add, x)
	return err
}

// Remove 将整数x从位图移除
func (r *roaring32Impl) Remove(ctx context.Context, x uint32) error {
	_, err := r.script.Do(ctx, r.proxy, r.getKeyName(), Remove, x)
	return err
}

// Contains 整数x是否包含在位图
func (r *roaring32Impl) Contains(ctx context.Context, x uint32) (bool, error) {
	return redis.Bool(r.script.Do(ctx, r.proxy, r.getKeyName(), Contains, x))
}

// IsEmpty 判断位图是否为空
func (r *roaring32Impl) IsEmpty(ctx context.Context) (bool, error) {
	return redis.Bool(r.script.Do(ctx, r.proxy, r.getKeyName(), IsEmpty))
}

// Len 返回位图中存储的元素个数
func (r *roaring32Impl) Len(ctx context.Context) (uint32, error) {
	total, err := redis.Uint64(r.script.Do(ctx, r.proxy, r.getKeyName(), Len))
	return uint32(total), err
}

// Clear 清空位图
func (r *roaring32Impl) Clear(ctx context.Context) error {
	_, err := r.script.Do(ctx, r.proxy, r.getKeyName(), Clear)
	return err
}

func (r *roaring32Impl) getKeyName() string {
	return fmt.Sprintf("{%s}", r.name)
}
