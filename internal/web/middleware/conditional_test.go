package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConditionalMiddleware(t *testing.T) {
	var middlewareCalled bool
	var handlerCalled bool

	testMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	// Test with true predicate
	middlewareCalled = false
	handlerCalled = false
	predicate := func(r *http.Request) bool { return true }
	conditional := Conditional(predicate, testMiddleware)
	wrapped := conditional(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if !middlewareCalled {
		t.Error("Middleware should be called when predicate is true")
	}
	if !handlerCalled {
		t.Error("Handler should be called")
	}

	// Test with false predicate
	middlewareCalled = false
	handlerCalled = false
	predicate = func(r *http.Request) bool { return false }
	conditional = Conditional(predicate, testMiddleware)
	wrapped = conditional(handler)

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if middlewareCalled {
		t.Error("Middleware should not be called when predicate is false")
	}
	if !handlerCalled {
		t.Error("Handler should still be called")
	}
}

func TestPathPrefix(t *testing.T) {
	tests := []struct {
		path     string
		prefix   string
		expected bool
	}{
		{"/api/users", "/api", true},
		{"/api/posts", "/api", true},
		{"/public/images", "/api", false},
		{"/api", "/api", true},
		{"/v1/api", "/api", false},
	}

	for _, test := range tests {
		predicate := PathPrefix(test.prefix)
		req := httptest.NewRequest(http.MethodGet, test.path, nil)
		result := predicate(req)
		if result != test.expected {
			t.Errorf("PathPrefix(%s) for path %s: expected %v, got %v",
				test.prefix, test.path, test.expected, result)
		}
	}
}

func TestPathSuffix(t *testing.T) {
	tests := []struct {
		path     string
		suffix   string
		expected bool
	}{
		{"/file.json", ".json", true},
		{"/file.xml", ".json", false},
		{"/api/data.csv", ".csv", true},
		{"/test", ".json", false},
	}

	for _, test := range tests {
		predicate := PathSuffix(test.suffix)
		req := httptest.NewRequest(http.MethodGet, test.path, nil)
		result := predicate(req)
		if result != test.expected {
			t.Errorf("PathSuffix(%s) for path %s: expected %v, got %v",
				test.suffix, test.path, test.expected, result)
		}
	}
}

func TestPathEquals(t *testing.T) {
	tests := []struct {
		requestPath string
		matchPath   string
		expected    bool
	}{
		{"/api/users", "/api/users", true},
		{"/api/users", "/api/posts", false},
		{"/api/users/", "/api/users", false},
	}

	for _, test := range tests {
		predicate := PathEquals(test.matchPath)
		req := httptest.NewRequest(http.MethodGet, test.requestPath, nil)
		result := predicate(req)
		if result != test.expected {
			t.Errorf("PathEquals(%s) for path %s: expected %v, got %v",
				test.matchPath, test.requestPath, test.expected, result)
		}
	}
}

func TestMethod(t *testing.T) {
	tests := []struct {
		requestMethod string
		matchMethod   string
		expected      bool
	}{
		{http.MethodGet, http.MethodGet, true},
		{http.MethodPost, http.MethodGet, false},
		{http.MethodPut, http.MethodPut, true},
	}

	for _, test := range tests {
		predicate := Method(test.matchMethod)
		req := httptest.NewRequest(test.requestMethod, "/", nil)
		result := predicate(req)
		if result != test.expected {
			t.Errorf("Method(%s) for request method %s: expected %v, got %v",
				test.matchMethod, test.requestMethod, test.expected, result)
		}
	}
}

func TestHeader(t *testing.T) {
	tests := []struct {
		headerKey   string
		headerValue string
		matchKey    string
		matchValue  string
		expected    bool
	}{
		{"Content-Type", "application/json", "Content-Type", "application/json", true},
		{"Content-Type", "application/json", "Content-Type", "text/html", false},
		{"Authorization", "Bearer token", "Authorization", "Bearer token", true},
		{"X-Custom", "value", "X-Other", "value", false},
	}

	for _, test := range tests {
		predicate := Header(test.matchKey, test.matchValue)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(test.headerKey, test.headerValue)
		result := predicate(req)
		if result != test.expected {
			t.Errorf("Header(%s, %s) with header %s=%s: expected %v, got %v",
				test.matchKey, test.matchValue, test.headerKey, test.headerValue, test.expected, result)
		}
	}
}

func TestHasHeader(t *testing.T) {
	tests := []struct {
		headerKey string
		checkKey  string
		expected  bool
	}{
		{"Authorization", "Authorization", true},
		{"Content-Type", "Authorization", false},
	}

	for _, test := range tests {
		predicate := HasHeader(test.checkKey)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if test.headerKey != "" {
			req.Header.Set(test.headerKey, "value")
		}
		result := predicate(req)
		if result != test.expected {
			t.Errorf("HasHeader(%s) with header %s: expected %v, got %v",
				test.checkKey, test.headerKey, test.expected, result)
		}
	}
}

func TestAnd(t *testing.T) {
	tests := []struct {
		path     string
		method   string
		expected bool
	}{
		{"/api/users", http.MethodGet, true},
		{"/api/users", http.MethodPost, false},
		{"/public/users", http.MethodGet, false},
		{"/public/users", http.MethodPost, false},
	}

	predicate := And(
		PathPrefix("/api"),
		Method(http.MethodGet),
	)

	for _, test := range tests {
		req := httptest.NewRequest(test.method, test.path, nil)
		result := predicate(req)
		if result != test.expected {
			t.Errorf("And predicate for %s %s: expected %v, got %v",
				test.method, test.path, test.expected, result)
		}
	}
}

func TestOr(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/users", true},
		{"/public/images", true},
		{"/private/data", false},
	}

	predicate := Or(
		PathPrefix("/api"),
		PathPrefix("/public"),
	)

	for _, test := range tests {
		req := httptest.NewRequest(http.MethodGet, test.path, nil)
		result := predicate(req)
		if result != test.expected {
			t.Errorf("Or predicate for path %s: expected %v, got %v",
				test.path, test.expected, result)
		}
	}
}

