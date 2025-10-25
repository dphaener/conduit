package watch

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher monitors file system changes and triggers callbacks
type FileWatcher struct {
	watcher   *fsnotify.Watcher
	debouncer *Debouncer
	patterns  []string
	ignored   []string
	onChange  func([]string) error
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// NewFileWatcher creates a new file watcher instance
func NewFileWatcher(patterns, ignored []string, onChange func([]string) error) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	fw := &FileWatcher{
		watcher:   watcher,
		debouncer: NewDebouncer(100 * time.Millisecond),
		patterns:  patterns,
		ignored:   ignored,
		onChange:  onChange,
		stopChan:  make(chan struct{}),
	}

	// Set debouncer callback
	fw.debouncer.SetCallback(func(files []string) {
		if err := fw.onChange(files); err != nil {
			log.Printf("Error handling file changes: %v", err)
		}
	})

	return fw, nil
}

// Start begins watching the file system
func (fw *FileWatcher) Start() error {
	// Add directories to watch
	dirs, err := fw.findDirectories()
	if err != nil {
		return fmt.Errorf("failed to find directories: %w", err)
	}

	for _, dir := range dirs {
		if err := fw.watcher.Add(dir); err != nil {
			return fmt.Errorf("failed to watch directory %s: %w", dir, err)
		}
		log.Printf("[Watch] Watching directory: %s", dir)
	}

	// Start watching in background
	fw.wg.Add(1)
	go fw.watch()

	return nil
}

// Stop stops the file watcher
func (fw *FileWatcher) Stop() error {
	// Check if already stopped
	select {
	case <-fw.stopChan:
		// Already stopped
		return nil
	default:
		close(fw.stopChan)
	}

	fw.wg.Wait()
	fw.debouncer.Stop()
	return fw.watcher.Close()
}

// watch is the main event loop
func (fw *FileWatcher) watch() {
	defer fw.wg.Done()

	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Filter ignored files
			if fw.shouldIgnore(event.Name) {
				continue
			}

			// Only handle Write and Create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// Check if file matches patterns
				if fw.matchesPattern(event.Name) {
					log.Printf("[Watch] File changed: %s", event.Name)
					fw.debouncer.Add(event.Name)
				}
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[Watch] Error: %v", err)

		case <-fw.stopChan:
			return
		}
	}
}

// findDirectories discovers all directories to watch
func (fw *FileWatcher) findDirectories() ([]string, error) {
	dirs := make([]string, 0)

	// Common Conduit project directories
	candidates := []string{
		"app",
		"resources",
		"ui",
		"config",
		"public",
	}

	for _, dir := range candidates {
		// Check if directory exists
		if info, err := filepath.Glob(dir); err == nil && len(info) > 0 {
			dirs = append(dirs, dir)
		}
	}

	// Always include current directory
	dirs = append(dirs, ".")

	return dirs, nil
}

// shouldIgnore checks if a file path should be ignored
func (fw *FileWatcher) shouldIgnore(path string) bool {
	// Ignore build directory
	if strings.Contains(path, "build/") {
		return true
	}

	// Ignore hidden files and directories
	baseName := filepath.Base(path)
	if strings.HasPrefix(baseName, ".") {
		return true
	}

	// Check ignored patterns
	for _, pattern := range fw.ignored {
		if matched, _ := filepath.Match(pattern, baseName); matched {
			return true
		}
	}

	return false
}

// matchesPattern checks if a file matches any of the watch patterns
func (fw *FileWatcher) matchesPattern(path string) bool {
	// If no patterns specified, match all
	if len(fw.patterns) == 0 {
		return true
	}

	ext := filepath.Ext(path)
	for _, pattern := range fw.patterns {
		// Handle extension patterns
		if strings.HasPrefix(pattern, "*.") {
			if ext == pattern[1:] {
				return true
			}
		}

		// Handle glob patterns
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}

	return false
}

// Debouncer collects file changes and triggers callbacks after a delay
type Debouncer struct {
	duration time.Duration
	timer    *time.Timer
	files    map[string]struct{}
	mutex    sync.Mutex
	callback func([]string)
	stopChan chan struct{}
}

// NewDebouncer creates a new debouncer instance
func NewDebouncer(duration time.Duration) *Debouncer {
	return &Debouncer{
		duration: duration,
		files:    make(map[string]struct{}),
		stopChan: make(chan struct{}),
	}
}

// Add adds a file to the debouncer
func (d *Debouncer) Add(file string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.files[file] = struct{}{}

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.duration, func() {
		d.flush()
	})
}

// flush triggers the callback with accumulated files
func (d *Debouncer) flush() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if len(d.files) == 0 {
		return
	}

	files := make([]string, 0, len(d.files))
	for file := range d.files {
		files = append(files, file)
	}

	d.files = make(map[string]struct{})

	if d.callback != nil {
		d.callback(files)
	}
}

// SetCallback sets the callback function
func (d *Debouncer) SetCallback(callback func([]string)) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.callback = callback
}

// Stop stops the debouncer
func (d *Debouncer) Stop() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.timer != nil {
		d.timer.Stop()
	}

	// Check if already stopped
	select {
	case <-d.stopChan:
		// Already stopped
	default:
		close(d.stopChan)
	}
}
