package metadata

import (
	"encoding/json"
	"testing"
)

func TestRegisterMetadata_Success(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{Name: "User", FilePath: "/app/user.cdt"},
		},
	}

	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Failed to marshal metadata: %v", err)
	}

	if err := RegisterMetadata(data); err != nil {
		t.Fatalf("RegisterMetadata failed: %v", err)
	}

	// Verify metadata was registered
	registered := GetMetadata()
	if registered == nil {
		t.Fatal("GetMetadata returned nil")
	}

	if registered.Version != meta.Version {
		t.Errorf("Version mismatch: got %s, want %s", registered.Version, meta.Version)
	}

	if len(registered.Resources) != 1 {
		t.Errorf("Resources count: got %d, want 1", len(registered.Resources))
	}
}

func TestRegisterMetadata_InvalidJSON(t *testing.T) {
	defer Reset()

	invalidJSON := []byte(`{"invalid": json}`)

	err := RegisterMetadata(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestGetMetadata_NotRegistered(t *testing.T) {
	defer Reset()

	meta := GetMetadata()
	if meta != nil {
		t.Error("Expected nil metadata when not registered")
	}
}

func TestQueryResources(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{Name: "User", FilePath: "/app/user.cdt"},
			{Name: "Post", FilePath: "/app/post.cdt"},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	resources := QueryResources()
	if len(resources) != 2 {
		t.Errorf("QueryResources count: got %d, want 2", len(resources))
	}
}

func TestQueryResource_Found(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{Name: "User", FilePath: "/app/user.cdt"},
			{Name: "Post", FilePath: "/app/post.cdt"},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	resource, err := QueryResource("Post")
	if err != nil {
		t.Fatalf("QueryResource failed: %v", err)
	}

	if resource.Name != "Post" {
		t.Errorf("Resource name: got %s, want Post", resource.Name)
	}
}

func TestQueryResource_NotFound(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{Name: "User", FilePath: "/app/user.cdt"},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	_, err := QueryResource("NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent resource")
	}
}

func TestQueryPatterns(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Patterns: []PatternMetadata{
			{Name: "pattern1", Template: "template1"},
			{Name: "pattern2", Template: "template2"},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	patterns := QueryPatterns()
	if len(patterns) != 2 {
		t.Errorf("QueryPatterns count: got %d, want 2", len(patterns))
	}
}

func TestQueryRoutes(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Routes: []RouteMetadata{
			{Method: "GET", Path: "/users"},
			{Method: "POST", Path: "/users"},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	routes := QueryRoutes()
	if len(routes) != 2 {
		t.Errorf("QueryRoutes count: got %d, want 2", len(routes))
	}
}

func TestReset(t *testing.T) {
	meta := &Metadata{Version: "1.0.0"}
	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	if GetMetadata() == nil {
		t.Fatal("Metadata should be registered")
	}

	Reset()

	if GetMetadata() != nil {
		t.Error("Metadata should be nil after Reset()")
	}
}

func TestQueryRoutesByMethod(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Routes: []RouteMetadata{
			{Method: "GET", Path: "/users", Handler: "ListUsers"},
			{Method: "GET", Path: "/posts", Handler: "ListPosts"},
			{Method: "POST", Path: "/users", Handler: "CreateUser"},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// Test GET routes
	getRoutes := QueryRoutesByMethod("GET")
	if len(getRoutes) != 2 {
		t.Errorf("Expected 2 GET routes, got %d", len(getRoutes))
	}

	// Test POST routes
	postRoutes := QueryRoutesByMethod("POST")
	if len(postRoutes) != 1 {
		t.Errorf("Expected 1 POST route, got %d", len(postRoutes))
	}

	// Test case insensitivity
	getRoutesLower := QueryRoutesByMethod("get")
	if len(getRoutesLower) != 2 {
		t.Errorf("Expected case-insensitive query, got %d routes", len(getRoutesLower))
	}

	// Test non-existent method
	deleteRoutes := QueryRoutesByMethod("DELETE")
	if len(deleteRoutes) != 0 {
		t.Errorf("Expected 0 DELETE routes, got %d", len(deleteRoutes))
	}
}

func TestQueryRoutesByPath(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Routes: []RouteMetadata{
			{Method: "GET", Path: "/users", Handler: "ListUsers"},
			{Method: "POST", Path: "/users", Handler: "CreateUser"},
			{Method: "GET", Path: "/posts", Handler: "ListPosts"},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// Test /users path
	usersRoutes := QueryRoutesByPath("/users")
	if len(usersRoutes) != 2 {
		t.Errorf("Expected 2 routes for /users, got %d", len(usersRoutes))
	}

	// Test /posts path
	postsRoutes := QueryRoutesByPath("/posts")
	if len(postsRoutes) != 1 {
		t.Errorf("Expected 1 route for /posts, got %d", len(postsRoutes))
	}

	// Test non-existent path
	nonExistent := QueryRoutesByPath("/nonexistent")
	if len(nonExistent) != 0 {
		t.Errorf("Expected 0 routes for non-existent path, got %d", len(nonExistent))
	}
}

func TestQueryPattern(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Patterns: []PatternMetadata{
			{Name: "auth_handler", Template: "@before create: [auth]"},
			{Name: "cached_handler", Template: "@on list: [cache(300)]"},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// Test finding existing pattern
	pattern, err := QueryPattern("auth_handler")
	if err != nil {
		t.Fatalf("QueryPattern failed: %v", err)
	}
	if pattern.Name != "auth_handler" {
		t.Errorf("Expected pattern name 'auth_handler', got '%s'", pattern.Name)
	}

	// Test non-existent pattern
	_, err = QueryPattern("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent pattern")
	}
}

func TestQueryRelationshipsTo(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Post",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
				},
			},
			{
				Name: "Comment",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
					{Name: "post", TargetResource: "Post", Type: "belongs_to"},
				},
			},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// Query relationships pointing to User
	userRels := QueryRelationshipsTo("User")
	if len(userRels) != 2 {
		t.Errorf("Expected 2 relationships to User, got %d", len(userRels))
	}

	// Verify source resources
	sources := make(map[string]bool)
	for _, rel := range userRels {
		sources[rel.SourceResource] = true
	}
	if !sources["Post"] || !sources["Comment"] {
		t.Error("Expected relationships from Post and Comment")
	}

	// Query relationships pointing to Post
	postRels := QueryRelationshipsTo("Post")
	if len(postRels) != 1 {
		t.Errorf("Expected 1 relationship to Post, got %d", len(postRels))
	}
	if postRels[0].SourceResource != "Comment" {
		t.Errorf("Expected relationship from Comment, got %s", postRels[0].SourceResource)
	}

	// Query non-existent target
	nonExistent := QueryRelationshipsTo("NonExistent")
	if len(nonExistent) != 0 {
		t.Errorf("Expected 0 relationships to NonExistent, got %d", len(nonExistent))
	}
}

func TestQueryRelationshipsFrom(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Post",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User", Type: "belongs_to"},
					{Name: "category", TargetResource: "Category", Type: "belongs_to"},
				},
			},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// Query relationships from Post
	rels, err := QueryRelationshipsFrom("Post")
	if err != nil {
		t.Fatalf("QueryRelationshipsFrom failed: %v", err)
	}
	if len(rels) != 2 {
		t.Errorf("Expected 2 relationships from Post, got %d", len(rels))
	}

	// Query non-existent resource
	_, err = QueryRelationshipsFrom("NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent resource")
	}
}

