package watch

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestFileWatcher_Start(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "watch-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.cdt")
	if err := os.WriteFile(testFile, []byte("initial content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Track changes
	var mu sync.Mutex
	var changes [][]string

	// Create watcher
	watcher, err := NewFileWatcher(
		[]string{"*.cdt"},
		[]string{},
		func(files []string) error {
			mu.Lock()
			defer mu.Unlock()
			changes = append(changes, files)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	// Override directory finder to watch temp dir
	watcher.watcher.Add(tmpDir)

	// Start watching
	if err := watcher.Start(); err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	// Modify file
	time.Sleep(200 * time.Millisecond) // Allow watcher to initialize
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	// Wait for debounce
	time.Sleep(300 * time.Millisecond)

	// Verify changes were detected
	mu.Lock()
	defer mu.Unlock()

	if len(changes) == 0 {
		t.Error("Expected changes to be detected")
	}
}

func TestDebouncer_Add(t *testing.T) {
	var mu sync.Mutex
	var called bool
	var files []string

	debouncer := NewDebouncer(50 * time.Millisecond)
	debouncer.SetCallback(func(f []string) {
		mu.Lock()
		defer mu.Unlock()
		called = true
		files = f
	})

	// Add multiple files
	debouncer.Add("file1.cdt")
	debouncer.Add("file2.cdt")
	debouncer.Add("file1.cdt") // Duplicate

	// Wait for debounce
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !called {
		t.Error("Expected callback to be called")
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 unique files, got %d", len(files))
	}
}

func TestDebouncer_MultipleFlushes(t *testing.T) {
	var mu sync.Mutex
	var callCount int

	debouncer := NewDebouncer(30 * time.Millisecond)
	debouncer.SetCallback(func(f []string) {
		mu.Lock()
		defer mu.Unlock()
		callCount++
	})

	// First batch
	debouncer.Add("file1.cdt")
	time.Sleep(50 * time.Millisecond)

	// Second batch
	debouncer.Add("file2.cdt")
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if callCount != 2 {
		t.Errorf("Expected 2 callback calls, got %d", callCount)
	}
}

func TestFileWatcher_ShouldIgnore(t *testing.T) {
	watcher := &FileWatcher{
		ignored: []string{"*.swp", ".DS_Store"},
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"test.cdt", false},
		{"test.swp", true},
		{".DS_Store", true},
		{"build/test.cdt", true}, // build directory
		{".hidden", true},        // hidden file
		{"normal.go", false},
	}

	for _, tt := range tests {
		result := watcher.shouldIgnore(tt.path)
		if result != tt.expected {
			t.Errorf("shouldIgnore(%q) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}

func TestFileWatcher_MatchesPattern(t *testing.T) {
	tests := []struct {
		patterns []string
		path     string
		expected bool
	}{
		{[]string{"*.cdt"}, "test.cdt", true},
		{[]string{"*.cdt"}, "test.go", false},
		{[]string{"*.cdt", "*.css"}, "style.css", true},
		{[]string{}, "anything.txt", true}, // No patterns = match all
	}

	for _, tt := range tests {
		watcher := &FileWatcher{patterns: tt.patterns}
		result := watcher.matchesPattern(tt.path)
		if result != tt.expected {
			t.Errorf("matchesPattern(%v, %q) = %v, expected %v",
				tt.patterns, tt.path, result, tt.expected)
		}
	}
}

func TestFileWatcher_Stop(t *testing.T) {
	watcher, err := NewFileWatcher(
		[]string{"*.cdt"},
		[]string{},
		func(files []string) error { return nil },
	)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := watcher.Start(); err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	// Stop should not error
	if err := watcher.Stop(); err != nil {
		t.Errorf("Stop() returned error: %v", err)
	}

	// Second stop should not panic
	if err := watcher.Stop(); err == nil {
		// It's okay if it errors, just shouldn't panic
	}
}

func BenchmarkDebouncer_Add(b *testing.B) {
	debouncer := NewDebouncer(100 * time.Millisecond)
	debouncer.SetCallback(func(files []string) {})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		debouncer.Add("file.cdt")
	}
}
