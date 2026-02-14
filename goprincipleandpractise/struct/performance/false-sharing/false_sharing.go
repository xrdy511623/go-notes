package falsesharing

import "sync/atomic"

// False sharing（伪共享）性能影响
//
// 现代 CPU 以 cache line（通常 64 字节）为单位在核心间同步缓存。
// 当两个 goroutine 在不同核心上修改同一 cache line 中的不同变量时，
// 即使逻辑上没有数据竞争，硬件也必须不断使对方的缓存失效，
// 导致性能严重下降——这就是 false sharing。

const CacheLineSize = 64

// NoPadCounter 两个计数器紧挨着，可能落在同一 cache line
type NoPadCounter struct {
	A atomic.Int64
	B atomic.Int64
}

// PaddedCounter 在两个计数器之间插入 padding，确保各占独立的 cache line
type PaddedCounter struct {
	A atomic.Int64
	_ [CacheLineSize - 8]byte // padding: 确保 A 和 B 不在同一 cache line
	B atomic.Int64
}

// IncrementNoPad 两个 goroutine 分别递增 A 和 B（无 padding）
func IncrementNoPad(c *NoPadCounter, n int) {
	done := make(chan struct{})
	go func() {
		for range n {
			c.A.Add(1)
		}
		done <- struct{}{}
	}()
	go func() {
		for range n {
			c.B.Add(1)
		}
		done <- struct{}{}
	}()
	<-done
	<-done
}

// IncrementPadded 两个 goroutine 分别递增 A 和 B（有 padding）
func IncrementPadded(c *PaddedCounter, n int) {
	done := make(chan struct{})
	go func() {
		for range n {
			c.A.Add(1)
		}
		done <- struct{}{}
	}()
	go func() {
		for range n {
			c.B.Add(1)
		}
		done <- struct{}{}
	}()
	<-done
	<-done
}
