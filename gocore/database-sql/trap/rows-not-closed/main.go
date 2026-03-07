package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

/*
陷阱：不关闭 Rows 导致连接泄漏

运行：go run .

预期行为：
  db.Query() 从连接池取出一个连接并绑定到返回的 *Rows。
  如果不调用 rows.Close()，该连接永远不会归还到连接池。
  当连接池耗尽（达到 MaxOpenConns）时，后续查询会阻塞直到超时。

  正确做法：始终在 err 检查之后 defer rows.Close()
*/

func main() {
	tmpDir, err := os.MkdirTemp("", "rows-not-closed-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := sql.Open("sqlite", filepath.Join(tmpDir, "test.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(3) // 故意设小，便于观察泄漏

	// 建表并插入测试数据
	db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	for i := 1; i <= 10; i++ {
		db.Exec("INSERT INTO users (name) VALUES (?)", fmt.Sprintf("user_%d", i))
	}

	fmt.Println("=== 错误做法：不关闭 Rows ===")
	for i := 1; i <= 5; i++ {
		stats := db.Stats()
		fmt.Printf("  第 %d 次查询前: InUse=%d, Idle=%d, Open=%d\n",
			i, stats.InUse, stats.Idle, stats.OpenConnections)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		// 错误：Query 返回的 rows 没有 Close
		rows, err := db.QueryContext(ctx, "SELECT id, name FROM users") //nolint:sqlclosecheck
		cancel()
		if err != nil {
			fmt.Printf("  第 %d 次查询失败（连接池已耗尽）: %v\n", i, err)
			continue
		}
		// 只读取第一行就丢弃 rows，不调用 Close
		if rows.Next() {
			var id int
			var name string
			rows.Scan(&id, &name)
			fmt.Printf("  读取到: id=%d, name=%s\n", id, name)
		}
		// rows 没有 Close！连接不会归还到池中
	}

	stats := db.Stats()
	fmt.Printf("\n泄漏后: InUse=%d, Idle=%d, Open=%d (MaxOpen=3)\n",
		stats.InUse, stats.Idle, stats.OpenConnections)

	// 尝试在连接池耗尽后执行查询（会阻塞/超时）
	fmt.Println("\n尝试在连接池耗尽后查询（2秒超时）...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = db.QueryContext(ctx, "SELECT 1")
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
	}

	fmt.Println("\n=== 正确做法：始终 defer rows.Close() ===")
	db2, _ := sql.Open("sqlite", filepath.Join(tmpDir, "test2.db"))
	defer db2.Close()
	db2.SetMaxOpenConns(3)
	db2.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
	for i := 1; i <= 10; i++ {
		db2.Exec("INSERT INTO users (name) VALUES (?)", fmt.Sprintf("user_%d", i))
	}

	var wg sync.WaitGroup
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			rows, err := db2.Query("SELECT id, name FROM users")
			if err != nil {
				fmt.Printf("  goroutine %d 查询失败: %v\n", n, err)
				return
			}
			defer rows.Close() // 正确：始终 Close

			count := 0
			for rows.Next() {
				var id int
				var name string
				rows.Scan(&id, &name)
				count++
			}
			if err := rows.Err(); err != nil {
				fmt.Printf("  goroutine %d 迭代错误: %v\n", n, err)
			}
		}(i)
	}
	wg.Wait()

	stats2 := db2.Stats()
	fmt.Printf("正确关闭后: InUse=%d, Idle=%d, Open=%d\n",
		stats2.InUse, stats2.Idle, stats2.OpenConnections)

	fmt.Println("\n总结:")
	fmt.Println("  1. db.Query() 返回的 *Rows 持有一个连接，必须 Close 归还")
	fmt.Println("  2. 始终在 err 检查之后写 defer rows.Close()")
	fmt.Println("  3. 不关闭会导致连接池耗尽，后续查询阻塞或超时")
}
