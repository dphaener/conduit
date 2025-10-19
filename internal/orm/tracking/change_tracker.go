// Package tracking provides change tracking functionality for ORM resources.
// It tracks field modifications to enable efficient UPDATE queries and conditional hook logic.
package tracking

import (
	"reflect"
	"sync"
)

// FieldChange represents a change to a single field
type FieldChange struct {
	Field    string
	OldValue interface{}
	NewValue interface{}
}

// ChangeTracker tracks field changes on a resource instance
type ChangeTracker struct {
	mu       sync.RWMutex
	original map[string]interface{}
	current  map[string]interface{}
	changes  map[string]*FieldChange
}

// NewChangeTracker creates a new change tracker for a resource
// original: the initial state loaded from database
// current: the current state with modifications
func NewChangeTracker(original, current map[string]interface{}) *ChangeTracker {
	ct := &ChangeTracker{
		original: deepCopyMap(original),
		current:  deepCopyMap(current),
		changes:  make(map[string]*FieldChange),
	}
	ct.computeChanges()
	return ct
}

// deepCopyMap creates a deep copy of a map
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return make(map[string]interface{})
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = deepCopyValue(v)
	}
	return result
}

// deepCopyValue creates a deep copy of a value
func deepCopyValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Slice:
		// Deep copy slices
		slice := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			slice[i] = deepCopyValue(val.Index(i).Interface())
		}
		return slice
	case reflect.Map:
		// Deep copy maps
		m := make(map[interface{}]interface{})
		for _, key := range val.MapKeys() {
			m[deepCopyValue(key.Interface())] = deepCopyValue(val.MapIndex(key).Interface())
		}
		return m
	default:
		// For primitives, structs, pointers - return as-is
		// Structs are copied by value in Go
		return v
	}
}

// computeChanges calculates which fields have changed
func (ct *ChangeTracker) computeChanges() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Check all fields in current state
	for field, newValue := range ct.current {
		oldValue, hadOldValue := ct.original[field]

		// Skip internal tracking fields
		if field == "__changes__" {
			continue
		}

		// Field is changed if:
		// 1. It exists in current but not in original (new field)
		// 2. Values are not deeply equal
		if !hadOldValue || !deepEqual(oldValue, newValue) {
			ct.changes[field] = &FieldChange{
				Field:    field,
				OldValue: oldValue,
				NewValue: newValue,
			}
		}
	}

	// Check for deleted fields (in original but not in current)
	for field, oldValue := range ct.original {
		if field == "__changes__" {
			continue
		}
		if _, exists := ct.current[field]; !exists {
			ct.changes[field] = &FieldChange{
				Field:    field,
				OldValue: oldValue,
				NewValue: nil,
			}
		}
	}
}

// deepEqual compares two values for equality, handling nil and different types
func deepEqual(a, b interface{}) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Use reflect.DeepEqual for comprehensive comparison
	return reflect.DeepEqual(a, b)
}

// Changed returns true if the specified field has changed
func (ct *ChangeTracker) Changed(field string) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	_, ok := ct.changes[field]
	return ok
}

// ChangedFields returns a list of all fields that have changed
func (ct *ChangeTracker) ChangedFields() []string {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	fields := make([]string, 0, len(ct.changes))
	for field := range ct.changes {
		fields = append(fields, field)
	}
	return fields
}

// PreviousValue returns the previous value of a field
// Returns nil if the field didn't exist in the original state
func (ct *ChangeTracker) PreviousValue(field string) interface{} {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.original[field]
}

// CurrentValue returns the current value of a field
func (ct *ChangeTracker) CurrentValue(field string) interface{} {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.current[field]
}

// GetChange returns the FieldChange for a specific field, or nil if unchanged
func (ct *ChangeTracker) GetChange(field string) *FieldChange {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.changes[field]
}

// Changes returns a copy of all changes
func (ct *ChangeTracker) Changes() map[string]*FieldChange {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	result := make(map[string]*FieldChange, len(ct.changes))
	for k, v := range ct.changes {
		result[k] = v
	}
	return result
}

// HasChanges returns true if any fields have changed
func (ct *ChangeTracker) HasChanges() bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return len(ct.changes) > 0
}

// ChangedTo returns true if the field changed to the specified value
func (ct *ChangeTracker) ChangedTo(field string, value interface{}) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	change, ok := ct.changes[field]
	if !ok {
		return false
	}
	return deepEqual(change.NewValue, value)
}

// ChangedFrom returns true if the field changed from the specified value
func (ct *ChangeTracker) ChangedFrom(field string, value interface{}) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	change, ok := ct.changes[field]
	if !ok {
		return false
	}
	return deepEqual(change.OldValue, value)
}

// Reset clears all tracked changes and updates the original state
// This should be called after a successful save operation
func (ct *ChangeTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.original = deepCopyMap(ct.current)
	ct.changes = make(map[string]*FieldChange)
}

// SetFieldValue updates a field value and recomputes changes
func (ct *ChangeTracker) SetFieldValue(field string, value interface{}) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.current[field] = value

	// Recompute this field's change status
	oldValue, hadOldValue := ct.original[field]

	if field == "__changes__" {
		return
	}

	if !hadOldValue || !deepEqual(oldValue, value) {
		ct.changes[field] = &FieldChange{
			Field:    field,
			OldValue: oldValue,
			NewValue: value,
		}
	} else {
		// Value reverted to original, remove from changes
		delete(ct.changes, field)
	}
}

// GetChangedData returns a map of only the changed fields with their new values
// This is useful for generating efficient UPDATE queries
func (ct *ChangeTracker) GetChangedData() map[string]interface{} {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	result := make(map[string]interface{}, len(ct.changes))
	for field, change := range ct.changes {
		result[field] = change.NewValue
	}
	return result
}
