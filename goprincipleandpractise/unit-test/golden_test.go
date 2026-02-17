package unittest

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------- Golden File：快照测试 ----------
//
// Golden File 测试将函数的实际输出与预先保存的"黄金文件"进行比较。
// 当输出格式发生变化时，用 -update 标志重新生成黄金文件。
//
// 适用场景：
//   - 格式化输出（报告、模板渲染）
//   - 序列化结果（JSON、YAML）
//   - 编译器/代码生成器输出
//
// 使用方式：
//   go test -run TestGoldenFile ./...              # 对比现有快照
//   go test -run TestGoldenFile ./... -update      # 重新生成快照

var update = flag.Bool("update", false, "update golden files")

// GenerateReport 生成一份格式化的用户报告
func GenerateReport(users []User) string {
	var b strings.Builder
	b.WriteString("=== User Report ===\n")
	b.WriteString(fmt.Sprintf("Total Users: %d\n", len(users)))
	b.WriteString("-------------------\n")
	for i, u := range users {
		b.WriteString(fmt.Sprintf("[%d] ID: %s\n", i+1, u.ID))
		b.WriteString(fmt.Sprintf("    Name:  %s\n", u.Name))
		b.WriteString(fmt.Sprintf("    Email: %s\n", u.Email))
	}
	b.WriteString("=== End Report ===\n")
	return b.String()
}

func TestGoldenFile_UserReport(t *testing.T) {
	users := []User{
		{ID: "1", Name: "Alice", Email: "alice@example.com"},
		{ID: "2", Name: "Bob", Email: "bob@example.com"},
		{ID: "3", Name: "Charlie", Email: "charlie@example.com"},
	}

	actual := GenerateReport(users)

	goldenPath := filepath.Join("testdata", "user_report.golden")

	if *update {
		// 重新生成黄金文件
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatalf("create testdata dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(actual), 0644); err != nil {
			t.Fatalf("update golden file: %v", err)
		}
		t.Log("golden file updated")
		return
	}

	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file (run with -update to create): %v", err)
	}

	if actual != string(expected) {
		t.Errorf("output mismatch (run with -update to regenerate)\ngot:\n%s\nwant:\n%s", actual, string(expected))
	}
}
