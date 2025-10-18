package query

import (
	"strings"
	"testing"
)

func TestOperator_String(t *testing.T) {
	tests := []struct {
		op       Operator
		expected string
	}{
		{OpEqual, "="},
		{OpNotEqual, "!="},
		{OpGreaterThan, ">"},
		{OpGreaterThanOrEqual, ">="},
		{OpLessThan, "<"},
		{OpLessThanOrEqual, "<="},
		{OpIn, "IN"},
		{OpNotIn, "NOT IN"},
		{OpLike, "LIKE"},
		{OpILike, "ILIKE"},
		{OpIsNull, "IS NULL"},
		{OpIsNotNull, "IS NOT NULL"},
		{OpBetween, "BETWEEN"},
	}

	for _, tt := range tests {
		result := tt.op.String()
		if result != tt.expected {
			t.Errorf("Operator.String() = %s, want %s", result, tt.expected)
		}
	}
}

func TestConditionToSQL_Equal(t *testing.T) {
	cond := &Condition{
		Field:    "status",
		Operator: OpEqual,
		Value:    "published",
		Or:       false,
	}

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := conditionToSQL(cond, &paramCounter, &args)
	if err != nil {
		t.Fatalf("conditionToSQL failed: %v", err)
	}

	expectedSQL := "status = $1"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}

	if len(args) != 1 {
		t.Fatalf("Expected 1 arg, got %d", len(args))
	}

	if args[0] != "published" {
		t.Errorf("Expected arg 'published', got %v", args[0])
	}

	if paramCounter != 2 {
		t.Errorf("Expected paramCounter 2, got %d", paramCounter)
	}
}

func TestConditionToSQL_NotEqual(t *testing.T) {
	cond := &Condition{
		Field:    "status",
		Operator: OpNotEqual,
		Value:    "draft",
	}

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := conditionToSQL(cond, &paramCounter, &args)
	if err != nil {
		t.Fatalf("conditionToSQL failed: %v", err)
	}

	expectedSQL := "status != $1"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}
}

func TestConditionToSQL_GreaterThan(t *testing.T) {
	cond := &Condition{
		Field:    "views",
		Operator: OpGreaterThan,
		Value:    100,
	}

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := conditionToSQL(cond, &paramCounter, &args)
	if err != nil {
		t.Fatalf("conditionToSQL failed: %v", err)
	}

	expectedSQL := "views > $1"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}

	if args[0] != 100 {
		t.Errorf("Expected arg 100, got %v", args[0])
	}
}

func TestConditionToSQL_In(t *testing.T) {
	cond := &Condition{
		Field:    "status",
		Operator: OpIn,
		Value:    []interface{}{"published", "draft", "archived"},
	}

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := conditionToSQL(cond, &paramCounter, &args)
	if err != nil {
		t.Fatalf("conditionToSQL failed: %v", err)
	}

	expectedSQL := "status IN ($1, $2, $3)"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}

	if len(args) != 3 {
		t.Fatalf("Expected 3 args, got %d", len(args))
	}

	if paramCounter != 4 {
		t.Errorf("Expected paramCounter 4, got %d", paramCounter)
	}
}

func TestConditionToSQL_InEmpty(t *testing.T) {
	cond := &Condition{
		Field:    "status",
		Operator: OpIn,
		Value:    []interface{}{},
	}

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := conditionToSQL(cond, &paramCounter, &args)
	if err != nil {
		t.Fatalf("conditionToSQL failed: %v", err)
	}

	// IN with empty array should return FALSE
	if sql != "FALSE" {
		t.Errorf("Expected SQL: FALSE, got: %s", sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

func TestConditionToSQL_NotIn(t *testing.T) {
	cond := &Condition{
		Field:    "status",
		Operator: OpNotIn,
		Value:    []interface{}{"archived", "deleted"},
	}

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := conditionToSQL(cond, &paramCounter, &args)
	if err != nil {
		t.Fatalf("conditionToSQL failed: %v", err)
	}

	expectedSQL := "status NOT IN ($1, $2)"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}
}

func TestConditionToSQL_NotInEmpty(t *testing.T) {
	cond := &Condition{
		Field:    "status",
		Operator: OpNotIn,
		Value:    []interface{}{},
	}

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := conditionToSQL(cond, &paramCounter, &args)
	if err != nil {
		t.Fatalf("conditionToSQL failed: %v", err)
	}

	// NOT IN with empty array should return TRUE
	if sql != "TRUE" {
		t.Errorf("Expected SQL: TRUE, got: %s", sql)
	}
}

func TestConditionToSQL_Like(t *testing.T) {
	cond := &Condition{
		Field:    "title",
		Operator: OpLike,
		Value:    "%test%",
	}

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := conditionToSQL(cond, &paramCounter, &args)
	if err != nil {
		t.Fatalf("conditionToSQL failed: %v", err)
	}

	expectedSQL := "title LIKE $1"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}
}

func TestConditionToSQL_ILike(t *testing.T) {
	cond := &Condition{
		Field:    "title",
		Operator: OpILike,
		Value:    "%TEST%",
	}

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := conditionToSQL(cond, &paramCounter, &args)
	if err != nil {
		t.Fatalf("conditionToSQL failed: %v", err)
	}

	expectedSQL := "title ILIKE $1"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}
}

