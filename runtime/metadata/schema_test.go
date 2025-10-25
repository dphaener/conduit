package metadata

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestMetadataJSONSerialization tests that Metadata can be serialized to and from JSON.
func TestMetadataJSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second) // Truncate for comparison

	original := Metadata{
		Version:    "1.0",
		Generated:  now,
		SourceHash: "abc123def456",
		Resources: []ResourceMetadata{
			{
				Name:          "Post",
				Documentation: "Blog post resource",
				FilePath:      "/app/resources/post.cdt",
				Fields: []FieldMetadata{
					{
						Name:          "id",
						Type:          "uuid",
						Nullable:      false,
						Required:      true,
						Constraints:   []string{"@primary", "@auto"},
						Documentation: "Primary key",
					},
					{
						Name:          "title",
						Type:          "string",
						Nullable:      false,
						Required:      true,
						Constraints:   []string{"@min(5)", "@max(200)"},
						Documentation: "Post title",
					},
				},
				Relationships: []RelationshipMetadata{
					{
						Name:           "author",
						Type:           "belongs_to",
						TargetResource: "User",
						ForeignKey:     "author_id",
						OnDelete:       "restrict",
					},
				},
				Hooks: []HookMetadata{
					{
						Type:        "before_create",
						Transaction: true,
						Async:       false,
						SourceCode:  "self.slug = String.slugify(self.title)",
						LineNumber:  15,
					},
				},
			},
		},
		Routes: []RouteMetadata{
			{
				Method:       "GET",
				Path:         "/posts",
				Handler:      "ListPosts",
				Resource:     "Post",
				Operation:    "list",
				Middleware:   []string{"auth"},
				ResponseBody: "[]Post",
			},
		},
		Patterns: []PatternMetadata{
			{
				ID:          "pattern-1",
				Name:        "slug-generation",
				Category:    "hook",
				Description: "Auto-generate URL slug from title",
				Template:    "@before create { self.slug = String.slugify(self.title) }",
				Examples: []PatternExample{
					{
						Resource:   "Post",
						FilePath:   "/app/resources/post.cdt",
						LineNumber: 15,
						Code:       "self.slug = String.slugify(self.title)",
					},
				},
				Frequency:  3,
				Confidence: 0.95,
			},
		},
		Dependencies: DependencyGraph{
			Nodes: map[string]*DependencyNode{
				"post": {
					ID:       "post",
					Type:     "resource",
					Name:     "Post",
					FilePath: "/app/resources/post.cdt",
				},
				"user": {
					ID:       "user",
					Type:     "resource",
					Name:     "User",
					FilePath: "/app/resources/user.cdt",
				},
			},
			Edges: []DependencyEdge{
				{
					From:         "post",
					To:           "user",
					Relationship: "belongs_to",
					Weight:       1,
				},
			},
		},
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal metadata: %v", err)
	}

	// Deserialize from JSON
	var decoded Metadata
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal metadata: %v", err)
	}

	// Verify core fields
	if decoded.Version != original.Version {
		t.Errorf("Version mismatch: got %s, want %s", decoded.Version, original.Version)
	}

	if !decoded.Generated.Equal(original.Generated) {
		t.Errorf("Generated timestamp mismatch: got %v, want %v", decoded.Generated, original.Generated)
	}

	if decoded.SourceHash != original.SourceHash {
		t.Errorf("SourceHash mismatch: got %s, want %s", decoded.SourceHash, original.SourceHash)
	}

	// Verify resources
	if len(decoded.Resources) != len(original.Resources) {
		t.Fatalf("Resources count mismatch: got %d, want %d", len(decoded.Resources), len(original.Resources))
	}

	// Verify routes
	if len(decoded.Routes) != len(original.Routes) {
		t.Fatalf("Routes count mismatch: got %d, want %d", len(decoded.Routes), len(original.Routes))
	}

	// Verify patterns
	if len(decoded.Patterns) != len(original.Patterns) {
		t.Fatalf("Patterns count mismatch: got %d, want %d", len(decoded.Patterns), len(original.Patterns))
	}

	// Verify dependency graph
	if len(decoded.Dependencies.Nodes) != len(original.Dependencies.Nodes) {
		t.Fatalf("Dependency nodes count mismatch: got %d, want %d",
			len(decoded.Dependencies.Nodes), len(original.Dependencies.Nodes))
	}
}

