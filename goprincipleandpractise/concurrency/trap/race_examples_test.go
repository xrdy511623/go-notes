package trap

import (
	"testing"
)

/*
数据竞争检测测试。

执行命令（必须带 -race 标志）:

	go test -race -run '^TestRace' -count=1 -v .

预期:
  - TestRaceCounter: 触发 WARNING: DATA RACE
  - TestRaceSliceAppend: 触发 WARNING: DATA RACE
  - TestSafeSliceIndex: 不触发竞争
  - TestRaceMap: 注释掉了，因为会直接 fatal（无法recover）

注意: -race 检测到竞争时进程会 exit(66)，不是panic。
*/

// TestRaceCounter 演示未保护计数器的数据竞争
// 用 go test -race 运行时会报 WARNING: DATA RACE
func TestRaceCounter(t *testing.T) {
	t.Skip("跳过：此测试会触发data race，用 go test -race -run TestRaceCounter 手动验证")
	result := RaceCounter()
	t.Logf("counter = %d (期望100，实际可能小于100)", result)
}

// TestRaceSliceAppend 演示slice并发append的数据竞争
func TestRaceSliceAppend(t *testing.T) {
	t.Skip("跳过：此测试会触发data race，用 go test -race -run TestRaceSliceAppend 手动验证")
	result := RaceSliceAppend()
	t.Logf("slice len = %d (期望100，实际可能小于100)", len(result))
}

// TestSafeSliceIndex 验证预分配+索引写入没有数据竞争
// 此测试即使带 -race 也不会报错
func TestSafeSliceIndex(t *testing.T) {
	result := SafeSliceIndex()
	if len(result) != 100 {
		t.Fatalf("len = %d, want 100", len(result))
	}
}
