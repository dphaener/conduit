package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/lexer"
	"github.com/conduit-lang/conduit/internal/compiler/parser"
	"github.com/conduit-lang/conduit/internal/orm/codegen"
	"github.com/conduit-lang/conduit/internal/orm/schema"
	utilstrings "github.com/conduit-lang/conduit/internal/util/strings"
)

// MigrationSQLGenerator generates SQL migration files from resource definitions
type MigrationSQLGenerator struct {
	ddlGenerator *codegen.DDLGenerator
}

// NewMigrationSQLGenerator creates a new MigrationSQLGenerator
func NewMigrationSQLGenerator() *MigrationSQLGenerator {
	return &MigrationSQLGenerator{
		ddlGenerator: codegen.NewDDLGenerator(),
	}
}

// GenerateFromResource generates SQL migration content from a resource name
// Returns (upSQL, downSQL, error)
func (g *MigrationSQLGenerator) GenerateFromResource(resourceName string) (string, string, error) {
	// Find the resource file
	resourceFile, err := g.findResourceFile(resourceName)
	if err != nil {
		return "", "", err
	}

	// Parse the resource file
	resourceSchema, err := g.parseResourceFile(resourceFile)
	if err != nil {
		return "", "", err
	}

	// Generate the up migration (CREATE TABLE)
	upSQL, err := g.generateUpMigration(resourceSchema)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate up migration: %w", err)
	}

	// Generate the down migration (DROP TABLE)
	downSQL := g.generateDownMigration(resourceSchema)

	return upSQL, downSQL, nil
}

// findResourceFile finds the resource definition file for the given resource name
func (g *MigrationSQLGenerator) findResourceFile(resourceName string) (string, error) {
	// Validate resource name - must be alphanumeric with optional underscores
	if !isValidResourceName(resourceName) {
		return "", fmt.Errorf("invalid resource name: must contain only letters, numbers, and underscores")
	}

	// Check app directory
	appDir := "app"
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		return "", fmt.Errorf("app/ directory not found - are you in a Conduit project?")
	}

	// Convert app directory to absolute path for safe path checking
	absAppDir, err := filepath.Abs(appDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve app directory: %w", err)
	}

	// Try lowercase .cdt file
	filename := filepath.Join(absAppDir, strings.ToLower(resourceName)+".cdt")

	// Security: Verify the resolved path is within app directory
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return "", fmt.Errorf("failed to resolve file path: %w", err)
	}

	// CRITICAL: Resolve symlinks to prevent traversal via symlinks
	realPath, err := filepath.EvalSymlinks(absFilename)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Use realPath for the prefix check (or absFilename if file doesn't exist yet)
	pathToCheck := realPath
	if os.IsNotExist(err) {
		pathToCheck = absFilename
	}

	if !strings.HasPrefix(pathToCheck, absAppDir+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid resource name: path traversal detected")
	}

	if _, err := os.Stat(filename); err == nil {
		return filename, nil
	}

	// Try exact case .cdt file
	filename = filepath.Join(absAppDir, resourceName+".cdt")
	absFilename, err = filepath.Abs(filename)
	if err != nil {
		return "", fmt.Errorf("failed to resolve file path: %w", err)
	}

	// CRITICAL: Resolve symlinks to prevent traversal via symlinks
	realPath, err = filepath.EvalSymlinks(absFilename)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to resolve symlinks: %w", err)
	}

	// Use realPath for the prefix check (or absFilename if file doesn't exist yet)
	pathToCheck = realPath
	if os.IsNotExist(err) {
		pathToCheck = absFilename
	}

	if !strings.HasPrefix(pathToCheck, absAppDir+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid resource name: path traversal detected")
	}

	if _, err := os.Stat(filename); err == nil {
		return filename, nil
	}

	return "", fmt.Errorf("resource file not found for '%s' (tried: %s.cdt, %s.cdt)",
		resourceName, strings.ToLower(resourceName), resourceName)
}