// TestResourceMetadataCompleteness tests that ResourceMetadata captures all required information.
func TestResourceMetadataCompleteness(t *testing.T) {
	resource := ResourceMetadata{
		Name:          "User",
		Documentation: "User account resource",
		FilePath:      "/app/resources/user.cdt",
		Fields: []FieldMetadata{
			{
				Name:          "id",
				Type:          "uuid",
				Nullable:      false,
				Required:      true,
				Constraints:   []string{"@primary", "@auto"},
				Documentation: "Primary key",
			},
			{
				Name:          "email",
				Type:          "string",
				Nullable:      false,
				Required:      true,
				Constraints:   []string{"@unique", "@email"},
				Documentation: "User email address",
			},
			{
				Name:          "bio",
				Type:          "text",
				Nullable:      true,
				Required:      false,
				DefaultValue:  "",
				Documentation: "Optional user bio",
			},
		},
		Relationships: []RelationshipMetadata{
			{
				Name:           "posts",
				Type:           "has_many",
				TargetResource: "Post",
				ForeignKey:     "author_id",
			},
		},
		Hooks: []HookMetadata{
			{
				Type:        "after_create",
				Transaction: true,
				Async:       true,
				SourceCode:  "Email.send_welcome(self.email)",
				LineNumber:  25,
			},
		},
		Validations: []ValidationMetadata{
			{
				Field:      "email",
				Type:       "email",
				Message:    "Invalid email format",
				LineNumber: 10,
			},
		},
		Constraints: []ConstraintMetadata{
			{
				Name:       "email_required_for_active",
				Operations: []string{"create", "update"},
				Condition:  "self.status == 'active'",
				When:       "String.length(self.email) > 0",
				Error:      "Active users must have an email",
				LineNumber: 30,
			},
		},
		Middleware: map[string][]string{
			"update": {"auth", "rate_limit"},
			"delete": {"auth", "admin"},
		},
		Scopes: []ScopeMetadata{
			{
				Name:       "active",
				Query:      "status == 'active'",
				LineNumber: 40,
			},
		},
		ComputedFields: []ComputedFieldMetadata{
			{
				Name:       "full_name",
				Type:       "string",
				Expression: "self.first_name + ' ' + self.last_name",
				LineNumber: 45,
			},
		},
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Failed to marshal resource: %v", err)
	}

	// Deserialize from JSON
	var decoded ResourceMetadata
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal resource: %v", err)
	}

	// Verify all sections are present
	if decoded.Name != resource.Name {
		t.Errorf("Name mismatch: got %s, want %s", decoded.Name, resource.Name)
	}

	if len(decoded.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(decoded.Fields))
	}

	if len(decoded.Relationships) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(decoded.Relationships))
	}

	if len(decoded.Hooks) != 1 {
		t.Errorf("Expected 1 hook, got %d", len(decoded.Hooks))
	}

	if len(decoded.Validations) != 1 {
		t.Errorf("Expected 1 validation, got %d", len(decoded.Validations))
	}

	if len(decoded.Constraints) != 1 {
		t.Errorf("Expected 1 constraint, got %d", len(decoded.Constraints))
	}

	if len(decoded.Middleware) != 2 {
		t.Errorf("Expected 2 middleware entries, got %d", len(decoded.Middleware))
	}

	if len(decoded.Scopes) != 1 {
		t.Errorf("Expected 1 scope, got %d", len(decoded.Scopes))
	}

	if len(decoded.ComputedFields) != 1 {
		t.Errorf("Expected 1 computed field, got %d", len(decoded.ComputedFields))
	}
}

