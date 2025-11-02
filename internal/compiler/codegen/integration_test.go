package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

func TestGeneratedCode_Compiles(t *testing.T) {
	// Create a simple resource
	resource := &ast.ResourceNode{
		Name: "User",
		Fields: []*ast.FieldNode{
			{
				Name: "username",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
			{
				Name: "email",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "conduit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create models directory
	modelsDir := filepath.Join(tmpDir, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		t.Fatalf("Failed to create models dir: %v", err)
	}

	// Write generated code to file
	filePath := filepath.Join(modelsDir, "user.go")
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write generated code: %v", err)
	}

	// Create go.mod file
	goModContent := `module test-conduit

go 1.23

require github.com/google/uuid v1.6.0
`
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Try to compile the generated code
	cmd := exec.Command("go", "build", "./models")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Generated code failed to compile: %v\nOutput:\n%s\nGenerated code:\n%s",
			err, string(output), code)
	}
}

func TestGeneratedCode_GoFmt(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "User",
		Fields: []*ast.FieldNode{
			{
				Name: "username",
				Type: &ast.TypeNode{
					Kind:     ast.TypePrimitive,
					Name:     "string",
					Nullable: false,
				},
				Nullable: false,
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "*.go")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write code to file
	if _, err := tmpFile.Write([]byte(code)); err != nil {
		t.Fatalf("Failed to write code: %v", err)
	}
	tmpFile.Close()

	// Run gofmt
	cmd := exec.Command("gofmt", "-l", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gofmt failed: %v\nOutput: %s", err, string(output))
	}

	// If gofmt outputs the filename, it means the code is not formatted
	if strings.TrimSpace(string(output)) != "" {
		// Run gofmt -d to show the diff
		diffCmd := exec.Command("gofmt", "-d", tmpFile.Name())
		diffOutput, _ := diffCmd.CombinedOutput()
		t.Errorf("Generated code is not properly formatted according to gofmt\nDiff:\n%s\nGenerated code:\n%s",
			string(diffOutput), code)
	}
}

func TestGeneratedSQL_Valid(t *testing.T) {
	resources := []*ast.ResourceNode{
		{
			Name: "User",
			Fields: []*ast.FieldNode{
				{
					Name: "username",
					Type: &ast.TypeNode{
						Kind:     ast.TypePrimitive,
						Name:     "string",
						Nullable: false,
					},
					Nullable: false,
					Constraints: []*ast.ConstraintNode{
						{
							Name: "min",
							Arguments: []ast.ExprNode{
								&ast.LiteralExpr{Value: int64(3)},
							},
						},
						{
							Name: "max",
							Arguments: []ast.ExprNode{
								&ast.LiteralExpr{Value: int64(50)},
							},
						},
						{Name: "unique"},
					},
				},
				{
					Name: "email",
					Type: &ast.TypeNode{
						Kind:     ast.TypePrimitive,
						Name:     "string",
						Nullable: false,
					},
					Nullable: false,
				},
			},
		},
	}

	gen := NewGenerator()
	sql, err := gen.GenerateMigrations(resources)
	if err != nil {
		t.Fatalf("GenerateMigrations failed: %v", err)
	}

	// Basic SQL validation - check for required elements
	requiredElements := []string{
		"CREATE TABLE users",
		"username VARCHAR",
		"email VARCHAR",
		"NOT NULL",
		"CHECK (length(username) >= 3)",
		"UNIQUE",
	}

	for _, element := range requiredElements {
		if !strings.Contains(sql, element) {
			t.Errorf("Generated SQL missing required element: %s\nGenerated SQL:\n%s",
				element, sql)
		}
	}

	// Verify SQL syntax is valid (basic check)
	if !strings.Contains(sql, ");") {
		t.Error("Generated SQL should end table definitions with );")
	}
}

// TestAcceptanceCriteria_ResourceGeneration verifies AC1: Resource Generation
func TestAcceptanceCriteria_ResourceGeneration(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "User",
		Fields: []*ast.FieldNode{
			{Name: "id", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid"}, Nullable: false},
			{Name: "email", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
			{Name: "bio", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "text"}, Nullable: true},
			{Name: "age", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "int"}, Nullable: true},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// AC1.1: Generated struct has correct name
	if !strings.Contains(code, "type User struct {") {
		t.Error("AC1.1 FAIL: Generated struct should have correct name")
	}

	// AC1.2: Fields have correct Go types
	tests := []struct {
		field    string
		expected string
	}{
		{"ID", "uuid.UUID"},
		{"Email", "string"},
		{"Bio", "*string"},
		{"Age", "*int64"},
	}

	for _, tt := range tests {
		if !strings.Contains(code, tt.field) || !strings.Contains(code, tt.expected) {
			t.Errorf("AC1.2 FAIL: Field %s should map to %s", tt.field, tt.expected)
		}
	}

	// AC1.3: Struct tags include db and json
	if !strings.Contains(code, "`db:") || !strings.Contains(code, "json:") {
		t.Error("AC1.3 FAIL: Struct tags should include db and json")
	}

	// AC1.4: Nullable fields have omitempty
	if !strings.Contains(code, `json:"bio,omitempty"`) {
		t.Error("AC1.4 FAIL: Nullable fields should have omitempty in JSON tags")
	}

	t.Log("AC1: Resource Generation - PASS")
}

// TestAcceptanceCriteria_RepositoryGeneration verifies AC2: Repository Generation
func TestAcceptanceCriteria_RepositoryGeneration(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Article",
		Fields: []*ast.FieldNode{
			{Name: "id", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid"}, Nullable: false},
			{Name: "title", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// AC2.1: Create method exists with proper signature
	if !strings.Contains(code, "func (a *Article) Create(ctx context.Context, db *sql.DB) error {") {
		t.Error("AC2.1 FAIL: Create method missing or incorrect signature")
	}

	// AC2.2: FindByID method exists
	if !strings.Contains(code, "func FindArticleByID(ctx context.Context, db *sql.DB, id") {
		t.Error("AC2.2 FAIL: FindByID method missing")
	}

	// AC2.3: Update method exists
	if !strings.Contains(code, "func (a *Article) Update(ctx context.Context, db *sql.DB) error {") {
		t.Error("AC2.3 FAIL: Update method missing")
	}

	// AC2.4: Delete method exists
	if !strings.Contains(code, "func (a *Article) Delete(ctx context.Context, db *sql.DB) error {") {
		t.Error("AC2.4 FAIL: Delete method missing")
	}

	// AC2.5: FindAll method exists with pagination
	if !strings.Contains(code, "func FindAllArticle(ctx context.Context, db *sql.DB, limit, offset int)") {
		t.Error("AC2.5 FAIL: FindAll method missing or lacks pagination")
	}

	// AC2.6: Methods call Validate()
	if !strings.Contains(code, "a.Validate()") {
		t.Error("AC2.6 FAIL: CRUD methods should call Validate()")
	}

	t.Log("AC2: Repository Generation - PASS")
}

//  TestAcceptanceCriteria_HookGeneration verifies AC3: Hook Generation
func TestAcceptanceCriteria_HookGeneration(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Post",
		Fields: []*ast.FieldNode{
			{Name: "title", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
		},
		Hooks: []*ast.HookNode{
			{
				Timing:        "before",
				Event:         "create",
				IsTransaction: false,
				Body: []ast.StmtNode{
					&ast.ExprStmt{
						Expr: &ast.CallExpr{
							Namespace: "Logger",
							Function:  "info",
							Arguments: []ast.ExprNode{
								&ast.LiteralExpr{Value: "Creating post"},
							},
						},
					},
				},
			},
			{
				Timing:        "after",
				Event:         "update",
				IsTransaction: true,
				Body: []ast.StmtNode{
					&ast.ExprStmt{
						Expr: &ast.CallExpr{
							Namespace: "Cache",
							Function:  "invalidate",
							Arguments:      []ast.ExprNode{},
						},
					},
					&ast.BlockStmt{
						IsAsync: true,
						Statements: []ast.StmtNode{
							&ast.ExprStmt{
								Expr: &ast.CallExpr{
									Namespace: "Email",
									Function:  "notify",
									Arguments:      []ast.ExprNode{},
								},
							},
						},
					},
				},
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResourceWithHooks(resource)
	if err != nil {
		t.Fatalf("GenerateResourceWithHooks failed: %v", err)
	}

	// AC3.1: @before hooks generate BeforeCreate/Update/Delete methods
	if !strings.Contains(code, "func (p *Post) BeforeCreate(ctx context.Context, db *sql.DB) error {") {
		t.Error("AC3.1 FAIL: @before create hook should generate BeforeCreate method")
	}

	// AC3.2: @after hooks generate AfterCreate/Update/Delete methods
	if !strings.Contains(code, "func (p *Post) AfterUpdate(ctx context.Context, db *sql.DB) error {") {
		t.Error("AC3.2 FAIL: @after update hook should generate AfterUpdate method")
	}

	// AC3.3: @transaction wraps hook body in Begin/Commit/Rollback
	if !strings.Contains(code, "tx, err := db.Begin()") ||
		!strings.Contains(code, "defer tx.Rollback()") {
		t.Error("AC3.3 FAIL: @transaction should wrap code in transaction")
	}

	// AC3.4: @async spawns goroutine
	if !strings.Contains(code, "go func() {") {
		t.Error("AC3.4 FAIL: @async should spawn goroutine")
	}

	// AC3.5: Hook body statements are compiled
	if !strings.Contains(code, "stdlib.Logger_info(\"Creating post\")") {
		t.Error("AC3.5 FAIL: Hook body should be compiled to Go")
	}

	t.Log("AC3: Hook Generation - PASS")
}

// TestAcceptanceCriteria_HandlerGeneration verifies AC4: Handler Generation
func TestAcceptanceCriteria_HandlerGeneration(t *testing.T) {
	resources := []*ast.ResourceNode{
		{
			Name: "Comment",
			Fields: []*ast.FieldNode{
				{Name: "id", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid"}, Nullable: false},
				{Name: "content", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "text"}, Nullable: false},
			},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateHandlers(resources, "example.com/testapp")
	if err != nil {
		t.Fatalf("GenerateHandlers failed: %v", err)
	}

	// AC4.1: LIST handler exists
	if !strings.Contains(code, "func ListCommentHandler(db *sql.DB) http.HandlerFunc {") {
		t.Error("AC4.1 FAIL: LIST handler missing")
	}

	// AC4.2: GET handler exists
	if !strings.Contains(code, "func GetCommentHandler(db *sql.DB) http.HandlerFunc {") {
		t.Error("AC4.2 FAIL: GET handler missing")
	}

	// AC4.3: CREATE handler exists
	if !strings.Contains(code, "func CreateCommentHandler(db *sql.DB) http.HandlerFunc {") {
		t.Error("AC4.3 FAIL: CREATE handler missing")
	}

	// AC4.4: UPDATE handler exists
	if !strings.Contains(code, "func UpdateCommentHandler(db *sql.DB) http.HandlerFunc {") {
		t.Error("AC4.4 FAIL: UPDATE handler missing")
	}

	// AC4.5: DELETE handler exists
	if !strings.Contains(code, "func DeleteCommentHandler(db *sql.DB) http.HandlerFunc {") {
		t.Error("AC4.5 FAIL: DELETE handler missing")
	}

	// AC4.6: RegisterRoutes function exists
	if !strings.Contains(code, "func RegisterCommentRoutes(r chi.Router, db *sql.DB) {") {
		t.Error("AC4.6 FAIL: RegisterRoutes function missing")
	}

	// AC4.7: Routes are registered
	routes := []string{
		`r.Get("/comments", ListCommentHandler(db))`,
		`r.Post("/comments", CreateCommentHandler(db))`,
		`r.Get("/comments/{id}", GetCommentHandler(db))`,
		`r.Put("/comments/{id}", UpdateCommentHandler(db))`,
		`r.Delete("/comments/{id}", DeleteCommentHandler(db))`,
	}

	for _, route := range routes {
		if !strings.Contains(code, route) {
			t.Errorf("AC4.7 FAIL: Route %s not registered", route)
		}
	}

	t.Log("AC4: Handler Generation - PASS")
}

// TestAcceptanceCriteria_MigrationGeneration verifies AC5: Migration Generation
func TestAcceptanceCriteria_MigrationGeneration(t *testing.T) {
	resources := []*ast.ResourceNode{
		{
			Name: "Task",
			Fields: []*ast.FieldNode{
				{Name: "id", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "uuid"}, Nullable: false},
				{Name: "title", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
				{Name: "completed", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "bool"}, Nullable: false},
			},
		},
	}

	gen := NewGenerator()
	sql, err := gen.GenerateMigrations(resources)
	if err != nil {
		t.Fatalf("GenerateMigrations failed: %v", err)
	}

	// AC5.1: CREATE TABLE statement exists
	if !strings.Contains(sql, "CREATE TABLE tasks") {
		t.Error("AC5.1 FAIL: CREATE TABLE statement missing")
	}

	// AC5.2: Columns with correct SQL types
	if !strings.Contains(sql, "UUID") {
		t.Error("AC5.2 FAIL: UUID column missing")
	}
	if !strings.Contains(sql, "VARCHAR") && !strings.Contains(sql, "TEXT") {
		t.Error("AC5.2 FAIL: VARCHAR/TEXT column missing")
	}
	if !strings.Contains(sql, "BOOLEAN") {
		t.Error("AC5.2 FAIL: BOOLEAN column missing")
	}

	// AC5.3: PRIMARY KEY or auto-ID
	if !strings.Contains(sql, "PRIMARY KEY") && !strings.Contains(sql, "BIGSERIAL") {
		t.Error("AC5.3 FAIL: PRIMARY KEY or auto-ID missing")
	}

	// AC5.4: NOT NULL constraints
	if !strings.Contains(sql, "NOT NULL") {
		t.Error("AC5.4 FAIL: NOT NULL constraint missing")
	}

	t.Log("AC5: Migration Generation - PASS")
}

// TestAcceptanceCriteria_CodeQuality verifies AC6: Code Quality
func TestAcceptanceCriteria_CodeQuality(t *testing.T) {
	resource := &ast.ResourceNode{
		Name: "Sample",
		Fields: []*ast.FieldNode{
			{Name: "name", Type: &ast.TypeNode{Kind: ast.TypePrimitive, Name: "string"}, Nullable: false},
		},
	}

	gen := NewGenerator()
	code, err := gen.GenerateResource(resource)
	if err != nil {
		t.Fatalf("GenerateResource failed: %v", err)
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "codegen-*.go")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(code); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// AC6.1: gofmt compliance
	fmtCmd := exec.Command("gofmt", "-l", tmpFile.Name())
	output, err := fmtCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gofmt command failed: %v", err)
	}
	if len(output) > 0 {
		t.Error("AC6.1 FAIL: Generated code is not gofmt compliant")
	}

	// AC6.2: go vet passes
	vetCmd := exec.Command("go", "vet", tmpFile.Name())
	if output, err := vetCmd.CombinedOutput(); err != nil {
		t.Errorf("AC6.2 FAIL: go vet reported issues:\n%s", output)
	}

	t.Log("AC6: Code Quality - PASS")
}
