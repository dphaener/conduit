package build

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// CacheEntry represents a cached compilation result
type CacheEntry struct {
	FilePath     string
	FileHash     string
	Compiled     *CompiledFile
	Dependencies []string
	CachedAt     time.Time
}

// Cache manages the build cache
type Cache struct {
	cacheDir string
	entries  map[string]*CacheEntry
	mu       sync.RWMutex
}

// NewCache creates a new build cache
func NewCache(cacheDir string) (*Cache, error) {
	// Create cache directory
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := &Cache{
		cacheDir: cacheDir,
		entries:  make(map[string]*CacheEntry),
	}

	// Load existing cache
	if err := cache.load(); err != nil {
		// Non-fatal - just start with empty cache
		// Log error in production
	}

	return cache, nil
}

// Get retrieves a compiled file from cache
func (c *Cache) Get(filePath string) (*CompiledFile, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[filePath]
	if !ok {
		return nil, fmt.Errorf("file not in cache")
	}

	// Verify file hasn't changed
	currentHash, err := computeFileHash(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute file hash: %w", err)
	}

	if currentHash != entry.FileHash {
		return nil, fmt.Errorf("file has changed")
	}

	// Check dependencies
	for _, dep := range entry.Dependencies {
		// Verify dependency file exists
		if _, err := os.Stat(dep); err != nil {
			return nil, fmt.Errorf("dependency missing: %s", dep)
		}

		depHash, err := computeFileHash(dep)
		if err != nil {
			// Dependency might have been deleted
			return nil, fmt.Errorf("dependency changed or missing")
		}

		// Find dependency in cache
		depEntry, ok := c.entries[dep]
		if !ok || depEntry.FileHash != depHash {
			return nil, fmt.Errorf("dependency has changed")
		}
	}

	return entry.Compiled, nil
}

// Put stores a compiled file in the cache
func (c *Cache) Put(filePath string, compiled *CompiledFile) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[filePath] = &CacheEntry{
		FilePath:     filePath,
		FileHash:     compiled.Hash,
		Compiled:     compiled,
		Dependencies: []string{}, // Dependencies will be set by dependency graph
		CachedAt:     time.Now(),
	}

	// Persist cache asynchronously
	go c.persist()

	return nil
}

// Invalidate removes a file from the cache
func (c *Cache) Invalidate(filePath string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, filePath)
}

// Clear removes all entries from the cache
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)

	// Remove cache files
	if err := os.RemoveAll(c.cacheDir); err != nil {
		return fmt.Errorf("failed to clear cache directory: %w", err)
	}

	// Recreate cache directory
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to recreate cache directory: %w", err)
	}

	return nil
}

// Stats returns cache statistics
func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CacheStats{
		TotalEntries: len(c.entries),
	}

	// Calculate total size
	for range c.entries {
		// Rough estimate: AST size + metadata
		stats.TotalSize += 1024 // Simplified
	}

	return stats
}

// CacheStats contains cache statistics
type CacheStats struct {
	TotalEntries int
	TotalSize    int64
	HitRate      float64
}

// load loads the cache from disk
func (c *Cache) load() error {
	indexPath := filepath.Join(c.cacheDir, "index.gob")

	file, err := os.Open(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache to load
		}
		return fmt.Errorf("failed to open cache index: %w", err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&c.entries); err != nil {
		return fmt.Errorf("failed to decode cache: %w", err)
	}

	return nil
}

// persist saves the cache to disk
func (c *Cache) persist() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Ensure cache directory exists
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	indexPath := filepath.Join(c.cacheDir, "index.gob")

	// Create temporary file
	tmpPath := indexPath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp cache file: %w", err)
	}

	// Encode cache
	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(c.entries); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to encode cache: %w", err)
	}

	file.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, indexPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save cache: %w", err)
	}

	return nil
}

// init registers types with gob
func init() {
	gob.Register(&ast.Program{})
	gob.Register(&ast.ResourceNode{})
	gob.Register(&CompiledFile{})
	gob.Register(&CacheEntry{})
}
