package query

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func createTestScope() *schema.Scope {
	return &schema.Scope{
		Name: "published",
		Arguments: []*schema.ScopeArgument{
			{
				Name: "since",
				Type: &schema.TypeSpec{
					BaseType: schema.TypeTimestamp,
					Nullable: true,
				},
			},
		},
		OrderBy: "created_at DESC",
		Limit:   intPtr(10),
	}
}

func intPtr(i int) *int {
	return &i
}

func TestNewScopeCompiler(t *testing.T) {
	resource := createTestResource()
	compiler := NewScopeCompiler(resource)

	if compiler == nil {
		t.Fatal("NewScopeCompiler returned nil")
	}

	if compiler.resource != resource {
		t.Error("Resource not set correctly")
	}
}

func TestScopeCompiler_CompileScope(t *testing.T) {
	resource := createTestResource()
	compiler := NewScopeCompiler(resource)

	scopeDef := createTestScope()

	compiled, err := compiler.CompileScope(scopeDef)
	if err != nil {
		t.Fatalf("CompileScope failed: %v", err)
	}

	if compiled == nil {
		t.Fatal("CompileScope returned nil")
	}

	if compiled.Name != "published" {
		t.Errorf("Expected name 'published', got '%s'", compiled.Name)
	}

	if len(compiled.Arguments) != 1 {
		t.Errorf("Expected 1 argument, got %d", len(compiled.Arguments))
	}

	if len(compiled.OrderBy) != 1 {
		t.Errorf("Expected 1 order by, got %d", len(compiled.OrderBy))
	}

	if compiled.Limit == nil || *compiled.Limit != 10 {
		t.Error("Limit not set correctly")
	}
}

func TestCompiledScope_Bind(t *testing.T) {
	resource := createTestResource()
	compiler := NewScopeCompiler(resource)

	scopeDef := createTestScope()
	compiled, err := compiler.CompileScope(scopeDef)
	if err != nil {
		t.Fatalf("CompileScope failed: %v", err)
	}

	// Bind with correct arguments
	args := []interface{}{nil} // nullable timestamp
	bound, err := compiled.Bind(args)
	if err != nil {
		t.Fatalf("Bind failed: %v", err)
	}

	if bound == nil {
		t.Fatal("Bind returned nil")
	}

	if bound.Name != "published" {
		t.Errorf("Expected name 'published', got '%s'", bound.Name)
	}

	if bound.Arguments["since"] != nil {
		t.Error("Expected nil argument")
	}
}

func TestCompiledScope_BindInvalidArgumentCount(t *testing.T) {
	resource := createTestResource()
	compiler := NewScopeCompiler(resource)

	scopeDef := createTestScope()
	compiled, err := compiler.CompileScope(scopeDef)
	if err != nil {
		t.Fatalf("CompileScope failed: %v", err)
	}

	// Bind with wrong number of arguments
	args := []interface{}{} // Should be 1
	_, err = compiled.Bind(args)
	if err == nil {
		t.Error("Expected error for wrong argument count")
	}
}

func TestNewScopeRegistry(t *testing.T) {
	registry := NewScopeRegistry()

	if registry == nil {
		t.Fatal("NewScopeRegistry returned nil")
	}

	if len(registry.List()) != 0 {
		t.Error("Registry should be empty initially")
	}
}

func TestScopeRegistry_RegisterAndGet(t *testing.T) {
	registry := NewScopeRegistry()

	resource := createTestResource()
	compiler := NewScopeCompiler(resource)
	scopeDef := createTestScope()
	compiled, _ := compiler.CompileScope(scopeDef)

	registry.Register(compiled)

	retrieved, err := registry.Get("published")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Name != "published" {
		t.Errorf("Expected name 'published', got '%s'", retrieved.Name)
	}
}

func TestScopeRegistry_GetNonExistent(t *testing.T) {
	registry := NewScopeRegistry()

	_, err := registry.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent scope")
	}
}

func TestScopeRegistry_Has(t *testing.T) {
	registry := NewScopeRegistry()

	if registry.Has("published") {
		t.Error("Should not have 'published' scope yet")
	}

	resource := createTestResource()
	compiler := NewScopeCompiler(resource)
	scopeDef := createTestScope()
	compiled, _ := compiler.CompileScope(scopeDef)
	registry.Register(compiled)

	if !registry.Has("published") {
		t.Error("Should have 'published' scope")
	}
}

