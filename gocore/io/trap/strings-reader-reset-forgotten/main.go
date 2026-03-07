package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

/*
陷阱：strings.Reader / bytes.Reader 重用时忘记 Reset/Seek

运行：go run .

预期行为：
  strings.Reader 和 bytes.Reader 内部维护读取位置。
  第一次读取后，位置移到末尾。
  如果不调用 Reset() 或 Seek(0, io.SeekStart)，第二次读取得到空数据。
  这在循环中复用 Reader 或多次传递同一个 Reader 时尤为常见。

  正确做法：重用前调用 Reset() 或 Seek(0, io.SeekStart) 重置位置。
*/

func main() {
	fmt.Println("=== 错误做法：重用 strings.Reader 不 Reset ===")
	r := strings.NewReader("hello world")

	data1, _ := io.ReadAll(r)
	fmt.Printf("  第一次读取: %q\n", string(data1))

	data2, _ := io.ReadAll(r)
	fmt.Printf("  第二次读取: %q ← 空！游标已在末尾\n", string(data2))

	fmt.Println("\n=== 方案 1：用 Seek 回退 ===")
	r.Seek(0, io.SeekStart)
	data3, _ := io.ReadAll(r)
	fmt.Printf("  Seek 后读取: %q\n", string(data3))

	fmt.Println("\n=== 方案 2：用 Reset 重置（可同时换内容）===")
	r.Reset("new content")
	data4, _ := io.ReadAll(r)
	fmt.Printf("  Reset 后读取: %q\n", string(data4))

	// 相同内容重置
	r.Reset("new content")
	data5, _ := io.ReadAll(r)
	fmt.Printf("  再次 Reset 后: %q\n", string(data5))

	fmt.Println("\n=== bytes.Reader 同样需要 Seek ===")
	br := bytes.NewReader([]byte("bytes data"))

	d1, _ := io.ReadAll(br)
	fmt.Printf("  第一次读取: %q\n", string(d1))

	d2, _ := io.ReadAll(br)
	fmt.Printf("  第二次读取: %q ← 空\n", string(d2))

	br.Seek(0, io.SeekStart)
	d3, _ := io.ReadAll(br)
	fmt.Printf("  Seek 后读取: %q\n", string(d3))

	fmt.Println("\n=== 循环中复用的正确模式 ===")
	content := "reusable"
	reader := strings.NewReader(content)
	for i := 1; i <= 3; i++ {
		reader.Reset(content) // 每次循环开始时 Reset
		data, _ := io.ReadAll(reader)
		fmt.Printf("  第 %d 次迭代: %q\n", i, string(data))
	}

	fmt.Println("\n总结:")
	fmt.Println("  1. strings.Reader 和 bytes.Reader 有内部读取位置")
	fmt.Println("  2. 读完后位置在末尾，再次读取得到空数据")
	fmt.Println("  3. Seek(0, io.SeekStart) 回退位置，Reset() 可同时换内容")
	fmt.Println("  4. 循环中复用 Reader 时，每次迭代前必须 Reset 或 Seek")
}
