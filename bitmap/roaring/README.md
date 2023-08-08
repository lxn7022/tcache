# 支持在redis使用的roaring-bitmap

### 背景

在视频业务的不少场景中，都会涉及存储某个用户的某种状态。比如典型的一起看活动预约场景，就需要存储某个用户(vuid维度)是否预约了某场活动。这种场景下显然使用位图是更加合适的选择。而redis自带的[位图](https://redis.io/docs/data-types/bitmaps/)存在诸多缺点，比如限制最大512M，也就限制了最大表示的数就是2^32-1。而视频vuid是int64类型的数，显然存在表示不全的问题。除此之外，redis自带的位图实现，会导致仅设置一个值2^32-1的时候，也会扩容到512M，这显然非常浪费空间。[redis-roaring](https://github.com/aviggiano/redis-roaring)作为redis可以加载的拓展模块，可以很好的解决上述redis原生位图存在的问题。但是目前腾讯云提供的redis，均不支持加载redis-roaring模块。出于以上考虑，考虑使用lua实现一个roaring bitmap，结合redis数据结构特性，方便大家使用。

### 目前解决上述问题的一些可选方案
- 使用DB存储。缺点：性能低，无法满足高并发场景需求
- 使用redis原生位图。缺点：浪费内存；有最大数限制
- 本地使用[roaring-bitmap](https://github.com/RoaringBitmap/roaring)，序列化后存储。缺点：需要配合加锁使用;数据量大时，增加/删除/序列化/反序列化均非常耗时。具体耗时可以参考下面的性能对比
- 本地使用[roaring-bitmap](https://github.com/RoaringBitmap/roaring)，异步批量处理。缺点：需保证单线程进行；设计复杂度高；非实时，数据有延迟
- ......

### 我们的目标
- 功能完善：使用redis lua环境支持的功能和模块，结合redis数据结构，实现roaring bitmap的基本功能
- 使用方便：封装redis接口，屏蔽实现细节，对外提供简便的调用方式
- 高可用性：提供高性能，无锁，实时的操作；提供最大2^64-1数据范围

关于更多bitmap和roaring的技术细节，可以参考：关于更多bitmap的技术细节原理，参考：https://km.woa.com/articles/show/569551

关于roaring bitmap源码分析，可以参考：https://doc.weixin.qq.com/doc/w3_AHYAlgZeAFIWKJDFrV6To0nkx0jCg?scode=AJEAIQdfAAoNQy6j31AHYAlgZeAFI

### 设计思路
**Map-based 64-bit roaring-bitmap**: 基于map + roaring-bitmap32。

由于需要支持最大2^64-1的数范围，因此考虑分片组合多个32位roaring-bitmap的方式。即当设置一个数x时，首先取高32位作为索引值，映射一个32位的roaring-bitmap，低32位作为该位图的输入。该32位的roaring-bitmap按照标准设计，取入参的高16位作为容器索引值，低16位存储到具体容器（array_container或者bitmap_container）。存储container时，需要序列化后存储。为了快速计算位图中的总元素，考虑设置多一个key。每当成功add或remove的时候，相应的值+1或-1。

初始设计：
![](https://vip.image.video.qpic.cn/vupload/20230327/9edafd1679925604336.jpg)


由于redis每次操作都需要执行序列化和反序列化，会增加redis的cpu负载，故改进为使用redis原始数据结构，避免序列化和反序列化。
![](https://vip.image.video.qpic.cn/vupload/20230412/e2eaac1681302341208.jpg)

### 如何使用
该模块的使用方式非常简单，参考如下示例。

```go
package main

import (
	"git.code.oa.com/trpc-go/trpc-database/redis"
	"git.code.oa.com/trpc-go/trpc-go"
	"git.code.oa.com/trpc-go/trpc-go/client"
	"git.code.oa.com/video_pay_root/pay-go-comm/tcache/bitmap/roaring"
)

func main() {
	clientProxy := redis.NewClientProxy("trpc.redis.xxx.xxx",
		client.WithNamespace("Production"),
		client.WithTarget("xxx"),
		client.WithPassword("xxx"),
		client.WithTimeout(200*time.Millisecond),
	)

	ctx := trpc.BackgroundContext()
	bitmapAPI := roaring.NewRoaring64("bitmap1", clientProxy)

    // 设置元素
    bitmapAPI.Add(ctx, 1234567899)

    // 判断存在性
    bitmapAPI.Contains(ctx, 1234567899)

    // 获取元素总量
    bitmapAPI.Len(ctx)

    // 删除元素
    bitmapAPI.Remove(ctx, 1234567899)

    // 判断是否为空
    bitmapAPI.IsEmpty(ctx)
}

```

### 性能测试
因为该[实现](./roaring.lua)最终需要运行到redis的lua沙箱环境中，会涉及redis命令的调用，故无法与[Golang实现版本](https://github.com/aviggiano/redis-roaring)做性能对比。考虑实现[另一个版本](./roaring_test.lua)专门用作性能测试，该版本中使用本地操作模拟了需要调用的redis命令，因此一定程度上可以与[Golang实现版本](https://github.com/aviggiano/redis-roaring)做一个本地性能对比。

由于lua没有很好的性能测试框架，故使用原始的比较方法：即执行**相同的命令相同次数**，计算执行总耗时、内存消耗和平均耗时，从这几个维度来初步进行性能对比。其次，根据我们解决问题的场景限定，综合对比两个版本解决方案的性能差距。

#### 本地单机性能测试
以下为总体测试数据，主要为不同随机元素个数下各个接口的耗时和平均耗时。其中，'no need'说明其实现不涉及该操作。元素数目太少不便于观察，因此选择了50000/100000/200000/500000/700000/1000000作为梯度。
![](https://vip.image.video.qpic.cn/vupload/20230327/6becbf1679973744032.png)

##### **Add**
从图表中可以看到，随着随机元素个数的增加，lua版本实现的roaring-bitmap基本保持不变的平均耗时，而golang版本则随着元素个数增加，耗时不断增大。且当元素数量超过200000时，golang版本平均耗时显著提升。
![](https://vip.image.video.qpic.cn/vupload/20230327/fde6931679974125155.png)

造成该结果的原因，查看golang版本源码实现，可以发现golang版本中并没有使用map来存储32位roaring-bitmap中高16位索引和低16位container间的映射。取而代之的是，通过存储有序keys和containers(下图中Bitmap的两个属性)，来保证其映射关系。

![](https://vip.image.video.qpic.cn/vupload/20230327/33c3af1679975452790.png)

这导致了如果每次add进来的元素位置如果在keys中间的时候，此时为了保证有序，会造成 keys,containers和copyOnWrite标记的三重内存拷贝。而像我们的测试用例中，key分布比较随机，所以基本上都需要进行内存拷贝，因此性能比用map实现低。lua使用map实现，只会在map扩容的时候进行一次拷贝。
![](https://vip.image.video.qpic.cn/vupload/20230327/4c8a151679975817081.png)

但golang这种实现，如果数据连续性比较好，并且是从小到大的插入，由于插入的key都会落到有序数组的最后一个元素，无需进行内存拷贝，此时性能就很好。这种实现方法搭配上[RoaringFormatSpec](https://github.com/RoaringBitmap/RoaringFormatSpec)序列化方法，还可以实现在序列化的数据上完成contains/len操作。

##### **Contains**
从图标中可以看出，随着元素个数增加，两种版本实现的roaring-bitmap基本保持contains方法平均耗时的稳定。其中，golang版本实现平均耗时均低于lua版本。但究其原因，其实是lua实现比golang多了一个反序列化的操作。因为lua实现数据需要从redis中取，取出来的数据需要进行一步**反序列化**才能使用，因此lua和golang版本差距就是lua需要多执行一个反序列化操作。
![](https://vip.image.video.qpic.cn/vupload/20230327/8bd13b1679974812790.png)

##### **Remove**
Remove方法结果和分析都和Add方法类似，这里不做赘述。
![](https://vip.image.video.qpic.cn/vupload/20230328/268f7a1679976929106.png)

##### **Len**
由于lua版本采用了单独的key存储总元素数量，因此在获取总元素时基本无耗时。而golang版本需要遍历所有容器，将元素数目相加才能得到最终答案，因此存在一定耗时。
![](https://vip.image.video.qpic.cn/vupload/20230328/2741b11679977453158.png)

##### **Serialize**/**Deserialize**
lua版本需要结合redis使用，因此每次**add**和**remove**操作都会执行反序列化->修改->序列化的操作。因此不需要单独序列化或者反序列化整个bitmap的操作，也就是说lua版本序列化的粒度是container，golang版本序列化的粒度是bitmap。序列化一般是性能瓶颈所在，降低序列化和反序列化的粒度，能有效提高位图性能。

![](https://vip.image.video.qpic.cn/vupload/20230328/34da431679981430543.png)

从上图可以看出，随着元素个数的增加，序列化和反序列化的耗时不断增长。当位图中有1000000个元素时，序列化+反序列化用了接近1s的时间。再次可以看出，**大位图序列化/反序列化是性能的瓶颈。**

##### 内存占用

lua和golang版本实现类似，内存占用大致相同，具体内存占用以golang版本为例，如下：
![](https://vip.image.video.qpic.cn/vupload/20230328/3021a31679984195426.png)

#### 限定场景综合性能分析
结合具体场景，分析lua和golang版本roring解决方案。场景：支持分布式环境下的roaring使用。
1. 使用golang本地版 + 锁 + 序列化/反序列化 + redis存储的方式实现。
2. 使用该redis-lua roaring实现。

在场景1中，我们继续使用golang版roaring，存储则选择redis单key存储整个bitmap序列化值。由于可能存在并发覆盖问题，因此该场景中还需要配合分布式锁使用。这种情况下，单次add操作流程如下：加锁->redis GET->反序列化->add->序列化->redis SET->释放锁。因此单个add的耗时=2次redis操作+add+serialize+desrialize，按照上面的结论，百万key的时候这耗时已经超过1s。同时，抢锁失败也是一个问题。因此当元素数量多，并发度大的时候，这种方案已经完全不可用了。

而考虑场景2，使用我们的redis-lua roaring。整个过程封装到lua脚本中，利用redis单线程代替了加锁操作，保证了并发安全。而底层数据结构使用redis支持的数据结构，序列化和反序列化也做到了最小粒度，而且无论元素数量多大，理论上耗时不会增加。

结论：使用该redis-lua roaring，理论上可以替代其他本地位图+redis存储的方式。

#### 性能
性能根据单机压测数据来看。对于普通1G单机版meepo Redis，主调方大概**1WQPS**时，达到redis的参考性能5WQPS。
![](https://vip.image.video.qpic.cn/vupload/20230412/bc95121681292911523.png)
![](https://vip.image.video.qpic.cn/vupload/20230412/69982e1681292921843.png)

#### 使用建议
该实现提供了64位和32位两个版本，不关心性能问题可以直接使用64位版本，关心性能问题的可以自己客户端分桶使用32位版本。

### 不足
- 该设计未采用分redis节点存储的方式，即同一个key所有数据均存在同一个节点（根据hash确定），因此可能存在热key的情况。
- 该roaring-bitmap实现未实现run_container，数据极度离散的情况下并没有做存储优化。
- 还未实现批量操作，如有批量操作需求，可以提issue后续补充上去。

### TODO 
目前实现中使用了个set来存储映射key，目的是方便清除，但会消耗额外内存。后续可能无需存储这个set，通过客户端scan+unlink的方式来清理位图。