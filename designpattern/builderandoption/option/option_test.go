package option

import (
	"testing"
	"time"
)

func TestNewServer_Defaults(t *testing.T) {
	srv, err := NewServer("localhost", 8080)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv.Host != "localhost" || srv.Port != 8080 {
		t.Fatalf("expected localhost:8080, got %s:%d", srv.Host, srv.Port)
	}
	if srv.ReadTimeout != 30*time.Second {
		t.Fatalf("expected 30s read timeout, got %v", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 30*time.Second {
		t.Fatalf("expected 30s write timeout, got %v", srv.WriteTimeout)
	}
	if srv.MaxConnections != 1000 {
		t.Fatalf("expected 1000 max connections, got %d", srv.MaxConnections)
	}
	if srv.TLSEnabled {
		t.Fatal("expected TLS disabled by default")
	}
}

func TestNewServer_AllOptions(t *testing.T) {
	srv, err := NewServer("0.0.0.0", 443,
		WithReadTimeout(10*time.Second),
		WithWriteTimeout(15*time.Second),
		WithMaxConnections(5000),
		WithTLS("cert.pem", "key.pem"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv.ReadTimeout != 10*time.Second {
		t.Errorf("ReadTimeout = %v, want 10s", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 15*time.Second {
		t.Errorf("WriteTimeout = %v, want 15s", srv.WriteTimeout)
	}
	if srv.MaxConnections != 5000 {
		t.Errorf("MaxConnections = %d, want 5000", srv.MaxConnections)
	}
	if !srv.TLSEnabled || srv.CertFile != "cert.pem" || srv.KeyFile != "key.pem" {
		t.Errorf("TLS not configured correctly")
	}
}

func TestNewServer_EmptyHost(t *testing.T) {
	_, err := NewServer("", 8080)
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestNewServer_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero", 0},
		{"negative", -1},
		{"too_large", 70000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewServer("localhost", tt.port)
			if err == nil {
				t.Fatal("expected error for invalid port")
			}
		})
	}
}

func TestNewServer_NegativeReadTimeout(t *testing.T) {
	_, err := NewServer("localhost", 8080, WithReadTimeout(-1*time.Second))
	if err == nil {
		t.Fatal("expected error for negative read timeout")
	}
}

func TestNewServer_NegativeWriteTimeout(t *testing.T) {
	_, err := NewServer("localhost", 8080, WithWriteTimeout(-1*time.Second))
	if err == nil {
		t.Fatal("expected error for negative write timeout")
	}
}

func TestNewServer_NegativeMaxConnections(t *testing.T) {
	_, err := NewServer("localhost", 8080, WithMaxConnections(-1))
	if err == nil {
		t.Fatal("expected error for negative max connections")
	}
}

func TestNewServer_TLS_EmptyCert(t *testing.T) {
	_, err := NewServer("localhost", 8080, WithTLS("", "key.pem"))
	if err == nil {
		t.Fatal("expected error for empty cert file")
	}
}

func TestNewServer_TLS_EmptyKey(t *testing.T) {
	_, err := NewServer("localhost", 8080, WithTLS("cert.pem", ""))
	if err == nil {
		t.Fatal("expected error for empty key file")
	}
}

func TestNewServer_OptionErrorStopsApplication(t *testing.T) {
	_, err := NewServer("localhost", 8080,
		WithReadTimeout(-1*time.Second),
		WithMaxConnections(5000),
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "option: read timeout must be positive, got -1s" {
		t.Fatalf("unexpected error: %v", got)
	}
}

func TestNewServer_NoOptions(t *testing.T) {
	srv, err := NewServer("localhost", 8080)
	if err != nil {
		t.Fatal(err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestNewServer_IndependentInstances(t *testing.T) {
	s1, _ := NewServer("localhost", 8080)
	s2, _ := NewServer("localhost", 8080, WithMaxConnections(2000))
	if s1.MaxConnections == s2.MaxConnections {
		t.Fatal("expected different max connections")
	}
}
