package profiling

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if !config.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if config.Path != "/debug/pprof" {
		t.Errorf("Expected Path '/debug/pprof', got %q", config.Path)
	}

	if !config.EnableCPUProfile {
		t.Error("Expected EnableCPUProfile to be true")
	}

	if !config.EnableMemProfile {
		t.Error("Expected EnableMemProfile to be true")
	}

	if !config.EnableBlockProfile {
		t.Error("Expected EnableBlockProfile to be true")
	}

	if !config.EnableMutexProfile {
		t.Error("Expected EnableMutexProfile to be true")
	}

	if config.BlockRate != 1 {
		t.Errorf("Expected BlockRate 1, got %d", config.BlockRate)
	}

	if config.MutexFraction != 1 {
		t.Errorf("Expected MutexFraction 1, got %d", config.MutexFraction)
	}
}

func TestRegisterRoutes_Enabled(t *testing.T) {
	router := chi.NewRouter()
	config := DefaultConfig()

	RegisterRoutes(router, config)

	// Test that routes are registered
	tests := []struct {
		path       string
		shouldWork bool
	}{
		{"/debug/pprof/", true},
		{"/debug/pprof/cmdline", true},
		{"/debug/pprof/profile", true},
		{"/debug/pprof/symbol", true},
		{"/debug/pprof/trace", true},
		{"/debug/pprof/allocs", true},
		{"/debug/pprof/block", true},
		{"/debug/pprof/goroutine", true},
		{"/debug/pprof/heap", true},
		{"/debug/pprof/mutex", true},
		{"/debug/pprof/threadcreate", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			// Should not return 404 if route is registered
			if rec.Code == http.StatusNotFound && tt.shouldWork {
				t.Errorf("Route %s not found (got 404)", tt.path)
			}
		})
	}
}

func TestRegisterRoutes_Disabled(t *testing.T) {
	router := chi.NewRouter()
	config := DefaultConfig()
	config.Enabled = false

	RegisterRoutes(router, config)

	// Test that routes are not registered
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should return 404 since profiling is disabled
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for disabled profiling, got %d", rec.Code)
	}
}

func TestRegisterRoutes_NilConfig(t *testing.T) {
	router := chi.NewRouter()

	// Should use default config when nil
	RegisterRoutes(router, nil)

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should work with default config
	if rec.Code == http.StatusNotFound {
		t.Error("Route should be registered with default config")
	}
}

func TestRegisterRoutes_BlockProfileConfig(t *testing.T) {
	router := chi.NewRouter()
	config := DefaultConfig()
	config.EnableBlockProfile = true
	config.BlockRate = 100

	RegisterRoutes(router, config)

	// Note: We can't directly verify the rate was set to 100,
	// but we can verify the route was registered
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/block", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Error("Block profile route should be registered")
	}
}

func TestRegisterRoutes_MutexProfileConfig(t *testing.T) {
	router := chi.NewRouter()
	config := DefaultConfig()
	config.EnableMutexProfile = true
	config.MutexFraction = 10

	RegisterRoutes(router, config)

	// Mutex profile fraction should be set
	// Note: We can't directly verify the fraction was set to 10,
	// but we can verify the route was registered
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/mutex", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Error("Mutex profile route should be registered")
	}
}

func TestRegisterSimple(t *testing.T) {
	type mockRouter struct {
		routes map[string]http.HandlerFunc
	}

	router := &mockRouter{
		routes: make(map[string]http.HandlerFunc),
	}

	// Implement Get method
	getFunc := func(pattern string, handler http.HandlerFunc) {
		router.routes[pattern] = handler
	}

	adapter := &RouterAdapter{getFunc: getFunc}

	config := DefaultConfig()
	RegisterSimple(adapter, config)

	// Check that routes were registered
	expectedRoutes := []string{
		"/debug/pprof/",
		"/debug/pprof/cmdline",
		"/debug/pprof/profile",
		"/debug/pprof/symbol",
		"/debug/pprof/trace",
		"/debug/pprof/allocs",
		"/debug/pprof/block",
		"/debug/pprof/goroutine",
		"/debug/pprof/heap",
		"/debug/pprof/mutex",
		"/debug/pprof/threadcreate",
	}

	for _, route := range expectedRoutes {
		if _, ok := router.routes[route]; !ok {
			t.Errorf("Expected route %s to be registered", route)
		}
	}
}

func TestRegisterSimple_Disabled(t *testing.T) {
	type mockRouter struct {
		routes map[string]http.HandlerFunc
	}

	router := &mockRouter{
		routes: make(map[string]http.HandlerFunc),
	}

	getFunc := func(pattern string, handler http.HandlerFunc) {
		router.routes[pattern] = handler
	}

	adapter := &RouterAdapter{getFunc: getFunc}

	config := DefaultConfig()
	config.Enabled = false

	RegisterSimple(adapter, config)

	// No routes should be registered when disabled
	if len(router.routes) > 0 {
		t.Errorf("Expected no routes when disabled, got %d", len(router.routes))
	}
}

func TestHandler_Enabled(t *testing.T) {
	config := DefaultConfig()
	handler := Handler(config)

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 200 for index page
	if rec.Code == http.StatusForbidden {
		t.Error("Expected profiling to be enabled")
	}
}

