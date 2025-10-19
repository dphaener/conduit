// Package query provides query building functionality for the Conduit ORM
package query

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// RelationshipLoader is an interface for loading relationships
// This avoids circular dependencies between query and relationships packages
type RelationshipLoader interface {
	EagerLoad(ctx context.Context, records []map[string]interface{}, resource *schema.ResourceSchema, includes []string) error
}

// QueryBuilder provides a fluent API for building SQL queries
type QueryBuilder struct {
	resource     *schema.ResourceSchema
	db           *sql.DB
	schemas      map[string]*schema.ResourceSchema
	loader       RelationshipLoader // Optional relationship loader

	conditions   []*Condition
	joins        []*Join
	orderBy      []string
	groupBy      []string
	having       []*Condition
	limit        *int
	offset       *int
	includes     []string // For eager loading
	scopeNames   []string // Applied scope names

	// For building SQL
	paramCounter int
	args         []interface{}
}

// JoinType represents the type of SQL join
type JoinType int

const (
	InnerJoin JoinType = iota
	LeftJoin
	RightJoin
)

// String returns the string representation of the join type
func (j JoinType) String() string {
	switch j {
	case InnerJoin:
		return "INNER"
	case LeftJoin:
		return "LEFT"
	case RightJoin:
		return "RIGHT"
	default:
		return "INNER"
	}
}

// Join represents a SQL join clause
type Join struct {
	Type      JoinType
	Table     string
	Condition string
	Alias     string
}

// NewQueryBuilder creates a new query builder for the given resource
func NewQueryBuilder(resource *schema.ResourceSchema, db *sql.DB, schemas map[string]*schema.ResourceSchema) *QueryBuilder {
	return &QueryBuilder{
		resource:     resource,
		db:           db,
		schemas:      schemas,
		loader:       nil, // Set via WithLoader() if needed
		conditions:   make([]*Condition, 0),
		joins:        make([]*Join, 0),
		orderBy:      make([]string, 0),
		groupBy:      make([]string, 0),
		having:       make([]*Condition, 0),
		includes:     make([]string, 0),
		scopeNames:   make([]string, 0),
		paramCounter: 1,
		args:         make([]interface{}, 0),
	}
}

// WithLoader sets the relationship loader for eager loading
func (qb *QueryBuilder) WithLoader(loader RelationshipLoader) *QueryBuilder {
	qb.loader = loader
	return qb
}

// Where adds a WHERE condition to the query
func (qb *QueryBuilder) Where(field string, op Operator, value interface{}) *QueryBuilder {
	// Validate field exists in schema
	if qb.resource != nil {
		if _, exists := qb.resource.Fields[field]; !exists {
			panic(fmt.Sprintf("field %s does not exist on resource %s", field, qb.resource.Name))
		}
	}

	qb.conditions = append(qb.conditions, &Condition{
		Field:    field,
		Operator: op,
		Value:    value,
		Or:       false,
	})
	return qb
}

// OrWhere adds an OR WHERE condition to the query
func (qb *QueryBuilder) OrWhere(field string, op Operator, value interface{}) *QueryBuilder {
	// Validate field exists in schema
	if qb.resource != nil {
		if _, exists := qb.resource.Fields[field]; !exists {
			panic(fmt.Sprintf("field %s does not exist on resource %s", field, qb.resource.Name))
		}
	}

	qb.conditions = append(qb.conditions, &Condition{
		Field:    field,
		Operator: op,
		Value:    value,
		Or:       true,
	})
	return qb
}

// WhereIn adds a WHERE IN condition
func (qb *QueryBuilder) WhereIn(field string, values []interface{}) *QueryBuilder {
	return qb.Where(field, OpIn, values)
}

// WhereNotIn adds a WHERE NOT IN condition
func (qb *QueryBuilder) WhereNotIn(field string, values []interface{}) *QueryBuilder {
	return qb.Where(field, OpNotIn, values)
}

