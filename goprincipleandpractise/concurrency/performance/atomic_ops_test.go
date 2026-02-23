package performance

import "testing"

/*
对比 atomic、类型化atomic、mutex、rwmutex 在不同竞争场景下的性能。

执行命令:

	go test -run '^$' -bench '^BenchmarkCounter' -benchtime=3s -count=5 -benchmem .

关注指标:
  - ns/op: 每次操作延迟
  - 对比 Parallel（高竞争）vs 非Parallel（无竞争）

预期结论:
 1. 无竞争时差异较小（都在个位数ns）
 2. 高竞争时 atomic ≈ 类型化atomic < RWMutex < Mutex
 3. atomic.Value 在读密集场景下远快于 RWMutex
*/

// ==================== 高竞争累加 ====================

func BenchmarkCounterAtomicParallel(b *testing.B) {
	c := &CounterAtomic{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkCounterAtomicTypedParallel(b *testing.B) {
	c := &CounterAtomicTyped{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkCounterMutexParallel(b *testing.B) {
	c := &CounterMutex{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkCounterRWMutexParallel(b *testing.B) {
	c := &CounterRWMutex{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

// ==================== 读多写少（9:1比例） ====================

func BenchmarkCounterAtomicRead9Write1(b *testing.B) {
	c := &CounterAtomic{}
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				c.Inc()
			} else {
				_ = c.Get()
			}
			i++
		}
	})
}

func BenchmarkCounterRWMutexRead9Write1(b *testing.B) {
	c := &CounterRWMutex{}
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				c.Inc()
			} else {
				_ = c.Get()
			}
			i++
		}
	})
}

func BenchmarkCounterMutexRead9Write1(b *testing.B) {
	c := &CounterMutex{}
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				c.Inc()
			} else {
				_ = c.Get()
			}
			i++
		}
	})
}

// ==================== atomic.Value vs RWMutex 配置读取 ====================

func BenchmarkConfigAtomicValueLoad(b *testing.B) {
	s := NewConfigStoreAtomic(&Config{MaxConn: 100, Timeout: 30})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = s.Load()
		}
	})
}

func BenchmarkConfigRWMutexLoad(b *testing.B) {
	s := NewConfigStoreMutex(&Config{MaxConn: 100, Timeout: 30})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = s.Load()
		}
	})
}
