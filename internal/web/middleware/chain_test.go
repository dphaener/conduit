package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewChain(t *testing.T) {
	chain := NewChain()
	if chain == nil {
		t.Fatal("NewChain returned nil")
	}
	if len(chain.middlewares) != 0 {
		t.Errorf("Expected empty chain, got %d middlewares", len(chain.middlewares))
	}
}

func TestNewChainWithMiddlewares(t *testing.T) {
	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	chain := NewChain(m1, m2)
	if len(chain.middlewares) != 2 {
		t.Errorf("Expected 2 middlewares, got %d", len(chain.middlewares))
	}
}

func TestChainUse(t *testing.T) {
	chain := NewChain()
	m := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	result := chain.Use(m)
	if result != chain {
		t.Error("Use should return the same chain for chaining")
	}
	if len(chain.middlewares) != 1 {
		t.Errorf("Expected 1 middleware, got %d", len(chain.middlewares))
	}
}

func TestChainApply(t *testing.T) {
	var called []string

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = append(called, "m1-before")
			next.ServeHTTP(w, r)
			called = append(called, "m1-after")
		})
	}

	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = append(called, "m2-before")
			next.ServeHTTP(w, r)
			called = append(called, "m2-after")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = append(called, "handler")
	})

	chain := NewChain(m1, m2)
	wrapped := chain.Apply(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	if len(called) != len(expected) {
		t.Fatalf("Expected %d calls, got %d: %v", len(expected), len(called), called)
	}

	for i, exp := range expected {
		if called[i] != exp {
			t.Errorf("Call %d: expected %s, got %s", i, exp, called[i])
		}
	}
}

func TestChainThen(t *testing.T) {
	var executed bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executed = true
	})

	chain := NewChain()
	wrapped := chain.Then(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if !executed {
		t.Error("Handler was not executed")
	}
}

func TestChainThenFunc(t *testing.T) {
	var executed bool
	handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executed = true
	})

	chain := NewChain()
	wrapped := chain.ThenFunc(handlerFunc)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if !executed {
		t.Error("Handler was not executed")
	}
}

func TestChainAppend(t *testing.T) {
	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
	m3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	chain1 := NewChain(m1, m2)
	chain2 := chain1.Append(m3)

	if len(chain1.middlewares) != 2 {
		t.Errorf("Original chain should have 2 middlewares, got %d", len(chain1.middlewares))
	}
	if len(chain2.middlewares) != 3 {
		t.Errorf("New chain should have 3 middlewares, got %d", len(chain2.middlewares))
	}
}

func TestChainExtend(t *testing.T) {
	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	chain := NewChain(m1)
	result := chain.Extend(m2)

	if result != chain {
		t.Error("Extend should return the same chain")
	}
	if len(chain.middlewares) != 2 {
		t.Errorf("Expected 2 middlewares, got %d", len(chain.middlewares))
	}
}

func TestChainOrdering(t *testing.T) {
	var order []int

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 1)
			next.ServeHTTP(w, r)
		})
	}

	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 2)
			next.ServeHTTP(w, r)
		})
	}

	m3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, 3)
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, 4)
	})

	chain := NewChain().Use(m1).Use(m2).Use(m3)
	wrapped := chain.Apply(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	expected := []int{1, 2, 3, 4}
	if len(order) != len(expected) {
		t.Fatalf("Expected %d items in order, got %d: %v", len(expected), len(order), order)
	}

	for i, exp := range expected {
		if order[i] != exp {
			t.Errorf("Position %d: expected %d, got %d", i, exp, order[i])
		}
	}
}
