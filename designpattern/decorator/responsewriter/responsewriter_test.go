package responsewriter

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseRecorder_DefaultStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rr := NewResponseRecorder(rec)

	rr.Write([]byte("hello"))

	if rr.StatusCode() != http.StatusOK {
		t.Errorf("StatusCode() = %d, want %d", rr.StatusCode(), http.StatusOK)
	}
}

func TestResponseRecorder_ExplicitStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rr := NewResponseRecorder(rec)

	rr.WriteHeader(http.StatusNotFound)
	rr.Write([]byte("not found"))

	if rr.StatusCode() != http.StatusNotFound {
		t.Errorf("StatusCode() = %d, want %d", rr.StatusCode(), http.StatusNotFound)
	}
}

func TestResponseRecorder_DoubleWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rr := NewResponseRecorder(rec)

	rr.WriteHeader(http.StatusCreated)
	rr.WriteHeader(http.StatusNotFound)

	if rr.StatusCode() != http.StatusCreated {
		t.Errorf("StatusCode() = %d, want %d (first call wins)", rr.StatusCode(), http.StatusCreated)
	}
}

func TestResponseRecorder_BytesWritten(t *testing.T) {
	rec := httptest.NewRecorder()
	rr := NewResponseRecorder(rec)

	rr.Write([]byte("hello"))
	rr.Write([]byte(" world"))

	if rr.BytesWritten() != 11 {
		t.Errorf("BytesWritten() = %d, want 11", rr.BytesWritten())
	}
}

func TestResponseRecorder_ZeroBytes(t *testing.T) {
	rec := httptest.NewRecorder()
	rr := NewResponseRecorder(rec)

	rr.WriteHeader(http.StatusNoContent)

	if rr.BytesWritten() != 0 {
		t.Errorf("BytesWritten() = %d, want 0", rr.BytesWritten())
	}
}

func TestResponseRecorder_PassesThrough(t *testing.T) {
	rec := httptest.NewRecorder()
	rr := NewResponseRecorder(rec)

	rr.Header().Set("X-Custom", "test")
	rr.WriteHeader(http.StatusAccepted)
	rr.Write([]byte("body"))

	if rec.Code != http.StatusAccepted {
		t.Errorf("underlying status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if rec.Body.String() != "body" {
		t.Errorf("underlying body = %q, want %q", rec.Body.String(), "body")
	}
	if rec.Header().Get("X-Custom") != "test" {
		t.Errorf("underlying header X-Custom = %q, want %q", rec.Header().Get("X-Custom"), "test")
	}
}

func TestResponseRecorder_Unwrap(t *testing.T) {
	rec := httptest.NewRecorder()
	rr := NewResponseRecorder(rec)

	if rr.Unwrap() != rec {
		t.Fatal("Unwrap() should return the underlying ResponseWriter")
	}
}

func TestResponseRecorder_InMiddleware(t *testing.T) {
	var loggedStatus int
	var loggedBytes int64

	loggingMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			rr := NewResponseRecorder(w)
			next(rr, r)
			loggedStatus = rr.StatusCode()
			loggedBytes = rr.BytesWritten()
		}
	}

	handler := loggingMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":"123"}`)
	})

	req := httptest.NewRequest("POST", "/users", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if loggedStatus != http.StatusCreated {
		t.Errorf("logged status = %d, want %d", loggedStatus, http.StatusCreated)
	}
	if loggedBytes != 12 {
		t.Errorf("logged bytes = %d, want 12", loggedBytes)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("response status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestResponseRecorder_StackedDecorators(t *testing.T) {
	rec := httptest.NewRecorder()
	inner := NewResponseRecorder(rec)
	outer := NewResponseRecorder(inner)

	outer.WriteHeader(http.StatusOK)
	outer.Write([]byte("stacked"))

	if inner.BytesWritten() != outer.BytesWritten() {
		t.Errorf("inner=%d, outer=%d — both should see same bytes",
			inner.BytesWritten(), outer.BytesWritten())
	}
	if rec.Body.String() != "stacked" {
		t.Errorf("underlying body = %q, want %q", rec.Body.String(), "stacked")
	}
}

func TestResponseRecorder_WriteImpliesOK(t *testing.T) {
	rec := httptest.NewRecorder()
	rr := NewResponseRecorder(rec)

	rr.Write([]byte("implicit 200"))

	if !rr.wroteHeader {
		t.Error("Write should implicitly call WriteHeader")
	}
	if rr.StatusCode() != http.StatusOK {
		t.Errorf("StatusCode() = %d, want %d", rr.StatusCode(), http.StatusOK)
	}
}

func BenchmarkResponseRecorder(b *testing.B) {
	body := []byte("benchmark response body content")

	b.Run("plain_writer", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			rec := httptest.NewRecorder()
			rec.WriteHeader(http.StatusOK)
			rec.Write(body)
		}
	})

	b.Run("recorder_wrapped", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			rec := httptest.NewRecorder()
			rr := NewResponseRecorder(rec)
			rr.WriteHeader(http.StatusOK)
			rr.Write(body)
		}
	})
}