func TestConditionToSQL_IsNull(t *testing.T) {
	cond := &Condition{
		Field:    "deleted_at",
		Operator: OpIsNull,
		Value:    nil,
	}

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := conditionToSQL(cond, &paramCounter, &args)
	if err != nil {
		t.Fatalf("conditionToSQL failed: %v", err)
	}

	expectedSQL := "deleted_at IS NULL"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

func TestConditionToSQL_IsNotNull(t *testing.T) {
	cond := &Condition{
		Field:    "published_at",
		Operator: OpIsNotNull,
		Value:    nil,
	}

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := conditionToSQL(cond, &paramCounter, &args)
	if err != nil {
		t.Fatalf("conditionToSQL failed: %v", err)
	}

	expectedSQL := "published_at IS NOT NULL"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}
}

func TestConditionToSQL_Between(t *testing.T) {
	cond := &Condition{
		Field:    "views",
		Operator: OpBetween,
		Value:    []interface{}{100, 1000},
	}

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := conditionToSQL(cond, &paramCounter, &args)
	if err != nil {
		t.Fatalf("conditionToSQL failed: %v", err)
	}

	expectedSQL := "views BETWEEN $1 AND $2"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}

	if len(args) != 2 {
		t.Fatalf("Expected 2 args, got %d", len(args))
	}

	if args[0] != 100 || args[1] != 1000 {
		t.Errorf("Expected args [100, 1000], got %v", args)
	}

	if paramCounter != 3 {
		t.Errorf("Expected paramCounter 3, got %d", paramCounter)
	}
}

func TestConditionToSQL_BetweenInvalid(t *testing.T) {
	cond := &Condition{
		Field:    "views",
		Operator: OpBetween,
		Value:    []interface{}{100}, // Only one value
	}

	paramCounter := 1
	args := make([]interface{}, 0)

	_, err := conditionToSQL(cond, &paramCounter, &args)
	if err == nil {
		t.Error("Expected error for invalid BETWEEN values")
	}
}

func TestPredicateGroup(t *testing.T) {
	group := NewPredicateGroup(false)

	group.AddCondition(&Condition{
		Field:    "status",
		Operator: OpEqual,
		Value:    "published",
	})

	group.AddCondition(&Condition{
		Field:    "views",
		Operator: OpGreaterThan,
		Value:    100,
	})

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := group.ToSQL(&paramCounter, &args)
	if err != nil {
		t.Fatalf("PredicateGroup.ToSQL failed: %v", err)
	}

	expectedSQL := "status = $1 AND views > $2"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
}

func TestPredicateGroup_Or(t *testing.T) {
	group := NewPredicateGroup(true)

	group.AddCondition(&Condition{
		Field:    "status",
		Operator: OpEqual,
		Value:    "published",
	})

	group.AddCondition(&Condition{
		Field:    "status",
		Operator: OpEqual,
		Value:    "draft",
	})

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := group.ToSQL(&paramCounter, &args)
	if err != nil {
		t.Fatalf("PredicateGroup.ToSQL failed: %v", err)
	}

	expectedSQL := "status = $1 OR status = $2"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}
}

func TestPredicateGroup_Nested(t *testing.T) {
	rootGroup := NewPredicateGroup(false)

	// (status = 'published' AND views > 100)
	rootGroup.AddCondition(&Condition{
		Field:    "status",
		Operator: OpEqual,
		Value:    "published",
	})

	// OR (status = 'featured' AND views > 50)
	nestedGroup := NewPredicateGroup(false)
	nestedGroup.AddCondition(&Condition{
		Field:    "status",
		Operator: OpEqual,
		Value:    "featured",
	})
	nestedGroup.AddCondition(&Condition{
		Field:    "views",
		Operator: OpGreaterThan,
		Value:    50,
	})

	rootGroup.AddGroup(nestedGroup)

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := rootGroup.ToSQL(&paramCounter, &args)
	if err != nil {
		t.Fatalf("PredicateGroup.ToSQL failed: %v", err)
	}

	// Should produce: status = $1 AND (status = $2 AND views > $3)
	if !strings.Contains(sql, "status = $1") {
		t.Error("Missing first condition")
	}
	if !strings.Contains(sql, "(status = $2 AND views > $3)") {
		t.Error("Missing nested group")
	}
}

