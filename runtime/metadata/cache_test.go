package metadata

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestLRUCache_BasicOperations(t *testing.T) {
	cache := newLRUCache()

	// Test set and get
	cache.set("key1", "value1")
	val, ok := cache.get("key1")
	if !ok {
		t.Error("Expected cache hit")
	}
	if val != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}

	// Test miss
	_, ok = cache.get("key2")
	if ok {
		t.Error("Expected cache miss")
	}
}

func TestLRUCache_UpdateExisting(t *testing.T) {
	cache := newLRUCache()

	// Set initial value
	cache.set("key1", "value1")

	// Update value
	cache.set("key1", "value2")

	val, ok := cache.get("key1")
	if !ok {
		t.Error("Expected cache hit")
	}
	if val != "value2" {
		t.Errorf("Expected 'value2', got %v", val)
	}
}

func TestLRUCache_MaxEntries(t *testing.T) {
	cache := newLRUCache()
	cache.maxEntries = 3 // Override for testing

	// Add 4 entries
	cache.set("key1", "value1")
	cache.set("key2", "value2")
	cache.set("key3", "value3")
	cache.set("key4", "value4")

	// key1 should be evicted (LRU)
	_, ok := cache.get("key1")
	if ok {
		t.Error("Expected key1 to be evicted")
	}

	// Other keys should still exist
	if _, ok := cache.get("key2"); !ok {
		t.Error("Expected key2 to exist")
	}
	if _, ok := cache.get("key3"); !ok {
		t.Error("Expected key3 to exist")
	}
	if _, ok := cache.get("key4"); !ok {
		t.Error("Expected key4 to exist")
	}
}

func TestLRUCache_LRUOrdering(t *testing.T) {
	cache := newLRUCache()
	cache.maxEntries = 3

	// Add 3 entries
	cache.set("key1", "value1")
	cache.set("key2", "value2")
	cache.set("key3", "value3")

	// Access key1 (moves it to front)
	cache.get("key1")

	// Add key4 (should evict key2, not key1)
	cache.set("key4", "value4")

	// key1 should still exist (was accessed recently)
	if _, ok := cache.get("key1"); !ok {
		t.Error("Expected key1 to exist (was accessed recently)")
	}

	// key2 should be evicted
	if _, ok := cache.get("key2"); ok {
		t.Error("Expected key2 to be evicted")
	}

	// key3 and key4 should exist
	if _, ok := cache.get("key3"); !ok {
		t.Error("Expected key3 to exist")
	}
	if _, ok := cache.get("key4"); !ok {
		t.Error("Expected key4 to exist")
	}
}

func TestLRUCache_MemoryLimit(t *testing.T) {
	cache := newLRUCache()
	cache.maxMemory = 2000  // 2KB limit
	cache.maxEntries = 100  // High enough to not be the limiting factor

	// Add multiple small items until we exceed limit
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key%d", i)
		// Each string value is estimated at 1KB by default
		cache.set(key, fmt.Sprintf("value%d", i))
	}

	// First entries should be evicted due to memory limit
	if _, ok := cache.get("key0"); ok {
		t.Error("Expected key0 to be evicted due to memory limit")
	}

	// Latest entries should still exist
	if _, ok := cache.get("key4"); !ok {
		t.Error("Expected key4 to exist")
	}

	// Verify cache size is within limit
	if cache.currentSize > cache.maxMemory {
		t.Errorf("Cache size %d exceeds limit %d", cache.currentSize, cache.maxMemory)
	}

	// Verify we have at most 2 entries (2KB limit / 1KB per entry)
	if len(cache.entries) > 2 {
		t.Errorf("Expected at most 2 entries with 2KB limit, got %d", len(cache.entries))
	}
}

func TestLRUCache_Stats(t *testing.T) {
	cache := newLRUCache()

	cache.set("key1", "value1")

	// Hit
	cache.get("key1")
	// Miss
	cache.get("key2")
	// Hit
	cache.get("key1")
	// Miss
	cache.get("key3")

	hits, misses, hitRate := cache.stats()

	if hits != 2 {
		t.Errorf("Expected 2 hits, got %d", hits)
	}
	if misses != 2 {
		t.Errorf("Expected 2 misses, got %d", misses)
	}
	if hitRate != 0.5 {
		t.Errorf("Expected hit rate 0.5, got %f", hitRate)
	}
}

func TestLRUCache_Clear(t *testing.T) {
	cache := newLRUCache()

	cache.set("key1", "value1")
	cache.set("key2", "value2")
	cache.get("key1")

	cache.clear()

	// All entries should be gone
	if _, ok := cache.get("key1"); ok {
		t.Error("Expected cache to be empty after clear")
	}

	// Stats should be reset
	hits, missesAfterClear, _ := cache.stats()
	if hits != 0 || missesAfterClear != 1 { // The get above counts as a miss
		t.Errorf("Expected stats to be reset, got hits=%d, misses=%d", hits, missesAfterClear)
	}

	// Size should be 0
	if cache.currentSize != 0 {
		t.Errorf("Expected size 0, got %d", cache.currentSize)
	}
}

func TestLRUCache_EstimateSize(t *testing.T) {
	// Test DependencyGraph estimation
	graph := &DependencyGraph{
		Nodes: map[string]*DependencyNode{
			"A": {ID: "A"},
			"B": {ID: "B"},
		},
		Edges: []DependencyEdge{
			{From: "A", To: "B"},
		},
	}

	size := estimateSize(graph)
	if size <= 0 {
		t.Error("Expected positive size for DependencyGraph")
	}

	// Test ResourceMetadata slice
	resources := []ResourceMetadata{
		{Name: "Post"},
		{Name: "User"},
	}
	size = estimateSize(resources)
	if size != 1000 { // 2 * 500
		t.Errorf("Expected size 1000 for 2 resources, got %d", size)
	}

	// Test default estimation
	size = estimateSize("some string")
	if size != 1024 {
		t.Errorf("Expected default size 1024, got %d", size)
	}
}

func TestGetCacheStats(t *testing.T) {
	defer Reset()

	// Register some metadata
	meta := &Metadata{
		Version: "1.0.0",
		Resources: []ResourceMetadata{
			{Name: "Post"},
		},
	}
	data, _ := json.Marshal(meta)
	RegisterMetadata(data)

	// Make some queries to populate cache
	QueryResourcesByPattern("Post*")
	QueryResourcesByPattern("Post*") // Cache hit

	hits, _, hitRate := GetCacheStats()

	// Should have at least 1 hit
	if hits < 1 {
		t.Errorf("Expected at least 1 hit, got %d", hits)
	}

	// Hit rate should be > 0
	if hitRate <= 0 {
		t.Errorf("Expected positive hit rate, got %f", hitRate)
	}
}
