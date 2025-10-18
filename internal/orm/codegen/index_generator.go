// Package codegen provides index generation for database schemas
package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// IndexGenerator generates CREATE INDEX statements
type IndexGenerator struct{}

// NewIndexGenerator creates a new index generator
func NewIndexGenerator() *IndexGenerator {
	return &IndexGenerator{}
}

// GenerateIndexes generates CREATE INDEX statements for a resource
func (g *IndexGenerator) GenerateIndexes(resource *schema.ResourceSchema) []string {
	var indexes []string
	tableName := resource.TableName
	if tableName == "" {
		tableName = toSnakeCase(resource.Name)
	}

	// Process fields for @index and @unique annotations
	for fieldName, field := range resource.Fields {
		columnName := toSnakeCase(fieldName)

		// Check for @index annotation
		hasIndex := false
		hasUnique := false

		for _, annotation := range field.Annotations {
			if annotation.Name == "index" {
				hasIndex = true
			}
			if annotation.Name == "unique" {
				hasUnique = true
			}
		}

		// Generate index for @index fields
		if hasIndex {
			indexName := fmt.Sprintf("idx_%s_%s", tableName, columnName)
			indexes = append(indexes,
				fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s);",
					QuoteIdentifier(indexName), QuoteIdentifier(tableName), QuoteIdentifier(columnName)))
		}

		// Generate unique index for @unique fields
		if hasUnique {
			indexName := fmt.Sprintf("idx_%s_%s_unique", tableName, columnName)
			indexes = append(indexes,
				fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s);",
					QuoteIdentifier(indexName), QuoteIdentifier(tableName), QuoteIdentifier(columnName)))
		}
	}

	// Sort for deterministic output
	sort.Strings(indexes)

	return indexes
}

// GenerateForeignKeyIndexes generates indexes on foreign key columns
func (g *IndexGenerator) GenerateForeignKeyIndexes(resource *schema.ResourceSchema) []string {
	var indexes []string
	tableName := resource.TableName
	if tableName == "" {
		tableName = toSnakeCase(resource.Name)
	}

	// Process belongs_to relationships
	for _, rel := range resource.Relationships {
		if rel.Type != schema.RelationshipBelongsTo {
			continue
		}

		// Determine foreign key column name
		foreignKeyColumn := rel.ForeignKey
		if foreignKeyColumn == "" {
			foreignKeyColumn = toSnakeCase(rel.TargetResource) + "_id"
		} else {
			foreignKeyColumn = toSnakeCase(foreignKeyColumn)
		}

		// Check if the field already has an index
		hasExistingIndex := false
		if field, exists := resource.Fields[rel.FieldName]; exists {
			for _, annotation := range field.Annotations {
				if annotation.Name == "index" || annotation.Name == "unique" {
					hasExistingIndex = true
					break
				}
			}
		}

		// Only create index if one doesn't already exist
		if !hasExistingIndex {
			indexName := fmt.Sprintf("idx_%s_%s", tableName, foreignKeyColumn)
			indexes = append(indexes,
				fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s);",
					QuoteIdentifier(indexName), QuoteIdentifier(tableName), QuoteIdentifier(foreignKeyColumn)))
		}
	}

	// Sort for deterministic output
	sort.Strings(indexes)

	return indexes
}

// GenerateAllIndexes generates all indexes for a resource (field + FK)
func (g *IndexGenerator) GenerateAllIndexes(resource *schema.ResourceSchema) []string {
	var allIndexes []string

	// Field indexes
	fieldIndexes := g.GenerateIndexes(resource)
	allIndexes = append(allIndexes, fieldIndexes...)

	// Foreign key indexes
	fkIndexes := g.GenerateForeignKeyIndexes(resource)
	allIndexes = append(allIndexes, fkIndexes...)

	// Remove duplicates and sort
	indexMap := make(map[string]bool)
	uniqueIndexes := make([]string, 0)
	for _, idx := range allIndexes {
		if !indexMap[idx] {
			indexMap[idx] = true
			uniqueIndexes = append(uniqueIndexes, idx)
		}
	}

	sort.Strings(uniqueIndexes)

	return uniqueIndexes
}

// GenerateDropIndexes generates DROP INDEX statements
func (g *IndexGenerator) GenerateDropIndexes(resource *schema.ResourceSchema) []string {
	var dropStatements []string
	tableName := resource.TableName
	if tableName == "" {
		tableName = toSnakeCase(resource.Name)
	}

	// Drop field indexes
	for fieldName, field := range resource.Fields {
		columnName := toSnakeCase(fieldName)

		hasIndex := false
		hasUnique := false

		for _, annotation := range field.Annotations {
			if annotation.Name == "index" {
				hasIndex = true
			}
			if annotation.Name == "unique" {
				hasUnique = true
			}
		}

		if hasIndex {
			indexName := fmt.Sprintf("idx_%s_%s", tableName, columnName)
			dropStatements = append(dropStatements,
				fmt.Sprintf("DROP INDEX IF EXISTS %s;", QuoteIdentifier(indexName)))
		}

		if hasUnique {
			indexName := fmt.Sprintf("idx_%s_%s_unique", tableName, columnName)
			dropStatements = append(dropStatements,
				fmt.Sprintf("DROP INDEX IF EXISTS %s;", QuoteIdentifier(indexName)))
		}
	}

	// Drop foreign key indexes
	for _, rel := range resource.Relationships {
		if rel.Type != schema.RelationshipBelongsTo {
			continue
		}

		foreignKeyColumn := rel.ForeignKey
		if foreignKeyColumn == "" {
			foreignKeyColumn = toSnakeCase(rel.TargetResource) + "_id"
		} else {
			foreignKeyColumn = toSnakeCase(foreignKeyColumn)
		}

		indexName := fmt.Sprintf("idx_%s_%s", tableName, foreignKeyColumn)
		dropStatements = append(dropStatements,
			fmt.Sprintf("DROP INDEX IF EXISTS %s;", QuoteIdentifier(indexName)))
	}

	// Remove duplicates and sort
	indexMap := make(map[string]bool)
	uniqueDrops := make([]string, 0)
	for _, stmt := range dropStatements {
		if !indexMap[stmt] {
			indexMap[stmt] = true
			uniqueDrops = append(uniqueDrops, stmt)
		}
	}

	sort.Strings(uniqueDrops)

	return uniqueDrops
}

// GenerateCompositeIndex generates a composite index on multiple columns
func (g *IndexGenerator) GenerateCompositeIndex(tableName string, indexName string, columns []string, unique bool) string {
	// Quote each column name individually
	quotedColumns := make([]string, len(columns))
	for i, col := range columns {
		quotedColumns[i] = QuoteIdentifier(col)
	}

	if unique {
		return fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s);",
			QuoteIdentifier(indexName), QuoteIdentifier(tableName), strings.Join(quotedColumns, ", "))
	}
	return fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s);",
		QuoteIdentifier(indexName), QuoteIdentifier(tableName), strings.Join(quotedColumns, ", "))
}

// GeneratePartialIndex generates a partial index with a WHERE clause
func (g *IndexGenerator) GeneratePartialIndex(tableName string, indexName string, column string, whereClause string) string {
	return fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s) WHERE %s;",
		QuoteIdentifier(indexName), QuoteIdentifier(tableName), QuoteIdentifier(column), whereClause)
}

// GenerateExpressionIndex generates an index on an expression
func (g *IndexGenerator) GenerateExpressionIndex(tableName string, indexName string, expression string) string {
	return fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s);",
		QuoteIdentifier(indexName), QuoteIdentifier(tableName), expression)
}
