package logadapter

import (
	"bytes"
	"log"
	"log/slog"
	"strings"
	"testing"
)

func TestStdAdapter_Info(t *testing.T) {
	var buf bytes.Buffer
	l := FromStd(log.New(&buf, "", 0))
	l.Info("hello %s", "world")

	got := buf.String()
	if !strings.Contains(got, "[INFO]") {
		t.Errorf("expected [INFO] prefix, got %q", got)
	}
	if !strings.Contains(got, "hello world") {
		t.Errorf("expected formatted message, got %q", got)
	}
}

func TestStdAdapter_Debug(t *testing.T) {
	var buf bytes.Buffer
	l := FromStd(log.New(&buf, "", 0))
	l.Debug("count=%d", 42)

	got := buf.String()
	if !strings.Contains(got, "[DEBUG]") {
		t.Errorf("expected [DEBUG], got %q", got)
	}
	if !strings.Contains(got, "count=42") {
		t.Errorf("expected formatted args, got %q", got)
	}
}

func TestStdAdapter_Error(t *testing.T) {
	var buf bytes.Buffer
	l := FromStd(log.New(&buf, "", 0))
	l.Error("failed: %v", "timeout")

	got := buf.String()
	if !strings.Contains(got, "[ERROR]") {
		t.Errorf("expected [ERROR], got %q", got)
	}
	if !strings.Contains(got, "failed: timeout") {
		t.Errorf("expected formatted message, got %q", got)
	}
}

func TestSlogAdapter_Info(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := FromSlog(slog.New(handler))
	l.Info("hello %s", "world")

	got := buf.String()
	if !strings.Contains(got, "hello world") {
		t.Errorf("expected formatted message, got %q", got)
	}
	if !strings.Contains(got, "INFO") {
		t.Errorf("expected INFO level, got %q", got)
	}
}

func TestSlogAdapter_Debug(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := FromSlog(slog.New(handler))
	l.Debug("value=%d", 99)

	got := buf.String()
	if !strings.Contains(got, "value=99") {
		t.Errorf("expected formatted message, got %q", got)
	}
	if !strings.Contains(got, "DEBUG") {
		t.Errorf("expected DEBUG level, got %q", got)
	}
}

func TestNop(t *testing.T) {
	l := Nop()
	l.Debug("should not panic")
	l.Info("should not panic")
	l.Error("should not panic")
}

func TestAdapters_SatisfyInterface(t *testing.T) {
	var buf bytes.Buffer
	adapters := []Logger{
		FromStd(log.New(&buf, "", 0)),
		FromSlog(slog.New(slog.NewTextHandler(&buf, nil))),
		Nop(),
	}
	for _, a := range adapters {
		a.Info("interface check")
	}
}

func TestStdAdapter_WithPrefix(t *testing.T) {
	var buf bytes.Buffer
	l := FromStd(log.New(&buf, "[myapp] ", 0))
	l.Info("started")

	got := buf.String()
	if !strings.Contains(got, "[myapp]") {
		t.Errorf("expected logger prefix preserved, got %q", got)
	}
	if !strings.Contains(got, "[INFO] started") {
		t.Errorf("expected level + message, got %q", got)
	}
}

func TestSlogAdapter_Error(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	l := FromSlog(slog.New(handler))
	l.Error("disk full: %s", "/var/log")

	got := buf.String()
	if !strings.Contains(got, "disk full: /var/log") {
		t.Errorf("expected formatted message, got %q", got)
	}
	if !strings.Contains(got, "ERROR") {
		t.Errorf("expected ERROR level, got %q", got)
	}
}
