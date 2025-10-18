package query

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// Helper function to create a test resource schema
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
			Nullable: false,
		},
	}

	resource.Fields["status"] = &schema.Field{
		Name: "status",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeString,
			Nullable: false,
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

	return resource
}

func TestNewQueryBuilder(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	qb := NewQueryBuilder(resource, nil, schemas)

	if qb == nil {
		t.Fatal("NewQueryBuilder returned nil")
	}

	if qb.resource != resource {
		t.Error("QueryBuilder resource not set correctly")
	}

	if len(qb.conditions) != 0 {
		t.Errorf("Expected 0 conditions, got %d", len(qb.conditions))
	}
}

func TestWhere(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.Where("title", OpEqual, "Test Post")

	if len(qb.conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(qb.conditions))
	}

	cond := qb.conditions[0]
	if cond.Field != "title" {
		t.Errorf("Expected field 'title', got '%s'", cond.Field)
	}
	if cond.Operator != OpEqual {
		t.Errorf("Expected OpEqual, got %v", cond.Operator)
	}
	if cond.Value != "Test Post" {
		t.Errorf("Expected value 'Test Post', got %v", cond.Value)
	}
	if cond.Or {
		t.Error("Expected AND condition, got OR")
	}
}

func TestOrWhere(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.Where("status", OpEqual, "published").
		OrWhere("status", OpEqual, "draft")

	if len(qb.conditions) != 2 {
		t.Fatalf("Expected 2 conditions, got %d", len(qb.conditions))
	}

	if !qb.conditions[1].Or {
		t.Error("Second condition should be OR")
	}
}

func TestWhereIn(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	values := []interface{}{"published", "draft", "archived"}
	qb.WhereIn("status", values)

	if len(qb.conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(qb.conditions))
	}

	cond := qb.conditions[0]
	if cond.Operator != OpIn {
		t.Errorf("Expected OpIn, got %v", cond.Operator)
	}
}

func TestWhereNull(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.WhereNull("published_at")

	if len(qb.conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(qb.conditions))
	}

	cond := qb.conditions[0]
	if cond.Operator != OpIsNull {
		t.Errorf("Expected OpIsNull, got %v", cond.Operator)
	}
}

func TestWhereNotNull(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.WhereNotNull("published_at")

	if len(qb.conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(qb.conditions))
	}

	cond := qb.conditions[0]
	if cond.Operator != OpIsNotNull {
		t.Errorf("Expected OpIsNotNull, got %v", cond.Operator)
	}
}

func TestWhereLike(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.WhereLike("title", "%test%")

	if len(qb.conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(qb.conditions))
	}

	cond := qb.conditions[0]
	if cond.Operator != OpLike {
		t.Errorf("Expected OpLike, got %v", cond.Operator)
	}
}

func TestWhereILike(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.WhereILike("title", "%TEST%")

	if len(qb.conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(qb.conditions))
	}

	cond := qb.conditions[0]
	if cond.Operator != OpILike {
		t.Errorf("Expected OpILike, got %v", cond.Operator)
	}
}

func TestWhereBetween(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.WhereBetween("views", 100, 1000)

	if len(qb.conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(qb.conditions))
	}

	cond := qb.conditions[0]
	if cond.Operator != OpBetween {
		t.Errorf("Expected OpBetween, got %v", cond.Operator)
	}
}

func TestOrderBy(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.OrderBy("title", "ASC").OrderBy("created_at", "DESC")

	if len(qb.orderBy) != 2 {
		t.Fatalf("Expected 2 order by clauses, got %d", len(qb.orderBy))
	}

	if qb.orderBy[0] != "title ASC" {
		t.Errorf("Expected 'title ASC', got '%s'", qb.orderBy[0])
	}
	if qb.orderBy[1] != "created_at DESC" {
		t.Errorf("Expected 'created_at DESC', got '%s'", qb.orderBy[1])
	}
}

