package wiredemo

import "testing"

func TestInitializeApp(t *testing.T) {
	cfg := &Config{
		DBHost:  "localhost",
		DBPort:  5432,
		AppPort: 8080,
	}

	app, err := InitializeApp(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if app.Port != 8080 {
		t.Errorf("app.Port = %d, want 8080", app.Port)
	}

	got := app.UserSvc.GetUser("123")
	want := "user-123 from localhost:5432"
	if got != want {
		t.Errorf("GetUser = %q, want %q", got, want)
	}

	t.Log("wire自动生成了依赖组装代码，查看 wire_gen.go")
}

func TestInitializeApp_InvalidConfig(t *testing.T) {
	cfg := &Config{DBHost: ""} // 缺少必要配置

	_, err := InitializeApp(cfg)
	if err == nil {
		t.Error("expected error for empty DBHost")
	}
	t.Logf("wire生成的代码正确传播了Provider的错误: %v", err)
}
