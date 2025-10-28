// Package metadata provides structures for capturing and serializing introspection
// metadata about Conduit resources, patterns, and dependencies.
//
// # Overview
//
// The metadata package defines the schema for the introspection system that enables
// LLMs and developer tooling to query information about compiled Conduit applications.
// All metadata is JSON-serializable and optimized for size (<2KB per resource when compressed).
//
// # Core Structures
//
// The package defines several key types:
//
//   - Metadata: Top-level container for all introspection data
//   - ResourceMetadata: Complete information about a single resource
//   - FieldMetadata: Field type, nullability, and constraints
//   - RelationshipMetadata: Resource relationships (belongs_to, has_many, etc.)
//   - HookMetadata: Lifecycle hooks with transaction and async flags
//   - PatternMetadata: Discovered usage patterns for LLM learning
//   - DependencyGraph: Resource dependency graph for tooling
//
// # Example Usage
//
// Creating and serializing metadata:
//
//	meta := metadata.Metadata{
//		Version:    "1.0",
//		Generated:  time.Now(),
//		SourceHash: "abc123",
//		Resources: []metadata.ResourceMetadata{
//			{
//				Name: "Post",
//				Fields: []metadata.FieldMetadata{
//					{
//						Name:     "title",
//						Type:     "string",
//						Nullable: false,
//						Required: true,
//						Constraints: []string{"@min(5)", "@max(200)"},
//					},
//				},
//			},
//		},
//	}
//
//	// Serialize to JSON
//	jsonData, _ := json.Marshal(meta)
//
//	// Compress for distribution
//	var compressed bytes.Buffer
//	gzipWriter := gzip.NewWriter(&compressed)
//	gzipWriter.Write(jsonData)
//	gzipWriter.Close()
//
// # Example JSON Output
//
// A complete metadata file looks like:
//
//	{
//	  "version": "1.0",
//	  "generated": "2025-10-25T12:00:00Z",
//	  "source_hash": "abc123def456",
//	  "resources": [
//	    {
//	      "name": "Post",
//	      "documentation": "Blog post resource",
//	      "file_path": "/app/resources/post.cdt",
//	      "fields": [
//	        {
//	          "name": "id",
//	          "type": "uuid",
//	          "nullable": false,
//	          "required": true,
//	          "constraints": ["@primary", "@auto"]
//	        },
//	        {
//	          "name": "title",
//	          "type": "string",
//	          "nullable": false,
//	          "required": true,
//	          "constraints": ["@min(5)", "@max(200)"]
//	        }
//	      ],
//	      "relationships": [
//	        {
//	          "name": "author",
//	          "type": "belongs_to",
//	          "target_resource": "User",
//	          "foreign_key": "author_id",
//	          "on_delete": "restrict"
//	        }
//	      ],
//	      "hooks": [
//	        {
//	          "type": "before_create",
//	          "transaction": true,
//	          "async": false,
//	          "source_code": "self.slug = String.slugify(self.title)",
//	          "line_number": 15
//	        }
//	      ]
//	    }
//	  ],
//	  "routes": [
//	    {
//	      "method": "GET",
//	      "path": "/posts",
//	      "handler": "ListPosts",
//	      "resource": "Post",
//	      "operation": "list",
//	      "middleware": ["auth"]
//	    }
//	  ],
//	  "patterns": [
//	    {
//	      "id": "pattern-1",
//	      "name": "slug-generation",
//	      "category": "hook",
//	      "description": "Auto-generate URL slug from title",
//	      "template": "@before create { self.slug = String.slugify(self.title) }",
//	      "examples": [
//	        {
//	          "resource": "Post",
//	          "file_path": "/app/resources/post.cdt",
//	          "line_number": 15,
//	          "code": "self.slug = String.slugify(self.title)"
//	        }
//	      ],
//	      "frequency": 3,
//	      "confidence": 0.95
//	    }
//	  ],
//	  "dependencies": {
//	    "nodes": {
//	      "post": {
//	        "id": "post",
//	        "type": "resource",
//	        "name": "Post",
//	        "file_path": "/app/resources/post.cdt"
//	      },
//	      "user": {
//	        "id": "user",
//	        "type": "resource",
//	        "name": "User",
//	        "file_path": "/app/resources/user.cdt"
//	      }
//	    },
//	    "edges": [
//	      {
//	        "from": "post",
//	        "to": "user",
//	        "relationship": "belongs_to",
//	        "weight": 1
//	      }
//	    ]
//	  }
//	}
//
// # Size Characteristics
//
// The metadata schema is designed to be compact:
//
//   - Uncompressed: ~2KB per typical resource
//   - Compressed (gzip): ~700 bytes per resource
//   - Compression ratio: ~35%
//
// Optional fields use `omitempty` JSON tags to reduce size when not needed.
//
// # Schema Versioning
//
// The Metadata.Version field supports schema evolution. Future versions
// can add new fields while maintaining backward compatibility through
// careful use of optional fields and version checks.
//
// Current version: "1.0"
//
// # Field Type Reference
//
// Common field types in FieldMetadata.Type:
//
//   - Primitives: string, integer, float, boolean
//   - Special: uuid, timestamp, date, time
//   - Text: text (unlimited length)
//   - Custom: Any user-defined type name
//
// # Relationship Types
//
// RelationshipMetadata.Type supports:
//
//   - belongs_to: N:1 relationship with foreign key
//   - has_many: 1:N relationship via foreign key
//   - has_many_through: M:N relationship via join table
//
// # Hook Types
//
// HookMetadata.Type follows the pattern "<timing>_<operation>":
//
//   - before_create, after_create
//   - before_update, after_update
//   - before_delete, after_delete
//   - before_save, after_save
//
// # Pattern Categories
//
// PatternMetadata.Category groups patterns by type:
//
//   - hook: Lifecycle hook patterns
//   - validation: Validation patterns
//   - middleware: Middleware chain patterns
//   - query: Common query patterns
//   - relationship: Relationship usage patterns
//
// # Dependency Node Types
//
// DependencyNode.Type identifies different kinds of dependencies:
//
//   - resource: Conduit resource
//   - function: Standalone function
//   - middleware: Middleware component
//
// # Dependency Relationship Types
//
// DependencyEdge.Relationship describes how nodes relate:
//
//   - uses: Generic usage dependency
//   - calls: Function/method invocation
//   - belongs_to: Resource relationship
//   - has_many: Collection relationship
//
// # Public API
//
// The package provides an ergonomic public API for runtime introspection
// through the GetRegistry() function:
//
//	registry := metadata.GetRegistry()
//
//	// Query all resources
//	resources := registry.Resources()
//
//	// Query single resource by name
//	post, err := registry.Resource("Post")
//
//	// Query routes with filtering
//	routes := registry.Routes(metadata.RouteFilter{
//		Method: "GET",
//		Resource: "Post",
//	})
//
//	// Query patterns by category
//	hookPatterns := registry.Patterns("hook")
//
//	// Query dependency graph
//	deps, err := registry.Dependencies("Post", metadata.DependencyOptions{
//		Depth: 2,
//		Reverse: false,
//	})
//
//	// Get complete schema
//	schema := registry.GetSchema()
//
// All query methods leverage pre-computed indexes for fast lookups
// (<1ms for typical queries) and return defensive copies to prevent
// external mutation. The API is designed for simplicity, type-safety,
// and error-safety without panics.
package metadata