// WhereNull adds a WHERE IS NULL condition
func (qb *QueryBuilder) WhereNull(field string) *QueryBuilder {
	return qb.Where(field, OpIsNull, nil)
}

// WhereNotNull adds a WHERE IS NOT NULL condition
func (qb *QueryBuilder) WhereNotNull(field string) *QueryBuilder {
	return qb.Where(field, OpIsNotNull, nil)
}

// WhereLike adds a WHERE LIKE condition
func (qb *QueryBuilder) WhereLike(field string, pattern string) *QueryBuilder {
	return qb.Where(field, OpLike, pattern)
}

// WhereILike adds a WHERE ILIKE condition (case-insensitive)
func (qb *QueryBuilder) WhereILike(field string, pattern string) *QueryBuilder {
	return qb.Where(field, OpILike, pattern)
}

// WhereBetween adds a WHERE BETWEEN condition
func (qb *QueryBuilder) WhereBetween(field string, min, max interface{}) *QueryBuilder {
	return qb.Where(field, OpBetween, []interface{}{min, max})
}

// OrderBy adds an ORDER BY clause
func (qb *QueryBuilder) OrderBy(field string, direction string) *QueryBuilder {
	dir := strings.ToUpper(direction)
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}
	qb.orderBy = append(qb.orderBy, fmt.Sprintf("%s %s", field, dir))
	return qb
}

// OrderByAsc adds an ascending ORDER BY clause
func (qb *QueryBuilder) OrderByAsc(field string) *QueryBuilder {
	return qb.OrderBy(field, "ASC")
}

// OrderByDesc adds a descending ORDER BY clause
func (qb *QueryBuilder) OrderByDesc(field string) *QueryBuilder {
	return qb.OrderBy(field, "DESC")
}

// GroupBy adds a GROUP BY clause
func (qb *QueryBuilder) GroupBy(fields ...string) *QueryBuilder {
	qb.groupBy = append(qb.groupBy, fields...)
	return qb
}

// Having adds a HAVING condition
func (qb *QueryBuilder) Having(field string, op Operator, value interface{}) *QueryBuilder {
	// Note: HAVING clauses can reference aggregates like COUNT(*), so we only validate
	// if the field doesn't contain parentheses (which indicates an aggregate function)
	if qb.resource != nil && !strings.Contains(field, "(") {
		if _, exists := qb.resource.Fields[field]; !exists {
			panic(fmt.Sprintf("field %s does not exist on resource %s", field, qb.resource.Name))
		}
	}

	qb.having = append(qb.having, &Condition{
		Field:    field,
		Operator: op,
		Value:    value,
		Or:       false,
	})
	return qb
}

// Limit sets the LIMIT clause
func (qb *QueryBuilder) Limit(n int) *QueryBuilder {
	qb.limit = &n
	return qb
}

// Offset sets the OFFSET clause
func (qb *QueryBuilder) Offset(n int) *QueryBuilder {
	qb.offset = &n
	return qb
}

// Includes adds relationships to eager load
func (qb *QueryBuilder) Includes(relationships ...string) *QueryBuilder {
	qb.includes = append(qb.includes, relationships...)
	return qb
}

// Join adds a JOIN clause
func (qb *QueryBuilder) Join(joinType JoinType, table string, condition string) *QueryBuilder {
	validateIdentifier(table)
	if !isValidJoinCondition(condition) {
		panic(fmt.Sprintf("invalid join condition: %s", condition))
	}
	qb.joins = append(qb.joins, &Join{
		Type:      joinType,
		Table:     table,
		Condition: condition,
	})
	return qb
}

// InnerJoin adds an INNER JOIN clause
func (qb *QueryBuilder) InnerJoin(table string, condition string) *QueryBuilder {
	return qb.Join(InnerJoin, table, condition)
}

// LeftJoin adds a LEFT JOIN clause
func (qb *QueryBuilder) LeftJoin(table string, condition string) *QueryBuilder {
	return qb.Join(LeftJoin, table, condition)
}

