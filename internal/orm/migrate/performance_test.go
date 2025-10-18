package migrate

import (
	"fmt"
	"testing"
	"time"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// BenchmarkDifferComputeDiff measures diff generation performance
func BenchmarkDifferComputeDiff(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("resources=%d", size), func(b *testing.B) {
			oldSchemas := generateTestSchemas(size)
			newSchemas := generateTestSchemas(size)

			// Modify a few resources
			modifySchemas(newSchemas, size/10)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				differ := NewDiffer(oldSchemas, newSchemas)
				_ = differ.ComputeDiff()
			}
		})
	}
}

// BenchmarkGeneratorGenerateMigration measures migration generation performance
func BenchmarkGeneratorGenerateMigration(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("resources=%d", size), func(b *testing.B) {
			oldSchemas := generateTestSchemas(size)
			newSchemas := generateTestSchemas(size)
			modifySchemas(newSchemas, size/10)

			gen := NewGenerator()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = gen.GenerateMigration(oldSchemas, newSchemas)
			}
		})
	}
}

// TestPerformanceTargets validates that performance targets are met
func TestPerformanceTargets(t *testing.T) {
	// Target: Diff generation <200ms for 100 resources
	t.Run("DiffGeneration100Resources", func(t *testing.T) {
		oldSchemas := generateTestSchemas(100)
		newSchemas := generateTestSchemas(100)
		modifySchemas(newSchemas, 10)

		start := time.Now()
		differ := NewDiffer(oldSchemas, newSchemas)
		_ = differ.ComputeDiff()
		duration := time.Since(start)

		if duration > 200*time.Millisecond {
			t.Errorf("Diff generation took %v, expected <200ms", duration)
		}

		t.Logf("Diff generation for 100 resources: %v", duration)
	})

	// Target: Migration generation <500ms
	t.Run("MigrationGeneration100Resources", func(t *testing.T) {
		oldSchemas := generateTestSchemas(100)
		newSchemas := generateTestSchemas(100)
		modifySchemas(newSchemas, 10)

		gen := NewGenerator()

		start := time.Now()
		_, err := gen.GenerateMigration(oldSchemas, newSchemas)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("GenerateMigration() failed: %v", err)
		}

		if duration > 500*time.Millisecond {
			t.Errorf("Migration generation took %v, expected <500ms", duration)
		}

		t.Logf("Migration generation for 100 resources: %v", duration)
	})

	// Target: Large schema diff <500ms
	t.Run("LargeSchemaDiff", func(t *testing.T) {
		oldSchemas := generateTestSchemas(200)
		newSchemas := generateTestSchemas(200)
		modifySchemas(newSchemas, 20)

		start := time.Now()
		differ := NewDiffer(oldSchemas, newSchemas)
		changes := differ.ComputeDiff()
		duration := time.Since(start)

		t.Logf("Diff of 200 resources with %d changes took %v", len(changes), duration)

		// Should still be fast even for large schemas
		if duration > 500*time.Millisecond {
			t.Errorf("Large schema diff took %v, expected <500ms", duration)
		}
	})
}

// Helper functions for generating test schemas

func generateTestSchemas(count int) map[string]*schema.ResourceSchema {
	schemas := make(map[string]*schema.ResourceSchema)

	for i := 0; i < count; i++ {
		name := fmt.Sprintf("Resource%d", i)
		schemas[name] = generateTestResource(name, 10)
	}

	return schemas
}

func generateTestResource(name string, fieldCount int) *schema.ResourceSchema {
	fields := make(map[string]*schema.Field)

	// Add ID field
	fields["id"] = &schema.Field{
		Name: "id",
		Type: &schema.TypeSpec{
			BaseType: schema.TypeUUID,
			Nullable: false,
		},
		Annotations: []schema.Annotation{
			{Name: "primary"},
		},
	}

	// Add other fields
	for i := 0; i < fieldCount-1; i++ {
		fieldName := fmt.Sprintf("field%d", i)
		fields[fieldName] = &schema.Field{
			Name: fieldName,
			Type: &schema.TypeSpec{
				BaseType: schema.TypeString,
				Nullable: i%2 == 0,
			},
		}
	}

	return &schema.ResourceSchema{
		Name:          name,
		TableName:     toSnakeCase(name),
		Fields:        fields,
		Relationships: make(map[string]*schema.Relationship),
	}
}

func modifySchemas(schemas map[string]*schema.ResourceSchema, modifyCount int) {
	count := 0
	for _, resource := range schemas {
		if count >= modifyCount {
			break
		}

		// Add a new field to this resource
		newFieldName := fmt.Sprintf("new_field_%d", count)
		resource.Fields[newFieldName] = &schema.Field{
			Name: newFieldName,
			Type: &schema.TypeSpec{
				BaseType: schema.TypeString,
				Nullable: true,
			},
		}

		count++
	}
}

// BenchmarkMigrationNameGeneration measures migration name generation performance
func BenchmarkMigrationNameGeneration(b *testing.B) {
	changes := make([]SchemaChange, 0, 100)

	// Generate 100 changes
	for i := 0; i < 100; i++ {
		changes = append(changes, SchemaChange{
			Type:     ChangeAddField,
			Resource: fmt.Sprintf("Resource%d", i),
			Field:    fmt.Sprintf("field%d", i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GenerateMigrationName(changes)
	}
}

// TestMemoryUsage tests memory efficiency
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	// Generate large schemas
	oldSchemas := generateTestSchemas(500)
	newSchemas := generateTestSchemas(500)
	modifySchemas(newSchemas, 50)

	// Run diff
	differ := NewDiffer(oldSchemas, newSchemas)
	changes := differ.ComputeDiff()

	t.Logf("Generated %d changes from 500 resources", len(changes))

	// Generate migration
	gen := NewGenerator()
	migration, err := gen.GenerateMigration(oldSchemas, newSchemas)
	if err != nil {
		t.Fatalf("GenerateMigration() failed: %v", err)
	}

	t.Logf("Migration SQL length: up=%d, down=%d",
		len(migration.Up), len(migration.Down))

	// This test mainly validates that we don't panic or run out of memory
	// with large schemas
}
