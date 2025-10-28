package metadata

import (
	"encoding/json"
	"testing"
	"time"
)

// BenchmarkRegistry_Resource benchmarks single resource lookup
func BenchmarkRegistry_Resource(b *testing.B) {
	// Setup: Register test metadata
	setupBenchMetadata(b)
	defer Reset()

	registry := GetRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := registry.Resource("Post")
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

// BenchmarkRegistry_Resources benchmarks all resources query
func BenchmarkRegistry_Resources(b *testing.B) {
	// Setup: Register test metadata
	setupBenchMetadata(b)
	defer Reset()

	registry := GetRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resources := registry.Resources()
		if len(resources) == 0 {
			b.Fatal("expected non-empty resources")
		}
	}
}

// BenchmarkRegistry_Routes benchmarks route filtering
func BenchmarkRegistry_Routes(b *testing.B) {
	// Setup: Register test metadata
	setupBenchMetadata(b)
	defer Reset()

	registry := GetRegistry()

	benchmarks := []struct {
		name   string
		filter RouteFilter
	}{
		{
			name:   "all routes",
			filter: RouteFilter{},
		},
		{
			name:   "by method",
			filter: RouteFilter{Method: "GET"},
		},
		{
			name:   "by resource",
			filter: RouteFilter{Resource: "Post"},
		},
		{
			name:   "by method and resource",
			filter: RouteFilter{Method: "GET", Resource: "Post"},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				routes := registry.Routes(bm.filter)
				_ = routes
			}
		})
	}
}

// BenchmarkRegistry_Patterns benchmarks pattern queries
func BenchmarkRegistry_Patterns(b *testing.B) {
	// Setup: Register test metadata
	setupBenchMetadata(b)
	defer Reset()

	registry := GetRegistry()

	benchmarks := []struct {
		name     string
		category string
	}{
		{
			name:     "all patterns",
			category: "",
		},
		{
			name:     "by category",
			category: "hook",
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				patterns := registry.Patterns(bm.category)
				_ = patterns
			}
		})
	}
}

// BenchmarkRegistry_Dependencies benchmarks dependency graph queries
func BenchmarkRegistry_Dependencies(b *testing.B) {
	// Setup: Register test metadata
	setupBenchMetadata(b)
	defer Reset()

	registry := GetRegistry()

	benchmarks := []struct {
		name string
		opts DependencyOptions
	}{
		{
			name: "forward unlimited",
			opts: DependencyOptions{Depth: 0, Reverse: false},
		},
		{
			name: "forward depth 1",
			opts: DependencyOptions{Depth: 1, Reverse: false},
		},
		{
			name: "reverse unlimited",
			opts: DependencyOptions{Depth: 0, Reverse: true},
		},
		{
			name: "with type filter",
			opts: DependencyOptions{Depth: 0, Reverse: false, Types: []string{"belongs_to"}},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := registry.Dependencies("Post", bm.opts)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// BenchmarkRegistry_GetSchema benchmarks full schema retrieval
func BenchmarkRegistry_GetSchema(b *testing.B) {
	// Setup: Register test metadata
	setupBenchMetadata(b)
	defer Reset()

	registry := GetRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		schema := registry.GetSchema()
		if schema == nil {
			b.Fatal("expected non-nil schema")
		}
	}
}

// BenchmarkRegistry_ParallelResource benchmarks parallel resource lookups
func BenchmarkRegistry_ParallelResource(b *testing.B) {
	// Setup: Register test metadata
	setupBenchMetadata(b)
	defer Reset()

	registry := GetRegistry()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := registry.Resource("Post")
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})
}

// BenchmarkRegistry_ParallelRoutes benchmarks parallel route queries
func BenchmarkRegistry_ParallelRoutes(b *testing.B) {
	// Setup: Register test metadata
	setupBenchMetadata(b)
	defer Reset()

	registry := GetRegistry()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			routes := registry.Routes(RouteFilter{Method: "GET"})
			_ = routes
		}
	})
}

// BenchmarkRegistry_ParallelDependencies benchmarks parallel dependency queries
func BenchmarkRegistry_ParallelDependencies(b *testing.B) {
	// Setup: Register test metadata
	setupBenchMetadata(b)
	defer Reset()

	registry := GetRegistry()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := registry.Dependencies("Post", DependencyOptions{
				Depth:   1,
				Reverse: false,
			})
			if err != nil {
				b.Fatalf("unexpected error: %v", err)
			}
		}
	})
}