// RightJoin adds a RIGHT JOIN clause
func (qb *QueryBuilder) RightJoin(table string, condition string) *QueryBuilder {
	return qb.Join(RightJoin, table, condition)
}

// Scope applies a named scope to the query
func (qb *QueryBuilder) Scope(scopeName string, args ...interface{}) (*QueryBuilder, error) {
	scope, ok := qb.resource.Scopes[scopeName]
	if !ok {
		return nil, fmt.Errorf("unknown scope: %s", scopeName)
	}

	// Bind and apply the scope
	if err := qb.applyScope(scope, args); err != nil {
		return nil, err
	}

	qb.scopeNames = append(qb.scopeNames, scopeName)
	return qb, nil
}

// applyScope applies a scope to the query builder
func (qb *QueryBuilder) applyScope(scope *schema.Scope, args []interface{}) error {
	// Validate argument count
	if len(args) != len(scope.Arguments) {
		return fmt.Errorf("scope %s expects %d arguments, got %d",
			scope.Name, len(scope.Arguments), len(args))
	}

	// Build argument map for substitution
	argMap := make(map[string]interface{})
	for i, argDef := range scope.Arguments {
		argMap[argDef.Name] = args[i]
	}

	// Apply where conditions from scope
	// Parse the scope.Where map which contains field -> condition mappings
	for field, conditionExpr := range scope.Where {
		// Parse the condition expression
		// Format could be: "= $arg_name", "> 100", "IN $categories", etc.
		exprStr, ok := conditionExpr.(string)
		if !ok {
			return fmt.Errorf("scope condition must be a string, got %T", conditionExpr)
		}
		cond, err := parseScopeCondition(field, exprStr, argMap)
		if err != nil {
			return fmt.Errorf("failed to parse scope condition: %w", err)
		}
		qb.conditions = append(qb.conditions, cond)
	}

	// Apply order by
	if scope.OrderBy != "" {
		// Parse "field_name DESC" or "field_name ASC"
		qb.orderBy = append(qb.orderBy, scope.OrderBy)
	}

	// Apply limit
	if scope.Limit != nil {
		qb.limit = scope.Limit
	}

	// Apply offset
	if scope.Offset != nil {
		qb.offset = scope.Offset
	}

	return nil
}

// ToSQL generates the SQL query and parameter bindings
func (qb *QueryBuilder) ToSQL() (string, []interface{}, error) {
	var sql strings.Builder
	qb.args = make([]interface{}, 0)
	qb.paramCounter = 1

	tableName := toTableName(qb.resource.Name)
	sql.WriteString(fmt.Sprintf("SELECT * FROM %s", tableName))

	// JOINs
	for _, join := range qb.joins {
		sql.WriteString(fmt.Sprintf(" %s JOIN %s ON %s",
			join.Type.String(),
			join.Table,
			join.Condition,
		))
	}

	// WHERE clauses
	if len(qb.conditions) > 0 {
		sql.WriteString(" WHERE ")
		for i, cond := range qb.conditions {
			if i > 0 {
				if cond.Or {
					sql.WriteString(" OR ")
				} else {
					sql.WriteString(" AND ")
				}
			}
			condSQL, err := qb.conditionToSQL(cond)
			if err != nil {
				return "", nil, fmt.Errorf("failed to build condition: %w", err)
			}
			sql.WriteString(condSQL)
		}
	}

	// GROUP BY
	if len(qb.groupBy) > 0 {
		sql.WriteString(" GROUP BY ")
		sql.WriteString(strings.Join(qb.groupBy, ", "))
	}

	// HAVING
	if len(qb.having) > 0 {
		sql.WriteString(" HAVING ")
		for i, cond := range qb.having {
			if i > 0 {
				if cond.Or {
					sql.WriteString(" OR ")
				} else {
					sql.WriteString(" AND ")
				}
			}
			condSQL, err := qb.conditionToSQL(cond)
			if err != nil {
				return "", nil, fmt.Errorf("failed to build having condition: %w", err)
			}
			sql.WriteString(condSQL)
		}
	}

	// ORDER BY
	if len(qb.orderBy) > 0 {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(strings.Join(qb.orderBy, ", "))
	}

	// LIMIT
	if qb.limit != nil {
		sql.WriteString(fmt.Sprintf(" LIMIT $%d", qb.paramCounter))
		qb.args = append(qb.args, *qb.limit)
		qb.paramCounter++
	}

	// OFFSET
	if qb.offset != nil {
		sql.WriteString(fmt.Sprintf(" OFFSET $%d", qb.paramCounter))
		qb.args = append(qb.args, *qb.offset)
		qb.paramCounter++
	}

	return sql.String(), qb.args, nil
}

