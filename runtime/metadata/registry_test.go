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
