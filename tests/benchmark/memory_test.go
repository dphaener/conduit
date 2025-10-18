package benchmark

import (
	"runtime"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/codegen"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
	"github.com/conduit-lang/conduit/internal/compiler/parser"
	"github.com/conduit-lang/conduit/internal/compiler/typechecker"
)

// Memory target constants
const (
	MaxMemoryUsage_Per5000LOC = 50 // MB
)

// TestMemory_5000LOC tests memory usage for 5000 lines of code
func TestMemory_5000LOC(t *testing.T) {
	source := Generate5000LOC()
	loc := CountLOC(source)
	t.Logf("Testing memory usage with %d LOC", loc)

	// Force garbage collection before measurement
	runtime.GC()

	// Get baseline memory
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	baselineAlloc := memBefore.Alloc

	// Run full compilation pipeline
	lex := lexer.New(source)
	tokens, lexErrors := lex.ScanTokens()
	if len(lexErrors) > 0 {
		t.Fatalf("Lexer errors: %v", lexErrors)
	}

	p := parser.New(tokens)
	prog, parseErrors := p.Parse()
	if len(parseErrors) > 0 {
		t.Fatalf("Parser errors: %v", parseErrors)
	}

	tc := typechecker.NewTypeChecker()
	typeErrors := tc.CheckProgram(prog)
	if len(typeErrors) > 0 {
		t.Fatalf("Type checker errors: %v", typeErrors)
	}

	gen := codegen.NewGenerator()
	_, err := gen.GenerateProgram(prog)
	if err != nil {
		t.Fatalf("Code generation error: %v", err)
	}

	// Get memory after compilation
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	allocAfter := memAfter.Alloc

	// Calculate memory used (safe signed arithmetic)
	memoryUsedBytes := int64(allocAfter) - int64(baselineAlloc)
	if memoryUsedBytes < 0 {
		t.Logf("Memory decreased after GC: baseline=%d, after=%d", baselineAlloc, allocAfter)
		memoryUsedBytes = 0
	}
	memoryUsedMB := float64(memoryUsedBytes) / 1024 / 1024

	t.Logf("Memory used: %.2f MB (target: <%d MB)", memoryUsedMB, MaxMemoryUsage_Per5000LOC)

	if memoryUsedMB > MaxMemoryUsage_Per5000LOC {
		t.Errorf("Memory usage too high: %.2f MB (target: <%d MB)", memoryUsedMB, MaxMemoryUsage_Per5000LOC)
	}
}

// TestMemory_SimpleResource tests memory usage for simple resource
func TestMemory_SimpleResource(t *testing.T) {
	source := SimpleResource()

	runtime.GC()

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	baselineAlloc := memBefore.Alloc

	// Compile
	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()
	p := parser.New(tokens)
	prog, _ := p.Parse()
	tc := typechecker.NewTypeChecker()
	_ = tc.CheckProgram(prog)
	gen := codegen.NewGenerator()
	_, _ = gen.GenerateProgram(prog)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	allocAfter := memAfter.Alloc

	memoryUsedBytes := int64(allocAfter) - int64(baselineAlloc)
	if memoryUsedBytes < 0 {
		t.Logf("Memory decreased after GC: baseline=%d, after=%d", baselineAlloc, allocAfter)
		memoryUsedBytes = 0
	}
	memoryUsedMB := float64(memoryUsedBytes) / 1024 / 1024

	t.Logf("Memory used for simple resource: %.2f MB", memoryUsedMB)

	// Simple resource should use minimal memory (<5MB)
	if memoryUsedMB > 5.0 {
		t.Errorf("Memory usage too high for simple resource: %.2f MB", memoryUsedMB)
	}
}

// TestMemory_TypicalProject tests memory usage for typical project
func TestMemory_TypicalProject(t *testing.T) {
	source := TypicalProject()
	resourceCount := CountResources(source)
	t.Logf("Testing memory usage with %d resources", resourceCount)

	runtime.GC()

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	baselineAlloc := memBefore.Alloc

	// Compile
	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()
	p := parser.New(tokens)
	prog, _ := p.Parse()
	tc := typechecker.NewTypeChecker()
	_ = tc.CheckProgram(prog)
	gen := codegen.NewGenerator()
	_, _ = gen.GenerateProgram(prog)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	allocAfter := memAfter.Alloc

	memoryUsedBytes := int64(allocAfter) - int64(baselineAlloc)
	if memoryUsedBytes < 0 {
		t.Logf("Memory decreased after GC: baseline=%d, after=%d", baselineAlloc, allocAfter)
		memoryUsedBytes = 0
	}
	memoryUsedMB := float64(memoryUsedBytes) / 1024 / 1024

	t.Logf("Memory used for typical project: %.2f MB", memoryUsedMB)

	// Typical project (10 resources) should use reasonable memory (<15MB)
	if memoryUsedMB > 15.0 {
		t.Errorf("Memory usage too high for typical project: %.2f MB", memoryUsedMB)
	}
}

