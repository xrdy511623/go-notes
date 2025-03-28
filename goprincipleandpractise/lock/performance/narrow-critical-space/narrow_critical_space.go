package narrowcriticalspace

import (
	"sync"
	"time"
)

type counter struct {
	i int32
	m sync.Mutex
}

func countDefer(c *counter) {
	c.m.Lock()
	defer c.m.Unlock()
	c.i++
	// do something
	time.Sleep(time.Microsecond * 10)
}

func countNarrow(c *counter) {
	c.m.Lock()
	c.i++
	c.m.Unlock()
	// do something
	time.Sleep(time.Microsecond * 10)
}
