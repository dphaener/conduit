package errors

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Semantic error codes (SEM200-299)
const (
	// ErrUndefinedVariable indicates an undefined variable was referenced
	ErrUndefinedVariable ErrorCode = "SEM200"
	// ErrUndefinedFunction indicates an undefined function was called
	ErrUndefinedFunction ErrorCode = "SEM201"
	// ErrUndefinedType indicates an undefined type was referenced
	ErrUndefinedType ErrorCode = "SEM202"
	// ErrUndefinedField indicates an undefined field was accessed
	ErrUndefinedField ErrorCode = "SEM203"
	// ErrUndefinedResource indicates an undefined resource was referenced
	ErrUndefinedResource ErrorCode = "SEM204"
	// ErrCircularDependency indicates a circular dependency between resources
	ErrCircularDependency ErrorCode = "SEM205"
	// ErrRedeclaredVariable indicates a variable was declared twice
	ErrRedeclaredVariable ErrorCode = "SEM206"
	// ErrRedeclaredResource indicates a resource was declared twice
	ErrRedeclaredResource ErrorCode = "SEM207"
	// ErrInvalidSelfReference indicates invalid use of 'self' keyword
	ErrInvalidSelfReference ErrorCode = "SEM208"
	// ErrInvalidReturnContext indicates return used outside function
	ErrInvalidReturnContext ErrorCode = "SEM209"
	// ErrMissingReturn indicates a function is missing return statement
	ErrMissingReturn ErrorCode = "SEM210"
	// ErrInvalidBreakContext indicates break used outside loop
	ErrInvalidBreakContext ErrorCode = "SEM211"
	// ErrInvalidContinueContext indicates continue used outside loop
	ErrInvalidContinueContext ErrorCode = "SEM212"
	// ErrUnreachableCode indicates code after return/break/continue
	ErrUnreachableCode ErrorCode = "SEM213"
	// ErrInvalidAssignmentTarget indicates invalid left side of assignment
	ErrInvalidAssignmentTarget ErrorCode = "SEM214"
	// ErrConstantReassignment indicates reassignment of constant
	ErrConstantReassignment ErrorCode = "SEM215"
	// ErrInvalidHookContext indicates hook used in invalid context
	ErrInvalidHookContext ErrorCode = "SEM216"
	// ErrInvalidAsyncContext indicates async block used in invalid context
	ErrInvalidAsyncContext ErrorCode = "SEM217"
	// ErrInvalidTransactionContext indicates transaction annotation used incorrectly
	ErrInvalidTransactionContext ErrorCode = "SEM218"
)

// NewUndefinedVariable creates a SEM200 error
func NewUndefinedVariable(loc ast.SourceLocation, name string) *CompilerError {
	return newError(
		ErrUndefinedVariable,
		"undefined_variable",
		CategorySemantic,
		SeverityError,
		fmt.Sprintf("Undefined variable '%s'", name),
		loc,
	).WithSuggestion("Declare the variable with 'let' before using it")
}

// NewUndefinedFunction creates a SEM201 error
func NewUndefinedFunction(loc ast.SourceLocation, namespace, function string) *CompilerError {
	var funcName string
	if namespace != "" {
		funcName = namespace + "." + function
	} else {
		funcName = function
	}

	suggestion := "Check the function name and namespace"
	if namespace == "" {
		suggestion = "Custom functions must be defined with @function. Use namespaced stdlib functions (e.g., String.slugify())"
	}

	return newError(
		ErrUndefinedFunction,
		"undefined_function",
		CategorySemantic,
		SeverityError,
		fmt.Sprintf("Undefined function '%s'", funcName),
		loc,
	).WithSuggestion(suggestion)
}

// NewUndefinedType creates a SEM202 error
func NewUndefinedType(loc ast.SourceLocation, typeName string) *CompilerError {
	return newError(
		ErrUndefinedType,
		"undefined_type",
		CategorySemantic,
		SeverityError,
		fmt.Sprintf("Undefined type '%s'", typeName),
		loc,
	).WithSuggestion("Check for typos or ensure the resource/type is defined")
}

// NewUndefinedField creates a SEM203 error
func NewUndefinedField(loc ast.SourceLocation, fieldName, typeName string) *CompilerError {
	return newError(
		ErrUndefinedField,
		"undefined_field",
		CategorySemantic,
		SeverityError,
		fmt.Sprintf("Type '%s' has no field named '%s'", typeName, fieldName),
		loc,
	).WithSuggestion("Check field name spelling or add the field to the resource")
}

// NewUndefinedResource creates a SEM204 error
func NewUndefinedResource(loc ast.SourceLocation, resourceName string) *CompilerError {
	return newError(
		ErrUndefinedResource,
		"undefined_resource",
		CategorySemantic,
		SeverityError,
		fmt.Sprintf("Undefined resource '%s'", resourceName),
		loc,
	).WithSuggestion("Ensure the resource is defined in a .cdt file")
}

// NewCircularDependency creates a SEM205 error
func NewCircularDependency(loc ast.SourceLocation, resources []string) *CompilerError {
	cycle := ""
	for i, res := range resources {
		if i > 0 {
			cycle += " -> "
		}
		cycle += res
	}

	return newError(
		ErrCircularDependency,
		"circular_dependency",
		CategorySemantic,
		SeverityError,
		fmt.Sprintf("Circular dependency detected: %s", cycle),
		loc,
	).WithSuggestion("Break the circular dependency by using nullable relationships or restructuring resources")
}

