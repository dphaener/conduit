// Package docs provides comprehensive documentation generation for Conduit projects.
// It supports multiple output formats including OpenAPI, Markdown, and interactive HTML.
package docs

import (
	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// Generator orchestrates documentation generation across multiple formats
type Generator struct {
	config  *Config
	extractor *Extractor
}

// Config holds configuration for documentation generation
type Config struct {
	// ProjectName is the name of the Conduit project
	ProjectName string

	// ProjectVersion is the semantic version of the project
	ProjectVersion string

	// ProjectDescription is a short description of the project
	ProjectDescription string

	// OutputDir is the base directory for generated documentation
	OutputDir string

	// Formats specifies which formats to generate
	Formats []Format

	// BaseURL is the base URL for the API (used in OpenAPI spec)
	BaseURL string

	// ServerURLs are additional server URLs for the API
	ServerURLs []ServerURL
}

// Format represents a documentation output format
type Format string

const (
	// FormatOpenAPI generates OpenAPI 3.0 specification
	FormatOpenAPI Format = "openapi"

	// FormatMarkdown generates Markdown documentation
	FormatMarkdown Format = "markdown"

	// FormatHTML generates interactive HTML documentation
	FormatHTML Format = "html"
)

// ServerURL represents an API server URL in OpenAPI spec
type ServerURL struct {
	URL         string
	Description string
}

// Documentation represents extracted documentation from Conduit source
type Documentation struct {
	// Resources contains all documented resources
	Resources []*ResourceDoc

	// ProjectInfo contains project-level metadata
	ProjectInfo *ProjectInfo
}

// ProjectInfo contains project-level metadata
type ProjectInfo struct {
	Name        string
	Version     string
	Description string
}

// ResourceDoc represents documentation for a single resource
type ResourceDoc struct {
	// Name is the resource name
	Name string

	// Documentation is the extracted doc comment
	Documentation string

	// Fields contains all resource fields
	Fields []*FieldDoc

	// Relationships contains all relationships
	Relationships []*RelationshipDoc

	// Endpoints contains all REST endpoints for this resource
	Endpoints []*EndpointDoc

	// Hooks contains lifecycle hooks
	Hooks []*HookDoc

	// Validations contains validation rules
	Validations []*ValidationDoc

	// Constraints contains constraint rules
	Constraints []*ConstraintDoc
}

// FieldDoc represents documentation for a resource field
type FieldDoc struct {
	// Name is the field name
	Name string

	// Type is the field type (string!, int?, etc.)
	Type string

	// Description is extracted from comments
	Description string

	// Required indicates if the field is required
	Required bool

	// Constraints lists all field constraints
	Constraints []string

	// Default is the default value if any
	Default string

	// Example is an auto-generated example value
	Example interface{}
}

// RelationshipDoc represents documentation for a relationship
type RelationshipDoc struct {
	// Name is the relationship field name
	Name string

	// Type is the related resource type
	Type string

	// Kind is the relationship kind (belongs_to, has_many, etc.)
	Kind string

	// ForeignKey is the foreign key field
	ForeignKey string

	// Description is extracted from comments
	Description string
}

// EndpointDoc represents documentation for a REST endpoint
type EndpointDoc struct {
	// Method is the HTTP method (GET, POST, PUT, DELETE, PATCH)
	Method string

	// Path is the URL path
	Path string

	// Summary is a short description
	Summary string

	// Description is a detailed description
	Description string

	// Parameters contains path and query parameters
	Parameters []*ParameterDoc

	// RequestBody describes the request body
	RequestBody *RequestBodyDoc

	// Responses describes possible responses
	Responses map[int]*ResponseDoc

	// Middleware lists applied middleware
	Middleware []string
}

// ParameterDoc represents a parameter in an endpoint
type ParameterDoc struct {
	// Name is the parameter name
	Name string

	// In specifies where the parameter appears (path, query, header)
	In string

	// Type is the parameter type
	Type string

	// Required indicates if the parameter is required
	Required bool

	// Description explains the parameter
	Description string

	// Example provides an example value
	Example interface{}
}

// RequestBodyDoc describes a request body
type RequestBodyDoc struct {
	// Description explains the request body
	Description string

	// Required indicates if the body is required
	Required bool

	// ContentType is the media type (application/json)
	ContentType string

	// Schema describes the structure
	Schema *SchemaDoc

	// Example provides an example request
	Example interface{}
}

// ResponseDoc describes an HTTP response
type ResponseDoc struct {
	// StatusCode is the HTTP status code
	StatusCode int

	// Description explains the response
	Description string

	// ContentType is the media type
	ContentType string

	// Schema describes the response structure
	Schema *SchemaDoc

	// Example provides an example response
	Example interface{}
}

// SchemaDoc describes a JSON schema
type SchemaDoc struct {
	// Type is the schema type (object, array, string, etc.)
	Type string

	// Properties contains object properties
	Properties map[string]*PropertyDoc

	// Items describes array items
	Items *SchemaDoc

	// Required lists required properties
	Required []string
}

// PropertyDoc describes an object property
type PropertyDoc struct {
	// Type is the property type
	Type string

	// Description explains the property
	Description string

	// Format provides additional type information
	Format string

	// Example provides an example value
	Example interface{}

	// Enum lists allowed values
	Enum []interface{}
}

// HookDoc represents documentation for a lifecycle hook
type HookDoc struct {
	// Timing is "before" or "after"
	Timing string

	// Event is the lifecycle event (create, update, delete, save)
	Event string

	// Description explains what the hook does
	Description string

	// IsAsync indicates if the hook runs asynchronously
	IsAsync bool

	// IsTransaction indicates if the hook runs in a transaction
	IsTransaction bool
}

// ValidationDoc represents documentation for a validation rule
type ValidationDoc struct {
	// Name is the validation name
	Name string

	// Description explains the validation
	Description string

	// ErrorMessage is the error message shown when validation fails
	ErrorMessage string
}

// ConstraintDoc represents documentation for a constraint
type ConstraintDoc struct {
	// Name is the constraint name
	Name string

	// Description explains the constraint
	Description string

	// Arguments contains constraint arguments
	Arguments []string

	// On lists events this constraint applies to
	On []string
}

// NewGenerator creates a new documentation generator
func NewGenerator(config *Config) *Generator {
	return &Generator{
		config:    config,
		extractor: NewExtractor(),
	}
}

// Generate generates documentation in all configured formats
func (g *Generator) Generate(program *ast.Program) error {
	// Extract documentation from AST
	doc := g.extractor.Extract(program, g.config.ProjectName, g.config.ProjectVersion, g.config.ProjectDescription)

	// Generate each requested format
	for _, format := range g.config.Formats {
		switch format {
		case FormatOpenAPI:
			generator := NewOpenAPIGenerator(g.config)
			if err := generator.Generate(doc); err != nil {
				return err
			}
		case FormatMarkdown:
			generator := NewMarkdownGenerator(g.config)
			if err := generator.Generate(doc); err != nil {
				return err
			}
		case FormatHTML:
			generator := NewHTMLGenerator(g.config)
			if err := generator.Generate(doc); err != nil {
				return err
			}
		}
	}

	return nil
}
