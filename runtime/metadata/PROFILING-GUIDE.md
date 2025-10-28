# Profiling Guide

This guide explains how to profile the Conduit runtime introspection system to measure performance and identify potential regressions.

## Table of Contents

- [Quick Start](#quick-start)
- [Running Benchmarks](#running-benchmarks)
- [CPU Profiling](#cpu-profiling)
- [Memory Profiling](#memory-profiling)
- [Interpreting Results](#interpreting-results)
- [Identifying Regressions](#identifying-regressions)
- [Continuous Integration](#continuous-integration)
- [Common Issues](#common-issues)

---

## Quick Start

### Prerequisites

- Go 1.23 or later
- Graphviz (for pprof visualization): `brew install graphviz` (macOS)
- Basic understanding of Go benchmarks and pprof

### Run Full Benchmark Suite

```bash
cd runtime/metadata
go test -bench=. -benchmem -benchtime=1s
```

Expected output:
```
BenchmarkRegistryInit-10                 3537    342666 ns/op    207170 B/op    3653 allocs/op
BenchmarkSimpleQueries/Resource-10      26068107    45.10 ns/op    0 B/op       0 allocs/op
...
```

### Run Performance Regression Tests

```bash
go test -v -run=TestPerformance
```

Expected output:
```
=== RUN   TestPerformance_RegistryInit
--- PASS: TestPerformance_RegistryInit (0.42s)
=== RUN   TestPerformance_SimpleQuery
--- PASS: TestPerformance_SimpleQuery (0.35s)
...
```

---

## Running Benchmarks

### Basic Benchmarks

Run all benchmarks:
```bash
go test -bench=. -benchmem
```

Run specific benchmark:
```bash
go test -bench=BenchmarkRegistryInit -benchmem
```

Run benchmark with custom duration:
```bash
go test -bench=BenchmarkRegistryInit -benchmem -benchtime=5s
```

Run benchmark with custom iterations:
```bash
go test -bench=BenchmarkRegistryInit -benchmem -benchtime=1000x
```

### Benchmark Output Explained

```
BenchmarkRegistryInit-10    3537    342666 ns/op    207170 B/op    3653 allocs/op
```

- `BenchmarkRegistryInit`: Benchmark name
- `-10`: Number of CPU cores used (GOMAXPROCS)
- `3537`: Number of iterations run
- `342666 ns/op`: Average time per operation (342µs)
- `207170 B/op`: Bytes allocated per operation (207KB)
- `3653 allocs/op`: Number of allocations per operation

### Comparing Benchmarks

Run benchmark and save baseline:
```bash
go test -bench=. -benchmem > baseline.txt
```

Make changes, then run again:
```bash
go test -bench=. -benchmem > new.txt
```

Compare results:
```bash
go install golang.org/x/perf/cmd/benchstat@latest
benchstat baseline.txt new.txt
```

Expected output:
```
name              old time/op    new time/op    delta
RegistryInit-10     342µs ± 2%     340µs ± 1%   ~
```

---

## CPU Profiling

CPU profiling identifies where the program spends execution time.

### Generate CPU Profile

```bash
go test -bench=BenchmarkRegistryInit -cpuprofile=cpu.prof -benchtime=10s
```

### Analyze CPU Profile (Web UI)

```bash
go tool pprof -http=:8080 cpu.prof
```

Opens browser with interactive flame graph and call tree.

### Analyze CPU Profile (Terminal)

```bash
# Show top functions by CPU time
go tool pprof -top cpu.prof

# Show call graph for specific function
go tool pprof -list=buildIndexes cpu.prof

# Show cumulative time (includes called functions)
go tool pprof -cum cpu.prof
```

### Example CPU Profile Output

```
(pprof) top
Showing nodes accounting for 850ms, 85.00% of 1000ms total
      flat  flat%   sum%        cum   cum%
     508ms 50.80% 50.80%      508ms 50.80%  runtime.kevent
     152ms 15.20% 66.00%      152ms 15.20%  encoding/json.Unmarshal
      81ms  8.10% 74.10%       81ms  8.10%  runtime.mallocgc
      64ms  6.40% 80.50%       64ms  6.40%  Registry.buildIndexes
      45ms  4.50% 85.00%       45ms  4.50%  reflect.Value.Set
```

**Reading this output:**
- `flat`: Time spent in function itself (not callees)
- `flat%`: Percentage of total time
- `sum%`: Cumulative percentage
- `cum`: Time spent in function + callees
- `cum%`: Cumulative percentage

### Interpreting CPU Profile

**Expected patterns:**
- `runtime.kevent` high (50%+): Normal, OS event handling
- `encoding/json.Unmarshal` moderate (10-20%): Normal, JSON parsing
- `Registry.*` functions low (<10% each): Good, efficient implementation

**Warning signs:**
- Any `Registry.*` function >20%: Potential hotspot
- `reflect.*` high (>30%): Excessive reflection overhead
- `runtime.mallocgc` high (>30%): Memory allocation pressure

---

## Memory Profiling

Memory profiling identifies allocation patterns and potential leaks.

### Generate Memory Profile

```bash
go test -bench=BenchmarkRegistryInit -memprofile=mem.prof -benchtime=10s
```

### Analyze Memory Profile (Web UI)

```bash
go tool pprof -http=:8080 mem.prof
```

### Analyze Memory Profile (Terminal)

```bash
# Show top allocating functions
go tool pprof -top mem.prof

# Show allocation graph
go tool pprof -alloc_space mem.prof

# Show in-use memory (detects leaks)
go tool pprof -inuse_space mem.prof

# Show allocation counts (not just bytes)
go tool pprof -alloc_objects mem.prof
```

### Example Memory Profile Output

```
(pprof) top
Showing nodes accounting for 688MB, 100% of 688MB total
      flat  flat%   sum%        cum   cum%
     540MB 78.49% 78.49%      540MB 78.49%  reflect.growslice
      86MB 12.50% 90.99%       86MB 12.50%  Registry.buildIndexes
      45MB  6.54% 97.53%       45MB  6.54%  encoding/json.Unmarshal
      17MB  2.47%   100%       17MB  2.47%  (other)
```

### Memory Profile Options

View by allocation type:
```bash
# Total bytes allocated (default)
go tool pprof -alloc_space mem.prof

# Total allocations (count)
go tool pprof -alloc_objects mem.prof

# Bytes in use (detects leaks)
go tool pprof -inuse_space mem.prof

# Objects in use (detects object leaks)
go tool pprof -inuse_objects mem.prof
```

### Interpreting Memory Profile

**Expected patterns:**
- `reflect.growslice` high (70-80%): Normal, JSON array decoding
- `buildIndexes` moderate (10-15%): Normal, index construction
- `inuse_space` low after GC: No memory leaks

**Warning signs:**
- `inuse_space` growing over time: Memory leak
- `alloc_space` >1GB for 50 resources: Excessive allocations
- Any single function >50% of allocations: Potential optimization target

---

## Interpreting Results

### Performance Targets

From `IMPLEMENTATION-RUNTIME.md`:

| Metric | Target | Acceptable | Unacceptable |
|--------|--------|------------|--------------|
| Registry init | <10ms | <50ms | >100ms |
| Simple query | <1ms | <5ms | >10ms |
| Complex query (depth 3) | <20ms | <50ms | >100ms |
| Memory (50 resources) | <10MB | <50MB | >100MB |

### Benchmark Interpretation

**Registry Initialization:**
```
BenchmarkRegistryInit-10    3537    342666 ns/op    207170 B/op    3653 allocs/op
```
- ✅ **342µs < 10ms target** (29x faster)
- ✅ **207KB < 10MB target** (48x better)
- ✅ **3653 allocs** (reasonable for initialization)

**Simple Queries:**
```
BenchmarkSimpleQueries/Resource-10    26068107    45.10 ns/op    0 B/op    0 allocs/op
```
- ✅ **45ns < 1ms target** (1000x faster)
- ✅ **0 allocations** (perfect, no GC pressure)

**Complex Queries (Cold):**
```
BenchmarkComplexQueries/Depth3Cold-10    143779    8059 ns/op    4432 B/op    82 allocs/op
```
- ✅ **8µs < 20ms target** (2500x faster)
- ✅ **4.4KB allocated** (acceptable for cold query)

**Complex Queries (Cached):**
```
BenchmarkComplexQueries/Depth3Cached-10    10539357    112.2 ns/op    0 B/op    0 allocs/op
```
- ✅ **112ns < 1ms target** (1000x faster)
- ✅ **0 allocations** (cache hit)
- ✅ **70x speedup** from caching (8µs → 112ns)

### Memory Interpretation

**Total allocations during init:**
```
alloc_space: 688MB (mostly temporary, GC'd immediately)
inuse_space: 500KB (steady-state after GC)
```

- ✅ **688MB temporary** (normal for JSON decoding)
- ✅ **500KB steady-state** (far below 10MB target)
- ✅ **No leaks** (inuse much lower than alloc)

---

## Identifying Regressions

### Automated Regression Tests

Run performance regression tests:
```bash
go test -v -run=TestPerformance
```

These tests fail if performance degrades beyond **acceptable** thresholds (not targets):
- Registry init: >50ms (vs 10ms target)
- Simple query: >5ms (vs 1ms target)
- Complex query depth 3: >50ms (vs 20ms target)
- Memory usage: >50MB (vs 10MB target)

### Manual Regression Detection

1. **Establish baseline:**
   ```bash
   git checkout main
   go test -bench=. -benchmem > baseline.txt
   ```

2. **Make changes:**
   ```bash
   git checkout feature-branch
   # ... make changes ...
   ```

3. **Run benchmarks:**
   ```bash
   go test -bench=. -benchmem > feature.txt
   ```

4. **Compare:**
   ```bash
   benchstat baseline.txt feature.txt
   ```

5. **Interpret results:**
   ```
   name              old time/op    new time/op    delta
   RegistryInit-10     342µs ± 2%     450µs ± 3%   +31.58%  (p=0.000 n=10+10)
   ```
   - `+31.58%`: 31% slower (regression!)
   - `p=0.000`: Statistically significant difference
   - `n=10+10`: 10 samples for each version

### Regression Thresholds

- **<5% change**: Noise, ignore
- **5-20% slower**: Minor regression, investigate if cumulative
- **20-50% slower**: Moderate regression, requires fix before merge
- **>50% slower**: Major regression, block merge

- **>20% faster**: Verify correctness (might be skipping work)

### Common Regression Causes

1. **Adding reflection:**
   - Symptom: 10-100x slower queries
   - Fix: Use type assertions or pre-computed indexes

2. **Copying instead of referencing:**
   - Symptom: Allocations increase, queries 2-10x slower
   - Fix: Return pointers, not values

3. **Inefficient data structures:**
   - Symptom: O(n) queries become slow with large schemas
   - Fix: Use maps for O(1) lookups

4. **Missing cache:**
   - Symptom: Cold query performance degradation
   - Fix: Ensure cache is enabled and keyed correctly

5. **Excessive allocations:**
   - Symptom: High `alloc_space`, increased GC pressure
   - Fix: Reuse buffers, use sync.Pool, avoid unnecessary copies

---

## Continuous Integration

### GitHub Actions Workflow

Add to `.github/workflows/performance.yml`:

```yaml
name: Performance Tests

on:
  pull_request:
    paths:
      - 'runtime/metadata/**'
  push:
    branches: [main]

jobs:
  performance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Run performance regression tests
        run: |
          cd runtime/metadata
          go test -v -run=TestPerformance

      - name: Run benchmarks
        run: |
          cd runtime/metadata
          go test -bench=. -benchmem > benchmarks.txt
          cat benchmarks.txt

      - name: Compare with baseline (main branch)
        if: github.event_name == 'pull_request'
        run: |
          git fetch origin main
          git checkout origin/main
          cd runtime/metadata
          go test -bench=. -benchmem > baseline.txt

          git checkout -
          go install golang.org/x/perf/cmd/benchstat@latest
          benchstat baseline.txt benchmarks.txt
```

### Pre-commit Hook

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash

echo "Running performance regression tests..."
cd runtime/metadata
go test -run=TestPerformance -timeout=30s

if [ $? -ne 0 ]; then
    echo "❌ Performance regression detected!"
    echo "Run 'go test -v -run=TestPerformance' for details"
    exit 1
fi

echo "✅ Performance tests passed"
```

Make executable:
```bash
chmod +x .git/hooks/pre-commit
```

---

## Common Issues

### Issue: Benchmark Results Vary Wildly

**Symptom:**
```
BenchmarkRegistryInit-10    3537    342666 ns/op
BenchmarkRegistryInit-10    2134    782451 ns/op  (2nd run)
```

**Causes:**
- CPU throttling (thermal)
- Background processes
- Insufficient iterations

**Fixes:**
```bash
# Increase benchmark time
go test -bench=. -benchtime=10s

# Run on performance cores (macOS)
taskset -c 0-3 go test -bench=.

# Close background apps
# Disable CPU throttling in BIOS/system settings
```

### Issue: pprof Shows No Data

**Symptom:**
```
(pprof) top
Showing nodes accounting for 0, 0% of 0 total
```

**Causes:**
- Benchmark didn't run long enough
- Profile file empty or corrupted

**Fixes:**
```bash
# Increase benchmark duration
go test -bench=BenchmarkRegistryInit -cpuprofile=cpu.prof -benchtime=30s

# Verify profile file exists and has content
ls -lh cpu.prof

# Regenerate profile
rm cpu.prof
go test -bench=. -cpuprofile=cpu.prof
```

### Issue: Memory Profile Shows Unexpected Leaks

**Symptom:**
```
(pprof) top -inuse_space
500MB still in use after test completion
```

**Causes:**
- Test harness holding references
- Deferred cleanup not running
- Global variables

**Fixes:**
```bash
# Force GC before profile
go test -bench=. -memprofile=mem.prof -memprofilerate=1

# Check for global variables
grep -r "var registry" .

# Use memprofile at specific point
# (modify test to call runtime.MemProfileRate)
```

### Issue: Benchmarks Too Fast to Measure

**Symptom:**
```
BenchmarkSimpleQueries/Resource-10    1000000000    0.00 ns/op
```

**Causes:**
- Compiler optimized away the code
- Result not used (dead code elimination)

**Fixes:**
```go
// Store result in package-level var to prevent optimization
var result *metadata.Resource

func BenchmarkSimpleQueries(b *testing.B) {
    var r *metadata.Resource
    for i := 0; i < b.N; i++ {
        r = registry.Resource("Post")
    }
    result = r  // Prevent optimization
}
```

### Issue: CPU Profile Shows Unexpected Functions

**Symptom:**
```
(pprof) top
90%  testing.(*B).launch
```

**Causes:**
- Profiling overhead dominates
- Benchmark runs too fast

**Fixes:**
```bash
# Increase work per iteration
go test -bench=. -benchtime=1000000x -cpuprofile=cpu.prof

# Profile only the hot path
# (use runtime/pprof.StartCPUProfile in specific function)
```

---

## Advanced Profiling

### Mutex Profiling (Concurrency Bottlenecks)

```bash
go test -bench=. -mutexprofile=mutex.prof
go tool pprof -http=:8080 mutex.prof
```

Expected: **No mutex contention** (all queries are read-only).

### Block Profiling (Goroutine Blocking)

```bash
go test -bench=. -blockprofile=block.prof
go tool pprof -http=:8080 block.prof
```

Expected: **No blocking** (no goroutines or channels in introspection).

### Trace Profiling (Detailed Execution Timeline)

```bash
go test -bench=BenchmarkRegistryInit -trace=trace.out
go tool trace trace.out
```

Opens browser with detailed execution timeline showing:
- Goroutine execution
- Network/syscall blocking
- GC pauses
- Scheduler decisions

### Continuous Profiling (Production)

For production monitoring, use [pprof HTTP endpoints](https://pkg.go.dev/net/http/pprof):

```go
import _ "net/http/pprof"

func main() {
    go func() {
        http.ListenAndServe("localhost:6060", nil)
    }()
    // ... rest of application
}
```

Then profile live application:
```bash
# CPU profile (30 seconds)
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# Heap profile
curl http://localhost:6060/debug/pprof/heap > heap.prof

# Goroutine profile
curl http://localhost:6060/debug/pprof/goroutine > goroutine.prof
```

---

## Further Reading

- [Go Blog: Profiling Go Programs](https://go.dev/blog/pprof)
- [pprof Documentation](https://github.com/google/pprof/blob/main/doc/README.md)
- [Go Performance Wiki](https://github.com/golang/go/wiki/Performance)
- [Benchstat Documentation](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [Dave Cheney: High Performance Go](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html)

---

## Summary

**Key Commands:**
```bash
# Run all benchmarks
go test -bench=. -benchmem

# Run regression tests
go test -v -run=TestPerformance

# Generate CPU profile
go test -bench=. -cpuprofile=cpu.prof -benchtime=10s
go tool pprof -http=:8080 cpu.prof

# Generate memory profile
go test -bench=. -memprofile=mem.prof -benchtime=10s
go tool pprof -http=:8080 mem.prof

# Compare benchmarks
benchstat baseline.txt new.txt
```

**Performance Targets:**
- Registry init: <10ms (acceptable: <50ms)
- Simple query: <1ms (acceptable: <5ms)
- Complex query depth 3: <20ms (acceptable: <50ms)
- Memory: <10MB for 50 resources (acceptable: <50MB)

**Current Performance:**
- ✅ All targets exceeded by 10-1000x
- ✅ Zero allocations on hot path
- ✅ Sub-microsecond query latency

For detailed performance analysis, see `PERFORMANCE.md`.
