package main

import (
	"bufio"
	"fmt"
	"strings"
)

/*
陷阱：bufio.Scanner 缓冲区过小导致静默截断

运行：go run .

预期行为：
  bufio.Scanner 默认最大 token 大小为 MaxScanTokenSize（64KB）。
  当某一行超过此限制时，Scanner 停止扫描，Err() 返回 bufio.ErrTooLong。
  后续所有行都不会被读取，数据静默丢失。

  正确做法：处理可能含长行的输入时，用 scanner.Buffer() 调大缓冲区限制。
*/

func main() {
	shortLine := "这是一行普通的短文本"
	longLine := strings.Repeat("X", 70000) // 70KB，超过默认 64KB 限制
	afterLong := "这行在长行之后，默认 Scanner 永远读不到这里"

	input := shortLine + "\n" + longLine + "\n" + afterLong + "\n"

	fmt.Println("=== 输入数据 ===")
	fmt.Printf("  第 1 行: %d 字节（短行）\n", len(shortLine))
	fmt.Printf("  第 2 行: %d 字节（超过 64KB 限制）\n", len(longLine))
	fmt.Printf("  第 3 行: %d 字节（长行之后）\n", len(afterLong))

	fmt.Println("\n=== 错误做法：使用默认 Scanner ===")
	scanner1 := bufio.NewScanner(strings.NewReader(input))
	count1 := 0
	for scanner1.Scan() {
		count1++
		line := scanner1.Text()
		if len(line) > 50 {
			fmt.Printf("  读取第 %d 行: [%d 字节]\n", count1, len(line))
		} else {
			fmt.Printf("  读取第 %d 行: %s\n", count1, line)
		}
	}
	if err := scanner1.Err(); err != nil {
		fmt.Printf("  Scanner 错误: %v\n", err)
	}
	fmt.Printf("  总共读取 %d 行（期望 3 行）\n", count1)

	fmt.Println("\n=== 正确做法：用 Buffer() 调大缓冲区 ===")
	scanner2 := bufio.NewScanner(strings.NewReader(input))
	scanner2.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 最大 1MB
	count2 := 0
	for scanner2.Scan() {
		count2++
		line := scanner2.Text()
		if len(line) > 50 {
			fmt.Printf("  读取第 %d 行: [%d 字节]\n", count2, len(line))
		} else {
			fmt.Printf("  读取第 %d 行: %s\n", count2, line)
		}
	}
	if err := scanner2.Err(); err != nil {
		fmt.Printf("  Scanner 错误: %v\n", err)
	}
	fmt.Printf("  总共读取 %d 行（期望 3 行）\n", count2)

	fmt.Println("\n总结:")
	fmt.Println("  1. Scanner 默认最大 token 大小为 64KB (bufio.MaxScanTokenSize)")
	fmt.Println("  2. 超长行导致 Scanner 停止，后续所有行丢失")
	fmt.Println("  3. 处理不可控输入时，始终用 scanner.Buffer() 设置合理的上限")
	fmt.Println("  4. 始终检查 scanner.Err()，不要只检查 Scan() 的返回值")
}
