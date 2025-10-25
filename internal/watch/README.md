# Watch Mode & Hot Reload Implementation

This package implements the watch mode and hot reload system for the Conduit programming language, enabling rapid development iteration with sub-500ms save-to-reload times.

## Overview

The watch system provides:

- **File System Watching**: Real-time monitoring of `.cdt` files and assets
- **Incremental Compilation**: Only recompiles changed files for faster builds
- **Hot Module Replacement**: Automatic browser reload via WebSocket
- **Error Recovery**: Graceful error handling with in-browser error overlay
- **Asset Watching**: CSS hot reload without full page refresh

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       DevServer                              │
│  ┌────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │   File     │→ │ Incremental  │→ │    Reload    │        │
│  │  Watcher   │  │  Compiler    │  │    Server    │        │
│  └────────────┘  └──────────────┘  └──────────────┘        │
│         │               │                    │               │
│         ↓               ↓                    ↓               │
│  ┌────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │  Debouncer │  │    Asset     │  │  WebSocket   │        │
│  │  (100ms)   │  │   Watcher    │  │  Clients     │        │
│  └────────────┘  └──────────────┘  └──────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. File Watcher (`watcher.go`)

Monitors the file system for changes using `fsnotify`.

**Key Features:**
- Watches `.cdt`, `.css`, `.js`, `.html` files
- Debounces file changes (100ms window)
- Ignores build artifacts and hidden files
- Pattern-based file matching

**Usage:**
```go
watcher, err := NewFileWatcher(
    []string{"*.cdt"},
    []string{"*.swp"},
    handleChange,
)
watcher.Start()
```

### 2. Incremental Compiler (`incremental.go`)

Compiles only changed files while maintaining a cache of parsed resources.

**Key Features:**
- Caches parsed ASTs per file
- Combines cached + changed resources for type checking
- Generates Go code for the entire program
- Sub-200ms incremental compile time

**Usage:**
```go
compiler := NewIncrementalCompiler()
result, err := compiler.IncrementalBuild(changedFiles)
```

### 3. Reload Server (`reload_server.go`)

WebSocket server that broadcasts reload messages to connected browsers.

**Key Features:**
- WebSocket-based real-time communication
- Broadcasts to multiple connected clients
- Sends build status, errors, and reload commands
- Auto-reconnection on disconnect

**Message Types:**
- `building`: Build started
- `success`: Build completed successfully
- `error`: Compilation error with details
- `reload`: Trigger browser reload (scoped: ui/backend/config)

**Usage:**
```go
rs := NewReloadServer()
rs.NotifyBuilding(files)
rs.NotifySuccess(duration)
rs.NotifyReload("ui")
```

### 4. Browser Reload Script (`assets/reload.js`)

Injected JavaScript that connects to the reload server and handles browser-side reload logic.

**Key Features:**
- Auto-connects to WebSocket server
- CSS hot reload (no full page refresh)
- Error overlay display
- Auto-reconnection with exponential backoff
- Status indicator in browser

### 5. Asset Watcher (`assets.go`)

Handles static asset changes (CSS, JS, images) with intelligent reload strategies.

**Key Features:**
- CSS changes → hot reload (no full page refresh)
- JS changes → full page reload
- Image changes → reload affected elements
- Impact analysis (UI, Backend, Config scopes)

### 6. Development Server (`dev_server.go`)

Orchestrates all components into a unified development experience.

**Key Features:**
- Initial full build on startup
- Coordinates file watching → compilation → reload
- Manages application server lifecycle (start/stop/restart)
- HTTP proxy with reload script injection
- Graceful shutdown handling

**Usage:**
```bash
conduit watch                              # Start with defaults
conduit watch --port 8080 --app-port 8081 # Custom ports
conduit watch --verbose                    # Verbose logging
```

## Performance Targets

| Metric | Target | Typical |
|--------|--------|---------|
| File change detection | <10ms | ~5ms |
| Incremental compile | <200ms | ~150ms |
| Browser reload | <100ms | ~50ms |
| **Total (save to visible)** | **<500ms** | **~250ms** |

## File Structure

```
internal/watch/
├── watcher.go              # File system watcher with debouncing
├── watcher_test.go         # File watcher tests
├── incremental.go          # Incremental compilation logic
├── incremental_test.go     # Incremental compiler tests
├── reload_server.go        # WebSocket reload server
├── reload_server_test.go   # Reload server tests
├── assets.go               # Asset watching and impact analysis
├── assets_test.go          # Asset watcher tests
├── dev_server.go           # Integrated development server
├── dev_server_test.go      # Dev server tests
└── assets/
    └── reload.js           # Browser reload script

internal/cli/commands/
├── watch.go                # CLI watch command
└── watch_test.go           # CLI command tests
```

