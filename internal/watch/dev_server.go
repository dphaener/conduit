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
	"github.com/conduit-lang/conduit/internal/tooling/build"
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
	autoMigrate    bool
	verbose        bool
	watchPatterns  []string
	ignorePatterns []string

	// State
	isBuilding      bool
	buildMutex      sync.Mutex
	appProcessMutex sync.Mutex
	stopChan        chan struct{}
	wg              sync.WaitGroup
}

// DevServerConfig holds configuration for the dev server
type DevServerConfig struct {
	Port           int
	AppPort        int
	AutoMigrate    bool
	Verbose        bool
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
		autoMigrate:    config.AutoMigrate,
		verbose:        config.Verbose,
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

		// Generate migrations if schema changed
		if err := ds.compiler.HandleMigrations(); err != nil {
			log.Printf("[DevServer] Warning: Migration generation failed: %v", err)
		}

		// Build Go binary
		if err := ds.buildBinary(); err != nil {
			log.Printf("[DevServer] Failed to build binary: %v", err)
		} else {
			// Check for pending migrations before starting server
			migrationStatus, err := build.CheckMigrationStatus()
			if err != nil {
				log.Printf("[DevServer] Warning: Failed to check migration status: %v", err)
			} else if migrationStatus.DatabaseSkipped {
				// DATABASE_URL not set
				if ds.autoMigrate {
					log.Printf("[DevServer] Warning: DATABASE_URL not set - cannot auto-migrate")
				}
			} else if migrationStatus.DatabaseError != nil {
				// Database connection check failed
				if ds.autoMigrate {
					log.Printf("[DevServer] Warning: Database connection failed - auto-migrate disabled for this build")
					log.Printf("[DevServer] Database error: %v", migrationStatus.DatabaseError)
				}
			} else if len(migrationStatus.Pending) > 0 {
				// Migrations detected on initial start
				if err := ds.handleMigrations(migrationStatus); err != nil {
					log.Printf("[DevServer] Migration handling failed: %v", err)
					ds.displayMigrationWarning(migrationStatus)
				}
			}

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

	if ds.verbose {
		log.Printf("[DevServer] Processing %d changed files: %v", len(files), files)
	}

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

	// Generate migrations if schema changed
	if err := ds.compiler.HandleMigrations(); err != nil {
		log.Printf("[DevServer] Warning: Migration generation failed: %v", err)
	}

	// Build Go binary
	if ds.verbose {
		log.Printf("[DevServer] Building Go binary...")
	}
	if err := ds.buildBinary(); err != nil {
		log.Printf("[DevServer] Failed to build binary: %v", err)
		ds.reloadServer.NotifyError(&ErrorInfo{
			Message: fmt.Sprintf("Failed to build binary: %v", err),
			Phase:   "build",
		})
		return err
	}
	if ds.verbose {
		log.Printf("[DevServer] Go binary built successfully")
	}

	// Check for pending migrations
	if ds.verbose {
		log.Printf("[DevServer] Checking for pending migrations...")
	}
	migrationStatus, err := build.CheckMigrationStatus()
	if err != nil {
		log.Printf("[DevServer] Warning: Failed to check migration status: %v", err)
		// Don't attempt auto-migration if we can't check status
		if ds.autoMigrate {
			log.Printf("[DevServer] Auto-migrate disabled for this build (migration check failed)")
		}
	} else if migrationStatus.DatabaseSkipped {
		// DATABASE_URL not set
		if ds.autoMigrate {
			log.Printf("[DevServer] Warning: DATABASE_URL not set - cannot auto-migrate")
		}
	} else if migrationStatus.DatabaseError != nil {
		// Database connection check failed
		if ds.autoMigrate {
			log.Printf("[DevServer] Warning: Database connection failed - auto-migrate disabled for this build")
			log.Printf("[DevServer] Database error: %v", migrationStatus.DatabaseError)
		}
	} else if len(migrationStatus.Pending) > 0 {
		// Migrations detected
		if err := ds.handleMigrations(migrationStatus); err != nil {
			log.Printf("[DevServer] Migration handling failed: %v", err)
			log.Printf("[DevServer] Server NOT restarted (pending migration)")

			// Display warning but don't restart server
			ds.displayMigrationWarning(migrationStatus)
			return nil
		}
	}

	// Restart server (only if no migrations or migrations were applied successfully)
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

	ds.appProcessMutex.Lock()
	ds.appProcess = cmd.Process
	pid := ds.appProcess.Pid
	ds.appProcessMutex.Unlock()

	log.Printf("[DevServer] App server started (PID: %d)", pid)

	// Monitor process in background and clean up if it exits unexpectedly
	go func() {
		cmd.Wait()
		ds.appProcessMutex.Lock()
		if ds.appProcess != nil && ds.appProcess.Pid == cmd.Process.Pid {
			ds.appProcess = nil
		}
		ds.appProcessMutex.Unlock()
	}()

	return nil
}

// stopAppServer stops the application server
func (ds *DevServer) stopAppServer() error {
	ds.appProcessMutex.Lock()
	proc := ds.appProcess
	ds.appProcessMutex.Unlock()

	if proc == nil {
		return nil
	}

	log.Printf("[DevServer] Stopping app server (PID: %d)", proc.Pid)

	// Send SIGTERM for graceful shutdown
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		ds.appProcessMutex.Lock()
		ds.appProcess = nil
		ds.appProcessMutex.Unlock()
		return nil
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		_, err := proc.Wait()
		done <- err
	}()

	select {
	case <-done:
		// Process exited
	case <-time.After(5 * time.Second):
		// Timeout - force kill
		log.Printf("[DevServer] Timeout waiting for graceful shutdown, forcing kill")
		proc.Kill()
	}

	ds.appProcessMutex.Lock()
	ds.appProcess = nil
	ds.appProcessMutex.Unlock()

	return nil
}

