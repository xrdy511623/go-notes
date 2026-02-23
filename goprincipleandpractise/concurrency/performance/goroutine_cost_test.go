package performance

import (
	"sync"
	"testing"
)

/*
Goroutine创建和上下文切换的开销基准。

执行命令:

	go test -run '^$' -bench '^BenchmarkGoroutine' -benchtime=3s -count=5 -benchmem .

预期结论:
 1. goroutine创建+等待约300-500ns，初始栈约2-4KB
 2. channel ping-pong切换约100-200ns/次
 3. 相比OS线程（~1MB栈 + 微秒级创建），goroutine极为轻量
*/

func BenchmarkGoroutineCreate(b *testing.B) {
	for b.Loop() {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			wg.Done()
		}()
		wg.Wait()
	}
}

func BenchmarkGoroutineCreateBatch100(b *testing.B) {
	for b.Loop() {
		var wg sync.WaitGroup
		wg.Add(100)
		for i := 0; i < 100; i++ {
			go func() {
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func BenchmarkGoroutineSwitchPingPong(b *testing.B) {
	ping := make(chan struct{})
	pong := make(chan struct{})

	go func() {
		for range ping {
			pong <- struct{}{}
		}
	}()

	for b.Loop() {
		ping <- struct{}{}
		<-pong
	}
	close(ping)
}
