package unittest

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------- t.Cleanup()：资源清理 ----------
//
// t.Cleanup 注册一个在测试结束时执行的清理函数。
//
// 与 defer 的区别：
//   - defer 在当前函数返回时执行，仅限于当前函数作用域
//   - t.Cleanup 在测试结束时执行，即使是在 helper 函数中注册也能正确清理
//
// 多个 Cleanup 按 LIFO（后进先出）顺序执行，与 defer 一致。
//
// 适用场景：
//   - 在辅助函数中创建的临时资源（defer 无法穿越函数边界）
//   - 需要在子测试结束后清理的资源
//   - 比 TestMain 更细粒度的清理

// createTempFile 是一个辅助函数，创建临时文件并通过 t.Cleanup 注册清理。
// 如果用 defer，则文件会在此函数返回时立刻被删除，测试无法使用！
func createTempFile(t *testing.T, dir, content string) string {
	t.Helper()

	f, err := os.CreateTemp(dir, "test-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}

	if _, err := f.WriteString(content); err != nil {
		f.Close()
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()

	// 关键：在 helper 函数中注册清理，测试结束后才执行
	t.Cleanup(func() {
		os.Remove(f.Name())
	})

	return f.Name()
}

func TestCleanup_TempFile(t *testing.T) {
	// 使用 TestMain 创建的临时目录
	path := createTempFile(t, testTmpDir, "hello world")

	// 文件在测试期间可用
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read temp file: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("content = %q, want %q", string(data), "hello world")
	}

	// 测试结束后 t.Cleanup 会自动删除文件
}

func TestCleanup_LIFOOrder(t *testing.T) {
	// 演示 LIFO 执行顺序
	var order []string

	t.Cleanup(func() { order = append(order, "first registered") })
	t.Cleanup(func() { order = append(order, "second registered") })
	t.Cleanup(func() {
		// 验证执行顺序：后注册的先执行
		order = append(order, "third registered")
		// 此时 order = ["third registered", "second registered", "first registered"]
		// 注意：这里无法用 t.Errorf 因为我们在 cleanup 中
	})
}

func TestCleanup_InSubtest(t *testing.T) {
	// 子测试中的 Cleanup 在子测试结束时执行
	dir, err := os.MkdirTemp(testTmpDir, "subtest-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	t.Run("write_and_read", func(t *testing.T) {
		path := filepath.Join(dir, "data.txt")
		if err := os.WriteFile(path, []byte("test data"), 0644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		t.Cleanup(func() { os.Remove(path) })

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read file: %v", err)
		}
		assertEqual(t, string(data), "test data")
	})
}
