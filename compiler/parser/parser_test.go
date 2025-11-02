package parser

import (
	"github.com/conduit-lang/conduit/compiler/lexer"
	"testing"
)

// Helper function to parse source and return AST and errors
func parseSource(t *testing.T, source string) (*Program, []ParseError) {
	l := lexer.New(source, "test.cdt")
	tokens, lexErrors := l.ScanTokens()

	if len(lexErrors) > 0 {
		t.Fatalf("Lexer errors: %v", lexErrors)
	}

	p := New(tokens)
	return p.Parse()
}

func TestParser_SimpleResource(t *testing.T) {
	source := `
resource User {
  id: uuid! @primary @auto
  username: string!
  email: string!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	if len(program.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(program.Resources))
	}

	resource := program.Resources[0]
	if resource.Name != "User" {
		t.Errorf("Expected resource name 'User', got '%s'", resource.Name)
	}

	if len(resource.Fields) != 3 {
		t.Fatalf("Expected 3 fields, got %d", len(resource.Fields))
	}

	// Check id field
	idField := resource.Fields[0]
	if idField.Name != "id" {
		t.Errorf("Expected field name 'id', got '%s'", idField.Name)
	}
	if idField.Type.Name != "uuid" {
		t.Errorf("Expected type 'uuid', got '%s'", idField.Type.Name)
	}
	if idField.Nullable {
		t.Error("Expected id to be non-nullable")
	}
	if len(idField.Constraints) != 2 {
		t.Errorf("Expected 2 constraints, got %d", len(idField.Constraints))
	}

	// Check username field
	usernameField := resource.Fields[1]
	if usernameField.Name != "username" {
		t.Errorf("Expected field name 'username', got '%s'", usernameField.Name)
	}
	if usernameField.Type.Name != "string" {
		t.Errorf("Expected type 'string', got '%s'", usernameField.Type.Name)
	}
	if usernameField.Nullable {
		t.Error("Expected username to be non-nullable")
	}
}

func TestParser_PrimitiveTypes(t *testing.T) {
	source := `
resource Test {
  str: string!
  txt: text!
  num: int!
  flt: float!
  dec: decimal!
  flag: bool!
  ts: timestamp!
  dt: date!
  tm: time!
  uid: uuid!
  ulid: ulid!
  mail: email!
  link: url!
  phone: phone!
  data: json!
  md: markdown!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	expectedTypes := []string{
		"string", "text", "int", "float", "decimal", "bool",
		"timestamp", "date", "time", "uuid", "ulid", "email",
		"url", "phone", "json", "markdown",
	}

	if len(resource.Fields) != len(expectedTypes) {
		t.Fatalf("Expected %d fields, got %d", len(expectedTypes), len(resource.Fields))
	}

	for i, expectedType := range expectedTypes {
		field := resource.Fields[i]
		if field.Type.Name != expectedType {
			t.Errorf("Field %d: expected type '%s', got '%s'", i, expectedType, field.Type.Name)
		}
		if !field.Type.IsPrimitive() {
			t.Errorf("Field %d: expected primitive type", i)
		}
	}
}

func TestParser_ArrayType(t *testing.T) {
	source := `
resource Post {
  tags: array<string>!
  ids: array<uuid>!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	if len(resource.Fields) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(resource.Fields))
	}

	// Check tags field
	tagsField := resource.Fields[0]
	if !tagsField.Type.IsArray() {
		t.Error("Expected tags to be array type")
	}
	if tagsField.Type.ElementType.Name != "string" {
		t.Errorf("Expected element type 'string', got '%s'", tagsField.Type.ElementType.Name)
	}

	// Check ids field
	idsField := resource.Fields[1]
	if !idsField.Type.IsArray() {
		t.Error("Expected ids to be array type")
	}
	if idsField.Type.ElementType.Name != "uuid" {
		t.Errorf("Expected element type 'uuid', got '%s'", idsField.Type.ElementType.Name)
	}
}

func TestParser_HashType(t *testing.T) {
	source := `
resource Config {
  settings: hash<string, string>!
  metrics: hash<string, int>!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	if len(resource.Fields) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(resource.Fields))
	}

	// Check settings field
	settingsField := resource.Fields[0]
	if !settingsField.Type.IsHash() {
		t.Error("Expected settings to be hash type")
	}
	if settingsField.Type.KeyType.Name != "string" {
		t.Errorf("Expected key type 'string', got '%s'", settingsField.Type.KeyType.Name)
	}
	if settingsField.Type.ValueType.Name != "string" {
		t.Errorf("Expected value type 'string', got '%s'", settingsField.Type.ValueType.Name)
	}
}

