package codegen

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestGenerateListHandler_Phase3Features(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{Name: "id", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int"}, Nullable: false},
			{Name: "title", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
			{Name: "content", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "text"}, Nullable: false},
			{Name: "authorId", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int"}, Nullable: false},
			{Name: "published", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "bool"}, Nullable: false},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateHandlers([]*ast.ResourceNode{resource}, "example.com/testapp")
	if err != nil {
		t.Fatalf("GenerateHandlers failed: %v", err)
	}

	// Phase 3: Check query package import
	if !strings.Contains(code, "github.com/conduit-lang/conduit/pkg/web/query") {
		t.Error("Generated code should import query package")
	}

	// Phase 3: Check ParseInclude
	if !strings.Contains(code, "query.ParseInclude(r)") {
		t.Error("Generated code should parse include parameter")
	}

	// Phase 3: Check ParseFields
	if !strings.Contains(code, "query.ParseFields(r)") {
		t.Error("Generated code should parse fields parameter")
	}

	// Phase 3: Check ParseFilter
	if !strings.Contains(code, "query.ParseFilter(r)") {
		t.Error("Generated code should parse filter parameter")
	}

	// Phase 3: Check ParseSort
	if !strings.Contains(code, "query.ParseSort(r)") {
		t.Error("Generated code should parse sort parameter")
	}

	// Phase 3: Check validFields generation
	if !strings.Contains(code, "validFields := []string{") {
		t.Error("Generated code should define validFields slice")
	}

	// Phase 3: Check field names are in snake_case
	expectedFields := []string{"id", "title", "content", "author_id", "published"}
	for _, field := range expectedFields {
		pattern := "\"" + field + "\","
		if !strings.Contains(code, pattern) {
			t.Errorf("Generated code should contain field %s in validFields", field)
		}
	}

	// Phase 3: Check BuildFilterClause
	if !strings.Contains(code, "query.BuildFilterClause(filters,") {
		t.Error("Generated code should call BuildFilterClause")
	}

	// Phase 3: Check BuildSortClause
	if !strings.Contains(code, "query.BuildSortClause(sorts,") {
		t.Error("Generated code should call BuildSortClause")
	}

	// Phase 3: Check custom query building
	if !strings.Contains(code, "baseQuery := \"SELECT * FROM posts\"") {
		t.Error("Generated code should build custom SQL query")
	}

	// Phase 3: Check filter WHERE clause application
	if !strings.Contains(code, "if whereClause != \"\" {") {
		t.Error("Generated code should conditionally apply WHERE clause")
	}

	// Phase 3: Check sort ORDER BY clause application
	if !strings.Contains(code, "if orderByClause != \"\" {") {
		t.Error("Generated code should conditionally apply ORDER BY clause")
	}

	// Phase 3: Check parameter indexing for pagination
	if !strings.Contains(code, "paramIndex := len(filterArgs) + 1") {
		t.Error("Generated code should calculate parameter index for pagination")
	}

	// Phase 3: Check ApplySparseFieldsets
	if !strings.Contains(code, "response.ApplySparseFieldsets(data, fields)") {
		t.Error("Generated code should apply sparse fieldsets")
	}

	// Phase 3: Check sparse fieldsets conditional
	if !strings.Contains(code, "if len(fields) > 0 {") {
		t.Error("Generated code should conditionally apply sparse fieldsets")
	}

	// Phase 3: Check includes TODO comment
	if !strings.Contains(code, "// TODO: Phase 3 - Load relationships if includes is not empty") {
		t.Error("Generated code should have TODO comment for includes")
	}

	// Phase 3: Check ScanRow usage
	if !strings.Contains(code, "item.ScanRow(rows)") {
		t.Error("Generated code should use ScanRow to scan query results")
	}

	// Phase 3: Check filtered count query
	if !strings.Contains(code, "countQuery := \"SELECT COUNT(*) FROM posts\"") {
		t.Error("Generated code should build count query with filters")
	}

	// Verify error handling for invalid filter fields
	if !strings.Contains(code, "response.RenderJSONAPIError(w, http.StatusBadRequest, err)") {
		t.Error("Generated code should handle filter validation errors")
	}
}
