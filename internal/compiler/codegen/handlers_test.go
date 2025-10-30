package codegen

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestGenerateHandlers_Basic(t *testing.T) {
	resources := []*ast.ResourceNode{
		{
			Name: "User",
			Fields: []*ast.FieldNode{
				{Name: "username", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
				{Name: "email", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateHandlers(resources, "example.com/testapp")
	if err != nil {
		t.Fatalf("GenerateHandlers failed: %v", err)
	}

	// Verify package declaration
	if !strings.Contains(code, "package handlers") {
		t.Error("Generated code should have handlers package")
	}

	// Verify imports
	if !strings.Contains(code, "\"net/http\"") {
		t.Error("Generated code should import net/http")
	}

	if !strings.Contains(code, "github.com/go-chi/chi/v5") {
		t.Error("Generated code should import chi router")
	}
}

func TestGenerateListHandler(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{Name: "title", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
		},
	}

	gen := NewGenerator()
	gen.reset()
	gen.imports = make(map[string]bool)
	gen.generateListHandler(resource)
	code := gen.buf.String()

	// Verify handler function
	if !strings.Contains(code, "func ListPostHandler(db *sql.DB) http.HandlerFunc") {
		t.Error("Generated code should contain ListPostHandler function")
	}

	// Verify pagination parsing
	if !strings.Contains(code, "limit := 50") {
		t.Error("Generated code should have default limit")
	}

	if !strings.Contains(code, "offset := 0") {
		t.Error("Generated code should have default offset")
	}

	// Verify FindAll call
	if !strings.Contains(code, "models.FindAllPost(ctx, db, limit, offset)") {
		t.Error("Generated code should call FindAllPost")
	}

	// Verify JSON response
	if !strings.Contains(code, "json.NewEncoder(w).Encode(results)") {
		t.Error("Generated code should encode JSON response")
	}
}

func TestGenerateGetHandler(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Comment",
		Fields: []*ast.FieldNode{
			{Name: "content", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "text"}, Nullable: false},
		},
	}

	gen := NewGenerator()
	gen.reset()
	gen.imports = make(map[string]bool)
	gen.generateGetHandler(resource)
	code := gen.buf.String()

	// Verify handler function
	if !strings.Contains(code, "func GetCommentHandler(db *sql.DB) http.HandlerFunc") {
		t.Error("Generated code should contain GetCommentHandler function")
	}

	// Verify ID parsing
	if !strings.Contains(code, "chi.URLParam(r, \"id\")") {
		t.Error("Generated code should parse ID from URL")
	}

	if !strings.Contains(code, "strconv.ParseInt(idStr, 10, 64)") {
		t.Error("Generated code should parse ID as int64")
	}

	// Verify FindByID call
	if !strings.Contains(code, "models.FindCommentByID(ctx, db, id)") {
		t.Error("Generated code should call FindCommentByID")
	}

	// Verify 404 handling
	if !strings.Contains(code, "if err == sql.ErrNoRows") {
		t.Error("Generated code should handle not found case")
	}

	if !strings.Contains(code, "http.StatusNotFound") {
		t.Error("Generated code should return 404 for not found")
	}
}

func TestGenerateCreateHandler(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Article",
		Fields: []*ast.FieldNode{
			{Name: "title", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
			{Name: "content", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "text"}, Nullable: false},
		},
	}

	gen := NewGenerator()
	gen.reset()
	gen.imports = make(map[string]bool)
	gen.generateCreateHandler(resource)
	code := gen.buf.String()

	// Verify handler function
	if !strings.Contains(code, "func CreateArticleHandler(db *sql.DB) http.HandlerFunc") {
		t.Error("Generated code should contain CreateArticleHandler function")
	}

	// Verify JSON decoding
	if !strings.Contains(code, "var a models.Article") {
		t.Error("Generated code should declare Article variable")
	}

	if !strings.Contains(code, "json.NewDecoder(r.Body).Decode(&a)") {
		t.Error("Generated code should decode request body")
	}

	// Verify Create call
	if !strings.Contains(code, "a.Create(ctx, db)") {
		t.Error("Generated code should call Create method")
	}

	// Verify 201 Created response
	if !strings.Contains(code, "http.StatusCreated") {
		t.Error("Generated code should return 201 Created")
	}
}

func TestGenerateUpdateHandler(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Task",
		Fields: []*ast.FieldNode{
			{Name: "title", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
			{Name: "completed", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "bool"}, Nullable: false},
		},
	}

	gen := NewGenerator()
	gen.reset()
	gen.imports = make(map[string]bool)
	gen.generateUpdateHandler(resource)
	code := gen.buf.String()

	// Verify handler function
	if !strings.Contains(code, "func UpdateTaskHandler(db *sql.DB) http.HandlerFunc") {
		t.Error("Generated code should contain UpdateTaskHandler function")
	}

	// Verify ID parsing and assignment
	if !strings.Contains(code, "chi.URLParam(r, \"id\")") {
		t.Error("Generated code should parse ID from URL")
	}

	if !strings.Contains(code, "t.ID = id") {
		t.Error("Generated code should set ID from URL")
	}

	// Verify Update call
	if !strings.Contains(code, "t.Update(ctx, db)") {
		t.Error("Generated code should call Update method")
	}
}

func TestGenerateDeleteHandler(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Note",
		Fields: []*ast.FieldNode{
			{Name: "content", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "text"}, Nullable: false},
		},
	}

	gen := NewGenerator()
	gen.reset()
	gen.imports = make(map[string]bool)
	gen.generateDeleteHandler(resource)
	code := gen.buf.String()

	// Verify handler function
	if !strings.Contains(code, "func DeleteNoteHandler(db *sql.DB) http.HandlerFunc") {
		t.Error("Generated code should contain DeleteNoteHandler function")
	}

	// Verify FindByID before delete
	if !strings.Contains(code, "models.FindNoteByID(ctx, db, id)") {
		t.Error("Generated code should fetch existing note before deleting")
	}

	// Verify Delete call
	if !strings.Contains(code, "n.Delete(ctx, db)") {
		t.Error("Generated code should call Delete method")
	}

	// Verify 204 No Content response
	if !strings.Contains(code, "http.StatusNoContent") {
		t.Error("Generated code should return 204 No Content")
	}
}

func TestGenerateRouterRegistration(t *testing.T) {
	resources := []*ast.ResourceNode{
		{
			Name: "Book",
			Fields: []*ast.FieldNode{
				{Name: "title", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateHandlers(resources, "example.com/testapp")
	if err != nil {
		t.Fatalf("GenerateHandlers failed: %v", err)
	}

	// Verify router registration function
	if !strings.Contains(code, "func RegisterBookRoutes(r chi.Router, db *sql.DB)") {
		t.Error("Generated code should contain RegisterBookRoutes function")
	}

	// Verify routes
	routes := []string{
		`r.Get("/books", ListBookHandler(db))`,
		`r.Post("/books", CreateBookHandler(db))`,
		`r.Get("/books/{id}", GetBookHandler(db))`,
		`r.Put("/books/{id}", UpdateBookHandler(db))`,
		`r.Delete("/books/{id}", DeleteBookHandler(db))`,
	}

	for _, route := range routes {
		if !strings.Contains(code, route) {
			t.Errorf("Generated code should contain route: %s", route)
		}
	}
}
