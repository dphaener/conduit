// Package query provides predicate construction for WHERE clauses
package query

import (
	"fmt"
	"strings"
)

// Operator represents a comparison operator
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

// String returns the string representation of the operator
func (o Operator) String() string {
	switch o {
	case OpEqual:
		return "="
	case OpNotEqual:
		return "!="
	case OpGreaterThan:
		return ">"
	case OpGreaterThanOrEqual:
		return ">="
	case OpLessThan:
		return "<"
	case OpLessThanOrEqual:
		return "<="
	case OpIn:
		return "IN"
	case OpNotIn:
		return "NOT IN"
	case OpLike:
		return "LIKE"
	case OpILike:
		return "ILIKE"
	case OpIsNull:
		return "IS NULL"
	case OpIsNotNull:
		return "IS NOT NULL"
	case OpBetween:
		return "BETWEEN"
	default:
		return "UNKNOWN"
	}
}

// Condition represents a WHERE condition
type Condition struct {
	Field    string
	Operator Operator
	Value    interface{}
	Or       bool // true for OR, false for AND
}

// PredicateGroup represents a group of predicates combined with AND/OR
type PredicateGroup struct {
	Conditions []*Condition
	Groups     []*PredicateGroup
	Or         bool // true for OR, false for AND
}

// NewPredicateGroup creates a new predicate group
func NewPredicateGroup(or bool) *PredicateGroup {
	return &PredicateGroup{
		Conditions: make([]*Condition, 0),
		Groups:     make([]*PredicateGroup, 0),
		Or:         or,
	}
}

// AddCondition adds a condition to the group
func (pg *PredicateGroup) AddCondition(cond *Condition) {
	pg.Conditions = append(pg.Conditions, cond)
}

// AddGroup adds a nested group
func (pg *PredicateGroup) AddGroup(group *PredicateGroup) {
	pg.Groups = append(pg.Groups, group)
}

// ToSQL converts the predicate group to SQL
func (pg *PredicateGroup) ToSQL(paramCounter *int, args *[]interface{}) (string, error) {
	if len(pg.Conditions) == 0 && len(pg.Groups) == 0 {
		return "", nil
	}

	parts := make([]string, 0)

	// Add conditions
	for _, cond := range pg.Conditions {
		sql, err := conditionToSQL(cond, paramCounter, args)
		if err != nil {
			return "", err
		}
		parts = append(parts, sql)
	}

	// Add nested groups
	for _, group := range pg.Groups {
		sql, err := group.ToSQL(paramCounter, args)
		if err != nil {
			return "", err
		}
		if sql != "" {
			parts = append(parts, fmt.Sprintf("(%s)", sql))
		}
	}

	if len(parts) == 0 {
		return "", nil
	}

	connector := " AND "
	if pg.Or {
		connector = " OR "
	}

	return strings.Join(parts, connector), nil
}