// isValidResourceName validates that a resource name contains only safe characters
func isValidResourceName(name string) bool {
	if name == "" || len(name) > 255 {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

// parseResourceFile parses a resource definition file and returns a ResourceSchema
func (g *MigrationSQLGenerator) parseResourceFile(filename string) (*schema.ResourceSchema, error) {
	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource file: %w", err)
	}

	// Lex the file
	l := lexer.New(string(content))
	tokens, lexErrors := l.ScanTokens()

	// Check for lexer errors
	if len(lexErrors) > 0 {
		var errMsgs []string
		for _, err := range lexErrors {
			errMsgs = append(errMsgs, err.Error())
		}
		return nil, fmt.Errorf("lexer errors in file %s:\n%s", filename, strings.Join(errMsgs, "\n"))
	}

	// Parse the file
	p := parser.New(tokens)
	program, parseErrors := p.Parse()

	if len(parseErrors) > 0 {
		var errMsgs []string
		for _, err := range parseErrors {
			errMsgs = append(errMsgs, err.Error())
		}
		return nil, fmt.Errorf("parse errors in file %s:\n%s", filename, strings.Join(errMsgs, "\n"))
	}

	// Expect exactly one resource
	if len(program.Resources) == 0 {
		return nil, fmt.Errorf("no resources found in file")
	}
	if len(program.Resources) > 1 {
		return nil, fmt.Errorf("multiple resources found in file (expected 1)")
	}

	// Convert AST to schema
	builder := schema.NewBuilder()
	resourceSchema, err := builder.Build(program.Resources[0])
	if err != nil {
		return nil, fmt.Errorf("failed to build schema: %w", err)
	}

	return resourceSchema, nil
}

// generateUpMigration generates the CREATE TABLE SQL for the up migration
func (g *MigrationSQLGenerator) generateUpMigration(resource *schema.ResourceSchema) (string, error) {
	var sql strings.Builder

	// Generate enum types first (if any)
	enumTypes := g.ddlGenerator.GenerateEnumTypes(resource)
	for _, enumType := range enumTypes {
		sql.WriteString(enumType)
		sql.WriteString("\n")
	}

	if len(enumTypes) > 0 {
		sql.WriteString("\n")
	}

	// Generate CREATE TABLE statement
	createTable, err := g.ddlGenerator.GenerateCreateTable(resource)
	if err != nil {
		return "", err
	}

	sql.WriteString(createTable)
	sql.WriteString("\n")

	// Generate indexes for @unique fields
	indexSQL := g.generateIndexes(resource)
	if indexSQL != "" {
		sql.WriteString("\n")
		sql.WriteString(indexSQL)
	}

	return sql.String(), nil
}

// generateDownMigration generates the DROP TABLE SQL for the down migration
func (g *MigrationSQLGenerator) generateDownMigration(resource *schema.ResourceSchema) string {
	var sql strings.Builder

	// Drop the table
	sql.WriteString(g.ddlGenerator.GenerateDropTable(resource))
	sql.WriteString("\n")

	// Drop enum types (if any)
	enumTypes := g.ddlGenerator.GenerateDropEnumTypes(resource)
	if len(enumTypes) > 0 {
		sql.WriteString("\n")
		for _, dropEnum := range enumTypes {
			sql.WriteString(dropEnum)
			sql.WriteString("\n")
		}
	}

	return sql.String()
}

// generateIndexes generates CREATE INDEX statements for @unique fields
func (g *MigrationSQLGenerator) generateIndexes(resource *schema.ResourceSchema) string {
	var sql strings.Builder

	tableName := resource.TableName
	if tableName == "" {
		tableName = utilstrings.ToSnakeCase(resource.Name)
	}

	for fieldName, field := range resource.Fields {
		// Check if field has @unique annotation
		hasUnique := false
		for _, annotation := range field.Annotations {
			if annotation.Name == "unique" {
				hasUnique = true
				break
			}
		}

		if hasUnique {
			// Don't create index for primary key fields (already indexed)
			isPrimary := false
			for _, annotation := range field.Annotations {
				if annotation.Name == "primary" {
					isPrimary = true
					break
				}
			}

			if !isPrimary {
				columnName := utilstrings.ToSnakeCase(fieldName)
				indexName := fmt.Sprintf("idx_%s_%s", tableName, columnName)
				sql.WriteString(fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s);\n",
					quoteIdentifier(indexName), quoteIdentifier(tableName), quoteIdentifier(columnName)))
			}
		}
	}

	return sql.String()
}

// quoteIdentifier wraps a SQL identifier in double quotes
func quoteIdentifier(identifier string) string {
	escaped := strings.ReplaceAll(identifier, `"`, `""`)
	return fmt.Sprintf(`"%s"`, escaped)
}
