# bitmap: bitmap组件统一接口封装

## 本地版bitmap


封装思路如下：

1. `bitmap`组件分为"前端"和"后端"两部分：
    - 前端定义统一的对外接口，屏蔽`bitmap`组件的接口差异
    - 后端利用不同的`bitmap`组件，对前端统一接口进行实现，方便扩展
2. 提供统一的uin64类型输入，支持诸如vuid、qq等超32位的数字存储。
3. 提供序列化和反序列化方法，方便存储和恢复。

### 如何使用

#### 1.使用默认bitmap实例

默认使用roaring

```go
package main

import (
    "git.code.oa.com/video_pay_root/pay-go-comm/tcache/bitmap"
)

func main() {
    // 设置位图
    bitmap.Add(1)
    bitmap.Add(2)

    fmt.Println(bitmap.Contains(1)) // return true 
    fmt.Println(bitmap.Contains(2)) // return true 
    fmt.Println(bitmap.Contains(3)) // return false 
    fmt.Println(bitmap.Len()) // return 2

    bitmap.Remove(1)
    bitmap.Remove(2)

    fmt.Println(bitmap.IsEmpty()) // return true 

    // write to stream
    buf := &bytes.Buffer{}
    if _, err := bitmap.WriteTo(buf); err != nil {
        panic(err)
    }

    // load from stream 
    if _, err := bitmap.ReadFrom(buf); err != nil {
        panic(err)
    }

    // clear all bitmap
    bitmap.Clear()
}
```

#### 2.自主选择bitmap实例

```go
package main

import (
    "git.code.oa.com/video_pay_root/pay-go-comm/tcache/bitmap"
)

func main() {
    // 选择fastcache引擎，最大内存限制500MB
    b, err := bitmap.New(tcache.EngineRoaring)
    if err != nil {
        panic(err)
    }

    // 使用方法同上
    b.Add(1)
    b.Add(2)
    ......
}
```

#### 3.扩展并注册bitmap实例

可以业务自己注册bitmap引擎，参考[bitset.go](./bitset.go)的实现
