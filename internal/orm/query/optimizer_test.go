package query

import (
	"testing"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func TestNewOptimizer(t *testing.T) {
	schemas := map[string]*schema.ResourceSchema{
		"Post": createTestResource(),
	}

	optimizer := NewOptimizer(schemas)

	if optimizer == nil {
		t.Fatal("NewOptimizer returned nil")
	}

	if optimizer.schemas == nil {
		t.Error("Optimizer schemas not set")
	}
}

func TestOptimizer_RemoveRedundantJoins(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)
	qb := NewQueryBuilder(resource, nil, schemas)

	// Add duplicate joins
	qb.InnerJoin("users", "users.id = posts.author_id")
	qb.InnerJoin("users", "users.id = posts.author_id") // Duplicate

	optimized := optimizer.removeRedundantJoins(qb)

	if len(optimized.joins) != 1 {
		t.Errorf("Expected 1 join after optimization, got %d", len(optimized.joins))
	}
}

func TestOptimizer_ScoreCondition(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)

	tests := []struct {
		name     string
		cond     *Condition
		maxScore int
	}{
		{
			name: "Equal is most selective",
			cond: &Condition{
				Field:    "id",
				Operator: OpEqual,
				Value:    "123",
			},
			maxScore: 2,
		},
		{
			name: "IN with small list is selective",
			cond: &Condition{
				Field:    "status",
				Operator: OpIn,
				Value:    []interface{}{"published", "draft"},
			},
			maxScore: 5,
		},
		{
			name: "LIKE is less selective",
			cond: &Condition{
				Field:    "title",
				Operator: OpLike,
				Value:    "%test%",
			},
			maxScore: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := optimizer.scoreCondition(tt.cond)
			if score > tt.maxScore {
				t.Errorf("Expected score <= %d, got %d", tt.maxScore, score)
			}
		})
	}
}

func TestOptimizer_ReorderConditions(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)
	qb := NewQueryBuilder(resource, nil, schemas)

	// Add conditions in order from least to most selective
	qb.Where("title", OpLike, "%test%")          // Less selective
	qb.Where("id", OpEqual, "123")               // Most selective
	qb.Where("status", OpIn, []interface{}{"published"}) // Moderately selective

	optimized := optimizer.reorderConditions(qb)

	// Most selective should be first (Equal)
	if optimized.conditions[0].Operator != OpEqual {
		t.Error("Most selective condition should be first after optimization")
	}
}

func TestOptimizer_EstimateCost(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)
	qb := NewQueryBuilder(resource, nil, schemas)

	qb.Where("status", OpEqual, "published")
	qb.OrderByDesc("created_at")
	qb.Limit(10)

	cost := optimizer.EstimateCost(qb)

	if cost == nil {
		t.Fatal("EstimateCost returned nil")
	}

	if cost.Cost <= 0 {
		t.Error("Cost should be positive")
	}

	if cost.IndexScans == 0 {
		t.Error("Should detect potential index usage")
	}
}

func TestOptimizer_EstimateCost_WithJoins(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)
	qb := NewQueryBuilder(resource, nil, schemas)

	qb.InnerJoin("users", "users.id = posts.author_id")
	qb.Where("status", OpEqual, "published")

	cost := optimizer.EstimateCost(qb)

	if cost.Joins != 1 {
		t.Errorf("Expected 1 join, got %d", cost.Joins)
	}

	// Cost should be higher with joins
	qbNoJoin := NewQueryBuilder(resource, nil, schemas)
	qbNoJoin.Where("status", OpEqual, "published")
	costNoJoin := optimizer.EstimateCost(qbNoJoin)

	if cost.Cost <= costNoJoin.Cost {
		t.Error("Cost with join should be higher")
	}
}

func TestOptimizer_Explain(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)
	qb := NewQueryBuilder(resource, nil, schemas)

	qb.Where("status", OpEqual, "published").
		OrderByDesc("created_at").
		Limit(10)

	plan, err := optimizer.Explain(qb)
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}

	if plan == nil {
		t.Fatal("Explain returned nil plan")
	}

	if plan.SQL == "" {
		t.Error("Plan should have SQL")
	}

	if plan.Cost == nil {
		t.Error("Plan should have cost estimate")
	}
}

func TestOptimizer_Analyze(t *testing.T) {
	resource := createTestResource()

	// Add index annotation to status field
	resource.Fields["status"].Annotations = append(resource.Fields["status"].Annotations,
		schema.Annotation{Name: "index"})

	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)
	qb := NewQueryBuilder(resource, nil, schemas)

	qb.Where("status", OpEqual, "published")

	analysis, err := optimizer.Analyze(qb)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if analysis == nil {
		t.Fatal("Analyze returned nil")
	}

	if !analysis.UsesIndex {
		t.Error("Should detect index usage on status field")
	}

	if analysis.ComplexityScore < 0 || analysis.ComplexityScore > 100 {
		t.Errorf("Complexity score should be 0-100, got %d", analysis.ComplexityScore)
	}
}

