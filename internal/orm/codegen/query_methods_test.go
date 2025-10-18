package codegen

import (
	"strings"
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func createTestResource() *schema.ResourceSchema {
	resource := schema.NewResourceSchema("Post")

	// Add fields
	resource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeUUID,
			Nullable: false,
		},
		Annotations: []schema.Annotation{
			{Name: "primary"},
			{Name: "auto"},
		},
	}

	resource.Fields["title"] = &schema.Field{
		Name: "title",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
		},
	}

	resource.Fields["content"] = &schema.Field{
		Name: "content",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeText,
			Nullable: true,
		},
	}

	resource.Fields["views"] = &schema.Field{
		Name: "views",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeInt,
			Nullable: false,
		},
	}

	resource.Fields["published_at"] = &schema.Field{
		Name: "published_at",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeTimestamp,
			Nullable: true,
		},
	}

	// Add relationship
	resource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
		ForeignKey:     "author_id",
	}

	// Add scope
	resource.Scopes["published"] = &schema.Scope{
		Name:      "published",
		Arguments: []*schema.ScopeArgument{},
		OrderBy:   "published_at DESC",
	}

	return resource
}

func TestNewQueryMethodGenerator(t *testing.T) {
	schemas := map[string]*schema.ResourceSchema{
		"Post": createTestResource(),
	}

	gen := NewQueryMethodGenerator(schemas)

	if gen == nil {
		t.Fatal("NewQueryMethodGenerator returned nil")
	}

	if gen.schemas == nil {
		t.Error("Schemas not set")
	}
}

func TestGenerate(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	gen := NewQueryMethodGenerator(schemas)

	code, err := gen.Generate(resource)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if code == "" {
		t.Fatal("Generated code is empty")
	}

	// Check that essential components are present
	expectedParts := []string{
		"type PostQuery struct",
		"func (r *Post) Query",
		"func (q *PostQuery) WhereTitle",
		"func (q *PostQuery) OrderByTitle",
		"func (q *PostQuery) All",
		"func (q *PostQuery) First",
		"func (q *PostQuery) Count",
	}

	for _, part := range expectedParts {
		if !strings.Contains(code, part) {
			t.Errorf("Generated code missing: %s", part)
		}
	}
}

func TestGenerateQueryBuilderStruct(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	gen := NewQueryMethodGenerator(schemas)

	code := gen.generateQueryBuilderStruct(resource)

	if !strings.Contains(code, "type PostQuery struct") {
		t.Error("Missing struct declaration")
	}

	if !strings.Contains(code, "*query.QueryBuilder") {
		t.Error("Missing QueryBuilder field")
	}
}

func TestGenerateConstructor(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	gen := NewQueryMethodGenerator(schemas)

	code := gen.generateConstructor(resource)

	if !strings.Contains(code, "func (r *Post) Query") {
		t.Error("Missing constructor function")
	}

	if !strings.Contains(code, "return &PostQuery") {
		t.Error("Missing return statement")
	}
}

func TestGenerateWhereMethod(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	gen := NewQueryMethodGenerator(schemas)

	field := resource.Fields["title"]
	code := gen.generateWhereMethod(resource, "title", field)

	if !strings.Contains(code, "func (q *PostQuery) WhereTitle") {
		t.Error("Missing WhereTitle method")
	}

	if !strings.Contains(code, "query.Operator") {
		t.Error("Missing Operator parameter")
	}

	// Non-nullable field should have Eq method
	if !strings.Contains(code, "WhereTitleEq") {
		t.Error("Missing convenience Eq method")
	}

	// Text field should have Like methods
	if !strings.Contains(code, "WhereTitleLike") {
		t.Error("Missing Like method")
	}

	if !strings.Contains(code, "WhereTitleILike") {
		t.Error("Missing ILike method")
	}
}

func TestGenerateWhereMethod_NumericField(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	gen := NewQueryMethodGenerator(schemas)

	field := resource.Fields["views"]
	code := gen.generateWhereMethod(resource, "views", field)

	// Numeric field should have range methods
	if !strings.Contains(code, "WhereViewsGt") {
		t.Error("Missing greater-than method")
	}

	if !strings.Contains(code, "WhereViewsLt") {
		t.Error("Missing less-than method")
	}
}

func TestGenerateWhereMethod_NullableField(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	gen := NewQueryMethodGenerator(schemas)

	field := resource.Fields["published_at"]
	code := gen.generateWhereMethod(resource, "published_at", field)

	// Nullable field should have null check methods
	if !strings.Contains(code, "WherePublishedAtNull") {
		t.Error("Missing null check method")
	}

	if !strings.Contains(code, "WherePublishedAtNotNull") {
		t.Error("Missing not-null check method")
	}
}

func TestGenerateOrderByMethod(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	gen := NewQueryMethodGenerator(schemas)

	code := gen.generateOrderByMethod(resource, "title")

	if !strings.Contains(code, "func (q *PostQuery) OrderByTitle") {
		t.Error("Missing OrderByTitle method")
	}

	if !strings.Contains(code, "OrderByTitleAsc") {
		t.Error("Missing ascending convenience method")
	}

	if !strings.Contains(code, "OrderByTitleDesc") {
		t.Error("Missing descending convenience method")
	}
}

