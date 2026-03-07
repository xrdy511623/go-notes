package main

import (
	"fmt"
	"time"
)

/*
陷阱：用 time.Sleep 模拟定时任务导致时间漂移

运行：go run .

预期行为：
  time.Sleep(interval) 的实际间隔 = interval + 任务执行时间。
  随着迭代次数增加，累积漂移越来越大。

  time.NewTicker 基于绝对时间点触发，不会累积漂移。

  正确做法：需要精确间隔时使用 Ticker，不要用 Sleep 循环
*/

func main() {
	const iterations = 10
	workDuration := 50 * time.Millisecond // 模拟每次任务执行 50ms
	interval := 100 * time.Millisecond

	fmt.Println("=== 错误做法：time.Sleep 循环 ===")
	fmt.Printf("  目标间隔: %v, 任务耗时: %v\n", interval, workDuration)
	fmt.Printf("  预期总耗时: %v\n\n", time.Duration(iterations)*interval)

	start := time.Now()
	for i := 0; i < iterations; i++ {
		iterStart := time.Now()

		// 模拟工作
		time.Sleep(workDuration)

		elapsed := time.Since(iterStart)
		fmt.Printf("  迭代 %2d: 实际耗时 %v", i+1, elapsed.Round(time.Millisecond))

		// Sleep 间隔（不包含工作时间的补偿）
		time.Sleep(interval) // 错误：Sleep 间隔 + 工作时间 > 目标间隔
		if i > 0 {
			fmt.Printf(", 累积漂移 %v", time.Since(start).Round(time.Millisecond)-time.Duration(i+1)*interval)
		}
		fmt.Println()
	}
	sleepTotal := time.Since(start)
	fmt.Printf("  实际总耗时: %v（漂移 %v）\n",
		sleepTotal.Round(time.Millisecond),
		(sleepTotal - time.Duration(iterations)*interval).Round(time.Millisecond))

	fmt.Println("\n=== 正确做法：time.NewTicker ===")
	fmt.Printf("  目标间隔: %v, 任务耗时: %v\n\n", interval, workDuration)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	start = time.Now()
	for i := 0; i < iterations; i++ {
		<-ticker.C // 等待 Ticker 触发

		// 模拟工作
		time.Sleep(workDuration)

		fmt.Printf("  迭代 %2d: 触发时间 %v\n", i+1,
			time.Since(start).Round(time.Millisecond))
	}
	tickerTotal := time.Since(start)
	fmt.Printf("  实际总耗时: %v（漂移 %v）\n",
		tickerTotal.Round(time.Millisecond),
		(tickerTotal - time.Duration(iterations)*interval).Round(time.Millisecond))

	fmt.Println("\n总结:")
	fmt.Println("  1. time.Sleep 循环的实际间隔 = Sleep + 工作时间，会累积漂移")
	fmt.Println("  2. Ticker 基于绝对时间触发，不累积漂移")
	fmt.Println("  3. 如果任务耗时 > Ticker 间隔，Ticker 会跳过中间的 tick")
	fmt.Println("  4. 需要精确定时（心跳、采集）时，必须用 Ticker")
}
