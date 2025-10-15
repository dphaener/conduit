package parser

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/compiler/lexer"
)

// Helper function to create a parser from source code
func parseSource(t *testing.T, source string) (*ast.Program, []ParseError) {
	t.Helper()

	lex := lexer.New(source)
	tokens, lexErrors := lex.ScanTokens()

	if len(lexErrors) > 0 {
		t.Fatalf("Lexer errors: %v", lexErrors)
	}

	parser := New(tokens)
	return parser.Parse()
}

// TestParseSimpleResource tests parsing a basic resource
func TestParseSimpleResource(t *testing.T) {
	source := `resource User {
  id: uuid! @primary @auto
  username: string!
  email_address: string!
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
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

	// Check first field
	if resource.Fields[0].Name != "id" {
		t.Errorf("Expected field name 'id', got '%s'", resource.Fields[0].Name)
	}

	if resource.Fields[0].Type.Kind != ast.TypePrimitive {
		t.Errorf("Expected primitive type for id")
	}

	if resource.Fields[0].Type.Name != "uuid" {
		t.Errorf("Expected type 'uuid', got '%s'", resource.Fields[0].Type.Name)
	}

	if len(resource.Fields[0].Constraints) != 2 {
		t.Errorf("Expected 2 constraints for id, got %d", len(resource.Fields[0].Constraints))
	}
}

// TestParseResourceWithNullableFields tests nullable vs required fields
func TestParseResourceWithNullableFields(t *testing.T) {
	source := `resource Post {
  title: string!
  subtitle: string?
  content: text!
  excerpt: text?
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	resource := program.Resources[0]

	// title should not be nullable
	if resource.Fields[0].Nullable {
		t.Error("Expected title to be non-nullable")
	}

	// subtitle should be nullable
	if !resource.Fields[1].Nullable {
		t.Error("Expected subtitle to be nullable")
	}
}

// TestParseArrayType tests parsing array types
func TestParseArrayType(t *testing.T) {
	source := `resource Post {
  tags: array<string!>!
  categories: array<int!>?
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	resource := program.Resources[0]

	// Check tags field
	tagsField := resource.Fields[0]
	if tagsField.Type.Kind != ast.TypeArray {
		t.Error("Expected array type for tags")
	}

	if tagsField.Type.ElementType.Name != "string" {
		t.Errorf("Expected string element type, got %s", tagsField.Type.ElementType.Name)
	}

	// Check categories field
	categoriesField := resource.Fields[1]
	if categoriesField.Type.Kind != ast.TypeArray {
		t.Error("Expected array type for categories")
	}

	if !categoriesField.Nullable {
		t.Error("Expected categories to be nullable")
	}
}

// TestParseHashType tests parsing hash types
func TestParseHashType(t *testing.T) {
	source := `resource Config {
  settings: hash<string!, string!>!
  metadata: hash<string!, int!>?
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	resource := program.Resources[0]

	// Check settings field
	settingsField := resource.Fields[0]
	if settingsField.Type.Kind != ast.TypeHash {
		t.Error("Expected hash type for settings")
	}

	if settingsField.Type.KeyType.Name != "string" {
		t.Errorf("Expected string key type, got %s", settingsField.Type.KeyType.Name)
	}

	if settingsField.Type.ValueType.Name != "string" {
		t.Errorf("Expected string value type, got %s", settingsField.Type.ValueType.Name)
	}
}

// TestParseEnumType tests parsing enum types
func TestParseEnumType(t *testing.T) {
	source := `resource Post {
  status: enum ["draft", "published", "archived"]!
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	resource := program.Resources[0]
	statusField := resource.Fields[0]

	if statusField.Type.Kind != ast.TypeEnum {
		t.Error("Expected enum type for status")
	}

	expectedValues := []string{"draft", "published", "archived"}
	if len(statusField.Type.EnumValues) != len(expectedValues) {
		t.Fatalf("Expected %d enum values, got %d", len(expectedValues), len(statusField.Type.EnumValues))
	}

	for i, expected := range expectedValues {
		if statusField.Type.EnumValues[i] != expected {
			t.Errorf("Expected enum value '%s', got '%s'", expected, statusField.Type.EnumValues[i])
		}
	}
}

// TestParseResourceType tests parsing resource types (relationships)
func TestParseResourceType(t *testing.T) {
	source := `resource Post {
  author: User!
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	resource := program.Resources[0]
	authorField := resource.Fields[0]

	if authorField.Type.Kind != ast.TypeResource {
		t.Error("Expected resource type for author")
	}

	if authorField.Type.Name != "User" {
		t.Errorf("Expected resource type 'User', got '%s'", authorField.Type.Name)
	}
}

// TestParseFieldConstraints tests parsing field constraints
func TestParseFieldConstraints(t *testing.T) {
	source := `resource User {
  username: string! @min(3) @max(50) @unique
  age: int! @min(0) @max(150)
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	resource := program.Resources[0]

	// Check username constraints
	usernameField := resource.Fields[0]
	if len(usernameField.Constraints) != 3 {
		t.Fatalf("Expected 3 constraints for username, got %d", len(usernameField.Constraints))
	}

	// Check constraint names
	expectedConstraints := []string{"min", "max", "unique"}
	for i, constraint := range usernameField.Constraints {
		if constraint.Name != expectedConstraints[i] {
			t.Errorf("Expected constraint '%s', got '%s'", expectedConstraints[i], constraint.Name)
		}
	}

	// Check min constraint has argument
	minConstraint := usernameField.Constraints[0]
	if len(minConstraint.Arguments) != 1 {
		t.Errorf("Expected 1 argument for @min, got %d", len(minConstraint.Arguments))
	}
}

// TestParseLifecycleHook tests parsing lifecycle hooks
func TestParseLifecycleHook(t *testing.T) {
	source := `resource Post {
  title: string!

  @before create {
    self.slug = String.slugify(self.title)
  }
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	resource := program.Resources[0]

	if len(resource.Hooks) != 1 {
		t.Fatalf("Expected 1 hook, got %d", len(resource.Hooks))
	}

	hook := resource.Hooks[0]
	if hook.Timing != "before" {
		t.Errorf("Expected timing 'before', got '%s'", hook.Timing)
	}

	if hook.Event != "create" {
		t.Errorf("Expected event 'create', got '%s'", hook.Event)
	}

	if len(hook.Body) == 0 {
		t.Error("Expected non-empty hook body")
	}
}

// TestParseHookWithTransaction tests parsing hooks with @transaction
func TestParseHookWithTransaction(t *testing.T) {
	source := `resource Post {
  title: string!

  @after create @transaction {
    self.slug = String.slugify(self.title)
  }
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	hook := program.Resources[0].Hooks[0]

	if !hook.IsTransaction {
		t.Error("Expected hook to have @transaction modifier")
	}
}

// TestParseHookWithAsync tests parsing hooks with @async
func TestParseHookWithAsync(t *testing.T) {
	source := `resource Post {
  title: string!

  @after create @async {
    Email.send(self.author, "post_created")
  }
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	hook := program.Resources[0].Hooks[0]

	if !hook.IsAsync {
		t.Error("Expected hook to have @async modifier")
	}
}

// TestParseExpressions tests parsing various expressions
func TestParseExpressions(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "Binary expression",
			source: `resource Test {
  value: int!
  @before create {
    self.value = 5 + 10
  }
}`,
		},
		{
			name: "Function call",
			source: `resource Test {
  slug: string!
  @before create {
    self.slug = String.slugify(self.title)
  }
}`,
		},
		{
			name: "Comparison",
			source: `resource Test {
  active: bool!
  @before create {
    self.active = self.score > 100
  }
}`,
		},
		{
			name: "Logical operators",
			source: `resource Test {
  valid: bool!
  @before create {
    self.valid = self.score > 0 && self.status == "active"
  }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, errors := parseSource(t, tt.source)

			if len(errors) > 0 {
				t.Fatalf("Parse errors: %v", errors)
			}

			if len(program.Resources) != 1 {
				t.Fatalf("Expected 1 resource, got %d", len(program.Resources))
			}

			resource := program.Resources[0]
			if len(resource.Hooks) == 0 {
				t.Fatal("Expected at least one hook")
			}

			hook := resource.Hooks[0]
			if len(hook.Body) == 0 {
				t.Error("Expected non-empty hook body")
			}
		})
	}
}

// TestParseConstraintBlock tests parsing constraint blocks
func TestParseConstraintBlock(t *testing.T) {
	source := `resource Post {
  title: string!

  @constraint published_requires_content {
    on: [create, update]
    when: self.status == "published"
    condition: String.length(self.content) >= 500
    error: "Published posts need 500+ characters"
  }
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	resource := program.Resources[0]

	if len(resource.Constraints) != 1 {
		t.Fatalf("Expected 1 constraint, got %d", len(resource.Constraints))
	}

	constraint := resource.Constraints[0]

	if constraint.Name != "published_requires_content" {
		t.Errorf("Expected constraint name 'published_requires_content', got '%s'", constraint.Name)
	}

	if len(constraint.On) != 2 {
		t.Errorf("Expected 2 events in 'on', got %d", len(constraint.On))
	}

	if constraint.When == nil {
		t.Error("Expected 'when' condition to be parsed")
	}

	if constraint.Condition == nil {
		t.Error("Expected 'condition' to be parsed")
	}

	if constraint.Error != "Published posts need 500+ characters" {
		t.Errorf("Expected error message, got '%s'", constraint.Error)
	}
}

// TestParseScope tests parsing scope definitions
func TestParseScope(t *testing.T) {
	source := `resource Post {
  title: string!

  @scope published {
    self.status == "published"
  }
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	resource := program.Resources[0]

	if len(resource.Scopes) != 1 {
		t.Fatalf("Expected 1 scope, got %d", len(resource.Scopes))
	}

	scope := resource.Scopes[0]

	if scope.Name != "published" {
		t.Errorf("Expected scope name 'published', got '%s'", scope.Name)
	}

	if scope.Condition == nil {
		t.Error("Expected scope condition to be parsed")
	}
}

// TestParseScopeWithArguments tests parsing scopes with arguments
func TestParseScopeWithArguments(t *testing.T) {
	source := `resource Post {
  title: string!

  @scope by_author(user_id: uuid!) {
    self.author_id == user_id
  }
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	scope := program.Resources[0].Scopes[0]

	if len(scope.Arguments) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(scope.Arguments))
	}

	arg := scope.Arguments[0]
	if arg.Name != "user_id" {
		t.Errorf("Expected argument name 'user_id', got '%s'", arg.Name)
	}

	if arg.Type == nil {
		t.Error("Expected argument type to be parsed")
	}
}

// TestParseComputed tests parsing computed fields
func TestParseComputed(t *testing.T) {
	source := `resource User {
  first_name: string!
  last_name: string!

  @computed full_name: string! {
    self.first_name + " " + self.last_name
  }
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	resource := program.Resources[0]

	if len(resource.Computed) != 1 {
		t.Fatalf("Expected 1 computed field, got %d", len(resource.Computed))
	}

	computed := resource.Computed[0]

	if computed.Name != "full_name" {
		t.Errorf("Expected computed field name 'full_name', got '%s'", computed.Name)
	}

	if computed.Type == nil {
		t.Error("Expected computed field type to be parsed")
	}

	if computed.Body == nil {
		t.Error("Expected computed field body to be parsed")
	}
}

// TestParseMultipleResources tests parsing multiple resources
func TestParseMultipleResources(t *testing.T) {
	source := `resource User {
  username: string!
}

resource Post {
  title: string!
}

resource Comment {
  body: text!
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	if len(program.Resources) != 3 {
		t.Fatalf("Expected 3 resources, got %d", len(program.Resources))
	}

	expectedNames := []string{"User", "Post", "Comment"}
	for i, name := range expectedNames {
		if program.Resources[i].Name != name {
			t.Errorf("Expected resource %d to be '%s', got '%s'", i, name, program.Resources[i].Name)
		}
	}
}

// TestParseErrorRecovery tests error recovery
func TestParseErrorRecovery(t *testing.T) {
	source := `resource User {
  username string!
  email: string!
}`

	program, errors := parseSource(t, source)

	if len(errors) == 0 {
		t.Fatal("Expected parse errors")
	}

	// Should still parse the resource
	if len(program.Resources) != 1 {
		t.Errorf("Expected 1 resource despite errors, got %d", len(program.Resources))
	}
}

// TestParseIfStatement tests parsing if statements
func TestParseIfStatement(t *testing.T) {
	source := `resource Post {
  status: string!

  @before update {
    if self.status == "published" {
      self.published_at = Time.now()
    }
  }
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	hook := program.Resources[0].Hooks[0]

	if len(hook.Body) != 1 {
		t.Fatalf("Expected 1 statement in hook body, got %d", len(hook.Body))
	}

	ifStmt, ok := hook.Body[0].(*ast.IfStmt)
	if !ok {
		t.Fatal("Expected if statement")
	}

	if ifStmt.Condition == nil {
		t.Error("Expected if condition")
	}

	if len(ifStmt.ThenBranch) == 0 {
		t.Error("Expected non-empty then branch")
	}
}

// TestParseIfElseStatement tests parsing if/else statements
func TestParseIfElseStatement(t *testing.T) {
	source := `resource Post {
  status: string!

  @before update {
    if self.status == "published" {
      self.published_at = Time.now()
    } else {
      self.published_at = null
    }
  }
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	ifStmt := program.Resources[0].Hooks[0].Body[0].(*ast.IfStmt)

	if len(ifStmt.ElseBranch) == 0 {
		t.Error("Expected non-empty else branch")
	}
}

// TestParseLetStatement tests parsing let statements
func TestParseLetStatement(t *testing.T) {
	source := `resource Post {
  title: string!

  @before create {
    let slug: string! = String.slugify(self.title)
    self.slug = slug
  }
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	hook := program.Resources[0].Hooks[0]

	if len(hook.Body) < 1 {
		t.Fatal("Expected at least 1 statement in hook body")
	}

	letStmt, ok := hook.Body[0].(*ast.LetStmt)
	if !ok {
		t.Fatal("Expected let statement")
	}

	if letStmt.Name != "slug" {
		t.Errorf("Expected variable name 'slug', got '%s'", letStmt.Name)
	}

	if letStmt.Type == nil {
		t.Error("Expected variable type")
	}

	if letStmt.Value == nil {
		t.Error("Expected variable value")
	}
}

// TestParseArrayLiteral tests parsing array literals
func TestParseArrayLiteral(t *testing.T) {
	source := `resource Test {
  items: array<int!>!

  @before create {
    self.items = [1, 2, 3, 4, 5]
  }
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	hook := program.Resources[0].Hooks[0]
	assignStmt := hook.Body[0].(*ast.AssignmentStmt)

	arrayLiteral, ok := assignStmt.Value.(*ast.ArrayLiteralExpr)
	if !ok {
		t.Fatal("Expected array literal")
	}

	if len(arrayLiteral.Elements) != 5 {
		t.Errorf("Expected 5 array elements, got %d", len(arrayLiteral.Elements))
	}
}

// TestParseHashLiteral tests parsing hash literals
func TestParseHashLiteral(t *testing.T) {
	source := `resource Test {
  config: hash<string!, string!>!

  @before create {
    self.config = {name: "test", version: "1.0"}
  }
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	hook := program.Resources[0].Hooks[0]
	assignStmt := hook.Body[0].(*ast.AssignmentStmt)

	hashLiteral, ok := assignStmt.Value.(*ast.HashLiteralExpr)
	if !ok {
		t.Fatal("Expected hash literal")
	}

	if len(hashLiteral.Pairs) != 2 {
		t.Errorf("Expected 2 hash pairs, got %d", len(hashLiteral.Pairs))
	}
}

// TestParseSelfExpression tests parsing self expressions
func TestParseSelfExpression(t *testing.T) {
	source := `resource Post {
  title: string!

  @before create {
    self.slug = String.slugify(self.title)
  }
}`

	program, errors := parseSource(t, source)

	if len(errors) > 0 {
		t.Fatalf("Parse errors: %v", errors)
	}

	hook := program.Resources[0].Hooks[0]
	assignStmt := hook.Body[0].(*ast.AssignmentStmt)

	// Check target is field access on self
	fieldAccess, ok := assignStmt.Target.(*ast.FieldAccessExpr)
	if !ok {
		t.Fatal("Expected field access expression")
	}

	_, ok = fieldAccess.Object.(*ast.SelfExpr)
	if !ok {
		t.Error("Expected self expression")
	}
}
