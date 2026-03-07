package pool

import "sync"

// WorkerPool 是基于固定 goroutine 池的舱壁实现。
// 与信号量舱壁不同，WorkerPool 预先创建固定数量的 goroutine，
// 通过任务队列接收工作。这种方式的内存占用更可预测。
type WorkerPool struct {
	tasks chan func()
	quit  chan struct{}
	wg    sync.WaitGroup
	once  sync.Once
}

// NewWorkerPool 创建并启动一个固定大小的工作池。
// workers 为并发 goroutine 数量，queueSize 为任务队列的缓冲大小。
func NewWorkerPool(workers, queueSize int) *WorkerPool {
	wp := &WorkerPool{
		tasks: make(chan func(), queueSize),
		quit:  make(chan struct{}),
	}
	wp.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go wp.worker()
	}
	return wp
}

func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for {
		select {
		case fn, ok := <-wp.tasks:
			if !ok {
				return
			}
			fn()
		case <-wp.quit:
			// drain remaining tasks
			for fn := range wp.tasks {
				fn()
			}
			return
		}
	}
}

// Submit 提交任务到工作池。队列满时阻塞直到有空位。
func (wp *WorkerPool) Submit(fn func()) {
	wp.tasks <- fn
}

// TrySubmit 尝试提交任务，队列满时立即返回 false。
func (wp *WorkerPool) TrySubmit(fn func()) bool {
	select {
	case wp.tasks <- fn:
		return true
	default:
		return false
	}
}

// Shutdown 优雅关闭工作池。关闭任务队列后等待所有已提交任务执行完毕。
// 多次调用 Shutdown 是安全的（通过 sync.Once 保护）。
func (wp *WorkerPool) Shutdown() {
	wp.once.Do(func() {
		close(wp.tasks)
	})
	wp.wg.Wait()
}