func TestScopeRegistry_List(t *testing.T) {
	registry := NewScopeRegistry()

	resource := createTestResource()
	compiler := NewScopeCompiler(resource)

	// Register multiple scopes
	scopeDef1 := createTestScope()
	compiled1, _ := compiler.CompileScope(scopeDef1)
	registry.Register(compiled1)

	scopeDef2 := &schema.Scope{Name: "draft"}
	compiled2, _ := compiler.CompileScope(scopeDef2)
	registry.Register(compiled2)

	names := registry.List()
	if len(names) != 2 {
		t.Errorf("Expected 2 scopes, got %d", len(names))
	}
}

func TestNewScopeChain(t *testing.T) {
	chain := NewScopeChain()

	if chain == nil {
		t.Fatal("NewScopeChain returned nil")
	}

	if len(chain.scopes) != 0 {
		t.Error("Chain should be empty initially")
	}
}

func TestScopeChain_Add(t *testing.T) {
	chain := NewScopeChain()

	bound := &BoundScope{
		Name:       "test",
		Conditions: make([]*Condition, 0),
		OrderBy:    make([]string, 0),
		Arguments:  make(map[string]interface{}),
	}

	chain.Add(bound)

	if len(chain.scopes) != 1 {
		t.Errorf("Expected 1 scope in chain, got %d", len(chain.scopes))
	}
}

func TestScopeChain_Apply(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	chain := NewScopeChain()

	// Add scope with condition
	bound := &BoundScope{
		Name: "published",
		Conditions: []*Condition{
			{
				Field:    "status",
				Operator: OpEqual,
				Value:    "published",
			},
		},
		OrderBy:   []string{"created_at DESC"},
		Arguments: make(map[string]interface{}),
	}

	chain.Add(bound)

	err := chain.Apply(qb)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(qb.conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(qb.conditions))
	}

	if len(qb.orderBy) != 1 {
		t.Errorf("Expected 1 order by, got %d", len(qb.orderBy))
	}
}

func TestScopeChain_ApplyMultiple(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	chain := NewScopeChain()

	// Add first scope
	bound1 := &BoundScope{
		Name: "published",
		Conditions: []*Condition{
			{
				Field:    "status",
				Operator: OpEqual,
				Value:    "published",
			},
		},
		Arguments: make(map[string]interface{}),
	}
	chain.Add(bound1)

	// Add second scope
	bound2 := &BoundScope{
		Name: "popular",
		Conditions: []*Condition{
			{
				Field:    "views",
				Operator: OpGreaterThan,
				Value:    1000,
			},
		},
		Arguments: make(map[string]interface{}),
	}
	chain.Add(bound2)

	err := chain.Apply(qb)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	if len(qb.conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(qb.conditions))
	}
}

func TestScopeChain_Merge(t *testing.T) {
	chain := NewScopeChain()

	bound1 := &BoundScope{
		Name: "published",
		Conditions: []*Condition{
			{Field: "status", Operator: OpEqual, Value: "published"},
		},
		OrderBy:   []string{"created_at DESC"},
		Arguments: make(map[string]interface{}),
	}

	bound2 := &BoundScope{
		Name: "popular",
		Conditions: []*Condition{
			{Field: "views", Operator: OpGreaterThan, Value: 1000},
		},
		Arguments: make(map[string]interface{}),
	}

	chain.Add(bound1)
	chain.Add(bound2)

	merged := chain.Merge()

	if len(merged.Conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(merged.Conditions))
	}

	if len(merged.OrderBy) != 1 {
		t.Errorf("Expected 1 order by, got %d", len(merged.OrderBy))
	}
}

func TestDefaultScopes_Recent(t *testing.T) {
	resource := createTestResource()
	ds := NewDefaultScopes(resource)

	scope := ds.Recent(10)

	if scope.Name != "recent" {
		t.Errorf("Expected name 'recent', got '%s'", scope.Name)
	}

	if scope.Limit == nil || *scope.Limit != 10 {
		t.Error("Limit not set correctly")
	}

	if len(scope.OrderBy) != 1 {
		t.Error("OrderBy not set")
	}

	if scope.OrderBy[0] != "created_at DESC" {
		t.Errorf("Expected 'created_at DESC', got '%s'", scope.OrderBy[0])
	}
}

func TestDefaultScopes_Active(t *testing.T) {
	resource := createTestResource()
	ds := NewDefaultScopes(resource)

	scope := ds.Active()

	if scope.Name != "active" {
		t.Errorf("Expected name 'active', got '%s'", scope.Name)
	}

	if len(scope.Conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(scope.Conditions))
	}

	if scope.Conditions[0].Field != "status" {
		t.Error("Condition should be on status field")
	}
}

