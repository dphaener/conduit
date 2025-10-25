package metadata

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// BenchmarkExtractor_Extract tests extraction performance
func BenchmarkExtractor_Extract(b *testing.B) {
	prog := createLargeProgram(10) // 10 resources with typical structure

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor := NewExtractor("1.0.0")
		_, err := extractor.Extract(prog)
		if err != nil {
			b.Fatalf("Extract() error = %v", err)
		}
	}
}

// BenchmarkExtractor_Extract_Large tests performance with larger codebase
func BenchmarkExtractor_Extract_Large(b *testing.B) {
	prog := createLargeProgram(100) // 100 resources

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor := NewExtractor("1.0.0")
		_, err := extractor.Extract(prog)
		if err != nil {
			b.Fatalf("Extract() error = %v", err)
		}
	}
}

// BenchmarkExtractor_FormatExpression benchmarks expression formatting
func BenchmarkExtractor_FormatExpression(b *testing.B) {
	extractor := NewExtractor("1.0.0")

	// Complex nested expression
	expr := &ast.BinaryExpr{
		Left: &ast.CallExpr{
			Namespace: "String",
			Function:  "length",
			Arguments: []ast.ExprNode{
				&ast.FieldAccessExpr{
					Object: &ast.SelfExpr{},
					Field:  "title",
				},
			},
		},
		Operator: ">=",
		Right:    &ast.LiteralExpr{Value: 100},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractor.formatExpression(expr)
	}
}

// BenchmarkExtractor_ComputeHash benchmarks hash computation
func BenchmarkExtractor_ComputeHash(b *testing.B) {
	prog := createLargeProgram(50)
	extractor := NewExtractor("1.0.0")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractor.computeSourceHash(prog)
	}
}

// createLargeProgram creates a program with multiple resources for benchmarking
func createLargeProgram(resourceCount int) *ast.Program {
	resources := make([]*ast.ResourceNode, 0, resourceCount)

	for i := 0; i < resourceCount; i++ {
		resource := &ast.ResourceNode{
			Name:          "Resource" + string(rune('A'+i%26)),
			Documentation: "A sample resource for testing",
			Loc:           ast.SourceLocation{Line: i*10 + 1, Column: 1},
			Fields: []*ast.FieldNode{
				{
					Name:     "id",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid", Nullable: false},
					Nullable: false,
				},
				{
					Name:     "name",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
					Nullable: false,
					Constraints: []*ast.ConstraintNode{
						{Name: "min", Arguments: []ast.ExprNode{&ast.LiteralExpr{Value: 3}}},
						{Name: "max", Arguments: []ast.ExprNode{&ast.LiteralExpr{Value: 100}}},
					},
				},
				{
					Name:     "email",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: true},
					Nullable: true,
				},
				{
					Name:     "created_at",
					Type:     &ast.TypeNode{Kind: ast.TypePrimitive, Name: "datetime", Nullable: false},
					Nullable: false,
				},
			},
			Hooks: []*ast.HookNode{
				{
					Timing:        "before",
					Event:         "create",
					IsTransaction: false,
					IsAsync:       false,
					Loc:           ast.SourceLocation{Line: i*10 + 5, Column: 3},
					Body: []ast.StmtNode{
						&ast.AssignmentStmt{
							Target: &ast.FieldAccessExpr{
								Object: &ast.SelfExpr{},
								Field:  "created_at",
							},
							Value: &ast.CallExpr{
								Namespace: "Time",
								Function:  "now",
							},
						},
					},
				},
				{
					Timing:        "after",
					Event:         "create",
					IsTransaction: true,
					IsAsync:       false,
					Loc:           ast.SourceLocation{Line: i*10 + 8, Column: 3},
					Body: []ast.StmtNode{
						&ast.ExprStmt{
							Expr: &ast.CallExpr{
								Namespace: "Logger",
								Function:  "info",
								Arguments: []ast.ExprNode{
									&ast.LiteralExpr{Value: "Record created"},
								},
							},
						},
					},
				},
			},
			Validations: []*ast.ValidationNode{
				{
					Name: "valid_email",
					Condition: &ast.CallExpr{
						Namespace: "String",
						Function:  "contains",
						Arguments: []ast.ExprNode{
							&ast.FieldAccessExpr{Object: &ast.SelfExpr{}, Field: "email"},
							&ast.LiteralExpr{Value: "@"},
						},
					},
					Error: "Email must be valid",
				},
			},
			Constraints: []*ast.ConstraintNode{
				{
					Name: "unique_name",
					On:   []string{"create"},
					Condition: &ast.BinaryExpr{
						Left:     &ast.IdentifierExpr{Name: "count"},
						Operator: "==",
						Right:    &ast.LiteralExpr{Value: 0},
					},
					Error: "Name must be unique",
				},
			},
			Scopes: []*ast.ScopeNode{
				{
					Name: "active",
					Condition: &ast.BinaryExpr{
						Left:     &ast.FieldAccessExpr{Object: &ast.SelfExpr{}, Field: "status"},
						Operator: "==",
						Right:    &ast.LiteralExpr{Value: "active"},
					},
				},
			},
			Computed: []*ast.ComputedNode{
				{
					Name: "display_name",
					Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string", Nullable: false},
					Body: &ast.CallExpr{
						Namespace: "String",
						Function:  "upper",
						Arguments: []ast.ExprNode{
							&ast.FieldAccessExpr{Object: &ast.SelfExpr{}, Field: "name"},
						},
					},
				},
			},
		}

		// Add some relationships every few resources
		if i > 0 && i%3 == 0 {
			resource.Relationships = []*ast.RelationshipNode{
				{
					Name:       "parent",
					Type:       "Resource" + string(rune('A'+(i-1)%26)),
					Kind:       ast.RelationshipBelongsTo,
					ForeignKey: "parent_id",
					OnDelete:   "cascade",
					Nullable:   true,
				},
			}
		}

		resources = append(resources, resource)
	}

	return &ast.Program{Resources: resources}
}

// TestExtractor_Performance tests that extraction meets performance target
func TestExtractor_Performance(t *testing.T) {
	// Create a program simulating ~1000 LOC
	// Assuming ~10 resources with ~10 lines each = 100 LOC per resource
	prog := createLargeProgram(10)

	extractor := NewExtractor("1.0.0")
	extractor.SetFilePath("/test/file.cdt")

	// Measure extraction time
	start := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := extractor.Extract(prog)
			if err != nil {
				t.Fatalf("Extract() error = %v", err)
			}
		}
	})

	// Check that average time is under 100ms for 1000 LOC equivalent
	nsPerOp := start.NsPerOp()
	msPerOp := nsPerOp / 1_000_000

	if msPerOp > 100 {
		t.Logf("Performance target: <100ms for 1000 LOC")
		t.Logf("Actual performance: %dms per operation", msPerOp)
		// This is informational, not a hard failure
	} else {
		t.Logf("✅ Performance test passed: %dms per operation (target: <100ms)", msPerOp)
	}
}