// conditionToSQL converts a condition to SQL
func (qb *QueryBuilder) conditionToSQL(cond *Condition) (string, error) {
	return conditionToSQL(cond, &qb.paramCounter, &qb.args)
}

// All executes the query and returns all matching rows
func (qb *QueryBuilder) All(ctx context.Context) ([]map[string]interface{}, error) {
	sql, args, err := qb.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	rows, err := qb.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	results, err := scanRows(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan rows: %w", err)
	}

	// Eager load relationships if any were specified
	if len(qb.includes) > 0 && len(results) > 0 {
		if err := qb.loadRelationships(ctx, results); err != nil {
			return nil, fmt.Errorf("failed to load relationships: %w", err)
		}
	}

	return results, nil
}

// loadRelationships loads the specified relationships for the given records
func (qb *QueryBuilder) loadRelationships(ctx context.Context, records []map[string]interface{}) error {
	if len(qb.includes) == 0 || len(records) == 0 {
		return nil
	}

	// If no loader is configured, skip relationship loading
	// This happens in tests or when the QueryBuilder is used standalone
	if qb.loader == nil {
		return nil
	}

	// Delegate to the relationship loader
	return qb.loader.EagerLoad(ctx, records, qb.resource, qb.includes)
}

// First executes the query and returns the first matching row
func (qb *QueryBuilder) First(ctx context.Context) (map[string]interface{}, error) {
	qb.Limit(1)
	results, err := qb.All(ctx)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, sql.ErrNoRows
	}
	return results[0], nil
}

// Count executes the query and returns the count
func (qb *QueryBuilder) Count(ctx context.Context) (int, error) {
	sqlStr, args, err := qb.ToSQL()
	if err != nil {
		return 0, fmt.Errorf("failed to generate SQL: %w", err)
	}

	// Replace SELECT * with SELECT COUNT(*)
	sqlStr = strings.Replace(sqlStr, "SELECT *", "SELECT COUNT(*)", 1)

	var count int
	err = qb.db.QueryRowContext(ctx, sqlStr, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to execute count query: %w", err)
	}

	return count, nil
}

