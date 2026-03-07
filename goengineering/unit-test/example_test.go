package unittest

import "fmt"

// ---------- Example_xxx：文档测试 ----------
//
// Example 函数具有双重用途：
// 1. 作为测试运行：go test 会执行并验证 // Output: 注释
// 2. 作为文档展示：godoc 会将其展示在 API 文档中
//
// 命名规则：
//   - Example()             → 包级别示例
//   - ExampleFuncName()     → 函数示例
//   - ExampleType()         → 类型示例
//   - ExampleType_Method()  → 方法示例
//   - Example_suffix()      → 带后缀的变体

// Example_getMaxOnlineTime 展示 getMaxOnlineTime 的基本用法。
// 因为 getMaxOnlineTime 是未导出函数，使用 Example_ 前缀的包级示例。
func Example_getMaxOnlineTime() {
	users := []*userOnLine{
		{uid: 1, loginTime: 100, logoutTime: 300},
		{uid: 2, loginTime: 200, logoutTime: 400},
		{uid: 3, loginTime: 250, logoutTime: 350},
	}
	peakTime := getMaxOnlineTime(users)
	fmt.Printf("Peak online time: %d seconds\n", peakTime)
	// Output: Peak online time: 250 seconds
}

// ExampleGenerateReport 展示报告生成
func ExampleGenerateReport() {
	users := []User{
		{ID: "1", Name: "Alice", Email: "alice@example.com"},
	}
	report := GenerateReport(users)
	fmt.Print(report)
	// Output:
	// === User Report ===
	// Total Users: 1
	// -------------------
	// [1] ID: 1
	//     Name:  Alice
	//     Email: alice@example.com
	// === End Report ===
}

// ExampleNewUserService 展示 UserService 的创建和使用
func ExampleNewUserService() {
	store := &StubUserStore{
		GetByIDFunc: func(id string) (*User, error) {
			return &User{ID: id, Name: "Alice"}, nil
		},
	}
	svc := NewUserService(store)
	user, err := svc.GetUser("1")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("User: %s\n", user.Name)
	// Output: User: Alice
}
