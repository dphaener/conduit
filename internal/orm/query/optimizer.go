// Package query provides query optimization functionality
package query

import (
	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// Optimizer optimizes query execution plans
type Optimizer struct {
	schemas map[string]*schema.ResourceSchema
}

// NewOptimizer creates a new query optimizer
func NewOptimizer(schemas map[string]*schema.ResourceSchema) *Optimizer {
	return &Optimizer{
		schemas: schemas,
	}
}

// Optimize optimizes a query builder
func (o *Optimizer) Optimize(qb *QueryBuilder) *QueryBuilder {
	optimized := qb.Clone()

	// Optimization pass 1: Remove redundant joins
	optimized = o.removeRedundantJoins(optimized)

	// Optimization pass 2: Push predicates down
	optimized = o.pushPredicatesDown(optimized)

	// Optimization pass 3: Reorder conditions (most selective first)
	optimized = o.reorderConditions(optimized)

	// Optimization pass 4: Optimize eager loading
	optimized = o.optimizeEagerLoading(optimized)

	return optimized
}

// removeRedundantJoins removes duplicate or unnecessary joins
func (o *Optimizer) removeRedundantJoins(qb *QueryBuilder) *QueryBuilder {
	if len(qb.joins) <= 1 {
		return qb
	}

	// Track unique joins by table name
	seen := make(map[string]bool)
	unique := make([]*Join, 0)

	for _, join := range qb.joins {
		key := join.Table
		if !seen[key] {
			seen[key] = true
			unique = append(unique, join)
		}
	}

	qb.joins = unique
	return qb
}

// pushPredicatesDown attempts to push WHERE predicates into JOINs where possible
// This can improve performance by filtering earlier in the query execution
func (o *Optimizer) pushPredicatesDown(qb *QueryBuilder) *QueryBuilder {
	// For now, this is a placeholder for a more sophisticated implementation
	// In practice, this would analyze which conditions reference joined tables
	// and potentially convert them to JOIN conditions
	return qb
}

// reorderConditions reorders WHERE conditions to put most selective ones first
// This is a heuristic-based optimization
func (o *Optimizer) reorderConditions(qb *QueryBuilder) *QueryBuilder {
	if len(qb.conditions) <= 1 {
		return qb
	}

	// Score each condition by selectivity (lower is more selective)
	type scoredCondition struct {
		condition *Condition
		score     int
	}

	scored := make([]scoredCondition, 0, len(qb.conditions))
	for _, cond := range qb.conditions {
		score := o.scoreCondition(cond)
		scored = append(scored, scoredCondition{cond, score})
	}

	// Sort by score (most selective first)
	// We use a simple bubble sort for stability and small arrays
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score < scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Rebuild conditions array
	reordered := make([]*Condition, 0, len(scored))
	for _, sc := range scored {
		reordered = append(reordered, sc.condition)
	}

	qb.conditions = reordered
	return qb
}

// scoreCondition assigns a selectivity score to a condition
// Lower scores are more selective (better to evaluate first)
func (o *Optimizer) scoreCondition(cond *Condition) int {
	switch cond.Operator {
	case OpEqual:
		// Equality is very selective
		return 1
	case OpIn:
		// IN with small lists is selective
		if values, ok := cond.Value.([]interface{}); ok {
			if len(values) <= 3 {
				return 2
			}
			return 4
		}
		return 4
	case OpIsNull, OpIsNotNull:
		// NULL checks can be selective depending on data distribution
		return 3
	case OpBetween:
		// BETWEEN is moderately selective
		return 5
	case OpGreaterThan, OpGreaterThanOrEqual, OpLessThan, OpLessThanOrEqual:
		// Range queries are less selective
		return 6
	case OpLike, OpILike:
		// Pattern matching is expensive
		return 8
	case OpNotEqual, OpNotIn:
		// Negative conditions are least selective
		return 10
	default:
		return 5
	}
}

// optimizeEagerLoading optimizes the order of eager loaded relationships
func (o *Optimizer) optimizeEagerLoading(qb *QueryBuilder) *QueryBuilder {
	if len(qb.includes) <= 1 {
		return qb
	}

	// For now, this is a placeholder
	// In practice, this would reorder includes to:
	// 1. Load belongs_to relationships first (they're referenced by foreign keys)
	// 2. Then has_one relationships
	// 3. Finally has_many relationships

	return qb
}

// EstimateCost estimates the query execution cost
// This is a simplified heuristic for testing and debugging
type QueryCost struct {
	TableScans    int     // Number of table scans
	IndexScans    int     // Number of index scans
	Joins         int     // Number of joins
	Sorts         int     // Number of sorts
	Aggregations  int     // Number of aggregations
	EstimatedRows int     // Estimated number of rows returned
	Cost          float64 // Total estimated cost
}

// EstimateCost estimates the execution cost of a query
func (o *Optimizer) EstimateCost(qb *QueryBuilder) *QueryCost {
	cost := &QueryCost{}

	// Assume table scan by default
	cost.TableScans = 1
	cost.EstimatedRows = 1000 // Default estimate

	// Check for index usage
	for _, cond := range qb.conditions {
		switch cond.Operator {
		case OpEqual, OpIn:
			// These can use indexes
			cost.IndexScans++
			cost.EstimatedRows = cost.EstimatedRows / 10 // Reduce estimated rows
		}
	}

	// Count joins
	cost.Joins = len(qb.joins)

	// Count sorts
	if len(qb.orderBy) > 0 {
		cost.Sorts = 1
	}

	// Count aggregations
	if len(qb.groupBy) > 0 {
		cost.Aggregations = 1
	}

	// Calculate total cost (simplified)
	cost.Cost = float64(cost.TableScans*100 +
		cost.IndexScans*10 +
		cost.Joins*50 +
		cost.Sorts*20 +
		cost.Aggregations*30)

	// Adjust for estimated rows
	cost.Cost *= float64(cost.EstimatedRows) / 1000.0

	return cost
}

// QueryPlan represents a query execution plan
type QueryPlan struct {
	SQL           string
	Args          []interface{}
	Cost          *QueryCost
	Optimizations []string // List of applied optimizations
}

// Explain generates an execution plan for the query
func (o *Optimizer) Explain(qb *QueryBuilder) (*QueryPlan, error) {
	optimized := o.Optimize(qb)

	sql, args, err := optimized.ToSQL()
	if err != nil {
		return nil, err
	}

	cost := o.EstimateCost(optimized)

	plan := &QueryPlan{
		SQL:           sql,
		Args:          args,
		Cost:          cost,
		Optimizations: o.getOptimizations(qb, optimized),
	}

	return plan, nil
}

// getOptimizations returns a list of optimizations applied
func (o *Optimizer) getOptimizations(original, optimized *QueryBuilder) []string {
	opts := make([]string, 0)

	// Check if joins were removed
	if len(original.joins) > len(optimized.joins) {
		opts = append(opts, "Removed redundant joins")
	}

	// Check if conditions were reordered
	if len(original.conditions) > 1 && len(optimized.conditions) > 1 {
		if original.conditions[0] != optimized.conditions[0] {
			opts = append(opts, "Reordered conditions for selectivity")
		}
	}

	return opts
}

// ShouldUseIndex determines if an index should be used for a condition
func (o *Optimizer) ShouldUseIndex(cond *Condition, resource *schema.ResourceSchema) bool {
	// Check if the field has an index annotation
	field, ok := resource.Fields[cond.Field]
	if !ok {
		return false
	}

	for _, annotation := range field.Annotations {
		if annotation.Name == "index" || annotation.Name == "unique" {
			// Index can be used for equality, range, and IN queries
			switch cond.Operator {
			case OpEqual, OpIn, OpGreaterThan, OpGreaterThanOrEqual,
				OpLessThan, OpLessThanOrEqual, OpBetween:
				return true
			}
		}
	}

	return false
}

// AnalyzeQuery provides detailed analysis of a query
type QueryAnalysis struct {
	SQL               string
	ParameterCount    int
	ConditionCount    int
	JoinCount         int
	UsesIndex         bool
	PotentialN1       bool // Might cause N+1 queries
	ComplexityScore   int  // 0-100, higher is more complex
	Warnings          []string
	Recommendations   []string
}

// Analyze performs detailed analysis of a query
func (o *Optimizer) Analyze(qb *QueryBuilder) (*QueryAnalysis, error) {
	sql, args, err := qb.ToSQL()
	if err != nil {
		return nil, err
	}

	analysis := &QueryAnalysis{
		SQL:            sql,
		ParameterCount: len(args),
		ConditionCount: len(qb.conditions),
		JoinCount:      len(qb.joins),
		Warnings:       make([]string, 0),
		Recommendations: make([]string, 0),
	}

	// Check for index usage
	for _, cond := range qb.conditions {
		if o.ShouldUseIndex(cond, qb.resource) {
			analysis.UsesIndex = true
			break
		}
	}

	// Check for potential N+1 queries
	if len(qb.includes) == 0 && len(qb.resource.Relationships) > 0 {
		analysis.PotentialN1 = true
		analysis.Warnings = append(analysis.Warnings,
			"No eager loading configured - may cause N+1 queries")
	}

	// Calculate complexity score
	analysis.ComplexityScore = o.calculateComplexity(qb)

	// Generate recommendations
	analysis.Recommendations = o.generateRecommendations(qb, analysis)

	return analysis, nil
}

// calculateComplexity calculates a complexity score for the query
func (o *Optimizer) calculateComplexity(qb *QueryBuilder) int {
	score := 0

	// Conditions add complexity
	score += len(qb.conditions) * 5

	// Joins add significant complexity
	score += len(qb.joins) * 15

	// Grouping adds complexity
	score += len(qb.groupBy) * 10

	// Having clauses add complexity
	score += len(qb.having) * 8

	// Ordering adds minor complexity
	score += len(qb.orderBy) * 3

	// Eager loading adds complexity
	score += len(qb.includes) * 12

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

// generateRecommendations generates optimization recommendations
func (o *Optimizer) generateRecommendations(qb *QueryBuilder, analysis *QueryAnalysis) []string {
	recommendations := make([]string, 0)

	// Recommend indexes
	if !analysis.UsesIndex && len(qb.conditions) > 0 {
		recommendations = append(recommendations,
			"Consider adding indexes on frequently queried fields")
	}

	// Recommend eager loading
	if analysis.PotentialN1 {
		recommendations = append(recommendations,
			"Use .Includes() to eager load relationships and prevent N+1 queries")
	}

	// Recommend pagination
	if qb.limit == nil && qb.offset == nil {
		recommendations = append(recommendations,
			"Consider adding pagination with Limit() and Offset()")
	}

	// Warn about complex queries
	if analysis.ComplexityScore > 70 {
		recommendations = append(recommendations,
			"Query is complex - consider breaking into smaller queries or using views")
	}

	return recommendations
}