// BenchmarkRegistry_MixedWorkload benchmarks a realistic mixed workload
func BenchmarkRegistry_MixedWorkload(b *testing.B) {
	// Setup: Register test metadata
	setupBenchMetadata(b)
	defer Reset()

	registry := GetRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate a typical workflow
		_ = registry.Resources()
		_, _ = registry.Resource("Post")
		_ = registry.Routes(RouteFilter{Method: "GET"})
		_ = registry.Patterns("hook")
		_, _ = registry.Dependencies("Post", DependencyOptions{Depth: 1})
	}
}

// setupBenchMetadata registers test metadata for benchmarks
func setupBenchMetadata(b *testing.B) {
	b.Helper()

	meta := Metadata{
		Version:    "1.0",
		Generated:  time.Now(),
		SourceHash: "bench-hash",
		Resources: []ResourceMetadata{
			{
				Name:     "Post",
				FilePath: "/bench/post.cdt",
				Fields: []FieldMetadata{
					{Name: "id", Type: "uuid", Required: true},
					{Name: "title", Type: "string", Required: true},
					{Name: "content", Type: "text", Required: true},
					{Name: "author_id", Type: "uuid", Required: true},
				},
				Relationships: []RelationshipMetadata{
					{
						Name:           "author",
						Type:           "belongs_to",
						TargetResource: "User",
						ForeignKey:     "author_id",
					},
				},
			},
			{
				Name:     "User",
				FilePath: "/bench/user.cdt",
				Fields: []FieldMetadata{
					{Name: "id", Type: "uuid", Required: true},
					{Name: "email", Type: "string", Required: true},
					{Name: "name", Type: "string", Required: true},
				},
			},
			{
				Name:     "Comment",
				FilePath: "/bench/comment.cdt",
				Fields: []FieldMetadata{
					{Name: "id", Type: "uuid", Required: true},
					{Name: "content", Type: "text", Required: true},
					{Name: "post_id", Type: "uuid", Required: true},
					{Name: "author_id", Type: "uuid", Required: true},
				},
				Relationships: []RelationshipMetadata{
					{
						Name:           "post",
						Type:           "belongs_to",
						TargetResource: "Post",
						ForeignKey:     "post_id",
					},
					{
						Name:           "author",
						Type:           "belongs_to",
						TargetResource: "User",
						ForeignKey:     "author_id",
					},
				},
			},
		},
		Routes: []RouteMetadata{
			{Method: "GET", Path: "/posts", Resource: "Post", Operation: "list"},
			{Method: "GET", Path: "/posts/:id", Resource: "Post", Operation: "show"},
			{Method: "POST", Path: "/posts", Resource: "Post", Operation: "create"},
			{Method: "PUT", Path: "/posts/:id", Resource: "Post", Operation: "update"},
			{Method: "DELETE", Path: "/posts/:id", Resource: "Post", Operation: "delete"},
			{Method: "GET", Path: "/users", Resource: "User", Operation: "list"},
			{Method: "GET", Path: "/users/:id", Resource: "User", Operation: "show"},
			{Method: "POST", Path: "/users", Resource: "User", Operation: "create"},
			{Method: "GET", Path: "/comments", Resource: "Comment", Operation: "list"},
			{Method: "POST", Path: "/comments", Resource: "Comment", Operation: "create"},
		},
		Patterns: []PatternMetadata{
			{
				ID:          "pattern-1",
				Name:        "slug-generation",
				Category:    "hook",
				Description: "Generate slug from title",
				Frequency:   5,
			},
			{
				ID:          "pattern-2",
				Name:        "timestamp-tracking",
				Category:    "hook",
				Description: "Track created_at and updated_at",
				Frequency:   10,
			},
			{
				ID:          "pattern-3",
				Name:        "email-validation",
				Category:    "validation",
				Description: "Validate email format",
				Frequency:   3,
			},
			{
				ID:          "pattern-4",
				Name:        "uuid-generation",
				Category:    "hook",
				Description: "Auto-generate UUID for primary key",
				Frequency:   15,
			},
		},
	}

	// Build dependency graph
	meta.Dependencies = *BuildDependencyGraph(&meta)

	// Marshal and register
	data, err := json.Marshal(meta)
	if err != nil {
		b.Fatalf("failed to marshal bench metadata: %v", err)
	}

	if err := RegisterMetadata(data); err != nil {
		b.Fatalf("failed to register bench metadata: %v", err)
	}
}
