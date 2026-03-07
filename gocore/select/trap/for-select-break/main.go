package main

import (
	"fmt"
	"time"
)

/*
陷阱：break 在 select 中只跳出 select，不跳出外层 for 循环

运行：go run .

预期行为：
  很多开发者习惯在 switch/case 中用 break 跳出循环，但在 select 中
  break 只跳出 select 语句本身，外层 for 循环不受影响。
  这会导致循环无法按预期退出，变成死循环或逻辑错误。

  正确做法：使用带标签的 break（break label）或 return。
*/

func main() {
	fmt.Println("=== 演示1: break 只跳出 select，不跳出 for ===")
	demoBreakOnlyExitsSelect()

	fmt.Println("\n=== 演示2: 正确做法 — 使用带标签的 break ===")
	demoBreakWithLabel()

	fmt.Println("\n=== 演示3: 正确做法 — 使用 return 或提取为函数 ===")
	result := demoReturnFromFunc()
	fmt.Printf("函数返回: %d\n", result)
}

func demoBreakOnlyExitsSelect() {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)

	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	iterations := 0
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				fmt.Println("  channel 已关闭，执行 break...")
				break // 注意：这里只跳出 select，不跳出 for！
			}
			fmt.Printf("  收到: %d\n", v)
		case <-tick.C:
			fmt.Println("  tick")
		}

		iterations++
		if iterations > 5 {
			// 如果上面的 break 能跳出 for，这行永远不会执行
			fmt.Println("  break 没有跳出 for！已循环", iterations, "次，强制退出")
			return
		}
	}
}

func demoBreakWithLabel() {
	ch := make(chan int, 3)
	ch <- 10
	ch <- 20
	ch <- 30
	close(ch)

	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

Loop: // 标签必须紧贴 for 语句
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				fmt.Println("  channel 已关闭，break Loop 跳出外层 for")
				break Loop // 正确：跳出外层 for 循环
			}
			fmt.Printf("  收到: %d\n", v)
		case <-tick.C:
			fmt.Println("  tick")
		}
	}
	fmt.Println("  已成功跳出 for 循环")
}

func demoReturnFromFunc() int {
	ch := make(chan int, 3)
	ch <- 100
	ch <- 200
	ch <- 300
	close(ch)

	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	sum := 0
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return sum // 直接 return，最简洁
			}
			sum += v
			fmt.Printf("  累加: %d, sum=%d\n", v, sum)
		case <-tick.C:
			fmt.Println("  tick")
		}
	}
}
