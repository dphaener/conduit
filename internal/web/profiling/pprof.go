package profiling

// Package profiling provides pprof profiling endpoints for performance analysis.
//
// SECURITY WARNING: Profiling endpoints expose sensitive runtime information
// about your application including memory contents, goroutine stacks, and
// performance characteristics. These endpoints should NEVER be exposed on
// public-facing servers without proper authentication and authorization.
//
// Best Practices:
//   - Only enable profiling in development and staging environments
//   - If profiling is needed in production, use a separate port (see StartProfilingServer)
//   - Protect profiling endpoints with authentication middleware
//   - Use firewall rules to restrict access to profiling ports
//   - Consider using IP allowlists to limit access to trusted networks
//   - Monitor access logs for unauthorized profiling requests
//
// Example safe usage in production:
//
//	// Run profiling server on internal-only port
//	go profiling.StartProfilingServer("localhost:6060", profiling.DefaultConfig())
//
// Example with authentication:
//
//	router.Group(func(r chi.Router) {
//	    r.Use(AuthenticationMiddleware)  // Require authentication
//	    r.Use(AdminOnlyMiddleware)       // Require admin role
//	    profiling.RegisterRoutes(r, profiling.DefaultConfig())
//	})

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"

	"github.com/go-chi/chi/v5"
)

// Config holds profiling configuration
type Config struct {
	// Enabled determines if profiling is enabled
	Enabled bool

	// Path is the URL path prefix for profiling endpoints (default: "/debug/pprof")
	Path string

	// EnableCPUProfile enables CPU profiling endpoint
	EnableCPUProfile bool

	// EnableMemProfile enables memory profiling endpoint
	EnableMemProfile bool

	// EnableBlockProfile enables block profiling
	EnableBlockProfile bool

	// EnableMutexProfile enables mutex profiling
	EnableMutexProfile bool

	// BlockRate sets the block profiling rate (0 = disabled)
	BlockRate int

	// MutexFraction sets the mutex profiling fraction (0 = disabled)
	MutexFraction int
}

// DefaultConfig returns default profiling configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:            true,
		Path:               "/debug/pprof",
		EnableCPUProfile:   true,
		EnableMemProfile:   true,
		EnableBlockProfile: true,
		EnableMutexProfile: true,
		BlockRate:          1,
		MutexFraction:      1,
	}
}

// RegisterRoutes registers pprof profiling routes with a router
func RegisterRoutes(router chi.Router, config *Config) {
	if config == nil {
		config = DefaultConfig()
	}

	if !config.Enabled {
		return
	}

	// Configure runtime profiling
	if config.EnableBlockProfile {
		runtime.SetBlockProfileRate(config.BlockRate)
	}
	if config.EnableMutexProfile {
		runtime.SetMutexProfileFraction(config.MutexFraction)
	}

	// Register pprof routes
	router.Route(config.Path, func(r chi.Router) {
		r.HandleFunc("/", pprof.Index)
		r.HandleFunc("/cmdline", pprof.Cmdline)
		r.HandleFunc("/profile", pprof.Profile)
		r.HandleFunc("/symbol", pprof.Symbol)
		r.HandleFunc("/trace", pprof.Trace)

		// Manually add routes for each profile type
		r.Handle("/allocs", pprof.Handler("allocs"))
		r.Handle("/block", pprof.Handler("block"))
		r.Handle("/goroutine", pprof.Handler("goroutine"))
		r.Handle("/heap", pprof.Handler("heap"))
		r.Handle("/mutex", pprof.Handler("mutex"))
		r.Handle("/threadcreate", pprof.Handler("threadcreate"))
	})
}

// HTTPRouter is a minimal interface for HTTP routers
type HTTPRouter interface {
	Get(pattern string, handler http.HandlerFunc)
}

// RegisterSimple registers pprof routes with any router that has a Get method
func RegisterSimple(router HTTPRouter, config *Config) {
	if config == nil {
		config = DefaultConfig()
	}

	if !config.Enabled {
		return
	}

	// Configure runtime profiling
	if config.EnableBlockProfile {
		runtime.SetBlockProfileRate(config.BlockRate)
	}
	if config.EnableMutexProfile {
		runtime.SetMutexProfileFraction(config.MutexFraction)
	}

	// Register pprof routes
	prefix := config.Path
	router.Get(prefix+"/", pprof.Index)
	router.Get(prefix+"/cmdline", pprof.Cmdline)
	router.Get(prefix+"/profile", pprof.Profile)
	router.Get(prefix+"/symbol", pprof.Symbol)
	router.Get(prefix+"/trace", pprof.Trace)

	// For handlers that return http.Handler, we need to wrap them
	router.Get(prefix+"/allocs", func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("allocs").ServeHTTP(w, r)
	})
	router.Get(prefix+"/block", func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("block").ServeHTTP(w, r)
	})
	router.Get(prefix+"/goroutine", func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("goroutine").ServeHTTP(w, r)
	})
	router.Get(prefix+"/heap", func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("heap").ServeHTTP(w, r)
	})
	router.Get(prefix+"/mutex", func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("mutex").ServeHTTP(w, r)
	})
	router.Get(prefix+"/threadcreate", func(w http.ResponseWriter, r *http.Request) {
		pprof.Handler("threadcreate").ServeHTTP(w, r)
	})
}

