package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

/*
陷阱：用 db.Query() 执行非 SELECT 语句导致连接泄漏

运行：go run .

预期行为：
  db.Query() 返回 *Rows，即使执行的是 INSERT/UPDATE/DELETE。
  返回的 *Rows 持有一个连接，如果调用者忽略了这个返回值（不消费、不关闭），
  连接永远不会归还到池中，导致泄漏。

  正确做法：非 SELECT 语句使用 db.Exec()，它会自动释放连接。
*/

func main() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(5)
	db.Exec("CREATE TABLE logs (id INTEGER PRIMARY KEY AUTOINCREMENT, msg TEXT)")

	fmt.Println("=== 错误做法：用 Query 执行 INSERT ===")
	for i := 1; i <= 10; i++ {
		// 错误：用 Query 执行 INSERT，忽略返回的 *Rows
		//nolint:rowserrcheck,sqlclosecheck
		db.Query("INSERT INTO logs (msg) VALUES (?)", fmt.Sprintf("msg_%d", i))

		stats := db.Stats()
		fmt.Printf("  第 %2d 次 INSERT: InUse=%d, Open=%d, MaxOpenConns=5\n",
			i, stats.InUse, stats.OpenConnections)
	}

	stats := db.Stats()
	fmt.Printf("\n泄漏统计: InUse=%d, Open=%d\n", stats.InUse, stats.OpenConnections)

	fmt.Println("\n=== 正确做法：用 Exec 执行 INSERT ===")
	db2, _ := sql.Open("sqlite", ":memory:")
	defer db2.Close()
	db2.SetMaxOpenConns(5)
	db2.Exec("CREATE TABLE logs (id INTEGER PRIMARY KEY AUTOINCREMENT, msg TEXT)")

	for i := 1; i <= 10; i++ {
		// 正确：用 Exec 执行 INSERT
		result, err := db2.Exec("INSERT INTO logs (msg) VALUES (?)", fmt.Sprintf("msg_%d", i))
		if err != nil {
			fmt.Printf("  INSERT 失败: %v\n", err)
			continue
		}
		id, _ := result.LastInsertId()

		stats := db2.Stats()
		fmt.Printf("  第 %2d 次 INSERT (id=%d): InUse=%d, Open=%d\n",
			i, id, stats.InUse, stats.OpenConnections)
	}

	stats2 := db2.Stats()
	fmt.Printf("\n正确统计: InUse=%d, Open=%d\n", stats2.InUse, stats2.OpenConnections)

	fmt.Println("\n总结:")
	fmt.Println("  1. db.Query() 返回 *Rows，即使是 INSERT/DELETE 也会占用连接")
	fmt.Println("  2. 非 SELECT 语句始终使用 db.Exec()")
	fmt.Println("  3. db.Exec() 自动释放连接，并返回 Result（LastInsertId, RowsAffected）")
}
