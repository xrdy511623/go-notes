package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

/*
陷阱：测试结束不清理资源，导致泄漏

运行：go run .

预期行为：
  演示三种资源泄漏场景：
  1. 临时文件未删除 → 磁盘残留
  2. 端口未释放 → 后续测试绑定失败
  3. 对比 defer 清理后的正确效果

  正确做法：使用 defer 确保资源释放，测试中使用 t.Cleanup()。
*/

func main() {
	fmt.Println("=== 场景一：临时文件未清理 ===")
	fmt.Println()

	tmpDir, _ := os.MkdirTemp("", "no-cleanup-trap-*")

	// ❌ 模拟创建临时文件但不清理
	leakedFiles := make([]string, 0)
	for i := 0; i < 5; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("test-data-%d.tmp", i))
		os.WriteFile(path, []byte("test data"), 0644)
		leakedFiles = append(leakedFiles, path)
	}

	// 查看泄漏的文件
	entries, _ := os.ReadDir(tmpDir)
	fmt.Printf("  创建了 %d 个临时文件（模拟 5 次测试运行）\n", len(entries))
	for _, e := range entries {
		info, _ := e.Info()
		fmt.Printf("    %s (%d bytes) — 泄漏！\n", e.Name(), info.Size())
	}
	fmt.Println("  如果每次 CI 泄漏 5 个文件，一年后 /tmp 下有上千个垃圾文件")

	// ✅ 清理
	os.RemoveAll(tmpDir)
	fmt.Println("  [已清理] 使用 defer os.RemoveAll() 即可避免")

	fmt.Println()
	fmt.Println("=== 场景二：端口未释放 ===")
	fmt.Println()

	// ❌ 占用一个端口但不释放
	listener1, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("  绑定失败: %v\n", err)
		return
	}
	port := listener1.Addr().(*net.TCPAddr).Port
	fmt.Printf("  测试 A 占用了端口 %d\n", port)

	// 尝试再次绑定同一个端口（模拟下一个测试）
	listener2, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		fmt.Printf("  测试 B 尝试绑定端口 %d → 失败: %v\n", port, err)
		fmt.Println("  原因: 测试 A 没有关闭 listener，端口仍被占用")
	} else {
		listener2.Close()
	}

	// ✅ 释放端口后再试
	listener1.Close()
	fmt.Printf("  [已修复] 测试 A 关闭了 listener\n")
	listener3, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		fmt.Printf("  测试 B 仍然失败: %v\n", err)
	} else {
		fmt.Printf("  测试 B 绑定端口 %d → 成功\n", port)
		listener3.Close()
	}

	fmt.Println()
	fmt.Println("=== 场景三：defer 的正确使用方式 ===")
	fmt.Println()

	tmpDir2, _ := os.MkdirTemp("", "cleanup-correct-*")

	// 模拟正确的测试写法
	func() {
		// ✅ 创建后立即 defer 清理
		defer os.RemoveAll(tmpDir2)

		path := filepath.Join(tmpDir2, "important.dat")
		os.WriteFile(path, []byte("test data"), 0644)

		_, err := os.Stat(path)
		fmt.Printf("  函数执行中: 文件存在=%v\n", err == nil)

		// 即使这里 panic，defer 也会执行清理
	}()

	_, err = os.Stat(tmpDir2)
	fmt.Printf("  函数返回后: 目录存在=%v\n", err == nil)
	fmt.Println("  defer 确保了资源释放，即使函数提前退出或 panic")

	fmt.Println()
	fmt.Println("总结:")
	fmt.Println("  1. 临时文件 → defer os.RemoveAll()")
	fmt.Println("  2. 端口/Listener → defer listener.Close()")
	fmt.Println("  3. Docker 容器 → t.Cleanup(func() { container.Terminate(ctx) })")
	fmt.Println("  4. 数据库连接 → defer db.Close()")
	fmt.Println("  5. 原则：获取资源后，下一行就写 defer 释放")
}
