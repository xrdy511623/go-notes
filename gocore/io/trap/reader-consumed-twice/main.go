package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

/*
陷阱：io.Reader 被消费后无法重读

运行：go run .

预期行为：
  大多数 io.Reader 是一次性的——读取后游标前进，无法回退。
  第一次 io.ReadAll 成功获取数据，第二次 io.ReadAll 返回空。
  这在处理 http.Response.Body、os.Stdin 等流式 Reader 时尤为常见。

  正确做法：
  1. 对可 Seek 的 Reader，用 Seek(0, io.SeekStart) 回退
  2. 用 io.TeeReader 在读取时同时保存一份副本
  3. 将数据读入 []byte，按需用 bytes.NewReader 重建
*/

func main() {
	fmt.Println("=== 错误做法：尝试读取 Reader 两次 ===")

	r := strings.NewReader("hello world")
	data1, _ := io.ReadAll(r)
	fmt.Printf("  第一次读取: %q (%d 字节)\n", string(data1), len(data1))

	data2, _ := io.ReadAll(r)
	fmt.Printf("  第二次读取: %q (%d 字节) ← 数据丢失！\n", string(data2), len(data2))

	fmt.Println("\n=== 方案 1：Seek 回退（仅限 Seeker）===")

	r2 := strings.NewReader("hello world")
	data3, _ := io.ReadAll(r2)
	fmt.Printf("  第一次读取: %q\n", string(data3))

	r2.Seek(0, io.SeekStart) // 回退到开头
	data4, _ := io.ReadAll(r2)
	fmt.Printf("  Seek 后读取: %q ← 数据恢复\n", string(data4))

	fmt.Println("\n=== 方案 2：TeeReader 边读边保存 ===")

	original := strings.NewReader("important data")
	var buf bytes.Buffer
	tee := io.TeeReader(original, &buf) // 读取 tee 时，数据同时写入 buf

	data5, _ := io.ReadAll(tee)
	fmt.Printf("  第一次读取（通过 TeeReader）: %q\n", string(data5))
	fmt.Printf("  缓存中的副本: %q ← 可以重复使用\n", buf.String())

	fmt.Println("\n=== 方案 3：读入 []byte，按需重建 Reader ===")

	source := strings.NewReader("reusable data")
	raw, _ := io.ReadAll(source)
	fmt.Printf("  原始数据: %q\n", string(raw))

	// 从同一份 []byte 创建多个 Reader
	r3 := bytes.NewReader(raw)
	d1, _ := io.ReadAll(r3)
	fmt.Printf("  第一个 Reader: %q\n", string(d1))

	r4 := bytes.NewReader(raw)
	d2, _ := io.ReadAll(r4)
	fmt.Printf("  第二个 Reader: %q\n", string(d2))

	fmt.Println("\n总结:")
	fmt.Println("  1. 大多数 io.Reader 是一次性的（流式），读完无法回退")
	fmt.Println("  2. http.Response.Body、os.Stdin 等都是不可重读的")
	fmt.Println("  3. strings.Reader 和 bytes.Reader 支持 Seek，可以回退")
	fmt.Println("  4. 需要重复读取时，先缓存到 []byte，再按需创建新 Reader")
}
