package main

import (
	"fmt"
	"io"
	"time"
)

/*
陷阱：io.Pipe 同步误用导致死锁

运行：go run .

预期行为：
  io.Pipe 是同步管道，没有内部缓冲区。
  PipeWriter.Write 阻塞直到 PipeReader.Read 消费数据。
  如果在同一个 goroutine 中先 Write 后 Read，Write 永远不会返回——死锁。

  正确做法：Write 和 Read 必须在不同的 goroutine 中执行。
*/

func main() {
	fmt.Println("=== 错误做法：同一 goroutine 中 Write 后 Read ===")
	demonstrateDeadlock()

	fmt.Println("\n=== 正确做法：Write 和 Read 在不同 goroutine ===")
	demonstrateCorrect()

	fmt.Println("\n=== io.Pipe 实际用途：连接 Writer API 和 Reader API ===")
	demonstratePractical()

	fmt.Println("\n总结:")
	fmt.Println("  1. io.Pipe 是同步的，没有内部缓冲区")
	fmt.Println("  2. Write 阻塞到 Read 消费，Read 阻塞到 Write 提供数据")
	fmt.Println("  3. 同一 goroutine 中 Write+Read = 死锁")
	fmt.Println("  4. 典型用途：连接需要 Writer 的 API 和需要 Reader 的 API")
}

func demonstrateDeadlock() {
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	done := make(chan bool, 1)

	go func() {
		// 模拟在同一逻辑中先写后读
		fmt.Println("  尝试写入数据...")

		// 用超时检测死锁
		writeDone := make(chan error, 1)
		go func() {
			_, err := pw.Write([]byte("hello"))
			writeDone <- err
		}()

		select {
		case <-writeDone:
			fmt.Println("  写入完成（不应该到这里）")
		case <-time.After(500 * time.Millisecond):
			fmt.Println("  写入阻塞超过 500ms — 死锁！（没有人在读取）")
		}
		done <- true
	}()

	<-done
}

func demonstrateCorrect() {
	pr, pw := io.Pipe()

	// Writer 在单独的 goroutine 中
	go func() {
		defer pw.Close()
		pw.Write([]byte("hello from pipe"))
		fmt.Println("  Writer: 数据已发送并被消费")
	}()

	// Reader 在当前 goroutine 中
	data, err := io.ReadAll(pr)
	if err != nil {
		fmt.Printf("  读取错误: %v\n", err)
		return
	}
	fmt.Printf("  Reader: 收到 %q\n", string(data))
}

func demonstratePractical() {
	// 场景：将结构化数据"流式"传递
	// 生产者以 Writer 方式写入，消费者以 Reader 方式读取
	pr, pw := io.Pipe()

	// 生产者：逐行写入
	go func() {
		defer pw.Close()
		lines := []string{"line 1: hello\n", "line 2: world\n", "line 3: done\n"}
		for _, line := range lines {
			pw.Write([]byte(line))
		}
	}()

	// 消费者：一次性读取
	data, _ := io.ReadAll(pr)
	fmt.Printf("  消费者收到 %d 字节:\n%s", len(data), string(data))
}
