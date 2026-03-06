package builder

import (
	"testing"
	"time"
)

func TestBuilder_Defaults(t *testing.T) {
	srv, err := NewBuilder("localhost", 8080).Build()
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

func TestBuilder_FluentChain(t *testing.T) {
	srv, err := NewBuilder("0.0.0.0", 443).
		ReadTimeout(10*time.Second).
		WriteTimeout(15*time.Second).
		MaxConnections(5000).
		TLS("cert.pem", "key.pem").
		Build()

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

func TestBuilder_EmptyHost(t *testing.T) {
	_, err := NewBuilder("", 8080).Build()
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestBuilder_InvalidPort(t *testing.T) {
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
			_, err := NewBuilder("localhost", tt.port).Build()
			if err == nil {
				t.Fatal("expected error for invalid port")
			}
		})
	}
}

func TestBuilder_NegativeReadTimeout(t *testing.T) {
	_, err := NewBuilder("localhost", 8080).
		ReadTimeout(-1 * time.Second).
		Build()
	if err == nil {
		t.Fatal("expected error for negative read timeout")
	}
}

func TestBuilder_NegativeWriteTimeout(t *testing.T) {
	_, err := NewBuilder("localhost", 8080).
		WriteTimeout(-1 * time.Second).
		Build()
	if err == nil {
		t.Fatal("expected error for negative write timeout")
	}
}

func TestBuilder_NegativeMaxConnections(t *testing.T) {
	_, err := NewBuilder("localhost", 8080).
		MaxConnections(-1).
		Build()
	if err == nil {
		t.Fatal("expected error for negative max connections")
	}
}

func TestBuilder_TLS_EmptyCert(t *testing.T) {
	_, err := NewBuilder("localhost", 8080).
		TLS("", "key.pem").
		Build()
	if err == nil {
		t.Fatal("expected error for empty cert file")
	}
}

func TestBuilder_TLS_EmptyKey(t *testing.T) {
	_, err := NewBuilder("localhost", 8080).
		TLS("cert.pem", "").
		Build()
	if err == nil {
		t.Fatal("expected error for empty key file")
	}
}

func TestBuilder_DeferredError_StopsChain(t *testing.T) {
	_, err := NewBuilder("localhost", 8080).
		ReadTimeout(-1 * time.Second).
		MaxConnections(5000). // should be skipped
		Build()

	if err == nil {
		t.Fatal("expected error")
	}
	// Error should be about ReadTimeout, not MaxConnections
	if got := err.Error(); got != "builder: read timeout must be positive, got -1s" {
		t.Fatalf("unexpected error: %v", got)
	}
}

func TestBuilder_ReturnsCopy(t *testing.T) {
	b := NewBuilder("localhost", 8080)
	s1, _ := b.Build()
	s2, _ := b.Build()
	if s1 == s2 {
		t.Fatal("expected different pointers from Build()")
	}
}
