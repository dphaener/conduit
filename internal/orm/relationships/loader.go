package relationships

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// EagerLoad loads relationships for a set of records in batched queries
func (l *Loader) EagerLoad(
	ctx context.Context,
	records []map[string]interface{},
	resource *schema.ResourceSchema,
	includes []string,
) error {
	if len(records) == 0 {
		return nil
	}

	loadCtx := NewLoadContext(10) // Max 10 levels deep
	return l.EagerLoadWithContext(ctx, records, resource, includes, loadCtx)
}

// EagerLoadWithContext loads relationships with circular reference prevention
func (l *Loader) EagerLoadWithContext(
	ctx context.Context,
	records []map[string]interface{},
	resource *schema.ResourceSchema,
	includes []string,
	loadCtx *LoadContext,
) error {
	if len(records) == 0 {
		return nil
	}

	// Check depth limit
	if err := loadCtx.IncrementDepth(); err != nil {
		return err
	}
	defer loadCtx.DecrementDepth()

	// Check for cycles (track by resource type to detect circular relationships)
	// Example: Post -> Author -> Post would create a cycle
	resourceKey := resource.Name
	if !loadCtx.MarkVisited(resourceKey) {
		// Already loading this resource type in the current path - circular reference detected
		// We skip silently to prevent infinite loops, but this is expected behavior for graphs
		return nil
	}
	// Unmark when we exit this level to allow the same resource in different branches
	defer func() {
		loadCtx.mu.Lock()
		delete(loadCtx.visited, resourceKey)
		loadCtx.mu.Unlock()
	}()

	// Load each requested relationship
	for _, include := range includes {
		// Parse nested includes (e.g., "author.posts")
		relation, nestedIncludes := parseInclude(include)

		rel, ok := resource.Relationships[relation]
		if !ok {
			return fmt.Errorf("%w: %s", ErrUnknownRelationship, relation)
		}

		// Load the relationship
		if err := l.loadRelationship(ctx, records, rel, resource); err != nil {
			return fmt.Errorf("failed to load relationship %s: %w", relation, err)
		}

		// If there are nested includes, recursively load them
		if len(nestedIncludes) > 0 && len(records) > 0 {
			targetSchema, ok := l.getSchema(rel.TargetResource)
			if !ok {
				return fmt.Errorf("unknown resource: %s", rel.TargetResource)
			}

			// Extract nested records
			nestedRecords := extractNestedRecords(records, rel)
			if len(nestedRecords) > 0 {
				if err := l.EagerLoadWithContext(ctx, nestedRecords, targetSchema, nestedIncludes, loadCtx); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// loadRelationship loads a single relationship type
func (l *Loader) loadRelationship(
	ctx context.Context,
	records []map[string]interface{},
	rel *schema.Relationship,
	resource *schema.ResourceSchema,
) error {
	switch rel.Type {
	case schema.RelationshipBelongsTo:
		return l.loadBelongsTo(ctx, records, rel, resource)
	case schema.RelationshipHasMany:
		return l.loadHasMany(ctx, records, rel, resource)
	case schema.RelationshipHasOne:
		return l.loadHasOne(ctx, records, rel, resource)
	case schema.RelationshipHasManyThrough:
		return l.loadHasManyThrough(ctx, records, rel, resource)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidRelationType, rel.Type)
	}
}

// LoadSingle loads a single relationship for lazy loading
func (l *Loader) LoadSingle(
	ctx context.Context,
	parentID interface{},
	rel *schema.Relationship,
	resource *schema.ResourceSchema,
) (interface{}, error) {
	switch rel.Type {
	case schema.RelationshipBelongsTo:
		return l.loadSingleBelongsTo(ctx, parentID, rel, resource)
	case schema.RelationshipHasMany:
		return l.loadSingleHasMany(ctx, parentID, rel, resource)
	case schema.RelationshipHasOne:
		return l.loadSingleHasOne(ctx, parentID, rel, resource)
	case schema.RelationshipHasManyThrough:
		return l.loadSingleHasManyThrough(ctx, parentID, rel, resource)
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidRelationType, rel.Type)
	}
}

// parseInclude parses nested includes like "author.posts" into ("author", ["posts"])
func parseInclude(include string) (string, []string) {
	// Simple implementation for now - can be enhanced for complex cases
	// e.g., "author.posts.comments" -> "author", ["posts.comments"]
	for i := 0; i < len(include); i++ {
		if include[i] == '.' {
			return include[:i], []string{include[i+1:]}
		}
	}
	return include, nil
}

// extractNestedRecords extracts nested records from parent records
func extractNestedRecords(records []map[string]interface{}, rel *schema.Relationship) []map[string]interface{} {
	var nested []map[string]interface{}
	seen := make(map[interface{}]bool)

	for _, record := range records {
		relData, ok := record[rel.FieldName]
		if !ok || relData == nil {
			continue
		}

		switch rel.Type {
		case schema.RelationshipBelongsTo, schema.RelationshipHasOne:
			// Single record
			if relMap, ok := relData.(map[string]interface{}); ok {
				if id := relMap["id"]; id != nil && !seen[id] {
					nested = append(nested, relMap)
					seen[id] = true
				}
			}
		case schema.RelationshipHasMany, schema.RelationshipHasManyThrough:
			// Multiple records
			if relSlice, ok := relData.([]map[string]interface{}); ok {
				for _, relMap := range relSlice {
					if id := relMap["id"]; id != nil && !seen[id] {
						nested = append(nested, relMap)
						seen[id] = true
					}
				}
			}
		}
	}

	return nested
}

// scanRow scans a single SQL row into a map
func scanRow(row *sql.Row, schema *schema.ResourceSchema) (map[string]interface{}, error) {
	// Create a map to hold column values
	columns := make([]string, 0, len(schema.Fields))
	for name := range schema.Fields {
		columns = append(columns, name)
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := row.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	record := make(map[string]interface{})
	for i, col := range columns {
		record[col] = values[i]
	}

	return record, nil
}

// scanRows scans multiple SQL rows into a slice of maps
func scanRows(rows *sql.Rows, schema *schema.ResourceSchema) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		record := make(map[string]interface{})
		for i, col := range columns {
			// Handle []byte conversion to string for text fields
			if b, ok := values[i].([]byte); ok {
				record[col] = string(b)
			} else {
				record[col] = values[i]
			}
		}

		results = append(results, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