func TestOrderByAsc(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.OrderByAsc("title")

	if len(qb.orderBy) != 1 {
		t.Fatalf("Expected 1 order by clause, got %d", len(qb.orderBy))
	}

	if qb.orderBy[0] != "title ASC" {
		t.Errorf("Expected 'title ASC', got '%s'", qb.orderBy[0])
	}
}

func TestOrderByDesc(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.OrderByDesc("created_at")

	if len(qb.orderBy) != 1 {
		t.Fatalf("Expected 1 order by clause, got %d", len(qb.orderBy))
	}

	if qb.orderBy[0] != "created_at DESC" {
		t.Errorf("Expected 'created_at DESC', got '%s'", qb.orderBy[0])
	}
}

func TestLimit(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.Limit(10)

	if qb.limit == nil {
		t.Fatal("Limit not set")
	}
	if *qb.limit != 10 {
		t.Errorf("Expected limit 10, got %d", *qb.limit)
	}
}

func TestOffset(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.Offset(20)

	if qb.offset == nil {
		t.Fatal("Offset not set")
	}
	if *qb.offset != 20 {
		t.Errorf("Expected offset 20, got %d", *qb.offset)
	}
}

func TestIncludes(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.Includes("author", "comments")

	if len(qb.includes) != 2 {
		t.Fatalf("Expected 2 includes, got %d", len(qb.includes))
	}

	if qb.includes[0] != "author" {
		t.Errorf("Expected 'author', got '%s'", qb.includes[0])
	}
	if qb.includes[1] != "comments" {
		t.Errorf("Expected 'comments', got '%s'", qb.includes[1])
	}
}

func TestJoin(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.Join(InnerJoin, "users", "users.id = posts.author_id")

	if len(qb.joins) != 1 {
		t.Fatalf("Expected 1 join, got %d", len(qb.joins))
	}

	join := qb.joins[0]
	if join.Type != InnerJoin {
		t.Errorf("Expected InnerJoin, got %v", join.Type)
	}
	if join.Table != "users" {
		t.Errorf("Expected table 'users', got '%s'", join.Table)
	}
}

func TestInnerJoin(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.InnerJoin("users", "users.id = posts.author_id")

	if len(qb.joins) != 1 {
		t.Fatalf("Expected 1 join, got %d", len(qb.joins))
	}

	if qb.joins[0].Type != InnerJoin {
		t.Error("Expected InnerJoin type")
	}
}

func TestLeftJoin(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.LeftJoin("users", "users.id = posts.author_id")

	if len(qb.joins) != 1 {
		t.Fatalf("Expected 1 join, got %d", len(qb.joins))
	}

	if qb.joins[0].Type != LeftJoin {
		t.Error("Expected LeftJoin type")
	}
}

func TestGroupBy(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.GroupBy("status", "author_id")

	if len(qb.groupBy) != 2 {
		t.Fatalf("Expected 2 group by fields, got %d", len(qb.groupBy))
	}

	if qb.groupBy[0] != "status" {
		t.Errorf("Expected 'status', got '%s'", qb.groupBy[0])
	}
}

func TestHaving(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.Having("COUNT(*)", OpGreaterThan, 5)

	if len(qb.having) != 1 {
		t.Fatalf("Expected 1 having condition, got %d", len(qb.having))
	}

	cond := qb.having[0]
	if cond.Field != "COUNT(*)" {
		t.Errorf("Expected field 'COUNT(*)', got '%s'", cond.Field)
	}
}

