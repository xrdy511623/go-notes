package workerpool

import "sync"

// Pool 是有界工作池，复用固定数量的 goroutine 执行任务。
//
// 与 sync.Pool（临时对象池）不同，Pool 管理的是**持久资源**：
//   - 固定容量：创建时确定 worker 数量，不会无限增长
//   - 背压机制：任务队列满时 Submit 阻塞，天然限流
//   - 显式关闭：Close 等待所有任务完成后释放 worker，无 goroutine 泄漏
//
// 这是 database/sql 连接池、valyala/fasthttp workerPool 的简化模型。
// 核心思想一致：预创建一组资源（goroutine/连接），通过队列（channel/ring buffer）
// 分发工作，避免反复创建销毁的开销。
type Pool struct {
	tasks chan func()
	wg    sync.WaitGroup
}

// New 创建一个拥有 size 个 worker 的工作池。
// 每个 worker 是一个常驻 goroutine，从共享的 tasks channel 中获取并执行任务。
func New(size int) *Pool {
	p := &Pool{
		tasks: make(chan func()),
	}
	p.wg.Add(size)
	for range size {
		go p.worker()
	}
	return p
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for task := range p.tasks {
		task()
	}
}

// Submit 提交一个任务到工作池。
// 如果所有 worker 都在忙，Submit 会阻塞直到有 worker 空闲——这就是背压机制。
// 在 Close 之后调用 Submit 会 panic（向已关闭的 channel 发送数据）。
func (p *Pool) Submit(task func()) {
	p.tasks <- task
}

// Close 关闭工作池，等待所有已提交的任务执行完毕。
// 返回后所有 worker goroutine 已退出，不会有 goroutine 泄漏。
func (p *Pool) Close() {
	close(p.tasks)
	p.wg.Wait()
}