func TestOptimizer_Analyze_N1Warning(t *testing.T) {
	resource := createTestResource()

	// Add a relationship
	resource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
	}

	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)
	qb := NewQueryBuilder(resource, nil, schemas)

	// Query without eager loading
	qb.Where("status", OpEqual, "published")

	analysis, err := optimizer.Analyze(qb)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if !analysis.PotentialN1 {
		t.Error("Should warn about potential N+1 queries")
	}

	if len(analysis.Warnings) == 0 {
		t.Error("Should have warnings")
	}
}

func TestOptimizer_CalculateComplexity(t *testing.T) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)

	tests := []struct {
		name     string
		setup    func(*QueryBuilder)
		minScore int
	}{
		{
			name: "Simple query",
			setup: func(qb *QueryBuilder) {
				qb.Where("status", OpEqual, "published")
			},
			minScore: 0,
		},
		{
			name: "Complex query",
			setup: func(qb *QueryBuilder) {
				qb.Where("status", OpEqual, "published")
				qb.InnerJoin("users", "users.id = posts.author_id")
				qb.InnerJoin("categories", "categories.id = posts.category_id")
				qb.GroupBy("status")
				qb.Having("COUNT(*)", OpGreaterThan, 5)
				qb.OrderByDesc("created_at")
			},
			minScore: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewQueryBuilder(resource, nil, schemas)
			tt.setup(qb)

			score := optimizer.calculateComplexity(qb)
			if score < tt.minScore {
				t.Errorf("Expected complexity >= %d, got %d", tt.minScore, score)
			}
			if score > 100 {
				t.Errorf("Complexity should be capped at 100, got %d", score)
			}
		})
	}
}

func TestOptimizer_ShouldUseIndex(t *testing.T) {
	resource := createTestResource()

	// Add index to status field
	resource.Fields["status"].Annotations = append(resource.Fields["status"].Annotations,
		schema.Annotation{Name: "index"})

	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)

	tests := []struct {
		name     string
		cond     *Condition
		expected bool
	}{
		{
			name: "Indexed field with equality",
			cond: &Condition{
				Field:    "status",
				Operator: OpEqual,
				Value:    "published",
			},
			expected: true,
		},
		{
			name: "Indexed field with range",
			cond: &Condition{
				Field:    "status",
				Operator: OpGreaterThan,
				Value:    "a",
			},
			expected: true,
		},
		{
			name: "Non-indexed field",
			cond: &Condition{
				Field:    "title",
				Operator: OpEqual,
				Value:    "test",
			},
			expected: false,
		},
		{
			name: "Indexed field with LIKE",
			cond: &Condition{
				Field:    "status",
				Operator: OpLike,
				Value:    "%test%",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := optimizer.ShouldUseIndex(tt.cond, resource)
			if result != tt.expected {
				t.Errorf("ShouldUseIndex() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestOptimizer_GenerateRecommendations(t *testing.T) {
	resource := createTestResource()

	// Add a relationship
	resource.Relationships["author"] = &schema.Relationship{
		Type:           schema.RelationshipBelongsTo,
		TargetResource: "User",
		FieldName:      "author",
	}

	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)
	qb := NewQueryBuilder(resource, nil, schemas)

	// Simple query without optimization
	qb.Where("title", OpLike, "%test%")

	analysis := &QueryAnalysis{
		UsesIndex:   false,
		PotentialN1: true,
	}

	recommendations := optimizer.generateRecommendations(qb, analysis)

	if len(recommendations) == 0 {
		t.Error("Should have recommendations")
	}

	// Should recommend indexes
	hasIndexRec := false
	for _, rec := range recommendations {
		if contains(rec, "index") {
			hasIndexRec = true
			break
		}
	}

	if !hasIndexRec {
		t.Error("Should recommend adding indexes")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkOptimizer_Optimize(b *testing.B) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)
	qb := NewQueryBuilder(resource, nil, schemas)
	qb.Where("status", OpEqual, "published").
		Where("views", OpGreaterThan, 100).
		InnerJoin("users", "users.id = posts.author_id").
		OrderByDesc("created_at")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = optimizer.Optimize(qb)
	}
}

func BenchmarkOptimizer_EstimateCost(b *testing.B) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)
	qb := NewQueryBuilder(resource, nil, schemas)
	qb.Where("status", OpEqual, "published").
		OrderByDesc("created_at")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = optimizer.EstimateCost(qb)
	}
}

func BenchmarkOptimizer_Analyze(b *testing.B) {
	resource := createTestResource()
	schemas := map[string]*schema.ResourceSchema{
		"Post": resource,
	}

	optimizer := NewOptimizer(schemas)
	qb := NewQueryBuilder(resource, nil, schemas)
	qb.Where("status", OpEqual, "published")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = optimizer.Analyze(qb)
	}
}
