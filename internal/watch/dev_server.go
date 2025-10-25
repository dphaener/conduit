package watch

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/conduit-lang/conduit/compiler/errors"
)

//go:embed assets/reload.js
var reloadScript string

// DevServer manages the development server with hot reload
type DevServer struct {
	// Components
	watcher          *FileWatcher
	compiler         *IncrementalCompiler
	reloadServer     *ReloadServer
	assetWatcher     *AssetWatcher
	appProcess       *os.Process
	httpServer       *http.Server

	// Configuration
	port           int
	appPort        int
	watchPatterns  []string
	ignorePatterns []string

	// State
	isBuilding     bool
	buildMutex     sync.Mutex
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// DevServerConfig holds configuration for the dev server
type DevServerConfig struct {
	Port           int
	AppPort        int
	WatchPatterns  []string
	IgnorePatterns []string
}

// NewDevServer creates a new development server
func NewDevServer(config *DevServerConfig) (*DevServer, error) {
	if config == nil {
		config = &DevServerConfig{
			Port:    3000,
			AppPort: 3001,
			WatchPatterns: []string{
				"*.cdt",
				"*.css",
				"*.js",
				"*.html",
			},
			IgnorePatterns: []string{
				"*.swp",
				"*.swo",
				"*~",
				".DS_Store",
			},
		}
	}

	ds := &DevServer{
		compiler:       NewIncrementalCompiler(),
		reloadServer:   NewReloadServer(),
		port:           config.Port,
		appPort:        config.AppPort,
		watchPatterns:  config.WatchPatterns,
		ignorePatterns: config.IgnorePatterns,
		stopChan:       make(chan struct{}),
	}

	ds.assetWatcher = NewAssetWatcher(ds.reloadServer)

	// Create file watcher
	var err error
	ds.watcher, err = NewFileWatcher(ds.watchPatterns, ds.ignorePatterns, ds.handleFileChange)
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return ds, nil
}

// Start starts the development server
func (ds *DevServer) Start() error {
	log.Printf("[DevServer] Starting development server on port %d", ds.port)

	// Initial build
	log.Printf("[DevServer] Performing initial build...")
	result, err := ds.compiler.FullBuild()
	if err != nil {
		log.Printf("[DevServer] Initial build failed: %v", err)
		ds.displayErrors(result.Errors)
		// Continue anyway to allow watching for fixes
	} else {
		log.Printf("[DevServer] Initial build successful (%.2fs)", result.Duration.Seconds())

		// Build Go binary
		if err := ds.buildBinary(); err != nil {
			log.Printf("[DevServer] Failed to build binary: %v", err)
		} else {
			// Start app server
			if err := ds.startAppServer(); err != nil {
				log.Printf("[DevServer] Failed to start app server: %v", err)
			}
		}
	}

	// Start file watcher
	if err := ds.watcher.Start(); err != nil {
		return fmt.Errorf("failed to start file watcher: %w", err)
	}

	// Start HTTP server for reload WebSocket
	if err := ds.startHTTPServer(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	log.Printf("[DevServer] Development server ready!")
	log.Printf("[DevServer] - Reload server: http://localhost:%d", ds.port)
	log.Printf("[DevServer] - Application: http://localhost:%d", ds.appPort)
	log.Printf("[DevServer] Watching for changes...")

	return nil
}

// Stop stops the development server
func (ds *DevServer) Stop() error {
	log.Printf("[DevServer] Stopping development server...")

	// Ensure app server is stopped first to prevent orphan process
	if err := ds.stopAppServer(); err != nil {
		log.Printf("[DevServer] Error stopping app server: %v", err)
	}

	close(ds.stopChan)
	ds.wg.Wait()

	// Stop file watcher
	if ds.watcher != nil {
		ds.watcher.Stop()
	}

	// Stop reload server
	if ds.reloadServer != nil {
		ds.reloadServer.Close()
	}

	// Stop HTTP server
	if ds.httpServer != nil {
		ds.httpServer.Close()
	}

	log.Printf("[DevServer] Stopped")
	return nil
}

// handleFileChange handles file system changes
func (ds *DevServer) handleFileChange(files []string) error {
	ds.buildMutex.Lock()
	if ds.isBuilding {
		ds.buildMutex.Unlock()
		log.Printf("[DevServer] Build already in progress, skipping...")
		return nil
	}
	ds.isBuilding = true
	ds.buildMutex.Unlock()

	defer func() {
		ds.buildMutex.Lock()
		ds.isBuilding = false
		ds.buildMutex.Unlock()
	}()

	// Analyze impact
	impact := AnalyzeImpact(files)

	// Notify clients that build is starting
	ds.reloadServer.NotifyBuilding(files)

	// Handle based on scope
	switch impact.Scope {
	case ScopeUI:
		// Asset change - just notify for reload
		log.Printf("[DevServer] Asset changed: %v", files)
		ds.assetWatcher.HandleAssetChange(files)
		return nil

	case ScopeBackend:
		// Source code change - rebuild and restart
		log.Printf("[DevServer] Source changed: %v", files)
		return ds.handleBackendChange(files, impact)

	case ScopeConfig:
		// Config change - full restart
		log.Printf("[DevServer] Config changed: %v", files)
		return ds.handleConfigChange(files)
	}

	return nil
}

// handleBackendChange handles backend code changes
func (ds *DevServer) handleBackendChange(files []string, impact *ChangeImpact) error {
	start := time.Now()

	// Incremental build
	result, err := ds.compiler.IncrementalBuild(files)
	if err != nil {
		log.Printf("[DevServer] ✗ Build failed: %v", err)
		ds.displayErrors(result.Errors)

		// Notify clients of errors
		errorInfos := make([]*ErrorInfo, len(result.Errors))
		for i, compErr := range result.Errors {
			severity := "error"
			if compErr.Severity == errors.Warning {
				severity = "warning"
			}
			errorInfos[i] = &ErrorInfo{
				Message:  compErr.Message,
				File:     compErr.Location.File,
				Line:     compErr.Location.Line,
				Column:   compErr.Location.Column,
				Code:     compErr.Code,
				Phase:    compErr.Phase,
				Severity: severity,
			}
		}
		ds.reloadServer.NotifyErrors(errorInfos)

		return err
	}

	log.Printf("[DevServer] ✓ Build successful (%.0fms)", result.Duration.Seconds()*1000)

	// Build Go binary
	if err := ds.buildBinary(); err != nil {
		log.Printf("[DevServer] Failed to build binary: %v", err)
		ds.reloadServer.NotifyError(&ErrorInfo{
			Message: fmt.Sprintf("Failed to build binary: %v", err),
			Phase:   "build",
		})
		return err
	}

	// Restart server
	if err := ds.restartAppServer(); err != nil {
		log.Printf("[DevServer] Failed to restart server: %v", err)
		ds.reloadServer.NotifyError(&ErrorInfo{
			Message: fmt.Sprintf("Failed to restart server: %v", err),
			Phase:   "runtime",
		})
		return err
	}

	duration := time.Since(start)
	log.Printf("[DevServer] 🔄 Hot reloaded in %.0fms", duration.Seconds()*1000)

	// Notify success and reload
	ds.reloadServer.NotifySuccess(duration)
	ds.reloadServer.NotifyReload("backend")

	return nil
}

// handleConfigChange handles configuration changes
func (ds *DevServer) handleConfigChange(files []string) error {
	log.Printf("[DevServer] 🔄 Full restart triggered by config change")

	// Full rebuild
	result, err := ds.compiler.FullBuild()
	if err != nil {
		log.Printf("[DevServer] Build failed: %v", err)
		return err
	}

	log.Printf("[DevServer] Build successful (%.2fs)", result.Duration.Seconds())

	// Build binary
	if err := ds.buildBinary(); err != nil {
		return err
	}

	// Restart server
	if err := ds.restartAppServer(); err != nil {
		return err
	}

	ds.reloadServer.NotifyReload("config")
	return nil
}

// buildBinary builds the Go binary from generated code
func (ds *DevServer) buildBinary() error {
	cmd := exec.Command("go", "build", "-o", "build/app", "./build/generated")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	return nil
}

// startAppServer starts the application server
func (ds *DevServer) startAppServer() error {
	cmd := exec.Command("./build/app")
	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", ds.appPort))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set process group to allow proper cleanup
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start app: %w", err)
	}

	ds.appProcess = cmd.Process
	log.Printf("[DevServer] App server started (PID: %d)", ds.appProcess.Pid)

	// Monitor process in background and clean up if it exits unexpectedly
	go func() {
		cmd.Wait()
		ds.buildMutex.Lock()
		if ds.appProcess != nil && ds.appProcess.Pid == cmd.Process.Pid {
			ds.appProcess = nil
		}
		ds.buildMutex.Unlock()
	}()

	return nil
}

// stopAppServer stops the application server
func (ds *DevServer) stopAppServer() error {
	if ds.appProcess == nil {
		return nil
	}

	log.Printf("[DevServer] Stopping app server (PID: %d)", ds.appProcess.Pid)

	// Send SIGTERM for graceful shutdown
	if err := ds.appProcess.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		return nil
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		_, err := ds.appProcess.Wait()
		done <- err
	}()

	select {
	case <-done:
		// Process exited
	case <-time.After(5 * time.Second):
		// Timeout - force kill
		log.Printf("[DevServer] Timeout waiting for graceful shutdown, forcing kill")
		ds.appProcess.Kill()
	}

	ds.appProcess = nil
	return nil
}

