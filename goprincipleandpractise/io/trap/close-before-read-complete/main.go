package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

/*
陷阱：底层资源提前关闭导致读取失败

运行：go run .

预期行为：
  装饰器 Reader（bufio.Scanner、json.Decoder、gzip.Reader 等）不拥有底层资源。
  它们在每次 Read/Scan 时才调用底层 Reader 的 Read 方法。
  如果在装饰器还在使用时关闭了底层文件，后续读取会失败。

  正确做法：确保底层资源的生命周期覆盖所有读取操作。用 defer f.Close() 在打开后立即注册。
*/

func main() {
	tmpDir, err := os.MkdirTemp("", "close-before-read-*")
	if err != nil {
		fmt.Println("创建临时目录失败:", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	// 准备测试文件（足够大，使 bufio 无法一次缓冲全部内容）
	testFile := filepath.Join(tmpDir, "data.txt")
	func() {
		f, _ := os.Create(testFile)
		defer f.Close()
		for i := 1; i <= 10000; i++ {
			fmt.Fprintf(f, "第 %d 行数据内容，需要足够长以超出 bufio 的默认缓冲区大小\n", i)
		}
	}()

	fmt.Println("=== 错误做法：读取一半就关闭文件 ===")
	readWithEarlyClose(testFile)

	fmt.Println("\n=== 正确做法：defer Close 保证生命周期 ===")
	readCorrectly(testFile)

	fmt.Println("\n总结:")
	fmt.Println("  1. bufio.Scanner/json.Decoder 等不拥有底层资源")
	fmt.Println("  2. 底层文件关闭后，Scanner 的后续 Scan 会失败")
	fmt.Println("  3. 正确做法：打开文件后立即 defer f.Close()，让 Scanner 在函数内自由使用")
	fmt.Println("  4. 关闭顺序：先关外层装饰器（如果有 Close），再关底层资源")
}

func readWithEarlyClose(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("  打开文件失败:", err)
		return
	}

	scanner := bufio.NewScanner(f)

	// 读取第一行
	if scanner.Scan() {
		fmt.Printf("  读取成功: %s\n", scanner.Text())
	}

	// 错误：在 Scanner 还在使用时关闭文件
	f.Close()
	fmt.Println("  文件已关闭，Scanner 仍在使用...")

	// 尝试继续读取
	count := 0
	for scanner.Scan() {
		count++
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("  Scanner 错误（文件已关闭）: %v\n", err)
		fmt.Printf("  文件关闭后只从缓冲区中读取了 %d 行（总共 10000 行）\n", count)
	}
}

func readCorrectly(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("  打开文件失败:", err)
		return
	}
	defer f.Close() // 正确：函数返回时才关闭

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		count++
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("  Scanner 错误: %v\n", err)
	}
	fmt.Printf("  总共读取 %d 行\n", count)
}
