package sender

import (
	"errors"
	"sync"
	"testing"
)

type stubSender struct{ name string }

func (s *stubSender) Send(string) error { return nil }

// closableStub 实现 Sender + Closer，用于测试 CloseAll。
type closableStub struct {
	name     string
	closeErr error
	closed   bool
}

func (c *closableStub) Send(string) error { return nil }
func (c *closableStub) Close() error      { c.closed = true; return c.closeErr }

func newStubFactory(name string) Factory {
	return func(cfg Config) (Sender, error) { return &stubSender{name: name}, nil }
}

func TestRegister_And_New(t *testing.T) {
	t.Cleanup(func() { Reset() })

	Register("email", newStubFactory("email"))

	s, err := New("email", nil)
	if err != nil {
		t.Fatalf("New(email) unexpected error: %v", err)
	}
	if stub, ok := s.(*stubSender); !ok || stub.name != "email" {
		t.Fatalf("expected stubSender{email}, got %T", s)
	}
}

func TestNew_ReturnsNewInstanceEachCall(t *testing.T) {
	t.Cleanup(func() { Reset() })

	Register("sms", newStubFactory("sms"))

	s1, _ := New("sms", nil)
	s2, _ := New("sms", nil)
	if s1 == s2 {
		t.Fatal("expected different instances, got same pointer")
	}
}

func TestNew_UnknownName(t *testing.T) {
	t.Cleanup(func() { Reset() })

	_, err := New("nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for unknown sender, got nil")
	}
}

func TestMustNew_Success(t *testing.T) {
	t.Cleanup(func() { Reset() })

	Register("sms", newStubFactory("sms"))
	s := MustNew("sms", nil)
	if s == nil {
		t.Fatal("MustNew returned nil")
	}
}

func TestMustNew_PanicsOnUnknown(t *testing.T) {
	t.Cleanup(func() { Reset() })

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unknown sender")
		}
	}()
	MustNew("nonexistent", nil)
}

func TestCount(t *testing.T) {
	t.Cleanup(func() { Reset() })

	if Count() != 0 {
		t.Fatalf("expected 0, got %d", Count())
	}
	Register("a", newStubFactory("a"))
	Register("b", newStubFactory("b"))
	if Count() != 2 {
		t.Fatalf("expected 2, got %d", Count())
	}
}

func TestRegister_EmptyNamePanics(t *testing.T) {
	t.Cleanup(func() { Reset() })

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty name")
		}
	}()
	Register("", newStubFactory("x"))
}

func TestRegister_NilFactoryPanics(t *testing.T) {
	t.Cleanup(func() { Reset() })

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil factory")
		}
	}()
	Register("valid", nil)
}

func TestRegister_DuplicatePanics(t *testing.T) {
	t.Cleanup(func() { Reset() })

	Register("dup", newStubFactory("dup"))
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for duplicate registration")
		}
	}()
	Register("dup", newStubFactory("dup"))
}

func TestList_Sorted(t *testing.T) {
	t.Cleanup(func() { Reset() })

	Register("weixin", newStubFactory("weixin"))
	Register("dingding", newStubFactory("dingding"))
	Register("feishu", newStubFactory("feishu"))

	names := List()
	expected := []string{"dingding", "feishu", "weixin"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d names, got %d", len(expected), len(names))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("List()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestList_EmptyRegistry(t *testing.T) {
	t.Cleanup(func() { Reset() })

	names := List()
	if len(names) != 0 {
		t.Fatalf("expected empty list, got %v", names)
	}
}

func TestConcurrentRegisterAndNew(t *testing.T) {
	t.Cleanup(func() { Reset() })

	const n = 100
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		name := string(rune('A'+i%26)) + string(rune('0'+i/26))
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Register(%q) unexpected panic: %v", name, r)
				}
			}()
			Register(name, newStubFactory(name))
		}(name)
	}
	wg.Wait()

	if got := Count(); got != n {
		t.Fatalf("expected %d registrations, got %d", n, got)
	}

	for _, name := range List() {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if s, err := New(name, nil); err != nil || s == nil {
				t.Errorf("concurrent New(%q) failed: %v", name, err)
			}
		}(name)
	}
	wg.Wait()
}

// --- 新增测试：参数化工厂 + 错误返回 ---