func TestToSQL_SimpleSelect(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	expectedSQL := "SELECT * FROM posts"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

func TestToSQL_WithWhere(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.Where("status", OpEqual, "published")

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	expectedSQL := "SELECT * FROM posts WHERE status = $1"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 1 {
		t.Fatalf("Expected 1 arg, got %d", len(args))
	}

	if args[0] != "published" {
		t.Errorf("Expected arg 'published', got %v", args[0])
	}
}

func TestToSQL_MultipleConditions(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.Where("status", OpEqual, "published").
		Where("views", OpGreaterThan, 100)

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	expectedSQL := "SELECT * FROM posts WHERE status = $1 AND views > $2"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 {
		t.Fatalf("Expected 2 args, got %d", len(args))
	}
}

func TestToSQL_WithOrCondition(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.Where("status", OpEqual, "published").
		OrWhere("status", OpEqual, "draft")

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	expectedSQL := "SELECT * FROM posts WHERE status = $1 OR status = $2"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
}

func TestToSQL_WithOrderBy(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.OrderByDesc("created_at")

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	expectedSQL := "SELECT * FROM posts ORDER BY created_at DESC"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

func TestToSQL_WithLimitAndOffset(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.Limit(10).Offset(20)

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	expectedSQL := "SELECT * FROM posts LIMIT $1 OFFSET $2"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 2 {
		t.Fatalf("Expected 2 args, got %d", len(args))
	}

	if args[0] != 10 {
		t.Errorf("Expected limit 10, got %v", args[0])
	}
	if args[1] != 20 {
		t.Errorf("Expected offset 20, got %v", args[1])
	}
}

func TestToSQL_ComplexQuery(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.Where("status", OpEqual, "published").
		Where("views", OpGreaterThan, 100).
		OrderByDesc("created_at").
		Limit(10)

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	expectedSQL := "SELECT * FROM posts WHERE status = $1 AND views > $2 ORDER BY created_at DESC LIMIT $3"
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 3 {
		t.Fatalf("Expected 3 args, got %d", len(args))
	}
}

func TestClone(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	qb.Where("status", OpEqual, "published").
		OrderByDesc("created_at").
		Limit(10)

	clone := qb.Clone()

	// Modify clone
	clone.Where("views", OpGreaterThan, 100)

	// Original should be unchanged
	if len(qb.conditions) != 1 {
		t.Errorf("Original query builder was modified")
	}

	// Clone should have new condition
	if len(clone.conditions) != 2 {
		t.Errorf("Clone should have 2 conditions, got %d", len(clone.conditions))
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Post", "post"},
		{"BlogPost", "blog_post"},
		{"UserProfile", "user_profile"},
		{"HTTPServer", "http_server"},
		{"ID", "id"},
	}

	for _, tt := range tests {
		result := toSnakeCase(tt.input)
		if result != tt.expected {
			t.Errorf("toSnakeCase(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"post", "posts"},
		{"box", "boxes"},
		{"category", "categories"},
		{"user", "users"},
		{"glass", "glasses"},
	}

	for _, tt := range tests {
		result := pluralize(tt.input)
		if result != tt.expected {
			t.Errorf("pluralize(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestToTableName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Post", "posts"},
		{"BlogPost", "blog_posts"},
		{"Category", "categories"},
	}

	for _, tt := range tests {
		result := toTableName(tt.input)
		if result != tt.expected {
			t.Errorf("toTableName(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

// Tests for Fix #1: SQL Injection Validation in JOIN conditions
func TestJoin_InvalidTableName(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid table name with special characters")
		}
	}()

	// Should panic due to SQL injection attempt
	qb.Join(InnerJoin, "users; DROP TABLE posts;", "users.id = posts.author_id")
}

func TestJoin_InvalidCondition(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid join condition")
		}
	}()

	// Should panic due to invalid condition format
	qb.Join(InnerJoin, "users", "1=1 OR 1=1")
}

func TestJoin_ValidCondition(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	// Should succeed with valid condition
	qb.Join(InnerJoin, "users", "users.id = posts.author_id")

	if len(qb.joins) != 1 {
		t.Fatalf("Expected 1 join, got %d", len(qb.joins))
	}
}

func TestInnerJoin_ValidationWorks(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid table name")
		}
	}()

	qb.InnerJoin("users' OR '1'='1", "users.id = posts.author_id")
}

func TestLeftJoin_ValidationWorks(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid condition")
		}
	}()

	qb.LeftJoin("users", "DROP TABLE posts")
}

// Tests for Fix #2: Field Validation in WHERE clauses
func TestWhere_InvalidField(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for non-existent field")
		}
	}()

	qb.Where("nonexistent_field", OpEqual, "value")
}

