package pattern

import "context"

// Generator 将一组值发送到channel（pipeline的数据源阶段）
func Generator(ctx context.Context, nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			select {
			case out <- n:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Square 对输入channel中的每个值做平方运算（pipeline的处理阶段）
func Square(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			select {
			case out <- n * n:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Double 对输入channel中的每个值做翻倍运算（pipeline的处理阶段）
func Double(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			select {
			case out <- n * 2:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Filter 过滤出满足条件的值
func Filter(ctx context.Context, in <-chan int, predicate func(int) bool) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			if predicate(n) {
				select {
				case out <- n:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}

// Collect 从channel收集所有值到slice
func Collect(ch <-chan int) []int {
	var result []int
	for v := range ch {
		result = append(result, v)
	}
	return result
}
