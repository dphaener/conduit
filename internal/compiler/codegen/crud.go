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

	// 1. Generate @auto fields FIRST (UUIDs, timestamps)
	g.writeLine("// Generate @auto fields (UUIDs, timestamps)")
	g.generateAutoFields(resource, "create")
	if hasAutoFields(resource) {
		g.writeLine("")
	}

	// 2. Call BeforeCreate hook if it exists
	if hasHook(resource, "before", "create") {
		g.writeLine("// Call BeforeCreate hook")
		g.writeLine("if err := %s.BeforeCreate(ctx, db); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("before create hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// 3. Call BeforeSave hook if it exists
	if hasHook(resource, "before", "save") {
		g.writeLine("// Call BeforeSave hook")
		g.writeLine("if err := %s.BeforeSave(ctx, db); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("before save hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// 4. Validate AFTER hooks have run
	g.writeLine("// Validate after hooks have run")
	g.writeLine("if err := %s.Validate(); err != nil {", receiverName)
	g.indent++
	g.writeLine(`return fmt.Errorf("validation failed: %w", err)`)
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// 5. Begin transaction
	g.writeLine("// Begin transaction")
	g.writeLine("tx, err := db.BeginTx(ctx, nil)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine(`return fmt.Errorf("failed to begin transaction: %w", err)`)
	g.indent--
	g.writeLine("}")
	g.writeLine("defer tx.Rollback()")
	g.writeLine("")

	// 6. Build INSERT query
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

	// Execute INSERT
	g.writeLine("// Execute INSERT")
	if needsReturningID {
		g.writeLine("err = tx.QueryRowContext(ctx, query, %s).Scan(&%s.ID)",
			strings.Join(values, ", "), receiverName)
		g.writeLine("if err != nil {")
		g.indent++
		g.writeLine("return fmt.Errorf(\"failed to insert %s: %%w\", err)", strings.ToLower(resource.Name))
		g.indent--
		g.writeLine("}")
	} else {
		g.writeLine("_, err = tx.ExecContext(ctx, query, %s)", strings.Join(values, ", "))
		g.writeLine("if err != nil {")
		g.indent++
		g.writeLine("return fmt.Errorf(\"failed to insert %s: %%w\", err)", strings.ToLower(resource.Name))
		g.indent--
		g.writeLine("}")
	}
	g.writeLine("")

	// 7. Call AfterCreate hook if it exists
	if hasHook(resource, "after", "create") {
		g.writeLine("// Call AfterCreate hook")
		g.writeLine("if err := %s.AfterCreate(ctx, tx); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("after create hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// 8. Call AfterSave hook if it exists
	if hasHook(resource, "after", "save") {
		g.writeLine("// Call AfterSave hook")
		g.writeLine("if err := %s.AfterSave(ctx, tx); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("after save hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// 9. Commit transaction
	g.writeLine("// Commit transaction")
	g.writeLine("if err := tx.Commit(); err != nil {")
	g.indent++
	g.writeLine(`return fmt.Errorf("failed to commit transaction: %w", err)`)
	g.indent--
	g.writeLine("}")
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

	// 1. Generate @auto_update fields
	g.writeLine("// Generate @auto_update fields (timestamps)")
	g.generateAutoFields(resource, "update")
	if hasAutoUpdateFields(resource) {
		g.writeLine("")
	}

	// 2. Call BeforeUpdate hook if it exists
	if hasHook(resource, "before", "update") {
		g.writeLine("// Call BeforeUpdate hook")
		g.writeLine("if err := %s.BeforeUpdate(ctx, db); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("before update hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// 3. Call BeforeSave hook if it exists
	if hasHook(resource, "before", "save") {
		g.writeLine("// Call BeforeSave hook")
		g.writeLine("if err := %s.BeforeSave(ctx, db); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("before save hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// 4. Validate AFTER hooks have run
	g.writeLine("// Validate after hooks have run")
	g.writeLine("if err := %s.Validate(); err != nil {", receiverName)
	g.indent++
	g.writeLine(`return fmt.Errorf("validation failed: %w", err)`)
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// 5. Begin transaction
	g.writeLine("// Begin transaction")
	g.writeLine("tx, err := db.BeginTx(ctx, nil)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine(`return fmt.Errorf("failed to begin transaction: %w", err)`)
	g.indent--
	g.writeLine("}")
	g.writeLine("defer tx.Rollback()")
	g.writeLine("")

	// 6. Build UPDATE query
	setClauses, values := g.buildUpdateQuery(resource)

	g.writeLine("query := `UPDATE %s SET %s WHERE id = $%d`",
		g.toTableName(resource.Name), strings.Join(setClauses, ", "), len(setClauses)+1)
	g.writeLine("")

	// Add ID to values
	values = append(values, fmt.Sprintf("%s.ID", receiverName))

	// Execute UPDATE
	g.writeLine("// Execute UPDATE")
	g.writeLine("_, err = tx.ExecContext(ctx, query, %s)", strings.Join(values, ", "))
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return fmt.Errorf(\"failed to update %s: %%w\", err)", strings.ToLower(resource.Name))
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// 7. Call AfterUpdate hook if it exists
	if hasHook(resource, "after", "update") {
		g.writeLine("// Call AfterUpdate hook")
		g.writeLine("if err := %s.AfterUpdate(ctx, tx); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("after update hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// 8. Call AfterSave hook if it exists
	if hasHook(resource, "after", "save") {
		g.writeLine("// Call AfterSave hook")
		g.writeLine("if err := %s.AfterSave(ctx, tx); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("after save hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// 9. Commit transaction
	g.writeLine("// Commit transaction")
	g.writeLine("if err := tx.Commit(); err != nil {")
	g.indent++
	g.writeLine(`return fmt.Errorf("failed to commit transaction: %w", err)`)
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("return nil")
	g.indent--
	g.writeLine("}")
}

// generatePatch generates the Patch() method for partial updates
func (g *Generator) generatePatch(resource *ast.ResourceNode) {
	receiverName := strings.ToLower(resource.Name[0:1])

	g.writeLine("// Patch partially updates an existing %s in the database", resource.Name)
	g.writeLine("func (%s *%s) Patch(ctx context.Context, db *sql.DB, partialJSON []byte) error {",
		receiverName, resource.Name)
	g.indent++

	// Parse partial JSON into a map to identify which fields were provided
	g.writeLine("// Parse partial JSON to identify provided fields")
	g.writeLine("var partialData map[string]interface{}")
	g.writeLine("if err := json.Unmarshal(partialJSON, &partialData); err != nil {")
	g.indent++
	g.writeLine(`return fmt.Errorf("invalid JSON: %%w", err)`)
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Check for empty PATCH
	g.writeLine("// Reject empty PATCH requests")
	g.writeLine("if len(partialData) == 0 {")
	g.indent++
	g.writeLine(`return fmt.Errorf("empty PATCH request: no fields provided")`)
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Validate no read-only fields
	g.writeLine("// Validate no read-only fields are being updated")
	g.writeLine("readOnlyFields := map[string]bool{")
	g.indent++
	g.writeLine(`"id": true,`)
	g.writeLine(`"created_at": true,`)
	g.writeLine(`"updated_at": true,`)
	g.indent--
	g.writeLine("}")
	g.writeLine("for field := range partialData {")
	g.indent++
	g.writeLine("if readOnlyFields[field] {")
	g.indent++
	g.writeLine(`return fmt.Errorf("field '%%s' is read-only and cannot be updated", field)`)
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Build list of valid fields
	g.writeLine("// Build list of valid fields")
	g.writeLine("validFields := map[string]bool{")
	g.indent++
	for _, field := range resource.Fields {
		if field.Name != "id" && !hasConstraint(field, "auto") && !hasConstraint(field, "auto_update") {
			columnName := g.toDBColumnName(field.Name)
			g.writeLine("\"%s\": true,", columnName)
		}
	}
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Validate unknown fields
	g.writeLine("// Validate no unknown fields")
	g.writeLine("for field := range partialData {")
	g.indent++
	g.writeLine("if !validFields[field] && !readOnlyFields[field] {")
	g.indent++
	g.writeLine("// Build list of valid field names for error message")
	g.writeLine("validFieldNames := make([]string, 0, len(validFields))")
	g.writeLine("for f := range validFields {")
	g.indent++
	g.writeLine("validFieldNames = append(validFieldNames, f)")
	g.indent--
	g.writeLine("}")
	g.writeLine(`return fmt.Errorf("unknown field '%%s': valid fields are: %%v", field, validFieldNames)`)
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Apply partial updates to existing resource
	g.writeLine("// Apply partial updates to existing resource.")
	g.writeLine("// Note: Nullable fields (pointer types) can be set to null explicitly.")
	g.writeLine("// Non-nullable fields retain their existing values if not provided.")
	g.writeLine("// Nested objects/arrays must be provided in full if updating.")
	g.writeLine("if err := json.Unmarshal(partialJSON, &%s); err != nil {", receiverName)
	g.indent++
	g.writeLine(`return fmt.Errorf("failed to apply partial update: %%w", err)`)
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Generate @auto_update fields
	g.writeLine("// Generate @auto_update fields (timestamps)")
	g.generateAutoFields(resource, "update")
	if hasAutoUpdateFields(resource) {
		g.writeLine("")
	}

	// Call BeforeUpdate hook if it exists
	if hasHook(resource, "before", "update") {
		g.writeLine("// Call BeforeUpdate hook")
		g.writeLine("if err := %s.BeforeUpdate(ctx, db); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("before update hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// Call BeforeSave hook if it exists
	if hasHook(resource, "before", "save") {
		g.writeLine("// Call BeforeSave hook")
		g.writeLine("if err := %s.BeforeSave(ctx, db); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("before save hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// Validate merged result AFTER hooks
	g.writeLine("// Validate the merged result after hooks have run")
	g.writeLine("if err := %s.Validate(); err != nil {", receiverName)
	g.indent++
	g.writeLine(`return fmt.Errorf("validation failed: %w", err)`)
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Begin transaction
	g.writeLine("// Begin transaction")
	g.writeLine("tx, err := db.BeginTx(ctx, nil)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine(`return fmt.Errorf("failed to begin transaction: %w", err)`)
	g.indent--
	g.writeLine("}")
	g.writeLine("defer tx.Rollback()")
	g.writeLine("")

	// Build UPDATE query for all fields (same as Update)
	setClauses, values := g.buildUpdateQuery(resource)

	g.writeLine("query := `UPDATE %s SET %s WHERE id = $%d`",
		g.toTableName(resource.Name), strings.Join(setClauses, ", "), len(setClauses)+1)
	g.writeLine("")

	// Add ID to values
	values = append(values, fmt.Sprintf("%s.ID", receiverName))

	// Execute UPDATE
	g.writeLine("// Execute UPDATE")
	g.writeLine("_, err = tx.ExecContext(ctx, query, %s)", strings.Join(values, ", "))
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return fmt.Errorf(\"failed to patch %s: %%w\", err)", strings.ToLower(resource.Name))
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Call AfterUpdate hook if it exists
	if hasHook(resource, "after", "update") {
		g.writeLine("// Call AfterUpdate hook")
		g.writeLine("if err := %s.AfterUpdate(ctx, tx); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("after update hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// Call AfterSave hook if it exists
	if hasHook(resource, "after", "save") {
		g.writeLine("// Call AfterSave hook")
		g.writeLine("if err := %s.AfterSave(ctx, tx); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("after save hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// Commit transaction
	g.writeLine("// Commit transaction")
	g.writeLine("if err := tx.Commit(); err != nil {")
	g.indent++
	g.writeLine(`return fmt.Errorf("failed to commit transaction: %w", err)`)
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

	// 1. Call BeforeDelete hook if it exists
	if hasHook(resource, "before", "delete") {
		g.writeLine("// Call BeforeDelete hook")
		g.writeLine("if err := %s.BeforeDelete(ctx, db); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("before delete hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// 2. Begin transaction
	g.writeLine("// Begin transaction")
	g.writeLine("tx, err := db.BeginTx(ctx, nil)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine(`return fmt.Errorf("failed to begin transaction: %w", err)`)
	g.indent--
	g.writeLine("}")
	g.writeLine("defer tx.Rollback()")
	g.writeLine("")

	// 3. Execute DELETE
	g.writeLine("query := `DELETE FROM %s WHERE id = $1`", g.toTableName(resource.Name))
	g.writeLine("")

	g.writeLine("// Execute DELETE")
	g.writeLine("_, err = tx.ExecContext(ctx, query, %s.ID)", receiverName)
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return fmt.Errorf(\"failed to delete %s: %%w\", err)", strings.ToLower(resource.Name))
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// 4. Call AfterDelete hook if it exists
	if hasHook(resource, "after", "delete") {
		g.writeLine("// Call AfterDelete hook")
		g.writeLine("if err := %s.AfterDelete(ctx, tx); err != nil {", receiverName)
		g.indent++
		g.writeLine(`return fmt.Errorf("after delete hook failed: %w", err)`)
		g.indent--
		g.writeLine("}")
		g.writeLine("")
	}

	// 5. Commit transaction
	g.writeLine("// Commit transaction")
	g.writeLine("if err := tx.Commit(); err != nil {")
	g.indent++
	g.writeLine(`return fmt.Errorf("failed to commit transaction: %w", err)`)
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
		// Skip database-generated ID fields (int with auto_increment)
		// But include client-generated UUID IDs
		if field.Name == "id" && hasConstraint(field, "auto") && field.Type.Name != "uuid" {
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
		// Note: For nullable fields (pointer types), scanTargets will be **T (pointer-to-pointer).
		// The database/sql package handles this correctly:
		// - NULL values: sets the pointer field to nil
		// - Non-NULL values: allocates memory and sets the pointer field to point to the value
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

// hasHook checks if a resource has a specific lifecycle hook
func hasHook(resource *ast.ResourceNode, timing, event string) bool {
	for _, hook := range resource.Hooks {
		if hook.Timing == timing && hook.Event == event {
			return true
		}
	}
	return false
}