// restartAppServer restarts the application server
func (ds *DevServer) restartAppServer() error {
	if ds.verbose {
		log.Printf("[DevServer] Restarting application server...")
	}

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

// handleMigrations handles pending migrations based on auto-migrate setting
func (ds *DevServer) handleMigrations(status *build.MigrationStatus) error {
	if ds.autoMigrate {
		// Check if any pending migrations are breaking or involve data loss
		hasBreaking := status.HasBreaking || status.HasDataLoss

		if hasBreaking {
			// Require confirmation for breaking migrations
			log.Printf("[DevServer] WARNING: Pending migrations contain breaking changes or data loss risk")

			migrator := build.NewAutoMigrator(build.AutoMigrateOptions{
				Mode:        build.AutoMigrateApply,
				SkipConfirm: false, // Require confirmation for breaking changes
			})

			if err := migrator.Run(); err != nil {
				return fmt.Errorf("auto-migrate failed: %w", err)
			}
		} else {
			// Safe migrations - auto-apply without confirmation
			log.Printf("[DevServer] Auto-applying %d safe migration(s)...", len(status.Pending))

			migrator := build.NewAutoMigrator(build.AutoMigrateOptions{
				Mode:        build.AutoMigrateApply,
				SkipConfirm: true,
			})

			if err := migrator.Run(); err != nil {
				return fmt.Errorf("auto-migrate failed: %w", err)
			}
		}

		log.Printf("[DevServer] ✓ Applied %d migration(s)", len(status.Pending))
		return nil
	}

	// Without auto-migrate, return error to prevent server restart
	return fmt.Errorf("pending migrations require manual application")
}

// displayMigrationWarning displays a warning about pending migrations
func (ds *DevServer) displayMigrationWarning(status *build.MigrationStatus) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("⚠️  SCHEMA CHANGED")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()

	for _, m := range status.Pending {
		migrationFile := fmt.Sprintf("migrations/%03d_%s.sql", m.Version, m.Name)
		fmt.Printf("  Created: %s\n", migrationFile)
	}

	fmt.Println()
	fmt.Println("  ⚠️  Server NOT restarted (pending migration)")
	fmt.Println()
	fmt.Println("  Apply migration:")
	fmt.Println("    conduit migrate up (in new terminal)")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
}
