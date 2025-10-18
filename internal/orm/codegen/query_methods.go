// Package codegen provides code generation for type-safe query methods
package codegen

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// QueryMethodGenerator generates type-safe query methods for resources
type QueryMethodGenerator struct {
	schemas map[string]*schema.ResourceSchema
}

// NewQueryMethodGenerator creates a new query method generator
func NewQueryMethodGenerator(schemas map[string]*schema.ResourceSchema) *QueryMethodGenerator {
	return &QueryMethodGenerator{
		schemas: schemas,
	}
}

// Generate generates all query methods for a resource
func (g *QueryMethodGenerator) Generate(resource *schema.ResourceSchema) (string, error) {
	var code strings.Builder

	// Generate query builder struct
	code.WriteString(g.generateQueryBuilderStruct(resource))
	code.WriteString("\n\n")

	// Generate constructor
	code.WriteString(g.generateConstructor(resource))
	code.WriteString("\n\n")

	// Generate where methods for each field
	for fieldName, field := range resource.Fields {
		code.WriteString(g.generateWhereMethod(resource, fieldName, field))
		code.WriteString("\n\n")
		code.WriteString(g.generateOrderByMethod(resource, fieldName))
		code.WriteString("\n\n")
	}

	// Generate relationship methods
	for relName, rel := range resource.Relationships {
		code.WriteString(g.generateRelationshipMethod(resource, relName, rel))
		code.WriteString("\n\n")
	}

	// Generate scope methods
	for scopeName, scopeDef := range resource.Scopes {
		code.WriteString(g.generateScopeMethod(resource, scopeName, scopeDef))
		code.WriteString("\n\n")
	}

	// Generate terminal methods
	code.WriteString(g.generateTerminalMethods(resource))

	return code.String(), nil
}

// generateQueryBuilderStruct generates the query builder struct for a resource
func (g *QueryMethodGenerator) generateQueryBuilderStruct(resource *schema.ResourceSchema) string {
	return fmt.Sprintf(`// %sQuery provides type-safe query building for %s
type %sQuery struct {
	builder *query.QueryBuilder
}`, resource.Name, resource.Name, resource.Name)
}

// generateConstructor generates the query builder constructor
func (g *QueryMethodGenerator) generateConstructor(resource *schema.ResourceSchema) string {
	return fmt.Sprintf(`// Query creates a new query builder for %s
func (r *%s) Query(db *sql.DB) *%sQuery {
	return &%sQuery{
		builder: query.NewQueryBuilder(resourceSchema_%s, db, allSchemas),
	}
}`, resource.Name, resource.Name, resource.Name, resource.Name, resource.Name)
}

// generateWhereMethod generates type-safe where methods for a field
func (g *QueryMethodGenerator) generateWhereMethod(resource *schema.ResourceSchema, fieldName string, field *schema.Field) string {
	var code strings.Builder

	// Get Go type for the field
	goType := g.mapTypeToGo(field.Type)

	// Generate Where method
	methodName := toPascalCase(fieldName)
	code.WriteString(fmt.Sprintf(`// Where%s adds a WHERE condition on %s
func (q *%sQuery) Where%s(op query.Operator, value %s) *%sQuery {
	q.builder.Where("%s", op, value)
	return q
}`, methodName, fieldName, resource.Name, methodName, goType, resource.Name, fieldName))

	code.WriteString("\n\n")

	// Generate convenience methods for common operators
	if !field.Type.Nullable {
		// Eq method
		code.WriteString(fmt.Sprintf(`// Where%sEq is a convenience method for equality comparison
func (q *%sQuery) Where%sEq(value %s) *%sQuery {
	return q.Where%s(query.OpEqual, value)
}`, methodName, resource.Name, methodName, goType, resource.Name, methodName))

		code.WriteString("\n\n")
	}

	// For numeric types, generate range methods
	if field.Type.IsNumeric() {
		code.WriteString(fmt.Sprintf(`// Where%sGt adds a greater-than condition
func (q *%sQuery) Where%sGt(value %s) *%sQuery {
	return q.Where%s(query.OpGreaterThan, value)
}

// Where%sLt adds a less-than condition
func (q *%sQuery) Where%sLt(value %s) *%sQuery {
	return q.Where%s(query.OpLessThan, value)
}`,
			methodName, resource.Name, methodName, goType, resource.Name, methodName,
			methodName, resource.Name, methodName, goType, resource.Name, methodName))

		code.WriteString("\n\n")
	}

	// For text types, generate LIKE methods
	if field.Type.IsText() {
		code.WriteString(fmt.Sprintf(`// Where%sLike adds a LIKE condition
func (q *%sQuery) Where%sLike(pattern string) *%sQuery {
	return q.Where%s(query.OpLike, pattern)
}

// Where%sILike adds a case-insensitive LIKE condition
func (q *%sQuery) Where%sILike(pattern string) *%sQuery {
	return q.Where%s(query.OpILike, pattern)
}`,
			methodName, resource.Name, methodName, resource.Name, methodName,
			methodName, resource.Name, methodName, resource.Name, methodName))

		code.WriteString("\n\n")
	}

	// For nullable fields, generate null check methods
	if field.Type.Nullable {
		code.WriteString(fmt.Sprintf(`// Where%sNull adds an IS NULL condition
func (q *%sQuery) Where%sNull() *%sQuery {
	q.builder.WhereNull("%s")
	return q
}

// Where%sNotNull adds an IS NOT NULL condition
func (q *%sQuery) Where%sNotNull() *%sQuery {
	q.builder.WhereNotNull("%s")
	return q
}`,
			methodName, resource.Name, methodName, resource.Name, fieldName,
			methodName, resource.Name, methodName, resource.Name, fieldName))
	}

	return code.String()
}

