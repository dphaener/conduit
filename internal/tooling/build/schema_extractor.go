package build

import (
	"fmt"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// SchemaExtractor extracts ResourceSchema instances from compiled AST
type SchemaExtractor struct {
	builder *schema.Builder
}

// NewSchemaExtractor creates a new schema extractor
func NewSchemaExtractor() *SchemaExtractor {
	return &SchemaExtractor{
		builder: schema.NewBuilder(),
	}
}

// ExtractSchemas extracts all resource schemas from compiled files
func (e *SchemaExtractor) ExtractSchemas(compiled []*CompiledFile) (map[string]*schema.ResourceSchema, error) {
	schemas := make(map[string]*schema.ResourceSchema)

	for _, cf := range compiled {
		for _, resource := range cf.Program.Resources {
			resourceSchema, err := e.builder.Build(resource)
			if err != nil {
				return nil, fmt.Errorf("failed to build schema for resource %s in %s: %w",
					resource.Name, cf.Path, err)
			}

			// Store schema by resource name
			if existing, exists := schemas[resource.Name]; exists {
				return nil, fmt.Errorf("duplicate resource name %s (defined in %s and %s)",
					resource.Name, existing.FilePath, cf.Path)
			}

			resourceSchema.FilePath = cf.Path
			schemas[resource.Name] = resourceSchema
		}
	}

	return schemas, nil
}

// ExtractSchemasFromProgram extracts schemas from a single AST program
func (e *SchemaExtractor) ExtractSchemasFromProgram(program *ast.Program, filePath string) (map[string]*schema.ResourceSchema, error) {
	schemas := make(map[string]*schema.ResourceSchema)

	for _, resource := range program.Resources {
		resourceSchema, err := e.builder.Build(resource)
		if err != nil {
			return nil, fmt.Errorf("failed to build schema for resource %s: %w", resource.Name, err)
		}

		if _, exists := schemas[resource.Name]; exists {
			return nil, fmt.Errorf("duplicate resource name: %s", resource.Name)
		}

		resourceSchema.FilePath = filePath
		schemas[resource.Name] = resourceSchema
	}

	return schemas, nil
}