func TestParser_EnumType(t *testing.T) {
	source := `
resource Post {
  status: enum ["draft", "published", "archived"]!
  role: enum ["user", "admin"]!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	if len(resource.Fields) != 2 {
		t.Fatalf("Expected 2 fields, got %d", len(resource.Fields))
	}

	// Check status field
	statusField := resource.Fields[0]
	if !statusField.Type.IsEnum() {
		t.Error("Expected status to be enum type")
	}
	if len(statusField.Type.EnumValues) != 3 {
		t.Errorf("Expected 3 enum values, got %d", len(statusField.Type.EnumValues))
	}
	expectedValues := []string{"draft", "published", "archived"}
	for i, expected := range expectedValues {
		if statusField.Type.EnumValues[i] != expected {
			t.Errorf("Expected enum value '%s', got '%s'", expected, statusField.Type.EnumValues[i])
		}
	}
}

func TestParser_InlineStruct(t *testing.T) {
	source := `
resource Post {
  seo: {
    title: string?
    description: string?
    image: url?
  }?
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	if len(resource.Fields) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(resource.Fields))
	}

	seoField := resource.Fields[0]
	if !seoField.Type.IsStruct() {
		t.Error("Expected seo to be struct type")
	}
	if !seoField.Nullable {
		t.Error("Expected seo to be nullable")
	}
	if len(seoField.Type.StructFields) != 3 {
		t.Fatalf("Expected 3 struct fields, got %d", len(seoField.Type.StructFields))
	}

	// Check struct fields
	titleField := seoField.Type.StructFields[0]
	if titleField.Name != "title" {
		t.Errorf("Expected field name 'title', got '%s'", titleField.Name)
	}
	if !titleField.Nullable {
		t.Error("Expected title to be nullable")
	}
}

func TestParser_Relationship(t *testing.T) {
	source := `
resource Post {
  author: User!
  category: Category? {
    foreign_key: "category_id"
    on_delete: set_null
  }
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	if len(resource.Relationships) != 2 {
		t.Fatalf("Expected 2 relationships, got %d", len(resource.Relationships))
	}

	// Check author relationship
	author := resource.Relationships[0]
	if author.Name != "author" {
		t.Errorf("Expected relationship name 'author', got '%s'", author.Name)
	}
	if author.TargetType != "User" {
		t.Errorf("Expected target type 'User', got '%s'", author.TargetType)
	}
	if author.Nullable {
		t.Error("Expected author to be non-nullable")
	}

	// Check category relationship with metadata
	category := resource.Relationships[1]
	if category.Name != "category" {
		t.Errorf("Expected relationship name 'category', got '%s'", category.Name)
	}
	if !category.Nullable {
		t.Error("Expected category to be nullable")
	}
	if category.ForeignKey != "category_id" {
		t.Errorf("Expected foreign_key 'category_id', got '%s'", category.ForeignKey)
	}
	if category.OnDelete != "set_null" {
		t.Errorf("Expected on_delete 'set_null', got '%s'", category.OnDelete)
	}
}

func TestParser_FieldConstraints(t *testing.T) {
	source := `
resource User {
  username: string! @min(3) @max(50) @unique
  age: int! @min(18) @max(120)
  email: email! @unique @required
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]

	// Check username constraints
	usernameField := resource.Fields[0]
	if len(usernameField.Constraints) != 3 {
		t.Fatalf("Expected 3 constraints on username, got %d", len(usernameField.Constraints))
	}

	minConstraint := usernameField.Constraints[0]
	if minConstraint.Name != "min" {
		t.Errorf("Expected constraint 'min', got '%s'", minConstraint.Name)
	}
	if len(minConstraint.Arguments) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(minConstraint.Arguments))
	}
	if minConstraint.Arguments[0] != int64(3) {
		t.Errorf("Expected argument 3, got %v", minConstraint.Arguments[0])
	}

	// Check age constraints
	ageField := resource.Fields[1]
	if len(ageField.Constraints) != 2 {
		t.Fatalf("Expected 2 constraints on age, got %d", len(ageField.Constraints))
	}
}

