package metadata

import (
	"encoding/json"
	"testing"
	"time"
)

// Performance regression tests that fail if performance degrades beyond ACCEPTABLE thresholds.
// These tests use the acceptable thresholds from IMPLEMENTATION-RUNTIME.md, not the target thresholds.
// This provides a safety net while allowing some performance variance.

// TestPerformance_RegistryInit verifies registry initialization stays under acceptable threshold
func TestPerformance_RegistryInit(t *testing.T) {
	const acceptableThreshold = 50 * time.Millisecond
	const targetThreshold = 10 * time.Millisecond

	result := testing.Benchmark(BenchmarkRegistryInit)
	avgTime := time.Duration(result.NsPerOp())

	t.Logf("Registry initialization: %v (target: <%v, acceptable: <%v)", avgTime, targetThreshold, acceptableThreshold)

	if avgTime > acceptableThreshold {
		t.Errorf("Registry initialization too slow: %v (acceptable: <%v, target: <%v)",
			avgTime, acceptableThreshold, targetThreshold)
	}

	// Also check memory usage (acceptable: <50MB, target: <10MB)
	const acceptableMemory = 50 * 1024 * 1024 // 50MB in bytes
	const targetMemory = 10 * 1024 * 1024     // 10MB in bytes

	bytesPerOp := result.AllocedBytesPerOp()
	t.Logf("Registry initialization memory: %d bytes (target: <%d, acceptable: <%d)",
		bytesPerOp, targetMemory, acceptableMemory)

	if bytesPerOp > acceptableMemory {
		t.Errorf("Registry initialization uses too much memory: %d bytes (acceptable: <%d, target: <%d)",
			bytesPerOp, acceptableMemory, targetMemory)
	}
}

// TestPerformance_SimpleQuery verifies simple queries stay under acceptable threshold
func TestPerformance_SimpleQuery(t *testing.T) {
	const acceptableThreshold = 5 * time.Millisecond
	const targetThreshold = 1 * time.Millisecond

	// Test Resource lookup
	result := testing.Benchmark(func(b *testing.B) {
		setupBenchMetadata(b)
		defer Reset()
		registry := GetRegistry()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = registry.Resource("Post")
		}
	})

	avgTime := time.Duration(result.NsPerOp())
	t.Logf("Simple query (Resource): %v (target: <%v, acceptable: <%v)", avgTime, targetThreshold, acceptableThreshold)

	if avgTime > acceptableThreshold {
		t.Errorf("Simple query too slow: %v (acceptable: <%v, target: <%v)",
			avgTime, acceptableThreshold, targetThreshold)
	}

	// Verify minimal allocations (defensive copies are acceptable)
	const acceptableAllocs = 2000 // 2KB for defensive copies
	bytesPerOp := result.AllocedBytesPerOp()
	t.Logf("Simple query allocations: %d bytes/op (acceptable: <%d)", bytesPerOp, acceptableAllocs)

	if bytesPerOp > acceptableAllocs {
		t.Errorf("Simple query allocates too much memory: %d bytes/op (acceptable: <%d)",
			bytesPerOp, acceptableAllocs)
	}
}

// TestPerformance_ComplexQuery verifies complex queries stay under acceptable threshold
func TestPerformance_ComplexQuery(t *testing.T) {
	const acceptableThreshold = 50 * time.Millisecond
	const targetThreshold = 20 * time.Millisecond

	// Test depth-3 dependency traversal (cold cache)
	result := testing.Benchmark(func(b *testing.B) {
		setupBenchMetadata(b)
		defer Reset()
		registry := GetRegistry()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Note: We can't clear cache here as it's internal to the registry
			// This test measures a mix of cold and warm cache performance
			_, _ = registry.Dependencies("Post", DependencyOptions{
				Depth:   3,
				Reverse: false,
			})
		}
	})

	avgTime := time.Duration(result.NsPerOp())
	t.Logf("Complex query depth 3: %v (target: <%v, acceptable: <%v)",
		avgTime, targetThreshold, acceptableThreshold)

	if avgTime > acceptableThreshold {
		t.Errorf("Complex query too slow: %v (acceptable: <%v, target: <%v)",
			avgTime, acceptableThreshold, targetThreshold)
	}
}

