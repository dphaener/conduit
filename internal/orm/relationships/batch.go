package relationships

import (
	"context"
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/orm/schema"
	"github.com/lib/pq"
)

// loadBelongsTo loads belongs-to relationships using a batched IN query
// Example: Post belongs_to User
//   - Collect all unique author_ids from posts
//   - Single query: SELECT * FROM users WHERE id = ANY($1)
//   - Map users back to posts
func (l *Loader) loadBelongsTo(
	ctx context.Context,
	records []map[string]interface{},
	rel *schema.Relationship,
	resource *schema.ResourceSchema,
) error {
	// Determine foreign key column
	fk := rel.ForeignKey
	if fk == "" {
		fk = toSnakeCase(rel.TargetResource) + "_id"
	}

	// Collect foreign key IDs
	var ids []interface{}
	idMap := make(map[string][]int) // id -> record indices

	for i, record := range records {
		if id, ok := record[fk]; ok && id != nil {
			idStr, err := idToString(id)
			if err != nil {
				return fmt.Errorf("invalid foreign key type for %s: %w", fk, err)
			}
			if _, seen := idMap[idStr]; !seen {
				ids = append(ids, id)
			}
			idMap[idStr] = append(idMap[idStr], i)
		}
	}

	if len(ids) == 0 {
		// No foreign keys to load - set all to nil
		for _, record := range records {
			if rel.Nullable {
				record[rel.FieldName] = nil
			}
		}
		return nil
	}

	// Get target schema (thread-safe)
	targetSchema, ok := l.getSchema(rel.TargetResource)
	if !ok {
		return fmt.Errorf("unknown resource: %s", rel.TargetResource)
	}

	// Build and execute query
	tableName := toTableName(rel.TargetResource)
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ANY($1)", pq.QuoteIdentifier(tableName))

	rows, err := l.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to query belongs_to relationship: %w", err)
	}
	defer rows.Close()

	// Map results back to parent records
	related := make(map[string]map[string]interface{})
	results, err := scanRows(rows, targetSchema)
	if err != nil {
		return fmt.Errorf("failed to scan belongs_to records: %w", err)
	}

	for _, record := range results {
		idStr, err := idToString(record["id"])
		if err != nil {
			return fmt.Errorf("invalid ID type in results: %w", err)
		}
		related[idStr] = record
	}

	// Attach to parent records
	for _, record := range records {
		if id, ok := record[fk]; ok && id != nil {
			idStr, err := idToString(id)
			if err != nil {
				return fmt.Errorf("invalid foreign key value: %w", err)
			}
			if relRecord, ok := related[idStr]; ok {
				record[rel.FieldName] = relRecord
			} else if rel.Nullable {
				record[rel.FieldName] = nil
			}
		} else if rel.Nullable {
			record[rel.FieldName] = nil
		}
	}

	return nil
}

// loadHasMany loads has-many relationships using a batched IN query
// Example: Post has_many Comment
//   - Collect all post IDs
//   - Single query: SELECT * FROM comments WHERE post_id = ANY($1)
//   - Group comments by post_id
//   - Attach to posts
func (l *Loader) loadHasMany(
	ctx context.Context,
	records []map[string]interface{},
	rel *schema.Relationship,
	resource *schema.ResourceSchema,
) error {
	// Collect parent IDs
	var parentIDs []interface{}
	for _, record := range records {
		if id, ok := record["id"]; ok && id != nil {
			// Validate ID type
			if _, err := idToString(id); err != nil {
				return fmt.Errorf("invalid parent ID type: %w", err)
			}
			parentIDs = append(parentIDs, id)
		}
	}

	if len(parentIDs) == 0 {
		return nil
	}

	// Determine foreign key column
	fk := rel.ForeignKey
	if fk == "" {
		// Default: parent_resource_id (e.g., post_id)
		fk = toSnakeCase(resource.Name) + "_id"
	}

	// Get target schema (thread-safe)
	targetSchema, ok := l.getSchema(rel.TargetResource)
	if !ok {
		return fmt.Errorf("unknown resource: %s", rel.TargetResource)
	}

	// Build and execute query
	tableName := toTableName(rel.TargetResource)
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ANY($1)", pq.QuoteIdentifier(tableName), pq.QuoteIdentifier(fk))

	if rel.OrderBy != "" {
		// Split OrderBy clause to quote identifiers
		query += fmt.Sprintf(" ORDER BY %s", quoteSortClause(rel.OrderBy))
	}

	rows, err := l.db.QueryContext(ctx, query, pq.Array(parentIDs))
	if err != nil {
		return fmt.Errorf("failed to query has_many relationship: %w", err)
	}
	defer rows.Close()

	// Group by parent ID
	grouped := make(map[string][]map[string]interface{})
	results, err := scanRows(rows, targetSchema)
	if err != nil {
		return fmt.Errorf("failed to scan has_many records: %w", err)
	}

	for _, record := range results {
		parentIDStr, err := idToString(record[fk])
		if err != nil {
			return fmt.Errorf("invalid parent ID in results: %w", err)
		}
		grouped[parentIDStr] = append(grouped[parentIDStr], record)
	}

	// Attach to parent records (always return empty slice, not nil)
	for _, record := range records {
		idStr, err := idToString(record["id"])
		if err != nil {
			return fmt.Errorf("invalid parent record ID: %w", err)
		}
		if children, ok := grouped[idStr]; ok {
			record[rel.FieldName] = children
		} else {
			record[rel.FieldName] = []map[string]interface{}{}
		}
	}

	return nil
}

