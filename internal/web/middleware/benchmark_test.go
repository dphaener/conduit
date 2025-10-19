package middleware

import (
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func BenchmarkChainApply(b *testing.B) {
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

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	chain := NewChain(m1, m2, m3)
	wrapped := chain.Apply(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapped.ServeHTTP(rec, req)
	}
}

func BenchmarkRecoveryMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := Recovery()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapped.ServeHTTP(rec, req)
	}
}

func BenchmarkRequestIDMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestID()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapped.ServeHTTP(rec, req)
	}
}

func BenchmarkCORSMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := CORS()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapped.ServeHTTP(rec, req)
	}
}

func BenchmarkLoggingMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	config := LoggingConfig{
		Logger: func(entry LogEntry) {
			// Discard logs for benchmark
		},
	}

	middleware := LoggingWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapped.ServeHTTP(rec, req)
	}
}

func BenchmarkCompressionMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("test data ", 200)))
	})

	config := CompressionConfig{
		Level:   gzip.DefaultCompression,
		MinSize: 100,
	}

	middleware := CompressionWithConfig(config)
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}
}

func BenchmarkCompressionNoGzip(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("test data ", 200)))
	})

	middleware := Compression()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No Accept-Encoding header

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}
}

func BenchmarkConditionalMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	testMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	predicate := PathPrefix("/api")
	conditional := Conditional(predicate, testMiddleware)
	wrapped := conditional(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapped.ServeHTTP(rec, req)
	}
}

func BenchmarkFullMiddlewareStack(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"success"}`))
	})

	// Simulate a typical middleware stack
	chain := NewChain(
		Recovery(),
		RequestID(),
		Logging(),
		CORS(),
		Compression(),
	)
	wrapped := chain.Apply(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Accept-Encoding", "gzip")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}
}

func BenchmarkChainAppend(b *testing.B) {
	m := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	chain := NewChain(m)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chain.Append(m)
	}
}

func BenchmarkPredicateAnd(b *testing.B) {
	p1 := PathPrefix("/api")
	p2 := Method(http.MethodGet)
	predicate := And(p1, p2)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = predicate(req)
	}
}

func BenchmarkPredicateOr(b *testing.B) {
	p1 := PathPrefix("/api")
	p2 := PathPrefix("/public")
	predicate := Or(p1, p2)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = predicate(req)
	}
}

func BenchmarkGetRequestID(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = GetRequestID(r.Context())
	})

	middleware := RequestID()
	wrapped := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrapped.ServeHTTP(rec, req)
	}
}
