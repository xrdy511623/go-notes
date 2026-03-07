package main

import (
	"fmt"
	"unsafe"
)

/*
陷阱：通过 unsafe 修改 string 底层数据的危险

运行：go run .

预期行为：
  使用 unsafe.Pointer 强制修改 string 底层字节可能导致不可预测的行为。
  字符串字面量存储在只读段，修改会直接触发段错误（SIGSEGV）。
  即使修改堆上的字符串，也破坏了 string 不变性的语义保证。
*/

func main() {
	fmt.Println("=== 演示1: 零拷贝转换后修改 []byte 的风险 ===")
	demonstrateUnsafeConversion()

	fmt.Println("\n=== 演示2: 字符串字面量不可修改 ===")
	fmt.Println("以下操作会触发段错误，因此注释掉以演示说明。")
	fmt.Println("字符串字面量存储在程序的只读数据段（rodata），")
	fmt.Println("通过 unsafe 指针修改会导致 SIGSEGV。")

	// 取消下面注释会导致程序崩溃：
	// modifyStringLiteral()
}

// stringToBytes 零拷贝将 string 转换为 []byte（不安全！）
func stringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

func demonstrateUnsafeConversion() {
	// 使用 make 创建的字符串（在堆上），可以通过 unsafe 修改
	// 但这违反了 Go 的 string 不变性语义
	buf := make([]byte, 5)
	copy(buf, "hello")
	s := string(buf) // 这里发生了拷贝

	fmt.Printf("原始字符串: %q\n", s)

	// 通过 unsafe 直接操作底层数据
	b := stringToBytes(s)
	b[0] = 'H'

	fmt.Printf("修改后字符串: %q\n", s)
	fmt.Println("看似修改成功，但这破坏了 string 的不变性保证！")
	fmt.Println("在并发场景下，其他 goroutine 可能看到不一致的数据。")
	fmt.Println("如果 s 被用作 map key，哈希值将不再匹配。")
}

// modifyStringLiteral 尝试修改字符串字面量 —— 会触发 SIGSEGV
// func modifyStringLiteral() {
// 	s := "hello"
// 	b := stringToBytes(s)
// 	b[0] = 'H' // SIGSEGV: 字符串字面量存储在只读段
// }
