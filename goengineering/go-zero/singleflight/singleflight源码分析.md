
---
singleflight源码分析
---

源码位置: core/syncx/singleflight.go。

在go-zero中SingleFlight的作用是：将多个并发请求合并成一个请求，以减少对下层服务的压力。

# 1 应用场景

SingleFlight 适用于:
热点数据查询:大量并发请求同一个资源
缓存击穿防护:热点 key 过期瞬间的雪崩保护
计算密集型操作:昂贵的计算结果需要共享
外部 API 调用:高延迟的第三方服务调用

最佳实践场景:
热点商品详情:秒杀、爆款商品
首页配置/Banner:所有用户访问同一资源
外部 API 调用:汇率、天气、地图服务等高延迟接口
计算密集型报表:需要长时间计算的聚合数据
缓存击穿防护:热点 key 过期瞬间的保护

关键判断标准:
是否有大量并发请求同一个 key?
获取数据的成本是否很高(延迟>100ms 或计算密集)?
是否需要防止缓存击穿?

如果以上都是"是", 那就应该使用 SingleFlight。

SingleFlight 合并的是"时间窗口内对同一个 key 的并发请求"。
比如，缓存每5秒过期一次，如果某个热点key过期，假设第一个请请求耗时0.5s(cache miss+db query+set cache), 那么这
0.5s内同一个key的其他查询请求(假设有10万次)都会阻塞等待，直到第一个请求返回结果，也就是只有一次db查询，这样就防止了
这10万次请求都来查询db，解决了缓存击穿的问题。当然，后面4.5s内，所有的请求都会直接命中缓存，不再查询db，db零压力。


# 2 应用方式

```go
package product

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/allegro/bigcache/v3"
    "github.com/zeromicro/go-zero/core/syncx"
)

// Detail 商品详情
type Detail struct {
    ProductID   int64   `json:"productId"`
    ProductName string  `json:"productName"`
    Price       float64 `json:"price"`
    Stock       int     `json:"stock"`
    Description string  `json:"description"`
}

// Repository 数据库接口
type Repository interface {
    FindByID(ctx context.Context, productID int64) (*Detail, error)
}

// Cache 商品缓存服务(使用 SingleFlight 防止缓存击穿)
type Cache struct {
    repo        Repository
    cache       *bigcache.BigCache
    singleFlight syncx.SingleFlight // go-zero 的 SingleFlight 实现
}

// NewProductCache 创建商品缓存服务
func NewProductCache(repo Repository) (*Cache, error) {
    config := bigcache.DefaultConfig(5 * time.Minute) // 5分钟过期
    config.HardMaxCacheSize = 100 // 100MB
    
    cache, err := bigcache.New(context.Background(), config)
    if err != nil {
        return nil, err
    }
    
    return &Cache{
        repo:        repo,
        cache:       cache,
        singleFlight: syncx.NewSingleFlight(),
    }, nil
}

// GetProductByID 获取商品详情(带 SingleFlight 保护)
func (pc *Cache) GetProductByID(ctx context.Context, productID int64) (*Detail, error) {
    cacheKey := fmt.Sprintf("product:%d", productID)
    
    // 1. 尝试从缓存获取
    if cached, err := pc.cache.Get(cacheKey); err == nil {
        var product Detail
        if err := json.Unmarshal(cached, &product); err == nil {
            return &product, nil
        }
        // 缓存数据损坏,删除
        _ = pc.cache.Delete(cacheKey)
    }
    
    // 2. 缓存未命中,使用 SingleFlight 合并并发请求
    // 关键点:同一时刻,对于同一个 productID,只有一个请求会真正执行 loadProduct
    val, err := pc.singleFlight.Do(cacheKey, func() (interface{}, error) {
        return pc.loadProduct(ctx, productID, cacheKey)
    })
    
    if err != nil {
        return nil, err
    }
    
    return val.(*Detail), nil
}

// loadProduct 从数据库加载商品并缓存(私有方法,只被 SingleFlight 调用)
func (pc *Cache) loadProduct(ctx context.Context, productID int64, cacheKey string) (*Detail, error) {
    // 模拟数据库查询(可能很慢)
    product, err := pc.repo.FindByID(ctx, productID)
    if err != nil {
        return nil, fmt.Errorf("load product from db failed: %w", err)
    }
    
    // 写入缓存
    if data, err := json.Marshal(product); err == nil {
        _ = pc.cache.Set(cacheKey, data)
    }
    
    return product, nil
}

// 对比:不使用 SingleFlight 的实现
type CacheWithoutSF struct {
    repo  Repository
    cache *bigcache.BigCache
}

func (pc *CacheWithoutSF) GetProductByID(ctx context.Context, productID int64) (*Detail, error) {
    cacheKey := fmt.Sprintf("product:%d", productID)
    
    // 1. 尝试从缓存获取
    if cached, err := pc.cache.Get(cacheKey); err == nil {
        var product Detail
        if err := json.Unmarshal(cached, &product); err == nil {
            return &product, nil
        }
        _ = pc.cache.Delete(cacheKey)
    }
    
    // 2. 缓存未命中,直接查数据库
    // 问题:如果1000个并发请求同时到达,会产生1000次数据库查询!
    product, err := pc.repo.FindByID(ctx, productID)
    if err != nil {
        return nil, err
    }
    
    // 3. 写入缓存
    if data, err := json.Marshal(product); err == nil {
        _ = pc.cache.Set(cacheKey, data)
    }
    
    return product, nil
}
```

