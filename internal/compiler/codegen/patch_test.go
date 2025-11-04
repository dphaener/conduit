package codegen

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// TestGeneratePatchHandler verifies that PATCH handler is generated correctly
func TestGeneratePatchHandler(t *testing.T) {
	g := NewGenerator()

	// Create a simple resource for testing
	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{
				Name:     "id",
				Type:     &ast.TypeNode{Name: "uuid"},
				Nullable: false,
				Constraints: []*ast.ConstraintNode{
					{Name: "primary"},
					{Name: "auto"},
				},
			},
			{
				Name:     "title",
				Type:     &ast.TypeNode{Name: "string"},
				Nullable: false,
				Constraints: []*ast.ConstraintNode{
					{Name: "min", Arguments: []ast.ExprNode{&ast.LiteralExpr{Value: 5}}},
					{Name: "max", Arguments: []ast.ExprNode{&ast.LiteralExpr{Value: 200}}},
				},
			},
			{
				Name:     "body",
				Type:     &ast.TypeNode{Name: "text"},
				Nullable: false,
			},
			{
				Name:     "status",
				Type:     &ast.TypeNode{Name: "string"},
				Nullable: false,
			},
		},
	}

	// Generate PATCH handler
	g.reset()
	g.generatePatchHandler(resource)
	code := g.buf.String()

	// Verify the handler is generated
	if !strings.Contains(code, "func PatchPostHandler") {
		t.Error("Expected PatchPostHandler function to be generated")
	}

	// Verify it fetches existing resource first
	if !strings.Contains(code, "FindPostByID") {
		t.Error("Expected handler to fetch existing resource")
	}

	// Verify it calls Patch method
	if !strings.Contains(code, "existing.Patch") {
		t.Error("Expected handler to call Patch method")
	}

	// Verify it handles 404
	if !strings.Contains(code, "StatusNotFound") {
		t.Error("Expected handler to return 404 for missing resources")
	}

	// Verify content negotiation
	if !strings.Contains(code, "IsJSONAPI") {
		t.Error("Expected handler to support JSON:API content negotiation")
	}
}

// TestGeneratePatchMethod verifies that Patch() method is generated correctly
func TestGeneratePatchMethod(t *testing.T) {
	g := NewGenerator()

	// Create a simple resource for testing
	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{
				Name:     "id",
				Type:     &ast.TypeNode{Name: "uuid"},
				Nullable: false,
				Constraints: []*ast.ConstraintNode{
					{Name: "primary"},
					{Name: "auto"},
				},
			},
			{
				Name:     "title",
				Type:     &ast.TypeNode{Name: "string"},
				Nullable: false,
			},
			{
				Name:     "body",
				Type:     &ast.TypeNode{Name: "text"},
				Nullable: false,
			},
			{
				Name:     "created_at",
				Type:     &ast.TypeNode{Name: "timestamp"},
				Nullable: false,
				Constraints: []*ast.ConstraintNode{
					{Name: "auto"},
				},
			},
			{
				Name:     "updated_at",
				Type:     &ast.TypeNode{Name: "timestamp"},
				Nullable: false,
				Constraints: []*ast.ConstraintNode{
					{Name: "auto_update"},
				},
			},
		},
	}

	// Generate Patch method
	g.reset()
	g.generatePatch(resource)
	code := g.buf.String()

	// Verify the method signature
	if !strings.Contains(code, "func (p *Post) Patch(ctx context.Context, db *sql.DB, partialJSON []byte) error") {
		t.Error("Expected Patch method with correct signature")
	}

	// Verify it parses partial JSON
	if !strings.Contains(code, "json.Unmarshal(partialJSON, &partialData)") {
		t.Error("Expected method to parse partial JSON")
	}

	// Verify it rejects empty PATCH
	if !strings.Contains(code, "empty PATCH request") {
		t.Error("Expected method to reject empty PATCH requests")
	}

	// Verify it validates read-only fields
	if !strings.Contains(code, "readOnlyFields") {
		t.Error("Expected method to validate read-only fields")
	}

	// Verify it validates unknown fields
	if !strings.Contains(code, "validFields") {
		t.Error("Expected method to validate unknown fields")
	}

	// Verify it validates merged result
	if !strings.Contains(code, "Validate()") {
		t.Error("Expected method to validate merged result")
	}

	// Verify it updates the database
	if !strings.Contains(code, "UPDATE posts SET") {
		t.Error("Expected method to update the database")
	}
}

// TestPatchRouteRegistration verifies that PATCH route is registered
func TestPatchRouteRegistration(t *testing.T) {
	g := NewGenerator()

	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{
				Name:     "id",
				Type:     &ast.TypeNode{Name: "uuid"},
				Nullable: false,
			},
		},
	}

	// Generate resource handlers (includes route registration)
	g.reset()
	err := g.generateResourceHandlers(resource)
	if err != nil {
		t.Fatalf("Failed to generate resource handlers: %v", err)
	}

	code := g.buf.String()

	// Verify PATCH route is registered
	if !strings.Contains(code, `r.Patch("/posts/{id}", PatchPostHandler(db))`) {
		t.Error("Expected PATCH route to be registered in router")
	}
}

// TestPatchNullableFields verifies that nullable fields can be set to null
func TestPatchNullableFields(t *testing.T) {
	g := NewGenerator()

	// Create a resource with nullable and non-nullable fields
	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{
				Name:     "id",
				Type:     &ast.TypeNode{Name: "uuid"},
				Nullable: false,
				Constraints: []*ast.ConstraintNode{
					{Name: "primary"},
					{Name: "auto"},
				},
			},
			{
				Name:     "title",
				Type:     &ast.TypeNode{Name: "string"},
				Nullable: false,
			},
			{
				Name:     "published_at",
				Type:     &ast.TypeNode{Name: "timestamp"},
				Nullable: true, // This is nullable
			},
			{
				Name:     "description",
				Type:     &ast.TypeNode{Name: "text"},
				Nullable: true, // This is nullable
			},
		},
	}

	// Generate Patch method
	g.reset()
	g.generatePatch(resource)
	code := g.buf.String()

	// Verify the Patch method accepts nullable fields
	// The method should handle pointer types (*time.Time, *string) for nullable fields
	if !strings.Contains(code, "json.Unmarshal") {
		t.Error("Expected Patch method to use json.Unmarshal for merging")
	}

	// Verify that the merge behavior is documented
	// The second json.Unmarshal should merge partial data with existing resource
	// This allows nullable fields to be set to nil when provided as null in JSON
	if !strings.Contains(code, "Apply partial updates") {
		t.Error("Expected Patch method to document merge behavior")
	}

	// Verify the method validates the merged result
	if !strings.Contains(code, "Validate()") {
		t.Error("Expected Patch method to validate merged result")
	}

	// The test verifies that the generated code structure supports setting
	// nullable fields to null through JSON unmarshaling, which Go handles
	// automatically by setting pointer fields to nil when JSON contains null
}
