package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

/*
陷阱：defer rows.Close() 放在错误检查之前导致 panic

运行：go run .

预期行为：
  当 db.Query() 返回错误时，rows 为 nil。
  如果在 err 检查之前写了 defer rows.Close()，
  会在函数返回时对 nil 指针调用方法，触发 panic。

  正确做法：先检查 err，确认 rows 非 nil 后再 defer rows.Close()
*/

func main() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")

	fmt.Println("=== 错误做法：defer 放在 err 检查之前 ===")
	fmt.Println("以下代码会 panic（已用 recover 保护）:")
	fmt.Println()
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("  捕获 panic: %v\n", r)
			}
		}()

		// 查询一个不存在的表，故意触发错误
		rows, err := db.Query("SELECT * FROM nonexistent_table")
		// 错误：defer 放在 err 检查之前
		defer rows.Close() // rows 为 nil → panic!
		if err != nil {
			fmt.Printf("  查询错误: %v\n", err)
			return
		}
		_ = rows
	}()

	fmt.Println("\n=== 正确做法：先检查 err，再 defer Close ===")
	err = queryCorrect(db)
	if err != nil {
		fmt.Printf("  查询返回错误（正常处理，无 panic）: %v\n", err)
	}

	// 正常查询也演示正确写法
	db.Exec("INSERT INTO users (name) VALUES ('Alice')")
	err = queryCorrect(db)
	if err != nil {
		fmt.Printf("  查询失败: %v\n", err)
	}

	fmt.Println("\n总结:")
	fmt.Println("  1. db.Query() 返回错误时 rows 为 nil")
	fmt.Println("  2. 对 nil 调用 Close() 会 panic: nil pointer dereference")
	fmt.Println("  3. 正确顺序: rows, err := db.Query(...) → if err != nil { return } → defer rows.Close()")
}

func queryCorrect(db *sql.DB) error {
	rows, err := db.Query("SELECT id, name FROM users")
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close() // 正确：err 检查之后

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		fmt.Printf("  读取到: id=%d, name=%s\n", id, name)
	}
	return rows.Err()
}
