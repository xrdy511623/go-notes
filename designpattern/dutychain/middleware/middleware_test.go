package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func traceMW(name string, order *[]string) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			*order = append(*order, name+":before")
			next(w, r)
			*order = append(*order, name+":after")
		}
	}
}

func assertOrder(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d steps %v, got %d steps %v", len(want), want, len(got), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("step[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestChain_ExecutionOrder(t *testing.T) {
	var order []string

	chain := Chain(traceMW("A", &order), traceMW("B", &order), traceMW("C", &order))
	handler := chain(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	req := httptest.NewRequest("GET", "/", nil)
	handler(httptest.NewRecorder(), req)

	assertOrder(t, order, []string{
		"A:before", "B:before", "C:before",
		"handler",
		"C:after", "B:after", "A:after",
	})
}

func TestChain_ShortCircuit(t *testing.T) {
	var reached bool

	blocker := Middleware(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "blocked", http.StatusForbidden)
		}
	})

	handler := Chain(blocker)(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if reached {
		t.Fatal("handler should not be reached when middleware short-circuits")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestChain_Empty(t *testing.T) {
	handler := Chain()(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Body.String() != "ok" {
		t.Fatalf("expected %q, got %q", "ok", rec.Body.String())
	}
}

func TestChain_Single(t *testing.T) {
	var called bool

	mw := Middleware(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			called = true
			next(w, r)
		}
	})

	handler := Chain(mw)(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	handler(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("middleware should be called")
	}
}

func TestApply(t *testing.T) {
	var order []string

	handler := Apply(
		func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "handler")
		},
		traceMW("A", &order), traceMW("B", &order),
	)

	req := httptest.NewRequest("GET", "/", nil)
	handler(httptest.NewRecorder(), req)

	assertOrder(t, order, []string{"A:before", "B:before", "handler", "B:after", "A:after"})
}

func TestChain_Reusable(t *testing.T) {
	counter := 0
	mw := Middleware(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			counter++
			next(w, r)
		}
	})

	chain := Chain(mw)
	h1 := chain(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("h1")) })
	h2 := chain(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("h2")) })

	req := httptest.NewRequest("GET", "/", nil)
	h1(httptest.NewRecorder(), req)
	h2(httptest.NewRecorder(), req)

	if counter != 2 {
		t.Fatalf("expected 2 calls, got %d", counter)
	}
}

func TestChain_Nested(t *testing.T) {
	var order []string

	inner := Chain(traceMW("B", &order), traceMW("C", &order))
	outer := Chain(traceMW("A", &order), inner)
	handler := outer(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	req := httptest.NewRequest("GET", "/", nil)
	handler(httptest.NewRecorder(), req)

	assertOrder(t, order, []string{
		"A:before", "B:before", "C:before",
		"handler",
		"C:after", "B:after", "A:after",
	})
}

func TestChain_ResponseModification(t *testing.T) {
	headerMW := Middleware(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Request-Id", "test-123")
			next(w, r)
		}
	})

	handler := Chain(headerMW)(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if got := rec.Header().Get("X-Request-Id"); got != "test-123" {
		t.Fatalf("expected X-Request-Id=test-123, got %q", got)
	}
}

func TestChain_PanicRecovery(t *testing.T) {
	recoverer := Middleware(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					http.Error(w, "recovered", http.StatusInternalServerError)
				}
			}()
			next(w, r)
		}
	})

	handler := Chain(recoverer)(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestChain_ShortCircuitStopsSubsequentMiddleware(t *testing.T) {
	var secondCalled bool

	blocker := Middleware(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "stop", http.StatusForbidden)
		}
	})
	tracker := Middleware(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			secondCalled = true
			next(w, r)
		}
	})

	handler := Chain(blocker, tracker)(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest("GET", "/", nil)
	handler(httptest.NewRecorder(), req)

	if secondCalled {
		t.Fatal("middleware after short-circuit should not execute")
	}
}
