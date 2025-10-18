package benchmark

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/codegen"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
	"github.com/conduit-lang/conduit/internal/compiler/parser"
	"github.com/conduit-lang/conduit/internal/compiler/typechecker"
)

// BenchmarkLexer_1000LOC benchmarks lexer performance on 1000 lines of code
func BenchmarkLexer_1000LOC(b *testing.B) {
	source := Generate1000LOC()
	loc := CountLOC(source)
	b.Logf("Benchmarking lexer with %d LOC", loc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lex := lexer.New(source)
		_, _ = lex.ScanTokens()
	}
}

// BenchmarkParser_1000LOC benchmarks parser performance on 1000 lines of code
func BenchmarkParser_1000LOC(b *testing.B) {
	source := Generate1000LOC()
	loc := CountLOC(source)
	b.Logf("Benchmarking parser with %d LOC", loc)

	// Pre-lex the source
	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := parser.New(tokens)
		_, _ = p.Parse()
	}
}

// BenchmarkTypeChecker_PerResource benchmarks type checker on single resource
func BenchmarkTypeChecker_PerResource(b *testing.B) {
	source := ComplexResource()

	// Parse source
	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()
	p := parser.New(tokens)
	prog, _ := p.Parse()

	resourceCount := len(prog.Resources)
	b.Logf("Benchmarking type checker with %d resources", resourceCount)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc := typechecker.NewTypeChecker()
		_ = tc.CheckProgram(prog)
	}
}

// BenchmarkCodeGenerator_50Resources benchmarks code generation for 50 resources
func BenchmarkCodeGenerator_50Resources(b *testing.B) {
	source := Generate50Resources()

	// Parse and type check
	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()
	p := parser.New(tokens)
	prog, _ := p.Parse()
	tc := typechecker.NewTypeChecker()
	_ = tc.CheckProgram(prog)

	resourceCount := len(prog.Resources)
	b.Logf("Benchmarking code generator with %d resources", resourceCount)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen := codegen.NewGenerator()
		_, _ = gen.GenerateProgram(prog)
	}
}

// BenchmarkFullPipeline_TypicalProject benchmarks full compilation pipeline
func BenchmarkFullPipeline_TypicalProject(b *testing.B) {
	source := TypicalProject()
	resourceCount := CountResources(source)
	b.Logf("Benchmarking full pipeline with %d resources", resourceCount)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Lexer
		lex := lexer.New(source)
		tokens, _ := lex.ScanTokens()

		// Parser
		p := parser.New(tokens)
		prog, _ := p.Parse()

		// Type Checker
		tc := typechecker.NewTypeChecker()
		_ = tc.CheckProgram(prog)

		// Code Generator
		gen := codegen.NewGenerator()
		_, _ = gen.GenerateProgram(prog)
	}
}

// BenchmarkLexer_SimpleResource benchmarks lexer on minimal resource
func BenchmarkLexer_SimpleResource(b *testing.B) {
	source := SimpleResource()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lex := lexer.New(source)
		_, _ = lex.ScanTokens()
	}
}

// BenchmarkParser_SimpleResource benchmarks parser on minimal resource
func BenchmarkParser_SimpleResource(b *testing.B) {
	source := SimpleResource()
	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := parser.New(tokens)
		_, _ = p.Parse()
	}
}

// BenchmarkTypeChecker_SimpleResource benchmarks type checker on minimal resource
func BenchmarkTypeChecker_SimpleResource(b *testing.B) {
	source := SimpleResource()
	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()
	p := parser.New(tokens)
	prog, _ := p.Parse()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc := typechecker.NewTypeChecker()
		_ = tc.CheckProgram(prog)
	}
}

// BenchmarkCodeGenerator_SimpleResource benchmarks code gen on minimal resource
func BenchmarkCodeGenerator_SimpleResource(b *testing.B) {
	source := SimpleResource()
	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()
	p := parser.New(tokens)
	prog, _ := p.Parse()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen := codegen.NewGenerator()
		_, _ = gen.GenerateProgram(prog)
	}
}

// BenchmarkLexer_WithHooks benchmarks lexer on resource with hooks
func BenchmarkLexer_WithHooks(b *testing.B) {
	source := ResourceWithHooks()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lex := lexer.New(source)
		_, _ = lex.ScanTokens()
	}
}

// BenchmarkParser_WithHooks benchmarks parser on resource with hooks
func BenchmarkParser_WithHooks(b *testing.B) {
	source := ResourceWithHooks()
	lex := lexer.New(source)
	tokens, _ := lex.ScanTokens()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := parser.New(tokens)
		_, _ = p.Parse()
	}
}

// BenchmarkFullPipeline_WithRelationships benchmarks full pipeline with relationships
func BenchmarkFullPipeline_WithRelationships(b *testing.B) {
	source := ResourceWithRelationships()

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