func TestGenerateRelationshipMethod_BelongsTo(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	gen := NewQueryMethodGenerator(schemas)

	rel := resource.Relationships["author"]
	code := gen.generateRelationshipMethod(resource, "author", rel)

	if !strings.Contains(code, "func (q *PostQuery) JoinAuthor") {
		t.Error("Missing JoinAuthor method")
	}

	if !strings.Contains(code, "func (q *PostQuery) IncludeAuthor") {
		t.Error("Missing IncludeAuthor method")
	}

	// Should generate JOIN with proper condition
	if !strings.Contains(code, "InnerJoin") {
		t.Error("Missing InnerJoin call")
	}
}

func TestGenerateScopeMethod(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	gen := NewQueryMethodGenerator(schemas)

	scopeDef := resource.Scopes["published"]
	code := gen.generateScopeMethod(resource, "published", scopeDef)

	if !strings.Contains(code, "func (q *PostQuery) Published") {
		t.Error("Missing Published method")
	}

	if !strings.Contains(code, "q.builder.Scope") {
		t.Error("Missing Scope call")
	}
}

func TestGenerateScopeMethod_WithArguments(t *testing.T) {
	resource := createTestResource()

	// Add scope with arguments
	resource.Scopes["since"] = &schema.Scope{
		Name: "since",
		Arguments: []*schema.ScopeArgument{
			{
				Name: "date",
				Type: &schema.TypeSpec{
					BaseType: schema.TypeTimestamp,
					Nullable: false,
				},
			},
		},
	}

	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	gen := NewQueryMethodGenerator(schemas)

	scopeDef := resource.Scopes["since"]
	code := gen.generateScopeMethod(resource, "since", scopeDef)

	// Should have parameter
	if !strings.Contains(code, "date time.Time") {
		t.Error("Missing scope parameter")
	}

	// Should pass argument to Scope call
	if !strings.Contains(code, ", date)") {
		t.Error("Missing argument in Scope call")
	}
}

func TestGenerateTerminalMethods(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	gen := NewQueryMethodGenerator(schemas)

	code := gen.generateTerminalMethods(resource)

	expectedMethods := []string{
		"func (q *PostQuery) All",
		"func (q *PostQuery) First",
		"func (q *PostQuery) Count",
		"func (q *PostQuery) Exists",
		"func (q *PostQuery) Limit",
		"func (q *PostQuery) Offset",
	}

	for _, method := range expectedMethods {
		if !strings.Contains(code, method) {
			t.Errorf("Missing terminal method: %s", method)
		}
	}
}

func TestMapTypeToGo(t *testing.T) {
	gen := NewQueryMethodGenerator(nil)

	tests := []struct {
		name     string
		typeSpec *schema.TypeSpec
		expected string
	}{
		{
			name: "String",
			typeSpec: &schema.TypeSpec{
				BaseType: schema.TypeString,
				Nullable: false,
			},
			expected: "string",
		},
		{
			name: "Nullable String",
			typeSpec: &schema.TypeSpec{
				BaseType: schema.TypeString,
				Nullable: true,
			},
			expected: "*string",
		},
		{
			name: "Int",
			typeSpec: &schema.TypeSpec{
				BaseType: schema.TypeInt,
				Nullable: false,
			},
			expected: "int",
		},
		{
			name: "BigInt",
			typeSpec: &schema.TypeSpec{
				BaseType: schema.TypeBigInt,
				Nullable: false,
			},
			expected: "int64",
		},
		{
			name: "Float",
			typeSpec: &schema.TypeSpec{
				BaseType: schema.TypeFloat,
				Nullable: false,
			},
			expected: "float64",
		},
		{
			name: "Bool",
			typeSpec: &schema.TypeSpec{
				BaseType: schema.TypeBool,
				Nullable: false,
			},
			expected: "bool",
		},
		{
			name: "Timestamp",
			typeSpec: &schema.TypeSpec{
				BaseType: schema.TypeTimestamp,
				Nullable: false,
			},
			expected: "time.Time",
		},
		{
			name: "UUID",
			typeSpec: &schema.TypeSpec{
				BaseType: schema.TypeUUID,
				Nullable: false,
			},
			expected: "uuid.UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gen.mapTypeToGo(tt.typeSpec)
			if result != tt.expected {
				t.Errorf("mapTypeToGo() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"title", "Title"},
		{"published_at", "PublishedAt"},
		{"author_id", "AuthorId"},
		{"first_name", "FirstName"},
		{"id", "Id"},
	}

	for _, tt := range tests {
		result := toPascalCase(tt.input)
		if result != tt.expected {
			t.Errorf("toPascalCase(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

// Benchmark tests
func BenchmarkGenerate(b *testing.B) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	gen := NewQueryMethodGenerator(schemas)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.Generate(resource)
	}
}

func BenchmarkGenerateWhereMethod(b *testing.B) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	gen := NewQueryMethodGenerator(schemas)
	field := resource.Fields["title"]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gen.generateWhereMethod(resource, "title", field)
	}
}

func BenchmarkMapTypeToGo(b *testing.B) {
	gen := NewQueryMethodGenerator(nil)
	typeSpec := &schema.TypeSpec{
		BaseType: schema.TypeString,
		Nullable: false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gen.mapTypeToGo(typeSpec)
	}
}