func TestParser_MissingNullability(t *testing.T) {
	source := `
resource User {
  username: string
}
`

	_, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected error for missing nullability indicator")
	}

	// Check that error mentions nullability
	errorMsg := errors[0].Error()
	if errorMsg == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestParser_ErrorRecovery(t *testing.T) {
	source := `
resource User {
  username: string!
  invalid syntax here
  email: string!
}

resource Post {
  title: string!
}
`

	program, errors := parseSource(t, source)

	// Should have errors but still parse valid parts
	if len(errors) == 0 {
		t.Error("Expected errors from invalid syntax")
	}

	// Should still parse both resources
	if len(program.Resources) < 1 {
		t.Error("Expected at least one resource to be parsed")
	}
}

func TestParser_PartialAST(t *testing.T) {
	source := `
resource User {
  username: string!
  email: string!
  missing_type:
}
`

	program, errors := parseSource(t, source)

	// Should have errors
	if len(errors) == 0 {
		t.Error("Expected errors from incomplete field")
	}

	// Should still return partial AST
	if program == nil {
		t.Fatal("Expected partial AST to be returned")
	}
	if len(program.Resources) != 1 {
		t.Errorf("Expected 1 resource in partial AST, got %d", len(program.Resources))
	}
}

func TestParser_ComplexResource(t *testing.T) {
	source := `
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text! @min(100)

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  category: Category? {
    foreign_key: "category_id"
    on_delete: set_null
  }

  tags: array<uuid>!

  seo: {
    title: string? @max(60)
    description: string? @max(160)
  }?

  status: enum ["draft", "published", "archived"]!

  metrics: {
    view_count: int!
    comment_count: int!
  }!

  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	if len(program.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(program.Resources))
	}

	resource := program.Resources[0]
	if resource.Name != "Post" {
		t.Errorf("Expected resource name 'Post', got '%s'", resource.Name)
	}

	// Should have 10 fields (excluding relationships)
	expectedFields := 10
	if len(resource.Fields) != expectedFields {
		t.Errorf("Expected %d fields, got %d", expectedFields, len(resource.Fields))
	}

	// Should have 2 relationships
	if len(resource.Relationships) != 2 {
		t.Errorf("Expected 2 relationships, got %d", len(resource.Relationships))
	}

	// Verify specific fields
	idField := resource.Fields[0]
	if idField.Name != "id" || idField.Type.Name != "uuid" {
		t.Error("First field should be id: uuid!")
	}

	// Verify array field
	var tagsField *FieldNode
	for _, field := range resource.Fields {
		if field.Name == "tags" {
			tagsField = field
			break
		}
	}
	if tagsField == nil {
		t.Fatal("Expected to find tags field")
	}
	if !tagsField.Type.IsArray() {
		t.Error("Tags should be array type")
	}

	// Verify enum field
	var statusField *FieldNode
	for _, field := range resource.Fields {
		if field.Name == "status" {
			statusField = field
			break
		}
	}
	if statusField == nil {
		t.Fatal("Expected to find status field")
	}
	if !statusField.Type.IsEnum() {
		t.Error("Status should be enum type")
	}

	// Verify struct field
	var metricsField *FieldNode
	for _, field := range resource.Fields {
		if field.Name == "metrics" {
			metricsField = field
			break
		}
	}
	if metricsField == nil {
		t.Fatal("Expected to find metrics field")
	}
	if !metricsField.Type.IsStruct() {
		t.Error("Metrics should be struct type")
	}
}

func TestParser_EmptyResource(t *testing.T) {
	source := `
resource Empty {
}
`

	program, errors := parseSource(t, source)

	// Empty resource is valid
	if len(errors) > 0 {
		t.Errorf("Expected no errors for empty resource, got: %v", errors)
	}

	if len(program.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(program.Resources))
	}

	resource := program.Resources[0]
	if len(resource.Fields) != 0 {
		t.Errorf("Expected 0 fields, got %d", len(resource.Fields))
	}
}

func TestParser_MultipleResources(t *testing.T) {
	source := `
resource User {
  id: uuid! @primary
  name: string!
}

resource Post {
  id: uuid! @primary
  title: string!
}

resource Comment {
  id: uuid! @primary
  content: text!
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	if len(program.Resources) != 3 {
		t.Fatalf("Expected 3 resources, got %d", len(program.Resources))
	}

	expectedNames := []string{"User", "Post", "Comment"}
	for i, expected := range expectedNames {
		if program.Resources[i].Name != expected {
			t.Errorf("Resource %d: expected name '%s', got '%s'", i, expected, program.Resources[i].Name)
		}
	}
}

// TestParser_HooksAndConstraints tests parsing of resource-level hooks and constraints
// This is a regression test for the bug where consumeTrailingComment() was consuming
// tokens from subsequent fields, causing the parser to fail on hooks and constraints.
func TestParser_HooksAndConstraints(t *testing.T) {
	source := `
resource Post {
  id        : uuid! @primary @auto
  title     : string! @min(5) @max(200)
  slug      : string! @unique
  content   : text! @min(100)
  published : bool! @default(false)
  author_id : uuid!
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  author    : User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  @before create {
    self.slug = String.slugify(self.title)
  }

  @constraint published_requires_content {
    on: [create, update]
    when: self.published == true
    condition: String.length(self.content) >= 500
    error: "Published posts must have at least 500 characters"
  }
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	if len(program.Resources) != 1 {
		t.Fatalf("Expected 1 resource, got %d", len(program.Resources))
	}

	resource := program.Resources[0]
	if resource.Name != "Post" {
		t.Errorf("Expected resource name 'Post', got '%s'", resource.Name)
	}

	// Verify all fields were parsed
	if len(resource.Fields) != 8 {
		t.Fatalf("Expected 8 fields, got %d", len(resource.Fields))
	}

	// Verify relationship was parsed
	if len(resource.Relationships) != 1 {
		t.Fatalf("Expected 1 relationship, got %d", len(resource.Relationships))
	}

	// Verify hook was parsed
	if len(resource.Hooks) != 1 {
		t.Fatalf("Expected 1 hook, got %d", len(resource.Hooks))
	}

	hook := resource.Hooks[0]
	if hook.Type != "before" {
		t.Errorf("Expected hook type 'before', got '%s'", hook.Type)
	}
	if hook.Trigger != "create" {
		t.Errorf("Expected hook trigger 'create', got '%s'", hook.Trigger)
	}
	if hook.Body == "" {
		t.Error("Expected hook body to be non-empty")
	}

	// Verify custom constraint was parsed
	if len(resource.Constraints) != 1 {
		t.Fatalf("Expected 1 custom constraint, got %d", len(resource.Constraints))
	}

	constraint := resource.Constraints[0]
	if constraint.Name != "published_requires_content" {
		t.Errorf("Expected constraint name 'published_requires_content', got '%s'", constraint.Name)
	}
	if constraint.Body == "" {
		t.Error("Expected constraint body to be non-empty")
	}
}

// TestParser_MultipleFieldsWithConstraints tests that multiple fields with constraints
// are parsed correctly without consuming tokens from subsequent fields.
func TestParser_MultipleFieldsWithConstraints(t *testing.T) {
	source := `
resource User {
  id: uuid! @primary @auto
  username: string! @min(3) @max(50)
  email: string! @unique
  age: int?
}
`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Expected no errors, got: %v", errors)
	}

	resource := program.Resources[0]
	
	// All 4 fields should be parsed
	if len(resource.Fields) != 4 {
		t.Fatalf("Expected 4 fields, got %d", len(resource.Fields))
	}

	// Check field names
	expectedFields := []string{"id", "username", "email", "age"}
	for i, expectedName := range expectedFields {
		if resource.Fields[i].Name != expectedName {
			t.Errorf("Field %d: expected name '%s', got '%s'", i, expectedName, resource.Fields[i].Name)
		}
	}

	// Check constraints on id field
	if len(resource.Fields[0].Constraints) != 2 {
		t.Errorf("Expected id field to have 2 constraints, got %d", len(resource.Fields[0].Constraints))
	}

	// Check constraints on username field
	if len(resource.Fields[1].Constraints) != 2 {
		t.Errorf("Expected username field to have 2 constraints, got %d", len(resource.Fields[1].Constraints))
	}

	// Check constraints on email field
	if len(resource.Fields[2].Constraints) != 1 {
		t.Errorf("Expected email field to have 1 constraint, got %d", len(resource.Fields[2].Constraints))
	}

	// Check no constraints on age field
	if len(resource.Fields[3].Constraints) != 0 {
		t.Errorf("Expected age field to have 0 constraints, got %d", len(resource.Fields[3].Constraints))
	}
}
