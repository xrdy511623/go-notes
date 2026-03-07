package container

import (
	"testing"
)

func TestCreateUser_Success(t *testing.T) {
	a := InitializeTestApp()

	u, err := a.UserService.CreateUser("alice", "alice@example.com")
	if err != nil {
		t.Fatalf("CreateUser unexpected error: %v", err)
	}
	if u.Name != "alice" {
		t.Errorf("Name = %q, want %q", u.Name, "alice")
	}
	if u.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", u.Email, "alice@example.com")
	}
	if u.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestCreateUser_EmptyName(t *testing.T) {
	a := InitializeTestApp()

	_, err := a.UserService.CreateUser("", "alice@example.com")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCreateUser_EmptyEmail(t *testing.T) {
	a := InitializeTestApp()

	_, err := a.UserService.CreateUser("alice", "")
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestGetUser_Found(t *testing.T) {
	a := InitializeTestApp()

	created, err := a.UserService.CreateUser("bob", "bob@example.com")
	if err != nil {
		t.Fatalf("CreateUser unexpected error: %v", err)
	}

	found, err := a.UserService.GetUser(created.ID)
	if err != nil {
		t.Fatalf("GetUser unexpected error: %v", err)
	}
	if found.Name != "bob" {
		t.Errorf("Name = %q, want %q", found.Name, "bob")
	}
	if found.Email != "bob@example.com" {
		t.Errorf("Email = %q, want %q", found.Email, "bob@example.com")
	}
}

func TestGetUser_NotFound(t *testing.T) {
	a := InitializeTestApp()

	_, err := a.UserService.GetUser("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestGetUser_EmptyID(t *testing.T) {
	a := InitializeTestApp()

	_, err := a.UserService.GetUser("")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestUpdateEmail_Success(t *testing.T) {
	a := InitializeTestApp()

	created, err := a.UserService.CreateUser("carol", "carol@old.com")
	if err != nil {
		t.Fatalf("CreateUser unexpected error: %v", err)
	}

	if err := a.UserService.UpdateEmail(created.ID, "carol@new.com"); err != nil {
		t.Fatalf("UpdateEmail unexpected error: %v", err)
	}

	updated, err := a.UserService.GetUser(created.ID)
	if err != nil {
		t.Fatalf("GetUser unexpected error: %v", err)
	}
	if updated.Email != "carol@new.com" {
		t.Errorf("Email = %q, want %q", updated.Email, "carol@new.com")
	}
}

func TestUpdateEmail_UserNotFound(t *testing.T) {
	a := InitializeTestApp()

	err := a.UserService.UpdateEmail("nonexistent-id", "new@example.com")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestUpdateEmail_EmptyEmail(t *testing.T) {
	a := InitializeTestApp()

	created, err := a.UserService.CreateUser("dave", "dave@example.com")
	if err != nil {
		t.Fatalf("CreateUser unexpected error: %v", err)
	}

	err = a.UserService.UpdateEmail(created.ID, "")
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestInitializeApp_Production(t *testing.T) {
	cfg := Config{MySQLDSN: "user:pass@tcp(localhost:3306)/testdb"}
	a := InitializeApp(cfg)

	if a == nil {
		t.Fatal("App should not be nil")
	}
	if a.UserService == nil {
		t.Fatal("UserService should not be nil")
	}
}

func TestIsolation_MultipleApps(t *testing.T) {
	app1 := InitializeTestApp()
	app2 := InitializeTestApp()

	// 在 app1 中创建用户
	created, err := app1.UserService.CreateUser("eve", "eve@example.com")
	if err != nil {
		t.Fatalf("app1 CreateUser unexpected error: %v", err)
	}

	// app2 不应能查到 app1 的用户——两个 App 使用独立的 InMemory 存储
	_, err = app2.UserService.GetUser(created.ID)
	if err == nil {
		t.Fatal("app2 should NOT find user created in app1 — state must be isolated")
	}
}
