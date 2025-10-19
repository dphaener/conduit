package middleware

import (
	"net/http"
	"strings"
)

// Predicate is a function that determines if middleware should be applied
type Predicate func(*http.Request) bool

// Conditional wraps middleware to only apply it when a predicate is true
func Conditional(predicate Predicate, middleware Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if predicate(r) {
				// Apply middleware
				middleware(next).ServeHTTP(w, r)
			} else {
				// Skip middleware
				next.ServeHTTP(w, r)
			}
		})
	}
}

// Common predicates for convenience

// PathPrefix creates a predicate that matches requests with a path prefix
func PathPrefix(prefix string) Predicate {
	return func(r *http.Request) bool {
		return strings.HasPrefix(r.URL.Path, prefix)
	}
}

// PathSuffix creates a predicate that matches requests with a path suffix
func PathSuffix(suffix string) Predicate {
	return func(r *http.Request) bool {
		return strings.HasSuffix(r.URL.Path, suffix)
	}
}

// PathEquals creates a predicate that matches requests with an exact path
func PathEquals(path string) Predicate {
	return func(r *http.Request) bool {
		return r.URL.Path == path
	}
}

// Method creates a predicate that matches requests with a specific HTTP method
func Method(method string) Predicate {
	return func(r *http.Request) bool {
		return r.Method == method
	}
}

// Header creates a predicate that matches requests with a specific header
func Header(key, value string) Predicate {
	return func(r *http.Request) bool {
		return r.Header.Get(key) == value
	}
}

// HasHeader creates a predicate that matches requests with a specific header key
func HasHeader(key string) Predicate {
	return func(r *http.Request) bool {
		return r.Header.Get(key) != ""
	}
}

// And combines multiple predicates with logical AND
func And(predicates ...Predicate) Predicate {
	return func(r *http.Request) bool {
		for _, p := range predicates {
			if !p(r) {
				return false
			}
		}
		return true
	}
}

// Or combines multiple predicates with logical OR
func Or(predicates ...Predicate) Predicate {
	return func(r *http.Request) bool {
		for _, p := range predicates {
			if p(r) {
				return true
			}
		}
		return false
	}
}

// Not negates a predicate
func Not(predicate Predicate) Predicate {
	return func(r *http.Request) bool {
		return !predicate(r)
	}
}