// restartAppServer restarts the application server
func (ds *DevServer) restartAppServer() error {
	if err := ds.stopAppServer(); err != nil {
		return err
	}

	// Small delay to ensure port is released
	time.Sleep(100 * time.Millisecond)

	return ds.startAppServer()
}

// startHTTPServer starts the HTTP server for reload WebSocket
func (ds *DevServer) startHTTPServer() error {
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/__conduit_reload", ds.reloadServer.HandleWebSocket)

	// Serve reload script
	mux.HandleFunc("/__conduit/reload.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(reloadScript))
	})

	// Proxy all other requests to app server
	mux.HandleFunc("/", ds.proxyToApp)

	ds.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", ds.port),
		Handler: ds.injectReloadScript(mux),
	}

	ds.wg.Add(1)
	go func() {
		defer ds.wg.Done()
		if err := ds.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[DevServer] HTTP server error: %v", err)
		}
	}()

	return nil
}

// proxyToApp proxies requests to the application server
func (ds *DevServer) proxyToApp(w http.ResponseWriter, r *http.Request) {
	// Sanitize path to prevent directory traversal
	cleanPath := path.Clean(r.URL.Path)

	// Simple reverse proxy
	targetURL := fmt.Sprintf("http://localhost:%d%s", ds.appPort, cleanPath)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Failed to reach app server", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	// Copy response body
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

// injectReloadScript injects the reload script into HTML responses
func (ds *DevServer) injectReloadScript(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only inject for HTML responses
		if !strings.HasSuffix(r.URL.Path, ".html") && r.URL.Path != "/" {
			next.ServeHTTP(w, r)
			return
		}

		// TODO: Implement response body injection
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}

// displayErrors displays compilation errors in a formatted way
func (ds *DevServer) displayErrors(errs []errors.CompilerError) {
	if len(errs) == 0 {
		return
	}

	fmt.Fprintf(os.Stderr, "\n❌ Compilation failed with %d error(s):\n\n", len(errs))

	for i, err := range errs {
		fmt.Fprintf(os.Stderr, "%d. [%s] %s", i+1, err.Phase, err.Message)
		if err.Location.File != "" {
			fmt.Fprintf(os.Stderr, "\n   %s:%d:%d",
				err.Location.File, err.Location.Line, err.Location.Column)
		}
		fmt.Fprintln(os.Stderr)

		if i < len(errs)-1 {
			fmt.Fprintln(os.Stderr, strings.Repeat("-", 60))
		}
	}
	fmt.Fprintln(os.Stderr)
}
