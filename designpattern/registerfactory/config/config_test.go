package config

import (
	"errors"
	"strings"
	"testing"

	"go-notes/designpattern/registerfactory/sender"
)

type testSender struct{ name string }

func (t *testSender) Send(string) error { return nil }

func setup(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { sender.Reset() })

	sender.Register("mock-ok", func(cfg sender.Config) (sender.Sender, error) {
		return &testSender{name: cfg["id"]}, nil
	})
	sender.Register("mock-fail", func(cfg sender.Config) (sender.Sender, error) {
		return nil, errors.New("invalid config")
	})
}

func TestBuildSenders_Happy(t *testing.T) {
	setup(t)

	app := AppConfig{
		Senders: []SenderEntry{
			{Name: "mock-ok", Config: sender.Config{"id": "s1"}},
		},
	}

	result, err := BuildSenders(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 sender, got %d", len(result))
	}
	if result["mock-ok"] == nil {
		t.Fatal("expected mock-ok sender in result")
	}
}

func TestBuildSenders_DuplicateName(t *testing.T) {
	setup(t)

	app := AppConfig{
		Senders: []SenderEntry{
			{Name: "mock-ok", Config: sender.Config{"id": "s1"}},
			{Name: "mock-ok", Config: sender.Config{"id": "s2"}},
		},
	}

	_, err := BuildSenders(app)
	if err == nil {
		t.Fatal("expected error for duplicate sender name")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("error should mention duplicate, got: %v", err)
	}
}

func TestBuildSenders_Unknown(t *testing.T) {
	setup(t)

	app := AppConfig{
		Senders: []SenderEntry{
			{Name: "nonexistent", Config: nil},
		},
	}

	_, err := BuildSenders(app)
	if err == nil {
		t.Fatal("expected error for unknown sender")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Fatalf("error should mention sender name, got: %v", err)
	}
}

func TestBuildSenders_InvalidConfig(t *testing.T) {
	setup(t)

	app := AppConfig{
		Senders: []SenderEntry{
			{Name: "mock-fail", Config: nil},
		},
	}

	_, err := BuildSenders(app)
	if err == nil {
		t.Fatal("expected error from factory")
	}
	if !strings.Contains(err.Error(), "invalid config") {
		t.Fatalf("error should propagate factory error, got: %v", err)
	}
}

func TestParseAndBuild_Happy(t *testing.T) {
	setup(t)

	jsonData := `{"senders": [{"name": "mock-ok", "config": {"id": "from-json"}}]}`

	result, err := ParseAndBuild([]byte(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["mock-ok"] == nil {
		t.Fatal("expected mock-ok sender from JSON config")
	}
}

func TestParseAndBuild_InvalidJSON(t *testing.T) {
	setup(t)

	_, err := ParseAndBuild([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("error should mention invalid JSON, got: %v", err)
	}
}
