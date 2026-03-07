package sender

import (
	"sync"
	"testing"
)

type stubSender struct{ name string }

func (s *stubSender) Send(string) error { return nil }

func newStubFactory(name string) Factory {
	return func() Sender { return &stubSender{name: name} }
}

func TestRegister_And_New(t *testing.T) {
	t.Cleanup(func() { Reset() })

	Register("email", newStubFactory("email"))

	s, err := New("email")
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

	s1, _ := New("sms")
	s2, _ := New("sms")
	if s1 == s2 {
		t.Fatal("expected different instances, got same pointer")
	}
}

func TestNew_UnknownName(t *testing.T) {
	t.Cleanup(func() { Reset() })

	_, err := New("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown sender, got nil")
	}
}

func TestMustNew_Success(t *testing.T) {
	t.Cleanup(func() { Reset() })

	Register("sms", newStubFactory("sms"))
	s := MustNew("sms")
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
	MustNew("nonexistent")
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
			if s, err := New(name); err != nil || s == nil {
				t.Errorf("concurrent New(%q) failed: %v", name, err)
			}
		}(name)
	}
	wg.Wait()
}
