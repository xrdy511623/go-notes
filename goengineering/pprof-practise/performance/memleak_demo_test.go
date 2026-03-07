package performance

import (
	"runtime"
	"testing"
	"time"
)

// TestGoroutineLeakDemo 演示 goroutine 泄漏的检测方式
//
// 运行:
//
//	go test -v -run TestGoroutineLeakDemo ./goprincipleandpractise/pprof-practise/performance/
func TestGoroutineLeakDemo(t *testing.T) {
	before := runtime.NumGoroutine()
	t.Logf("初始 goroutine 数: %d", before)

	// 模拟泄漏: 启动 goroutine 但 channel 永远没有写入者
	ch := make(chan int)
	for i := 0; i < 10; i++ {
		go func() {
			<-ch // 永远阻塞，goroutine 无法退出
		}()
	}

	time.Sleep(100 * time.Millisecond)
	after := runtime.NumGoroutine()
	t.Logf("泄漏后 goroutine 数: %d (增加了 %d)", after, after-before)

	if after-before >= 10 {
		t.Logf("检测到 goroutine 泄漏！增加了 %d 个无法退出的 goroutine", after-before)
	}

	// 修复方式: 关闭 channel 让所有阻塞的 goroutine 退出
	close(ch)
	time.Sleep(100 * time.Millisecond)
	fixed := runtime.NumGoroutine()
	t.Logf("修复后 goroutine 数: %d", fixed)
}

// TestTimeAfterLeakDemo 演示 time.After 在循环中的泄漏问题
//
// 运行:
//
//	go test -v -run TestTimeAfterLeakDemo ./goprincipleandpractise/pprof-practise/performance/
func TestTimeAfterLeakDemo(t *testing.T) {
	// 错误方式: 循环中使用 time.After（每次创建新 timer，旧的无法被回收直到触发）
	var leakedTimers int
	ch := make(chan struct{})

	go func() {
		for i := 0; i < 100; i++ {
			select {
			case <-ch:
				return
			case <-time.After(time.Hour): // 每次循环创建一个 1 小时的 timer！
				return
			default:
				leakedTimers++
			}
		}
	}()
	time.Sleep(50 * time.Millisecond)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	t.Logf("[time.After 方式] HeapObjects: %d", m.HeapObjects)

	// 正确方式: 复用 timer
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()

	go func() {
		for i := 0; i < 100; i++ {
			select {
			case <-ch:
				return
			case <-timer.C:
				timer.Reset(time.Hour)
				return
			default:
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(time.Hour)
		}
	}()
	time.Sleep(50 * time.Millisecond)

	runtime.ReadMemStats(&m)
	t.Logf("[timer.Reset 方式] HeapObjects: %d", m.HeapObjects)

	close(ch)
	t.Log("生产环境排查方式: go tool pprof -inuse_space -base heap1.prof heap2.prof")
}
