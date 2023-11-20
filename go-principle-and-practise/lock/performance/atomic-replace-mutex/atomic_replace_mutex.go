package atomicreplacemutex

import (
	"sync"
	"sync/atomic"
)

type counter struct {
	i int32
}

type counterAtomic struct {
	i int32
}

type counterMutex struct {
	i int32
	m sync.Mutex
}

func add(c *counter, n int) {
	for i := 0; i < n; i++ {
		c.i++
	}
}

func addUseAtomic(c *counterAtomic, n int) {
	for i := 0; i < n; i++ {
		atomic.AddInt32(&c.i, 1)
	}
}

func addUseMutex(c *counterMutex, n int) {
	for i := 0; i < n; i++ {
		c.m.Lock()
		c.i++
		c.m.Unlock()
	}
}
