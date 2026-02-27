package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

/*
陷阱：fmt.Sprintf 拼接 SQL 导致注入

运行：go run .

预期行为：
  使用 fmt.Sprintf 拼接用户输入到 SQL 语句中，攻击者可以通过构造
  恶意输入来修改 SQL 语义，实现未授权数据访问、数据篡改甚至删表。

  参数化查询（? 占位符）将用户输入作为参数传递给数据库驱动，
  驱动负责正确转义，SQL 结构不会被用户输入改变。

  正确做法：始终使用参数化查询（? 占位符），绝不拼接用户输入。
*/

func main() {
	tmpDir, err := os.MkdirTemp("", "sql-injection-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := sql.Open("sqlite", filepath.Join(tmpDir, "test.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 建表并插入测试数据
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT,
			role TEXT,
			password TEXT
		)
	`)
	if err != nil {
		log.Fatal(err)
	}
	db.Exec("INSERT INTO users (name, role, password) VALUES ('alice', 'admin', 'secret123')")
	db.Exec("INSERT INTO users (name, role, password) VALUES ('bob', 'user', 'pass456')")
	db.Exec("INSERT INTO users (name, role, password) VALUES ('charlie', 'user', 'mypass')")

	fmt.Println("=== 数据库初始状态 ===")
	printUsers(db)

	// --- 错误做法：fmt.Sprintf 拼接 SQL ---
	fmt.Println("\n=== 错误做法：fmt.Sprintf 拼接 SQL ===")

	// 攻击 1：万能条件绕过认证
	maliciousName := "' OR '1'='1"
	query := fmt.Sprintf("SELECT id, name, role FROM users WHERE name = '%s'", maliciousName) //nolint:gosec // 故意演示反例
	fmt.Printf("  拼接后 SQL: %s\n", query)
	rows, err := db.Query(query) //nolint:gosec // 故意演示反例
	if err != nil {
		fmt.Printf("  查询失败: %v\n", err)
	} else {
		fmt.Println("  注入结果（返回了所有用户！）:")
		for rows.Next() {
			var id int
			var name, role string
			rows.Scan(&id, &name, &role)
			fmt.Printf("    id=%d, name=%s, role=%s\n", id, name, role)
		}
		rows.Close()
	}

	// 攻击 2：UNION 注入获取密码
	fmt.Println("\n  攻击 2：UNION 注入")
	maliciousName2 := "' UNION SELECT id, password, role FROM users--"
	query2 := fmt.Sprintf("SELECT id, name, role FROM users WHERE name = '%s'", maliciousName2) //nolint:gosec // 故意演示反例
	fmt.Printf("  拼接后 SQL: %s\n", query2)
	rows2, err := db.Query(query2) //nolint:gosec // 故意演示反例
	if err != nil {
		fmt.Printf("  查询失败: %v\n", err)
	} else {
		fmt.Println("  注入结果（泄漏了密码！）:")
		for rows2.Next() {
			var id int
			var col2, col3 string
			rows2.Scan(&id, &col2, &col3)
			fmt.Printf("    id=%d, col2=%s, col3=%s\n", id, col2, col3)
		}
		rows2.Close()
	}

	// --- 正确做法：参数化查询 ---
	fmt.Println("\n=== 正确做法：参数化查询（? 占位符） ===")

	// 同样的恶意输入，但使用参数化查询
	fmt.Printf("  查询参数: %s\n", maliciousName)
	rows3, err := db.Query("SELECT id, name, role FROM users WHERE name = ?", maliciousName)
	if err != nil {
		fmt.Printf("  查询失败: %v\n", err)
	} else {
		count := 0
		for rows3.Next() {
			var id int
			var name, role string
			rows3.Scan(&id, &name, &role)
			fmt.Printf("    id=%d, name=%s, role=%s\n", id, name, role)
			count++
		}
		rows3.Close()
		if count == 0 {
			fmt.Println("  结果为空（恶意输入被当作普通字符串，注入无效）")
		}
	}

	// LIKE 子句的安全写法
	fmt.Println("\n=== LIKE 子句安全写法 ===")
	searchTerm := "ali"
	// 错误：拼接 LIKE
	badLike := fmt.Sprintf("SELECT name FROM users WHERE name LIKE '%%%s%%'", searchTerm) //nolint:gosec // 故意演示
	fmt.Printf("  错误: %s\n", badLike)
	// 正确：参数化 LIKE
	fmt.Println("  正确: SELECT name FROM users WHERE name LIKE '%' || ? || '%'")
	rows4, _ := db.Query("SELECT name FROM users WHERE name LIKE '%' || ? || '%'", searchTerm)
	for rows4.Next() {
		var name string
		rows4.Scan(&name)
		fmt.Printf("    找到: %s\n", name)
	}
	rows4.Close()

	fmt.Println("\n总结:")
	fmt.Println("  1. 绝不用 fmt.Sprintf 拼接用户输入到 SQL")
	fmt.Println("  2. 始终用 ? 占位符（PostgreSQL 用 $1, $2）")
	fmt.Println("  3. LIKE 子句用 '%' || ? || '%' 而非字符串拼接")
	fmt.Println("  4. IN 子句动态生成占位符: IN (?, ?, ?) 配合 ...args")
	fmt.Println("  5. gosec G201 规则可自动检测 SQL 拼接")
}

func printUsers(db *sql.DB) {
	rows, _ := db.Query("SELECT id, name, role FROM users")
	defer rows.Close()
	for rows.Next() {
		var id int
		var name, role string
		rows.Scan(&id, &name, &role)
		fmt.Printf("  id=%d, name=%s, role=%s\n", id, name, role)
	}
}
