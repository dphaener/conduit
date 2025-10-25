# Performance Examples

This directory contains examples demonstrating the performance optimization features of the Conduit web framework.

## Examples

### basic_server.go

A simple HTTP server with basic middleware and graceful shutdown.

**Run:**
```bash
go run basic_server.go
```

**Test:**
```bash
curl http://localhost:8080/
curl http://localhost:8080/health
```

### optimized_server.go

A production-ready server with all performance optimizations enabled:

- HTTP/2 support
- Database connection pooling
- Compression middleware
- Timeout middleware
- Static file serving with caching
- Response streaming
- Profiling endpoints
- Graceful shutdown

**Prerequisites:**
- PostgreSQL database running
- Create database: `createdb conduit`

**Run:**
```bash
go run optimized_server.go
```

**Test:**
```bash
# Health check
curl http://localhost:8080/health

# List posts (regular JSON)
curl http://localhost:8080/api/posts

# Stream posts (chunked transfer encoding)
curl http://localhost:8080/api/posts/stream

# Access profiling
open http://localhost:8080/debug/pprof/
```

## Load Testing

Run load tests against the examples:

```bash
# Start the server
go run optimized_server.go

# In another terminal, run load tests
cd ../..
./scripts/load_test.sh
```

## Benchmarking

Run benchmarks:

```bash
# Run all benchmarks
go test -bench=. ../../benchmarks/...

# Run specific benchmark
go test -bench=BenchmarkJSONResponse ../../benchmarks/...

# With memory allocation stats
go test -bench=. -benchmem ../../benchmarks/...

# Save results for comparison
go test -bench=. ../../benchmarks/... > before.txt
# Make changes...
go test -bench=. ../../benchmarks/... > after.txt
benchcmp before.txt after.txt
```

## Profiling

### CPU Profile

```bash
# Collect CPU profile for 30 seconds
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof

# Analyze
go tool pprof cpu.prof
```

### Memory Profile

```bash
# Collect heap profile
curl http://localhost:8080/debug/pprof/heap > heap.prof

# Analyze
go tool pprof heap.prof
```

### Goroutine Profile

```bash
# View goroutines
curl http://localhost:8080/debug/pprof/goroutine

# Analyze
curl http://localhost:8080/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof
```

## Performance Targets

The optimized server is designed to achieve:

- **Throughput**: 30,000+ requests/second
- **Latency P50**: <5ms
- **Latency P95**: <10ms
- **Latency P99**: <50ms
- **Memory**: <500MB for 10K concurrent connections
- **CPU**: <80% utilization at peak load

## Tips

1. **Use HTTP/2**: Significantly reduces latency for multiple requests
2. **Enable Compression**: Reduces bandwidth by 60-80% for text content
3. **Configure Timeouts**: Prevents resource exhaustion from slow clients
4. **Connection Pooling**: Reuses database connections efficiently
5. **Static File Caching**: Set appropriate cache headers for assets
6. **Response Streaming**: Use for large datasets to reduce memory usage
7. **Profile Regularly**: Identify bottlenecks in production

## See Also

- [Server Package Documentation](../../internal/web/server/README.md)
- [Load Testing Script](../../scripts/load_test.sh)
- [Benchmark Suite](../../benchmarks/)
