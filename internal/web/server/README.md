# Server Package

Production-ready HTTP server with HTTP/2 support, connection pooling, graceful shutdown, and performance optimizations.

## Features

- **HTTP/2 Support**: Automatic HTTP/2 upgrade with TLS
- **Connection Pooling**: Optimized database connection pool configuration
- **Keep-Alive**: Configurable connection keep-alive settings
- **Graceful Shutdown**: Clean shutdown with timeout and cleanup hooks
- **Optimized Timeouts**: Configurable read/write/idle timeouts
- **Production Ready**: Battle-tested configuration for high-throughput workloads

## Quick Start

### Basic Server

```go
package main

import (
    "net/http"
    "github.com/conduit-lang/conduit/internal/web/server"
    "github.com/conduit-lang/conduit/internal/web/router"
)

func main() {
    // Create router
    r := router.NewRouter()
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, World!"))
    })

    // Create server with default config
    config := server.DefaultConfig(r)
    srv, err := server.New(config)
    if err != nil {
        panic(err)
    }

    // Start server
    srv.ListenAndServe()
}
```

### Server with Graceful Shutdown

```go
package main

import (
    "github.com/conduit-lang/conduit/internal/web/server"
    "github.com/conduit-lang/conduit/internal/web/router"
)

func main() {
    r := router.NewRouter()
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello!"))
    })

    // Create server
    config := server.DefaultConfig(r)
    srv, _ := server.New(config)

    // Start with graceful shutdown
    shutdownConfig := server.DefaultShutdownConfig()
    server.StartWithGracefulShutdown(srv, shutdownConfig)
}
```

### HTTPS with HTTP/2

```go
package main

import (
    "crypto/tls"
    "github.com/conduit-lang/conduit/internal/web/server"
    "github.com/conduit-lang/conduit/internal/web/router"
)

func main() {
    r := router.NewRouter()

    config := server.DefaultConfig(r)
    config.Address = ":8443"
    config.TLSConfig = &server.TLSConfig{
        CertFile:   "cert.pem",
        KeyFile:    "key.pem",
        MinVersion: tls.VersionTLS12,
    }
    config.EnableHTTP2 = true

    srv, _ := server.New(config)
    srv.ListenAndServe()
}
```

### Database Connection Pooling

```go
package main

import (
    "database/sql"
    "time"
    "github.com/conduit-lang/conduit/internal/web/server"
    _ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
    // Open database connection
    db, err := sql.Open("pgx", "postgres://localhost/mydb")
    if err != nil {
        panic(err)
    }

    // Configure connection pool
    dbConfig := &server.DatabaseConfig{
        DB:              db,
        MaxOpenConns:    100,
        MaxIdleConns:    10,
        ConnMaxLifetime: time.Hour,
        ConnMaxIdleTime: 10 * time.Minute,
    }

    // Create server with database config
    config := server.DefaultConfig(handler)
    config.Database = dbConfig

    srv, _ := server.New(config)
    srv.ListenAndServe()
}
```

## Configuration

### Server Config

```go
type Config struct {
    // Address is the server listen address (e.g., ":8080")
    Address string

    // Handler is the HTTP handler for the server
    Handler http.Handler

    // TLS configuration
    TLSConfig *TLSConfig

    // Timeouts
    ReadTimeout       time.Duration  // Default: 15s
    WriteTimeout      time.Duration  // Default: 15s
    IdleTimeout       time.Duration  // Default: 60s
    ReadHeaderTimeout time.Duration  // Default: 10s

    // Connection limits
    MaxHeaderBytes int  // Default: 1MB

    // Database configuration for connection pooling
    Database *DatabaseConfig

    // HTTP/2 settings
    EnableHTTP2 bool  // Default: true
}
```

### Default Configuration

The default configuration is optimized for production workloads:

- **ReadTimeout**: 15 seconds
- **WriteTimeout**: 15 seconds
- **IdleTimeout**: 60 seconds (keep-alive)
- **ReadHeaderTimeout**: 10 seconds
- **MaxHeaderBytes**: 1 MB
- **HTTP/2**: Enabled

### Database Connection Pool

```go
type DatabaseConfig struct {
    DB              *sql.DB
    MaxOpenConns    int           // Default: 100
    MaxIdleConns    int           // Default: 10
    ConnMaxLifetime time.Duration // Default: 1 hour
    ConnMaxIdleTime time.Duration // Default: 10 minutes
}
```

## Graceful Shutdown

The server supports graceful shutdown with cleanup hooks:

```go
// Create graceful shutdown handler
gs := server.NewGracefulShutdown(srv, nil)

// Register cleanup hooks
gs.RegisterHook(func(ctx context.Context) error {
    // Close database connections
    return db.Close()
})

gs.RegisterHook(func(ctx context.Context) error {
    // Close cache connections
    return cache.Close()
})

// Start server and wait for shutdown signal
gs.Start()
```

### Shutdown Configuration

```go
type ShutdownConfig struct {
    // Timeout is the maximum time to wait for shutdown
    Timeout time.Duration  // Default: 30s

    // Signals to listen for (default: SIGINT, SIGTERM)
    Signals []os.Signal

    // Logger for shutdown messages
    Logger Logger
}
```

## Performance Targets

The server is optimized to achieve:

- **Throughput**: 30,000+ requests/second
- **Latency P50**: <5ms
- **Latency P95**: <10ms
- **Latency P99**: <50ms
- **Memory**: <500MB for 10K concurrent connections
- **CPU**: <80% utilization at peak load

## Production Deployment

### Recommended Configuration

```go
config := &server.Config{
    Address:           ":8080",
    Handler:           handler,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      15 * time.Second,
    IdleTimeout:       60 * time.Second,
    ReadHeaderTimeout: 10 * time.Second,
    MaxHeaderBytes:    1 << 20, // 1 MB
    EnableHTTP2:       true,
    Database: &server.DatabaseConfig{
        DB:              db,
        MaxOpenConns:    100,
        MaxIdleConns:    10,
        ConnMaxLifetime: time.Hour,
        ConnMaxIdleTime: 10 * time.Minute,
    },
}
```

### With Middleware

```go
r := router.NewRouter()

// Add middleware
r.Use(middleware.RequestID())
r.Use(middleware.Recovery())
r.Use(middleware.Logging())
r.Use(middleware.Compression())
r.Use(middleware.Timeout(30 * time.Second))

// Routes
r.Get("/", homeHandler)
r.Get("/api/posts", postsHandler)

// Create server
config := server.DefaultConfig(r)
srv, _ := server.New(config)
```

## Examples

See the `examples/` directory for complete examples:

- `examples/basic_server.go` - Basic HTTP server
- `examples/https_server.go` - HTTPS with HTTP/2
- `examples/graceful_shutdown.go` - Graceful shutdown
- `examples/production_server.go` - Production-ready configuration

## Testing

Run benchmarks:

```bash
go test -bench=. ./benchmarks/...
```

Run load tests:

```bash
./scripts/load_test.sh
```

## Profiling

Enable profiling in production:

```go
import "github.com/conduit-lang/conduit/internal/web/profiling"

// Register profiling routes
profiling.EnableDefaultProfiling(router)

// Access profiles at:
// http://localhost:8080/debug/pprof/
```

## See Also

- [Middleware Package](../middleware/README.md)
- [Profiling Package](../profiling/README.md)
- [Load Testing Guide](../../scripts/load_test.sh)
