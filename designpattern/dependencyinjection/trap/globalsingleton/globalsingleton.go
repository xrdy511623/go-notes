// Package globalsingleton 演示"全局单例"反模式。
//
// 这是依赖注入的常见对立面：将依赖藏在全局变量中，函数直接访问全局状态。
//
// ⚠️  此代码仅作反面教材，请勿在生产环境中效仿。
//
// 问题清单：
//
//  1. 全局状态导致测试互相污染
//     - TestA 写入的数据会被 TestB 看到
//     - 无法安全地并行执行测试（-race 会报警）
//
//  2. 隐藏依赖——调用方看不出 GetUserByID 依赖了数据库
//     - 函数签名 GetUserByID(id string) 没有任何参数暗示需要 DB
//     - 只有深入阅读实现才能发现它调用了全局 db
//
//  3. 无法替换实现
//     - 测试时想用 SQLite 替代 MySQL？必须修改全局变量
//     - 两个测试想用不同的存储？做不到——全局只有一份
//
//  4. 初始化顺序不确定
//     - 如果多个 init() 都依赖 db，执行顺序由编译器决定
//     - 跨包的 init() 依赖是 bug 的温床
package globalsingleton

import "database/sql"

// ─── 反模式：全局单例 ─────────────────────────────────────────

// db 是全局数据库连接——任何包都可以直接访问和修改。
var db *sql.DB

// InitDB 初始化全局数据库连接。
// 通常在 main() 或 init() 中调用一次。
func InitDB(dsn string) error {
	var err error
	db, err = sql.Open("mysql", dsn)
	return err
}

// GetDB 返回全局数据库连接。
// 调用方无法从函数签名判断这个函数依赖了什么——它看起来像纯函数，实际上读取全局状态。
func GetDB() *sql.DB {
	return db
}

// UserService 直接使用全局 db，没有通过构造函数接收依赖。
type UserService struct{}

// GetUserByID 演示全局依赖的典型用法。
//
// 正确做法（依赖注入）：
//
//	func (s *UserService) GetUserByID(id string) (*User, error) {
//	    return s.db.QueryRow(...)  // db 通过构造函数注入
//	}
//
// 反模式（全局单例）：
func (s *UserService) GetUserByID(id string) (string, error) {
	conn := GetDB() // 隐式依赖全局变量！
	if conn == nil {
		return "", errDBNotInit
	}
	// 使用全局 db 查询——测试时无法替换为 mock
	row := conn.QueryRow("SELECT name FROM users WHERE id = ?", id)
	var name string
	err := row.Scan(&name)
	return name, err
}

var errDBNotInit = &DBNotInitError{}

// DBNotInitError 表示数据库未初始化。
type DBNotInitError struct{}

func (e *DBNotInitError) Error() string {
	return "globalsingleton: database not initialized — did you forget to call InitDB()?"
}
