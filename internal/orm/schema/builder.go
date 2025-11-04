// Package schema provides schema building functionality to convert AST to ResourceSchema
package schema

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Builder builds ResourceSchema from AST nodes
type Builder struct {
	errors   []error
	warnings []string
}

// NewBuilder creates a new schema builder
func NewBuilder() *Builder {
	return &Builder{
		errors:   make([]error, 0),
		warnings: make([]string, 0),
	}
}

// Build converts an AST ResourceNode to a ResourceSchema
func (b *Builder) Build(node *ast.ResourceNode) (*ResourceSchema, error) {
	schema := NewResourceSchema(node.Name)
	schema.Documentation = node.Documentation
	schema.Location = node.Loc

	// Build fields
	for _, fieldNode := range node.Fields {
		field, err := b.buildField(fieldNode)
		if err != nil {
			b.errors = append(b.errors, err)
			continue
		}
		schema.Fields[field.Name] = field
	}

	// Build relationships
	for _, relNode := range node.Relationships {
		rel, err := b.buildRelationship(relNode)
		if err != nil {
			b.errors = append(b.errors, err)
			continue
		}
		schema.Relationships[rel.FieldName] = rel
	}

	// Build hooks
	for _, hookNode := range node.Hooks {
		hook, err := b.buildHook(hookNode)
		if err != nil {
			b.errors = append(b.errors, err)
			continue
		}
		schema.Hooks[hook.Type] = append(schema.Hooks[hook.Type], hook)
	}

	// Build constraint blocks
	for _, constraintNode := range node.Constraints {
		constraint, err := b.buildConstraintBlock(constraintNode)
		if err != nil {
			b.errors = append(b.errors, err)
			continue
		}
		schema.ConstraintBlocks = append(schema.ConstraintBlocks, constraint)
	}

	// Build scopes
	for _, scopeNode := range node.Scopes {
		scope, err := b.buildScope(scopeNode)
		if err != nil {
			b.errors = append(b.errors, err)
			continue
		}
		schema.Scopes[scope.Name] = scope
	}

	// Build computed fields
	for _, computedNode := range node.Computed {
		computed, err := b.buildComputedField(computedNode)
		if err != nil {
			b.errors = append(b.errors, err)
			continue
		}
		schema.Computed[computed.Name] = computed
	}

	if len(b.errors) > 0 {
		var errMsgs []string
		for _, err := range b.errors {
			errMsgs = append(errMsgs, err.Error())
		}
		return nil, fmt.Errorf("schema building failed with %d errors:\n%s",
			len(b.errors), strings.Join(errMsgs, "\n"))
	}

	return schema, nil
}

// buildField converts an AST FieldNode to a Field
func (b *Builder) buildField(node *ast.FieldNode) (*Field, error) {
	typeSpec, err := b.buildTypeSpec(node.Type)
	if err != nil {
		return nil, fmt.Errorf("field %s: %w", node.Name, err)
	}

	field := &Field{
		Name:        node.Name,
		Type:        typeSpec,
		Constraints: make([]Constraint, 0),
		Annotations: make([]Annotation, 0),
		Location:    node.Loc,
	}

	// Build constraints
	for _, constraintNode := range node.Constraints {
		constraint, err := b.buildConstraint(constraintNode)
		if err != nil {
			return nil, fmt.Errorf("field %s constraint: %w", node.Name, err)
		}
		field.Constraints = append(field.Constraints, constraint)

		// Also add as annotation for DDL generation
		annotation := Annotation{
			Name: constraintNode.Name,
			Args: make([]interface{}, 0),
		}
		for _, arg := range constraintNode.Arguments {
			if val, err := b.extractValue(arg); err == nil {
				annotation.Args = append(annotation.Args, val)
			}
		}
		field.Annotations = append(field.Annotations, annotation)

		// For @max constraint, also update the TypeSpec.Length
		if constraintNode.Name == "max" && len(constraintNode.Arguments) > 0 {
			if maxVal, err := b.extractValue(constraintNode.Arguments[0]); err == nil {
				if maxInt, ok := maxVal.(int); ok {
					field.Type.Length = &maxInt
				} else if maxInt64, ok := maxVal.(int64); ok {
					maxInt := int(maxInt64)
					field.Type.Length = &maxInt
				}
			}
		}
	}

	return field, nil
}

