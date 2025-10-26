package metadata

import (
	"encoding/json"
	"fmt"
	"testing"
)

// BenchmarkRegistryInit measures initialization time with pre-computed indexes
func BenchmarkRegistryInit(b *testing.B) {
	meta := generateLargeMetadata(50) // 50 resources
	data, _ := json.Marshal(meta)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Reset()
		if err := RegisterMetadata(data); err != nil {
			b.Fatalf("RegisterMetadata failed: %v", err)
		}
	}
}

// BenchmarkQueryResource measures indexed resource lookup performance
func BenchmarkQueryResource(b *testing.B) {
	meta := generateLargeMetadata(50)
	data, _ := json.Marshal(meta)
	RegisterMetadata(data)
	defer Reset()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := QueryResource("Resource25")
		if err != nil {
			b.Fatalf("QueryResource failed: %v", err)
		}
	}
}

// BenchmarkQueryRoutesByMethod measures indexed route lookup
func BenchmarkQueryRoutesByMethod(b *testing.B) {
	meta := generateLargeMetadata(50)
	data, _ := json.Marshal(meta)
	RegisterMetadata(data)
	defer Reset()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		routes := QueryRoutesByMethod("GET")
		if len(routes) == 0 {
			b.Fatal("Expected routes")
		}
	}
}

// BenchmarkQueryRelationshipsTo measures relationship index lookup
func BenchmarkQueryRelationshipsTo(b *testing.B) {
	meta := generateLargeMetadata(50)
	data, _ := json.Marshal(meta)
	RegisterMetadata(data)
	defer Reset()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rels := QueryRelationshipsTo("Resource0")
		_ = rels
	}
}

// BenchmarkQueryResourcesByPattern measures pattern matching with caching
func BenchmarkQueryResourcesByPattern(b *testing.B) {
	meta := generateLargeMetadata(100)
	data, _ := json.Marshal(meta)
	RegisterMetadata(data)
	defer Reset()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := QueryResourcesByPattern("Resource*")
		if len(results) == 0 {
			b.Fatal("Expected results")
		}
	}
}

// BenchmarkQueryFieldsByType measures field type queries with caching
func BenchmarkQueryFieldsByType(b *testing.B) {
	meta := generateLargeMetadata(100)
	data, _ := json.Marshal(meta)
	RegisterMetadata(data)
	defer Reset()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fields := QueryFieldsByType("uuid")
		if len(fields) == 0 {
			b.Fatal("Expected fields")
		}
	}
}

// BenchmarkConcurrentQueries measures thread-safety under concurrent load
func BenchmarkConcurrentQueries(b *testing.B) {
	meta := generateLargeMetadata(50)
	data, _ := json.Marshal(meta)
	RegisterMetadata(data)
	defer Reset()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix of different query types
			QueryResource("Resource10")
			QueryRoutesByMethod("GET")
			QueryRelationshipsTo("Resource0")
			QueryResourcesByPattern("Resource*")
		}
	})
}

// generateLargeMetadata creates test metadata with N resources
func generateLargeMetadata(n int) *Metadata {
	meta := &Metadata{
		Version:   "1.0.0",
		Resources: make([]ResourceMetadata, n),
		Routes:    make([]RouteMetadata, 0, n*5),
		Patterns:  make([]PatternMetadata, 0, 10),
	}

	for i := 0; i < n; i++ {
		// Generate resource names like Resource0, Resource1, ..., Resource99
		resourceName := fmt.Sprintf("Resource%d", i)

		meta.Resources[i] = ResourceMetadata{
			Name: resourceName,
			Fields: []FieldMetadata{
				{Name: "id", Type: "uuid!"},
				{Name: "name", Type: "string!"},
				{Name: "email", Type: "string!"},
				{Name: "created_at", Type: "timestamp!"},
			},
			Relationships: []RelationshipMetadata{},
		}

		// Add a relationship to the first resource (creates many-to-one)
		if i > 0 {
			meta.Resources[i].Relationships = append(meta.Resources[i].Relationships,
				RelationshipMetadata{
					Name:           "related",
					TargetResource: "Resource0",
					Type:           "belongs_to",
				},
			)
		}

		// Generate standard CRUD routes
		basePath := "/" + resourceName
		meta.Routes = append(meta.Routes,
			RouteMetadata{Method: "GET", Path: basePath, Handler: "Index", Resource: resourceName},
			RouteMetadata{Method: "GET", Path: basePath + "/:id", Handler: "Show", Resource: resourceName},
			RouteMetadata{Method: "POST", Path: basePath, Handler: "Create", Resource: resourceName},
			RouteMetadata{Method: "PUT", Path: basePath + "/:id", Handler: "Update", Resource: resourceName},
			RouteMetadata{Method: "DELETE", Path: basePath + "/:id", Handler: "Delete", Resource: resourceName},
		)
	}

	// Add some patterns
	meta.Patterns = []PatternMetadata{
		{Name: "auth_handler", Template: "@before create: [auth]", Frequency: n / 2},
		{Name: "cached_handler", Template: "@on list: [cache(300)]", Frequency: n / 3},
		{Name: "transactional", Template: "@after create @transaction", Frequency: n / 4},
	}

	return meta
}