// TestFieldMetadataTypes tests different field type scenarios.
func TestFieldMetadataTypes(t *testing.T) {
	tests := []struct {
		name     string
		field    FieldMetadata
		wantJSON string
	}{
		{
			name: "required_non_nullable",
			field: FieldMetadata{
				Name:     "title",
				Type:     "string",
				Nullable: false,
				Required: true,
			},
			wantJSON: `{"name":"title","type":"string","nullable":false,"required":true}`,
		},
		{
			name: "optional_nullable",
			field: FieldMetadata{
				Name:     "bio",
				Type:     "text",
				Nullable: true,
				Required: false,
			},
			wantJSON: `{"name":"bio","type":"text","nullable":true,"required":false}`,
		},
		{
			name: "with_default_value",
			field: FieldMetadata{
				Name:         "status",
				Type:         "string",
				Nullable:     false,
				Required:     true,
				DefaultValue: "draft",
			},
			wantJSON: `{"name":"status","type":"string","nullable":false,"required":true,"default_value":"draft"}`,
		},
		{
			name: "with_constraints",
			field: FieldMetadata{
				Name:        "age",
				Type:        "integer",
				Nullable:    false,
				Required:    true,
				Constraints: []string{"@min(0)", "@max(120)"},
			},
			wantJSON: `{"name":"age","type":"integer","nullable":false,"required":true,"constraints":["@min(0)","@max(120)"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.field)
			if err != nil {
				t.Fatalf("Failed to marshal field: %v", err)
			}

			if string(jsonData) != tt.wantJSON {
				t.Errorf("JSON mismatch:\ngot:  %s\nwant: %s", string(jsonData), tt.wantJSON)
			}

			// Verify round-trip
			var decoded FieldMetadata
			if err := json.Unmarshal(jsonData, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal field: %v", err)
			}

			if decoded.Name != tt.field.Name {
				t.Errorf("Name mismatch after round-trip: got %s, want %s", decoded.Name, tt.field.Name)
			}
		})
	}
}

// TestRelationshipTypes tests different relationship scenarios.
func TestRelationshipTypes(t *testing.T) {
	tests := []struct {
		name         string
		relationship RelationshipMetadata
	}{
		{
			name: "belongs_to",
			relationship: RelationshipMetadata{
				Name:           "author",
				Type:           "belongs_to",
				TargetResource: "User",
				ForeignKey:     "author_id",
				OnDelete:       "restrict",
			},
		},
		{
			name: "has_many",
			relationship: RelationshipMetadata{
				Name:           "posts",
				Type:           "has_many",
				TargetResource: "Post",
				ForeignKey:     "user_id",
			},
		},
		{
			name: "has_many_through",
			relationship: RelationshipMetadata{
				Name:           "followers",
				Type:           "has_many_through",
				TargetResource: "User",
				ThroughTable:   "follows",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.relationship)
			if err != nil {
				t.Fatalf("Failed to marshal relationship: %v", err)
			}

			var decoded RelationshipMetadata
			if err := json.Unmarshal(jsonData, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal relationship: %v", err)
			}

			if decoded.Name != tt.relationship.Name {
				t.Errorf("Name mismatch: got %s, want %s", decoded.Name, tt.relationship.Name)
			}

			if decoded.Type != tt.relationship.Type {
				t.Errorf("Type mismatch: got %s, want %s", decoded.Type, tt.relationship.Type)
			}

			if decoded.TargetResource != tt.relationship.TargetResource {
				t.Errorf("TargetResource mismatch: got %s, want %s",
					decoded.TargetResource, tt.relationship.TargetResource)
			}
		})
	}
}

// TestHookMetadata tests hook metadata serialization.
func TestHookMetadata(t *testing.T) {
	tests := []struct {
		name string
		hook HookMetadata
	}{
		{
			name: "sync_transactional",
			hook: HookMetadata{
				Type:        "before_create",
				Transaction: true,
				Async:       false,
				SourceCode:  "self.slug = String.slugify(self.title)",
				LineNumber:  10,
			},
		},
		{
			name: "async_non_transactional",
			hook: HookMetadata{
				Type:        "after_create",
				Transaction: false,
				Async:       true,
				SourceCode:  "Email.send_notification(self.email)",
				LineNumber:  20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.hook)
			if err != nil {
				t.Fatalf("Failed to marshal hook: %v", err)
			}

			var decoded HookMetadata
			if err := json.Unmarshal(jsonData, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal hook: %v", err)
			}

			if decoded.Type != tt.hook.Type {
				t.Errorf("Type mismatch: got %s, want %s", decoded.Type, tt.hook.Type)
			}

			if decoded.Transaction != tt.hook.Transaction {
				t.Errorf("Transaction mismatch: got %v, want %v", decoded.Transaction, tt.hook.Transaction)
			}

			if decoded.Async != tt.hook.Async {
				t.Errorf("Async mismatch: got %v, want %v", decoded.Async, tt.hook.Async)
			}
		})
	}
}

// TestPatternMetadata tests pattern discovery metadata.
func TestPatternMetadata(t *testing.T) {
	pattern := PatternMetadata{
		ID:          uuid.New().String(),
		Name:        "auth-middleware-chain",
		Category:    "middleware",
		Description: "Common authentication middleware pattern",
		Template:    "@on <operation>: [auth, rate_limit]",
		Examples: []PatternExample{
			{
				Resource:   "Post",
				FilePath:   "/app/resources/post.cdt",
				LineNumber: 15,
				Code:       "@on update: [auth, rate_limit]",
			},
			{
				Resource:   "User",
				FilePath:   "/app/resources/user.cdt",
				LineNumber: 20,
				Code:       "@on delete: [auth, admin]",
			},
		},
		Frequency:  5,
		Confidence: 0.87,
	}

	jsonData, err := json.Marshal(pattern)
	if err != nil {
		t.Fatalf("Failed to marshal pattern: %v", err)
	}

	var decoded PatternMetadata
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal pattern: %v", err)
	}

	if decoded.Name != pattern.Name {
		t.Errorf("Name mismatch: got %s, want %s", decoded.Name, pattern.Name)
	}

	if decoded.Category != pattern.Category {
		t.Errorf("Category mismatch: got %s, want %s", decoded.Category, pattern.Category)
	}

	if len(decoded.Examples) != len(pattern.Examples) {
		t.Errorf("Examples count mismatch: got %d, want %d", len(decoded.Examples), len(pattern.Examples))
	}

	if decoded.Frequency != pattern.Frequency {
		t.Errorf("Frequency mismatch: got %d, want %d", decoded.Frequency, pattern.Frequency)
	}

	if decoded.Confidence != pattern.Confidence {
		t.Errorf("Confidence mismatch: got %f, want %f", decoded.Confidence, pattern.Confidence)
	}
}

// TestDependencyGraph tests dependency graph serialization.
func TestDependencyGraph(t *testing.T) {
	graph := DependencyGraph{
		Nodes: map[string]*DependencyNode{
			"post": {
				ID:       "post",
				Type:     "resource",
				Name:     "Post",
				FilePath: "/app/resources/post.cdt",
			},
			"user": {
				ID:       "user",
				Type:     "resource",
				Name:     "User",
				FilePath: "/app/resources/user.cdt",
			},
			"comment": {
				ID:       "comment",
				Type:     "resource",
				Name:     "Comment",
				FilePath: "/app/resources/comment.cdt",
			},
		},
		Edges: []DependencyEdge{
			{
				From:         "post",
				To:           "user",
				Relationship: "belongs_to",
				Weight:       1,
			},
			{
				From:         "comment",
				To:           "post",
				Relationship: "belongs_to",
				Weight:       1,
			},
			{
				From:         "comment",
				To:           "user",
				Relationship: "belongs_to",
				Weight:       1,
			},
		},
	}

	jsonData, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("Failed to marshal graph: %v", err)
	}

	var decoded DependencyGraph
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal graph: %v", err)
	}

	if len(decoded.Nodes) != len(graph.Nodes) {
		t.Errorf("Nodes count mismatch: got %d, want %d", len(decoded.Nodes), len(graph.Nodes))
	}

	if len(decoded.Edges) != len(graph.Edges) {
		t.Errorf("Edges count mismatch: got %d, want %d", len(decoded.Edges), len(graph.Edges))
	}

	// Verify node details
	for id, node := range graph.Nodes {
		decodedNode, ok := decoded.Nodes[id]
		if !ok {
			t.Errorf("Node %s not found in decoded graph", id)
			continue
		}

		if decodedNode.Name != node.Name {
			t.Errorf("Node %s name mismatch: got %s, want %s", id, decodedNode.Name, node.Name)
		}
	}
}

// TestMetadataSizeCompression tests that metadata meets size requirements.
func TestMetadataSizeCompression(t *testing.T) {
	// Create a realistic resource with all metadata
	resource := ResourceMetadata{
		Name:          "Post",
		Documentation: "Blog post with title, content, and author relationship",
		FilePath:      "/app/resources/post.cdt",
		Fields: []FieldMetadata{
			{Name: "id", Type: "uuid", Nullable: false, Required: true, Constraints: []string{"@primary", "@auto"}},
			{Name: "title", Type: "string", Nullable: false, Required: true, Constraints: []string{"@min(5)", "@max(200)"}},
			{Name: "slug", Type: "string", Nullable: false, Required: true, Constraints: []string{"@unique"}},
			{Name: "content", Type: "text", Nullable: false, Required: true, Constraints: []string{"@min(100)"}},
			{Name: "status", Type: "string", Nullable: false, Required: true, DefaultValue: "draft"},
			{Name: "published_at", Type: "timestamp", Nullable: true, Required: false},
			{Name: "created_at", Type: "timestamp", Nullable: false, Required: true, Constraints: []string{"@auto"}},
			{Name: "updated_at", Type: "timestamp", Nullable: false, Required: true, Constraints: []string{"@auto"}},
		},
		Relationships: []RelationshipMetadata{
			{Name: "author", Type: "belongs_to", TargetResource: "User", ForeignKey: "author_id", OnDelete: "restrict"},
			{Name: "comments", Type: "has_many", TargetResource: "Comment", ForeignKey: "post_id"},
			{Name: "tags", Type: "has_many_through", TargetResource: "Tag", ThroughTable: "post_tags"},
		},
		Hooks: []HookMetadata{
			{Type: "before_create", Transaction: true, Async: false, SourceCode: "self.slug = String.slugify(self.title)", LineNumber: 15},
			{Type: "after_create", Transaction: false, Async: true, SourceCode: "Email.send_notification('new_post', self.id)", LineNumber: 20},
		},
		Validations: []ValidationMetadata{
			{Field: "title", Type: "min", Value: "5", LineNumber: 5},
			{Field: "title", Type: "max", Value: "200", LineNumber: 5},
			{Field: "content", Type: "min", Value: "100", LineNumber: 8},
		},
		Constraints: []ConstraintMetadata{
			{
				Name:       "published_requires_content",
				Operations: []string{"create", "update"},
				Condition:  "self.status == 'published'",
				When:       "String.length(self.content) >= 500",
				Error:      "Published posts need 500+ characters",
				LineNumber: 25,
			},
		},
		Middleware: map[string][]string{
			"create": {"auth"},
			"update": {"auth", "owner"},
			"delete": {"auth", "owner"},
		},
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Failed to marshal resource: %v", err)
	}

	uncompressedSize := len(jsonData)
	t.Logf("Uncompressed JSON size: %d bytes", uncompressedSize)

	// Compress with gzip
	var compressed bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressed)
	if _, err := gzipWriter.Write(jsonData); err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}
	gzipWriter.Close()

	compressedSize := compressed.Len()
	t.Logf("Compressed JSON size: %d bytes", compressedSize)

	// Verify size target (<2KB compressed)
	const maxCompressedSize = 2048 // 2KB
	if compressedSize > maxCompressedSize {
		t.Errorf("Compressed size %d bytes exceeds target of %d bytes", compressedSize, maxCompressedSize)
	}

	// Calculate compression ratio
	ratio := float64(compressedSize) / float64(uncompressedSize) * 100
	t.Logf("Compression ratio: %.1f%%", ratio)
}

// TestOmitEmptyFields tests that empty optional fields are omitted from JSON.
func TestOmitEmptyFields(t *testing.T) {
	// Field with minimal required data
	field := FieldMetadata{
		Name:     "id",
		Type:     "uuid",
		Nullable: false,
		Required: true,
	}

	jsonData, err := json.Marshal(field)
	if err != nil {
		t.Fatalf("Failed to marshal field: %v", err)
	}

	jsonStr := string(jsonData)

	// Verify omitempty fields are not present
	if contains(jsonStr, "default_value") {
		t.Error("default_value should be omitted when empty")
	}

	if contains(jsonStr, "constraints") {
		t.Error("constraints should be omitted when empty")
	}

	if contains(jsonStr, "documentation") {
		t.Error("documentation should be omitted when empty")
	}

	if contains(jsonStr, "tags") {
		t.Error("tags should be omitted when empty")
	}

	t.Logf("Minimal field JSON: %s", jsonStr)
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

// TestRouteMetadata tests route metadata serialization.
func TestRouteMetadata(t *testing.T) {
	route := RouteMetadata{
		Method:       "POST",
		Path:         "/posts",
		Handler:      "CreatePost",
		Resource:     "Post",
		Operation:    "create",
		Middleware:   []string{"auth", "rate_limit"},
		RequestBody:  "Post",
		ResponseBody: "Post",
	}

	jsonData, err := json.Marshal(route)
	if err != nil {
		t.Fatalf("Failed to marshal route: %v", err)
	}

	var decoded RouteMetadata
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal route: %v", err)
	}

	if decoded.Method != route.Method {
		t.Errorf("Method mismatch: got %s, want %s", decoded.Method, route.Method)
	}

	if decoded.Path != route.Path {
		t.Errorf("Path mismatch: got %s, want %s", decoded.Path, route.Path)
	}

	if decoded.Operation != route.Operation {
		t.Errorf("Operation mismatch: got %s, want %s", decoded.Operation, route.Operation)
	}
}