func TestNot(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/users", false},
		{"/public/images", true},
	}

	predicate := Not(PathPrefix("/api"))

	for _, test := range tests {
		req := httptest.NewRequest(http.MethodGet, test.path, nil)
		result := predicate(req)
		if result != test.expected {
			t.Errorf("Not predicate for path %s: expected %v, got %v",
				test.path, test.expected, result)
		}
	}
}

func TestComplexPredicates(t *testing.T) {
	// Test: (path starts with /api AND method is GET) OR (path starts with /public)
	predicate := Or(
		And(
			PathPrefix("/api"),
			Method(http.MethodGet),
		),
		PathPrefix("/public"),
	)

	tests := []struct {
		method   string
		path     string
		expected bool
	}{
		{http.MethodGet, "/api/users", true},
		{http.MethodPost, "/api/users", false},
		{http.MethodGet, "/public/images", true},
		{http.MethodPost, "/public/images", true},
		{http.MethodGet, "/private/data", false},
	}

	for _, test := range tests {
		req := httptest.NewRequest(test.method, test.path, nil)
		result := predicate(req)
		if result != test.expected {
			t.Errorf("Complex predicate for %s %s: expected %v, got %v",
				test.method, test.path, test.expected, result)
		}
	}
}

func TestConditionalWithRealMiddleware(t *testing.T) {
	var headerSet bool
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Auth-Applied", "true")
			headerSet = true
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Apply auth only to /api paths
	conditional := Conditional(PathPrefix("/api"), authMiddleware)
	wrapped := conditional(handler)

	// Test /api path
	headerSet = false
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if !headerSet {
		t.Error("Auth middleware should be applied to /api paths")
	}
	if rec.Header().Get("X-Auth-Applied") != "true" {
		t.Error("Expected X-Auth-Applied header")
	}

	// Test /public path
	headerSet = false
	req = httptest.NewRequest(http.MethodGet, "/public/images", nil)
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if headerSet {
		t.Error("Auth middleware should not be applied to /public paths")
	}
	if rec.Header().Get("X-Auth-Applied") != "" {
		t.Error("Should not have X-Auth-Applied header")
	}
}