// conditionToSQL converts a condition to SQL with parameterized values
func conditionToSQL(cond *Condition, paramCounter *int, args *[]interface{}) (string, error) {
	switch cond.Operator {
	case OpEqual:
		*args = append(*args, cond.Value)
		sql := fmt.Sprintf("%s = $%d", cond.Field, *paramCounter)
		*paramCounter++
		return sql, nil

	case OpNotEqual:
		*args = append(*args, cond.Value)
		sql := fmt.Sprintf("%s != $%d", cond.Field, *paramCounter)
		*paramCounter++
		return sql, nil

	case OpGreaterThan:
		*args = append(*args, cond.Value)
		sql := fmt.Sprintf("%s > $%d", cond.Field, *paramCounter)
		*paramCounter++
		return sql, nil

	case OpGreaterThanOrEqual:
		*args = append(*args, cond.Value)
		sql := fmt.Sprintf("%s >= $%d", cond.Field, *paramCounter)
		*paramCounter++
		return sql, nil

	case OpLessThan:
		*args = append(*args, cond.Value)
		sql := fmt.Sprintf("%s < $%d", cond.Field, *paramCounter)
		*paramCounter++
		return sql, nil

	case OpLessThanOrEqual:
		*args = append(*args, cond.Value)
		sql := fmt.Sprintf("%s <= $%d", cond.Field, *paramCounter)
		*paramCounter++
		return sql, nil

	case OpIn:
		// Convert slice to PostgreSQL array format
		values, ok := cond.Value.([]interface{})
		if !ok {
			return "", fmt.Errorf("IN operator requires []interface{} value")
		}
		if len(values) == 0 {
			// IN with empty array always returns false
			return "FALSE", nil
		}

		placeholders := make([]string, len(values))
		for i, v := range values {
			*args = append(*args, v)
			placeholders[i] = fmt.Sprintf("$%d", *paramCounter)
			*paramCounter++
		}
		return fmt.Sprintf("%s IN (%s)", cond.Field, strings.Join(placeholders, ", ")), nil

	case OpNotIn:
		// Convert slice to PostgreSQL array format
		values, ok := cond.Value.([]interface{})
		if !ok {
			return "", fmt.Errorf("NOT IN operator requires []interface{} value")
		}
		if len(values) == 0 {
			// NOT IN with empty array always returns true
			return "TRUE", nil
		}

		placeholders := make([]string, len(values))
		for i, v := range values {
			*args = append(*args, v)
			placeholders[i] = fmt.Sprintf("$%d", *paramCounter)
			*paramCounter++
		}
		return fmt.Sprintf("%s NOT IN (%s)", cond.Field, strings.Join(placeholders, ", ")), nil

	case OpLike:
		*args = append(*args, cond.Value)
		sql := fmt.Sprintf("%s LIKE $%d", cond.Field, *paramCounter)
		*paramCounter++
		return sql, nil

	case OpILike:
		*args = append(*args, cond.Value)
		sql := fmt.Sprintf("%s ILIKE $%d", cond.Field, *paramCounter)
		*paramCounter++
		return sql, nil

	case OpIsNull:
		return fmt.Sprintf("%s IS NULL", cond.Field), nil

	case OpIsNotNull:
		return fmt.Sprintf("%s IS NOT NULL", cond.Field), nil

	case OpBetween:
		values, ok := cond.Value.([]interface{})
		if !ok || len(values) != 2 {
			return "", fmt.Errorf("BETWEEN operator requires [min, max] values")
		}
		*args = append(*args, values[0], values[1])
		sql := fmt.Sprintf("%s BETWEEN $%d AND $%d", cond.Field, *paramCounter, *paramCounter+1)
		*paramCounter += 2
		return sql, nil

	default:
		return "", fmt.Errorf("unsupported operator: %v", cond.Operator)
	}
}

// PredicateBuilder provides a fluent API for building complex predicates
type PredicateBuilder struct {
	root *PredicateGroup
}

// NewPredicateBuilder creates a new predicate builder
func NewPredicateBuilder() *PredicateBuilder {
	return &PredicateBuilder{
		root: NewPredicateGroup(false), // Default to AND
	}
}

// And adds an AND condition
func (pb *PredicateBuilder) And(field string, op Operator, value interface{}) *PredicateBuilder {
	pb.root.AddCondition(&Condition{
		Field:    field,
		Operator: op,
		Value:    value,
		Or:       false,
	})
	return pb
}

// Or adds an OR condition
func (pb *PredicateBuilder) Or(field string, op Operator, value interface{}) *PredicateBuilder {
	pb.root.AddCondition(&Condition{
		Field:    field,
		Operator: op,
		Value:    value,
		Or:       true,
	})
	return pb
}

// AndGroup adds an AND group
func (pb *PredicateBuilder) AndGroup(fn func(*PredicateBuilder)) *PredicateBuilder {
	group := NewPredicateGroup(false)
	builder := &PredicateBuilder{root: group}
	fn(builder)
	pb.root.AddGroup(group)
	return pb
}

// OrGroup adds an OR group
func (pb *PredicateBuilder) OrGroup(fn func(*PredicateBuilder)) *PredicateBuilder {
	group := NewPredicateGroup(true)
	builder := &PredicateBuilder{root: group}
	fn(builder)
	pb.root.AddGroup(group)
	return pb
}

// ToSQL converts the predicate builder to SQL
func (pb *PredicateBuilder) ToSQL(paramCounter *int, args *[]interface{}) (string, error) {
	return pb.root.ToSQL(paramCounter, args)
}

// ValidateField validates that a field exists in the resource schema
func ValidateField(field string, fields map[string]*interface{}) error {
	if _, exists := fields[field]; !exists {
		return fmt.Errorf("unknown field: %s", field)
	}
	return nil
}

// ValidateOperator validates that an operator is compatible with a field type
func ValidateOperator(op Operator, fieldType string) error {
	switch op {
	case OpLike, OpILike:
		if fieldType != "string" && fieldType != "text" {
			return fmt.Errorf("operator %s only works with text fields", op.String())
		}
	case OpBetween:
		if fieldType != "int" && fieldType != "float" && fieldType != "decimal" && fieldType != "timestamp" && fieldType != "date" {
			return fmt.Errorf("operator %s only works with numeric or date fields", op.String())
		}
	}
	return nil
}