func TestPredicateBuilder(t *testing.T) {
	builder := NewPredicateBuilder()

	builder.And("status", OpEqual, "published").
		And("views", OpGreaterThan, 100)

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := builder.ToSQL(&paramCounter, &args)
	if err != nil {
		t.Fatalf("PredicateBuilder.ToSQL failed: %v", err)
	}

	expectedSQL := "status = $1 AND views > $2"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}
}

func TestPredicateBuilder_Or(t *testing.T) {
	builder := NewPredicateBuilder()

	// The root group is AND by default, but individual conditions can be OR
	builder.And("status", OpEqual, "published")

	// Create a new builder with OR root for this test
	orBuilder := &PredicateBuilder{
		root: NewPredicateGroup(true),
	}
	orBuilder.Or("status", OpEqual, "published")
	orBuilder.Or("status", OpEqual, "draft")

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := orBuilder.ToSQL(&paramCounter, &args)
	if err != nil {
		t.Fatalf("PredicateBuilder.ToSQL failed: %v", err)
	}

	expectedSQL := "status = $1 OR status = $2"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}
}

func TestPredicateBuilder_AndGroup(t *testing.T) {
	builder := NewPredicateBuilder()

	builder.And("category", OpEqual, "tech").
		AndGroup(func(b *PredicateBuilder) {
			// Create OR group inside
			b.root.Or = true
			b.And("status", OpEqual, "published")
			b.And("status", OpEqual, "featured")
		})

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := builder.ToSQL(&paramCounter, &args)
	if err != nil {
		t.Fatalf("PredicateBuilder.ToSQL failed: %v", err)
	}

	// Should produce: category = $1 AND (status = $2 OR status = $3)
	if !strings.Contains(sql, "category = $1") {
		t.Error("Missing category condition")
	}
	// The nested group should be there
	if !strings.Contains(sql, "status = $2") || !strings.Contains(sql, "status = $3") {
		t.Error("Missing nested group conditions")
	}
}

func TestPredicateBuilder_OrGroup(t *testing.T) {
	builder := NewPredicateBuilder()

	builder.And("published", OpEqual, true).
		OrGroup(func(b *PredicateBuilder) {
			b.And("views", OpGreaterThan, 1000).
				And("featured", OpEqual, true)
		})

	paramCounter := 1
	args := make([]interface{}, 0)

	sql, err := builder.ToSQL(&paramCounter, &args)
	if err != nil {
		t.Fatalf("PredicateBuilder.ToSQL failed: %v", err)
	}

	// Should contain the group
	if !strings.Contains(sql, "published = $1") {
		t.Error("Missing published condition")
	}
}

// Benchmark tests
func BenchmarkConditionToSQL_Simple(b *testing.B) {
	cond := &Condition{
		Field:    "status",
		Operator: OpEqual,
		Value:    "published",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		paramCounter := 1
		args := make([]interface{}, 0)
		_, _ = conditionToSQL(cond, &paramCounter, &args)
	}
}

func BenchmarkConditionToSQL_In(b *testing.B) {
	cond := &Condition{
		Field:    "status",
		Operator: OpIn,
		Value:    []interface{}{"published", "draft", "archived"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		paramCounter := 1
		args := make([]interface{}, 0)
		_, _ = conditionToSQL(cond, &paramCounter, &args)
	}
}

func BenchmarkPredicateGroup_Simple(b *testing.B) {
	group := NewPredicateGroup(false)
	group.AddCondition(&Condition{
		Field:    "status",
		Operator: OpEqual,
		Value:    "published",
	})
	group.AddCondition(&Condition{
		Field:    "views",
		Operator: OpGreaterThan,
		Value:    100,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		paramCounter := 1
		args := make([]interface{}, 0)
		_, _ = group.ToSQL(&paramCounter, &args)
	}
}

func BenchmarkPredicateBuilder(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewPredicateBuilder()
		builder.And("status", OpEqual, "published").
			And("views", OpGreaterThan, 100).
			AndGroup(func(b *PredicateBuilder) {
				b.And("featured", OpEqual, true).
					Or("promoted", OpEqual, true)
			})

		paramCounter := 1
		args := make([]interface{}, 0)
		_, _ = builder.ToSQL(&paramCounter, &args)
	}
}
