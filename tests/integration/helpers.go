package integration

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/codegen"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
	"github.com/conduit-lang/conduit/internal/compiler/parser"
	"github.com/conduit-lang/conduit/internal/compiler/typechecker"
	_ "github.com/lib/pq"
)

// TestDB provides a test database connection
type TestDB struct {
	DB     *sql.DB
	DBName string
}

// SetupTestDB creates a test database connection
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Skip if no test database is available
	if os.Getenv("SKIP_DB_TESTS") == "1" {
		t.Skip("Skipping database tests (SKIP_DB_TESTS=1)")
	}

	connStr := "host=localhost port=5433 user=conduit_test password=test_password dbname=conduit_test sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("Could not ping test database: %v", err)
	}

	// Register cleanup using t.Cleanup for automatic execution
	tdb := &TestDB{DB: db, DBName: "conduit_test"}
	t.Cleanup(func() { tdb.Cleanup(t) })

	return tdb
}

// Cleanup closes the database connection and cleans up test data
func (tdb *TestDB) Cleanup(t *testing.T) {
	t.Helper()

	// Ensure connection is always closed, even on panic
	defer tdb.DB.Close()

	// Reset connection state before cleanup
	if err := tdb.DB.Ping(); err != nil {
		t.Logf("Warning: database connection unhealthy: %v", err)
		return
	}

	_, err := tdb.DB.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`)
	if err != nil {
		t.Errorf("Failed to clean database (may affect other tests): %v", err)
	}
}

// CompileResult holds the result of compilation
type CompileResult struct {
	Files       map[string]string
	AST         *ast.Program
	LexErrors   []lexer.LexError
	ParseErrors []parser.ParseError
	TypeErrors  typechecker.ErrorList
	Success     bool
}

// CompileSource compiles Conduit source code through the full pipeline
func CompileSource(t *testing.T, source string) *CompileResult {
	t.Helper()

	result := &CompileResult{Success: false}

	// Lexer
	lex := lexer.New(source)
	tokens, lexErrors := lex.ScanTokens()
	result.LexErrors = lexErrors
	if len(lexErrors) > 0 {
		return result // Return early with errors, don't fatal
	}

	// Parser
	p := parser.New(tokens)
	prog, parseErrors := p.Parse()
	result.ParseErrors = parseErrors
	if len(parseErrors) > 0 {
		return result // Return early with errors
	}

	result.AST = prog

	// Type Checker
	tc := typechecker.NewTypeChecker()
	typeErrors := tc.CheckProgram(prog)
	result.TypeErrors = typeErrors

	if len(typeErrors) > 0 {
		return result
	}

	// Code Generator
	gen := codegen.NewGenerator()
	files, err := gen.GenerateProgram(prog, "test-app", "", "")
	if err != nil {
		t.Fatalf("Code generation error: %v", err)
	}

	result.Files = files
	result.Success = true

	return result
}

// WriteGeneratedFiles writes generated files to a temporary directory
func WriteGeneratedFiles(t *testing.T, files map[string]string) string {
	t.Helper()

	tmpDir := t.TempDir()

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}

	return tmpDir
}

// RunGoVet runs go vet on generated code
func RunGoVet(t *testing.T, dir string) error {
	t.Helper()

	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go vet failed: %v\nOutput: %s", err, stderr.String())
	}

	return nil
}

// RunGoFmt runs gofmt to check formatting
func RunGoFmt(t *testing.T, dir string) error {
	t.Helper()

	var allErrors []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			cmd := exec.Command("gofmt", "-l", path)
			output, err := cmd.Output()
			if err != nil {
				return fmt.Errorf("gofmt failed on %s: %v", path, err)
			}

			if len(output) > 0 {
				allErrors = append(allErrors, fmt.Sprintf("File %s is not gofmt-compliant", path))
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("formatting errors:\n%s", strings.Join(allErrors, "\n"))
	}

	return nil
}

// CreateTestResource creates a simple test resource
func CreateTestResource() string {
	return `
resource User {
	id: uuid! @primary @auto
	email: string! @unique
	name: string!
	created_at: timestamp! @auto
}
`
}

// CreateResourceWithHooks creates a resource with lifecycle hooks
func CreateResourceWithHooks() string {
	return `
resource Post {
	id: uuid! @primary @auto
	title: string! @min(5) @max(200)
	slug: string! @unique
	content: text!
	view_count: int! @min(0)
	created_at: timestamp! @auto

	@before create {
		self.view_count = 0
	}
}
`
}

// CreateResourceWithRelationships creates resources with relationships
func CreateResourceWithRelationships() string {
	return `
resource User {
	id: uuid! @primary @auto
	email: string! @unique
	name: string!
}

resource Post {
	id: uuid! @primary @auto
	title: string!
	content: text!
	author_id: uuid!

	author: User! {
		foreign_key: "author_id"
		on_delete: restrict
	}
}
`
}

// CountLinesOfCode counts non-empty, non-comment lines
func CountLinesOfCode(source string) int {
	lines := strings.Split(source, "\n")
	count := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "///") {
			count++
		}
	}

	return count
}
