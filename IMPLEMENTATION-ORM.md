# IMPLEMENTATION-ORM.md

**Component:** Resource System & ORM
**Status:** Implementation Ready
**Last Updated:** 2025-10-13
**Estimated Effort:** 31-35 weeks (155-175 person-days)

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Component 1: Schema Definition & Validation](#component-1-schema-definition--validation)
4. [Component 2: Database Schema Generation](#component-2-database-schema-generation)
5. [Component 3: Migration System](#component-3-migration-system)
6. [Component 4: Query Builder & Scopes](#component-4-query-builder--scopes)
7. [Component 5: Relationship Loading](#component-5-relationship-loading)
8. [Component 6: CRUD Operations](#component-6-crud-operations)
9. [Component 7: Validation Engine](#component-7-validation-engine)
10. [Component 8: Lifecycle Hooks](#component-8-lifecycle-hooks)
11. [Component 9: Transaction Management](#component-9-transaction-management)
12. [Component 10: Change Tracking](#component-10-change-tracking)
13. [Development Phases](#development-phases)
14. [Testing Strategy](#testing-strategy)
15. [Integration Points](#integration-points)
16. [Performance Targets](#performance-targets)
17. [Risk Mitigation](#risk-mitigation)
18. [Success Criteria](#success-criteria)

---

## Overview

### Purpose

The Resource System & ORM is the **foundational data layer** of Conduit, providing:

1. **Declarative data modeling** with zero ambiguity
2. **Automatic CRUD generation** with full type safety
3. **Explicit relationship management** with clear ownership
4. **Lifecycle hooks** with transaction boundaries
5. **Multi-layer validation** from compile-time to runtime
6. **Safe schema evolution** with migration support

### Design Philosophy

**Explicit over Implicit**
- All relationships explicitly declared with foreign keys
- All nullability explicit (`!` vs `?`)
- All transaction boundaries marked (`@transaction`, `@async`)
- No "magic" methods or convention-based behavior

**Progressive Disclosure**
- Minimal resource: 3 lines (id + 1 field + basic info)
- Medium complexity: Add validations, hooks, scopes
- Advanced: Full transaction control, custom functions, invariants

**LLM-First Design**
- Zero ambiguity in syntax and semantics
- 95%+ first-time compile success for LLM-generated code
- All behavior visible in single resource file
- Type errors caught at compile time

### Key Innovations

1. **Type Safety**: Explicit nullability on every field prevents nil errors
2. **N+1 Prevention**: Built-in eager loading with compile-time detection
3. **Explicit Transactions**: `@transaction` and `@async` annotations make boundaries clear
4. **Multi-Layer Validation**: Field-level → constraints → procedural → invariants
5. **Change Tracking**: Built-in `field_changed?` methods for update hooks
6. **Pattern Enforcement**: Language-level patterns for data modeling

### Compilation Target

**Primary:** PostgreSQL (JSONB, arrays, full-text search)
**Secondary:** MySQL/MariaDB (v1.1)
**Development:** SQLite (testing)

**Generated Code:** Go structs + CRUD methods + query builders

---

## Architecture

### High-Level Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    Resource Definition (.cdt)                    │
│                                                                   │
│  resource Post {                                                  │
│    title: string! @min(5)                                        │
│    author: User! { foreign_key: "author_id" }                   │
│    @before create @transaction { ... }                          │
│  }                                                               │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Parser → AST                                │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Schema Analyzer                               │
│  - Type resolution                                               │
│  - Relationship graph construction                               │
│  - Constraint validation                                         │
│  - Circular dependency detection                                 │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                ┌───────────┴───────────┐
                ▼                       ▼
┌───────────────────────────┐  ┌────────────────────────────┐
│  Schema Generator         │  │   Migration System         │
│                           │  │                            │
│  Generates:               │  │  Generates:                │
│  - SQL DDL                │  │  - Schema diffs            │
│  - Indexes                │  │  - Up/Down migrations      │
│  - Foreign keys           │  │  - Safety checks           │
└───────────┬───────────────┘  └────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Code Generator                                │
│                                                                   │
│  Generates Go code:                                              │
│  - Structs (type Post struct { ... })                           │
│  - CRUD methods (Create, Update, Delete, Find)                  │
│  - Query builders (Post.Where(...).OrderBy(...))                │
│  - Relationship loaders (post.Author(), post.Comments())        │
│  - Hook executors                                                │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Runtime ORM Layer                          │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ Query Builder│  │  Relationship│  │   Lifecycle  │         │
│  │              │  │   Loader     │  │   Hooks      │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │  Validation  │  │  Transaction │  │    Change    │         │
│  │   Engine     │  │   Manager    │  │   Tracking   │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Database (PostgreSQL)                         │
└─────────────────────────────────────────────────────────────────┘
```

### Layered Architecture

**Layer 1: Schema Definition**
- Parse resource syntax
- Build type system
- Validate relationships
- Detect circular dependencies

**Layer 2: Schema Generation**
- Map types to SQL
- Generate DDL
- Create indexes
- Define constraints

**Layer 3: Runtime Operations**
- Query building
- Relationship loading
- Hook execution
- Validation

**Layer 4: Database Interaction**
- Connection pooling
- Transaction management
- Query execution
- Result mapping

---

## Component 1: Schema Definition & Validation

### Responsibility

Parse resource definitions and validate semantic correctness.

### Key Data Structures

#### Type System

```go
// Type specification with explicit nullability
type TypeSpec struct {
    BaseType    PrimitiveType      // string, int, float, bool, etc.
    Nullable    bool               // ! = false, ? = true
    Constraints []Constraint       // @min, @max, @pattern, @unique
    Default     *Value             // @default value

    // Complex types
    ArrayElement  *TypeSpec                  // For array<T>
    HashKey       *TypeSpec                  // For hash<K,V>
    HashValue     *TypeSpec
    StructFields  map[string]*TypeSpec       // For inline structs
}

// Primitive types
type PrimitiveType int
const (
    TypeString PrimitiveType = iota
    TypeText
    TypeMarkdown
    TypeInt
    TypeFloat
    TypeDecimal
    TypeBool
    TypeTimestamp
    TypeDate
    TypeTime
    TypeUUID
    TypeULID
    TypeEmail
    TypeURL
    TypePhone
    TypeJSON
    TypeEnum
)
```

#### Field Definition

```go
type Field struct {
    Name        string
    Type        *TypeSpec
    Constraints []Constraint
    Annotations []Annotation      // @primary, @auto, @unique, @index

    // For nested structs
    IsNested     bool
    NestedFields map[string]*Field

    // Source location for errors
    Location     SourceLocation
}

type Annotation struct {
    Name   string
    Args   []interface{}
}

type Constraint struct {
    Type         ConstraintType
    Value        interface{}
    ErrorMessage string
}

type ConstraintType int
const (
    ConstraintMin ConstraintType = iota
    ConstraintMax
    ConstraintPattern
    ConstraintUnique
    ConstraintIndex
    ConstraintDefault
)
```

#### Relationship Definition

```go
type Relationship struct {
    Type           RelationType    // BelongsTo, HasMany, HasManyThrough
    TargetResource string
    FieldName      string          // Local field name
    Nullable       bool            // Can be nil?

    // Foreign key configuration
    ForeignKey     string          // Explicit foreign key column
    OnDelete       CascadeAction   // restrict, cascade, set_null, no_action
    OnUpdate       CascadeAction

    // For has_many
    OrderBy        string

    // For has_many_through
    ThroughResource string
    JoinTable       string
    AssociationKey  string

    Location       SourceLocation
}

type RelationType int
const (
    BelongsTo RelationType = iota
    HasMany
    HasManyThrough
    HasOne
)

type CascadeAction int
const (
    CascadeRestrict CascadeAction = iota
    CascadeCascade
    CascadeSetNull
    CascadeNoAction
)
```

#### Resource Schema

```go
type ResourceSchema struct {
    Name          string
    Documentation string
    FilePath      string

    Fields        map[string]*Field
    Relationships map[string]*Relationship

    // Lifecycle hooks
    Hooks         map[HookType][]*Hook

    // Validations
    Validators    []*Validator
    Constraints   []*Constraint
    Invariants    []*Invariant

    // Query scopes
    Scopes        map[string]*Scope

    // Computed fields
    Computed      map[string]*ComputedField

    // Custom functions
    Functions     map[string]*Function

    // Middleware
    Middleware    map[string][]string  // operation -> middleware list

    // Metadata
    Nested        *NestedConfig
    TableName     string               // Generated table name
}

type HookType int
const (
    BeforeCreate HookType = iota
    BeforeUpdate
    BeforeDelete
    BeforeSave
    AfterCreate
    AfterUpdate
    AfterDelete
    AfterSave
)

type Hook struct {
    Type        HookType
    Transaction bool          // @transaction annotation
    Async       bool          // @async block present
    Code        *AST          // Parsed hook code
    Location    SourceLocation
}
```

### Validation Rules

#### 1. Nullability Enforcement

```go
func (v *SchemaValidator) ValidateNullability(schema *ResourceSchema) error {
    for name, field := range schema.Fields {
        // Every field MUST have explicit nullability
        if field.Type.Nullable == nil {
            return fmt.Errorf(
                "field %s.%s missing nullability marker (! or ?)",
                schema.Name, name,
            )
        }

        // Default values must respect nullability
        if field.Type.Default != nil && field.Type.Nullable {
            // Optional field with default is OK, but warn
            v.warnings = append(v.warnings, Warning{
                Message: fmt.Sprintf(
                    "field %s.%s is optional but has default value",
                    schema.Name, name,
                ),
            })
        }
    }

    // Relationships must specify nullability
    for name, rel := range schema.Relationships {
        if rel.Type == BelongsTo && rel.Nullable == nil {
            return fmt.Errorf(
                "relationship %s.%s missing nullability",
                schema.Name, name,
            )
        }
    }

    return nil
}
```

#### 2. Relationship Validation

```go
func (v *SchemaValidator) ValidateRelationships(
    schema *ResourceSchema,
    registry map[string]*ResourceSchema,
) error {
    for name, rel := range schema.Relationships {
        // Target resource must exist
        target, exists := registry[rel.TargetResource]
        if !exists {
            return fmt.Errorf(
                "relationship %s.%s references unknown resource %s",
                schema.Name, name, rel.TargetResource,
            )
        }

        // Validate cascade actions
        if err := v.validateCascade(rel, target); err != nil {
            return err
        }

        // For has_many_through, validate join table
        if rel.Type == HasManyThrough {
            if rel.JoinTable == "" {
                return fmt.Errorf(
                    "relationship %s.%s missing join_table",
                    schema.Name, name,
                )
            }
        }
    }

    return nil
}

func (v *SchemaValidator) validateCascade(
    rel *Relationship,
    target *ResourceSchema,
) error {
    // set_null requires nullable relationship
    if rel.OnDelete == CascadeSetNull && !rel.Nullable {
        return fmt.Errorf(
            "relationship %s cannot use on_delete: set_null (not nullable)",
            rel.FieldName,
        )
    }

    return nil
}
```

#### 3. Circular Dependency Detection

```go
type RelationshipGraph struct {
    nodes map[string]*ResourceSchema
    edges map[string][]string  // resource -> dependencies
}

func BuildRelationshipGraph(schemas map[string]*ResourceSchema) *RelationshipGraph {
    graph := &RelationshipGraph{
        nodes: schemas,
        edges: make(map[string][]string),
    }

    for name, schema := range schemas {
        for _, rel := range schema.Relationships {
            if rel.Type == BelongsTo {
                // This resource depends on target resource
                graph.edges[name] = append(graph.edges[name], rel.TargetResource)
            }
        }
    }

    return graph
}

func (g *RelationshipGraph) DetectCycles() [][]string {
    var cycles [][]string
    visited := make(map[string]bool)
    recursionStack := make(map[string]bool)

    var dfs func(node string, path []string) bool
    dfs = func(node string, path []string) bool {
        visited[node] = true
        recursionStack[node] = true
        path = append(path, node)

        for _, neighbor := range g.edges[node] {
            if !visited[neighbor] {
                if dfs(neighbor, path) {
                    return true
                }
            } else if recursionStack[neighbor] {
                // Found cycle
                cycleStart := -1
                for i, n := range path {
                    if n == neighbor {
                        cycleStart = i
                        break
                    }
                }
                if cycleStart >= 0 {
                    cycles = append(cycles, path[cycleStart:])
                }
                return true
            }
        }

        recursionStack[node] = false
        return false
    }

    for node := range g.nodes {
        if !visited[node] {
            dfs(node, []string{})
        }
    }

    return cycles
}

func (g *RelationshipGraph) TopologicalSort() ([]string, error) {
    // Kahn's algorithm for topological sort
    inDegree := make(map[string]int)
    for node := range g.nodes {
        inDegree[node] = 0
    }

    for _, neighbors := range g.edges {
        for _, neighbor := range neighbors {
            inDegree[neighbor]++
        }
    }

    queue := []string{}
    for node, degree := range inDegree {
        if degree == 0 {
            queue = append(queue, node)
        }
    }

    result := []string{}
    for len(queue) > 0 {
        node := queue[0]
        queue = queue[1:]
        result = append(result, node)

        for _, neighbor := range g.edges[node] {
            inDegree[neighbor]--
            if inDegree[neighbor] == 0 {
                queue = append(queue, neighbor)
            }
        }
    }

    // Check if all nodes processed (no cycles)
    if len(result) != len(g.nodes) {
        return nil, fmt.Errorf("circular dependency detected")
    }

    return result, nil
}
```

#### 4. Type Compatibility

```go
func (v *SchemaValidator) ValidateTypeCompatibility(schema *ResourceSchema) error {
    for name, field := range schema.Fields {
        // Default value must match field type
        if field.Type.Default != nil {
            if err := v.checkTypeMatch(field.Type, field.Type.Default); err != nil {
                return fmt.Errorf(
                    "field %s.%s default value type mismatch: %w",
                    schema.Name, name, err,
                )
            }
        }

        // Constraint values must match field type
        for _, constraint := range field.Constraints {
            if err := v.checkConstraintType(field.Type, constraint); err != nil {
                return fmt.Errorf(
                    "field %s.%s constraint type mismatch: %w",
                    schema.Name, name, err,
                )
            }
        }
    }

    // Computed fields return type must match declared type
    for name, computed := range schema.Computed {
        // This will be validated during code generation
        _ = name
        _ = computed
    }

    return nil
}
```

### Implementation Details

#### Schema Registry

```go
type SchemaRegistry struct {
    schemas    map[string]*ResourceSchema
    validator  *SchemaValidator
    mu         sync.RWMutex
}

func NewSchemaRegistry() *SchemaRegistry {
    return &SchemaRegistry{
        schemas:   make(map[string]*ResourceSchema),
        validator: NewSchemaValidator(),
    }
}

func (r *SchemaRegistry) Register(schema *ResourceSchema) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    // Validate schema
    if err := r.validator.Validate(schema, r.schemas); err != nil {
        return err
    }

    // Store schema
    r.schemas[schema.Name] = schema

    return nil
}

func (r *SchemaRegistry) Get(name string) (*ResourceSchema, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    schema, exists := r.schemas[name]
    return schema, exists
}

func (r *SchemaRegistry) All() map[string]*ResourceSchema {
    r.mu.RLock()
    defer r.mu.RUnlock()

    // Return copy to prevent external modification
    result := make(map[string]*ResourceSchema, len(r.schemas))
    for k, v := range r.schemas {
        result[k] = v
    }
    return result
}

func (r *SchemaRegistry) ValidateAll() error {
    r.mu.RLock()
    defer r.mu.RUnlock()

    // Build relationship graph
    graph := BuildRelationshipGraph(r.schemas)

    // Check for cycles
    cycles := graph.DetectCycles()
    if len(cycles) > 0 {
        return fmt.Errorf("circular dependencies detected: %v", cycles)
    }

    return nil
}
```

### Error Messages

```go
type ValidationError struct {
    Resource string
    Field    string
    Message  string
    Location SourceLocation
    Hint     string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf(
        "%s:%d:%d: %s.%s: %s\n%s",
        e.Location.File,
        e.Location.Line,
        e.Location.Column,
        e.Resource,
        e.Field,
        e.Message,
        e.Hint,
    )
}

// Example error:
// post.cdt:5:12: Post.title: missing nullability marker (! or ?)
// Add ! for required or ? for optional: title: string!
```

### Testing Strategy

**Unit Tests:**
- Test each validation rule independently
- Test type compatibility checking
- Test circular dependency detection
- Test error message formatting

**Integration Tests:**
- Parse complete resource definitions
- Validate complex relationships
- Test edge cases (self-referential, polymorphic)

**Coverage Target:** >95%

### Estimated Effort

**Time:** 3 weeks
**Team:** 2 engineers
**Complexity:** Medium
**Risk:** Low

---

## Component 2: Database Schema Generation

### Responsibility

Convert resource definitions to SQL DDL (Data Definition Language).

### Type Mapping

#### PostgreSQL Type Mapping

```go
var PostgreSQLTypeMapping = map[PrimitiveType]string{
    TypeString:    "VARCHAR(%d)",           // With max length
    TypeText:      "TEXT",
    TypeMarkdown:  "TEXT",
    TypeInt:       "INTEGER",
    TypeFloat:     "DOUBLE PRECISION",
    TypeDecimal:   "DECIMAL(%d,%d)",        // With precision
    TypeBool:      "BOOLEAN",
    TypeTimestamp: "TIMESTAMPTZ",
    TypeDate:      "DATE",
    TypeTime:      "TIME",
    TypeUUID:      "UUID",
    TypeULID:      "VARCHAR(26)",
    TypeEmail:     "VARCHAR(255)",
    TypeURL:       "TEXT",
    TypePhone:     "VARCHAR(20)",
    TypeJSON:      "JSONB",
    TypeEnum:      "VARCHAR(50)",           // Or custom enum type
}

// Complex types use JSONB
// array<T>       -> JSONB or T[] for primitives
// hash<K,V>      -> JSONB
// inline struct  -> JSONB
```

### Schema Generator

```go
type SchemaGenerator struct {
    dialect   SQLDialect
    schemas   map[string]*ResourceSchema
}

func NewSchemaGenerator(dialect SQLDialect) *SchemaGenerator {
    return &SchemaGenerator{
        dialect: dialect,
        schemas: make(map[string]*ResourceSchema),
    }
}

func (g *SchemaGenerator) GenerateCreateTable(schema *ResourceSchema) string {
    var sql strings.Builder

    tableName := g.toTableName(schema.Name)
    sql.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", tableName))

    // Primary key (always UUID)
    sql.WriteString("  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),\n")

    // Fields
    for name, field := range g.orderedFields(schema) {
        sql.WriteString(g.generateColumn(name, field))
    }

    // Foreign keys (from BelongsTo relationships)
    for _, rel := range schema.Relationships {
        if rel.Type == BelongsTo {
            sql.WriteString(g.generateForeignKey(rel))
        }
    }

    // Timestamps (automatic)
    sql.WriteString("  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),\n")
    sql.WriteString("  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()\n")

    sql.WriteString(");\n\n")

    // Indexes
    sql.WriteString(g.generateIndexes(schema))

    // Triggers (for updated_at)
    sql.WriteString(g.generateTriggers(schema))

    return sql.String()
}

func (g *SchemaGenerator) generateColumn(name string, field *Field) string {
    sqlType := g.mapType(field.Type)

    var parts []string
    parts = append(parts, fmt.Sprintf("  %s %s", name, sqlType))

    // Nullability
    if !field.Type.Nullable {
        parts = append(parts, "NOT NULL")
    }

    // Default value
    if field.Type.Default != nil {
        defaultSQL := g.formatDefault(field.Type.Default, field.Type)
        parts = append(parts, fmt.Sprintf("DEFAULT %s", defaultSQL))
    }

    // Unique constraint
    if g.hasAnnotation(field, "unique") {
        parts = append(parts, "UNIQUE")
    }

    // Check constraints (from @min, @max)
    if checkSQL := g.generateCheckConstraints(name, field); checkSQL != "" {
        parts = append(parts, checkSQL)
    }

    return strings.Join(parts, " ") + ",\n"
}

func (g *SchemaGenerator) generateForeignKey(rel *Relationship) string {
    fk := rel.ForeignKey
    if fk == "" {
        fk = g.toSnakeCase(rel.TargetResource) + "_id"
    }

    var sql strings.Builder
    sql.WriteString(fmt.Sprintf("  %s UUID", fk))

    // Nullability
    if !rel.Nullable {
        sql.WriteString(" NOT NULL")
    }

    tableName := g.toTableName(rel.TargetResource)
    sql.WriteString(fmt.Sprintf(" REFERENCES %s(id)", tableName))

    // Cascade actions
    sql.WriteString(fmt.Sprintf(" ON DELETE %s", g.toCascadeSQL(rel.OnDelete)))
    sql.WriteString(fmt.Sprintf(" ON UPDATE %s", g.toCascadeSQL(rel.OnUpdate)))

    sql.WriteString(",\n")
    return sql.String()
}

func (g *SchemaGenerator) generateIndexes(schema *ResourceSchema) string {
    var sql strings.Builder
    tableName := g.toTableName(schema.Name)

    // Unique indexes
    for name, field := range schema.Fields {
        if g.hasAnnotation(field, "unique") {
            sql.WriteString(fmt.Sprintf(
                "CREATE UNIQUE INDEX idx_%s_%s ON %s(%s);\n",
                tableName, name, tableName, name,
            ))
        }

        if g.hasAnnotation(field, "index") {
            sql.WriteString(fmt.Sprintf(
                "CREATE INDEX idx_%s_%s ON %s(%s);\n",
                tableName, name, tableName, name,
            ))
        }
    }

    // Foreign key indexes (always created for performance)
    for _, rel := range schema.Relationships {
        if rel.Type == BelongsTo {
            fk := rel.ForeignKey
            if fk == "" {
                fk = g.toSnakeCase(rel.TargetResource) + "_id"
            }
            sql.WriteString(fmt.Sprintf(
                "CREATE INDEX idx_%s_%s ON %s(%s);\n",
                tableName, fk, tableName, fk,
            ))
        }
    }

    sql.WriteString("\n")
    return sql.String()
}

func (g *SchemaGenerator) generateTriggers(schema *ResourceSchema) string {
    tableName := g.toTableName(schema.Name)

    // Trigger for updated_at
    return fmt.Sprintf(`
CREATE OR REPLACE FUNCTION update_%s_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_%s_updated_at
  BEFORE UPDATE ON %s
  FOR EACH ROW
  EXECUTE FUNCTION update_%s_updated_at();

`, tableName, tableName, tableName, tableName)
}

func (g *SchemaGenerator) mapType(typeSpec *TypeSpec) string {
    switch {
    case typeSpec.ArrayElement != nil:
        // array<T> -> JSONB or T[] for primitives
        if g.isPrimitive(typeSpec.ArrayElement) {
            elemType := g.mapPrimitive(typeSpec.ArrayElement.BaseType)
            return elemType + "[]"
        }
        return "JSONB"

    case typeSpec.HashKey != nil:
        // hash<K,V> -> JSONB
        return "JSONB"

    case len(typeSpec.StructFields) > 0:
        // inline struct -> JSONB
        return "JSONB"

    default:
        return g.mapPrimitive(typeSpec.BaseType)
    }
}

func (g *SchemaGenerator) mapPrimitive(t PrimitiveType) string {
    template := PostgreSQLTypeMapping[t]

    // Handle parameterized types
    switch t {
    case TypeString:
        return fmt.Sprintf(template, 255)  // Default max length
    case TypeDecimal:
        return fmt.Sprintf(template, 10, 2)  // Default precision
    default:
        return template
    }
}

func (g *SchemaGenerator) generateCheckConstraints(fieldName string, field *Field) string {
    var constraints []string

    for _, c := range field.Constraints {
        switch c.Type {
        case ConstraintMin:
            if field.Type.BaseType == TypeInt || field.Type.BaseType == TypeFloat {
                constraints = append(constraints,
                    fmt.Sprintf("CHECK (%s >= %v)", fieldName, c.Value),
                )
            } else if field.Type.BaseType == TypeString {
                constraints = append(constraints,
                    fmt.Sprintf("CHECK (length(%s) >= %v)", fieldName, c.Value),
                )
            }

        case ConstraintMax:
            if field.Type.BaseType == TypeInt || field.Type.BaseType == TypeFloat {
                constraints = append(constraints,
                    fmt.Sprintf("CHECK (%s <= %v)", fieldName, c.Value),
                )
            } else if field.Type.BaseType == TypeString {
                constraints = append(constraints,
                    fmt.Sprintf("CHECK (length(%s) <= %v)", fieldName, c.Value),
                )
            }

        case ConstraintPattern:
            constraints = append(constraints,
                fmt.Sprintf("CHECK (%s ~ '%s')", fieldName, c.Value),
            )
        }
    }

    if len(constraints) > 0 {
        return strings.Join(constraints, " ")
    }

    return ""
}

func (g *SchemaGenerator) toCascadeSQL(action CascadeAction) string {
    switch action {
    case CascadeRestrict:
        return "RESTRICT"
    case CascadeCascade:
        return "CASCADE"
    case CascadeSetNull:
        return "SET NULL"
    case CascadeNoAction:
        return "NO ACTION"
    default:
        return "RESTRICT"  // Safe default
    }
}

func (g *SchemaGenerator) toTableName(resourceName string) string {
    // Convert PascalCase to snake_case plural
    // Post -> posts
    // BlogPost -> blog_posts
    snake := g.toSnakeCase(resourceName)
    return g.pluralize(snake)
}

func (g *SchemaGenerator) toSnakeCase(s string) string {
    var result []rune
    for i, r := range s {
        if i > 0 && unicode.IsUpper(r) {
            result = append(result, '_')
        }
        result = append(result, unicode.ToLower(r))
    }
    return string(result)
}

func (g *SchemaGenerator) pluralize(s string) string {
    // Simple pluralization rules
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
```

### Complete Example

#### Input Resource

```
resource Product {
  id: uuid! @primary @auto
  sku: string! @unique @pattern(/^[A-Z0-9-]+$/)
  name: string! @min(3) @max(200)
  slug: string! @unique @index
  description: text!

  category: Category! {
    foreign_key: "category_id"
    on_delete: restrict
  }

  pricing: {
    regular: decimal(10,2)! @min(0.01)
    sale: decimal(10,2)? @min(0.01)
    currency: enum ["USD", "EUR", "GBP"]! @default("USD")
  }!

  inventory: {
    quantity: int! @default(0) @min(0)
    reserved: int! @default(0) @min(0)
  }!

  status: enum ["draft", "active", "discontinued"]! @default("draft")
  featured: bool! @default(false)
}
```

#### Generated SQL

```sql
CREATE TABLE products (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sku VARCHAR(255) NOT NULL UNIQUE CHECK (sku ~ '^[A-Z0-9-]+$'),
  name VARCHAR(200) NOT NULL CHECK (length(name) >= 3 AND length(name) <= 200),
  slug VARCHAR(255) NOT NULL UNIQUE,
  description TEXT NOT NULL,
  category_id UUID NOT NULL REFERENCES categories(id) ON DELETE RESTRICT ON UPDATE CASCADE,
  pricing JSONB NOT NULL,
  inventory JSONB NOT NULL DEFAULT '{"quantity": 0, "reserved": 0}',
  status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'discontinued')),
  featured BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_products_sku ON products(sku);
CREATE UNIQUE INDEX idx_products_slug ON products(slug);
CREATE INDEX idx_products_slug ON products(slug);
CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_status ON products(status) WHERE status = 'active';

CREATE OR REPLACE FUNCTION update_products_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_products_updated_at
  BEFORE UPDATE ON products
  FOR EACH ROW
  EXECUTE FUNCTION update_products_updated_at();
```

### Testing Strategy

**Unit Tests:**
- Test type mapping for all primitive types
- Test complex type mapping (arrays, hashes, structs)
- Test constraint generation
- Test foreign key generation
- Test index generation

**Integration Tests:**
- Generate schema for example resources
- Apply to test database
- Verify schema with introspection queries
- Test on all supported databases (PostgreSQL, MySQL, SQLite)

**Coverage Target:** >90%

### Estimated Effort

**Time:** 2 weeks
**Team:** 1 engineer
**Complexity:** Medium
**Risk:** Low

---

## Component 3: Migration System

### Responsibility

Safe, versioned schema evolution with rollback support and data loss prevention.

### Key Data Structures

```go
type Migration struct {
    Version   int64                  // Unix timestamp
    Name      string                 // Human-readable name
    Up        string                 // SQL to apply
    Down      string                 // SQL to rollback
    Applied   bool
    AppliedAt time.Time
    Breaking  bool                   // Requires manual review
    DataLoss  bool                   // May cause data loss
}

type MigrationSystem struct {
    db          *sql.DB
    migrations  []*Migration
    generator   *MigrationGenerator
    validator   *MigrationValidator
}

type MigrationGenerator struct {
    differ      *SchemaDiffer
    sqlGen      *SchemaGenerator
}

type SchemaDiffer struct {
    oldSchema map[string]*ResourceSchema
    newSchema map[string]*ResourceSchema
}

type SchemaChange struct {
    Type      ChangeType
    Resource  string
    Field     string
    OldValue  interface{}
    NewValue  interface{}
    Breaking  bool
    DataLoss  bool
}

type ChangeType int
const (
    ChangeAddResource ChangeType = iota
    ChangeDropResource
    ChangeAddField
    ChangeDropField
    ChangeModifyField
    ChangeAddRelationship
    ChangeDropRelationship
    ChangeModifyRelationship
    ChangeAddIndex
    ChangeDropIndex
)
```

### Schema Differ

```go
func (d *SchemaDiffer) ComputeDiff() []SchemaChange {
    var changes []SchemaChange

    // 1. Detect added/removed resources
    oldResources := getResourceNames(d.oldSchema)
    newResources := getResourceNames(d.newSchema)

    for _, name := range setDifference(newResources, oldResources) {
        changes = append(changes, SchemaChange{
            Type:     ChangeAddResource,
            Resource: name,
            Breaking: false,
            DataLoss: false,
        })
    }

    for _, name := range setDifference(oldResources, newResources) {
        changes = append(changes, SchemaChange{
            Type:     ChangeDropResource,
            Resource: name,
            Breaking: true,
            DataLoss: true,
        })
    }

    // 2. For each resource, detect field changes
    for name, newResource := range d.newSchema {
        oldResource, exists := d.oldSchema[name]
        if !exists {
            continue  // Already handled as ChangeAddResource
        }

        changes = append(changes, d.diffFields(name, oldResource, newResource)...)
        changes = append(changes, d.diffRelationships(name, oldResource, newResource)...)
    }

    return changes
}

func (d *SchemaDiffer) diffFields(
    resourceName string,
    old, new *ResourceSchema,
) []SchemaChange {
    var changes []SchemaChange

    oldFields := getFieldNames(old)
    newFields := getFieldNames(new)

    // Added fields
    for _, fieldName := range setDifference(newFields, oldFields) {
        newField := new.Fields[fieldName]
        breaking := !newField.Type.Nullable && newField.Type.Default == nil

        changes = append(changes, SchemaChange{
            Type:     ChangeAddField,
            Resource: resourceName,
            Field:    fieldName,
            NewValue: newField,
            Breaking: breaking,
            DataLoss: false,
        })
    }

    // Removed fields
    for _, fieldName := range setDifference(oldFields, newFields) {
        changes = append(changes, SchemaChange{
            Type:     ChangeDropField,
            Resource: resourceName,
            Field:    fieldName,
            OldValue: old.Fields[fieldName],
            Breaking: true,
            DataLoss: true,
        })
    }

    // Modified fields
    for _, fieldName := range setIntersection(oldFields, newFields) {
        oldField := old.Fields[fieldName]
        newField := new.Fields[fieldName]

        if !d.fieldsEqual(oldField, newField) {
            breaking := d.isBreakingFieldChange(oldField, newField)
            dataLoss := d.causesDataLoss(oldField, newField)

            changes = append(changes, SchemaChange{
                Type:     ChangeModifyField,
                Resource: resourceName,
                Field:    fieldName,
                OldValue: oldField,
                NewValue: newField,
                Breaking: breaking,
                DataLoss: dataLoss,
            })
        }
    }

    return changes
}

func (d *SchemaDiffer) isBreakingFieldChange(old, new *Field) bool {
    // Nullability change: optional -> required
    if old.Type.Nullable && !new.Type.Nullable {
        return true
    }

    // Type change
    if old.Type.BaseType != new.Type.BaseType {
        return true
    }

    // Added stricter constraint
    if d.hasStricterConstraints(old, new) {
        return true
    }

    return false
}

func (d *SchemaDiffer) causesDataLoss(old, new *Field) bool {
    // Type change may cause data loss
    if old.Type.BaseType != new.Type.BaseType {
        return true
    }

    // String length reduction
    if old.Type.BaseType == TypeString {
        oldMax := d.getMaxConstraint(old)
        newMax := d.getMaxConstraint(new)
        if newMax > 0 && (oldMax == 0 || newMax < oldMax) {
            return true
        }
    }

    return false
}
```

### Migration Generator

```go
func (g *MigrationGenerator) GenerateMigration(
    old, new map[string]*ResourceSchema,
) (*Migration, error) {
    differ := NewSchemaDiffer(old, new)
    changes := differ.ComputeDiff()

    if len(changes) == 0 {
        return nil, nil  // No changes
    }

    migration := &Migration{
        Version: time.Now().Unix(),
        Name:    g.generateMigrationName(changes),
    }

    // Check if migration has breaking changes
    migration.Breaking = g.hasBreakingChanges(changes)
    migration.DataLoss = g.hasDataLoss(changes)

    // Generate SQL
    migration.Up = g.generateUpSQL(changes)
    migration.Down = g.generateDownSQL(changes)

    return migration, nil
}

func (g *MigrationGenerator) generateUpSQL(changes []SchemaChange) string {
    var sql strings.Builder

    for _, change := range changes {
        switch change.Type {
        case ChangeAddResource:
            schema := g.newSchema[change.Resource]
            sql.WriteString(g.sqlGen.GenerateCreateTable(schema))

        case ChangeDropResource:
            tableName := g.sqlGen.toTableName(change.Resource)
            sql.WriteString(fmt.Sprintf("DROP TABLE %s CASCADE;\n\n", tableName))

        case ChangeAddField:
            sql.WriteString(g.generateAddColumn(change))

        case ChangeDropField:
            sql.WriteString(g.generateDropColumn(change))

        case ChangeModifyField:
            sql.WriteString(g.generateModifyColumn(change))

        case ChangeAddRelationship:
            sql.WriteString(g.generateAddForeignKey(change))

        case ChangeDropRelationship:
            sql.WriteString(g.generateDropForeignKey(change))
        }
    }

    return sql.String()
}

func (g *MigrationGenerator) generateDownSQL(changes []SchemaChange) string {
    var sql strings.Builder

    // Reverse the changes
    for i := len(changes) - 1; i >= 0; i-- {
        change := changes[i]

        switch change.Type {
        case ChangeAddResource:
            tableName := g.sqlGen.toTableName(change.Resource)
            sql.WriteString(fmt.Sprintf("DROP TABLE %s CASCADE;\n\n", tableName))

        case ChangeDropResource:
            schema := g.oldSchema[change.Resource]
            sql.WriteString(g.sqlGen.GenerateCreateTable(schema))

        case ChangeAddField:
            sql.WriteString(g.generateDropColumn(change))

        case ChangeDropField:
            sql.WriteString(g.generateAddColumn(change))

        case ChangeModifyField:
            // Reverse the modification
            reverseChange := change
            reverseChange.OldValue, reverseChange.NewValue = change.NewValue, change.OldValue
            sql.WriteString(g.generateModifyColumn(reverseChange))
        }
    }

    return sql.String()
}

func (g *MigrationGenerator) generateAddColumn(change SchemaChange) string {
    field := change.NewValue.(*Field)
    tableName := g.sqlGen.toTableName(change.Resource)

    columnDef := g.sqlGen.generateColumn(change.Field, field)

    return fmt.Sprintf(
        "ALTER TABLE %s ADD COLUMN %s;\n\n",
        tableName,
        strings.TrimSuffix(columnDef, ",\n"),
    )
}

func (g *MigrationGenerator) generateDropColumn(change SchemaChange) string {
    tableName := g.sqlGen.toTableName(change.Resource)

    return fmt.Sprintf(
        "ALTER TABLE %s DROP COLUMN %s;\n\n",
        tableName,
        change.Field,
    )
}

func (g *MigrationGenerator) generateModifyColumn(change SchemaChange) string {
    newField := change.NewValue.(*Field)
    tableName := g.sqlGen.toTableName(change.Resource)
    newType := g.sqlGen.mapType(newField.Type)

    var sql strings.Builder

    // Change type
    sql.WriteString(fmt.Sprintf(
        "ALTER TABLE %s ALTER COLUMN %s TYPE %s;\n",
        tableName, change.Field, newType,
    ))

    // Change nullability
    if newField.Type.Nullable {
        sql.WriteString(fmt.Sprintf(
            "ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;\n",
            tableName, change.Field,
        ))
    } else {
        sql.WriteString(fmt.Sprintf(
            "ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;\n",
            tableName, change.Field,
        ))
    }

    sql.WriteString("\n")
    return sql.String()
}
```

### Migration Validator

```go
type MigrationValidator struct {
    db *sql.DB
}

func (v *MigrationValidator) Validate(migration *Migration) error {
    // Check for breaking changes
    if migration.Breaking {
        return fmt.Errorf(
            "migration contains breaking changes, requires manual review",
        )
    }

    // Check for data loss
    if migration.DataLoss {
        return fmt.Errorf(
            "migration may cause data loss, requires manual review",
        )
    }

    // Validate SQL syntax (dry run)
    if err := v.validateSQL(migration.Up); err != nil {
        return fmt.Errorf("invalid up SQL: %w", err)
    }

    if err := v.validateSQL(migration.Down); err != nil {
        return fmt.Errorf("invalid down SQL: %w", err)
    }

    return nil
}

func (v *MigrationValidator) validateSQL(sql string) error {
    // Use EXPLAIN to validate without executing
    _, err := v.db.Exec(fmt.Sprintf("EXPLAIN %s", sql))
    return err
}
```

### Migration Execution

```go
func (m *MigrationSystem) Migrate() error {
    // Get pending migrations
    pending := m.getPendingMigrations()

    for _, migration := range pending {
        // Validate migration
        if err := m.validator.Validate(migration); err != nil {
            return fmt.Errorf("migration %s validation failed: %w", migration.Name, err)
        }

        // Apply migration in transaction
        if err := m.applyMigration(migration); err != nil {
            return fmt.Errorf("migration %s failed: %w", migration.Name, err)
        }

        log.Printf("Applied migration: %s", migration.Name)
    }

    return nil
}

func (m *MigrationSystem) applyMigration(migration *Migration) error {
    tx, err := m.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Execute migration SQL
    if _, err := tx.Exec(migration.Up); err != nil {
        return err
    }

    // Record migration
    _, err = tx.Exec(
        "INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, NOW())",
        migration.Version,
        migration.Name,
    )
    if err != nil {
        return err
    }

    return tx.Commit()
}

func (m *MigrationSystem) Rollback() error {
    // Get last applied migration
    last := m.getLastMigration()
    if last == nil {
        return fmt.Errorf("no migrations to rollback")
    }

    tx, err := m.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Execute down migration
    if _, err := tx.Exec(last.Down); err != nil {
        return err
    }

    // Remove migration record
    _, err = tx.Exec(
        "DELETE FROM schema_migrations WHERE version = $1",
        last.Version,
    )
    if err != nil {
        return err
    }

    return tx.Commit()
}
```

### Safe Migration Rules

**Non-Breaking (auto-apply):**
- Add new resource
- Add new nullable field
- Add new relationship
- Add index
- Add constraint on nullable field

**Breaking (require explicit approval):**
- Drop resource
- Drop field
- Make field non-nullable
- Change field type
- Add non-nullable field without default
- Change cascade behavior

### Testing Strategy

**Unit Tests:**
- Test schema diffing algorithm
- Test breaking change detection
- Test data loss detection
- Test SQL generation

**Integration Tests:**
- Apply migrations to test database
- Test rollback functionality
- Simulate production scenarios
- Test all change types

**Chaos Testing:**
- Random schema changes
- Concurrent migrations
- Transaction failures

**Coverage Target:** >95%

### Estimated Effort

**Time:** 4 weeks
**Team:** 2 engineers
**Complexity:** High
**Risk:** High (data loss potential)

---

## Component 4: Query Builder & Scopes

### Responsibility

Type-safe query construction with N+1 prevention and SQL injection protection.

### Key Data Structures

```go
type QueryBuilder struct {
    resource    *ResourceSchema
    db          *sql.DB
    schemas     map[string]*ResourceSchema

    conditions  []*Condition
    joins       []*Join
    orderBy     []string
    groupBy     []string
    having      []*Condition
    limit       *int
    offset      *int
    includes    []string      // For eager loading
    scopes      []string      // Applied scopes

    // For building SQL
    paramCounter int
    args         []interface{}
}

type Condition struct {
    Field    string
    Operator Operator
    Value    interface{}
    Or       bool          // OR instead of AND
}

type Operator int
const (
    OpEqual Operator = iota
    OpNotEqual
    OpGreaterThan
    OpGreaterThanOrEqual
    OpLessThan
    OpLessThanOrEqual
    OpIn
    OpNotIn
    OpLike
    OpILike
    OpIsNull
    OpIsNotNull
    OpBetween
)

type Join struct {
    Type       JoinType
    Table      string
    Condition  string
}

type JoinType int
const (
    InnerJoin JoinType = iota
    LeftJoin
    RightJoin
)

type Scope struct {
    Name       string
    Parameters []Parameter
    Conditions []*Condition
    OrderBy    []string
    Limit      *int
    Includes   []string
}

type Parameter struct {
    Name string
    Type *TypeSpec
}
```

### Query Builder Implementation

```go
func NewQueryBuilder(resource *ResourceSchema, db *sql.DB, schemas map[string]*ResourceSchema) *QueryBuilder {
    return &QueryBuilder{
        resource:     resource,
        db:           db,
        schemas:      schemas,
        conditions:   []*Condition{},
        joins:        []*Join{},
        orderBy:      []string{},
        includes:     []string{},
        scopes:       []string{},
        paramCounter: 1,
        args:         []interface{}{},
    }
}

func (qb *QueryBuilder) Where(field string, op Operator, value interface{}) *QueryBuilder {
    qb.conditions = append(qb.conditions, &Condition{
        Field:    field,
        Operator: op,
        Value:    value,
        Or:       false,
    })
    return qb
}

func (qb *QueryBuilder) OrWhere(field string, op Operator, value interface{}) *QueryBuilder {
    qb.conditions = append(qb.conditions, &Condition{
        Field:    field,
        Operator: op,
        Value:    value,
        Or:       true,
    })
    return qb
}

func (qb *QueryBuilder) WhereIn(field string, values []interface{}) *QueryBuilder {
    return qb.Where(field, OpIn, values)
}

func (qb *QueryBuilder) WhereNull(field string) *QueryBuilder {
    return qb.Where(field, OpIsNull, nil)
}

func (qb *QueryBuilder) WhereNotNull(field string) *QueryBuilder {
    return qb.Where(field, OpIsNotNull, nil)
}

func (qb *QueryBuilder) WhereLike(field string, pattern string) *QueryBuilder {
    return qb.Where(field, OpLike, pattern)
}

func (qb *QueryBuilder) OrderBy(field string, direction string) *QueryBuilder {
    dir := strings.ToUpper(direction)
    if dir != "ASC" && dir != "DESC" {
        dir = "ASC"
    }
    qb.orderBy = append(qb.orderBy, fmt.Sprintf("%s %s", field, dir))
    return qb
}

func (qb *QueryBuilder) Limit(n int) *QueryBuilder {
    qb.limit = &n
    return qb
}

func (qb *QueryBuilder) Offset(n int) *QueryBuilder {
    qb.offset = &n
    return qb
}

func (qb *QueryBuilder) Includes(relationships ...string) *QueryBuilder {
    qb.includes = append(qb.includes, relationships...)
    return qb
}

func (qb *QueryBuilder) Scope(scopeName string, args ...interface{}) (*QueryBuilder, error) {
    scope, ok := qb.resource.Scopes[scopeName]
    if !ok {
        return nil, fmt.Errorf("unknown scope: %s", scopeName)
    }

    // Bind parameters
    boundScope, err := qb.bindScope(scope, args)
    if err != nil {
        return nil, err
    }

    // Apply scope conditions
    qb.conditions = append(qb.conditions, boundScope.Conditions...)

    // Apply scope ordering
    if len(boundScope.OrderBy) > 0 {
        qb.orderBy = append(qb.orderBy, boundScope.OrderBy...)
    }

    // Apply scope limit
    if boundScope.Limit != nil {
        qb.limit = boundScope.Limit
    }

    // Apply scope includes
    if len(boundScope.Includes) > 0 {
        qb.includes = append(qb.includes, boundScope.Includes...)
    }

    qb.scopes = append(qb.scopes, scopeName)
    return qb, nil
}

func (qb *QueryBuilder) ToSQL() (string, []interface{}) {
    var sql strings.Builder
    qb.args = []interface{}{}
    qb.paramCounter = 1

    tableName := toTableName(qb.resource.Name)
    sql.WriteString(fmt.Sprintf("SELECT * FROM %s", tableName))

    // JOINs
    for _, join := range qb.joins {
        sql.WriteString(fmt.Sprintf(" %s JOIN %s ON %s",
            qb.joinTypeSQL(join.Type),
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
            sql.WriteString(qb.conditionToSQL(cond))
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

    return sql.String(), qb.args
}

func (qb *QueryBuilder) conditionToSQL(cond *Condition) string {
    switch cond.Operator {
    case OpEqual:
        return fmt.Sprintf("%s = $%d", cond.Field, qb.addArg(cond.Value))
    case OpNotEqual:
        return fmt.Sprintf("%s != $%d", cond.Field, qb.addArg(cond.Value))
    case OpGreaterThan:
        return fmt.Sprintf("%s > $%d", cond.Field, qb.addArg(cond.Value))
    case OpGreaterThanOrEqual:
        return fmt.Sprintf("%s >= $%d", cond.Field, qb.addArg(cond.Value))
    case OpLessThan:
        return fmt.Sprintf("%s < $%d", cond.Field, qb.addArg(cond.Value))
    case OpLessThanOrEqual:
        return fmt.Sprintf("%s <= $%d", cond.Field, qb.addArg(cond.Value))
    case OpIn:
        return fmt.Sprintf("%s = ANY($%d)", cond.Field, qb.addArg(pq.Array(cond.Value)))
    case OpNotIn:
        return fmt.Sprintf("%s != ALL($%d)", cond.Field, qb.addArg(pq.Array(cond.Value)))
    case OpLike:
        return fmt.Sprintf("%s LIKE $%d", cond.Field, qb.addArg(cond.Value))
    case OpILike:
        return fmt.Sprintf("%s ILIKE $%d", cond.Field, qb.addArg(cond.Value))
    case OpIsNull:
        return fmt.Sprintf("%s IS NULL", cond.Field)
    case OpIsNotNull:
        return fmt.Sprintf("%s IS NOT NULL", cond.Field)
    case OpBetween:
        vals := cond.Value.([]interface{})
        return fmt.Sprintf("%s BETWEEN $%d AND $%d",
            cond.Field,
            qb.addArg(vals[0]),
            qb.addArg(vals[1]),
        )
    default:
        return ""
    }
}

func (qb *QueryBuilder) addArg(value interface{}) int {
    qb.args = append(qb.args, value)
    current := qb.paramCounter
    qb.paramCounter++
    return current
}

func (qb *QueryBuilder) Execute(ctx context.Context) ([]map[string]interface{}, error) {
    sql, args := qb.ToSQL()

    rows, err := qb.db.QueryContext(ctx, sql, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    results, err := qb.scanRows(rows)
    if err != nil {
        return nil, err
    }

    // Eager load relationships
    if len(qb.includes) > 0 {
        loader := NewRelationshipLoader(qb.db, qb.schemas)
        if err := loader.EagerLoad(results, qb.resource, qb.includes); err != nil {
            return nil, err
        }
    }

    return results, nil
}

func (qb *QueryBuilder) scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
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

    return results, nil
}

func (qb *QueryBuilder) First(ctx context.Context) (map[string]interface{}, error) {
    qb.Limit(1)
    results, err := qb.Execute(ctx)
    if err != nil {
        return nil, err
    }
    if len(results) == 0 {
        return nil, sql.ErrNoRows
    }
    return results[0], nil
}

func (qb *QueryBuilder) Count(ctx context.Context) (int, error) {
    sql, args := qb.ToSQL()
    // Replace SELECT * with SELECT COUNT(*)
    sql = strings.Replace(sql, "SELECT *", "SELECT COUNT(*)", 1)

    var count int
    err := qb.db.QueryRowContext(ctx, sql, args...).Scan(&count)
    return count, err
}
```

### Scope System

```go
// Example scope definition in resource:
// @scope published {
//   where: { status: "published", published_at: { lte: Time.now() } }
//   order_by: "published_at DESC"
// }

type ScopeCompiler struct {
    resource *ResourceSchema
}

func (c *ScopeCompiler) CompileScope(scopeDef *ScopeDefinition) (*Scope, error) {
    scope := &Scope{
        Name:       scopeDef.Name,
        Parameters: scopeDef.Parameters,
        Conditions: []*Condition{},
        OrderBy:    []string{},
    }

    // Compile where conditions
    for _, cond := range scopeDef.Where {
        compiled, err := c.compileCondition(cond)
        if err != nil {
            return nil, err
        }
        scope.Conditions = append(scope.Conditions, compiled)
    }

    // Compile order by
    if scopeDef.OrderBy != "" {
        scope.OrderBy = []string{scopeDef.OrderBy}
    }

    // Compile limit
    if scopeDef.Limit > 0 {
        scope.Limit = &scopeDef.Limit
    }

    return scope, nil
}

// Usage example:
// Post.Scope("published").Where("author_id", OpEqual, authorID).Execute(ctx)
```

### N+1 Query Detection

```go
type QueryCounter struct {
    queries  []string
    counts   map[string]int
    mu       sync.Mutex
}

func (qc *QueryCounter) Track(query string) {
    qc.mu.Lock()
    defer qc.mu.Unlock()

    qc.queries = append(qc.queries, query)
    qc.counts[query]++
}

func (qc *QueryCounter) DetectN1() []string {
    qc.mu.Lock()
    defer qc.mu.Unlock()

    var warnings []string

    // Look for repeated queries with only ID differences
    for query, count := range qc.counts {
        if count > 5 {  // Threshold
            warnings = append(warnings, fmt.Sprintf(
                "Potential N+1 query detected: %d executions of similar query\n%s",
                count, query,
            ))
        }
    }

    return warnings
}

// In development mode, attach to query builder
func (qb *QueryBuilder) Execute(ctx context.Context) ([]map[string]interface{}, error) {
    sql, args := qb.ToSQL()

    if devMode {
        queryCounter.Track(sql)
    }

    // ... rest of execution
}
```

### Testing Strategy

**Unit Tests:**
- Test each query builder method
- Test SQL generation
- Test parameter binding
- Test scope compilation

**Integration Tests:**
- Execute queries against test database
- Test complex query combinations
- Test N+1 detection
- Test eager loading

**Security Tests:**
- SQL injection attempts
- Parameter binding edge cases

**Coverage Target:** >90%

### Estimated Effort

**Time:** 3 weeks
**Team:** 1-2 engineers
**Complexity:** Medium
**Risk:** Medium (SQL injection, query performance)

---

## Component 5: Relationship Loading

### Responsibility

Efficient eager/lazy loading with N+1 prevention and circular reference handling.

### Key Data Structures

```go
type RelationshipLoader struct {
    db       *sql.DB
    schemas  map[string]*ResourceSchema
}

type LoadStrategy int
const (
    EagerLoad LoadStrategy = iota
    LazyLoad
)

type LazyRelation struct {
    loader     *RelationshipLoader
    parentID   interface{}
    relation   *Relationship
    loaded     bool
    value      interface{}
    mu         sync.Mutex
}
```

### Eager Loading Implementation

```go
func NewRelationshipLoader(db *sql.DB, schemas map[string]*ResourceSchema) *RelationshipLoader {
    return &RelationshipLoader{
        db:      db,
        schemas: schemas,
    }
}

// Eager loading prevents N+1 by loading all relationships in single queries
func (l *RelationshipLoader) EagerLoad(
    records []map[string]interface{},
    resource *ResourceSchema,
    includes []string,
) error {
    for _, include := range includes {
        rel, ok := resource.Relationships[include]
        if !ok {
            return fmt.Errorf("unknown relationship: %s", include)
        }

        switch rel.Type {
        case BelongsTo:
            if err := l.loadBelongsTo(records, rel); err != nil {
                return err
            }
        case HasMany:
            if err := l.loadHasMany(records, rel); err != nil {
                return err
            }
        case HasManyThrough:
            if err := l.loadHasManyThrough(records, rel); err != nil {
                return err
            }
        }
    }

    return nil
}

// BelongsTo: Load parent records
// Example: Post belongs_to User
//   - Collect all unique author_ids from posts
//   - Single query: SELECT * FROM users WHERE id = ANY($1)
//   - Map users back to posts
func (l *RelationshipLoader) loadBelongsTo(
    records []map[string]interface{},
    rel *Relationship,
) error {
    // Collect foreign key IDs
    var ids []interface{}
    idMap := make(map[string][]int)  // id -> record indices

    fk := rel.ForeignKey
    if fk == "" {
        fk = toSnakeCase(rel.TargetResource) + "_id"
    }

    for i, record := range records {
        if id, ok := record[fk]; ok && id != nil {
            idStr := fmt.Sprintf("%v", id)
            if _, seen := idMap[idStr]; !seen {
                ids = append(ids, id)
            }
            idMap[idStr] = append(idMap[idStr], i)
        }
    }

    if len(ids) == 0 {
        return nil
    }

    // Single query to fetch all related records
    targetSchema := l.schemas[rel.TargetResource]
    tableName := toTableName(rel.TargetResource)
    query := fmt.Sprintf(
        "SELECT * FROM %s WHERE id = ANY($1)",
        tableName,
    )

    rows, err := l.db.Query(query, pq.Array(ids))
    if err != nil {
        return err
    }
    defer rows.Close()

    // Map results back to parent records
    related := make(map[string]map[string]interface{})
    for rows.Next() {
        record, err := scanRow(rows, targetSchema)
        if err != nil {
            return err
        }
        id := fmt.Sprintf("%v", record["id"])
        related[id] = record
    }

    // Attach to parent records
    for i, record := range records {
        if id, ok := record[fk]; ok && id != nil {
            idStr := fmt.Sprintf("%v", id)
            if rel, ok := related[idStr]; ok {
                record[rel.FieldName] = rel
            }
        }
    }

    return nil
}

// HasMany: Load child records
// Example: Post has_many Comment
//   - Collect all post IDs
//   - Single query: SELECT * FROM comments WHERE post_id = ANY($1)
//   - Group comments by post_id
//   - Attach to posts
func (l *RelationshipLoader) loadHasMany(
    records []map[string]interface{},
    rel *Relationship,
) error {
    // Collect parent IDs
    var parentIDs []interface{}
    for _, record := range records {
        parentIDs = append(parentIDs, record["id"])
    }

    if len(parentIDs) == 0 {
        return nil
    }

    fk := rel.ForeignKey
    if fk == "" {
        fk = toSnakeCase(rel.TargetResource) + "_id"
    }

    // Single query to fetch all related records
    targetSchema := l.schemas[rel.TargetResource]
    tableName := toTableName(rel.TargetResource)
    query := fmt.Sprintf(
        "SELECT * FROM %s WHERE %s = ANY($1)",
        tableName,
        fk,
    )

    if rel.OrderBy != "" {
        query += fmt.Sprintf(" ORDER BY %s", rel.OrderBy)
    }

    rows, err := l.db.Query(query, pq.Array(parentIDs))
    if err != nil {
        return err
    }
    defer rows.Close()

    // Group by parent ID
    grouped := make(map[string][]map[string]interface{})
    for rows.Next() {
        record, err := scanRow(rows, targetSchema)
        if err != nil {
            return err
        }
        parentID := fmt.Sprintf("%v", record[fk])
        grouped[parentID] = append(grouped[parentID], record)
    }

    // Attach to parent records
    for _, record := range records {
        id := fmt.Sprintf("%v", record["id"])
        if children, ok := grouped[id]; ok {
            record[rel.FieldName] = children
        } else {
            record[rel.FieldName] = []map[string]interface{}{}
        }
    }

    return nil
}

// HasManyThrough: Load through junction table
// Example: Post has_many Tag through PostTag
//   - Three-way join through junction table
//   - Single query with JOIN
//   - Group by parent ID
func (l *RelationshipLoader) loadHasManyThrough(
    records []map[string]interface{},
    rel *Relationship,
) error {
    var parentIDs []interface{}
    for _, record := range records {
        parentIDs = append(parentIDs, record["id"])
    }

    if len(parentIDs) == 0 {
        return nil
    }

    targetSchema := l.schemas[rel.TargetResource]
    targetTable := toTableName(rel.TargetResource)

    query := fmt.Sprintf(`
        SELECT t.*, j.%s as __parent_id
        FROM %s t
        JOIN %s j ON t.id = j.%s
        WHERE j.%s = ANY($1)
    `,
        rel.ForeignKey,
        targetTable,
        rel.JoinTable,
        rel.AssociationKey,
        rel.ForeignKey,
    )

    rows, err := l.db.Query(query, pq.Array(parentIDs))
    if err != nil {
        return err
    }
    defer rows.Close()

    // Group by parent ID
    grouped := make(map[string][]map[string]interface{})
    for rows.Next() {
        record, err := scanRow(rows, targetSchema)
        if err != nil {
            return err
        }
        parentID := fmt.Sprintf("%v", record["__parent_id"])
        delete(record, "__parent_id")  // Remove join artifact
        grouped[parentID] = append(grouped[parentID], record)
    }

    // Attach to parent records
    for _, record := range records {
        id := fmt.Sprintf("%v", record["id"])
        if related, ok := grouped[id]; ok {
            record[rel.FieldName] = related
        } else {
            record[rel.FieldName] = []map[string]interface{}{}
        }
    }

    return nil
}
```

### Lazy Loading Implementation

```go
type LazyRelation struct {
    loader     *RelationshipLoader
    parentID   interface{}
    relation   *Relationship
    loaded     bool
    value      interface{}
    mu         sync.Mutex
}

func NewLazyRelation(
    loader *RelationshipLoader,
    parentID interface{},
    relation *Relationship,
) *LazyRelation {
    return &LazyRelation{
        loader:   loader,
        parentID: parentID,
        relation: relation,
        loaded:   false,
    }
}

func (lr *LazyRelation) Get() (interface{}, error) {
    lr.mu.Lock()
    defer lr.mu.Unlock()

    if lr.loaded {
        return lr.value, nil
    }

    // Load on demand
    value, err := lr.loader.LoadRelation(lr.parentID, lr.relation)
    if err != nil {
        return nil, err
    }

    lr.value = value
    lr.loaded = true
    return lr.value, nil
}

func (l *RelationshipLoader) LoadRelation(
    parentID interface{},
    rel *Relationship,
) (interface{}, error) {
    switch rel.Type {
    case BelongsTo:
        return l.loadSingleBelongsTo(parentID, rel)
    case HasMany:
        return l.loadSingleHasMany(parentID, rel)
    case HasManyThrough:
        return l.loadSingleHasManyThrough(parentID, rel)
    default:
        return nil, fmt.Errorf("unknown relationship type")
    }
}

func (l *RelationshipLoader) loadSingleBelongsTo(
    parentID interface{},
    rel *Relationship,
) (map[string]interface{}, error) {
    tableName := toTableName(rel.TargetResource)
    query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", tableName)

    targetSchema := l.schemas[rel.TargetResource]
    row := l.db.QueryRow(query, parentID)

    return scanSingleRow(row, targetSchema)
}

func (l *RelationshipLoader) loadSingleHasMany(
    parentID interface{},
    rel *Relationship,
) ([]map[string]interface{}, error) {
    fk := rel.ForeignKey
    if fk == "" {
        fk = toSnakeCase(rel.TargetResource) + "_id"
    }

    tableName := toTableName(rel.TargetResource)
    query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", tableName, fk)

    if rel.OrderBy != "" {
        query += fmt.Sprintf(" ORDER BY %s", rel.OrderBy)
    }

    rows, err := l.db.Query(query, parentID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    targetSchema := l.schemas[rel.TargetResource]
    return scanRows(rows, targetSchema)
}
```

### Circular Reference Prevention

```go
type LoadContext struct {
    visited map[string]bool
    depth   int
    maxDepth int
}

func NewLoadContext(maxDepth int) *LoadContext {
    return &LoadContext{
        visited:  make(map[string]bool),
        depth:    0,
        maxDepth: maxDepth,
    }
}

func (l *RelationshipLoader) EagerLoadWithContext(
    records []map[string]interface{},
    resource *ResourceSchema,
    includes []string,
    ctx *LoadContext,
) error {
    // Check depth limit
    if ctx.depth >= ctx.maxDepth {
        return fmt.Errorf("max relationship depth exceeded (%d)", ctx.maxDepth)
    }

    // Check for cycles
    resourceKey := fmt.Sprintf("%s:%v", resource.Name, records[0]["id"])
    if ctx.visited[resourceKey] {
        return nil  // Already visited, skip
    }
    ctx.visited[resourceKey] = true

    // Increment depth
    ctx.depth++
    defer func() { ctx.depth-- }()

    // Load relationships
    for _, include := range includes {
        rel, ok := resource.Relationships[include]
        if !ok {
            return fmt.Errorf("unknown relationship: %s", include)
        }

        // Load relationship
        switch rel.Type {
        case BelongsTo:
            if err := l.loadBelongsTo(records, rel); err != nil {
                return err
            }
        case HasMany:
            if err := l.loadHasMany(records, rel); err != nil {
                return err
            }
        }
    }

    return nil
}

// Usage:
// ctx := NewLoadContext(10)  // Max 10 levels deep
// loader.EagerLoadWithContext(posts, postSchema, []string{"author", "comments"}, ctx)
```

### Testing Strategy

**Unit Tests:**
- Test each relationship type loading
- Test ID collection and mapping
- Test empty result handling

**Integration Tests:**
- Test with real database
- Test complex relationship chains
- Test circular references
- Test N+1 prevention

**Performance Tests:**
- Benchmark eager vs lazy loading
- Test with 1000+ records
- Measure query counts
- Memory usage tests

**Coverage Target:** >90%

### Estimated Effort

**Time:** 5 weeks
**Team:** 2 engineers
**Complexity:** High
**Risk:** High (performance critical)

---

*[Due to length constraints, I'll continue with the remaining components in the next section]*

### To Be Continued

The remaining components (6-10), development phases, and testing strategy will complete this implementation guide. The document is structured to provide:

- Complete Go code examples for each component
- Clear data structures and algorithms
- Testing strategies with coverage targets
- Integration points with other system components
- Performance targets and optimization strategies
- Risk mitigation approaches

**Total Estimated Effort:** 31-35 weeks (155-175 person-days)
**Team Size:** 2-3 engineers
**Primary Risk Areas:** Migration safety, N+1 prevention, transaction management

---

**Document Status:** Part 1 Complete - Components 1-5
**Next:** Components 6-10, Development Phases, Testing Strategy

## Component 6: CRUD Operations

### Responsibility

Auto-generate type-safe CRUD methods for each resource with validation, hooks, and transaction support.

### Key Data Structures

```go
type CRUDOperations struct {
    resource   *ResourceSchema
    db         *sql.DB
    validator  *ValidationEngine
    hooks      *HookExecutor
    txManager  *TransactionManager
    loader     *RelationshipLoader
}

type Operation int
const (
    OperationCreate Operation = iota
    OperationRead
    OperationUpdate
    OperationDelete
)
```

### Implementation

```go
func NewCRUDOperations(
    resource *ResourceSchema,
    db *sql.DB,
    validator *ValidationEngine,
    hooks *HookExecutor,
    txManager *TransactionManager,
) *CRUDOperations {
    return &CRUDOperations{
        resource:  resource,
        db:        db,
        validator: validator,
        hooks:     hooks,
        txManager: txManager,
    }
}

func (c *CRUDOperations) Create(
    ctx context.Context,
    data map[string]interface{},
) (map[string]interface{}, error) {
    var result map[string]interface{}

    err := c.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
        // 1. Execute before hooks
        if err := c.hooks.ExecuteHooks(ctx, c.resource, BeforeCreate, data); err != nil {
            return err
        }

        // 2. Validate
        if err := c.validator.Validate(ctx, c.resource, data, OperationCreate); err != nil {
            return err
        }

        // 3. Insert
        record, err := c.insert(ctx, tx, data)
        if err != nil {
            return err
        }

        // 4. Execute after hooks
        if err := c.hooks.ExecuteHooks(ctx, c.resource, AfterCreate, record); err != nil {
            return err
        }

        result = record
        return nil
    })

    return result, err
}

func (c *CRUDOperations) Update(
    ctx context.Context,
    id interface{},
    data map[string]interface{},
) (map[string]interface{}, error) {
    var result map[string]interface{}

    err := c.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
        // 1. Load existing record (for change tracking)
        existing, err := c.findByID(ctx, tx, id)
        if err != nil {
            return err
        }

        // 2. Merge changes
        record := c.mergeChanges(existing, data)

        // 3. Set up change tracking
        changeTracker := NewChangeTracker(existing, record)
        record["__changes__"] = changeTracker

        // 4. Execute before hooks
        if err := c.hooks.ExecuteHooks(ctx, c.resource, BeforeUpdate, record); err != nil {
            return err
        }

        // 5. Validate
        if err := c.validator.Validate(ctx, c.resource, record, OperationUpdate); err != nil {
            return err
        }

        // 6. Update
        updated, err := c.update(ctx, tx, id, record)
        if err != nil {
            return err
        }

        // 7. Execute after hooks
        updated["__changes__"] = changeTracker
        if err := c.hooks.ExecuteHooks(ctx, c.resource, AfterUpdate, updated); err != nil {
            return err
        }

        result = updated
        return nil
    })

    return result, err
}

func (c *CRUDOperations) Delete(
    ctx context.Context,
    id interface{},
) error {
    return c.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
        // 1. Load existing record
        record, err := c.findByID(ctx, tx, id)
        if err != nil {
            return err
        }

        // 2. Execute before hooks
        if err := c.hooks.ExecuteHooks(ctx, c.resource, BeforeDelete, record); err != nil {
            return err
        }

        // 3. Delete
        if err := c.delete(ctx, tx, id); err != nil {
            return err
        }

        // 4. Execute after hooks
        if err := c.hooks.ExecuteHooks(ctx, c.resource, AfterDelete, record); err != nil {
            return err
        }

        return nil
    })
}

func (c *CRUDOperations) Find(
    ctx context.Context,
    id interface{},
) (map[string]interface{}, error) {
    return c.findByID(ctx, c.db, id)
}

func (c *CRUDOperations) FindBy(
    ctx context.Context,
    field string,
    value interface{},
) (map[string]interface{}, error) {
    tableName := toTableName(c.resource.Name)
    query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1 LIMIT 1", tableName, field)

    row := c.db.QueryRowContext(ctx, query, value)
    return scanSingleRow(row, c.resource)
}

func (c *CRUDOperations) Where(conditions map[string]interface{}) *QueryBuilder {
    qb := NewQueryBuilder(c.resource, c.db, nil)
    for field, value := range conditions {
        qb.Where(field, OpEqual, value)
    }
    return qb
}

func (c *CRUDOperations) insert(
    ctx context.Context,
    tx *sql.Tx,
    data map[string]interface{},
) (map[string]interface{}, error) {
    tableName := toTableName(c.resource.Name)

    // Build INSERT statement
    var fields []string
    var placeholders []string
    var values []interface{}
    counter := 1

    for field, value := range data {
        fields = append(fields, field)
        placeholders = append(placeholders, fmt.Sprintf("$%d", counter))
        values = append(values, value)
        counter++
    }

    query := fmt.Sprintf(
        "INSERT INTO %s (%s) VALUES (%s) RETURNING *",
        tableName,
        strings.Join(fields, ", "),
        strings.Join(placeholders, ", "),
    )

    row := tx.QueryRowContext(ctx, query, values...)
    return scanSingleRow(row, c.resource)
}

func (c *CRUDOperations) update(
    ctx context.Context,
    tx *sql.Tx,
    id interface{},
    data map[string]interface{},
) (map[string]interface{}, error) {
    tableName := toTableName(c.resource.Name)

    // Build UPDATE statement
    var sets []string
    var values []interface{}
    counter := 1

    for field, value := range data {
        if field == "id" || field == "created_at" || field == "__changes__" {
            continue  // Skip immutable fields
        }
        sets = append(sets, fmt.Sprintf("%s = $%d", field, counter))
        values = append(values, value)
        counter++
    }

    values = append(values, id)

    query := fmt.Sprintf(
        "UPDATE %s SET %s WHERE id = $%d RETURNING *",
        tableName,
        strings.Join(sets, ", "),
        counter,
    )

    row := tx.QueryRowContext(ctx, query, values...)
    return scanSingleRow(row, c.resource)
}

func (c *CRUDOperations) delete(
    ctx context.Context,
    tx *sql.Tx,
    id interface{},
) error {
    tableName := toTableName(c.resource.Name)
    query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", tableName)

    result, err := tx.ExecContext(ctx, query, id)
    if err != nil {
        return err
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return err
    }

    if rows == 0 {
        return sql.ErrNoRows
    }

    return nil
}

func (c *CRUDOperations) findByID(
    ctx context.Context,
    db interface{ QueryRowContext(context.Context, string, ...interface{}) *sql.Row },
    id interface{},
) (map[string]interface{}, error) {
    tableName := toTableName(c.resource.Name)
    query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", tableName)

    row := db.QueryRowContext(ctx, query, id)
    return scanSingleRow(row, c.resource)
}

func (c *CRUDOperations) mergeChanges(
    existing, updates map[string]interface{},
) map[string]interface{} {
    result := make(map[string]interface{})

    // Copy existing
    for k, v := range existing {
        result[k] = v
    }

    // Apply updates
    for k, v := range updates {
        result[k] = v
    }

    return result
}
```

### Batch Operations

```go
func (c *CRUDOperations) CreateMany(
    ctx context.Context,
    records []map[string]interface{},
) ([]map[string]interface{}, error) {
    var results []map[string]interface{}

    err := c.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
        for _, data := range records {
            record, err := c.insert(ctx, tx, data)
            if err != nil {
                return err
            }
            results = append(results, record)
        }
        return nil
    })

    return results, err
}

func (c *CRUDOperations) UpdateMany(
    ctx context.Context,
    conditions map[string]interface{},
    updates map[string]interface{},
) (int, error) {
    var count int

    err := c.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
        tableName := toTableName(c.resource.Name)

        // Build WHERE clause
        var whereConditions []string
        var whereValues []interface{}
        counter := 1
        for field, value := range conditions {
            whereConditions = append(whereConditions, fmt.Sprintf("%s = $%d", field, counter))
            whereValues = append(whereValues, value)
            counter++
        }

        // Build SET clause
        var sets []string
        var setValues []interface{}
        for field, value := range updates {
            sets = append(sets, fmt.Sprintf("%s = $%d", field, counter))
            setValues = append(setValues, value)
            counter++
        }

        values := append(whereValues, setValues...)

        query := fmt.Sprintf(
            "UPDATE %s SET %s WHERE %s",
            tableName,
            strings.Join(sets, ", "),
            strings.Join(whereConditions, " AND "),
        )

        result, err := tx.ExecContext(ctx, query, values...)
        if err != nil {
            return err
        }

        rows, err := result.RowsAffected()
        if err != nil {
            return err
        }

        count = int(rows)
        return nil
    })

    return count, err
}
```

### Testing Strategy

**Unit Tests:**
- Test each CRUD operation
- Test with various data types
- Test validation integration
- Test hook execution order

**Integration Tests:**
- Test full CRUD cycle
- Test transaction rollback
- Test error handling
- Test batch operations

**Coverage Target:** >90%

### Estimated Effort

**Time:** 2-3 weeks
**Team:** 1 engineer
**Complexity:** Medium
**Risk:** Low

---

## Component 7: Validation Engine

### Responsibility

Execute multi-layer validation from field-level constraints to runtime invariants.

### Key Data Structures

```go
type ValidationEngine struct {
    interpreter *Interpreter
}

type Validator struct {
    Type       ValidationType
    Code       *AST              // For procedural validation
    Constraint *Constraint       // For declarative constraints
    Invariant  *Invariant        // For runtime invariants
}

type ValidationType int
const (
    ValidationFieldConstraint ValidationType = iota
    ValidationResourceConstraint
    ValidationProcedural
    ValidationInvariant
)

type ValidationError struct {
    Errors []FieldError
}

type FieldError struct {
    Field   string
    Message string
}
```

### Validation Layers

```go
func (ve *ValidationEngine) Validate(
    ctx context.Context,
    resource *ResourceSchema,
    record map[string]interface{},
    operation Operation,
) error {
    var errors []FieldError

    // Layer 1: Field-level constraints (@min, @max, @pattern, etc.)
    for fieldName, field := range resource.Fields {
        value := record[fieldName]
        for _, constraint := range field.Constraints {
            if err := ve.validateConstraint(constraint, value); err != nil {
                errors = append(errors, FieldError{
                    Field:   fieldName,
                    Message: err.Error(),
                })
            }
        }
    }

    // Layer 2: Resource-level constraints (@constraint blocks)
    for _, constraint := range resource.Constraints {
        if !constraint.AppliesTo(operation) {
            continue
        }

        if err := ve.validateResourceConstraint(ctx, constraint, record); err != nil {
            errors = append(errors, FieldError{
                Field:   constraint.Name,
                Message: err.Error(),
            })
        }
    }

    // Layer 3: Procedural validation (@validate blocks)
    for _, validator := range resource.Validators {
        if err := ve.executeProcedural(ctx, validator, record); err != nil {
            errors = append(errors, FieldError{
                Field:   "procedural",
                Message: err.Error(),
            })
        }
    }

    // Layer 4: Invariants (@invariant blocks)
    for _, invariant := range resource.Invariants {
        if err := ve.checkInvariant(ctx, invariant, record); err != nil {
            errors = append(errors, FieldError{
                Field:   invariant.Name,
                Message: err.Error(),
            })
        }
    }

    if len(errors) > 0 {
        return &ValidationError{Errors: errors}
    }

    return nil
}

func (ve *ValidationEngine) validateConstraint(
    constraint *Constraint,
    value interface{},
) error {
    switch constraint.Type {
    case ConstraintMin:
        return ve.validateMin(value, constraint.Value)
    case ConstraintMax:
        return ve.validateMax(value, constraint.Value)
    case ConstraintPattern:
        return ve.validatePattern(value, constraint.Pattern)
    case ConstraintUnique:
        return ve.validateUnique(value, constraint.Field, constraint.Resource)
    default:
        return nil
    }
}

func (ve *ValidationEngine) validateMin(value interface{}, min interface{}) error {
    switch v := value.(type) {
    case int:
        if v < min.(int) {
            return fmt.Errorf("must be at least %v (got %v)", min, v)
        }
    case float64:
        if v < min.(float64) {
            return fmt.Errorf("must be at least %v (got %v)", min, v)
        }
    case string:
        if len(v) < min.(int) {
            return fmt.Errorf("must be at least %d characters (got %d)", min.(int), len(v))
        }
    }
    return nil
}

func (ve *ValidationEngine) validateMax(value interface{}, max interface{}) error {
    switch v := value.(type) {
    case int:
        if v > max.(int) {
            return fmt.Errorf("must be at most %v (got %v)", max, v)
        }
    case float64:
        if v > max.(float64) {
            return fmt.Errorf("must be at most %v (got %v)", max, v)
        }
    case string:
        if len(v) > max.(int) {
            return fmt.Errorf("must be at most %d characters (got %d)", max.(int), len(v))
        }
    }
    return nil
}

func (ve *ValidationEngine) validatePattern(value interface{}, pattern *regexp.Regexp) error {
    str, ok := value.(string)
    if !ok {
        return fmt.Errorf("pattern constraint requires string value")
    }

    if !pattern.MatchString(str) {
        return fmt.Errorf("does not match required pattern %s", pattern.String())
    }

    return nil
}

func (ve *ValidationEngine) validateResourceConstraint(
    ctx context.Context,
    constraint *Constraint,
    record map[string]interface{},
) error {
    // Evaluate "when" condition
    if constraint.When != nil {
        result, err := ve.interpreter.Evaluate(constraint.When, record)
        if err != nil {
            return err
        }
        if !result.(bool) {
            return nil  // Constraint doesn't apply
        }
    }

    // Evaluate "condition"
    result, err := ve.interpreter.Evaluate(constraint.Condition, record)
    if err != nil {
        return err
    }

    if !result.(bool) {
        return fmt.Errorf("%s", constraint.ErrorMessage)
    }

    return nil
}

func (ve *ValidationEngine) executeProcedural(
    ctx context.Context,
    validator *Validator,
    record map[string]interface{},
) error {
    execCtx := &ExecutionContext{
        Self:    record,
        Context: ctx,
    }

    _, err := ve.interpreter.Execute(validator.Code, execCtx)
    return err
}

func (ve *ValidationEngine) checkInvariant(
    ctx context.Context,
    invariant *Invariant,
    record map[string]interface{},
) error {
    result, err := ve.interpreter.Evaluate(invariant.Condition, record)
    if err != nil {
        return err
    }

    if !result.(bool) {
        return fmt.Errorf("%s", invariant.ErrorMessage)
    }

    return nil
}
```

### Error Formatting

```go
func (e *ValidationError) Error() string {
    var messages []string
    for _, err := range e.Errors {
        messages = append(messages, fmt.Sprintf("  - %s: %s", err.Field, err.Message))
    }
    return "Validation failed:\n" + strings.Join(messages, "\n")
}

// Example output:
// Validation failed:
//   - title: must be at least 5 characters (got 3)
//   - category: published posts must have a category
//   - published_at: must be a future date for scheduled posts
```

### Testing Strategy

**Unit Tests:**
- Test each constraint type
- Test validation failure paths
- Test error messages
- Test conditional constraints

**Integration Tests:**
- Test with CRUD operations
- Test validation order
- Test complex constraints

**Coverage Target:** >90%

### Estimated Effort

**Time:** 3 weeks
**Team:** 1-2 engineers
**Complexity:** Medium-High
**Risk:** Medium

---

## Component 8: Lifecycle Hooks

### Responsibility

Execute hooks in correct order with transaction support and async operations.

### Key Data Structures

```go
type HookExecutor struct {
    interpreter *Interpreter
    txManager   *TransactionManager
}

type ExecutionContext struct {
    Self          map[string]interface{}
    DB            *sql.Tx
    Context       context.Context
    CurrentUser   interface{}
    ChangeTracker *ChangeTracker
}

type ExecutionResult struct {
    Value             interface{}
    AsyncOperations   []*AsyncOperation
}

type AsyncOperation struct {
    Name    string
    Code    *AST
    Context *ExecutionContext
}
```

### Hook Execution

```go
func NewHookExecutor(interpreter *Interpreter, txManager *TransactionManager) *HookExecutor {
    return &HookExecutor{
        interpreter: interpreter,
        txManager:   txManager,
    }
}

func (e *HookExecutor) ExecuteHooks(
    ctx context.Context,
    resource *ResourceSchema,
    hookType HookType,
    record map[string]interface{},
) error {
    hooks := resource.Hooks[hookType]
    if len(hooks) == 0 {
        return nil
    }

    for _, hook := range hooks {
        if hook.Transaction {
            // Execute in transaction (already in one if from CRUD operation)
            if err := e.executeHook(ctx, hook, record, nil); err != nil {
                return err
            }
        } else {
            // Execute without transaction
            if err := e.executeHook(ctx, hook, record, nil); err != nil {
                return err
            }
        }
    }

    return nil
}

func (e *HookExecutor) executeHook(
    ctx context.Context,
    hook *Hook,
    record map[string]interface{},
    tx *sql.Tx,
) error {
    // Set up execution context
    execCtx := &ExecutionContext{
        Self:          record,
        DB:            tx,
        Context:       ctx,
        CurrentUser:   ctx.Value("user"),
        ChangeTracker: record["__changes__"].(*ChangeTracker),
    }

    // Execute hook code
    result, err := e.interpreter.Execute(hook.Code, execCtx)
    if err != nil {
        return err
    }

    // Handle async blocks
    if len(result.AsyncOperations) > 0 {
        return e.handleAsyncOperations(ctx, result.AsyncOperations)
    }

    return nil
}

func (e *HookExecutor) handleAsyncOperations(
    ctx context.Context,
    operations []*AsyncOperation,
) error {
    for _, op := range operations {
        go func(asyncOp *AsyncOperation) {
            // Execute async operation in background
            _, err := e.interpreter.Execute(asyncOp.Code, asyncOp.Context)
            if err != nil {
                // Log error, don't fail parent transaction
                log.Printf("async operation %s failed: %v", asyncOp.Name, err)
            }
        }(op)
    }

    return nil
}
```

### Hook Execution Order

```
CREATE:
  1. BeforeCreate hooks (in transaction)
  2. Validation
  3. INSERT
  4. AfterCreate hooks (in transaction)
  5. Async operations (background)

UPDATE:
  1. Load existing record (for change tracking)
  2. BeforeUpdate hooks (in transaction)
  3. Validation
  4. UPDATE
  5. AfterUpdate hooks (in transaction)
  6. Async operations (background)

DELETE:
  1. BeforeDelete hooks (in transaction)
  2. DELETE
  3. AfterDelete hooks (in transaction)
  4. Async operations (background)
```

### Testing Strategy

**Unit Tests:**
- Test each hook type
- Test hook execution order
- Test error propagation
- Test async operations

**Integration Tests:**
- Test with CRUD operations
- Test transaction rollback
- Test change tracking in update hooks

**Coverage Target:** >90%

### Estimated Effort

**Time:** 4 weeks
**Team:** 2 engineers
**Complexity:** High
**Risk:** Medium

---

## Component 9: Transaction Management

### Responsibility

Robust ACID transaction support with nested transactions and deadlock handling.

### Implementation

```go
type TransactionManager struct {
    db *sql.DB
}

func NewTransactionManager(db *sql.DB) *TransactionManager {
    return &TransactionManager{db: db}
}

func (tm *TransactionManager) WithTransaction(
    ctx context.Context,
    fn func(tx *sql.Tx) error,
) error {
    tx, err := tm.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }

    defer func() {
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p)  // Re-throw panic
        }
    }()

    if err := fn(tx); err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("tx failed: %v, rollback failed: %v", err, rbErr)
        }
        return err
    }

    return tx.Commit()
}

func (tm *TransactionManager) WithNestedTransaction(
    ctx context.Context,
    tx *sql.Tx,
    fn func(tx *sql.Tx) error,
) error {
    // Use savepoints for nested transactions
    savepoint := fmt.Sprintf("sp_%d", time.Now().UnixNano())

    if _, err := tx.Exec(fmt.Sprintf("SAVEPOINT %s", savepoint)); err != nil {
        return err
    }

    defer func() {
        if p := recover(); p != nil {
            tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", savepoint))
            panic(p)
        }
    }()

    if err := fn(tx); err != nil {
        if _, rbErr := tx.Exec(fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", savepoint)); rbErr != nil {
            return fmt.Errorf("tx failed: %v, rollback failed: %v", err, rbErr)
        }
        return err
    }

    _, err := tx.Exec(fmt.Sprintf("RELEASE SAVEPOINT %s", savepoint))
    return err
}

func (tm *TransactionManager) WithRetry(
    ctx context.Context,
    maxRetries int,
    fn func(tx *sql.Tx) error,
) error {
    var lastErr error

    for attempt := 0; attempt < maxRetries; attempt++ {
        err := tm.WithTransaction(ctx, fn)
        if err == nil {
            return nil
        }

        // Check if deadlock error
        if isDeadlockError(err) {
            lastErr = err
            // Exponential backoff
            time.Sleep(time.Duration(1<<uint(attempt)) * 100 * time.Millisecond)
            continue
        }

        // Non-deadlock error, fail immediately
        return err
    }

    return fmt.Errorf("transaction failed after %d retries: %w", maxRetries, lastErr)
}

func isDeadlockError(err error) bool {
    // PostgreSQL deadlock error code: 40P01
    return strings.Contains(err.Error(), "40P01") ||
           strings.Contains(err.Error(), "deadlock detected")
}
```

### Testing Strategy

**Unit Tests:**
- Test transaction commit
- Test transaction rollback
- Test nested transactions
- Test deadlock retry

**Integration Tests:**
- Test with concurrent operations
- Simulate deadlocks
- Test timeout behavior

**Coverage Target:** >90%

### Estimated Effort

**Time:** 2 weeks
**Team:** 1 engineer
**Complexity:** Medium
**Risk:** High (data consistency critical)

---

## Component 10: Change Tracking

### Responsibility

Track field changes for update hooks to enable conditional logic.

### Implementation

```go
type ChangeTracker struct {
    original map[string]interface{}
    current  map[string]interface{}
    changes  map[string]*FieldChange
}

type FieldChange struct {
    Field    string
    OldValue interface{}
    NewValue interface{}
}

func NewChangeTracker(original, current map[string]interface{}) *ChangeTracker {
    ct := &ChangeTracker{
        original: original,
        current:  current,
        changes:  make(map[string]*FieldChange),
    }

    ct.computeChanges()
    return ct
}

func (ct *ChangeTracker) computeChanges() {
    for field, newValue := range ct.current {
        oldValue := ct.original[field]

        if !reflect.DeepEqual(oldValue, newValue) {
            ct.changes[field] = &FieldChange{
                Field:    field,
                OldValue: oldValue,
                NewValue: newValue,
            }
        }
    }
}

func (ct *ChangeTracker) FieldChanged(field string) bool {
    _, ok := ct.changes[field]
    return ok
}

func (ct *ChangeTracker) FieldChangedTo(field string, value interface{}) bool {
    change, ok := ct.changes[field]
    if !ok {
        return false
    }
    return reflect.DeepEqual(change.NewValue, value)
}

func (ct *ChangeTracker) FieldChangedFrom(field string, value interface{}) bool {
    change, ok := ct.changes[field]
    if !ok {
        return false
    }
    return reflect.DeepEqual(change.OldValue, value)
}

func (ct *ChangeTracker) PreviousValue(field string) interface{} {
    return ct.original[field]
}

func (ct *ChangeTracker) Changes() map[string]*FieldChange {
    return ct.changes
}

func (ct *ChangeTracker) ChangedFields() []string {
    fields := make([]string, 0, len(ct.changes))
    for field := range ct.changes {
        fields = append(fields, field)
    }
    return fields
}
```

### Usage in Hooks

```go
// In update hook:
// @after update {
//   if self.status_changed_to?("published") {
//     self.published_at = Time.now()
//   }
//
//   if self.content_changed? {
//     Revision.create!({...})
//   }
// }

// Interpreter resolves:
// - self.status_changed_to?("published") -> changeTracker.FieldChangedTo("status", "published")
// - self.content_changed? -> changeTracker.FieldChanged("content")
```

### Testing Strategy

**Unit Tests:**
- Test change detection
- Test with various data types
- Test with nested fields
- Test with nil values

**Integration Tests:**
- Test with update operations
- Test in hooks

**Coverage Target:** >90%

### Estimated Effort

**Time:** 1 week
**Team:** 1 engineer
**Complexity:** Low
**Risk:** Low

---

## Development Phases

### Phase 0: Foundation (2 weeks)

**Goal:** Set up core infrastructure

**Deliverables:**
- [ ] Project structure
- [ ] Database connection management
- [ ] Basic type system definitions
- [ ] Test infrastructure

**Team:** 1 engineer

**Exit Criteria:**
- Can connect to PostgreSQL
- Basic types defined in Go
- Test suite runs

---

### Phase 1: Schema Definition & Validation (3 weeks)

**Goal:** Parse and validate resource definitions

**Deliverables:**
- [ ] Type system implementation
- [ ] Field definition parser
- [ ] Relationship definition parser
- [ ] Semantic validation
- [ ] Nullability enforcement
- [ ] Constraint validation

**Team:** 2 engineers

**Exit Criteria:**
- Can parse complete resource definitions
- All validations work
- Error messages are clear
- Test coverage >90%

---

### Phase 2: Schema Generation (2 weeks)

**Goal:** Generate SQL DDL from resource definitions

**Deliverables:**
- [ ] Type mapping (Go types → SQL types)
- [ ] CREATE TABLE generation
- [ ] Foreign key generation
- [ ] Index generation
- [ ] Multi-database support (PostgreSQL, MySQL, SQLite)

**Team:** 1 engineer

**Exit Criteria:**
- Generates correct SQL for all types
- Foreign keys work correctly
- Indexes created appropriately
- Works on PostgreSQL, MySQL, SQLite

---

### Phase 3: Migration System (4 weeks)

**Goal:** Safe schema evolution

**Deliverables:**
- [ ] Schema differ
- [ ] Migration generation
- [ ] Migration application
- [ ] Rollback support
- [ ] Breaking change detection
- [ ] Data loss prevention
- [ ] Dry-run mode

**Team:** 2 engineers

**Exit Criteria:**
- Can detect all schema changes
- Generates correct migrations
- Rollback works reliably
- Breaking changes require explicit approval
- Zero data loss in tests

**Risk:** HIGH (data loss) - Extra testing required

---

### Phase 4: Query Builder (3 weeks)

**Goal:** Type-safe query construction

**Deliverables:**
- [ ] Query builder API
- [ ] WHERE conditions
- [ ] ORDER BY / LIMIT / OFFSET
- [ ] Aggregations (COUNT, SUM, AVG, etc.)
- [ ] Scope system
- [ ] SQL generation
- [ ] Parameter binding (SQL injection prevention)

**Team:** 1-2 engineers

**Exit Criteria:**
- All query patterns work
- SQL injection impossible
- Scopes apply correctly
- Performance is good

---

### Phase 5: Relationship Loading (5 weeks)

**Goal:** Efficient relationship loading with N+1 prevention

**Deliverables:**
- [ ] Belongs-to loading
- [ ] Has-many loading
- [ ] Has-many-through loading
- [ ] Eager loading (`.Includes()`)
- [ ] Lazy loading
- [ ] N+1 detection
- [ ] Circular reference prevention
- [ ] Depth limiting

**Team:** 2 engineers

**Exit Criteria:**
- All relationship types work
- Eager loading prevents N+1
- Circular references handled gracefully
- Performance is excellent

**Risk:** HIGH (performance, complexity)

---

### Phase 6: CRUD Operations (2 weeks)

**Goal:** Auto-generated CRUD methods

**Deliverables:**
- [ ] Create operation
- [ ] Read operations (find, find_by, where)
- [ ] Update operation
- [ ] Delete operation
- [ ] Bulk operations
- [ ] Atomic increment/decrement

**Team:** 1 engineer

**Exit Criteria:**
- All CRUD operations work
- Atomic operations are atomic
- Error handling is robust

---

### Phase 7: Validation Engine (3 weeks)

**Goal:** Execute validations

**Deliverables:**
- [ ] Field-level constraints
- [ ] Resource-level constraints
- [ ] Procedural validation
- [ ] Invariants
- [ ] Error collection
- [ ] Custom error messages

**Team:** 1-2 engineers

**Exit Criteria:**
- All validation types work
- Error messages are clear
- Validation runs at correct time

---

### Phase 8: Change Tracking (1 week)

**Goal:** Track field changes for update hooks

**Deliverables:**
- [ ] Change tracker implementation
- [ ] `field_changed?` methods
- [ ] `field_changed_to?` methods
- [ ] `field_changed_from?` methods
- [ ] `previous_value` accessor
- [ ] Integration with update operation

**Team:** 1 engineer

**Exit Criteria:**
- All change tracking methods work
- Correctly detects changes
- Works with nested fields

---

### Phase 9: Lifecycle Hooks (4 weeks)

**Goal:** Execute hooks with transaction support

**Deliverables:**
- [ ] Hook executor
- [ ] Transaction support
- [ ] Async operation support
- [ ] Error handling (rescue blocks)
- [ ] Hook execution order
- [ ] Integration with CRUD

**Team:** 2 engineers

**Exit Criteria:**
- All hook types work
- Transactions work correctly
- Async operations don't block
- Error handling works

**Risk:** MEDIUM (complexity, transactions)

---

### Phase 10: Transaction Management (2 weeks)

**Goal:** Robust transaction support

**Deliverables:**
- [ ] Transaction manager
- [ ] Nested transactions (savepoints)
- [ ] Isolation level support
- [ ] Deadlock retry logic
- [ ] Transaction timeout
- [ ] Explicit locking (SELECT FOR UPDATE)

**Team:** 1 engineer

**Exit Criteria:**
- Transactions work reliably
- Nested transactions work
- Deadlock handling works
- Timeout works

**Risk:** HIGH (data consistency)

---

## Testing Strategy

### Testing Pyramid

```
        ┌─────────────────┐
        │   E2E Tests     │   10% - Full application flow
        │    (10%)        │
        ├─────────────────┤
        │ Integration     │   30% - Component interactions
        │   Tests (30%)   │
        ├─────────────────┤
        │   Unit Tests    │   60% - Individual functions
        │    (60%)        │
        └─────────────────┘
```

### Unit Tests (60%)

**Coverage:** Individual functions, methods

**Examples:**
- Type validation
- Constraint checking
- SQL generation
- Query building
- Change tracking

**Tools:**
- Go testing package
- Table-driven tests
- Mock database

**Target:** >90% code coverage

---

### Integration Tests (30%)

**Coverage:** Component interactions

**Examples:**
- Resource → Schema → Database
- Query builder → SQL → Results
- CRUD → Hooks → Validation
- Relationship loading → SQL → Objects

**Tools:**
- Real test database (PostgreSQL, SQLite)
- Fixtures
- Database cleanup between tests

**Target:** All major flows covered

---

### End-to-End Tests (10%)

**Coverage:** Full application scenarios

**Examples:**
- Create blog post with author and comments
- Update post, verify revision created
- Delete post, verify cascade deletion
- Complex query with eager loading

**Tools:**
- Test database
- Seed data
- Full resource definitions

**Target:** Common use cases covered

---

### Performance Tests

**Coverage:** Performance-critical paths

**Examples:**
- Query performance (10,000+ records)
- N+1 query detection
- Eager loading vs lazy loading
- Hook execution time
- Transaction throughput

**Tools:**
- Benchmarking framework
- Profiling tools
- Load testing

**Target:**
- Queries <50ms for 10K records
- Eager loading prevents N+1
- Hooks <10ms overhead

---

### Security Tests

**Coverage:** Security vulnerabilities

**Examples:**
- SQL injection attempts
- Parameter binding edge cases
- Mass assignment prevention
- Sensitive field protection

**Tools:**
- Security testing framework
- Fuzzing
- Static analysis

**Target:** Zero vulnerabilities

---

## Integration Points

### 1. Parser Integration

**Interface:**
```go
type Parser interface {
    ParseResource(source string) (*ResourceSchema, error)
    ParseExpression(source string) (*AST, error)
}
```

**Responsibility:**
- Parser provides AST
- ORM validates semantic correctness
- ORM generates SQL from schema

---

### 2. Expression Interpreter Integration

**Interface:**
```go
type Interpreter interface {
    Execute(ast *AST, ctx *ExecutionContext) (*ExecutionResult, error)
    Evaluate(ast *AST, ctx *ExecutionContext) (interface{}, error)
}
```

**Responsibility:**
- Interpreter executes hook code
- ORM provides execution context
- ORM handles transaction boundaries

---

### 3. Runtime API Integration

**Interface:**
```go
type RuntimeAPI interface {
    GetResource(name string) *CRUDOperations
    ListResources() []string
    GetSchema(name string) *ResourceSchema
}
```

**Responsibility:**
- Runtime provides access to resources
- ORM exposes CRUD operations
- Introspection queries schema

---

### 4. Web Framework Integration

**Interface:**
```go
type HTTPHandler interface {
    HandleCreate(w http.ResponseWriter, r *http.Request)
    HandleRead(w http.ResponseWriter, r *http.Request)
    HandleUpdate(w http.ResponseWriter, r *http.Request)
    HandleDelete(w http.ResponseWriter, r *http.Request)
    HandleList(w http.ResponseWriter, r *http.Request)
}
```

**Responsibility:**
- Web framework calls CRUD operations
- ORM returns JSON-serializable data
- ORM handles validation errors

---

## Performance Targets

### Compilation

- **Parse Speed:** < 10ms per resource file
- **Validation Speed:** < 50ms for complex resources
- **Schema Generation:** < 100ms per resource
- **Migration Generation:** < 200ms for typical changes

### Runtime

- **Query Performance:** < 50ms for 10,000 records
- **Hook Execution:** < 10ms overhead per hook
- **Validation:** < 5ms per record
- **Transaction Overhead:** < 1ms

### Memory

- **Schema Registry:** < 10MB per 100 resources
- **Query Builder:** < 1KB per query
- **Result Sets:** < 100MB for 10,000 records

### Concurrency

- **Throughput:** 10,000+ req/s per resource
- **Connections:** 100+ concurrent database connections
- **Transactions:** < 100ms average duration

---

## Risk Mitigation

### Risk 1: N+1 Query Problem

**Impact:** HIGH - Performance degradation
**Probability:** HIGH

**Mitigation:**
1. Always eager load by default in loops
2. Development-mode query counter
3. Static analysis for N+1 patterns
4. Clear documentation

**Status:** CRITICAL - Phase 5

---

### Risk 2: Migration Data Loss

**Impact:** CRITICAL - Permanent data loss
**Probability:** MEDIUM

**Mitigation:**
1. Dry-run mode
2. Backup requirement
3. Staged rollout
4. Manual review for breaking changes
5. Rollback support

**Status:** CRITICAL - Phase 3

---

### Risk 3: Transaction Deadlocks

**Impact:** HIGH - Failed requests
**Probability:** MEDIUM

**Mitigation:**
1. Transaction timeout
2. Deadlock retry logic
3. Lock ordering documentation
4. Async operations

**Status:** MEDIUM - Phase 10

---

### Risk 4: Circular Relationships

**Impact:** HIGH - Stack overflow
**Probability:** MEDIUM

**Mitigation:**
1. Depth limiting
2. Cycle detection
3. Lazy loading default
4. Explicit depth control

**Status:** HIGH - Phase 5

---

### Risk 5: Hook Performance

**Impact:** MEDIUM - Request latency
**Probability:** HIGH

**Mitigation:**
1. Performance monitoring
2. Async by default guidance
3. Hook profiling
4. Optimization suggestions

**Status:** MEDIUM - Phase 9

---

## Success Criteria

### Functional

- [ ] Parse all v2.0 resource syntax
- [ ] Generate correct SQL DDL for PostgreSQL
- [ ] Execute CRUD operations with validation
- [ ] Load relationships with N+1 prevention
- [ ] Execute lifecycle hooks with transactions
- [ ] Track changes for update hooks
- [ ] Generate safe migrations
- [ ] Support rollback

### Performance

- [ ] Compile < 1s for 100 resources
- [ ] Query < 50ms for 10K records
- [ ] Handle 10K req/s per resource
- [ ] Zero N+1 queries with eager loading

### Quality

- [ ] >90% code coverage
- [ ] Zero SQL injection vulnerabilities
- [ ] 95%+ LLM success rate
- [ ] Clear error messages

### Developer Experience

- [ ] < 2 minutes to understand any resource
- [ ] All behavior visible in resource file
- [ ] Zero "magic" surprises
- [ ] Excellent documentation

---

## Conclusion

The Resource System & ORM is the most complex component of Conduit, requiring careful design and implementation across 10 major components over 31-35 weeks.

**Key Success Factors:**
1. Explicit nullability prevents nil errors
2. N+1 query prevention built-in
3. Migration safety prevents data loss
4. Transaction boundaries are explicit
5. Comprehensive testing at all levels

**Timeline:**
- **Conservative:** 40 weeks (200 person-days)
- **Aggressive:** 31 weeks (155 person-days)
- **Recommended:** 35 weeks (175 person-days)

**Team Size:** 2-3 engineers

**Dependencies:**
- Parser (for resource syntax)
- Expression Interpreter (for hooks, validations)
- Runtime (for introspection, API)

---

**Document Status:** COMPLETE
**Date:** 2025-10-13
**Next Steps:** Review, approve, and begin Phase 0 implementation
