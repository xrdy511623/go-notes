package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

/*
陷阱：测试共享数据库状态，并行运行时互相干扰

运行：go run .

预期行为：
  模拟两个测试操作同一行数据（ID=1），并行运行时产生竞态：
  - TestCreate: INSERT (id=1)
  - TestUpdate: UPDATE WHERE id=1
  运行多次后你会看到结果不确定——有时全通过，有时 TestUpdate 失败。

  正确做法：每个测试使用独立数据（唯一 ID），互不干扰。
*/

// fakeDB 模拟一个简易数据库
type fakeDB struct {
	mu   sync.RWMutex
	rows map[int]int // id → balance
}

func (db *fakeDB) Insert(id, balance int) error {
	// 模拟网络延迟
	time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
	db.mu.Lock()
	defer db.mu.Unlock()
	if _, exists := db.rows[id]; exists {
		return fmt.Errorf("UNIQUE constraint failed: id=%d already exists", id)
	}
	db.rows[id] = balance
	return nil
}

func (db *fakeDB) Update(id, delta int) (int, error) {
	time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)
	db.mu.Lock()
	defer db.mu.Unlock()
	bal, exists := db.rows[id]
	if !exists {
		return 0, fmt.Errorf("no row with id=%d", id)
	}
	db.rows[id] = bal + delta
	return db.rows[id], nil
}

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	fmt.Println("=== 错误做法：两个测试操作同一个 ID ===")
	fmt.Println()

	// 运行 10 轮，展示结果的不确定性
	passCount := 0
	for round := 1; round <= 10; round++ {
		db := &fakeDB{rows: make(map[int]int)}

		var wg sync.WaitGroup
		var mu sync.Mutex
		results := make(map[string]string)

		// TestCreate: INSERT id=1
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := db.Insert(1, 100)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results["TestCreate"] = fmt.Sprintf("FAIL: %v", err)
			} else {
				results["TestCreate"] = "PASS"
			}
		}()

		// TestUpdate: UPDATE id=1（假设 id=1 已存在）
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := db.Update(1, 50)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results["TestUpdate"] = fmt.Sprintf("FAIL: %v", err)
			} else {
				results["TestUpdate"] = "PASS"
			}
		}()

		wg.Wait()

		allPass := results["TestCreate"] == "PASS" && results["TestUpdate"] == "PASS"
		if allPass {
			passCount++
		}
		symbol := "✓"
		if !allPass {
			symbol = "✗"
		}
		fmt.Printf("  第%2d轮 %s  TestCreate=%-6s  TestUpdate=%s\n",
			round, symbol, results["TestCreate"], results["TestUpdate"])
	}
	fmt.Printf("\n  10 轮中 %d 轮全部通过 — 结果不确定，这就是 Flaky Test\n", passCount)

	fmt.Println()
	fmt.Println("=== 正确做法：每个测试使用独立数据 ===")
	fmt.Println()

	passCount = 0
	for round := 1; round <= 10; round++ {
		db := &fakeDB{rows: make(map[int]int)}

		var wg sync.WaitGroup
		var mu sync.Mutex
		results := make(map[string]string)

		// TestCreate: 使用 uniqueID = round*100+1
		wg.Add(1)
		go func(uid int) {
			defer wg.Done()
			err := db.Insert(uid, 100)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results["TestCreate"] = fmt.Sprintf("FAIL: %v", err)
			} else {
				results["TestCreate"] = "PASS"
			}
		}(round*100 + 1)

		// TestUpdate: 自己先 INSERT 自己的数据，再 UPDATE
		wg.Add(1)
		go func(uid int) {
			defer wg.Done()
			db.Insert(uid, 100) // 自己准备数据
			_, err := db.Update(uid, 50)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results["TestUpdate"] = fmt.Sprintf("FAIL: %v", err)
			} else {
				results["TestUpdate"] = "PASS"
			}
		}(round*100 + 2)

		wg.Wait()

		allPass := results["TestCreate"] == "PASS" && results["TestUpdate"] == "PASS"
		if allPass {
			passCount++
		}
		fmt.Printf("  第%2d轮 ✓  TestCreate=%-6s  TestUpdate=%s\n",
			round, results["TestCreate"], results["TestUpdate"])
	}
	fmt.Printf("\n  10 轮中 %d 轮全部通过 — 每个测试独立数据，100%% 确定性\n", passCount)
}
