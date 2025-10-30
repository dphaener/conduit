package codegen

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestGenerateMain_Basic(t *testing.T) {
	resources := []*ast.ResourceNode{
		{
			Name: "User",
			Fields: []*ast.FieldNode{
				{Name: "username", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateMain(resources, "example.com/testapp")
	if err != nil {
		t.Fatalf("GenerateMain failed: %v", err)
	}

	// Verify package declaration
	if !strings.Contains(code, "package main") {
		t.Error("Generated code should have main package")
	}

	// Verify imports
	if !strings.Contains(code, "\"database/sql\"") {
		t.Error("Generated code should import database/sql")
	}

	if !strings.Contains(code, "\"net/http\"") {
		t.Error("Generated code should import net/http")
	}

	if !strings.Contains(code, "github.com/go-chi/chi/v5") {
		t.Error("Generated code should import chi router")
	}

	if !strings.Contains(code, "_ \"github.com/jackc/pgx/v5/stdlib\"") {
		t.Error("Generated code should import PostgreSQL driver")
	}
}

func TestGenerateMain_MainFunction(t *testing.T) {
	resources := []*ast.ResourceNode{
		{
			Name: "Post",
			Fields: []*ast.FieldNode{
				{Name: "title", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
			},
		},
		{
			Name: "Comment",
			Fields: []*ast.FieldNode{
				{Name: "content", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "text"}, Nullable: false},
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateMain(resources, "example.com/testapp")
	if err != nil {
		t.Fatalf("GenerateMain failed: %v", err)
	}

	// Verify main function
	if !strings.Contains(code, "func main()") {
		t.Error("Generated code should contain main function")
	}

	// Verify database initialization
	if !strings.Contains(code, "db, err := initDB()") {
		t.Error("Generated code should initialize database")
	}

	if !strings.Contains(code, "defer db.Close()") {
		t.Error("Generated code should defer database close")
	}

	// Verify router initialization
	if !strings.Contains(code, "r := chi.NewRouter()") {
		t.Error("Generated code should create chi router")
	}

	// Verify middleware
	if !strings.Contains(code, "r.Use(middleware.Logger)") {
		t.Error("Generated code should use Logger middleware")
	}

	if !strings.Contains(code, "r.Use(middleware.Recoverer)") {
		t.Error("Generated code should use Recoverer middleware")
	}

	if !strings.Contains(code, "r.Use(middleware.RequestID)") {
		t.Error("Generated code should use RequestID middleware")
	}

	// Verify resource route registration
	if !strings.Contains(code, "handlers.RegisterPostRoutes(r, db)") {
		t.Error("Generated code should register Post routes")
	}

	if !strings.Contains(code, "handlers.RegisterCommentRoutes(r, db)") {
		t.Error("Generated code should register Comment routes")
	}

	// Verify health check endpoint
	if !strings.Contains(code, `r.Get("/health"`) {
		t.Error("Generated code should have health check endpoint")
	}

	// Verify server start
	if !strings.Contains(code, "http.ListenAndServe(addr, r)") {
		t.Error("Generated code should start HTTP server")
	}
}

func TestGenerateInitDBFunction(t *testing.T) {
	resources := []*ast.ResourceNode{
		{
			Name: "User",
			Fields: []*ast.FieldNode{
				{Name: "email", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateMain(resources, "example.com/testapp")
	if err != nil {
		t.Fatalf("GenerateMain failed: %v", err)
	}

	// Verify initDB function
	if !strings.Contains(code, "func initDB() (*sql.DB, error)") {
		t.Error("Generated code should contain initDB function")
	}

	// Verify DATABASE_URL environment variable
	if !strings.Contains(code, `os.Getenv("DATABASE_URL")`) {
		t.Error("Generated code should read DATABASE_URL from environment")
	}

	// Verify default connection string
	if !strings.Contains(code, `postgres://localhost/conduit_dev`) {
		t.Error("Generated code should have default connection string")
	}

	// Verify database operations
	if !strings.Contains(code, `sql.Open("pgx", dbURL)`) {
		t.Error("Generated code should open PostgreSQL connection")
	}

	if !strings.Contains(code, "db.Ping()") {
		t.Error("Generated code should ping database")
	}

	// Verify connection pool settings
	if !strings.Contains(code, "db.SetMaxOpenConns(25)") {
		t.Error("Generated code should set max open connections")
	}

	if !strings.Contains(code, "db.SetMaxIdleConns(5)") {
		t.Error("Generated code should set max idle connections")
	}
}

func TestGenerateMain_PortConfiguration(t *testing.T) {
	resources := []*ast.ResourceNode{
		{
			Name: "Item",
			Fields: []*ast.FieldNode{
				{Name: "name", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateMain(resources, "example.com/testapp")
	if err != nil {
		t.Fatalf("GenerateMain failed: %v", err)
	}

	// Verify PORT environment variable
	if !strings.Contains(code, `os.Getenv("PORT")`) {
		t.Error("Generated code should read PORT from environment")
	}

	// Verify default port
	if !strings.Contains(code, `port = "8080"`) {
		t.Error("Generated code should default to port 8080")
	}

	// Verify address formatting
	if !strings.Contains(code, `fmt.Sprintf(":%s", port)`) {
		t.Error("Generated code should format address with port")
	}
}
