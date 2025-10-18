package codegen

import (
	"fmt"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

func BenchmarkDDLGeneration_100Resources(b *testing.B) {
	// Create 100 test resources
	resources := make([]*schema.ResourceSchema, 100)
	for i := 0; i < 100; i++ {
		resource := schema.NewResourceSchema(fmt.Sprintf("Resource%d", i))

		// Add primary key
		resource.Fields["id"] = &schema.Field{
			Name: "id",
			Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
			Annotations: []schema.Annotation{{Name: "primary"}},
		}

		// Add various fields
		resource.Fields["name"] = &schema.Field{
			Name: "name",
			Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
		}
		resource.Fields["count"] = &schema.Field{
			Name: "count",
			Type: &schema.TypeSpec{BaseType: schema.TypeInt, Nullable: false},
			Constraints: []schema.Constraint{
				{Type: schema.ConstraintMin, Value: 0},
			},
		}
		resource.Fields["active"] = &schema.Field{
			Name: "active",
			Type: &schema.TypeSpec{BaseType: schema.TypeBool, Nullable: false},
		}
		resource.Fields["created"] = &schema.Field{
			Name: "created",
			Type: &schema.TypeSpec{BaseType: schema.TypeTimestamp, Nullable: false},
			Annotations: []schema.Annotation{{Name: "auto"}},
		}

		resources[i] = resource
	}

	gen := NewDDLGenerator()
	constraintGen := NewConstraintGenerator()
	indexGen := NewIndexGenerator()

	// Reset the timer before the actual benchmark
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, resource := range resources {
			_, _ = gen.GenerateSchema(resource)
			_ = constraintGen.GenerateCheckConstraints(resource)
			_ = indexGen.GenerateAllIndexes(resource)
		}
	}
}

func TestPerformance_100Resources(t *testing.T) {
	// Create 100 test resources
	resources := make([]*schema.ResourceSchema, 100)
	for i := 0; i < 100; i++ {
		resource := schema.NewResourceSchema(fmt.Sprintf("Resource%d", i))

		// Add primary key
		resource.Fields["id"] = &schema.Field{
			Name: "id",
			Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
			Annotations: []schema.Annotation{{Name: "primary"}},
		}

		// Add various fields
		resource.Fields["name"] = &schema.Field{
			Name: "name",
			Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
			Annotations: []schema.Annotation{{Name: "unique"}},
		}
		resource.Fields["slug"] = &schema.Field{
			Name: "slug",
			Type: &schema.TypeSpec{BaseType: schema.TypeString, Nullable: false},
			Annotations: []schema.Annotation{{Name: "index"}},
		}
		resource.Fields["count"] = &schema.Field{
			Name: "count",
			Type: &schema.TypeSpec{BaseType: schema.TypeInt, Nullable: false},
			Constraints: []schema.Constraint{
				{Type: schema.ConstraintMin, Value: 0},
				{Type: schema.ConstraintMax, Value: 1000},
			},
		}
		resource.Fields["email"] = &schema.Field{
			Name: "email",
			Type: &schema.TypeSpec{BaseType: schema.TypeEmail, Nullable: false},
		}
		resource.Fields["active"] = &schema.Field{
			Name: "active",
			Type: &schema.TypeSpec{
				BaseType: schema.TypeBool,
				Nullable: false,
				Default:  true,
			},
		}
		resource.Fields["created"] = &schema.Field{
			Name: "created",
			Type: &schema.TypeSpec{BaseType: schema.TypeTimestamp, Nullable: false},
			Annotations: []schema.Annotation{{Name: "auto"}},
		}
		resource.Fields["content"] = &schema.Field{
			Name: "content",
			Type: &schema.TypeSpec{BaseType: schema.TypeText, Nullable: true},
		}

		resources[i] = resource
	}

	gen := NewDDLGenerator()
	constraintGen := NewConstraintGenerator()
	indexGen := NewIndexGenerator()

	start := time.Now()

	// Generate DDL for all 100 resources
	for _, resource := range resources {
		_, err := gen.GenerateSchema(resource)
		if err != nil {
			t.Fatalf("GenerateSchema() error = %v", err)
		}

		_ = constraintGen.GenerateCheckConstraints(resource)
		_ = indexGen.GenerateAllIndexes(resource)
	}

	elapsed := time.Since(start)

	// Target: <50ms for 100 resources
	if elapsed > 50*time.Millisecond {
		t.Errorf("Performance target missed: took %v, want < 50ms", elapsed)
	} else {
		t.Logf("Performance target met: took %v (< 50ms)", elapsed)
	}
}

func TestPerformance_SingleResource(t *testing.T) {
	resource := schema.NewResourceSchema("ComplexResource")

	// Add primary key
	resource.Fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{BaseType: schema.TypeUUID, Nullable: false},
		Annotations: []schema.Annotation{{Name: "primary"}, {Name: "auto"}},
	}

	// Add 20 fields of various types
	for i := 0; i < 20; i++ {
		fieldName := fmt.Sprintf("field_%d", i)

		var baseType schema.PrimitiveType
		var constraints []schema.Constraint

		switch i % 5 {
		case 0:
			baseType = schema.TypeString
			constraints = []schema.Constraint{
				{Type: schema.ConstraintMin, Value: 1},
				{Type: schema.ConstraintMax, Value: 255},
			}
		case 1:
			baseType = schema.TypeInt
			constraints = []schema.Constraint{
				{Type: schema.ConstraintMin, Value: 0},
			}
		case 2:
			baseType = schema.TypeBool
		case 3:
			baseType = schema.TypeTimestamp
		case 4:
			baseType = schema.TypeText
		}

		resource.Fields[fieldName] = &schema.Field{
			Name: fieldName,
			Type: &schema.TypeSpec{
				BaseType: baseType,
				Nullable: i%3 == 0, // Some nullable
			},
			Constraints: constraints,
		}
	}

	gen := NewDDLGenerator()
	constraintGen := NewConstraintGenerator()
	indexGen := NewIndexGenerator()

	start := time.Now()

	// Run 1000 iterations to get a good average
	for i := 0; i < 1000; i++ {
		_, _ = gen.GenerateSchema(resource)
		_ = constraintGen.GenerateCheckConstraints(resource)
		_ = indexGen.GenerateAllIndexes(resource)
	}

	elapsed := time.Since(start)
	perIteration := elapsed / 1000

	t.Logf("Average time per resource: %v", perIteration)

	// Sanity check: each resource should take < 500µs
	if perIteration > 500*time.Microsecond {
		t.Errorf("Single resource took too long: %v per iteration", perIteration)
	}
}
