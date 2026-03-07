package configmgr

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestInstance_DefaultConfig(t *testing.T) {
	ResetForTesting()
	t.Setenv("CONFIG_PATH", "")

	cfg, err := Instance()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppName != "my-app" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "my-app")
	}
	if cfg.ServerPort != 8080 {
		t.Errorf("ServerPort = %d, want %d", cfg.ServerPort, 8080)
	}
	if cfg.Debug {
		t.Error("Debug should be false by default")
	}
	if cfg.DBHost != "localhost" {
		t.Errorf("DBHost = %q, want %q", cfg.DBHost, "localhost")
	}
	if cfg.DBPort != 3306 {
		t.Errorf("DBPort = %d, want %d", cfg.DBPort, 3306)
	}
}

func TestInstance_FromFile(t *testing.T) {
	ResetForTesting()

	content := `{
		"app_name": "test-app",
		"debug": true,
		"server_port": 9090,
		"db_host": "db.example.com",
		"db_port": 5432
	}`
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_PATH", path)

	cfg, err := Instance()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppName != "test-app" {
		t.Errorf("AppName = %q, want %q", cfg.AppName, "test-app")
	}
	if !cfg.Debug {
		t.Error("Debug should be true")
	}
	if cfg.ServerPort != 9090 {
		t.Errorf("ServerPort = %d, want %d", cfg.ServerPort, 9090)
	}
	if cfg.DBHost != "db.example.com" {
		t.Errorf("DBHost = %q, want %q", cfg.DBHost, "db.example.com")
	}
	if cfg.DBPort != 5432 {
		t.Errorf("DBPort = %d, want %d", cfg.DBPort, 5432)
	}
}

func TestInstance_InvalidJSON(t *testing.T) {
	ResetForTesting()

	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte(`{invalid`), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_PATH", path)

	cfg, err := Instance()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if cfg != nil {
		t.Fatal("expected nil config on error")
	}
}

func TestInstance_FileNotFound(t *testing.T) {
	ResetForTesting()
	t.Setenv("CONFIG_PATH", "/nonexistent/path/config.json")

	cfg, err := Instance()
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if cfg != nil {
		t.Fatal("expected nil config on error")
	}
}

func TestInstance_ReturnsSamePointer(t *testing.T) {
	ResetForTesting()
	t.Setenv("CONFIG_PATH", "")

	cfg1, _ := Instance()
	cfg2, _ := Instance()

	if cfg1 != cfg2 {
		t.Fatal("Instance() should return the same pointer on every call")
	}
}

func TestInstance_ConcurrentAccess(t *testing.T) {
	ResetForTesting()
	t.Setenv("CONFIG_PATH", "")

	const n = 100
	configs := make([]*Config, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			configs[idx], errs[idx] = Instance()
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("goroutine %d: unexpected error: %v", i, errs[i])
		}
		if configs[i] != configs[0] {
			t.Fatalf("goroutine %d got a different instance", i)
		}
	}
}

func TestInstance_ErrorIsCached(t *testing.T) {
	ResetForTesting()
	t.Setenv("CONFIG_PATH", "/nonexistent/path/config.json")

	_, err1 := Instance()
	_, err2 := Instance()

	if err1 == nil || err2 == nil {
		t.Fatal("both calls should return error")
	}
	if err1.Error() != err2.Error() {
		t.Fatalf("error should be cached across calls: %q vs %q", err1, err2)
	}
}

func TestResetForTesting(t *testing.T) {
	ResetForTesting()
	t.Setenv("CONFIG_PATH", "")

	cfg1, _ := Instance()
	if cfg1.AppName != "my-app" {
		t.Fatalf("AppName = %q, want %q", cfg1.AppName, "my-app")
	}

	ResetForTesting()

	content := `{"app_name": "reloaded"}`
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_PATH", path)

	cfg2, err := Instance()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg2.AppName != "reloaded" {
		t.Errorf("AppName after reset = %q, want %q", cfg2.AppName, "reloaded")
	}
	if cfg1 == cfg2 {
		t.Fatal("reset should create a new instance")
	}
}