// buildTypeSpec converts an AST TypeNode to a TypeSpec
func (b *Builder) buildTypeSpec(node *ast.TypeNode) (*TypeSpec, error) {
	spec := &TypeSpec{
		Nullable:       node.Nullable,
		NullabilitySet: true, // Always true when building from AST
	}

	switch node.Kind {
	case ast.TypePrimitive:
		primitiveType, err := ParsePrimitiveType(node.Name)
		if err != nil {
			return nil, err
		}
		spec.BaseType = primitiveType

	case ast.TypeArray:
		if node.ElementType == nil {
			return nil, fmt.Errorf("array type missing element type")
		}
		elementType, err := b.buildTypeSpec(node.ElementType)
		if err != nil {
			return nil, fmt.Errorf("array element type: %w", err)
		}
		spec.ArrayElement = elementType

	case ast.TypeHash:
		if node.KeyType == nil || node.ValueType == nil {
			return nil, fmt.Errorf("hash type missing key or value type")
		}
		keyType, err := b.buildTypeSpec(node.KeyType)
		if err != nil {
			return nil, fmt.Errorf("hash key type: %w", err)
		}
		valueType, err := b.buildTypeSpec(node.ValueType)
		if err != nil {
			return nil, fmt.Errorf("hash value type: %w", err)
		}
		spec.HashKey = keyType
		spec.HashValue = valueType

	case ast.TypeEnum:
		spec.BaseType = TypeEnum
		spec.EnumValues = node.EnumValues

	case ast.TypeStruct:
		spec.StructFields = make(map[string]*TypeSpec)
		for _, structField := range node.StructFields {
			fieldType, err := b.buildTypeSpec(structField.Type)
			if err != nil {
				return nil, fmt.Errorf("struct field %s: %w", structField.Name, err)
			}
			spec.StructFields[structField.Name] = fieldType
		}

	case ast.TypeResource:
		// Resource types are handled as relationships
		return nil, fmt.Errorf("resource types should be defined as relationships")

	default:
		return nil, fmt.Errorf("unknown type kind: %d", node.Kind)
	}

	return spec, nil
}

// buildConstraint converts an AST ConstraintNode to a Constraint
func (b *Builder) buildConstraint(node *ast.ConstraintNode) (Constraint, error) {
	var constraintType ConstraintType
	var value interface{}

	switch node.Name {
	case "min":
		constraintType = ConstraintMin
		if len(node.Arguments) > 0 {
			var err error
			value, err = b.extractValue(node.Arguments[0])
			if err != nil {
				return Constraint{}, fmt.Errorf("min constraint: %w", err)
			}
		}
	case "max":
		constraintType = ConstraintMax
		if len(node.Arguments) > 0 {
			var err error
			value, err = b.extractValue(node.Arguments[0])
			if err != nil {
				return Constraint{}, fmt.Errorf("max constraint: %w", err)
			}
		}
	case "pattern":
		constraintType = ConstraintPattern
		if len(node.Arguments) > 0 {
			var err error
			value, err = b.extractValue(node.Arguments[0])
			if err != nil {
				return Constraint{}, fmt.Errorf("pattern constraint: %w", err)
			}
		}
	case "unique":
		constraintType = ConstraintUnique
	case "index":
		constraintType = ConstraintIndex
	case "primary":
		constraintType = ConstraintPrimary
	case "auto":
		constraintType = ConstraintAuto
	case "auto_update":
		constraintType = ConstraintAutoUpdate
	case "default":
		constraintType = ConstraintDefault
		if len(node.Arguments) > 0 {
			var err error
			value, err = b.extractValue(node.Arguments[0])
			if err != nil {
				return Constraint{}, fmt.Errorf("default constraint: %w", err)
			}
		}
	default:
		return Constraint{}, fmt.Errorf("unknown constraint type: %s", node.Name)
	}

	return Constraint{
		Type:         constraintType,
		Value:        value,
		ErrorMessage: node.Error,
		Location:     node.Loc,
	}, nil
}

// extractValue extracts a value from an expression node
func (b *Builder) extractValue(expr ast.ExprNode) (interface{}, error) {
	// This is a simplified version - in practice we'd need to handle all expression types
	switch e := expr.(type) {
	case *ast.LiteralExpr:
		return e.Value, nil
	case *ast.IdentifierExpr:
		return nil, fmt.Errorf("identifier expressions not yet supported in constraint values")
	default:
		return nil, fmt.Errorf("unsupported expression type %T in constraint value", expr)
	}
}