// Handler returns an http.Handler for profiling endpoints
func Handler(config *Config) http.Handler {
	if config == nil {
		config = DefaultConfig()
	}

	if !config.Enabled {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Profiling is disabled", http.StatusForbidden)
		})
	}

	// Configure runtime profiling
	if config.EnableBlockProfile {
		runtime.SetBlockProfileRate(config.BlockRate)
	}
	if config.EnableMutexProfile {
		runtime.SetMutexProfileFraction(config.MutexFraction)
	}

	router := chi.NewRouter()
	RegisterRoutes(router, config)

	return router
}

// StartProfilingServer starts a dedicated profiling server on a separate port
func StartProfilingServer(addr string, config *Config) error {
	if config == nil {
		config = DefaultConfig()
	}

	if !config.Enabled {
		return fmt.Errorf("profiling is disabled")
	}

	router := chi.NewRouter()
	RegisterRoutes(router, config)

	fmt.Printf("Starting profiling server on %s%s\n", addr, config.Path)
	return http.ListenAndServe(addr, router)
}

// EnableDefaultProfiling enables profiling with default configuration
func EnableDefaultProfiling(router chi.Router) {
	RegisterRoutes(router, DefaultConfig())
}

// EnableProfilingHTTP enables profiling with any HTTP router
func EnableProfilingHTTP(router HTTPRouter) {
	RegisterSimple(router, DefaultConfig())
}

// RouterAdapter wraps a router to match the HTTPRouter interface
type RouterAdapter struct {
	getFunc func(pattern string, handler http.HandlerFunc)
}

func (r *RouterAdapter) Get(pattern string, handler http.HandlerFunc) {
	r.getFunc(pattern, handler)
}

// WrapRouter creates an adapter for routers whose Get method returns a value
func WrapRouter(getFunc func(pattern string, handler http.HandlerFunc)) *RouterAdapter {
	return &RouterAdapter{getFunc: getFunc}
}

// RuntimeStats returns current runtime statistics
func RuntimeStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"goroutines": runtime.NumGoroutine(),
		"memory": map[string]interface{}{
			"alloc":       m.Alloc,
			"total_alloc": m.TotalAlloc,
			"sys":         m.Sys,
			"num_gc":      m.NumGC,
		},
		"cpu": map[string]interface{}{
			"num_cpu":      runtime.NumCPU(),
			"num_cgo_call": runtime.NumCgoCall(),
		},
	}
}

// StatsHandler returns an HTTP handler that serves runtime statistics
func StatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		stats := RuntimeStats()

		// Simple JSON serialization
		fmt.Fprintf(w, "{\n")
		fmt.Fprintf(w, "  \"goroutines\": %d,\n", stats["goroutines"])

		if mem, ok := stats["memory"].(map[string]interface{}); ok {
			fmt.Fprintf(w, "  \"memory\": {\n")
			fmt.Fprintf(w, "    \"alloc\": %d,\n", mem["alloc"])
			fmt.Fprintf(w, "    \"total_alloc\": %d,\n", mem["total_alloc"])
			fmt.Fprintf(w, "    \"sys\": %d,\n", mem["sys"])
			fmt.Fprintf(w, "    \"num_gc\": %d\n", mem["num_gc"])
			fmt.Fprintf(w, "  },\n")
		}

		if cpu, ok := stats["cpu"].(map[string]interface{}); ok {
			fmt.Fprintf(w, "  \"cpu\": {\n")
			fmt.Fprintf(w, "    \"num_cpu\": %d,\n", cpu["num_cpu"])
			fmt.Fprintf(w, "    \"num_cgo_call\": %d\n", cpu["num_cgo_call"])
			fmt.Fprintf(w, "  }\n")
		}

		fmt.Fprintf(w, "}\n")
	}
}