// loadHasOne loads has-one relationships using a batched IN query
// Example: User has_one Profile
//   - Collect all user IDs
//   - Single query: SELECT DISTINCT ON (user_id) * FROM profiles WHERE user_id = ANY($1)
//   - Map profiles back to users
func (l *Loader) loadHasOne(
	ctx context.Context,
	records []map[string]interface{},
	rel *schema.Relationship,
	resource *schema.ResourceSchema,
) error {
	// Collect parent IDs
	var parentIDs []interface{}
	for _, record := range records {
		if id, ok := record["id"]; ok && id != nil {
			// Validate ID type
			if _, err := idToString(id); err != nil {
				return fmt.Errorf("invalid parent ID type: %w", err)
			}
			parentIDs = append(parentIDs, id)
		}
	}

	if len(parentIDs) == 0 {
		return nil
	}

	// Determine foreign key column
	fk := rel.ForeignKey
	if fk == "" {
		fk = toSnakeCase(resource.Name) + "_id"
	}

	// Get target schema (thread-safe)
	targetSchema, ok := l.getSchema(rel.TargetResource)
	if !ok {
		return fmt.Errorf("unknown resource: %s", rel.TargetResource)
	}

	// Build and execute query with DISTINCT ON to ensure only one record per parent
	tableName := toTableName(rel.TargetResource)
	query := fmt.Sprintf(
		"SELECT DISTINCT ON (%s) * FROM %s WHERE %s = ANY($1) ORDER BY %s, id",
		pq.QuoteIdentifier(fk), pq.QuoteIdentifier(tableName), pq.QuoteIdentifier(fk), pq.QuoteIdentifier(fk),
	)

	rows, err := l.db.QueryContext(ctx, query, pq.Array(parentIDs))
	if err != nil {
		return fmt.Errorf("failed to query has_one relationship: %w", err)
	}
	defer rows.Close()

	// Map results to parent records
	related := make(map[string]map[string]interface{})
	results, err := scanRows(rows, targetSchema)
	if err != nil {
		return fmt.Errorf("failed to scan has_one records: %w", err)
	}

	for _, record := range results {
		parentIDStr, err := idToString(record[fk])
		if err != nil {
			return fmt.Errorf("invalid parent ID in results: %w", err)
		}
		related[parentIDStr] = record
	}

	// Attach to parent records
	for _, record := range records {
		idStr, err := idToString(record["id"])
		if err != nil {
			return fmt.Errorf("invalid parent record ID: %w", err)
		}
		if relRecord, ok := related[idStr]; ok {
			record[rel.FieldName] = relRecord
		} else if rel.Nullable {
			record[rel.FieldName] = nil
		}
	}

	return nil
}