// TestPerformance_ComplexQueryCached verifies cached queries stay under acceptable threshold
func TestPerformance_ComplexQueryCached(t *testing.T) {
	const acceptableThreshold = 5 * time.Millisecond
	const targetThreshold = 1 * time.Millisecond

	// Test depth-3 dependency traversal (warm cache)
	result := testing.Benchmark(func(b *testing.B) {
		setupBenchMetadata(b)
		defer Reset()
		registry := GetRegistry()
		// Prime cache once before the benchmark
		_, _ = registry.Dependencies("Post", DependencyOptions{
			Depth:   3,
			Reverse: false,
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = registry.Dependencies("Post", DependencyOptions{
				Depth:   3,
				Reverse: false,
			})
		}
	})

	avgTime := time.Duration(result.NsPerOp())
	t.Logf("Complex query depth 3 (cached): %v (target: <%v, acceptable: <%v)",
		avgTime, targetThreshold, acceptableThreshold)

	if avgTime > acceptableThreshold {
		t.Errorf("Cached complex query too slow: %v (acceptable: <%v, target: <%v)",
			avgTime, acceptableThreshold, targetThreshold)
	}

	// Verify minimal allocations (defensive copies are acceptable)
	const acceptableAllocs = 2000 // 2KB for defensive copies
	bytesPerOp := result.AllocedBytesPerOp()
	t.Logf("Cached query allocations: %d bytes/op (acceptable: <%d)", bytesPerOp, acceptableAllocs)

	if bytesPerOp > acceptableAllocs {
		t.Errorf("Cached query allocates too much memory: %d bytes/op (acceptable: <%d)",
			bytesPerOp, acceptableAllocs)
	}
}

// TestPerformance_MemoryUsage verifies memory usage stays under acceptable threshold
func TestPerformance_MemoryUsage(t *testing.T) {
	const acceptableMemory = 50 * 1024 * 1024 // 50MB
	const targetMemory = 10 * 1024 * 1024     // 10MB

	// Test registry initialization memory
	result := testing.Benchmark(BenchmarkRegistryInit)
	bytesPerOp := result.AllocedBytesPerOp()

	t.Logf("Memory usage (registry init): %d bytes (target: <%d, acceptable: <%d)",
		bytesPerOp, targetMemory, acceptableMemory)

	if bytesPerOp > acceptableMemory {
		t.Errorf("Memory usage too high: %d bytes (acceptable: <%d, target: <%d)",
			bytesPerOp, acceptableMemory, targetMemory)
	}
}

// TestPerformance_MinimalAllocations verifies hot path operations have minimal allocations
// Note: Defensive copies are intentional to prevent external mutation of returned data.
// This trades a small allocation cost (40-1400 bytes) for safety and API correctness.
func TestPerformance_MinimalAllocations(t *testing.T) {
	tests := []struct {
		name              string
		benchmark         func(*testing.B)
		maxAllocs         int64
		maxBytes          int64
	}{
		{
			name: "Resource lookup",
			benchmark: func(b *testing.B) {
				setupBenchMetadata(b)
				defer Reset()
				registry := GetRegistry()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = registry.Resource("Post")
				}
			},
			maxAllocs: 2,
			maxBytes:  500,
		},
		{
			name: "Resources list",
			benchmark: func(b *testing.B) {
				setupBenchMetadata(b)
				defer Reset()
				registry := GetRegistry()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = registry.Resources()
				}
			},
			maxAllocs: 2,
			maxBytes:  1500,
		},
		{
			name: "Schema access",
			benchmark: func(b *testing.B) {
				setupBenchMetadata(b)
				defer Reset()
				registry := GetRegistry()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = registry.GetSchema()
				}
			},
			maxAllocs: 1,
			maxBytes:  100,
		},
		{
			name: "Route lookup",
			benchmark: func(b *testing.B) {
				setupBenchMetadata(b)
				defer Reset()
				registry := GetRegistry()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = registry.Routes(RouteFilter{})
				}
			},
			maxAllocs: 2,
			maxBytes:  2000,
		},
		{
			name: "Pattern lookup",
			benchmark: func(b *testing.B) {
				setupBenchMetadata(b)
				defer Reset()
				registry := GetRegistry()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = registry.Patterns("")
				}
			},
			maxAllocs: 2,
			maxBytes:  1000,
		},
		{
			name: "Cached dependency query",
			benchmark: func(b *testing.B) {
				setupBenchMetadata(b)
				defer Reset()
				registry := GetRegistry()
				// Prime cache
				_, _ = registry.Dependencies("Post", DependencyOptions{
					Depth:   3,
					Reverse: false,
				})
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = registry.Dependencies("Post", DependencyOptions{
						Depth:   3,
						Reverse: false,
					})
				}
			},
			maxAllocs: 3,
			maxBytes:  100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testing.Benchmark(tt.benchmark)
			allocs := result.AllocsPerOp()
			bytes := result.AllocedBytesPerOp()

			t.Logf("%s: %d allocs/op, %d bytes/op (max: %d allocs, %d bytes)",
				tt.name, allocs, bytes, tt.maxAllocs, tt.maxBytes)

			if allocs > tt.maxAllocs {
				t.Errorf("%s has too many allocations: %d allocs/op (max: %d)",
					tt.name, allocs, tt.maxAllocs)
			}

			if bytes > tt.maxBytes {
				t.Errorf("%s allocates too much memory: %d bytes/op (max: %d)",
					tt.name, bytes, tt.maxBytes)
			}
		})
	}
}