func TestWhere_ValidField(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	// Should succeed with valid field
	qb.Where("title", OpEqual, "Test Post")

	if len(qb.conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(qb.conditions))
	}
}

func TestOrWhere_InvalidField(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for non-existent field in OrWhere")
		}
	}()

	qb.OrWhere("invalid_field", OpEqual, "value")
}

func TestWhereIn_InvalidField(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for non-existent field in WhereIn")
		}
	}()

	qb.WhereIn("invalid_field", []interface{}{"a", "b"})
}

func TestWhereNull_InvalidField(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for non-existent field in WhereNull")
		}
	}()

	qb.WhereNull("invalid_field")
}

func TestHaving_InvalidField(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for non-existent field in Having")
		}
	}()

	qb.Having("invalid_field", OpGreaterThan, 5)
}

func TestHaving_AllowsAggregates(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	// Should allow aggregate functions (containing parentheses)
	qb.Having("COUNT(*)", OpGreaterThan, 5)

	if len(qb.having) != 1 {
		t.Fatalf("Expected 1 having condition, got %d", len(qb.having))
	}
}

// Tests for Fix #3: Scope WHERE condition application
func TestScope_AppliesWhereConditions(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	// Create a scope with WHERE conditions
	scope := &schema.Scope{
		Name: "published",
		Where: map[string]interface{}{
			"status": "= published",
		},
		Arguments: []*schema.ScopeArgument{},
	}
	resource.Scopes["published"] = scope

	// Apply scope
	_, err := qb.Scope("published")
	if err != nil {
		t.Fatalf("Failed to apply scope: %v", err)
	}

	// Should have 1 condition from the scope
	if len(qb.conditions) != 1 {
		t.Fatalf("Expected 1 condition from scope, got %d", len(qb.conditions))
	}

	cond := qb.conditions[0]
	if cond.Field != "status" {
		t.Errorf("Expected field 'status', got '%s'", cond.Field)
	}
	if cond.Operator != OpEqual {
		t.Errorf("Expected OpEqual, got %v", cond.Operator)
	}
	if cond.Value != "published" {
		t.Errorf("Expected value 'published', got %v", cond.Value)
	}
}

func TestScope_AppliesWhereWithArguments(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	// Create a scope with arguments
	scope := &schema.Scope{
		Name: "with_min_views",
		Where: map[string]interface{}{
			"views": "> $min_views",
		},
		Arguments: []*schema.ScopeArgument{
			{
				Name: "min_views",
				Type: &schema.TypeSpec{BaseType: schema.TypeInt},
			},
		},
	}
	resource.Scopes["with_min_views"] = scope

	// Apply scope with argument
	_, err := qb.Scope("with_min_views", 100)
	if err != nil {
		t.Fatalf("Failed to apply scope: %v", err)
	}

	// Should have 1 condition from the scope
	if len(qb.conditions) != 1 {
		t.Fatalf("Expected 1 condition from scope, got %d", len(qb.conditions))
	}

	cond := qb.conditions[0]
	if cond.Field != "views" {
		t.Errorf("Expected field 'views', got '%s'", cond.Field)
	}
	if cond.Operator != OpGreaterThan {
		t.Errorf("Expected OpGreaterThan, got %v", cond.Operator)
	}
	if cond.Value != 100 {
		t.Errorf("Expected value 100, got %v", cond.Value)
	}
}

func TestScope_AppliesOrderBy(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	// Create a scope with ORDER BY
	scope := &schema.Scope{
		Name:      "recent",
		Where:     map[string]interface{}{},
		OrderBy:   "created_at DESC",
		Arguments: []*schema.ScopeArgument{},
	}
	resource.Scopes["recent"] = scope

	// Apply scope
	_, err := qb.Scope("recent")
	if err != nil {
		t.Fatalf("Failed to apply scope: %v", err)
	}

	// Should have ORDER BY clause
	if len(qb.orderBy) != 1 {
		t.Fatalf("Expected 1 order by clause from scope, got %d", len(qb.orderBy))
	}

	if qb.orderBy[0] != "created_at DESC" {
		t.Errorf("Expected 'created_at DESC', got '%s'", qb.orderBy[0])
	}
}

