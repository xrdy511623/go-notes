package main

import (
	"database/sql"
	"fmt"
	"log"
	"runtime"

	_ "modernc.org/sqlite"
)

/*
陷阱：循环中 Prepare 但不 Close 导致 Stmt 泄漏

运行：go run .

预期行为：
  每次 db.Prepare() 都会在数据库服务端创建一个预编译语句，
  并在 database/sql 内部注册跟踪信息。
  如果在循环中反复 Prepare 但不 Close：
  1. 数据库服务端的预编译语句数量持续增长
  2. 客户端内存持续增长（每个 Stmt 都持有引用）
  3. 最终可能触发数据库的 max_prepared_stmt_count 限制

  正确做法：
  - 循环外 Prepare 一次，循环内复用
  - 如果必须在循环内 Prepare，必须在每次迭代中 Close
*/

func main() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Exec("CREATE TABLE logs (id INTEGER PRIMARY KEY AUTOINCREMENT, msg TEXT)")

	fmt.Println("=== 错误做法：循环内 Prepare 不 Close ===")
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	heapBefore := ms.HeapInuse

	for i := 0; i < 10000; i++ {
		// 错误：每次循环都 Prepare，但从不 Close
		stmt, err := db.Prepare("INSERT INTO logs (msg) VALUES (?)") //nolint:sqlclosecheck
		if err != nil {
			log.Fatal(err)
		}
		stmt.Exec(fmt.Sprintf("msg_%d", i))
		// 没有 stmt.Close()！
	}

	runtime.GC()
	runtime.ReadMemStats(&ms)
	heapAfter := ms.HeapInuse

	stats := db.Stats()
	fmt.Printf("  执行 10000 次 Prepare 后:\n")
	fmt.Printf("  堆内存增长: %d KB\n", (heapAfter-heapBefore)/1024)
	fmt.Printf("  连接池: Open=%d, InUse=%d\n", stats.OpenConnections, stats.InUse)

	fmt.Println("\n=== 正确做法一：循环外 Prepare，循环内复用 ===")
	db2, _ := sql.Open("sqlite", ":memory:")
	defer db2.Close()
	db2.Exec("CREATE TABLE logs (id INTEGER PRIMARY KEY AUTOINCREMENT, msg TEXT)")

	runtime.GC()
	runtime.ReadMemStats(&ms)
	heapBefore = ms.HeapInuse

	// 正确：只 Prepare 一次
	stmt, err := db2.Prepare("INSERT INTO logs (msg) VALUES (?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close() // 用完后关闭

	for i := 0; i < 10000; i++ {
		stmt.Exec(fmt.Sprintf("msg_%d", i))
	}

	runtime.GC()
	runtime.ReadMemStats(&ms)
	heapAfter = ms.HeapInuse
	fmt.Printf("  堆内存增长: %d KB\n", (heapAfter-heapBefore)/1024)

	fmt.Println("\n=== 正确做法二：循环内 Prepare + 立即 Close ===")
	db3, _ := sql.Open("sqlite", ":memory:")
	defer db3.Close()
	db3.Exec("CREATE TABLE logs (id INTEGER PRIMARY KEY AUTOINCREMENT, msg TEXT)")

	for i := 0; i < 5; i++ {
		stmt, err := db3.Prepare("INSERT INTO logs (msg) VALUES (?)")
		if err != nil {
			log.Fatal(err)
		}
		stmt.Exec(fmt.Sprintf("msg_%d", i))
		stmt.Close() // 正确：立即关闭（循环中不适合用 defer）
	}
	fmt.Println("  每次迭代中 Close，无泄漏")

	fmt.Println("\n总结:")
	fmt.Println("  1. db.Prepare() 在数据库和客户端都分配资源，必须配对 Close()")
	fmt.Println("  2. 高频场景：循环外 Prepare 一次，循环内复用 stmt.Exec()")
	fmt.Println("  3. 循环内 Prepare 时，不要用 defer（defer 在函数返回才执行），直接调用 Close()")
}
