package unbounded

import (
	"fmt"
	"sync"
	"time"
)

// 扇出模式的典型陷阱：无界 goroutine 创建 + 丢失错误处理
//
// 以下两个反模式在生产代码中非常常见，是内存溢出和静默失败的根源。

// ❌ 陷阱一：无界 goroutine — N 个输入 = N 个 goroutine
//
// 当 urls 包含 10 万条记录时，瞬间创建 10 万个 goroutine：
//   - 每个 goroutine 栈初始 2-8 KB → 至少 200 MB 内存
//   - 10 万个并发 HTTP 连接打垮目标服务和自身 fd 限制
//   - runtime 调度 10 万个 goroutine 的 CPU 开销远超实际工作
//
// ✅ 正确做法：使用 parallel.Scatter 或 parallel.ForEach 限制并发数
//
//	parallel.ForEach(ctx, urls, 10, func(ctx context.Context, url string) error {
//	    return fetch(ctx, url)
//	})
func UnboundedFanOut(urls []string) {
	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fetch(url) // ⚠️ 10万URL = 10万goroutine → OOM
		}()
	}
	wg.Wait()
}

// ❌ 陷阱二：WaitGroup 丢失错误 — 错误被静默吞掉
//
// sync.WaitGroup 只能等待完成，无法传递错误。
// 如果 process 返回 error，调用方完全不知道哪些 item 处理失败了。
//
// ✅ 正确做法：使用 errgroup 或 parallel.Scatter 自动传播错误
//
//	results, err := parallel.Scatter(ctx, items, 8,
//	    func(ctx context.Context, v int) (int, error) {
//	        return process(v)
//	    },
//	)
func SwallowedErrors(items []int) []int {
	results := make([]int, len(items))
	var wg sync.WaitGroup
	for i, item := range items {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := process(item)
			if err != nil {
				// ⚠️ 错误去哪了？fmt.Println 不是错误处理
				fmt.Printf("error processing %d: %v\n", item, err)
			}
			results[i] = val // ⚠️ 出错时写入零值，调用方无法区分
		}()
	}
	wg.Wait()
	return results // ⚠️ 可能包含无效数据，但调用方以为全部成功
}

func fetch(url string) {
	time.Sleep(10 * time.Millisecond)
	fmt.Printf("fetched %s\n", url)
}

func process(v int) (int, error) {
	if v%7 == 0 {
		return 0, fmt.Errorf("unlucky number: %d", v)
	}
	return v * v, nil
}
