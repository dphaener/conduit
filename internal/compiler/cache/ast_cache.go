package cache

import (
	"sync"
	"time"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// CachedAST represents a cached AST with metadata
type CachedAST struct {
	Program     *ast.Program
	Hash        string
	Path        string
	CachedAt    time.Time
	LastChecked time.Time
}

// ASTCache provides in-memory caching of parsed ASTs for watch mode
type ASTCache struct {
	entries map[string]*CachedAST
	mu      sync.RWMutex
}

// NewASTCache creates a new AST cache
func NewASTCache() *ASTCache {
	return &ASTCache{
		entries: make(map[string]*CachedAST),
	}
}

// Get retrieves a cached AST by file path
// Note: LastChecked is NOT updated here to avoid race conditions under read lock.
// LastChecked is only updated during Set() and can be checked during Prune().
func (ac *ASTCache) Get(path string) (*CachedAST, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	entry, exists := ac.entries[path]
	return entry, exists
}

// GetByHash retrieves a cached AST by content hash
// Note: LastChecked is NOT updated here to avoid race conditions under read lock.
// LastChecked is only updated during Set() and can be checked during Prune().
func (ac *ASTCache) GetByHash(hash string) (*CachedAST, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	for _, entry := range ac.entries {
		if entry.Hash == hash {
			return entry, true
		}
	}
	return nil, false
}

// Set stores an AST in the cache
func (ac *ASTCache) Set(path string, program *ast.Program, hash string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	now := time.Now()
	ac.entries[path] = &CachedAST{
		Program:     program,
		Hash:        hash,
		Path:        path,
		CachedAt:    now,
		LastChecked: now,
	}
}

// Invalidate removes an entry from the cache
func (ac *ASTCache) Invalidate(path string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	delete(ac.entries, path)
}

// InvalidateAll clears the entire cache
func (ac *ASTCache) InvalidateAll() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.entries = make(map[string]*CachedAST)
}

// Size returns the number of cached entries
func (ac *ASTCache) Size() int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	return len(ac.entries)
}

// GetAll returns all cached ASTs
func (ac *ASTCache) GetAll() map[string]*CachedAST {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	result := make(map[string]*CachedAST, len(ac.entries))
	for k, v := range ac.entries {
		result[k] = v
	}
	return result
}

// Prune removes entries that haven't been checked in the given duration
func (ac *ASTCache) Prune(maxAge time.Duration) int {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	now := time.Now()
	pruned := 0

	for path, entry := range ac.entries {
		if now.Sub(entry.LastChecked) > maxAge {
			delete(ac.entries, path)
			pruned++
		}
	}

	return pruned
}
