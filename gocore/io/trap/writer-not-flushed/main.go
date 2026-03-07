package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

/*
陷阱：bufio.Writer 未 Flush 导致数据丢失

运行：go run .

预期行为：
  bufio.Writer 将数据缓存在内存中，只有调用 Flush() 才会写入底层 Writer。
  如果在 Flush 之前关闭了底层文件，缓冲区中的数据会静默丢失。
  尤其当写入数据量小于缓冲区大小（默认 4096 字节）时，Flush 从未被触发。

  正确做法：始终 defer bw.Flush()，且确保 Flush 在 f.Close() 之前执行。
*/

func main() {
	tmpDir, err := os.MkdirTemp("", "writer-not-flushed-*")
	if err != nil {
		fmt.Println("创建临时目录失败:", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	fmt.Println("=== 错误做法：不调用 Flush ===")
	path1 := filepath.Join(tmpDir, "no-flush.txt")
	writeWithoutFlush(path1)
	showFileSize(path1)

	fmt.Println("\n=== 错误做法：Flush 顺序不对（Close 在 Flush 之前）===")
	path2 := filepath.Join(tmpDir, "wrong-order.txt")
	writeWrongOrder(path2)
	showFileSize(path2)

	fmt.Println("\n=== 正确做法：defer Flush 在 defer Close 之后注册（LIFO 先执行）===")
	path3 := filepath.Join(tmpDir, "correct.txt")
	writeCorrect(path3)
	showFileSize(path3)

	fmt.Println("\n总结:")
	fmt.Println("  1. bufio.Writer 没有 Close 方法，不会自动 Flush")
	fmt.Println("  2. 必须显式调用 Flush()，否则缓冲区数据丢失")
	fmt.Println("  3. defer 是 LIFO 顺序：后注册的先执行")
	fmt.Println("  4. 正确写法：先 defer f.Close()，再 defer bw.Flush()（Flush 先执行）")
}

func writeWithoutFlush(path string) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Println("  创建文件失败:", err)
		return
	}

	bw := bufio.NewWriter(f)
	data := "这是一段重要的数据，共 100 字节左右的内容，不会超过 bufio 默认的 4096 字节缓冲区。"
	n, _ := bw.WriteString(data)
	fmt.Printf("  写入 %d 字节到 bufio.Writer（缓冲区中）\n", n)

	// 错误：直接关闭文件，没有 Flush
	f.Close()
	fmt.Println("  文件已关闭，但未调用 Flush")
}

func writeWrongOrder(path string) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Println("  创建文件失败:", err)
		return
	}

	bw := bufio.NewWriter(f)
	data := "这段数据也会丢失，因为 Close 发生在 Flush 之前。"
	n, _ := bw.WriteString(data)
	fmt.Printf("  写入 %d 字节到 bufio.Writer\n", n)

	// 错误：Close 在 Flush 之前（模拟 defer 顺序搞反的情况）
	f.Close()
	err = bw.Flush()
	if err != nil {
		fmt.Printf("  Flush 失败（文件已关闭）: %v\n", err)
	}
}

func writeCorrect(path string) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Println("  创建文件失败:", err)
		return
	}
	defer f.Close() // 第一个 defer，最后执行

	bw := bufio.NewWriter(f)
	defer bw.Flush() // 第二个 defer，先执行（LIFO）

	data := "这段数据会正确写入文件，因为 Flush 在 Close 之前执行。"
	n, _ := bw.WriteString(data)
	fmt.Printf("  写入 %d 字节到 bufio.Writer\n", n)
}

func showFileSize(path string) {
	info, err := os.Stat(path)
	if err != nil {
		fmt.Printf("  读取文件信息失败: %v\n", err)
		return
	}
	fmt.Printf("  文件实际大小: %d 字节\n", info.Size())
	if info.Size() == 0 {
		fmt.Println("  ⚠ 数据丢失！文件为空")
	}
}
