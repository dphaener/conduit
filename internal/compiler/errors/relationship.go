package errors

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Relationship error codes (REL300-399)
const (
	// ErrInvalidRelationshipType indicates an invalid relationship type
	ErrInvalidRelationshipType ErrorCode = "REL300"
	// ErrMissingForeignKey indicates a missing foreign key in relationship
	ErrMissingForeignKey ErrorCode = "REL301"
	// ErrInvalidForeignKey indicates an invalid foreign key specification
	ErrInvalidForeignKey ErrorCode = "REL302"
	// ErrInvalidOnDelete indicates an invalid on_delete action
	ErrInvalidOnDelete ErrorCode = "REL303"
	// ErrInvalidThroughTable indicates an invalid through table for has-many-through
	ErrInvalidThroughTable ErrorCode = "REL304"
	// ErrSelfReferentialRelationship indicates a resource referencing itself without proper setup
	ErrSelfReferentialRelationship ErrorCode = "REL305"
	// ErrConflictingRelationships indicates conflicting relationship definitions
	ErrConflictingRelationships ErrorCode = "REL306"
	// ErrMissingInverseRelationship indicates missing inverse relationship
	ErrMissingInverseRelationship ErrorCode = "REL307"
	// ErrInvalidRelationshipNullability indicates invalid nullability for relationship
	ErrInvalidRelationshipNullability ErrorCode = "REL308"
	// ErrPolymorphicNotSupported indicates polymorphic relationships not yet supported
	ErrPolymorphicNotSupported ErrorCode = "REL309"
)

// NewInvalidRelationshipType creates a REL300 error
func NewInvalidRelationshipType(loc ast.SourceLocation, relType string) *CompilerError {
	return newError(
		ErrInvalidRelationshipType,
		"invalid_relationship_type",
		CategoryRelationship,
		SeverityError,
		fmt.Sprintf("Invalid relationship type '%s'", relType),
		loc,
	).WithSuggestion("Valid relationship types: belongs_to, has_many, has_many_through, has_one").
		WithExamples(
			"author: User! { foreign_key: \"author_id\" }",
			"posts: array<Post>! @has_many",
		)
}

// NewMissingForeignKey creates a REL301 error
func NewMissingForeignKey(loc ast.SourceLocation, relationshipName string) *CompilerError {
	return newError(
		ErrMissingForeignKey,
		"missing_foreign_key",
		CategoryRelationship,
		SeverityError,
		fmt.Sprintf("Relationship '%s' is missing foreign_key specification", relationshipName),
		loc,
	).WithSuggestion("Add foreign_key to the relationship definition").
		WithExamples(
			fmt.Sprintf("%s: User! { foreign_key: \"user_id\" }", relationshipName),
		)
}

// NewInvalidForeignKey creates a REL302 error
func NewInvalidForeignKey(loc ast.SourceLocation, foreignKey, reason string) *CompilerError {
	return newError(
		ErrInvalidForeignKey,
		"invalid_foreign_key",
		CategoryRelationship,
		SeverityError,
		fmt.Sprintf("Invalid foreign key '%s': %s", foreignKey, reason),
		loc,
	).WithSuggestion("Foreign key must be a valid field name in snake_case")
}

// NewInvalidOnDelete creates a REL303 error
func NewInvalidOnDelete(loc ast.SourceLocation, action string) *CompilerError {
	return newError(
		ErrInvalidOnDelete,
		"invalid_on_delete",
		CategoryRelationship,
		SeverityError,
		fmt.Sprintf("Invalid on_delete action '%s'", action),
		loc,
	).WithSuggestion("Valid on_delete actions: cascade, restrict, nullify, set_default").
		WithExamples(
			"author: User! { foreign_key: \"author_id\", on_delete: cascade }",
			"category: Category? { foreign_key: \"category_id\", on_delete: nullify }",
		)
}

// NewInvalidThroughTable creates a REL304 error
func NewInvalidThroughTable(loc ast.SourceLocation, tableName, reason string) *CompilerError {
	return newError(
		ErrInvalidThroughTable,
		"invalid_through_table",
		CategoryRelationship,
		SeverityError,
		fmt.Sprintf("Invalid through table '%s': %s", tableName, reason),
		loc,
	).WithSuggestion("Through table must be a valid resource name for has_many_through relationships")
}

// NewSelfReferentialRelationship creates a REL305 warning
func NewSelfReferentialRelationship(loc ast.SourceLocation, resourceName string) *CompilerError {
	return newError(
		ErrSelfReferentialRelationship,
		"self_referential_relationship",
		CategoryRelationship,
		SeverityWarning,
		fmt.Sprintf("Resource '%s' references itself", resourceName),
		loc,
	).WithSuggestion("Ensure the relationship is nullable or has proper constraints to prevent infinite recursion")
}

// NewConflictingRelationships creates a REL306 error
func NewConflictingRelationships(loc ast.SourceLocation, field1, field2 string) *CompilerError {
	return newError(
		ErrConflictingRelationships,
		"conflicting_relationships",
		CategoryRelationship,
		SeverityError,
		fmt.Sprintf("Conflicting relationships: '%s' and '%s' use the same foreign key", field1, field2),
		loc,
	).WithSuggestion("Each relationship must use a unique foreign key")
}

// NewMissingInverseRelationship creates a REL307 warning
func NewMissingInverseRelationship(loc ast.SourceLocation, relationship, targetResource string) *CompilerError {
	return newError(
		ErrMissingInverseRelationship,
		"missing_inverse_relationship",
		CategoryRelationship,
		SeverityWarning,
		fmt.Sprintf("Relationship '%s' to '%s' has no inverse relationship defined", relationship, targetResource),
		loc,
	).WithSuggestion("Consider adding an inverse relationship for bidirectional navigation")
}

// NewInvalidRelationshipNullability creates a REL308 error
func NewInvalidRelationshipNullability(loc ast.SourceLocation, relationship, reason string) *CompilerError {
	return newError(
		ErrInvalidRelationshipNullability,
		"invalid_relationship_nullability",
		CategoryRelationship,
		SeverityError,
		fmt.Sprintf("Invalid nullability for relationship '%s': %s", relationship, reason),
		loc,
	).WithSuggestion("Ensure relationship nullability matches foreign key nullability and on_delete behavior")
}

// NewPolymorphicNotSupported creates a REL309 error
func NewPolymorphicNotSupported(loc ast.SourceLocation) *CompilerError {
	return newError(
		ErrPolymorphicNotSupported,
		"polymorphic_not_supported",
		CategoryRelationship,
		SeverityError,
		"Polymorphic relationships are not yet supported",
		loc,
	).WithSuggestion("Use explicit relationships to each target type instead")
}
