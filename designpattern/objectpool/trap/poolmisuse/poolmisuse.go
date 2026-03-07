package poolmisuse

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
)

// ❌ 错误示例 1：在 sync.Pool 中存值类型而非指针
//
// sync.Pool 的 Get/Put 通过 any（interface{}）传递对象。
// 存值类型时，每次 Put 会将值拷贝到 interface{} 并逃逸到堆上，
// 每次 Get 又要做一次类型断言拷贝——完全抵消了对象池减少分配的意义。
//
// 正确做法：始终存指针。return &bytes.Buffer{} 或 return new(bytes.Buffer)。
func ValueInPool() {
	pool := sync.Pool{
		New: func() any {
			return bytes.Buffer{} // ❌ 值类型：interface{} 装箱导致堆分配
		},
	}

	buf := pool.Get().(bytes.Buffer) // 类型断言拷贝
	buf.WriteString("hello")
	pool.Put(buf) // 再次装箱拷贝到堆
	// 每次 Get+Put 至少 2 次堆分配，池形同虚设

	_ = buf
}

// ❌ 错误示例 2：依赖 sync.Pool 中的对象在 GC 后仍然存在
//
// sync.Pool 在每次 GC 时清除 victim cache、将当前池降级为 victim。
// 两轮 GC 后池中对象全部消失。不能依赖 Pool 保持对象的持久性。
//
// 正确做法：如果需要持久化资源（如数据库连接），使用有界资源池
// （channel + slice，或 database/sql 的内置连接池），不要用 sync.Pool。
func PoolSurvivesGC() int {
	pool := sync.Pool{
		New: func() any { return new(bytes.Buffer) },
	}

	for range 10 {
		pool.Put(new(bytes.Buffer))
	}

	runtime.GC() // 第一轮：当前池 → victim cache
	runtime.GC() // 第二轮：清空 victim cache

	// 此时池已为空，Get 全部走 New 重建——没有任何复用
	newCreated := 0
	for range 10 {
		buf := pool.Get().(*bytes.Buffer)
		if buf.Cap() == 0 { // New 创建的空 buffer
			newCreated++
		}
	}
	return newCreated // 接近 10：几乎全部重建
}

// ❌ 错误示例 3：归还脏对象——不 Reset 就 Put
//
// 下一次 Get 拿到的 buffer 仍包含上一次写入的数据。
// 在高并发场景下可能导致数据泄漏（前一个请求的敏感数据被后一个请求读到）。
//
// 正确做法：Get 时 Reset（防御性清理），或 Put 前 Reset。
// 本项目的 bufferpool.Get() 在返回前自动调用 Reset。
func DirtyPut() string {
	pool := sync.Pool{
		New: func() any { return new(bytes.Buffer) },
	}

	buf := pool.Get().(*bytes.Buffer)
	buf.WriteString("secret-token-12345")
	pool.Put(buf) // ❌ 没有 Reset！

	leaked := pool.Get().(*bytes.Buffer)
	return leaked.String() // 可能返回 "secret-token-12345"——数据泄漏
}

// ❌ 错误示例 4：用 sync.Pool 做连接池
//
// sync.Pool 不适合管理需要生命周期的资源（如数据库连接、TCP 连接）：
//   - GC 随时清空池，连接会被丢弃——无法保持 min idle
//   - 没有容量上限——高并发时可能创建过多连接耗尽服务器资源
//   - 没有健康检查——取出的连接可能已经断开
//   - 没有 Close——连接泄漏
//
// 正确做法：使用 database/sql.DB（内置连接池）、自定义有界资源池，
// 或 workerpool 模式管理有限资源。
func SyncPoolAsConnPool() {
	type Conn struct{ id int }
	nextID := 0

	pool := sync.Pool{
		New: func() any {
			nextID++
			fmt.Printf("  创建新连接 #%d\n", nextID) // 每次 GC 后都会重建
			return &Conn{id: nextID}
		},
	}

	// 模拟使用
	for i := 0; i < 5; i++ {
		c := pool.Get().(*Conn)
		fmt.Printf("  使用连接 #%d\n", c.id)
		pool.Put(c)
	}

	runtime.GC() // GC 清空池

	// GC 后重新获取——全部是新建的连接，之前的连接丢失（未 Close）
	for i := 0; i < 3; i++ {
		c := pool.Get().(*Conn)
		fmt.Printf("  GC 后使用连接 #%d（新建）\n", c.id)
		// 旧连接去哪了？没有 Close，资源泄漏。
	}
}
