package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

/*
陷阱：不设置 MaxOpenConns 导致高并发下连接爆炸

运行：go run .

预期行为：
  sql.DB 的 MaxOpenConns 默认为 0，即无限制。
  在高并发场景下，每个 goroutine 都可能打开一个新连接。
  如果有 1000 个并发请求，可能同时打开 1000 个连接，导致：
  1. 数据库报 "too many connections" 错误
  2. 操作系统 file descriptor 耗尽
  3. 数据库服务器内存溢出

  正确做法：必须设置 SetMaxOpenConns() 限制最大连接数
*/

func main() {
	fmt.Println("=== 错误做法：不限制最大连接数 ===")
	db1, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db1.Close()
	// 没有设置 MaxOpenConns！默认 0 = 无限制
	db1.Exec("CREATE TABLE data (id INTEGER PRIMARY KEY, value TEXT)")

	simulateConcurrent(db1, "无限制", 100)

	fmt.Println("\n=== 正确做法：设置合理的最大连接数 ===")
	db2, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db2.Close()
	db2.SetMaxOpenConns(10) // 限制最大连接数
	db2.SetMaxIdleConns(10) // 空闲连接数等于最大连接数
	db2.Exec("CREATE TABLE data (id INTEGER PRIMARY KEY, value TEXT)")

	simulateConcurrent(db2, "MaxOpen=10", 100)

	fmt.Println("\n总结:")
	fmt.Println("  1. MaxOpenConns 默认为 0（无限制），高并发下会打开大量连接")
	fmt.Println("  2. 必须设置 SetMaxOpenConns()，推荐值 25-50（根据数据库和负载调整）")
	fmt.Println("  3. SetMaxIdleConns 设为 MaxOpenConns 的 50-100%，避免频繁创建连接")
	fmt.Println("  4. 设置 MaxOpenConns 后，超出限制的请求会排队等待，而不是创建新连接")
}

func simulateConcurrent(db *sql.DB, label string, concurrency int) {
	var wg sync.WaitGroup
	var maxOpen int
	var mu sync.Mutex

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			// 模拟一个耗时查询
			_, err := db.Exec("INSERT INTO data (value) VALUES (?)", fmt.Sprintf("val_%d", n))
			if err != nil {
				return
			}

			stats := db.Stats()
			mu.Lock()
			if stats.OpenConnections > maxOpen {
				maxOpen = stats.OpenConnections
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)
	stats := db.Stats()

	fmt.Printf("\n[%s] %d 个并发请求完成:\n", label, concurrency)
	fmt.Printf("  峰值连接数: %d\n", maxOpen)
	fmt.Printf("  最终连接数: Open=%d, InUse=%d, Idle=%d\n",
		stats.OpenConnections, stats.InUse, stats.Idle)
	fmt.Printf("  等待次数: %d, 等待总时长: %v\n",
		stats.WaitCount, stats.WaitDuration)
	fmt.Printf("  耗时: %v\n", elapsed)
}