// loadHasManyThrough loads has-many-through relationships using a JOIN query
// Example: Post has_many Tag through PostTag
//   - Three-way join through junction table
//   - Single query with JOIN
//   - Group by parent ID
func (l *Loader) loadHasManyThrough(
	ctx context.Context,
	records []map[string]interface{},
	rel *schema.Relationship,
	resource *schema.ResourceSchema,
) error {
	// Collect parent IDs
	var parentIDs []interface{}
	for _, record := range records {
		if id, ok := record["id"]; ok && id != nil {
			// Validate ID type
			if _, err := idToString(id); err != nil {
				return fmt.Errorf("invalid parent ID type: %w", err)
			}
			parentIDs = append(parentIDs, id)
		}
	}

	if len(parentIDs) == 0 {
		return nil
	}

	// Get target schema (thread-safe)
	targetSchema, ok := l.getSchema(rel.TargetResource)
	if !ok {
		return fmt.Errorf("unknown resource: %s", rel.TargetResource)
	}

	// Determine foreign keys
	sourceFk := rel.ForeignKey
	if sourceFk == "" {
		sourceFk = toSnakeCase(resource.Name) + "_id"
	}

	targetFk := rel.AssociationKey
	if targetFk == "" {
		targetFk = toSnakeCase(rel.TargetResource) + "_id"
	}

	// Build and execute join query
	targetTable := toTableName(rel.TargetResource)
	joinTable := rel.JoinTable
	if joinTable == "" {
		// Default join table name
		joinTable = toSnakeCase(resource.Name) + "_" + toSnakeCase(rel.TargetResource) + "s"
	}

	query := fmt.Sprintf(`
		SELECT t.*, j.%s as __parent_id
		FROM %s t
		INNER JOIN %s j ON t.id = j.%s
		WHERE j.%s = ANY($1)
	`, pq.QuoteIdentifier(sourceFk), pq.QuoteIdentifier(targetTable), pq.QuoteIdentifier(joinTable), pq.QuoteIdentifier(targetFk), pq.QuoteIdentifier(sourceFk))

	if rel.OrderBy != "" {
		// Split OrderBy clause to quote identifiers
		query += fmt.Sprintf(" ORDER BY %s", quoteSortClause(rel.OrderBy))
	}

	rows, err := l.db.QueryContext(ctx, query, pq.Array(parentIDs))
	if err != nil {
		return fmt.Errorf("failed to query has_many_through relationship: %w", err)
	}
	defer rows.Close()

	// Group by parent ID
	grouped := make(map[string][]map[string]interface{})
	results, err := scanRows(rows, targetSchema)
	if err != nil {
		return fmt.Errorf("failed to scan has_many_through records: %w", err)
	}

	for _, record := range results {
		parentIDStr, err := idToString(record["__parent_id"])
		if err != nil {
			return fmt.Errorf("invalid parent ID in through results: %w", err)
		}
		delete(record, "__parent_id") // Remove join artifact
		grouped[parentIDStr] = append(grouped[parentIDStr], record)
	}

	// Attach to parent records (always return empty slice, not nil)
	for _, record := range records {
		idStr, err := idToString(record["id"])
		if err != nil {
			return fmt.Errorf("invalid parent record ID: %w", err)
		}
		if related, ok := grouped[idStr]; ok {
			record[rel.FieldName] = related
		} else {
			record[rel.FieldName] = []map[string]interface{}{}
		}
	}

	return nil
}

// Lazy loading implementations - load single records

func (l *Loader) loadSingleBelongsTo(
	ctx context.Context,
	foreignKey interface{},
	rel *schema.Relationship,
	resource *schema.ResourceSchema,
) (map[string]interface{}, error) {
	targetSchema, ok := l.schemas[rel.TargetResource]
	if !ok {
		return nil, fmt.Errorf("unknown resource: %s", rel.TargetResource)
	}

	tableName := toTableName(rel.TargetResource)
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", pq.QuoteIdentifier(tableName))

	rows, err := l.db.QueryContext(ctx, query, foreignKey)
	if err != nil {
		return nil, fmt.Errorf("failed to query single belongs_to: %w", err)
	}
	defer rows.Close()

	results, err := scanRows(rows, targetSchema)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		if rel.Nullable {
			return nil, nil
		}
		return nil, ErrNoRecords
	}

	return results[0], nil
}

func (l *Loader) loadSingleHasMany(
	ctx context.Context,
	parentID interface{},
	rel *schema.Relationship,
	resource *schema.ResourceSchema,
) ([]map[string]interface{}, error) {
	fk := rel.ForeignKey
	if fk == "" {
		fk = toSnakeCase(resource.Name) + "_id"
	}

	targetSchema, ok := l.schemas[rel.TargetResource]
	if !ok {
		return nil, fmt.Errorf("unknown resource: %s", rel.TargetResource)
	}

	tableName := toTableName(rel.TargetResource)
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", pq.QuoteIdentifier(tableName), pq.QuoteIdentifier(fk))

	if rel.OrderBy != "" {
		query += fmt.Sprintf(" ORDER BY %s", quoteSortClause(rel.OrderBy))
	}

	rows, err := l.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query single has_many: %w", err)
	}
	defer rows.Close()

	results, err := scanRows(rows, targetSchema)
	if err != nil {
		return nil, err
	}

	// Always return empty slice, not nil
	if results == nil {
		return []map[string]interface{}{}, nil
	}

	return results, nil
}

func (l *Loader) loadSingleHasOne(
	ctx context.Context,
	parentID interface{},
	rel *schema.Relationship,
	resource *schema.ResourceSchema,
) (map[string]interface{}, error) {
	fk := rel.ForeignKey
	if fk == "" {
		fk = toSnakeCase(resource.Name) + "_id"
	}

	targetSchema, ok := l.schemas[rel.TargetResource]
	if !ok {
		return nil, fmt.Errorf("unknown resource: %s", rel.TargetResource)
	}

	tableName := toTableName(rel.TargetResource)
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1 LIMIT 1", pq.QuoteIdentifier(tableName), pq.QuoteIdentifier(fk))

	rows, err := l.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query single has_one: %w", err)
	}
	defer rows.Close()

	results, err := scanRows(rows, targetSchema)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		if rel.Nullable {
			return nil, nil
		}
		return nil, ErrNoRecords
	}

	return results[0], nil
}