// buildRelationship converts an AST RelationshipNode to a Relationship
func (b *Builder) buildRelationship(node *ast.RelationshipNode) (*Relationship, error) {
	var relType RelationType
	switch node.Kind {
	case ast.RelationshipBelongsTo:
		relType = RelationshipBelongsTo
	case ast.RelationshipHasMany:
		relType = RelationshipHasMany
	case ast.RelationshipHasManyThrough:
		relType = RelationshipHasManyThrough
	case ast.RelationshipHasOne:
		relType = RelationshipHasOne
	default:
		return nil, fmt.Errorf("unknown relationship kind: %d", node.Kind)
	}

	onDelete := CascadeRestrict
	if node.OnDelete != "" {
		action, err := ParseCascadeAction(node.OnDelete)
		if err != nil {
			return nil, err
		}
		onDelete = action
	}

	onUpdate := CascadeCascade
	// OnUpdate is typically cascade by default

	rel := &Relationship{
		Type:           relType,
		TargetResource: node.Type,
		FieldName:      node.Name,
		Nullable:       node.Nullable,
		ForeignKey:     node.ForeignKey,
		OnDelete:       onDelete,
		OnUpdate:       onUpdate,
		ThroughResource: node.Through,
		Location:       node.Loc,
	}

	return rel, nil
}

// buildHook converts an AST HookNode to a Hook
func (b *Builder) buildHook(node *ast.HookNode) (*Hook, error) {
	var hookType HookType
	hookKey := node.Timing + "_" + node.Event

	switch hookKey {
	case "before_create":
		hookType = BeforeCreate
	case "before_update":
		hookType = BeforeUpdate
	case "before_delete":
		hookType = BeforeDelete
	case "before_save":
		hookType = BeforeSave
	case "after_create":
		hookType = AfterCreate
	case "after_update":
		hookType = AfterUpdate
	case "after_delete":
		hookType = AfterDelete
	case "after_save":
		hookType = AfterSave
	default:
		return nil, fmt.Errorf("unknown hook type: %s", hookKey)
	}

	hook := &Hook{
		Type:        hookType,
		Transaction: node.IsTransaction,
		Async:       node.IsAsync,
		Body:        node.Body,
		Location:    node.Loc,
	}

	return hook, nil
}

// buildConstraintBlock converts an AST ConstraintNode to a ConstraintBlock
func (b *Builder) buildConstraintBlock(node *ast.ConstraintNode) (*ConstraintBlock, error) {
	constraint := &ConstraintBlock{
		Name:      node.Name,
		On:        node.On,
		When:      node.When,
		Condition: node.Condition,
		Error:     node.Error,
		Location:  node.Loc,
	}

	return constraint, nil
}

// buildScope converts an AST ScopeNode to a Scope
func (b *Builder) buildScope(node *ast.ScopeNode) (*Scope, error) {
	scope := &Scope{
		Name:      node.Name,
		Arguments: make([]*ScopeArgument, 0),
		Where:     make(map[string]interface{}),
		Location:  node.Loc,
	}

	// Build scope arguments
	for _, argNode := range node.Arguments {
		argType, err := b.buildTypeSpec(argNode.Type)
		if err != nil {
			return nil, fmt.Errorf("scope %s argument %s: %w", node.Name, argNode.Name, err)
		}

		arg := &ScopeArgument{
			Name:     argNode.Name,
			Type:     argType,
			Location: argNode.Loc,
		}

		if argNode.Default != nil {
			defaultVal, err := b.extractValue(argNode.Default)
			if err != nil {
				return nil, fmt.Errorf("scope %s argument %s default: %w", node.Name, argNode.Name, err)
			}
			arg.Default = defaultVal
		}

		scope.Arguments = append(scope.Arguments, arg)
	}

	return scope, nil
}

// buildComputedField converts an AST ComputedNode to a ComputedField
func (b *Builder) buildComputedField(node *ast.ComputedNode) (*ComputedField, error) {
	typeSpec, err := b.buildTypeSpec(node.Type)
	if err != nil {
		return nil, fmt.Errorf("computed field %s: %w", node.Name, err)
	}

	computed := &ComputedField{
		Name:     node.Name,
		Type:     typeSpec,
		Body:     node.Body,
		Location: node.Loc,
	}

	return computed, nil
}

// Errors returns all errors encountered during building
func (b *Builder) Errors() []error {
	return b.errors
}

// Warnings returns all warnings encountered during building
func (b *Builder) Warnings() []string {
	return b.warnings
}