// generateOrderByMethod generates order by methods for a field
func (g *QueryMethodGenerator) generateOrderByMethod(resource *schema.ResourceSchema, fieldName string) string {
	methodName := toPascalCase(fieldName)
	return fmt.Sprintf(`// OrderBy%s adds an ORDER BY clause on %s
func (q *%sQuery) OrderBy%s(direction string) *%sQuery {
	q.builder.OrderBy("%s", direction)
	return q
}

// OrderBy%sAsc adds ascending ORDER BY on %s
func (q *%sQuery) OrderBy%sAsc() *%sQuery {
	return q.OrderBy%s("ASC")
}

// OrderBy%sDesc adds descending ORDER BY on %s
func (q *%sQuery) OrderBy%sDesc() *%sQuery {
	return q.OrderBy%s("DESC")
}`,
		methodName, fieldName, resource.Name, methodName, resource.Name, fieldName,
		methodName, fieldName, resource.Name, methodName, resource.Name, methodName,
		methodName, fieldName, resource.Name, methodName, resource.Name, methodName)
}

// generateRelationshipMethod generates methods for joining relationships
func (g *QueryMethodGenerator) generateRelationshipMethod(resource *schema.ResourceSchema, relName string, rel *schema.Relationship) string {
	methodName := toPascalCase(relName)

	switch rel.Type {
	case schema.RelationshipBelongsTo:
		return g.generateBelongsToMethod(resource, methodName, relName, rel)
	case schema.RelationshipHasMany:
		return g.generateHasManyMethod(resource, methodName, relName, rel)
	case schema.RelationshipHasOne:
		return g.generateHasOneMethod(resource, methodName, relName, rel)
	default:
		return ""
	}
}

// generateBelongsToMethod generates a method for belongs_to relationships
func (g *QueryMethodGenerator) generateBelongsToMethod(resource *schema.ResourceSchema, methodName, relName string, rel *schema.Relationship) string {
	tableName := toTableName(rel.TargetResource)
	fk := rel.ForeignKey
	if fk == "" {
		fk = toSnakeCase(rel.TargetResource) + "_id"
	}

	return fmt.Sprintf(`// Join%s joins the %s relationship
func (q *%sQuery) Join%s() *%sQuery {
	q.builder.InnerJoin("%s", "%s.id = %s.%s")
	return q
}

// Include%s eager loads the %s relationship
func (q *%sQuery) Include%s() *%sQuery {
	q.builder.Includes("%s")
	return q
}`,
		methodName, relName, resource.Name, methodName, resource.Name,
		tableName, tableName, toTableName(resource.Name), fk,
		methodName, relName, resource.Name, methodName, resource.Name, relName)
}

// generateHasManyMethod generates a method for has_many relationships
func (g *QueryMethodGenerator) generateHasManyMethod(resource *schema.ResourceSchema, methodName, relName string, rel *schema.Relationship) string {
	return fmt.Sprintf(`// Include%s eager loads the %s relationship
func (q *%sQuery) Include%s() *%sQuery {
	q.builder.Includes("%s")
	return q
}`,
		methodName, relName, resource.Name, methodName, resource.Name, relName)
}

