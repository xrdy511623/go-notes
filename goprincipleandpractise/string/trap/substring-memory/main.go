package main

import (
	"fmt"
	"runtime"
	"strings"
)

/*
陷阱：子字符串切片导致大字符串无法被 GC 回收

运行：go run .

预期输出：
  演示子字符串 s[a:b] 如何共享底层数组，导致原始大字符串无法被 GC 回收。
  使用 strings.Clone (Go 1.20+) 可以解决这个问题。
*/

func main() {
	fmt.Println("=== 演示: 子字符串切片的内存保留问题 ===")

	// 模拟从网络/文件读取大字符串
	big := strings.Repeat("x", 1<<20) // 1 MB 字符串
	fmt.Printf("大字符串长度: %d 字节\n", len(big))

	// 只需要前 10 个字符
	sub := big[:10]
	fmt.Printf("子字符串: %q (长度: %d)\n", sub, len(sub))

	// 释放 big 的引用
	big = ""
	runtime.GC()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	fmt.Printf("GC 后堆内存: %d KB\n", ms.HeapInuse/1024)
	fmt.Println("即使 big 已置空，底层 1MB 数据仍被 sub 引用，无法回收！")

	fmt.Println("\n=== 正确做法: 使用 strings.Clone ===")

	big2 := strings.Repeat("y", 1<<20) // 又一个 1 MB 字符串
	cloned := strings.Clone(big2[:10])  // 独立拷贝，不再引用底层大数组

	big2 = ""
	runtime.GC()

	runtime.ReadMemStats(&ms)
	fmt.Printf("Clone 后 GC 堆内存: %d KB\n", ms.HeapInuse/1024)
	fmt.Printf("cloned 内容: %q\n", cloned)
	fmt.Println("使用 strings.Clone 后，原始大字符串可以被正常回收。")

	// 防止编译器优化掉 sub
	runtime.KeepAlive(sub)
	runtime.KeepAlive(cloned)
}
