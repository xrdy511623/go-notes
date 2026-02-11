package atomicreplacemutex

import (
	"sync"
)

type counterAtomic struct {
	i int64
}

type counterMutex struct {
	i int64
	m sync.Mutex
}
