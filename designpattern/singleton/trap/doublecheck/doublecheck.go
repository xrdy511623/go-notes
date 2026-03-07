package doublecheck

import "sync"

// ❌ 错误示例：手动实现双重检查锁（Double-Checked Locking）
//
// 这是从 Java/C++ 移植过来的经典反模式。在 Go 中存在数据竞争（data race）：
//
//   goroutine A 正在执行 instance = &Config{...}（写入）
//   goroutine B 同时执行 if instance == nil（读取）
//
// 外层 if instance == nil 是无锁读取，与 Lock 内部的写操作构成 data race。
// Go memory model 不保证 goroutine B 能看到 goroutine A 写入的完整 *Config。
// 用 go test -race 可以检测到这个问题。
//
// 正确做法：使用 sync.Once，它内部使用 atomic 保证了正确的 happens-before 语义。

type Config struct {
	Name string
}

var (
	instance *Config
	mu       sync.Mutex
)

func Instance() *Config {
	if instance == nil { // ← DATA RACE: 无保护的读取
		mu.Lock()
		defer mu.Unlock()
		if instance == nil {
			instance = &Config{Name: "default"}
		}
	}
	return instance
}
