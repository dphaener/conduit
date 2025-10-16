package errors

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Optimization hint codes (OPT700-799) - informational suggestions
const (
	// ErrMissingIndex indicates a field that could benefit from an index
	ErrMissingIndex ErrorCode = "OPT700"
	// ErrIneffectiveQuery indicates a query that could be optimized
	ErrIneffectiveQuery ErrorCode = "OPT701"
	// ErrNPlusOneQuery indicates potential N+1 query problem
	ErrNPlusOneQuery ErrorCode = "OPT702"
	// ErrLargePayload indicates a large response payload
	ErrLargePayload ErrorCode = "OPT703"
	// ErrUnusedEagerLoading indicates eager loading that's not used
	ErrUnusedEagerLoading ErrorCode = "OPT704"
	// ErrMissingCaching indicates an opportunity for caching
	ErrMissingCaching ErrorCode = "OPT705"
	// ErrIneffectiveIndex indicates an index that's not being used
	ErrIneffectiveIndex ErrorCode = "OPT706"
	// ErrSlowFunction indicates a function with high execution time
	ErrSlowFunction ErrorCode = "OPT707"
	// ErrMemoryIntensive indicates memory-intensive operation
	ErrMemoryIntensive ErrorCode = "OPT708"
)

// NewMissingIndex creates an OPT700 info message
func NewMissingIndex(loc ast.SourceLocation, field, resource string) *CompilerError {
	return newError(
		ErrMissingIndex,
		"missing_index",
		CategoryOptimization,
		SeverityInfo,
		fmt.Sprintf("Field '%s' in %s could benefit from a database index", field, resource),
		loc,
	).WithSuggestion("Add @index annotation if this field is frequently queried").
		WithExamples(
			fmt.Sprintf("%s: string! @index", field),
		)
}

// NewIneffectiveQuery creates an OPT701 info message
func NewIneffectiveQuery(loc ast.SourceLocation, reason string) *CompilerError {
	return newError(
		ErrIneffectiveQuery,
		"ineffective_query",
		CategoryOptimization,
		SeverityInfo,
		fmt.Sprintf("Query could be optimized: %s", reason),
		loc,
	).WithSuggestion("Consider using select fields, filters, or eager loading")
}

// NewNPlusOneQuery creates an OPT702 warning
func NewNPlusOneQuery(loc ast.SourceLocation, relationship string) *CompilerError {
	return newError(
		ErrNPlusOneQuery,
		"n_plus_one_query",
		CategoryOptimization,
		SeverityWarning,
		fmt.Sprintf("Potential N+1 query detected accessing relationship '%s'", relationship),
		loc,
	).WithSuggestion("Use eager loading with .include() to load related records in a single query").
		WithExamples(
			fmt.Sprintf("Post.query().include(%q).all()", relationship),
		)
}

// NewLargePayload creates an OPT703 info message
func NewLargePayload(loc ast.SourceLocation, size string) *CompilerError {
	return newError(
		ErrLargePayload,
		"large_payload",
		CategoryOptimization,
		SeverityInfo,
		fmt.Sprintf("Response payload is large (%s)", size),
		loc,
	).WithSuggestion("Consider pagination, field selection, or compression")
}

// NewUnusedEagerLoading creates an OPT704 info message
func NewUnusedEagerLoading(loc ast.SourceLocation, relationship string) *CompilerError {
	return newError(
		ErrUnusedEagerLoading,
		"unused_eager_loading",
		CategoryOptimization,
		SeverityInfo,
		fmt.Sprintf("Eager loaded relationship '%s' is not used", relationship),
		loc,
	).WithSuggestion("Remove unused .include() calls to improve performance")
}

// NewMissingCaching creates an OPT705 info message
func NewMissingCaching(loc ast.SourceLocation, operation string) *CompilerError {
	return newError(
		ErrMissingCaching,
		"missing_caching",
		CategoryOptimization,
		SeverityInfo,
		fmt.Sprintf("Operation '%s' could benefit from caching", operation),
		loc,
	).WithSuggestion("Consider adding caching for frequently accessed data")
}

// NewIneffectiveIndex creates an OPT706 info message
func NewIneffectiveIndex(loc ast.SourceLocation, field, reason string) *CompilerError {
	return newError(
		ErrIneffectiveIndex,
		"ineffective_index",
		CategoryOptimization,
		SeverityInfo,
		fmt.Sprintf("Index on '%s' is not being used effectively: %s", field, reason),
		loc,
	).WithSuggestion("Review query patterns or consider removing the index")
}

// NewSlowFunction creates an OPT707 warning
func NewSlowFunction(loc ast.SourceLocation, function, duration string) *CompilerError {
	return newError(
		ErrSlowFunction,
		"slow_function",
		CategoryOptimization,
		SeverityWarning,
		fmt.Sprintf("Function '%s' has high execution time (%s)", function, duration),
		loc,
	).WithSuggestion("Profile and optimize the function or move it to @async block")
}

// NewMemoryIntensive creates an OPT708 warning
func NewMemoryIntensive(loc ast.SourceLocation, operation, usage string) *CompilerError {
	return newError(
		ErrMemoryIntensive,
		"memory_intensive",
		CategoryOptimization,
		SeverityWarning,
		fmt.Sprintf("Operation '%s' uses significant memory (%s)", operation, usage),
		loc,
	).WithSuggestion("Consider pagination, streaming, or batch processing")
}