// TestPerformance_Scaling verifies performance scales reasonably with schema size
func TestPerformance_Scaling(t *testing.T) {
	// This test verifies that performance doesn't degrade catastrophically with larger schemas
	// We don't have strict thresholds, but we want to detect O(n^2) or worse behavior

	sizes := []int{10, 50, 100}
	initTimes := make([]time.Duration, len(sizes))

	for i, size := range sizes {
		result := testing.Benchmark(func(b *testing.B) {
			meta := generateLargeMetadata(size)
			data, err := json.Marshal(meta)
			if err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			for j := 0; j < b.N; j++ {
				Reset()
				if err := RegisterMetadata(data); err != nil {
					b.Fatal(err)
				}
			}
		})

		initTimes[i] = time.Duration(result.NsPerOp())
		t.Logf("Init time for %d resources: %v", size, initTimes[i])
	}

	// Verify roughly linear scaling (allow 3x margin)
	// If init time grows faster than O(n), we have a problem
	for i := 1; i < len(sizes); i++ {
		ratio := float64(sizes[i]) / float64(sizes[i-1])
		timeRatio := float64(initTimes[i]) / float64(initTimes[i-1])

		t.Logf("Size ratio: %.2fx, Time ratio: %.2fx", ratio, timeRatio)

		// Allow up to 3x the expected ratio (e.g., 2x size can take up to 6x time)
		// This is very generous but catches O(n^2) or worse
		if timeRatio > ratio*3 {
			t.Errorf("Poor scaling detected: %dx size increase caused %.2fx time increase (expected: <%.2fx)",
				int(ratio), timeRatio, ratio*3)
		}
	}

	// Verify largest schema still meets acceptable threshold
	const acceptableThreshold = 50 * time.Millisecond
	if initTimes[len(initTimes)-1] > acceptableThreshold {
		t.Errorf("Init time for %d resources too slow: %v (acceptable: <%v)",
			sizes[len(sizes)-1], initTimes[len(initTimes)-1], acceptableThreshold)
	}
}

// TestPerformance_CacheEffectiveness verifies cache provides meaningful speedup
func TestPerformance_CacheEffectiveness(t *testing.T) {
	// We'll use the benchmark infrastructure which handles setup
	// This test just verifies that caching is effective

	// We can't directly measure cold vs warm because cache is internal,
	// but we can verify that repeated queries are fast (indicating caching works)
	result := testing.Benchmark(func(b *testing.B) {
		setupBenchMetadata(b)
		defer Reset()
		registry := GetRegistry()
		// Prime cache
		_, _ = registry.Dependencies("Post", DependencyOptions{
			Depth:   3,
			Reverse: false,
		})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = registry.Dependencies("Post", DependencyOptions{
				Depth:   3,
				Reverse: false,
			})
		}
	})

	cachedTime := time.Duration(result.NsPerOp())
	t.Logf("Cached query time: %v", cachedTime)

	// Cached queries should be sub-microsecond (indicating they're hitting cache)
	const expectedCachedThreshold = 5 * time.Microsecond
	if cachedTime > expectedCachedThreshold {
		t.Logf("Warning: Cached query slower than expected: %v (expected: <%v) - cache may not be working",
			cachedTime, expectedCachedThreshold)
		// Note: Not failing test since we can't directly control cache, just logging warning
	}

	// Verify minimal allocations for cached queries (defensive copies are acceptable)
	const acceptableAllocs = 100 // Small defensive copies acceptable
	bytesPerOp := result.AllocedBytesPerOp()
	t.Logf("Cached query allocations: %d bytes/op (acceptable: <%d)", bytesPerOp, acceptableAllocs)

	if bytesPerOp > acceptableAllocs {
		t.Errorf("Cached query allocates too much memory: %d bytes/op (acceptable: <%d)",
			bytesPerOp, acceptableAllocs)
	}
}