func TestScope_AppliesLimitAndOffset(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	limit := 10
	offset := 5
	scope := &schema.Scope{
		Name:      "paginated",
		Where:     map[string]interface{}{},
		Limit:     &limit,
		Offset:    &offset,
		Arguments: []*schema.ScopeArgument{},
	}
	resource.Scopes["paginated"] = scope

	// Apply scope
	_, err := qb.Scope("paginated")
	if err != nil {
		t.Fatalf("Failed to apply scope: %v", err)
	}

	// Should have limit and offset
	if qb.limit == nil || *qb.limit != 10 {
		t.Errorf("Expected limit 10, got %v", qb.limit)
	}
	if qb.offset == nil || *qb.offset != 5 {
		t.Errorf("Expected offset 5, got %v", qb.offset)
	}
}

func TestScope_ErrorOnWrongArgumentCount(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	scope := &schema.Scope{
		Name:  "with_status",
		Where: map[string]interface{}{"status": "= $status"},
		Arguments: []*schema.ScopeArgument{
			{Name: "status", Type: &schema.TypeSpec{BaseType: schema.TypeString}},
		},
	}
	resource.Scopes["with_status"] = scope

	// Try to apply scope without arguments
	_, err := qb.Scope("with_status")
	if err == nil {
		t.Error("Expected error when applying scope with wrong argument count")
	}
}

func TestScope_ErrorOnUnknownArgument(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	scope := &schema.Scope{
		Name:  "test_scope",
		Where: map[string]interface{}{"status": "= $unknown_arg"},
		Arguments: []*schema.ScopeArgument{
			{Name: "known_arg", Type: &schema.TypeSpec{BaseType: schema.TypeString}},
		},
	}
	resource.Scopes["test_scope"] = scope

	// Try to apply scope - should fail because $unknown_arg is not in arguments
	_, err := qb.Scope("test_scope", "value")
	if err == nil {
		t.Error("Expected error when scope references unknown argument")
	}
}

// Additional tests for operator support in JOIN conditions
func TestJoin_ValidConditionWithGreaterThan(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	// Should not panic with > operator
	qb.InnerJoin("orders", "orders.created_at > users.signup_date")

	if len(qb.joins) != 1 {
		t.Errorf("expected 1 join, got %d", len(qb.joins))
	}
}

func TestJoin_ValidConditionWithCompoundAnd(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	// Should not panic with AND operator
	qb.InnerJoin("posts", "posts.user_id = users.id AND posts.status != users.status")

	if len(qb.joins) != 1 {
		t.Errorf("expected 1 join, got %d", len(qb.joins))
	}
}

func TestJoin_ValidConditionWithNotEqual(t *testing.T) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)

	// Should not panic with != operator
	qb.InnerJoin("items", "items.status != base.status")

	if len(qb.joins) != 1 {
		t.Errorf("expected 1 join, got %d", len(qb.joins))
	}
}

// Benchmark tests
func BenchmarkQueryBuilder_SimpleWhere(b *testing.B) {
	resource := createTestResource()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := NewQueryBuilder(resource, nil, nil)
		qb.Where("status", OpEqual, "published")
	}
}

func BenchmarkQueryBuilder_ToSQL(b *testing.B) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)
	qb.Where("status", OpEqual, "published").
		Where("views", OpGreaterThan, 100).
		OrderByDesc("created_at").
		Limit(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sql, args, err := qb.ToSQL()
		_ = sql
		_ = args
		_ = err
	}
}

func BenchmarkQueryBuilder_Clone(b *testing.B) {
	resource := createTestResource()
	qb := NewQueryBuilder(resource, nil, nil)
	qb.Where("status", OpEqual, "published").
		OrderByDesc("created_at").
		Limit(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clone := qb.Clone()
		_ = clone
	}
}
