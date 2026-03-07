package poll_vs_sleep

import (
	"fmt"
	"sync/atomic"
	"time"
)

// poll_vs_sleep 对比两种等待服务就绪策略的耗时：
//
// 1. 固定 Sleep：无论服务是否就绪，都等待固定时长
// 2. 轮询（Poll）：每隔一小段时间检查服务状态，就绪后立即返回
//
// 结论：轮询方式在平均情况下更快，且更可靠。
// 固定 Sleep 要么浪费时间（等太久），要么不够（等太短导致失败）。

// ServiceSimulator 模拟一个需要一定时间才能启动的服务
type ServiceSimulator struct {
	ready      atomic.Bool
	startedAt  time.Time
	readyAfter time.Duration
}

// NewServiceSimulator 创建模拟服务，在 readyAfter 后变为就绪状态
func NewServiceSimulator(readyAfter time.Duration) *ServiceSimulator {
	s := &ServiceSimulator{
		startedAt:  time.Now(),
		readyAfter: readyAfter,
	}
	go func() {
		time.Sleep(readyAfter)
		s.ready.Store(true)
	}()
	return s
}

// IsReady 检查服务是否就绪
func (s *ServiceSimulator) IsReady() bool {
	return s.ready.Load()
}

// Reset 重置服务状态
func (s *ServiceSimulator) Reset(readyAfter time.Duration) {
	s.ready.Store(false)
	s.readyAfter = readyAfter
	s.startedAt = time.Now()
	go func() {
		time.Sleep(readyAfter)
		s.ready.Store(true)
	}()
}

// WaitWithSleep 使用固定 Sleep 等待
func WaitWithSleep(sleepDuration time.Duration) time.Duration {
	start := time.Now()
	time.Sleep(sleepDuration)
	return time.Since(start)
}

// WaitWithPoll 使用轮询等待服务就绪
func WaitWithPoll(svc *ServiceSimulator, pollInterval, timeout time.Duration) (time.Duration, error) {
	start := time.Now()
	deadline := start.Add(timeout)

	for time.Now().Before(deadline) {
		if svc.IsReady() {
			return time.Since(start), nil
		}
		time.Sleep(pollInterval)
	}
	return time.Since(start), fmt.Errorf("timeout after %v", timeout)
}
