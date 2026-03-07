package performance

import (
	"testing"
)

/*
Channel vs Mutex vs Atomic 性能对比

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=3 -benchmem .

对比维度:
  1. Pingpong: 两个 goroutine 来回传递消息的往返延迟
     - Channel: 每次往返 = 1次 send + 1次 receive × 2
     - Mutex+Cond: 每次往返 = 1次 Lock + 1次 Wait/Signal × 2
  2. 扇入计数器: N 个 goroutine 竞争递增同一个计数器
     - Channel: goroutine → channel → 聚合 goroutine
     - Mutex: Lock/Unlock 保护共享变量
     - Atomic: 无锁 CAS 操作
  3. 吞吐量: 单生产者 → 单消费者 数据传递

结论:
  - Pingpong（通信延迟）: Channel 比 Mutex+Cond 慢约 2-3x，但语义更清晰
  - 扇入计数器（高竞争写入）: Atomic >> Mutex >> Channel
  - 吞吐量（数据传递）: Channel 与 Mutex 队列接近，Channel 代码更简洁

选型建议:
  - 需要在 goroutine 间传递数据所有权 → Channel
  - 保护共享状态的简单读写 → Mutex
  - 高频计数器/标志位 → Atomic
  - "不要通过共享内存来通信" 不意味着永远不用 Mutex，而是优先考虑 Channel
*/

// ------------------- Pingpong -------------------

func BenchmarkPingpongChannel(b *testing.B) {
	for b.Loop() {
		PingpongChannel(100)
	}
}

func BenchmarkPingpongMutexCond(b *testing.B) {
	for b.Loop() {
		PingpongMutexCond(100)
	}
}

// ------------------- 扇入计数器 -------------------

func BenchmarkContendedCounterChannel(b *testing.B) {
	for b.Loop() {
		counterSink = ContendedCounterChannel(8, 1000)
	}
}

func BenchmarkContendedCounterMutex(b *testing.B) {
	for b.Loop() {
		counterSink = ContendedCounterMutex(8, 1000)
	}
}

func BenchmarkContendedCounterAtomic(b *testing.B) {
	for b.Loop() {
		counterSink = ContendedCounterAtomic(8, 1000)
	}
}

// ------------------- 吞吐量 -------------------

func BenchmarkThroughputChannel(b *testing.B) {
	for b.Loop() {
		counterSink = ThroughputChannel(10000)
	}
}

func BenchmarkThroughputMutexQueue(b *testing.B) {
	for b.Loop() {
		counterSink = ThroughputMutexQueue(10000)
	}
}
