package transaction

import "testing"

func TestTransaction_CommitSuccess(t *testing.T) {
	s := NewStore()

	tx := Begin(
		Set(s, "name", "Alice"),
		Set(s, "age", "30"),
	)
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	name, _ := s.Get("name")
	age, _ := s.Get("age")
	if name != "Alice" || age != "30" {
		t.Fatalf("got name=%q age=%q", name, age)
	}
}

func TestTransaction_CommitFails_RollsBack(t *testing.T) {
	s := NewStore()
	s.Put("name", "Alice")

	tx := Begin(
		Set(s, "name", "Bob"),
		Del(s, "nonexistent"),
	)
	if err := tx.Commit(); err == nil {
		t.Fatal("expected commit to fail")
	}

	name, _ := s.Get("name")
	if name != "Alice" {
		t.Fatalf("after rollback: name=%q, want %q", name, "Alice")
	}
}

func TestTransaction_Rollback(t *testing.T) {
	s := NewStore()
	s.Put("x", "1")

	tx := Begin(Set(s, "x", "2"), Set(s, "y", "3"))
	tx.Commit()

	x, _ := s.Get("x")
	y, _ := s.Get("y")
	if x != "2" || y != "3" {
		t.Fatalf("after commit: x=%q y=%q", x, y)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback: %v", err)
	}

	x, _ = s.Get("x")
	_, hasY := s.Get("y")
	if x != "1" {
		t.Fatalf("after rollback: x=%q, want %q", x, "1")
	}
	if hasY {
		t.Fatal("after rollback: y should not exist")
	}
}

func TestTransaction_DoubleCommit(t *testing.T) {
	s := NewStore()
	tx := Begin(Set(s, "a", "1"))
	tx.Commit()
	if err := tx.Commit(); err == nil {
		t.Fatal("expected error on double commit")
	}
}

func TestTransaction_EmptyCommit(t *testing.T) {
	tx := Begin()
	if err := tx.Commit(); err != nil {
		t.Fatalf("empty commit should succeed: %v", err)
	}
}

func TestTransaction_Add(t *testing.T) {
	s := NewStore()
	tx := Begin()
	tx.Add(Set(s, "a", "1")).Add(Set(s, "b", "2"))

	if tx.Len() != 2 {
		t.Fatalf("Len = %d, want 2", tx.Len())
	}

	tx.Commit()
	a, _ := s.Get("a")
	b, _ := s.Get("b")
	if a != "1" || b != "2" {
		t.Fatalf("a=%q b=%q", a, b)
	}
}

func TestTransaction_PartialFailure_Rollback(t *testing.T) {
	s := NewStore()
	s.Put("balance", "100")

	tx := Begin(
		Set(s, "balance", "50"),
		Set(s, "status", "processing"),
		Del(s, "missing_key"),
	)
	err := tx.Commit()
	if err == nil {
		t.Fatal("expected commit failure")
	}

	balance, _ := s.Get("balance")
	if balance != "100" {
		t.Fatalf("balance should be restored to 100, got %q", balance)
	}
	_, hasStatus := s.Get("status")
	if hasStatus {
		t.Fatal("status should not exist after rollback")
	}
}

func TestAction_Set_Overwrite(t *testing.T) {
	s := NewStore()
	s.Put("k", "old")

	tx := Begin(Set(s, "k", "new"))
	tx.Commit()

	v, _ := s.Get("k")
	if v != "new" {
		t.Fatalf("after set: %q", v)
	}

	tx.Rollback()
	v, _ = s.Get("k")
	if v != "old" {
		t.Fatalf("after rollback: %q, want %q", v, "old")
	}
}

func TestAction_Del(t *testing.T) {
	s := NewStore()
	s.Put("k", "val")

	tx := Begin(Del(s, "k"))
	tx.Commit()

	_, ok := s.Get("k")
	if ok {
		t.Fatal("key should be deleted")
	}

	tx.Rollback()
	v, ok := s.Get("k")
	if !ok || v != "val" {
		t.Fatalf("after rollback: ok=%v v=%q", ok, v)
	}
}

func TestStore_Keys(t *testing.T) {
	s := NewStore()
	s.Put("a", "1")
	s.Put("b", "2")

	keys := s.Keys()
	if len(keys) != 2 {
		t.Fatalf("Keys len = %d, want 2", len(keys))
	}
}

func TestAction_String(t *testing.T) {
	s := NewStore()
	setA := Set(s, "key", "val")
	if setA.String() != `Set("key", "val")` {
		t.Errorf("Set.String() = %q", setA.String())
	}
	delA := Del(s, "key")
	if delA.String() != `Del("key")` {
		t.Errorf("Del.String() = %q", delA.String())
	}
}