# 3 实现原理

先看代码结构：

```go
type (
    // 定义接口，有2个方法 Do 和 DoEx，其实逻辑是一样的，DoEx 多了一个fresh标识，主要看Do的逻辑就够了
	SingleFlight interface {
		Do(key string, fn func() (any, error)) (any, error)
		DoEx(key string, fn func() (any, error)) (any, bool, error)
	}
   // 定义 call 的结构
	call struct {
		wg  sync.WaitGroup  // 用于实现只调用第一个 call，其他 call 阻塞等待
		val any             // 表示call操作的返回结果
		err error           // 表示call操作发生的错误
	}
    // 定义 flightGroup 的结构，实现SingleFlight接口
	flightGroup struct {
		calls map[string]*call  // 不同的key，对应不同的call
		lock  sync.Mutex        // 利用互斥锁实现并发操作安全
	}
)
```

然后看最核心的 Do 方法做了啥

```go
func (g *flightGroup) Do(key string, fn func() (any, error)) (any, error) {
	c, done := g.createCall(key)
	if done {
		return c.val, c.err
	}

	g.makeCall(c, key, fn)
	return c.val, c.err
}
```

代码很简洁，利用g.createCall(key)对 key 发起 call 请求（其实就是做一件事情），如果此时已经有其他协程已经在发起
call 请求就阻塞住（done 为 true 的情况），等待拿到结果后直接返回。如果 done 是 false，说明当前协程是第一个发起
call 的协程，那么就执行g.makeCall(c, key, fn)真正地发起 call 请求（此后的其他协程就阻塞在了g.createCall(key))。


![singleflight.png](images%2Fsingleflight.png)


从上图可知，其实关键就两步：

判断是第一个请求的协程（利用 map）
阻塞住其他所有协程（利用 sync.WaitGroup）

来看下g.createCall(key)如何实现的：


```go
func (g *flightGroup) createCall(key string) (c *call, done bool) {
	g.lock.Lock()
	if c, ok := g.calls[key]; ok {
		g.lock.Unlock()
		c.wg.Wait()
		return c, true
	}

	c = new(call)
	c.wg.Add(1)
	g.calls[key] = c
	g.lock.Unlock()

	return c, false
}
```

先看第一步：判断是否是第一个请求的协程（利用 map）

```go
g.lock.Lock()
if c, ok := g.calls[key]; ok {
    g.lock.Unlock()
    c.wg.Wait()
    return c, true
}
```

此处判断 map 中的 key 是否存在，如果已经存在，说明不是第一个请求的协程，当前协程只需要等待，等待是利用了
sync.WaitGroup的Wait()方法实现的，此处还是很巧妙的。要注意的是，map 在 Go 中是非并发安全的，所以需要加锁。

再看第二步：阻塞住其他所有协程（利用 sync.WaitGroup）

```go
c = new(call)
c.wg.Add(1)
g.calls[key] = c
g.lock.Unlock()
```

如果map 中的 key不存在， 说明是第一个发起 call 的协程，所以需要 new 这个 call，然后将wg.Add(1)，这样就对应了上面
的wg.Wait()，阻塞剩下的协程。随后将 new 的 call 放入 map 中，注意此时只是完成了初始化，并没有真正去执行 call 请求，
真正的处理逻辑在 g.makeCall(c, key, fn)中。

```go
func (g *flightGroup) makeCall(c *call, key string, fn func() (any, error)) {
	defer func() {
		g.lock.Lock()
		delete(g.calls, key)
		g.lock.Unlock()
		c.wg.Done()
	}()

	c.val, c.err = fn()
}

```

这个方法中做的事情很简单，就是执行了传递的匿名函数fn()（也就是真正 call 请求要做的事情）。最后处理收尾的事情（通过 defer），
也是分成两步：

删除 map 中的 key，使得下次发起请求可以获取新的值。
调用wg.Done()，让之前阻塞的协程全部获得结果并返回。

# 3  总结
map 非并发安全，记得加锁。
巧用 sync.WaitGroup 去完成需要阻塞控制协程的应用场景。
通过匿名函数 fn 去封装传递具体业务逻辑，在调用 fn 的上层函数中去完成统一的逻辑处理。