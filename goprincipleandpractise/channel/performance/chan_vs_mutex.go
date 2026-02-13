package performance

import (
	"sync"
	"sync/atomic"
)

var counterSink int64

// ------------------- Pingpong（双向通信延迟） -------------------

// PingpongChannel 使用两个 channel 在两个 goroutine 间来回传递值
// 测量单次 send+receive 往返延迟
func PingpongChannel(n int) {
	ping := make(chan struct{})
	pong := make(chan struct{})

	go func() {
		for range ping {
			pong <- struct{}{}
		}
	}()

	for range n {
		ping <- struct{}{}
		<-pong
	}
	close(ping)
}

// PingpongMutexCond 使用 Mutex+Cond 模拟 pingpong
func PingpongMutexCond(n int) {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	turn := 0 // 0=ping, 1=pong

	go func() {
		mu.Lock()
		defer mu.Unlock()
		for range n {
			for turn != 0 {
				cond.Wait()
			}
			turn = 1
			cond.Signal()
		}
	}()

	mu.Lock()
	for range n {
		for turn != 1 {
			cond.Wait()
		}
		turn = 0
		cond.Signal()
	}
	mu.Unlock()
}

// ------------------- 扇入计数器（N goroutine 竞争写入） -------------------

// ContendedCounterChannel 使用 channel 收集增量，单 goroutine 聚合
func ContendedCounterChannel(workers, opsPerWorker int) int64 {
	ch := make(chan int64, 64)
	var wg sync.WaitGroup
	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()
			for range opsPerWorker {
				ch <- 1
			}
		}()
	}

	var counter int64
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(ch)
	}()
	go func() {
		for v := range ch {
			counter += v
		}
		close(done)
	}()

	<-done
	return counter
}

// ContendedCounterMutex 使用 Mutex 保护共享计数器
func ContendedCounterMutex(workers, opsPerWorker int) int64 {
	var mu sync.Mutex
	var counter int64
	var wg sync.WaitGroup
	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()
			for range opsPerWorker {
				mu.Lock()
				counter++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return counter
}

// ContendedCounterAtomic 使用 atomic 操作计数器
func ContendedCounterAtomic(workers, opsPerWorker int) int64 {
	var counter atomic.Int64
	var wg sync.WaitGroup
	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()
			for range opsPerWorker {
				counter.Add(1)
			}
		}()
	}

	wg.Wait()
	return counter.Load()
}

// ------------------- 吞吐量（单生产者→单消费者） -------------------

// ThroughputChannel 使用 channel 传递数据
func ThroughputChannel(n int) int64 {
	ch := make(chan int64, 64)
	go func() {
		for i := range n {
			ch <- int64(i)
		}
		close(ch)
	}()

	var sum int64
	for v := range ch {
		sum += v
	}
	return sum
}

// ThroughputMutexQueue 使用 Mutex + 切片模拟队列传递数据
func ThroughputMutexQueue(n int) int64 {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	queue := make([]int64, 0, 64)
	done := false

	go func() {
		for i := range n {
			mu.Lock()
			queue = append(queue, int64(i))
			mu.Unlock()
			cond.Signal()
		}
		mu.Lock()
		done = true
		mu.Unlock()
		cond.Signal()
	}()

	var sum int64
	mu.Lock()
	for {
		for len(queue) == 0 && !done {
			cond.Wait()
		}
		if len(queue) > 0 {
			v := queue[0]
			queue = queue[1:]
			mu.Unlock()
			sum += v
			mu.Lock()
			continue
		}
		if done && len(queue) == 0 {
			break
		}
	}
	mu.Unlock()
	return sum
}