func TestNew_WithConfig(t *testing.T) {
	t.Cleanup(func() { Reset() })

	Register("cfg-test", func(cfg Config) (Sender, error) {
		return &stubSender{name: cfg["key"]}, nil
	})

	s, err := New("cfg-test", Config{"key": "my-value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub, ok := s.(*stubSender); !ok || stub.name != "my-value" {
		t.Fatalf("config not passed to factory: got %+v", s)
	}
}

func TestNew_NilConfig(t *testing.T) {
	t.Cleanup(func() { Reset() })

	Register("nil-cfg", func(cfg Config) (Sender, error) {
		// nil config 不应 panic，返回默认值
		name := cfg["key"] // map 的零值行为：对 nil map 取值返回零值
		return &stubSender{name: name}, nil
	})

	s, err := New("nil-cfg", nil)
	if err != nil {
		t.Fatalf("nil config should not cause error: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil Sender")
	}
}

func TestNew_FactoryError(t *testing.T) {
	t.Cleanup(func() { Reset() })

	Register("fail", func(cfg Config) (Sender, error) {
		return nil, errors.New("bad config")
	})

	_, err := New("fail", nil)
	if err == nil {
		t.Fatal("expected factory error, got nil")
	}
	if err.Error() != "bad config" {
		t.Fatalf("expected 'bad config', got %q", err.Error())
	}
}

func TestMustNew_FactoryErrorPanics(t *testing.T) {
	t.Cleanup(func() { Reset() })

	Register("fail", func(cfg Config) (Sender, error) {
		return nil, errors.New("factory boom")
	})

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from factory error")
		}
	}()
	MustNew("fail", nil)
}

// --- 新增测试：动态注销 ---

func TestUnregister_Exists(t *testing.T) {
	t.Cleanup(func() { Reset() })

	Register("removable", newStubFactory("removable"))
	if !Unregister("removable") {
		t.Fatal("expected Unregister to return true for existing entry")
	}
	if _, err := New("removable", nil); err == nil {
		t.Fatal("expected error after Unregister, got nil")
	}
}

func TestUnregister_NotExists(t *testing.T) {
	t.Cleanup(func() { Reset() })

	if Unregister("ghost") {
		t.Fatal("expected Unregister to return false for nonexistent entry")
	}
}

func TestUnregister_Concurrent(t *testing.T) {
	t.Cleanup(func() { Reset() })

	const n = 50
	for i := 0; i < n; i++ {
		name := string(rune('a' + i%26))
		// 防止重复注册 panic
		func() {
			defer func() { recover() }()
			Register(name, newStubFactory(name))
		}()
	}

	var wg sync.WaitGroup
	// 并发注册和注销
	for i := 0; i < n; i++ {
		name := string(rune('a' + i%26))
		wg.Add(2)
		go func(name string) {
			defer wg.Done()
			Unregister(name)
		}(name)
		go func(name string) {
			defer wg.Done()
			defer func() { recover() }() // 可能因并发重复注册而 panic
			Register(name, newStubFactory(name))
		}(name)
	}
	wg.Wait()
	// 不 panic 即为成功
}

// --- 新增测试：CloseAll ---

func TestCloseAll_Mixed(t *testing.T) {
	plain := &stubSender{name: "plain"}
	closable := &closableStub{name: "closable"}

	err := CloseAll([]Sender{plain, closable})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !closable.closed {
		t.Fatal("expected closable sender to be closed")
	}
}

func TestCloseAll_Empty(t *testing.T) {
	err := CloseAll(nil)
	if err != nil {
		t.Fatalf("CloseAll(nil) should return nil, got %v", err)
	}
	err = CloseAll([]Sender{})
	if err != nil {
		t.Fatalf("CloseAll([]) should return nil, got %v", err)
	}
}

func TestCloseAll_Error(t *testing.T) {
	errBoom := errors.New("boom")
	c1 := &closableStub{name: "c1", closeErr: errBoom}
	c2 := &closableStub{name: "c2"}

	err := CloseAll([]Sender{c1, c2})
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected first close error, got %v", err)
	}
	// 即使第一个出错，第二个也应被关闭
	if !c2.closed {
		t.Fatal("expected c2 to be closed even after c1 error")
	}
}
