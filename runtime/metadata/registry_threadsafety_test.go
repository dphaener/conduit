package metadata

import (
	"testing"
)

// TestQueryFunctionsReturnCopies verifies that query functions return defensive copies
// to prevent external mutation of shared data (CON-51 code review fix)
func TestQueryFunctionsReturnCopies(t *testing.T) {
	// Reset and setup test data
	Reset()

	testData := []byte(`{
		"version": "1.0.0",
		"resources": [
			{"name": "User", "table_name": "users", "fields": []},
			{"name": "Post", "table_name": "posts", "fields": []}
		],
		"patterns": [
			{"type": "crud", "resource": "User"}
		],
		"routes": [
			{"method": "GET", "path": "/users", "handler": "ListUsers"}
		]
	}`)

	if err := RegisterMetadata(testData); err != nil {
		t.Fatalf("Failed to register metadata: %v", err)
	}

	t.Run("QueryResources returns defensive copy", func(t *testing.T) {
		// Get resources twice
		resources1 := QueryResources()
		resources2 := QueryResources()

		// Verify we got data
		if len(resources1) != 2 {
			t.Fatalf("Expected 2 resources, got %d", len(resources1))
		}

		// Modify the first slice
		resources1[0].Name = "MODIFIED"

		// Verify the second slice is unaffected
		if resources2[0].Name == "MODIFIED" {
			t.Error("Modifying returned slice affected subsequent calls - not a defensive copy!")
		}

		// Verify original data is unaffected
		resources3 := QueryResources()
		if resources3[0].Name == "MODIFIED" {
			t.Error("Original metadata was modified - defensive copy not working!")
		}
	})

	t.Run("QueryPatterns returns defensive copy", func(t *testing.T) {
		patterns1 := QueryPatterns()
		patterns2 := QueryPatterns()

		if len(patterns1) != 1 {
			t.Fatalf("Expected 1 pattern, got %d", len(patterns1))
		}

		// Modify the first slice
		patterns1[0].Category = "MODIFIED"

		// Verify the second slice is unaffected
		if patterns2[0].Category == "MODIFIED" {
			t.Error("Modifying returned slice affected subsequent calls - not a defensive copy!")
		}
	})

	t.Run("QueryRoutes returns defensive copy", func(t *testing.T) {
		routes1 := QueryRoutes()
		routes2 := QueryRoutes()

		if len(routes1) != 1 {
			t.Fatalf("Expected 1 route, got %d", len(routes1))
		}

		// Modify the first slice
		routes1[0].Method = "MODIFIED"

		// Verify the second slice is unaffected
		if routes2[0].Method == "MODIFIED" {
			t.Error("Modifying returned slice affected subsequent calls - not a defensive copy!")
		}
	})

	// Cleanup
	Reset()
}