func TestHandler_Disabled(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false
	handler := Handler(config)

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 403 when disabled
	if rec.Code != http.StatusForbidden {
		t.Errorf("Expected 403 when disabled, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Profiling is disabled") {
		t.Errorf("Expected 'Profiling is disabled' message, got %q", body)
	}
}

func TestHandler_NilConfig(t *testing.T) {
	handler := Handler(nil)

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should work with default config
	if rec.Code == http.StatusForbidden {
		t.Error("Expected profiling to work with nil config (default)")
	}
}

func TestStartProfilingServer_Disabled(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = false

	err := StartProfilingServer(":0", config)
	if err == nil {
		t.Error("Expected error when starting disabled profiling server")
	}

	expectedMsg := "profiling is disabled"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestEnableDefaultProfiling(t *testing.T) {
	router := chi.NewRouter()
	EnableDefaultProfiling(router)

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// Should be enabled with default config
	if rec.Code == http.StatusNotFound {
		t.Error("Expected profiling to be enabled")
	}
}

func TestEnableProfilingHTTP(t *testing.T) {
	type mockRouter struct {
		routes map[string]http.HandlerFunc
	}

	router := &mockRouter{
		routes: make(map[string]http.HandlerFunc),
	}

	getFunc := func(pattern string, handler http.HandlerFunc) {
		router.routes[pattern] = handler
	}

	adapter := &RouterAdapter{getFunc: getFunc}

	EnableProfilingHTTP(adapter)

	// Should register routes
	if len(router.routes) == 0 {
		t.Error("Expected routes to be registered")
	}
}

func TestRouterAdapter(t *testing.T) {
	called := false
	var capturedPattern string
	var capturedHandler http.HandlerFunc

	getFunc := func(pattern string, handler http.HandlerFunc) {
		called = true
		capturedPattern = pattern
		capturedHandler = handler
	}

	adapter := WrapRouter(getFunc)
	adapter.Get("/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	if !called {
		t.Error("Expected getFunc to be called")
	}

	if capturedPattern != "/test" {
		t.Errorf("Expected pattern '/test', got %q", capturedPattern)
	}

	if capturedHandler == nil {
		t.Error("Expected handler to be captured")
	}
}

func TestRuntimeStats(t *testing.T) {
	stats := RuntimeStats()

	// Check that all expected keys exist
	if _, ok := stats["goroutines"]; !ok {
		t.Error("Expected 'goroutines' in stats")
	}

	if _, ok := stats["memory"]; !ok {
		t.Error("Expected 'memory' in stats")
	}

	if _, ok := stats["cpu"]; !ok {
		t.Error("Expected 'cpu' in stats")
	}

	// Check memory stats
	if mem, ok := stats["memory"].(map[string]interface{}); ok {
		expectedKeys := []string{"alloc", "total_alloc", "sys", "num_gc"}
		for _, key := range expectedKeys {
			if _, ok := mem[key]; !ok {
				t.Errorf("Expected %q in memory stats", key)
			}
		}
	} else {
		t.Error("Expected memory to be a map")
	}

	// Check CPU stats
	if cpu, ok := stats["cpu"].(map[string]interface{}); ok {
		expectedKeys := []string{"num_cpu", "num_cgo_call"}
		for _, key := range expectedKeys {
			if _, ok := cpu[key]; !ok {
				t.Errorf("Expected %q in CPU stats", key)
			}
		}
	} else {
		t.Error("Expected cpu to be a map")
	}

	// Verify goroutines value is reasonable
	if goroutines, ok := stats["goroutines"].(int); ok {
		if goroutines <= 0 {
			t.Errorf("Expected positive goroutines count, got %d", goroutines)
		}
	} else {
		t.Error("Expected goroutines to be an int")
	}
}

func TestStatsHandler(t *testing.T) {
	handler := StatsHandler()

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %q", contentType)
	}

	body := rec.Body.String()

	// Verify JSON structure
	expectedKeys := []string{"goroutines", "memory", "cpu"}
	for _, key := range expectedKeys {
		if !strings.Contains(body, key) {
			t.Errorf("Expected body to contain %q, got %q", key, body)
		}
	}

	// Verify it's valid JSON by parsing it
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Errorf("Failed to parse JSON: %v\nBody: %s", err, body)
	}
}

func TestStatsHandler_ValidJSONOutput(t *testing.T) {
	handler := StatsHandler()

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Parse the JSON to ensure it's valid
	var stats map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// Verify structure
	if _, ok := stats["goroutines"]; !ok {
		t.Error("Expected 'goroutines' in JSON")
	}

	if mem, ok := stats["memory"].(map[string]interface{}); ok {
		if _, ok := mem["alloc"]; !ok {
			t.Error("Expected 'alloc' in memory stats")
		}
	} else {
		t.Error("Expected 'memory' to be an object")
	}

	if cpu, ok := stats["cpu"].(map[string]interface{}); ok {
		if _, ok := cpu["num_cpu"]; !ok {
			t.Error("Expected 'num_cpu' in CPU stats")
		}
	} else {
		t.Error("Expected 'cpu' to be an object")
	}
}

func TestPprofIndex(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router, DefaultConfig())

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	// Index page should contain links to profiles
	if !strings.Contains(body, "allocs") || !strings.Contains(body, "heap") {
		t.Error("Expected index page to contain profile links")
	}
}

func TestPprofCmdline(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router, DefaultConfig())

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/cmdline", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestPprofHeapProfile(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router, DefaultConfig())

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/heap", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Should return pprof binary data
	body, _ := io.ReadAll(rec.Body)
	if len(body) == 0 {
		t.Error("Expected heap profile data")
	}
}

func TestPprofGoroutineProfile(t *testing.T) {
	router := chi.NewRouter()
	RegisterRoutes(router, DefaultConfig())

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/goroutine", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body, _ := io.ReadAll(rec.Body)
	if len(body) == 0 {
		t.Error("Expected goroutine profile data")
	}
}

func BenchmarkRuntimeStats(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RuntimeStats()
	}
}

func BenchmarkStatsHandler(b *testing.B) {
	handler := StatsHandler()
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}