// Exists checks if any rows match the query
func (qb *QueryBuilder) Exists(ctx context.Context) (bool, error) {
	count, err := qb.Count(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Sum calculates the sum of a column
func (qb *QueryBuilder) Sum(ctx context.Context, field string) (float64, error) {
	return qb.aggregate(ctx, "SUM", field)
}

// Avg calculates the average of a column
func (qb *QueryBuilder) Avg(ctx context.Context, field string) (float64, error) {
	return qb.aggregate(ctx, "AVG", field)
}

// Min finds the minimum value of a column
func (qb *QueryBuilder) Min(ctx context.Context, field string) (float64, error) {
	return qb.aggregate(ctx, "MIN", field)
}

// Max finds the maximum value of a column
func (qb *QueryBuilder) Max(ctx context.Context, field string) (float64, error) {
	return qb.aggregate(ctx, "MAX", field)
}

// aggregate performs an aggregate function
func (qb *QueryBuilder) aggregate(ctx context.Context, fn string, field string) (float64, error) {
	sqlStr, args, err := qb.ToSQL()
	if err != nil {
		return 0, fmt.Errorf("failed to generate SQL: %w", err)
	}

	// Replace SELECT * with SELECT AGG(field)
	sqlStr = strings.Replace(sqlStr, "SELECT *", fmt.Sprintf("SELECT %s(%s)", fn, field), 1)

	var result sql.NullFloat64
	err = qb.db.QueryRowContext(ctx, sqlStr, args...).Scan(&result)
	if err != nil {
		return 0, fmt.Errorf("failed to execute aggregate query: %w", err)
	}

	if !result.Valid {
		return 0, nil
	}

	return result.Float64, nil
}

// Clone creates a copy of the query builder
func (qb *QueryBuilder) Clone() *QueryBuilder {
	clone := &QueryBuilder{
		resource:     qb.resource,
		db:           qb.db,
		schemas:      qb.schemas,
		loader:       qb.loader, // Share the same loader
		conditions:   make([]*Condition, len(qb.conditions)),
		joins:        make([]*Join, len(qb.joins)),
		orderBy:      make([]string, len(qb.orderBy)),
		groupBy:      make([]string, len(qb.groupBy)),
		having:       make([]*Condition, len(qb.having)),
		includes:     make([]string, len(qb.includes)),
		scopeNames:   make([]string, len(qb.scopeNames)),
		paramCounter: 1,
		args:         make([]interface{}, 0),
	}

	copy(clone.conditions, qb.conditions)
	copy(clone.joins, qb.joins)
	copy(clone.orderBy, qb.orderBy)
	copy(clone.groupBy, qb.groupBy)
	copy(clone.having, qb.having)
	copy(clone.includes, qb.includes)
	copy(clone.scopeNames, qb.scopeNames)

	if qb.limit != nil {
		limit := *qb.limit
		clone.limit = &limit
	}

	if qb.offset != nil {
		offset := *qb.offset
		clone.offset = &offset
	}

	return clone
}

// scanRows scans SQL rows into a slice of maps
func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
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
			record[col] = values[i]
		}

		results = append(results, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// toTableName converts a resource name to a table name (snake_case plural)
func toTableName(resourceName string) string {
	snake := toSnakeCase(resourceName)
	return pluralize(snake)
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

// validateIdentifier validates that an identifier only contains safe characters
// (letters, digits, underscore, and dot for qualified names).
// Panics if invalid characters are found.
func validateIdentifier(identifier string) {
	// Only allow alphanumeric, underscore, and dot for qualified names
	for _, char := range identifier {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '.') {
			panic(fmt.Sprintf("invalid identifier: %s (contains invalid character: %c)", identifier, char))
		}
	}
}

// isValidJoinCondition validates that a join condition has proper format
// and contains only safe operators and identifiers
func isValidJoinCondition(condition string) bool {
	// Allow these safe comparison operators
	validOperators := []string{"=", "!=", "<>", "<=", ">=", "<", ">"}

	// Check if at least one valid operator exists
	hasValidOp := false
	for _, op := range validOperators {
		if strings.Contains(condition, " "+op+" ") ||
		   strings.Contains(condition, op) {
			hasValidOp = true
			break
		}
	}

	if !hasValidOp {
		return false
	}

	// Validate all table.column identifiers in the condition
	// Split by AND and OR first to handle compound conditions
	subconditions := []string{condition}

	// Split by AND
	if strings.Contains(condition, " AND ") {
		subconditions = strings.Split(condition, " AND ")
	}

	// Further split by OR
	allSubconditions := []string{}
	for _, sub := range subconditions {
		if strings.Contains(sub, " OR ") {
			allSubconditions = append(allSubconditions, strings.Split(sub, " OR ")...)
		} else {
			allSubconditions = append(allSubconditions, sub)
		}
	}

	// Validate each subcondition
	for _, sub := range allSubconditions {
		sub = strings.TrimSpace(sub)

		// Remove parentheses
		sub = strings.ReplaceAll(sub, "(", "")
		sub = strings.ReplaceAll(sub, ")", "")

		// Extract identifiers by removing operators
		// IMPORTANT: Replace longer operators first to avoid partial replacements
		cleaned := sub
		for _, op := range []string{"!=", "<>", "<=", ">=", "=", "<", ">"} {
			cleaned = strings.ReplaceAll(cleaned, op, " ")
		}

		// Check each token
		tokens := strings.Fields(cleaned)
		for _, token := range tokens {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}

			// Should be in "table.column" format
			if !strings.Contains(token, ".") {
				return false
			}

			parts := strings.Split(token, ".")
			if len(parts) != 2 {
				return false // Must be exactly table.column
			}

			// Validate each part contains only safe characters
			for _, part := range parts {
				if !isValidIdentifier(part) {
					return false
				}
			}
		}
	}

	return true
}

// isValidIdentifier checks if a string is a valid SQL identifier
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}

	for _, char := range s {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return false
		}
	}
	return true
}