// NewRedeclaredVariable creates a SEM206 error
func NewRedeclaredVariable(loc ast.SourceLocation, name string) *CompilerError {
	return newError(
		ErrRedeclaredVariable,
		"redeclared_variable",
		CategorySemantic,
		SeverityError,
		fmt.Sprintf("Variable '%s' is already declared in this scope", name),
		loc,
	).WithSuggestion("Use a different variable name or reassign the existing variable")
}

// NewRedeclaredResource creates a SEM207 error
func NewRedeclaredResource(loc ast.SourceLocation, name string) *CompilerError {
	return newError(
		ErrRedeclaredResource,
		"redeclared_resource",
		CategorySemantic,
		SeverityError,
		fmt.Sprintf("Resource '%s' is already declared", name),
		loc,
	).WithSuggestion("Use a different resource name or remove the duplicate definition")
}

// NewInvalidSelfReference creates a SEM208 error
func NewInvalidSelfReference(loc ast.SourceLocation, context string) *CompilerError {
	message := "Invalid use of 'self' keyword"
	if context != "" {
		message = fmt.Sprintf("Invalid use of 'self' in %s", context)
	}

	return newError(
		ErrInvalidSelfReference,
		"invalid_self_reference",
		CategorySemantic,
		SeverityError,
		message,
		loc,
	).WithSuggestion("'self' can only be used inside resource hooks, validations, and computed fields")
}

// NewInvalidReturnContext creates a SEM209 error
func NewInvalidReturnContext(loc ast.SourceLocation) *CompilerError {
	return newError(
		ErrInvalidReturnContext,
		"invalid_return_context",
		CategorySemantic,
		SeverityError,
		"Return statement outside of function",
		loc,
	).WithSuggestion("Return can only be used inside functions and computed fields")
}

// NewMissingReturn creates a SEM210 warning
func NewMissingReturn(loc ast.SourceLocation, functionName string) *CompilerError {
	return newError(
		ErrMissingReturn,
		"missing_return",
		CategorySemantic,
		SeverityWarning,
		fmt.Sprintf("Function '%s' may not return a value in all code paths", functionName),
		loc,
	).WithSuggestion("Ensure all code paths return a value")
}

// NewInvalidBreakContext creates a SEM211 error
func NewInvalidBreakContext(loc ast.SourceLocation) *CompilerError {
	return newError(
		ErrInvalidBreakContext,
		"invalid_break_context",
		CategorySemantic,
		SeverityError,
		"Break statement outside of loop or match statement",
		loc,
	).WithSuggestion("Break can only be used inside loops and match statements")
}

// NewInvalidContinueContext creates a SEM212 error
func NewInvalidContinueContext(loc ast.SourceLocation) *CompilerError {
	return newError(
		ErrInvalidContinueContext,
		"invalid_continue_context",
		CategorySemantic,
		SeverityError,
		"Continue statement outside of loop",
		loc,
	).WithSuggestion("Continue can only be used inside loops")
}

// NewUnreachableCode creates a SEM213 warning
func NewUnreachableCode(loc ast.SourceLocation) *CompilerError {
	return newError(
		ErrUnreachableCode,
		"unreachable_code",
		CategorySemantic,
		SeverityWarning,
		"Unreachable code detected",
		loc,
	).WithSuggestion("Remove code after return, break, or continue statements")
}

// NewInvalidAssignmentTarget creates a SEM214 error
func NewInvalidAssignmentTarget(loc ast.SourceLocation, target string) *CompilerError {
	return newError(
		ErrInvalidAssignmentTarget,
		"invalid_assignment_target",
		CategorySemantic,
		SeverityError,
		fmt.Sprintf("Cannot assign to '%s'", target),
		loc,
	).WithSuggestion("Assignment target must be a variable or field access")
}

// NewConstantReassignment creates a SEM215 error
func NewConstantReassignment(loc ast.SourceLocation, name string) *CompilerError {
	return newError(
		ErrConstantReassignment,
		"constant_reassignment",
		CategorySemantic,
		SeverityError,
		fmt.Sprintf("Cannot reassign constant '%s'", name),
		loc,
	).WithSuggestion("Constants cannot be reassigned after declaration")
}

// NewInvalidHookContext creates a SEM216 error
func NewInvalidHookContext(loc ast.SourceLocation, hookType string) *CompilerError {
	return newError(
		ErrInvalidHookContext,
		"invalid_hook_context",
		CategorySemantic,
		SeverityError,
		fmt.Sprintf("Hook '%s' cannot be used in this context", hookType),
		loc,
	).WithSuggestion("Hooks can only be defined inside resources")
}

// NewInvalidAsyncContext creates a SEM217 error
func NewInvalidAsyncContext(loc ast.SourceLocation) *CompilerError {
	return newError(
		ErrInvalidAsyncContext,
		"invalid_async_context",
		CategorySemantic,
		SeverityError,
		"@async block can only be used inside @after hooks",
		loc,
	).WithSuggestion("Move async block inside an @after hook or use @transaction instead")
}

// NewInvalidTransactionContext creates a SEM218 error
func NewInvalidTransactionContext(loc ast.SourceLocation) *CompilerError {
	return newError(
		ErrInvalidTransactionContext,
		"invalid_transaction_context",
		CategorySemantic,
		SeverityError,
		"@transaction annotation can only be used on hooks",
		loc,
	).WithSuggestion("Add @transaction to a @before or @after hook definition")
}
