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
陷阱：事务中混用 db 和 tx 导致查询结果不一致

运行：go run .

预期行为：
  事务（tx）绑定到一个特定的数据库连接。
  事务内的所有操作必须通过 tx 执行，才能看到该事务中的未提交更改。
  如果在事务中使用 db.Query()（而非 tx.Query()），
  db 会从连接池中取一个不同的连接，该连接看不到当前事务的未提交更改。

  正确做法：事务内所有操作使用 tx.Query/tx.Exec/tx.QueryRow
*/

func main() {
	// 使用临时文件 + WAL 模式，WAL 允许写事务期间其他连接读取（读到旧版本）
	tmpDir, err := os.MkdirTemp("", "tx-db-mixed-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(5)

	// 启用 WAL 模式，允许并发读写
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("CREATE TABLE accounts (id INTEGER PRIMARY KEY, name TEXT, balance REAL)")
	db.Exec("INSERT INTO accounts (name, balance) VALUES ('Alice', 1000)")

	fmt.Println("=== 初始状态 ===")
	printBalance(db, "Alice")

	fmt.Println("\n=== 错误做法：事务中用 db.Query ===")
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	// 在事务中修改余额
	_, err = tx.Exec("UPDATE accounts SET balance = balance - 200 WHERE name = 'Alice'")
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	fmt.Println("事务中执行: UPDATE balance - 200")

	// 错误：用 db.Query 查询——使用的是不同的连接！
	var balance float64
	err = db.QueryRow("SELECT balance FROM accounts WHERE name = 'Alice'").Scan(&balance)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	fmt.Printf("用 db.QueryRow 查询余额: %.0f（看到的是未修改的值！）\n", balance)

	// 正确：用 tx.Query 查询——使用事务绑定的连接
	err = tx.QueryRow("SELECT balance FROM accounts WHERE name = 'Alice'").Scan(&balance)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	fmt.Printf("用 tx.QueryRow 查询余额: %.0f（看到的是事务中的修改）\n", balance)

	tx.Rollback()
	fmt.Println("事务已回滚")

	fmt.Println("\n=== 正确做法：事务内一致使用 tx ===")
	err = transfer(db, "Alice", 200)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n总结:")
	fmt.Println("  1. 事务绑定到一个连接，只有通过 tx 执行的操作才在该事务中")
	fmt.Println("  2. db.Query/db.Exec 使用连接池中的其他连接，看不到未提交的更改")
	fmt.Println("  3. 事务内的所有操作必须使用 tx.Query/tx.Exec/tx.QueryRow")
}

func transfer(db *sql.DB, name string, amount float64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 所有操作都通过 tx
	var balance float64
	err = tx.QueryRow("SELECT balance FROM accounts WHERE name = ?", name).Scan(&balance)
	if err != nil {
		return fmt.Errorf("查询余额: %w", err)
	}
	fmt.Printf("事务内查询余额: %.0f\n", balance)

	if balance < amount {
		return fmt.Errorf("余额不足: %.0f < %.0f", balance, amount)
	}

	_, err = tx.Exec("UPDATE accounts SET balance = balance - ? WHERE name = ?", amount, name)
	if err != nil {
		return fmt.Errorf("扣款: %w", err)
	}

	err = tx.QueryRow("SELECT balance FROM accounts WHERE name = ?", name).Scan(&balance)
	if err != nil {
		return fmt.Errorf("查询扣款后余额: %w", err)
	}
	fmt.Printf("扣款后余额: %.0f\n", balance)

	return tx.Commit()
}

func printBalance(db *sql.DB, name string) {
	var balance float64
	err := db.QueryRow("SELECT balance FROM accounts WHERE name = ?", name).Scan(&balance)
	if err != nil {
		log.Printf("查询余额失败: %v", err)
		return
	}
	fmt.Printf("%s 的余额: %.0f\n", name, balance)
}
