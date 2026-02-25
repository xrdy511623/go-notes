package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

/*
陷阱：忽略 rows.Err() 导致静默丢失数据

运行：go run .

预期行为：
  rows.Next() 返回 false 有两种原因：
  1. 数据已全部读取完毕（正常结束）
  2. 迭代过程中发生了错误（网络中断、context 超时等）

  如果不检查 rows.Err()，第二种情况下程序会静默地丢失部分数据，
  而开发者完全不知道查询没有完整完成。

  正确做法：在 for rows.Next() 循环之后，始终检查 rows.Err()
*/

func main() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 建表并插入大量数据
	db.Exec("CREATE TABLE data (id INTEGER PRIMARY KEY, value TEXT)")
	tx, _ := db.Begin()
	for i := 1; i <= 1000; i++ {
		tx.Exec("INSERT INTO data (value) VALUES (?)", fmt.Sprintf("row_%d", i))
	}
	tx.Commit()

	fmt.Println("=== 演示：context 超时导致迭代中断 ===")

	// 用一个很短的超时来模拟中途中断
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// 等一下让 context 超时
	time.Sleep(5 * time.Millisecond)

	rows, err := db.QueryContext(ctx, "SELECT id, value FROM data")
	if err != nil {
		fmt.Printf("查询失败（context 已超时）: %v\n", err)
		fmt.Println("\n--- 换用正常 context 演示 rows.Err() 的重要性 ---")
	} else {
		defer rows.Close()

		count := 0
		for rows.Next() {
			var id int
			var value string
			rows.Scan(&id, &value)
			count++
		}
		fmt.Printf("  读取了 %d 行\n", count)

		// 关键：检查 rows.Err()
		if err := rows.Err(); err != nil {
			fmt.Printf("  rows.Err() = %v\n", err)
			fmt.Println("  ⚠ 数据不完整！如果忽略此错误，程序会使用不完整的数据继续运行")
		}
	}

	fmt.Println("\n=== 对比：正确做法 vs 错误做法 ===")

	fmt.Println("\n--- 错误做法：忽略 rows.Err() ---")
	results1 := queryWithoutErrCheck(db)
	fmt.Printf("  获取到 %d 条结果（但不知道是否完整）\n", len(results1))

	fmt.Println("\n--- 正确做法：检查 rows.Err() ---")
	results2, err := queryWithErrCheck(db)
	if err != nil {
		fmt.Printf("  查询出错: %v\n", err)
	} else {
		fmt.Printf("  获取到 %d 条完整结果\n", len(results2))
	}

	fmt.Println("\n总结:")
	fmt.Println("  1. rows.Next() 返回 false 不代表 '数据读完了'，也可能是出错了")
	fmt.Println("  2. 必须在 for rows.Next() 循环后检查 rows.Err()")
	fmt.Println("  3. 忽略 rows.Err() 会导致程序使用不完整的数据，引发难以追踪的 bug")
}

// 错误做法：不检查 rows.Err()
func queryWithoutErrCheck(db *sql.DB) []string {
	rows, err := db.Query("SELECT value FROM data")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var v string
		rows.Scan(&v)
		results = append(results, v)
	}
	// 错误：没有检查 rows.Err()
	return results
}

// 正确做法：检查 rows.Err()
func queryWithErrCheck(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT value FROM data")
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return results, nil
}
