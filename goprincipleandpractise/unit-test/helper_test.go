package unittest

import "testing"

// ---------- t.Helper()：自定义断言 ----------
//
// t.Helper() 将当前函数标记为测试辅助函数。
// 当辅助函数中调用 t.Errorf/t.Fatalf 时，
// 错误信息中显示的行号将指向调用方（测试函数），而非辅助函数内部。
//
// 不加 t.Helper()：
//   helper_test.go:25: got "foo", want "bar"   ← 指向辅助函数内部，不直观
//
// 加了 t.Helper()：
//   helper_test.go:48: got "foo", want "bar"   ← 指向测试函数的调用行，一眼定位

// assertEqual 是一个通用的相等断言辅助函数
func assertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper() // 关键：标记为辅助函数
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// requireNoError 断言 err 为 nil，否则立即终止测试
func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// requireError 断言 err 不为 nil
func requireError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------- 测试 ----------

func TestHelper_ErrorLineNumbers(t *testing.T) {
	// 使用自定义断言，错误行号会指向这里，而不是 assertEqual 内部
	assertEqual(t, 1+1, 2)
	assertEqual(t, "hello", "hello")

	store := &StubUserStore{
		GetByIDFunc: func(id string) (*User, error) {
			return &User{ID: id, Name: "Test"}, nil
		},
	}
	svc := NewUserService(store)
	user, err := svc.GetUser("1")
	requireNoError(t, err)
	assertEqual(t, user.Name, "Test")
}

func TestHelper_WithSubtests(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"1+1", 1, 1, 2},
		{"2+3", 2, 3, 5},
		{"0+0", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.a+tt.b, tt.want)
		})
	}
}
