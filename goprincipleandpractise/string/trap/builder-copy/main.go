package main

import (
	"fmt"
	"strings"
)

/*
陷阱：复制 strings.Builder 后写入导致 panic

运行：go run .

预期行为：
  复制一个已写入数据的 strings.Builder 后再进行写入操作，
  会触发 panic: strings: illegal use of non-zero Builder copied by value

  这是因为 Builder 内部的 copyCheck 机制会检测 addr 字段，
  发现复制后地址不匹配，从而 panic。
*/

func main() {
	fmt.Println("=== 演示: 复制 strings.Builder 后写入导致 panic ===")

	var b1 strings.Builder
	b1.WriteString("hello")
	fmt.Printf("b1 内容: %q\n", b1.String())

	// 值复制 Builder
	b2 := b1
	fmt.Printf("b2（复制后）内容: %q\n", b2.String()) // 读取是安全的

	fmt.Println("尝试对复制后的 b2 写入...")

	// 以下写入操作会触发 panic
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("捕获到 panic: %v\n", r)
			fmt.Println("\n原因分析:")
			fmt.Println("  Builder 内部有一个 addr *Builder 字段用于 copyCheck。")
			fmt.Println("  第一次写入时 addr 被设为自身地址。")
			fmt.Println("  值复制后 b2 有不同的地址，但 addr 仍指向 b1。")
			fmt.Println("  写入 b2 时检测到 addr != &b2，触发 panic。")
		}
	}()

	b2.WriteString(" world") // panic!
}