// generateHasOneMethod generates a method for has_one relationships
func (g *QueryMethodGenerator) generateHasOneMethod(resource *schema.ResourceSchema, methodName, relName string, rel *schema.Relationship) string {
	return fmt.Sprintf(`// Include%s eager loads the %s relationship
func (q *%sQuery) Include%s() *%sQuery {
	q.builder.Includes("%s")
	return q
}`,
		methodName, relName, resource.Name, methodName, resource.Name, relName)
}

// generateScopeMethod generates a method for applying a scope
func (g *QueryMethodGenerator) generateScopeMethod(resource *schema.ResourceSchema, scopeName string, scopeDef *schema.Scope) string {
	methodName := toPascalCase(scopeName)

	// Build parameter list
	params := make([]string, 0)
	for _, arg := range scopeDef.Arguments {
		goType := g.mapTypeToGo(arg.Type)
		params = append(params, fmt.Sprintf("%s %s", arg.Name, goType))
	}
	paramList := strings.Join(params, ", ")

	// Build argument list for scope call
	args := make([]string, 0)
	for _, arg := range scopeDef.Arguments {
		args = append(args, arg.Name)
	}
	argList := strings.Join(args, ", ")

	return fmt.Sprintf(`// %s applies the %s scope
func (q *%sQuery) %s(%s) (*%sQuery, error) {
	_, err := q.builder.Scope("%s", %s)
	if err != nil {
		return nil, err
	}
	return q, nil
}`,
		methodName, scopeName, resource.Name, methodName, paramList,
		resource.Name, scopeName, argList)
}

// generateTerminalMethods generates terminal methods (All, First, Count, etc.)
func (g *QueryMethodGenerator) generateTerminalMethods(resource *schema.ResourceSchema) string {
	return fmt.Sprintf(`// All executes the query and returns all matching rows
func (q *%sQuery) All(ctx context.Context) ([]*%s, error) {
	rows, err := q.builder.All(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]*%s, 0, len(rows))
	for _, row := range rows {
		result, err := scan%s(row)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// First executes the query and returns the first matching row
func (q *%sQuery) First(ctx context.Context) (*%s, error) {
	row, err := q.builder.First(ctx)
	if err != nil {
		return nil, err
	}
	return scan%s(row)
}

// Count executes the query and returns the count
func (q *%sQuery) Count(ctx context.Context) (int, error) {
	return q.builder.Count(ctx)
}

// Exists checks if any rows match the query
func (q *%sQuery) Exists(ctx context.Context) (bool, error) {
	return q.builder.Exists(ctx)
}

// Limit sets the LIMIT clause
func (q *%sQuery) Limit(n int) *%sQuery {
	q.builder.Limit(n)
	return q
}

// Offset sets the OFFSET clause
func (q *%sQuery) Offset(n int) *%sQuery {
	q.builder.Offset(n)
	return q
}`,
		resource.Name, resource.Name,
		resource.Name, resource.Name,
		resource.Name, resource.Name, resource.Name,
		resource.Name,
		resource.Name,
		resource.Name, resource.Name,
		resource.Name, resource.Name)
}

// mapTypeToGo maps a TypeSpec to a Go type
func (g *QueryMethodGenerator) mapTypeToGo(typeSpec *schema.TypeSpec) string {
	var baseType string

	switch typeSpec.BaseType {
	case schema.TypeString, schema.TypeText, schema.TypeMarkdown, schema.TypeEmail, schema.TypeURL, schema.TypePhone:
		baseType = "string"
	case schema.TypeInt:
		baseType = "int"
	case schema.TypeBigInt:
		baseType = "int64"
	case schema.TypeFloat:
		baseType = "float64"
	case schema.TypeDecimal:
		baseType = "float64" // Simplified
	case schema.TypeBool:
		baseType = "bool"
	case schema.TypeTimestamp, schema.TypeDate, schema.TypeTime:
		baseType = "time.Time"
	case schema.TypeUUID:
		baseType = "uuid.UUID"
	case schema.TypeULID:
		baseType = "string" // ULID as string
	case schema.TypeJSON, schema.TypeJSONB:
		baseType = "interface{}"
	case schema.TypeEnum:
		baseType = "string"
	default:
		baseType = "interface{}"
	}

	if typeSpec.Nullable {
		return "*" + baseType
	}
	return baseType
}

// toPascalCase converts a string to PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[0:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
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

