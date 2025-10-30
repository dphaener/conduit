package codegen

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// generateCreate generates the Create() method for a resource
func (g *Generator) generateCreate(resource *ast.ResourceNode) {
	receiverName := strings.ToLower(resource.Name[0:1])

	g.writeLine("// Create inserts a new %s into the database", resource.Name)
	g.writeLine("func (%s *%s) Create(ctx context.Context, db *sql.DB) error {",
		receiverName, resource.Name)
	g.indent++

	// Validate first
	g.writeLine("if err := %s.Validate(); err != nil {", receiverName)
	g.indent++
	g.writeLine(`return fmt.Errorf("validation failed: %w", err)`)
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Handle @auto fields (UUID generation, timestamps)
	g.generateAutoFields(resource, "create")
	if hasAutoFields(resource) {
		g.writeLine("")
	}

	// Build INSERT query
	columns, placeholders, values := g.buildInsertQuery(resource)

	// Check if we need to return ID
	needsReturningID := needsAutoID(resource)
	if needsReturningID {
		g.writeLine("query := `INSERT INTO %s (%s) VALUES (%s) RETURNING id`",
			g.toTableName(resource.Name), strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	} else {
		g.writeLine("query := `INSERT INTO %s (%s) VALUES (%s)`",
			g.toTableName(resource.Name), strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	}
	g.writeLine("")

	// Execute query
	if needsReturningID {
		g.writeLine("err := db.QueryRowContext(ctx, query, %s).Scan(&%s.ID)",
			strings.Join(values, ", "), receiverName)
		g.writeLine("if err != nil {")
		g.indent++
		g.writeLine("return fmt.Errorf(\"failed to create %s: %%w\", err)", strings.ToLower(resource.Name))
		g.indent--
		g.writeLine("}")
	} else {
		g.writeLine("_, err := db.ExecContext(ctx, query, %s)", strings.Join(values, ", "))
		g.writeLine("if err != nil {")
		g.indent++
		g.writeLine("return fmt.Errorf(\"failed to create %s: %%w\", err)", strings.ToLower(resource.Name))
		g.indent--
		g.writeLine("}")
	}

	g.writeLine("")
	g.writeLine("return nil")
	g.indent--
	g.writeLine("}")
}

// getIDGoType returns the Go type for the ID field
func (g *Generator) getIDGoType(resource *ast.ResourceNode) string {
	for _, field := range resource.Fields {
		if field.Name == "id" {
			return g.toGoType(field)
		}
	}
	// Default ID type if not explicitly defined
	return "int64"
}

// generateFindByID generates the FindByID() function for a resource
func (g *Generator) generateFindByID(resource *ast.ResourceNode) {
	idType := g.getIDGoType(resource)
	g.writeLine("// FindByID retrieves a %s by its ID", resource.Name)
	g.writeLine("func Find%sByID(ctx context.Context, db *sql.DB, id %s) (*%s, error) {",
		resource.Name, idType, resource.Name)
	g.indent++

	// Build SELECT query
	columns, scanTargets := g.buildSelectQuery(resource)

	g.writeLine("query := `SELECT %s FROM %s WHERE id = $1`",
		strings.Join(columns, ", "), g.toTableName(resource.Name))
	g.writeLine("")

	g.writeLine("%s := &%s{}", strings.ToLower(resource.Name[0:1]), resource.Name)
	g.writeLine("err := db.QueryRowContext(ctx, query, id).Scan(%s)",
		strings.Join(scanTargets, ", "))
	g.writeLine("")

	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return nil, fmt.Errorf(\"failed to find %s: %%w\", err)", strings.ToLower(resource.Name))
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("return %s, nil", strings.ToLower(resource.Name[0:1]))
	g.indent--
	g.writeLine("}")
}

// generateUpdate generates the Update() method for a resource
func (g *Generator) generateUpdate(resource *ast.ResourceNode) {
	receiverName := strings.ToLower(resource.Name[0:1])

	g.writeLine("// Update updates an existing %s in the database", resource.Name)
	g.writeLine("func (%s *%s) Update(ctx context.Context, db *sql.DB) error {",
		receiverName, resource.Name)
	g.indent++

	// Validate first
	g.writeLine("if err := %s.Validate(); err != nil {", receiverName)
	g.indent++
	g.writeLine(`return fmt.Errorf("validation failed: %w", err)`)
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Handle @auto_update fields
	g.generateAutoFields(resource, "update")
	if hasAutoUpdateFields(resource) {
		g.writeLine("")
	}

	// Build UPDATE query
	setClauses, values := g.buildUpdateQuery(resource)

	g.writeLine("query := `UPDATE %s SET %s WHERE id = $%d`",
		g.toTableName(resource.Name), strings.Join(setClauses, ", "), len(setClauses)+1)
	g.writeLine("")

	// Add ID to values
	values = append(values, fmt.Sprintf("%s.ID", receiverName))

	g.writeLine("_, err := db.ExecContext(ctx, query, %s)", strings.Join(values, ", "))
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return fmt.Errorf(\"failed to update %s: %%w\", err)", strings.ToLower(resource.Name))
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("return nil")
	g.indent--
	g.writeLine("}")
}

// generateDelete generates the Delete() method for a resource
func (g *Generator) generateDelete(resource *ast.ResourceNode) {
	receiverName := strings.ToLower(resource.Name[0:1])

	g.writeLine("// Delete removes a %s from the database", resource.Name)
	g.writeLine("func (%s *%s) Delete(ctx context.Context, db *sql.DB) error {",
		receiverName, resource.Name)
	g.indent++

	g.writeLine("query := `DELETE FROM %s WHERE id = $1`", g.toTableName(resource.Name))
	g.writeLine("")

	g.writeLine("_, err := db.ExecContext(ctx, query, %s.ID)", receiverName)
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return fmt.Errorf(\"failed to delete %s: %%w\", err)", strings.ToLower(resource.Name))
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("return nil")
	g.indent--
	g.writeLine("}")
}

// generateFindAll generates the FindAll() function with pagination support
func (g *Generator) generateFindAll(resource *ast.ResourceNode) {
	g.writeLine("// FindAll%s retrieves all %s with pagination", resource.Name, strings.ToLower(resource.Name)+"s")
	g.writeLine("func FindAll%s(ctx context.Context, db *sql.DB, limit, offset int) ([]*%s, error) {",
		resource.Name, resource.Name)
	g.indent++

	// Build SELECT query
	columns, _ := g.buildSelectQuery(resource)

	g.writeLine("query := `SELECT %s FROM %s ORDER BY id LIMIT $1 OFFSET $2`",
		strings.Join(columns, ", "), g.toTableName(resource.Name))
	g.writeLine("")

	g.writeLine("rows, err := db.QueryContext(ctx, query, limit, offset)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return nil, fmt.Errorf(\"failed to query %s: %%w\", err)", strings.ToLower(resource.Name)+"s")
	g.indent--
	g.writeLine("}")
	g.writeLine("defer rows.Close()")
	g.writeLine("")

	g.writeLine("var results []*%s", resource.Name)
	g.writeLine("for rows.Next() {")
	g.indent++

	_, scanTargets := g.buildSelectQuery(resource)
	g.writeLine("%s := &%s{}", strings.ToLower(resource.Name[0:1]), resource.Name)
	g.writeLine("if err := rows.Scan(%s); err != nil {", strings.Join(scanTargets, ", "))
	g.indent++
	g.writeLine("return nil, fmt.Errorf(\"failed to scan %s: %%w\", err)", strings.ToLower(resource.Name))
	g.indent--
	g.writeLine("}")
	g.writeLine("results = append(results, %s)", strings.ToLower(resource.Name[0:1]))

	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("if err := rows.Err(); err != nil {")
	g.indent++
	g.writeLine("return nil, fmt.Errorf(\"error iterating %s: %%w\", err)", strings.ToLower(resource.Name)+"s")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("return results, nil")
	g.indent--
	g.writeLine("}")
}

// generateCount generates the Count() function for a resource
func (g *Generator) generateCount(resource *ast.ResourceNode) {
	g.writeLine("// Count%s returns the total number of %s records", resource.Name, strings.ToLower(resource.Name)+"s")
	g.writeLine("func Count%s(ctx context.Context, db *sql.DB) (int, error) {", resource.Name)
	g.indent++

	g.writeLine("var count int")
	g.writeLine("query := `SELECT COUNT(*) FROM %s`", g.toTableName(resource.Name))
	g.writeLine("")

	g.writeLine("err := db.QueryRowContext(ctx, query).Scan(&count)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return 0, fmt.Errorf(\"failed to count %s: %%w\", err)", strings.ToLower(resource.Name)+"s")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("return count, nil")
	g.indent--
	g.writeLine("}")
}

// buildInsertQuery builds the column list, placeholders, and values for an INSERT query
func (g *Generator) buildInsertQuery(resource *ast.ResourceNode) (columns, placeholders, values []string) {
	receiverName := strings.ToLower(resource.Name[0:1])
	paramNum := 1

	for _, field := range resource.Fields {
		// Skip auto-generated ID fields
		if field.Name == "id" && hasConstraint(field, "auto") {
			continue
		}

		columnName := g.toDBColumnName(field.Name)
		columns = append(columns, columnName)
		placeholders = append(placeholders, fmt.Sprintf("$%d", paramNum))
		values = append(values, fmt.Sprintf("%s.%s", receiverName, g.toGoFieldName(field.Name)))
		paramNum++
	}

	return columns, placeholders, values
}

// buildSelectQuery builds the column list and scan targets for a SELECT query
func (g *Generator) buildSelectQuery(resource *ast.ResourceNode) (columns, scanTargets []string) {
	receiverName := strings.ToLower(resource.Name[0:1])

	// Always include ID
	columns = append(columns, "id")
	scanTargets = append(scanTargets, fmt.Sprintf("&%s.ID", receiverName))

	for _, field := range resource.Fields {
		if field.Name == "id" {
			continue // Already added
		}

		columnName := g.toDBColumnName(field.Name)
		columns = append(columns, columnName)
		scanTargets = append(scanTargets, fmt.Sprintf("&%s.%s", receiverName, g.toGoFieldName(field.Name)))
	}

	return columns, scanTargets
}

// buildUpdateQuery builds the SET clauses and values for an UPDATE query
func (g *Generator) buildUpdateQuery(resource *ast.ResourceNode) (setClauses, values []string) {
	receiverName := strings.ToLower(resource.Name[0:1])
	paramNum := 1

	for _, field := range resource.Fields {
		// Skip ID field
		if field.Name == "id" {
			continue
		}

		columnName := g.toDBColumnName(field.Name)
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", columnName, paramNum))
		values = append(values, fmt.Sprintf("%s.%s", receiverName, g.toGoFieldName(field.Name)))
		paramNum++
	}

	return setClauses, values
}

// generateAutoFields generates code to handle @auto and @auto_update fields
func (g *Generator) generateAutoFields(resource *ast.ResourceNode, operation string) {
	receiverName := strings.ToLower(resource.Name[0:1])

	for _, field := range resource.Fields {
		if operation == "create" && hasConstraint(field, "auto") {
			switch field.Type.Name {
			case "uuid":
				g.writeLine("%s.%s = uuid.New()", receiverName, g.toGoFieldName(field.Name))
			case "timestamp":
				g.writeLine("%s.%s = time.Now()", receiverName, g.toGoFieldName(field.Name))
			}
		}

		if operation == "update" && hasConstraint(field, "auto_update") {
			switch field.Type.Name {
			case "timestamp":
				g.writeLine("%s.%s = time.Now()", receiverName, g.toGoFieldName(field.Name))
			}
		}
	}
}

// hasConstraint checks if a field has a specific constraint
func hasConstraint(field *ast.FieldNode, constraintName string) bool {
	for _, constraint := range field.Constraints {
		if constraint.Name == constraintName {
			return true
		}
	}
	return false
}

// hasAutoFields checks if a resource has any @auto fields
func hasAutoFields(resource *ast.ResourceNode) bool {
	for _, field := range resource.Fields {
		if hasConstraint(field, "auto") {
			return true
		}
	}
	return false
}

// hasAutoUpdateFields checks if a resource has any @auto_update fields
func hasAutoUpdateFields(resource *ast.ResourceNode) bool {
	for _, field := range resource.Fields {
		if hasConstraint(field, "auto_update") {
			return true
		}
	}
	return false
}

// needsAutoID checks if the resource needs an auto-generated ID
func needsAutoID(resource *ast.ResourceNode) bool {
	for _, field := range resource.Fields {
		if field.Name == "id" {
			// If ID is explicitly defined but not auto, don't generate
			if !hasConstraint(field, "auto") {
				return false
			}
			// If ID is auto, check the type
			if field.Type.Name == "uuid" {
				return false // UUID is generated before insert
			}
			return true // Serial/auto-increment ID needs RETURNING
		}
	}
	return true // Default ID needs RETURNING
}