// BenchmarkMemory_Lexer benchmarks lexer memory allocations
func BenchmarkMemory_Lexer(b *testing.B) {
	source := Generate1000LOC()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		lex := lexer.New(source)
		_, _ = lex.ScanTokens()
	}
}

// BenchmarkMemory_Parser benchmarks parser memory allocations
func BenchmarkMemory_Parser(b *testing.B) {
	source := Generate1000LOC()
	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := parser.New(tokens)
		_, _ = p.Parse()
	}
}

// BenchmarkMemory_TypeChecker benchmarks type checker memory allocations
func BenchmarkMemory_TypeChecker(b *testing.B) {
	source := TypicalProject()
	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()
	p := parser.New(tokens)
	prog, _ := p.Parse()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tc := typechecker.NewTypeChecker()
		_ = tc.CheckProgram(prog)
	}
}

// BenchmarkMemory_CodeGenerator benchmarks code generator memory allocations
func BenchmarkMemory_CodeGenerator(b *testing.B) {
	source := TypicalProject()
	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()
	p := parser.New(tokens)
	prog, _ := p.Parse()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gen := codegen.NewGenerator()
		_, _ = gen.GenerateProgram(prog)
	}
}

// BenchmarkMemory_FullPipeline benchmarks full pipeline memory allocations
func BenchmarkMemory_FullPipeline(b *testing.B) {
	source := TypicalProject()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		lex := lexer.New(source)
		tokens, _ := lex.ScanTokens()
		p := parser.New(tokens)
		prog, _ := p.Parse()
		tc := typechecker.NewTypeChecker()
		_ = tc.CheckProgram(prog)
		gen := codegen.NewGenerator()
		_, _ = gen.GenerateProgram(prog)
	}
}

// TestMemory_NoLeaks tests for memory leaks by running multiple compilations
func TestMemory_NoLeaks(t *testing.T) {
	source := SimpleResource()

	runtime.GC()

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	baselineAlloc := memBefore.Alloc

	// Run compilation 100 times
	for i := 0; i < 100; i++ {
		lex := lexer.New(source)
		tokens, _ := lex.ScanTokens()
		p := parser.New(tokens)
		prog, _ := p.Parse()
		tc := typechecker.NewTypeChecker()
		_ = tc.CheckProgram(prog)
		gen := codegen.NewGenerator()
		_, _ = gen.GenerateProgram(prog)
	}

	// Force garbage collection
	runtime.GC()

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	allocAfter := memAfter.Alloc

	memoryUsedBytes := int64(allocAfter) - int64(baselineAlloc)
	if memoryUsedBytes < 0 {
		t.Logf("Memory decreased after GC: baseline=%d, after=%d", baselineAlloc, allocAfter)
		memoryUsedBytes = 0
	}
	memoryUsedMB := float64(memoryUsedBytes) / 1024 / 1024

	t.Logf("Memory used after 100 compilations: %.2f MB", memoryUsedMB)

	// After GC, memory should not grow significantly
	// Allow up to 10MB for retained data structures
	if memoryUsedMB > 10.0 {
		t.Errorf("Possible memory leak: %.2f MB after 100 compilations", memoryUsedMB)
	}
}

// TestMemory_LargeSource tests memory scaling with source size
func TestMemory_LargeSource(t *testing.T) {
	sizes := []int{100, 500, 1000, 2000, 5000}

	for _, size := range sizes {
		source := GenerateLargeSource(size)
		loc := CountLOC(source)

		runtime.GC()

		var memBefore runtime.MemStats
		runtime.ReadMemStats(&memBefore)
		baselineAlloc := memBefore.Alloc

		// Compile
		lex := lexer.New(source)
		tokens, _ := lex.ScanTokens()
		p := parser.New(tokens)
		prog, _ := p.Parse()
		tc := typechecker.NewTypeChecker()
		_ = tc.CheckProgram(prog)
		gen := codegen.NewGenerator()
		_, _ = gen.GenerateProgram(prog)

		var memAfter runtime.MemStats
		runtime.ReadMemStats(&memAfter)
		allocAfter := memAfter.Alloc

		memoryUsedBytes := int64(allocAfter) - int64(baselineAlloc)
		if memoryUsedBytes < 0 {
			t.Logf("Memory decreased after GC: baseline=%d, after=%d", baselineAlloc, allocAfter)
			memoryUsedBytes = 0
		}
		memoryUsedMB := float64(memoryUsedBytes) / 1024 / 1024
		memoryPerLOC := memoryUsedMB / float64(loc) * 1000 // KB per LOC

		t.Logf("%d LOC: %.2f MB (%.2f KB/LOC)", loc, memoryUsedMB, memoryPerLOC)

		// Memory should scale roughly linearly (< 15 KB per LOC)
		if memoryPerLOC > 15.0 {
			t.Errorf("Memory scaling poor at %d LOC: %.2f KB/LOC", loc, memoryPerLOC)
		}
	}
}
