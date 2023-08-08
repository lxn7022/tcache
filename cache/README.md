# cache: localcache组件统一接口封装

封装思路如下：

1. `localcache`组件分为"前端"和"后端"两部分：
    - 前端定义统一的对外接口，屏蔽`localcache`组件的接口差异
    - 后端利用不同的`localcache`组件，对前端统一接口进行实现，方便扩展
2. 当key不存在时，提供数据自动加载机制

### 如何使用

#### 1.使用默认cache实例

默认使用freecache

```go
package main

import (
    "git.code.oa.com/video_pay_root/pay-go-comm/tcache/cache"
)

func LoadData(ctx context.Context, key string, value interface{}) (ttl int64, err error) {
    if v, ok := value.(*string); ok {
        *v = "cat"
    }
    // 可以根据key的不同返回不同缓存时间
    return 5, nil
}

func main() {
    // 缓存key-value
    cache.Set("foo", "bar")

    // 获取key对应的value
    var value string
    cache.Get("foo", &value)

    // 获取key对应的value,如果key在cache中不存在，则使用自定义的LoadData函数从数据源加载，并缓存在cache中
    err := cache.GetWithLoad(context.TODO(), "tom", &value, LoadData)

    // 删除key
    cache.Del("foo")

    // 清空缓存
    cache.Clear()
}
```

#### 2.自主选择cache实例

```go
package main

import (
    "git.code.oa.com/video_pay_root/pay-go-comm/tcache/cache"
)

func LoadData(ctx context.Context, key string, value interface{}) (ttl int64, err error) {
    if v, ok := value.(*string); ok {
        *v = "cat"
    }
    // 可以根据key的不同返回不同缓存时间
    return 5, nil
}

func main() {
    // 选择fastcache引擎，最大内存限制500MB
    c, err := cache.New(tcache.EngineFastCache, 500)
    if err != nil {
        log.Fatal(err)
    }
    // 缓存key-value
    c.Set("foo", "bar")

    // 获取key对应的value
    var value string
    c.Get("foo", &value)

    // 获取key对应的value,如果key在cache中不存在，则使用自定义的LoadData函数从数据源加载，并缓存在cache中
    err := c.GetWithLoad(context.TODO(), "tom", &value, LoadData)

    // 删除key
    c.Del("foo")

    // 清空缓存
    c.Clear()
}
```

#### 3.扩展并注册cache实例

可以业务自己注册cache引擎，参考[fastcache.go](./fastcache.go)的实现