## Implementation Details

### Debouncing

File changes are debounced with a 100ms window to prevent excessive rebuilds during rapid file saves.

```go
debouncer := NewDebouncer(100 * time.Millisecond)
debouncer.Add("file.cdt")
// Waits 100ms after last change before triggering callback
```

### Change Impact Analysis

The system analyzes which files changed to determine the appropriate reload strategy:

- **UI Scope**: CSS/JS/HTML → Browser refresh only
- **Backend Scope**: `.cdt` files → Rebuild + restart server
- **Config Scope**: Config files → Full restart

### Error Handling

Compilation errors are:
1. Logged to console with detailed context
2. Sent to browser via WebSocket
3. Displayed in error overlay (file, line, column, message)
4. Automatically cleared on successful rebuild

### WebSocket Protocol

Messages are JSON objects with the following structure:

```javascript
{
  "type": "reload|error|building|success",
  "scope": "ui|backend|config",
  "timestamp": 1234567890,
  "error": {
    "message": "Syntax error",
    "file": "post.cdt",
    "line": 10,
    "column": 5,
    "code": "PARSE001",
    "phase": "parser"
  },
  "files": ["post.cdt", "user.cdt"],
  "duration": 150.5  // milliseconds
}
```

## Testing

### Unit Tests

```bash
# Run all watch tests
go test ./internal/watch/... -v

# Run with coverage
go test ./internal/watch/... -cover

# Run short tests (skip integration)
go test ./internal/watch/... -short
```

### Coverage

Current test coverage: **51.9%**

Core modules have high coverage:
- File Watcher: ~85%
- Incremental Compiler: ~81%
- Reload Server: ~90%
- Assets: ~95%

Dev server has lower coverage (integration component) but is validated through end-to-end tests.

### Benchmarks

```bash
go test ./internal/watch/... -bench=. -benchmem
```

Key benchmarks:
- Debouncer.Add: ~200ns/op
- AnalyzeImpact: ~1.5μs/op
- IncrementalBuild: ~150ms/op (varies with project size)

## Usage Examples

### Basic Usage

```bash
# In a Conduit project directory
conduit watch
```

This will:
1. Perform initial full build
2. Start app server on port 3001
3. Start dev server with WebSocket on port 3000
4. Watch for file changes
5. Automatically rebuild and reload

### Custom Configuration

```bash
# Use different ports
conduit watch --port 8080 --app-port 8081

# Enable verbose logging
conduit watch --verbose
```

### Programmatic Usage

```go
import "github.com/conduit-lang/conduit/internal/watch"

config := &watch.DevServerConfig{
    Port:    3000,
    AppPort: 3001,
    WatchPatterns: []string{"*.cdt", "*.css"},
    IgnorePatterns: []string{"*.swp"},
}

devServer, _ := watch.NewDevServer(config)
devServer.Start()

// Later...
devServer.Stop()
```

## Browser Integration

Add the reload script to your HTML templates:

```html
<!DOCTYPE html>
<html>
<head>
    <title>My App</title>
</head>
<body>
    <!-- Your content -->

    <!-- In development mode only -->
    <script src="http://localhost:3000/__conduit/reload.js"></script>
</body>
</html>
```

The script will:
- Connect to the WebSocket server
- Show a status indicator (bottom-right corner)
- Display compilation errors in an overlay
- Automatically reload on file changes
- Handle CSS hot reload without full page refresh

## Troubleshooting

### Watch not detecting changes

- Ensure you're in a Conduit project (has `app/` directory)
- Check file permissions
- Verify files match watch patterns (`*.cdt`)

### Compilation errors not showing

- Check WebSocket connection in browser DevTools
- Verify port 3000 is accessible
- Check browser console for connection errors

### Slow reload times

- Check file count (too many files can slow down incremental builds)
- Verify SSD/fast storage
- Check CPU usage during builds

## Performance Optimizations

1. **Incremental Compilation**: Only changed files are re-parsed
2. **Resource Caching**: Parsed ASTs cached per file
3. **Debouncing**: Prevents excessive rebuilds during rapid saves
4. **Parallel Processing**: File watching runs in goroutines
5. **WebSocket Batching**: Multiple changes batched into single rebuild

## Future Enhancements

Potential improvements (not in scope for CON-42):

- Source maps for debugging generated Go code
- Module-level hot reload (no server restart)
- CSS preprocessor support (SASS, LESS)
- Live reload for template files
- Performance metrics dashboard
- Build time visualization

## Dependencies

- `github.com/fsnotify/fsnotify`: File system notifications
- `github.com/gorilla/websocket`: WebSocket communication
- Standard library: `net/http`, `os/exec`, `sync`, etc.

## License

Part of the Conduit programming language project.