// parseScopeCondition parses a scope condition expression into a Condition
func parseScopeCondition(field string, expr string, args map[string]interface{}) (*Condition, error) {
	expr = strings.TrimSpace(expr)

	// Parse operator and value from expression
	// Examples: "= $arg", "> 100", "IN $categories", "LIKE '%golang%'"

	parts := strings.SplitN(expr, " ", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid scope condition format: %s", expr)
	}

	opStr := strings.TrimSpace(parts[0])
	valueStr := strings.TrimSpace(parts[1])

	// Map operator string to Operator type
	op, err := parseOperatorString(opStr)
	if err != nil {
		return nil, err
	}

	// Parse value (could be literal or argument reference)
	var value interface{}
	if strings.HasPrefix(valueStr, "$") {
		// Argument reference
		argName := strings.TrimPrefix(valueStr, "$")
		var ok bool
		value, ok = args[argName]
		if !ok {
			return nil, fmt.Errorf("unknown argument: %s", argName)
		}
	} else {
		// Literal value (string, number, etc.)
		value = parseLiteralValue(valueStr)
	}

	return &Condition{
		Field:    field,
		Operator: op,
		Value:    value,
		Or:       false,
	}, nil
}

// parseOperatorString converts an operator string to an Operator type
func parseOperatorString(opStr string) (Operator, error) {
	opStr = strings.ToUpper(opStr)
	switch opStr {
	case "=", "==":
		return OpEqual, nil
	case "!=", "<>":
		return OpNotEqual, nil
	case "<":
		return OpLessThan, nil
	case "<=":
		return OpLessThanOrEqual, nil
	case ">":
		return OpGreaterThan, nil
	case ">=":
		return OpGreaterThanOrEqual, nil
	case "IN":
		return OpIn, nil
	case "NOT":
		// Handle "NOT IN" case
		return OpNotIn, nil
	case "LIKE":
		return OpLike, nil
	case "ILIKE":
		return OpILike, nil
	case "IS":
		// Handle "IS NULL" and "IS NOT NULL" - need to check next word
		return OpIsNull, nil
	default:
		return OpEqual, fmt.Errorf("unknown operator: %s", opStr)
	}
}

// parseLiteralValue parses a literal value from a string
func parseLiteralValue(valueStr string) interface{} {
	// Remove quotes if present
	valueStr = strings.Trim(valueStr, "'\"")

	// Try parsing as bool first (must be exact match to "true" or "false")
	if valueStr == "true" {
		return true
	}
	if valueStr == "false" {
		return false
	}

	// Try parsing as integer
	if i, err := fmt.Sscanf(valueStr, "%d", new(int)); err == nil && i == 1 {
		var intVal int
		fmt.Sscanf(valueStr, "%d", &intVal)
		return intVal
	}

	// Try parsing as float (only if it contains a decimal point)
	if strings.Contains(valueStr, ".") {
		if f, err := fmt.Sscanf(valueStr, "%f", new(float64)); err == nil && f == 1 {
			var floatVal float64
			fmt.Sscanf(valueStr, "%f", &floatVal)
			return floatVal
		}
	}

	// Return as string (default case)
	return valueStr
}
