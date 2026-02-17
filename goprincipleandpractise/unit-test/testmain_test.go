package unittest

import (
	"fmt"
	"os"
	"testing"
)

// ---------- TestMain：全局 Setup 与 Teardown ----------
//
// TestMain 是整个 package 测试的入口点。它在所有测试函数之前执行，
// 提供了全局 setup/teardown 的机会。
//
// 注意事项：
//   - 一个 package 只能有一个 TestMain 函数
//   - 必须调用 m.Run() 来执行测试，否则不会运行任何测试
//   - 必须调用 os.Exit() 传递 m.Run() 的退出码
//
// TestMain vs t.Cleanup 的选择：
//   - TestMain：适合全局、昂贵的资源（数据库连接、临时目录、环境变量）
//   - t.Cleanup：适合单个测试的资源清理（文件、mock 重置）
//     详见 cleanup_test.go

// testTmpDir 供其他测试使用的临时目录
var testTmpDir string

func TestMain(m *testing.M) {
	// ── Setup ──
	fmt.Println("=== TestMain: global setup")

	// 创建临时目录
	var err error
	testTmpDir, err = os.MkdirTemp("", "unittest-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}

	// 设置测试环境变量
	os.Setenv("UNIT_TEST_MODE", "true")

	// ── Run ──
	exitCode := m.Run()

	// ── Teardown ──
	fmt.Println("=== TestMain: global teardown")
	os.RemoveAll(testTmpDir)
	os.Unsetenv("UNIT_TEST_MODE")

	os.Exit(exitCode)
}

// TestTestMainSetup 验证 TestMain 的 setup 是否生效
func TestTestMainSetup(t *testing.T) {
	// 验证临时目录存在
	if _, err := os.Stat(testTmpDir); os.IsNotExist(err) {
		t.Fatal("testTmpDir does not exist; TestMain setup failed")
	}

	// 验证环境变量
	if v := os.Getenv("UNIT_TEST_MODE"); v != "true" {
		t.Errorf("UNIT_TEST_MODE = %q, want %q", v, "true")
	}
}