func TestQueryResourcesByPattern(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{Name: "User"},
			{Name: "UserProfile"},
			{Name: "Post"},
			{Name: "PostComment"},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	tests := []struct {
		pattern  string
		expected int
		desc     string
	}{
		{"*", 4, "wildcard all"},
		{"User", 1, "exact match"},
		{"User*", 2, "prefix match"},
		{"*Profile", 1, "suffix match"},
		{"Post*", 2, "prefix Post"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			results := QueryResourcesByPattern(tt.pattern)
			if len(results) != tt.expected {
				t.Errorf("Pattern '%s': expected %d resources, got %d",
					tt.pattern, tt.expected, len(results))
			}
		})
	}
}

func TestQueryFieldsByType(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "User",
				Fields: []FieldMetadata{
					{Name: "id", Type: "uuid!"},
					{Name: "name", Type: "string!"},
				},
			},
			{
				Name: "Post",
				Fields: []FieldMetadata{
					{Name: "id", Type: "uuid!"},
					{Name: "title", Type: "string!"},
					{Name: "count", Type: "integer!"},
				},
			},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// Query uuid fields
	uuidFields := QueryFieldsByType("uuid")
	if len(uuidFields) != 2 {
		t.Errorf("Expected 2 uuid fields, got %d", len(uuidFields))
	}

	// Query string fields
	stringFields := QueryFieldsByType("string")
	if len(stringFields) != 2 {
		t.Errorf("Expected 2 string fields, got %d", len(stringFields))
	}

	// Query integer fields
	intFields := QueryFieldsByType("integer")
	if len(intFields) != 1 {
		t.Errorf("Expected 1 integer field, got %d", len(intFields))
	}

	// Verify field references
	for _, ref := range uuidFields {
		if ref.Field.Name != "id" {
			t.Errorf("Expected field name 'id', got '%s'", ref.Field.Name)
		}
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		s       string
		pattern string
		match   bool
	}{
		{"User", "User", true},
		{"User", "Post", false},
		{"User", "*", true},
		{"UserProfile", "User*", true},
		{"User", "User*", true},
		{"Profile", "User*", false},
		{"UserProfile", "*Profile", true},
		{"Profile", "*Profile", true},
		{"User", "*Profile", false},
		{"UserProfileManager", "User*Manager", true},
		{"UserAdmin", "User*Manager", false},
	}

	for _, tt := range tests {
		result := matchPattern(tt.s, tt.pattern)
		if result != tt.match {
			t.Errorf("matchPattern(%q, %q) = %v, want %v",
				tt.s, tt.pattern, result, tt.match)
		}
	}
}

