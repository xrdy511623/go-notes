package testwithoutrace

/*
陷阱：不使用 -race 标志运行测试

问题说明：
  Go 的 -race 标志使用 ThreadSanitizer 技术在运行时检测数据竞争。
  没有 -race 时，存在数据竞争的代码可能"碰巧"得到正确结果，
  因为竞争条件是否触发取决于 goroutine 调度时序。

  不加 -race 运行：
    $ go test -run TestCounterRace -count=5 ./...
    ok  （看起来全部通过）

  加 -race 运行：
    $ go test -race -run TestCounterRace ./...
    WARNING: DATA RACE
    FAIL

  正确做法：CI 中始终使用 go test -race ./...

  注意事项：
  - -race 使测试变慢 2-10 倍，内存增加 5-10 倍
  - 但这远低于生产环境数据竞争的修复成本
  - 推荐 CI 配置：
    env:
      GORACE: "halt_on_error=1"
    run: go test -race ./...
*/

import "sync"

// Counter 有数据竞争问题的计数器
// 在无 -race 检测时测试可能"通过"
type Counter struct {
	value int // 没有并发保护
}

// Increment 递增（有数据竞争）
func (c *Counter) Increment() {
	// read-modify-write 不是原子操作：
	// 1. 读取 c.value（可能读到过期值）
	// 2. 加 1
	// 3. 写回（可能覆盖其他 goroutine 的写入）
	c.value++
}

// Value 获取当前值（有数据竞争）
func (c *Counter) Value() int {
	return c.value
}

// SafeCounter 正确的并发安全计数器
type SafeCounter struct {
	mu    sync.Mutex
	value int
}

// Increment 安全递增
func (c *SafeCounter) Increment() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

// Value 安全获取
func (c *SafeCounter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

// RunConcurrentIncrements 并发递增
func RunConcurrentIncrements(increment func(), n int, goroutines int) {
	var wg sync.WaitGroup
	perG := n / goroutines
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perG; j++ {
				increment()
			}
		}()
	}
	wg.Wait()
}