func (l *Loader) loadSingleHasManyThrough(
	ctx context.Context,
	parentID interface{},
	rel *schema.Relationship,
	resource *schema.ResourceSchema,
) ([]map[string]interface{}, error) {
	targetSchema, ok := l.schemas[rel.TargetResource]
	if !ok {
		return nil, fmt.Errorf("unknown resource: %s", rel.TargetResource)
	}

	sourceFk := rel.ForeignKey
	if sourceFk == "" {
		sourceFk = toSnakeCase(resource.Name) + "_id"
	}

	targetFk := rel.AssociationKey
	if targetFk == "" {
		targetFk = toSnakeCase(rel.TargetResource) + "_id"
	}

	targetTable := toTableName(rel.TargetResource)
	joinTable := rel.JoinTable
	if joinTable == "" {
		joinTable = toSnakeCase(resource.Name) + "_" + toSnakeCase(rel.TargetResource) + "s"
	}

	query := fmt.Sprintf(`
		SELECT t.*
		FROM %s t
		INNER JOIN %s j ON t.id = j.%s
		WHERE j.%s = $1
	`, pq.QuoteIdentifier(targetTable), pq.QuoteIdentifier(joinTable), pq.QuoteIdentifier(targetFk), pq.QuoteIdentifier(sourceFk))

	if rel.OrderBy != "" {
		query += fmt.Sprintf(" ORDER BY %s", quoteSortClause(rel.OrderBy))
	}

	rows, err := l.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query single has_many_through: %w", err)
	}
	defer rows.Close()

	results, err := scanRows(rows, targetSchema)
	if err != nil {
		return nil, err
	}

	// Always return empty slice, not nil
	if results == nil {
		return []map[string]interface{}{}, nil
	}

	return results, nil
}

// toSnakeCase converts a string to snake_case
func toSnakeCase(s string) string {
	var result []rune
	runes := []rune(s)

	for i, r := range runes {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := runes[i-1]
			if prev >= 'a' && prev <= 'z' {
				result = append(result, '_')
			} else if i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z' {
				result = append(result, '_')
			}
		}
		if r >= 'A' && r <= 'Z' {
			result = append(result, r+('a'-'A'))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// toTableName converts a resource name to a table name (snake_case plural)
func toTableName(resourceName string) string {
	snake := toSnakeCase(resourceName)
	return pluralize(snake)
}

// pluralize adds simple pluralization
func pluralize(s string) string {
	if strings.HasSuffix(s, "s") ||
		strings.HasSuffix(s, "x") ||
		strings.HasSuffix(s, "z") {
		return s + "es"
	}
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	return s + "s"
}

// quoteSortClause safely quotes column identifiers in an ORDER BY clause
// Handles formats like "created_at DESC" or "name ASC, id DESC"
func quoteSortClause(orderBy string) string {
	parts := strings.Split(orderBy, ",")
	quoted := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		tokens := strings.Fields(part)

		if len(tokens) == 0 {
			continue
		}

		// Quote the column name (first token)
		quotedCol := pq.QuoteIdentifier(tokens[0])

		// Preserve direction (ASC/DESC) if present
		if len(tokens) > 1 {
			direction := strings.ToUpper(tokens[1])
			if direction == "ASC" || direction == "DESC" {
				quoted = append(quoted, quotedCol+" "+direction)
			} else {
				quoted = append(quoted, quotedCol)
			}
		} else {
			quoted = append(quoted, quotedCol)
		}
	}

	return strings.Join(quoted, ", ")
}

// idToString efficiently converts an ID to a string with type validation
// Supports common ID types: string, int, int64, []byte (UUID)
func idToString(id interface{}) (string, error) {
	if id == nil {
		return "", fmt.Errorf("ID cannot be nil")
	}

	switch v := id.(type) {
	case string:
		return v, nil
	case int:
		// More efficient than fmt.Sprintf for integers
		return fmt.Sprintf("%d", v), nil
	case int64:
		return fmt.Sprintf("%d", v), nil
	case int32:
		return fmt.Sprintf("%d", v), nil
	case uint:
		return fmt.Sprintf("%d", v), nil
	case uint64:
		return fmt.Sprintf("%d", v), nil
	case []byte:
		// UUID stored as bytes
		return string(v), nil
	default:
		// Fallback for other types (e.g., custom UUID types)
		// This is less efficient but handles edge cases
		return fmt.Sprintf("%v", v), nil
	}
}
