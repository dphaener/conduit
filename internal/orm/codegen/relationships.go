package codegen

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// GenerateRelationshipMethods generates WithRelationship() methods for query builder
func GenerateRelationshipMethods(resource *schema.ResourceSchema) string {
	var b strings.Builder

	for relName := range resource.Relationships {
		methodName := "With" + toPascalCase(relName)

		b.WriteString(fmt.Sprintf(`
// %s eagerly loads the %s relationship
func (qb *%sQueryBuilder) %s() *%sQueryBuilder {
	qb.includes = append(qb.includes, %q)
	return qb
}
`, methodName, relName, resource.Name, methodName, resource.Name, relName))
	}

	return b.String()
}

// GenerateLazyLoadMethods generates lazy loading methods on the resource struct
func GenerateLazyLoadMethods(resource *schema.ResourceSchema, schemas map[string]*schema.ResourceSchema) string {
	var b strings.Builder

	for relName, rel := range resource.Relationships {
		methodName := toPascalCase(relName)
		targetResource := rel.TargetResource

		// Determine return type based on relationship type
		var returnType string
		switch rel.Type {
		case schema.RelationshipBelongsTo, schema.RelationshipHasOne:
			if rel.Nullable {
				returnType = fmt.Sprintf("*%s", targetResource)
			} else {
				returnType = fmt.Sprintf("*%s", targetResource)
			}
		case schema.RelationshipHasMany, schema.RelationshipHasManyThrough:
			returnType = fmt.Sprintf("[]%s", targetResource)
		}

		b.WriteString(fmt.Sprintf(`
// %s lazily loads the %s relationship
func (r *%s) %s(ctx context.Context) (%s, error) {
	if r._%sLoaded {
		return r._%s, r._%sErr
	}

	// Load the relationship
	loader := GetRelationshipLoader() // Assumes global loader
	rel := &schema.Relationship{
		Type:           schema.Relationship%s,
		TargetResource: %q,
		FieldName:      %q,
		ForeignKey:     %q,
		Nullable:       %t,
		OnDelete:       schema.%s,
		OnUpdate:       schema.%s,
		OrderBy:        %q,
		JoinTable:      %q,
		AssociationKey: %q,
	}

	value, err := loader.LoadSingle(ctx, r.ID, rel, &schema.ResourceSchema{Name: %q})
	if err != nil {
		r._%sErr = err
		return nil, err
	}

	r._%sLoaded = true
	r._%s = value
	return r._%s, nil
}
`,
			methodName, relName, resource.Name, methodName, returnType,
			relName, relName, relName,
			relationshipTypeToString(rel.Type),
			targetResource,
			relName,
			rel.ForeignKey,
			rel.Nullable,
			cascadeActionToString(rel.OnDelete),
			cascadeActionToString(rel.OnUpdate),
			rel.OrderBy,
			rel.JoinTable,
			rel.AssociationKey,
			resource.Name,
			relName,
			relName, relName, relName,
		))
	}

	return b.String()
}

// GenerateRelationshipFields generates struct fields for caching loaded relationships
func GenerateRelationshipFields(resource *schema.ResourceSchema) string {
	var b strings.Builder

	for relName, rel := range resource.Relationships {
		// Determine field type based on relationship type
		var fieldType string
		switch rel.Type {
		case schema.RelationshipBelongsTo, schema.RelationshipHasOne:
			fieldType = fmt.Sprintf("*%s", rel.TargetResource)
		case schema.RelationshipHasMany, schema.RelationshipHasManyThrough:
			fieldType = fmt.Sprintf("[]%s", rel.TargetResource)
		}

		b.WriteString(fmt.Sprintf("\t_%s %s\n", relName, fieldType))
		b.WriteString(fmt.Sprintf("\t_%sLoaded bool\n", relName))
		b.WriteString(fmt.Sprintf("\t_%sErr error\n", relName))
	}

	return b.String()
}

// relationshipTypeToString converts a relationship type to a string
func relationshipTypeToString(rt schema.RelationType) string {
	switch rt {
	case schema.RelationshipBelongsTo:
		return "BelongsTo"
	case schema.RelationshipHasMany:
		return "HasMany"
	case schema.RelationshipHasOne:
		return "HasOne"
	case schema.RelationshipHasManyThrough:
		return "HasManyThrough"
	default:
		return "Unknown"
	}
}

// cascadeActionToString converts a cascade action to a string
func cascadeActionToString(ca schema.CascadeAction) string {
	switch ca {
	case schema.CascadeRestrict:
		return "CascadeRestrict"
	case schema.CascadeCascade:
		return "CascadeCascade"
	case schema.CascadeSetNull:
		return "CascadeSetNull"
	case schema.CascadeNoAction:
		return "CascadeNoAction"
	default:
		return "CascadeRestrict"
	}
}
