package main

import (
	"fmt"
	"time"
)

/*
陷阱：Timer.Reset 前未 drain channel 导致意外触发（Go < 1.23）

运行：go run .

预期行为（Go < 1.23）：
  Timer 的 channel 缓冲区大小为 1。当 Timer 到期时，runtime 向 channel
  发送一个值。如果在 Reset 之前没有消费掉旧值（drain），下次从 channel
  读取时会立即拿到旧值，而不是等待新的超时。

  Go 1.23+ 改变了行为：Reset 会自动清空 channel，无需手动 drain。
  但为了向后兼容，仍建议使用安全的 drain 模式。

  正确做法（Go < 1.23）：
    if !timer.Stop() {
        <-timer.C
    }
    timer.Reset(newDuration)
*/

func main() {
	fmt.Println("=== 演示 Timer Reset 的安全用法 ===")

	// 场景：用 Timer 实现"收到消息后重置超时"
	msgChan := make(chan string, 5)

	// 模拟消息到达
	go func() {
		msgs := []string{"msg1", "msg2", "msg3"}
		for _, msg := range msgs {
			time.Sleep(100 * time.Millisecond)
			msgChan <- msg
		}
		// 停止发送，让 Timer 超时
	}()

	fmt.Println("\n--- 错误做法（Go < 1.23 会出问题）---")
	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()

	count := 0
	for count < 4 {
		select {
		case msg := <-msgChan:
			fmt.Printf("  收到: %s\n", msg)
			// 错误：直接 Reset，没有 drain
			// 如果 timer 恰好在此时到期，channel 中已有旧值
			// 下次 select 会立即走 timer.C 分支
			timer.Reset(500 * time.Millisecond) //nolint
		case <-timer.C:
			fmt.Println("  超时！")
			count = 4 // 退出循环
		}
		count++
	}

	fmt.Println("\n--- 正确做法（兼容所有版本）---")
	timer2 := time.NewTimer(500 * time.Millisecond)
	defer timer2.Stop()

	// 再发一些消息
	go func() {
		msgs := []string{"msg4", "msg5", "msg6"}
		for _, msg := range msgs {
			time.Sleep(100 * time.Millisecond)
			msgChan <- msg
		}
	}()

	count = 0
	for count < 4 {
		select {
		case msg := <-msgChan:
			fmt.Printf("  收到: %s\n", msg)
			// 正确：先 Stop，drain，再 Reset
			if !timer2.Stop() {
				// timer 已到期，drain channel 中的旧值
				select {
				case <-timer2.C:
				default:
				}
			}
			timer2.Reset(500 * time.Millisecond)
		case <-timer2.C:
			fmt.Println("  超时！")
			count = 4
		}
		count++
	}

	fmt.Println("\n总结:")
	fmt.Println("  1. Go < 1.23：Reset 前必须 Stop + drain channel")
	fmt.Println("  2. Go 1.23+：Reset 自动清空 channel，可直接调用")
	fmt.Println("  3. 兼容写法：if !timer.Stop() { select { case <-timer.C: default: } }")
	fmt.Println("  4. Stop 返回 false 表示 timer 已到期或已被 Stop")
}