func TestIndexBuilding(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{
				Name: "Post",
				Relationships: []RelationshipMetadata{
					{Name: "author", TargetResource: "User"},
				},
			},
		},
		Routes: []RouteMetadata{
			{Method: "GET", Path: "/posts"},
			{Method: "POST", Path: "/posts"},
		},
		Patterns: []PatternMetadata{
			{Name: "pattern1"},
		},
	}

	data, _ := json.Marshal(meta)
	if err := RegisterMetadata(data); err != nil {
		t.Fatalf("RegisterMetadata failed: %v", err)
	}

	// Verify indexes were built
	if !globalRegistry.initialized.Load() {
		t.Error("Registry should be marked as initialized")
	}

	// Verify resource index
	if len(globalRegistry.resourcesByName) != 1 {
		t.Errorf("Expected 1 resource in index, got %d", len(globalRegistry.resourcesByName))
	}

	// Verify route indexes
	if len(globalRegistry.routesByPath) != 1 {
		t.Errorf("Expected 1 path in route index, got %d", len(globalRegistry.routesByPath))
	}
	if len(globalRegistry.routesByMethod) != 2 {
		t.Errorf("Expected 2 methods in route index, got %d", len(globalRegistry.routesByMethod))
	}

	// Verify pattern index
	if len(globalRegistry.patternsByName) != 1 {
		t.Errorf("Expected 1 pattern in index, got %d", len(globalRegistry.patternsByName))
	}

	// Verify relationship index
	if len(globalRegistry.relationshipIndex) != 1 {
		t.Errorf("Expected 1 entry in relationship index, got %d", len(globalRegistry.relationshipIndex))
	}
}

func TestCaching(t *testing.T) {
	defer Reset()

	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{Name: "User"},
			{Name: "UserProfile"},
		},
	}

	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// First query - should populate cache
	results1 := QueryResourcesByPattern("User*")
	if len(results1) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results1))
	}

	// Second query - should use cache
	results2 := QueryResourcesByPattern("User*")
	if len(results2) != 2 {
		t.Fatalf("Expected 2 results from cache, got %d", len(results2))
	}

	// Verify cache was used
	cached := globalRegistry.getCached("pattern:User*")
	if cached == nil {
		t.Error("Expected cached result")
	}
}
