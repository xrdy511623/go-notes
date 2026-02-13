package main

import (
	"fmt"
	"time"
)

/*
陷阱：nil channel 在 select 中的行为

运行：go run .

关键知识点：
  - nil channel 上的 send 和 receive 永远阻塞
  - 但在 select 中，nil channel 的 case 会被静默跳过（不参与选择）
  - 这个特性可以用来动态启用/禁用 select 的某个 case

本示例演示：
  1. nil channel 导致的永久阻塞（需要 select 兜底）
  2. 利用 nil channel 动态禁用 select case（合并多路 channel 的技巧）
*/

func main() {
	fmt.Println("=== 演示1: nil channel 在 select 中被跳过 ===")
	demoNilChannelSkipped()

	fmt.Println("\n=== 演示2: 利用 nil channel 动态禁用 case（合并 channel） ===")
	demoMergeWithNilDisable()
}

func demoNilChannelSkipped() {
	var nilCh chan int // nil channel

	select {
	case v := <-nilCh:
		// 永远不会执行：nil channel 的 case 在 select 中被跳过
		fmt.Println("received from nil channel:", v)
	case <-time.After(100 * time.Millisecond):
		fmt.Println("nil channel 的 case 被跳过，走到了 timeout 分支")
	}
}

// demoMergeWithNilDisable 演示经典技巧：合并两个 channel，
// 当其中一个关闭后，将其设为 nil 来禁用对应的 select case
func demoMergeWithNilDisable() {
	ch1 := make(chan string, 2)
	ch2 := make(chan string, 2)

	ch1 <- "a1"
	ch1 <- "a2"
	close(ch1)

	ch2 <- "b1"
	ch2 <- "b2"
	close(ch2)

	// 合并两个 channel 的输出
	var merged []string
	for ch1 != nil || ch2 != nil {
		select {
		case v, ok := <-ch1:
			if !ok {
				fmt.Println("ch1 已关闭，设为 nil 禁用此 case")
				ch1 = nil // 关键：设为 nil 后，下一轮 select 不再选择此 case
				continue
			}
			merged = append(merged, v)
		case v, ok := <-ch2:
			if !ok {
				fmt.Println("ch2 已关闭，设为 nil 禁用此 case")
				ch2 = nil
				continue
			}
			merged = append(merged, v)
		}
	}
	fmt.Printf("合并结果: %v\n", merged)
	fmt.Println("这是 or-channel / fan-in 模式的基础技巧")
}