func TestDefaultScopes_Archived(t *testing.T) {
	resource := createTestResource()
	ds := NewDefaultScopes(resource)

	scope := ds.Archived()

	if len(scope.Conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(scope.Conditions))
	}

	if scope.Conditions[0].Operator != OpIsNotNull {
		t.Error("Should check for IS NOT NULL")
	}
}

func TestDefaultScopes_NotArchived(t *testing.T) {
	resource := createTestResource()
	ds := NewDefaultScopes(resource)

	scope := ds.NotArchived()

	if len(scope.Conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(scope.Conditions))
	}

	if scope.Conditions[0].Operator != OpIsNull {
		t.Error("Should check for IS NULL")
	}
}

func TestDefaultScopes_CreatedAfter(t *testing.T) {
	resource := createTestResource()
	ds := NewDefaultScopes(resource)

	timestamp := "2024-01-01T00:00:00Z"
	scope := ds.CreatedAfter(timestamp)

	if len(scope.Conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(scope.Conditions))
	}

	if scope.Conditions[0].Field != "created_at" {
		t.Error("Condition should be on created_at field")
	}

	if scope.Conditions[0].Operator != OpGreaterThan {
		t.Error("Should use greater than operator")
	}
}

func TestDefaultScopes_Paginate(t *testing.T) {
	resource := createTestResource()
	ds := NewDefaultScopes(resource)

	scope := ds.Paginate(2, 20) // Page 2, 20 per page

	if scope.Limit == nil || *scope.Limit != 20 {
		t.Error("Limit not set correctly")
	}

	expectedOffset := 20 // (2-1) * 20
	if scope.Offset == nil || *scope.Offset != expectedOffset {
		t.Errorf("Expected offset %d, got %d", expectedOffset, *scope.Offset)
	}
}

func TestValidateArgumentType_String(t *testing.T) {
	typeSpec := &schema.TypeSpec{
		BaseType: schema.TypeString,
		Nullable: false,
	}

	err := validateArgumentType("test", typeSpec)
	if err != nil {
		t.Errorf("Valid string should not error: %v", err)
	}

	err = validateArgumentType(123, typeSpec)
	if err == nil {
		t.Error("Invalid type should error")
	}
}

func TestValidateArgumentType_Int(t *testing.T) {
	typeSpec := &schema.TypeSpec{
		BaseType: schema.TypeInt,
		Nullable: false,
	}

	err := validateArgumentType(42, typeSpec)
	if err != nil {
		t.Errorf("Valid int should not error: %v", err)
	}

	err = validateArgumentType("not an int", typeSpec)
	if err == nil {
		t.Error("Invalid type should error")
	}
}

func TestValidateArgumentType_Nullable(t *testing.T) {
	typeSpec := &schema.TypeSpec{
		BaseType: schema.TypeString,
		Nullable: true,
	}

	err := validateArgumentType(nil, typeSpec)
	if err != nil {
		t.Errorf("Nil should be valid for nullable type: %v", err)
	}

	typeSpec.Nullable = false
	err = validateArgumentType(nil, typeSpec)
	if err == nil {
		t.Error("Nil should not be valid for non-nullable type")
	}
}

// Benchmark tests
func BenchmarkScopeCompiler_CompileScope(b *testing.B) {
	resource := createTestResource()
	compiler := NewScopeCompiler(resource)
	scopeDef := createTestScope()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compiler.CompileScope(scopeDef)
	}
}

func BenchmarkCompiledScope_Bind(b *testing.B) {
	resource := createTestResource()
	compiler := NewScopeCompiler(resource)
	scopeDef := createTestScope()
	compiled, _ := compiler.CompileScope(scopeDef)

	args := []interface{}{nil}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compiled.Bind(args)
	}
}

func BenchmarkScopeChain_Apply(b *testing.B) {
	resource := createTestResource()
	chain := NewScopeChain()

	bound := &BoundScope{
		Name: "published",
		Conditions: []*Condition{
			{Field: "status", Operator: OpEqual, Value: "published"},
		},
		OrderBy:   []string{"created_at DESC"},
		Arguments: make(map[string]interface{}),
	}
	chain.Add(bound)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := NewQueryBuilder(resource, nil, nil)
		_ = chain.Apply(qb)
	}
}
