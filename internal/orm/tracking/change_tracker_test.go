package tracking

import (
	"sync"
	"testing"
)

func TestNewChangeTracker(t *testing.T) {
	original := map[string]interface{}{
		"id":    1,
		"title": "Original Title",
		"count": 10,
	}

	current := map[string]interface{}{
		"id":    1,
		"title": "Updated Title",
		"count": 10,
	}

	ct := NewChangeTracker(original, current)

	if ct == nil {
		t.Fatal("NewChangeTracker returned nil")
	}

	// Should detect title as changed
	if !ct.Changed("title") {
		t.Error("Expected title to be changed")
	}

	// Should detect count as unchanged
	if ct.Changed("count") {
		t.Error("Expected count to be unchanged")
	}

	// Should detect id as unchanged
	if ct.Changed("id") {
		t.Error("Expected id to be unchanged")
	}
}

func TestChangeTracker_Changed(t *testing.T) {
	tests := []struct {
		name     string
		original map[string]interface{}
		current  map[string]interface{}
		field    string
		want     bool
	}{
		{
			name:     "unchanged field",
			original: map[string]interface{}{"field": "value"},
			current:  map[string]interface{}{"field": "value"},
			field:    "field",
			want:     false,
		},
		{
			name:     "changed string field",
			original: map[string]interface{}{"field": "old"},
			current:  map[string]interface{}{"field": "new"},
			field:    "field",
			want:     true,
		},
		{
			name:     "changed numeric field",
			original: map[string]interface{}{"count": 5},
			current:  map[string]interface{}{"count": 10},
			field:    "count",
			want:     true,
		},
		{
			name:     "nil to value",
			original: map[string]interface{}{"field": nil},
			current:  map[string]interface{}{"field": "value"},
			field:    "field",
			want:     true,
		},
		{
			name:     "value to nil",
			original: map[string]interface{}{"field": "value"},
			current:  map[string]interface{}{"field": nil},
			field:    "field",
			want:     true,
		},
		{
			name:     "nil to nil",
			original: map[string]interface{}{"field": nil},
			current:  map[string]interface{}{"field": nil},
			field:    "field",
			want:     false,
		},
		{
			name:     "new field added",
			original: map[string]interface{}{},
			current:  map[string]interface{}{"field": "value"},
			field:    "field",
			want:     true,
		},
		{
			name:     "field removed",
			original: map[string]interface{}{"field": "value"},
			current:  map[string]interface{}{},
			field:    "field",
			want:     true,
		},
		{
			name:     "nonexistent field",
			original: map[string]interface{}{"other": "value"},
			current:  map[string]interface{}{"other": "value"},
			field:    "field",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct := NewChangeTracker(tt.original, tt.current)
			got := ct.Changed(tt.field)
			if got != tt.want {
				t.Errorf("Changed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChangeTracker_ChangedFields(t *testing.T) {
	original := map[string]interface{}{
		"id":     1,
		"title":  "Original",
		"count":  5,
		"status": "active",
	}

	current := map[string]interface{}{
		"id":     1,
		"title":  "Updated",
		"count":  5,
		"status": "inactive",
	}

	ct := NewChangeTracker(original, current)
	changed := ct.ChangedFields()

	// Should have exactly 2 changed fields
	if len(changed) != 2 {
		t.Errorf("Expected 2 changed fields, got %d", len(changed))
	}

	// Check that title and status are in the changed list
	hasTitle := false
	hasStatus := false
	for _, field := range changed {
		if field == "title" {
			hasTitle = true
		}
		if field == "status" {
			hasStatus = true
		}
	}

	if !hasTitle {
		t.Error("Expected title in changed fields")
	}
	if !hasStatus {
		t.Error("Expected status in changed fields")
	}
}

func TestChangeTracker_PreviousValue(t *testing.T) {
	original := map[string]interface{}{
		"title": "Original Title",
		"count": 5,
	}

	current := map[string]interface{}{
		"title": "New Title",
		"count": 10,
	}

	ct := NewChangeTracker(original, current)

	// Test getting previous value
	prev := ct.PreviousValue("title")
	if prev != "Original Title" {
		t.Errorf("Expected 'Original Title', got %v", prev)
	}

	prev = ct.PreviousValue("count")
	if prev != 5 {
		t.Errorf("Expected 5, got %v", prev)
	}

	// Test nonexistent field
	prev = ct.PreviousValue("nonexistent")
	if prev != nil {
		t.Errorf("Expected nil for nonexistent field, got %v", prev)
	}
}

func TestChangeTracker_CurrentValue(t *testing.T) {
	original := map[string]interface{}{
		"title": "Original Title",
	}

	current := map[string]interface{}{
		"title": "New Title",
		"count": 10,
	}

	ct := NewChangeTracker(original, current)

	curr := ct.CurrentValue("title")
	if curr != "New Title" {
		t.Errorf("Expected 'New Title', got %v", curr)
	}

	curr = ct.CurrentValue("count")
	if curr != 10 {
		t.Errorf("Expected 10, got %v", curr)
	}
}

func TestChangeTracker_GetChange(t *testing.T) {
	original := map[string]interface{}{
		"title": "Original",
		"count": 5,
	}

	current := map[string]interface{}{
		"title": "Updated",
		"count": 5,
	}

	ct := NewChangeTracker(original, current)

	// Get change for changed field
	change := ct.GetChange("title")
	if change == nil {
		t.Fatal("Expected change for title, got nil")
	}
	if change.Field != "title" {
		t.Errorf("Expected field 'title', got %s", change.Field)
	}
	if change.OldValue != "Original" {
		t.Errorf("Expected old value 'Original', got %v", change.OldValue)
	}
	if change.NewValue != "Updated" {
		t.Errorf("Expected new value 'Updated', got %v", change.NewValue)
	}

	// Get change for unchanged field
	change = ct.GetChange("count")
	if change != nil {
		t.Errorf("Expected nil for unchanged field, got %v", change)
	}
}

func TestChangeTracker_Changes(t *testing.T) {
	original := map[string]interface{}{
		"id":    1,
		"title": "Original",
		"count": 5,
	}

	current := map[string]interface{}{
		"id":    1,
		"title": "Updated",
		"count": 10,
	}

	ct := NewChangeTracker(original, current)
	changes := ct.Changes()

	// Should have 2 changes (title and count)
	if len(changes) != 2 {
		t.Errorf("Expected 2 changes, got %d", len(changes))
	}

	// Verify title change
	if titleChange, ok := changes["title"]; ok {
		if titleChange.OldValue != "Original" {
			t.Errorf("Expected old value 'Original', got %v", titleChange.OldValue)
		}
		if titleChange.NewValue != "Updated" {
			t.Errorf("Expected new value 'Updated', got %v", titleChange.NewValue)
		}
	} else {
		t.Error("Expected title in changes")
	}

	// Verify count change
	if countChange, ok := changes["count"]; ok {
		if countChange.OldValue != 5 {
			t.Errorf("Expected old value 5, got %v", countChange.OldValue)
		}
		if countChange.NewValue != 10 {
			t.Errorf("Expected new value 10, got %v", countChange.NewValue)
		}
	} else {
		t.Error("Expected count in changes")
	}
}

func TestChangeTracker_HasChanges(t *testing.T) {
	tests := []struct {
		name     string
		original map[string]interface{}
		current  map[string]interface{}
		want     bool
	}{
		{
			name:     "has changes",
			original: map[string]interface{}{"field": "old"},
			current:  map[string]interface{}{"field": "new"},
			want:     true,
		},
		{
			name:     "no changes",
			original: map[string]interface{}{"field": "value"},
			current:  map[string]interface{}{"field": "value"},
			want:     false,
		},
		{
			name:     "empty maps",
			original: map[string]interface{}{},
			current:  map[string]interface{}{},
			want:     false,
		},
		{
			name:     "field added",
			original: map[string]interface{}{},
			current:  map[string]interface{}{"field": "value"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct := NewChangeTracker(tt.original, tt.current)
			if got := ct.HasChanges(); got != tt.want {
				t.Errorf("HasChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChangeTracker_ChangedTo(t *testing.T) {
	original := map[string]interface{}{
		"status": "draft",
		"count":  5,
	}

	current := map[string]interface{}{
		"status": "published",
		"count":  5,
	}

	ct := NewChangeTracker(original, current)

	// Test changed to specific value
	if !ct.ChangedTo("status", "published") {
		t.Error("Expected status changed to 'published'")
	}

	if ct.ChangedTo("status", "draft") {
		t.Error("status did not change to 'draft'")
	}

	// Test unchanged field
	if ct.ChangedTo("count", 5) {
		t.Error("count did not change")
	}

	// Test nonexistent field
	if ct.ChangedTo("nonexistent", "value") {
		t.Error("nonexistent field should not be changed")
	}
}

func TestChangeTracker_ChangedFrom(t *testing.T) {
	original := map[string]interface{}{
		"status": "draft",
		"count":  5,
	}

	current := map[string]interface{}{
		"status": "published",
		"count":  5,
	}

	ct := NewChangeTracker(original, current)

	// Test changed from specific value
	if !ct.ChangedFrom("status", "draft") {
		t.Error("Expected status changed from 'draft'")
	}

	if ct.ChangedFrom("status", "published") {
		t.Error("status did not change from 'published'")
	}

	// Test unchanged field
	if ct.ChangedFrom("count", 5) {
		t.Error("count did not change")
	}
}

func TestChangeTracker_Reset(t *testing.T) {
	original := map[string]interface{}{
		"title": "Original",
	}

	current := map[string]interface{}{
		"title": "Updated",
	}

	ct := NewChangeTracker(original, current)

	// Should have changes initially
	if !ct.HasChanges() {
		t.Error("Expected changes before reset")
	}

	// Reset
	ct.Reset()

	// Should have no changes after reset
	if ct.HasChanges() {
		t.Error("Expected no changes after reset")
	}

	// Original should be updated to current
	if ct.PreviousValue("title") != "Updated" {
		t.Error("Expected original to be updated after reset")
	}
}

func TestChangeTracker_SetFieldValue(t *testing.T) {
	original := map[string]interface{}{
		"title": "Original",
		"count": 5,
	}

	current := map[string]interface{}{
		"title": "Original",
		"count": 5,
	}

	ct := NewChangeTracker(original, current)

	// No changes initially
	if ct.HasChanges() {
		t.Error("Expected no changes initially")
	}

	// Set a new value
	ct.SetFieldValue("title", "Updated")

	// Should now have changes
	if !ct.HasChanges() {
		t.Error("Expected changes after SetFieldValue")
	}

	if !ct.Changed("title") {
		t.Error("Expected title to be changed")
	}

	// Revert to original value
	ct.SetFieldValue("title", "Original")

	// Should no longer have changes
	if ct.HasChanges() {
		t.Error("Expected no changes after reverting")
	}
}

func TestChangeTracker_GetChangedData(t *testing.T) {
	original := map[string]interface{}{
		"id":     1,
		"title":  "Original",
		"count":  5,
		"status": "draft",
	}

	current := map[string]interface{}{
		"id":     1,
		"title":  "Updated",
		"count":  10,
		"status": "draft",
	}

	ct := NewChangeTracker(original, current)
	changedData := ct.GetChangedData()

	// Should only include changed fields
	if len(changedData) != 2 {
		t.Errorf("Expected 2 changed fields, got %d", len(changedData))
	}

	if changedData["title"] != "Updated" {
		t.Errorf("Expected title='Updated', got %v", changedData["title"])
	}

	if changedData["count"] != 10 {
		t.Errorf("Expected count=10, got %v", changedData["count"])
	}

	// Unchanged fields should not be included
	if _, ok := changedData["id"]; ok {
		t.Error("id should not be in changed data")
	}
	if _, ok := changedData["status"]; ok {
		t.Error("status should not be in changed data")
	}
}

func TestChangeTracker_InternalFieldsIgnored(t *testing.T) {
	original := map[string]interface{}{
		"title":       "Original",
		"__changes__": "internal",
	}

	current := map[string]interface{}{
		"title":       "Updated",
		"__changes__": "different",
	}

	ct := NewChangeTracker(original, current)

	// __changes__ should be ignored
	if ct.Changed("__changes__") {
		t.Error("Internal __changes__ field should be ignored")
	}

	// Should only have title as changed
	changed := ct.ChangedFields()
	if len(changed) != 1 || changed[0] != "title" {
		t.Errorf("Expected only 'title' changed, got %v", changed)
	}
}

func TestChangeTracker_ConcurrentAccess(t *testing.T) {
	original := map[string]interface{}{
		"count": 0,
	}

	current := map[string]interface{}{
		"count": 0,
	}

	ct := NewChangeTracker(original, current)

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ct.Changed("count")
			_ = ct.ChangedFields()
			_ = ct.PreviousValue("count")
			_ = ct.CurrentValue("count")
			_ = ct.HasChanges()
		}()
	}

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		value := i
		go func() {
			defer wg.Done()
			ct.SetFieldValue("count", value)
		}()
	}

	wg.Wait()

	// Test should complete without race conditions
	// (run with -race flag to detect data races)
}

func TestChangeTracker_ComplexTypes(t *testing.T) {
	type Address struct {
		Street string
		City   string
	}

	original := map[string]interface{}{
		"address": Address{Street: "123 Main", City: "NYC"},
		"tags":    []string{"go", "test"},
	}

	current := map[string]interface{}{
		"address": Address{Street: "456 Oak", City: "NYC"},
		"tags":    []string{"go", "test"},
	}

	ct := NewChangeTracker(original, current)

	// Address changed
	if !ct.Changed("address") {
		t.Error("Expected address to be changed")
	}

	// Tags unchanged (same slice contents)
	if ct.Changed("tags") {
		t.Error("Expected tags to be unchanged")
	}
}

func TestChangeTracker_NilMaps(t *testing.T) {
	ct := NewChangeTracker(nil, nil)

	if ct == nil {
		t.Fatal("Expected non-nil ChangeTracker")
	}

	if ct.HasChanges() {
		t.Error("Expected no changes with nil maps")
	}

	// Should not panic
	_ = ct.Changed("field")
	_ = ct.ChangedFields()
	_ = ct.PreviousValue("field")
}

func TestChangeTracker_DeepCopyNestedStructures(t *testing.T) {
	// Test that mutations to nested slices/maps don't affect the original state
	original := map[string]interface{}{
		"tags": []string{"go", "rust"},
		"metadata": map[string]interface{}{
			"author": "test",
			"version": 1,
		},
	}

	current := map[string]interface{}{
		"tags": []string{"go", "rust"},
		"metadata": map[string]interface{}{
			"author": "test",
			"version": 1,
		},
	}

	ct := NewChangeTracker(original, current)

	// Initially no changes
	if ct.HasChanges() {
		t.Error("Expected no changes initially")
	}

	// Mutate the slice in current
	current["tags"].([]string)[0] = "python"

	// The mutation should NOT affect the tracker's internal state
	// because we made a deep copy
	if ct.Changed("tags") {
		t.Error("Expected tags to remain unchanged after external mutation")
	}

	// Verify the original value is still preserved
	prevTags := ct.PreviousValue("tags")
	if prevTags == nil {
		t.Fatal("Expected previous tags value to exist")
	}

	// The previous value should still be the original slice
	// (converted to []interface{} by deepCopyValue)
	prevSlice, ok := prevTags.([]interface{})
	if !ok {
		t.Fatalf("Expected []interface{}, got %T", prevTags)
	}

	if len(prevSlice) != 2 {
		t.Errorf("Expected 2 elements in previous tags, got %d", len(prevSlice))
	}

	if prevSlice[0] != "go" {
		t.Errorf("Expected first tag to be 'go', got %v", prevSlice[0])
	}

	// Mutate the map in current
	current["metadata"].(map[string]interface{})["author"] = "hacker"

	// The mutation should NOT affect the tracker's internal state
	if ct.Changed("metadata") {
		t.Error("Expected metadata to remain unchanged after external mutation")
	}

	// Verify the original metadata is still preserved
	prevMetadata := ct.PreviousValue("metadata")
	if prevMetadata == nil {
		t.Fatal("Expected previous metadata value to exist")
	}

	prevMap, ok := prevMetadata.(map[interface{}]interface{})
	if !ok {
		t.Fatalf("Expected map[interface{}]interface{}, got %T", prevMetadata)
	}

	if prevMap["author"] != "test" {
		t.Errorf("Expected author to be 'test', got %v", prevMap["author"])
	}
}

func TestChangeTracker_DeepCopyOnReset(t *testing.T) {
	// Test that Reset() also performs deep copy
	original := map[string]interface{}{
		"tags": []string{"go"},
	}

	current := map[string]interface{}{
		"tags": []string{"go", "rust"},
	}

	ct := NewChangeTracker(original, current)

	// Should have changes initially
	if !ct.Changed("tags") {
		t.Error("Expected tags to be changed")
	}

	// Reset the tracker
	ct.Reset()

	// After reset, should have no changes
	if ct.HasChanges() {
		t.Error("Expected no changes after reset")
	}

	// Mutate the current map's slice
	current["tags"].([]string)[0] = "python"

	// The mutation should NOT affect the tracker's internal original state
	// because Reset() did a deep copy
	prevTags := ct.PreviousValue("tags")
	if prevTags == nil {
		t.Fatal("Expected previous tags value to exist")
	}

	prevSlice, ok := prevTags.([]interface{})
	if !ok {
		t.Fatalf("Expected []interface{}, got %T", prevTags)
	}

	if len(prevSlice) < 1 {
		t.Fatal("Expected at least 1 element in previous tags")
	}

	// Should still be "go", not "python"
	if prevSlice[0] != "go" {
		t.Errorf("Expected first tag to be 'go', got %v", prevSlice[0])
	}
}
