package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/conduit-lang/conduit/internal/cli/ui"
	"github.com/conduit-lang/conduit/runtime/metadata"
)

func TestIntrospectCommand(t *testing.T) {
	t.Run("has correct usage", func(t *testing.T) {
		cmd := NewIntrospectCommand()
		assert.Equal(t, "introspect", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)
	})

	t.Run("has global flags", func(t *testing.T) {
		cmd := NewIntrospectCommand()

		formatFlag := cmd.PersistentFlags().Lookup("format")
		require.NotNil(t, formatFlag)
		assert.Equal(t, "table", formatFlag.DefValue)

		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		require.NotNil(t, verboseFlag)
		assert.Equal(t, "false", verboseFlag.DefValue)

		noColorFlag := cmd.PersistentFlags().Lookup("no-color")
		require.NotNil(t, noColorFlag)
		assert.Equal(t, "false", noColorFlag.DefValue)
	})

	t.Run("has all subcommands", func(t *testing.T) {
		cmd := NewIntrospectCommand()

		expectedCommands := []string{
			"resources",
			"resource",
			"routes",
			"deps",
			"patterns",
		}

		for _, name := range expectedCommands {
			subCmd, _, err := cmd.Find([]string{name})
			require.NoError(t, err)
			assert.Equal(t, name, subCmd.Name())
		}
	})
}

func TestIntrospectResourcesCommand(t *testing.T) {
	t.Run("has correct usage", func(t *testing.T) {
		cmd := newIntrospectResourcesCommand()
		assert.Equal(t, "resources", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)
	})

	t.Run("accepts no arguments", func(t *testing.T) {
		cmd := newIntrospectResourcesCommand()
		// Resources command accepts no arguments (no Args validator set)
		if cmd.Args != nil {
			err := cmd.Args(cmd, []string{})
			assert.NoError(t, err)
		}
	})

	t.Run("returns error when registry not initialized", func(t *testing.T) {
		// This will be tested in integration tests below
	})
}

func TestRunIntrospectResourcesCommand(t *testing.T) {
	// Helper to create test metadata
	createTestMetadata := func() *metadata.Metadata {
		return &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{
				{
					Name: "User",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true},
						{Name: "email", Type: "string", Required: true},
						{Name: "name", Type: "string", Required: true},
						{Name: "bio", Type: "text", Nullable: true},
					},
					Relationships: []metadata.RelationshipMetadata{
						{Name: "posts", Type: "has_many", TargetResource: "Post"},
						{Name: "comments", Type: "has_many", TargetResource: "Comment"},
					},
					Hooks: []metadata.HookMetadata{
						{Type: "before_create", Transaction: true},
					},
					Middleware: map[string][]string{
						"create": {"auth", "validate"},
						"update": {"auth", "validate"},
					},
				},
				{
					Name: "Post",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true},
						{Name: "title", Type: "string", Required: true},
						{Name: "slug", Type: "string", Required: true},
						{Name: "content", Type: "text", Required: true},
						{Name: "author_id", Type: "uuid", Required: true},
					},
					Relationships: []metadata.RelationshipMetadata{
						{Name: "author", Type: "belongs_to", TargetResource: "User"},
						{Name: "comments", Type: "has_many", TargetResource: "Comment"},
						{Name: "tags", Type: "has_many_through", TargetResource: "Tag"},
					},
					Hooks: []metadata.HookMetadata{
						{Type: "before_create", Transaction: true},
						{Type: "after_create", Async: true},
					},
					Middleware: map[string][]string{
						"list": {"cache"},
					},
				},
				{
					Name: "Comment",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true},
						{Name: "content", Type: "text", Required: true},
						{Name: "author_id", Type: "uuid", Required: true},
						{Name: "post_id", Type: "uuid", Required: true},
					},
					Relationships: []metadata.RelationshipMetadata{
						{Name: "author", Type: "belongs_to", TargetResource: "User"},
						{Name: "post", Type: "belongs_to", TargetResource: "Post"},
					},
					Hooks: []metadata.HookMetadata{
						{Type: "after_create", Async: true},
					},
				},
				{
					Name: "Category",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true},
						{Name: "name", Type: "string", Required: true},
					},
					Relationships: []metadata.RelationshipMetadata{
						{Name: "posts", Type: "has_many", TargetResource: "Post"},
					},
				},
				{
					Name: "Tag",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true},
						{Name: "name", Type: "string", Required: true},
					},
				},
			},
		}
	}

	t.Run("handles empty registry gracefully", func(t *testing.T) {
		// Setup empty registry
		metadata.Reset()
		emptyMeta := &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{},
		}
		data, err := json.Marshal(emptyMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		cmd := newIntrospectResourcesCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "No resources found")
	})

	t.Run("formats default table output", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadata()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		// Reset global flags
		outputFormat = "table"
		verbose = false
		noColor = false

		cmd := newIntrospectResourcesCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		output := buf.String()

		// Check header
		assert.Contains(t, output, "RESOURCES (5 total)")

		// Check categories
		assert.Contains(t, output, "Core Resources:")
		assert.Contains(t, output, "Administrative:")

		// Check resources
		assert.Contains(t, output, "User")
		assert.Contains(t, output, "Post")
		assert.Contains(t, output, "Comment")
		assert.Contains(t, output, "Category")
		assert.Contains(t, output, "Tag")

		// Check counts
		assert.Contains(t, output, "4 fields")        // User has 4 fields
		assert.Contains(t, output, "2 relationships") // User has 2 relationships
		assert.Contains(t, output, "1 hook")          // User has 1 hook

		// Check flags
		assert.Contains(t, output, "auth required")
		assert.Contains(t, output, "cached")
		assert.Contains(t, output, "nested")
	})

	t.Run("formats verbose table output", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadata()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		// Set verbose flag
		outputFormat = "table"
		verbose = true
		noColor = false

		cmd := newIntrospectResourcesCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		output := buf.String()

		// In verbose mode, should show detailed breakdown
		assert.Contains(t, output, "User")
		assert.Contains(t, output, "Fields: 4")
		assert.Contains(t, output, "Relationships: 2")
		assert.Contains(t, output, "Hooks: 1")
		assert.Contains(t, output, "Flags: auth required")

		// Reset verbose flag
		verbose = false
	})

	t.Run("formats JSON output", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadata()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		// Set JSON format
		outputFormat = "json"
		verbose = false
		noColor = false

		cmd := newIntrospectResourcesCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		// Parse JSON output
		var result struct {
			TotalCount int `json:"total_count"`
			Resources  []struct {
				Name              string   `json:"name"`
				FieldCount        int      `json:"field_count"`
				RelationshipCount int      `json:"relationship_count"`
				HookCount         int      `json:"hook_count"`
				Category          string   `json:"category"`
				Flags             []string `json:"flags"`
			} `json:"resources"`
		}

		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		// Verify JSON structure
		assert.Equal(t, 5, result.TotalCount)
		assert.Len(t, result.Resources, 5)

		// Find User resource
		var userResource *struct {
			Name              string   `json:"name"`
			FieldCount        int      `json:"field_count"`
			RelationshipCount int      `json:"relationship_count"`
			HookCount         int      `json:"hook_count"`
			Category          string   `json:"category"`
			Flags             []string `json:"flags"`
		}
		for i := range result.Resources {
			if result.Resources[i].Name == "User" {
				userResource = &result.Resources[i]
				break
			}
		}

		require.NotNil(t, userResource)
		assert.Equal(t, 4, userResource.FieldCount)
		assert.Equal(t, 2, userResource.RelationshipCount)
		assert.Equal(t, 1, userResource.HookCount)
		assert.Equal(t, "Core Resources", userResource.Category)
		assert.Contains(t, userResource.Flags, "auth_required")

		// Reset format
		outputFormat = "table"
	})

	t.Run("categorizes resources correctly", func(t *testing.T) {
		tests := []struct {
			name     string
			expected string
		}{
			{"User", "Core Resources"},
			{"Post", "Core Resources"},
			{"Comment", "Core Resources"},
			{"Article", "Core Resources"},
			{"Category", "Administrative"},
			{"Tag", "Administrative"},
			{"Setting", "Administrative"},
			{"Log", "System"},
			{"Session", "System"},
			{"Unknown", "Core Resources"}, // Default
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := categorizeResource(tt.name)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("handles empty registry", func(t *testing.T) {
		// Setup empty registry
		metadata.Reset()
		emptyMeta := &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{},
		}
		data, err := json.Marshal(emptyMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		verbose = false
		noColor = false

		cmd := newIntrospectResourcesCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "No resources found")
	})

	t.Run("respects no-color flag", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadata()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		// Set no-color flag
		outputFormat = "table"
		verbose = false
		noColor = true

		cmd := newIntrospectResourcesCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		// Execute command
		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		// Note: Testing actual color removal is complex as it depends on the color library
		// We just verify the command runs successfully with the flag set
		output := buf.String()
		assert.NotEmpty(t, output)

		// Reset no-color flag
		noColor = false
	})

	// Cleanup after tests
	t.Cleanup(func() {
		metadata.Reset()
		outputFormat = "table"
		verbose = false
		noColor = false
	})
}

// BenchmarkIntrospectResourcesCommand benchmarks the resources command performance
func BenchmarkIntrospectResourcesCommand(b *testing.B) {
	// Setup test registry with realistic data
	testMeta := &metadata.Metadata{
		Version:   "1.0.0",
		Generated: time.Now(),
		Resources: make([]metadata.ResourceMetadata, 0, 50),
	}

	// Create 50 resources to simulate a realistic application
	for i := 0; i < 50; i++ {
		res := metadata.ResourceMetadata{
			Name: fmt.Sprintf("Resource%d", i),
			Fields: []metadata.FieldMetadata{
				{Name: "id", Type: "uuid", Required: true},
				{Name: "name", Type: "string", Required: true},
				{Name: "description", Type: "text", Nullable: true},
				{Name: "created_at", Type: "timestamp", Required: true},
				{Name: "updated_at", Type: "timestamp", Required: true},
			},
			Relationships: []metadata.RelationshipMetadata{
				{Name: "parent", Type: "belongs_to", TargetResource: "Parent"},
				{Name: "children", Type: "has_many", TargetResource: "Child"},
			},
			Hooks: []metadata.HookMetadata{
				{Type: "before_create", Transaction: true},
			},
		}
		testMeta.Resources = append(testMeta.Resources, res)
	}

	metadata.Reset()
	data, err := json.Marshal(testMeta)
	if err != nil {
		b.Fatal(err)
	}
	err = metadata.RegisterMetadata(data)
	if err != nil {
		b.Fatal(err)
	}

	// Reset flags
	outputFormat = "table"
	verbose = false
	noColor = true // Disable color for consistent benchmarking

	cmd := newIntrospectResourcesCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		err := cmd.RunE(cmd, []string{})
		if err != nil {
			b.Fatal(err)
		}
	}

	b.Cleanup(func() {
		metadata.Reset()
	})
}

func TestIntrospectResourceCommand(t *testing.T) {
	t.Run("has correct usage", func(t *testing.T) {
		cmd := newIntrospectResourceCommand()
		assert.Equal(t, "resource <name>", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		cmd := newIntrospectResourceCommand()

		// No args should fail
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)

		// One arg should succeed
		err = cmd.Args(cmd, []string{"Post"})
		assert.NoError(t, err)

		// Two args should fail
		err = cmd.Args(cmd, []string{"Post", "User"})
		assert.Error(t, err)
	})

	t.Run("returns error when resource not found", func(t *testing.T) {
		// Setup empty registry
		metadata.Reset()
		emptyMeta := &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{},
		}
		data, err := json.Marshal(emptyMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		cmd := newIntrospectResourceCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{"NonExistent"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource not found")
	})
}

func TestIntrospectRoutesCommand(t *testing.T) {
	t.Run("has correct usage", func(t *testing.T) {
		cmd := newIntrospectRoutesCommand()
		assert.Equal(t, "routes", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)
	})

	t.Run("has method flag", func(t *testing.T) {
		cmd := newIntrospectRoutesCommand()
		flag := cmd.Flags().Lookup("method")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
	})

	t.Run("has middleware flag", func(t *testing.T) {
		cmd := newIntrospectRoutesCommand()
		flag := cmd.Flags().Lookup("middleware")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
	})

	t.Run("has resource flag", func(t *testing.T) {
		cmd := newIntrospectRoutesCommand()
		flag := cmd.Flags().Lookup("resource")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
	})

	t.Run("handles empty routes gracefully", func(t *testing.T) {
		metadata.Reset()
		emptyMeta := &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{},
			Routes:    []metadata.RouteMetadata{},
		}
		data, err := json.Marshal(emptyMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		cmd := newIntrospectRoutesCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "No routes found")
	})
}

func TestIntrospectDepsCommand(t *testing.T) {
	t.Run("has correct usage", func(t *testing.T) {
		cmd := newIntrospectDepsCommand()
		assert.Equal(t, "deps <resource>", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		cmd := newIntrospectDepsCommand()

		// No args should fail
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)

		// One arg should succeed
		err = cmd.Args(cmd, []string{"Post"})
		assert.NoError(t, err)

		// Two args should fail
		err = cmd.Args(cmd, []string{"Post", "User"})
		assert.Error(t, err)
	})

	t.Run("has depth flag", func(t *testing.T) {
		cmd := newIntrospectDepsCommand()
		flag := cmd.Flags().Lookup("depth")
		require.NotNil(t, flag)
		assert.Equal(t, "1", flag.DefValue)
	})

	t.Run("has reverse flag", func(t *testing.T) {
		cmd := newIntrospectDepsCommand()
		flag := cmd.Flags().Lookup("reverse")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has type flag", func(t *testing.T) {
		cmd := newIntrospectDepsCommand()
		flag := cmd.Flags().Lookup("type")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
	})

	t.Run("returns registry not initialized error", func(t *testing.T) {
		metadata.Reset()
		cmd := newIntrospectDepsCommand()
		err := cmd.RunE(cmd, []string{"Post"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registry not initialized")
	})
}

func TestIntrospectPatternsCommand(t *testing.T) {
	t.Run("has correct usage", func(t *testing.T) {
		cmd := newIntrospectPatternsCommand()
		assert.Equal(t, "patterns [category]", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)
	})

	t.Run("accepts zero or one argument", func(t *testing.T) {
		cmd := newIntrospectPatternsCommand()

		// No args should succeed
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)

		// One arg should succeed
		err = cmd.Args(cmd, []string{"authentication"})
		assert.NoError(t, err)

		// Two args should fail
		err = cmd.Args(cmd, []string{"authentication", "authorization"})
		assert.Error(t, err)
	})

	t.Run("has min-frequency flag", func(t *testing.T) {
		cmd := newIntrospectPatternsCommand()
		flag := cmd.Flags().Lookup("min-frequency")
		require.NotNil(t, flag)
		assert.Equal(t, "1", flag.DefValue)
	})

	t.Run("handles empty patterns gracefully", func(t *testing.T) {
		metadata.Reset()
		emptyMeta := &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{},
			Patterns:  []metadata.PatternMetadata{},
		}
		data, err := json.Marshal(emptyMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		cmd := newIntrospectPatternsCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "No patterns found")
	})
}

func TestTableFormatter(t *testing.T) {
	t.Run("creates formatter with default writer", func(t *testing.T) {
		formatter := NewTableFormatter(nil)
		assert.NotNil(t, formatter)
		assert.NotNil(t, formatter.writer)
	})

	t.Run("creates formatter with custom writer", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewTableFormatter(buf)
		assert.NotNil(t, formatter)
		assert.Equal(t, buf, formatter.writer)
	})

	t.Run("formats data as table", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewTableFormatter(buf)

		data := map[string]interface{}{"key": "value"}
		err := formatter.Format(data)

		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "key")
		assert.Contains(t, output, "value")
	})

	t.Run("formats map with sorted keys", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewTableFormatter(buf)

		data := map[string]interface{}{
			"zebra":  "z",
			"apple":  "a",
			"banana": "b",
		}
		err := formatter.Format(data)

		require.NoError(t, err)
		output := buf.String()

		// Keys should appear in sorted order
		appleIndex := strings.Index(output, "apple:")
		bananaIndex := strings.Index(output, "banana:")
		zebraIndex := strings.Index(output, "zebra:")

		assert.True(t, appleIndex < bananaIndex, "apple should come before banana")
		assert.True(t, bananaIndex < zebraIndex, "banana should come before zebra")
	})

	t.Run("formats slice with numbered items", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewTableFormatter(buf)

		data := []interface{}{"first", "second", "third"}
		err := formatter.Format(data)

		require.NoError(t, err)
		output := buf.String()

		assert.Contains(t, output, "1. first")
		assert.Contains(t, output, "2. second")
		assert.Contains(t, output, "3. third")
	})

	t.Run("formats fallback for other types", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewTableFormatter(buf)

		data := "simple string"
		err := formatter.Format(data)

		require.NoError(t, err)
		output := buf.String()
		assert.Contains(t, output, "simple string")
	})
}

func TestJSONFormatter(t *testing.T) {
	t.Run("creates formatter with default writer", func(t *testing.T) {
		formatter := NewJSONFormatter(nil)
		assert.NotNil(t, formatter)
		assert.NotNil(t, formatter.writer)
	})

	t.Run("creates formatter with custom writer", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewJSONFormatter(buf)
		assert.NotNil(t, formatter)
		assert.Equal(t, buf, formatter.writer)
	})

	t.Run("formats data as JSON", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewJSONFormatter(buf)

		data := map[string]string{"key": "value"}
		err := formatter.Format(data)

		require.NoError(t, err)

		// Verify it's valid JSON
		var result map[string]string
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "value", result["key"])
	})

	t.Run("formats with indentation", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter := NewJSONFormatter(buf)

		data := map[string]interface{}{
			"key":    "value",
			"nested": map[string]string{"inner": "data"},
		}
		err := formatter.Format(data)

		require.NoError(t, err)
		output := buf.String()

		// Check for indentation (should have spaces)
		assert.Contains(t, output, "  ")
	})
}

func TestGetFormatter(t *testing.T) {
	t.Run("returns JSON formatter for json format", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter, err := GetFormatter("json", buf)

		require.NoError(t, err)
		assert.IsType(t, &JSONFormatter{}, formatter)
	})

	t.Run("returns table formatter for table format", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter, err := GetFormatter("table", buf)

		require.NoError(t, err)
		assert.IsType(t, &TableFormatter{}, formatter)
	})

	t.Run("handles uppercase format", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter, err := GetFormatter("JSON", buf)

		require.NoError(t, err)
		assert.IsType(t, &JSONFormatter{}, formatter)
	})

	t.Run("returns error for invalid format", func(t *testing.T) {
		buf := &bytes.Buffer{}
		formatter, err := GetFormatter("invalid", buf)

		require.Error(t, err)
		assert.Nil(t, formatter)
		assert.Contains(t, err.Error(), "unsupported format")
	})

	t.Run("uses os.Stdout when writer is nil", func(t *testing.T) {
		formatter, err := GetFormatter("json", nil)

		require.NoError(t, err)
		assert.IsType(t, &JSONFormatter{}, formatter)
	})
}

func TestFlagParsing(t *testing.T) {
	t.Run("parses format flag", func(t *testing.T) {
		// Reset registry for this test
		metadata.Reset()

		cmd := NewIntrospectCommand()
		cmd.SetArgs([]string{"resources", "--format", "json"})

		err := cmd.Execute()
		// Expected to fail since metadata file doesn't exist
		require.Error(t, err)
		assert.Contains(t, err.Error(), "metadata file not found")

		// Check the flag was set correctly
		formatFlag := cmd.PersistentFlags().Lookup("format")
		require.NotNil(t, formatFlag)
		assert.Equal(t, "json", formatFlag.Value.String())
	})

	t.Run("parses verbose flag", func(t *testing.T) {
		// Reset registry for this test
		metadata.Reset()

		cmd := NewIntrospectCommand()
		cmd.SetArgs([]string{"resources", "--verbose"})

		err := cmd.Execute()
		require.Error(t, err)

		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		require.NotNil(t, verboseFlag)
		assert.Equal(t, "true", verboseFlag.Value.String())
	})

	t.Run("parses no-color flag", func(t *testing.T) {
		// Reset registry for this test
		metadata.Reset()

		cmd := NewIntrospectCommand()
		cmd.SetArgs([]string{"resources", "--no-color"})

		err := cmd.Execute()
		require.Error(t, err)

		noColorFlag := cmd.PersistentFlags().Lookup("no-color")
		require.NotNil(t, noColorFlag)
		assert.Equal(t, "true", noColorFlag.Value.String())
	})

	t.Run("parses multiple flags together", func(t *testing.T) {
		// Reset registry for this test
		metadata.Reset()

		cmd := NewIntrospectCommand()
		cmd.SetArgs([]string{"resources", "--format", "json", "--verbose", "--no-color"})

		err := cmd.Execute()
		require.Error(t, err)

		formatFlag := cmd.PersistentFlags().Lookup("format")
		verboseFlag := cmd.PersistentFlags().Lookup("verbose")
		noColorFlag := cmd.PersistentFlags().Lookup("no-color")

		assert.Equal(t, "json", formatFlag.Value.String())
		assert.Equal(t, "true", verboseFlag.Value.String())
		assert.Equal(t, "true", noColorFlag.Value.String())
	})

	t.Run("parses command-specific flags", func(t *testing.T) {
		cmd := NewIntrospectCommand()
		cmd.SetArgs([]string{"routes", "--method", "GET", "--middleware", "auth"})

		err := cmd.Execute()
		require.Error(t, err)

		routesCmd, _, err := cmd.Find([]string{"routes"})
		require.NoError(t, err)

		methodFlag := routesCmd.Flags().Lookup("method")
		middlewareFlag := routesCmd.Flags().Lookup("middleware")

		require.NotNil(t, methodFlag)
		require.NotNil(t, middlewareFlag)
	})
}

func TestCompletionCommand(t *testing.T) {
	t.Run("has correct usage", func(t *testing.T) {
		cmd := NewCompletionCommand()
		assert.Equal(t, "completion [bash|zsh|fish|powershell]", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
	})

	t.Run("accepts valid shell arguments", func(t *testing.T) {
		cmd := NewCompletionCommand()

		validShells := []string{"bash", "zsh", "fish", "powershell"}
		for _, shell := range validShells {
			err := cmd.Args(cmd, []string{shell})
			assert.NoError(t, err, "should accept %s", shell)
		}
	})

	t.Run("requires exactly one argument", func(t *testing.T) {
		cmd := NewCompletionCommand()

		// No args should fail
		err := cmd.Args(cmd, []string{})
		assert.Error(t, err)

		// Two args should fail
		err = cmd.Args(cmd, []string{"bash", "zsh"})
		assert.Error(t, err)
	})

	t.Run("rejects invalid shell", func(t *testing.T) {
		cmd := NewCompletionCommand()
		err := cmd.Args(cmd, []string{"invalid"})
		assert.Error(t, err)
	})

	t.Run("generates bash completion", func(t *testing.T) {
		// Note: Testing actual bash completion generation is complex
		// We just verify the command structure is correct
		cmd := NewCompletionCommand()
		assert.NotNil(t, cmd.RunE)
	})
}

func TestHelpOutput(t *testing.T) {
	t.Run("introspect command shows help", func(t *testing.T) {
		cmd := NewIntrospectCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"--help"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "introspect")
		assert.Contains(t, output, "Understanding the structure")
		assert.Contains(t, output, "--format")
		assert.Contains(t, output, "--verbose")
		assert.Contains(t, output, "--no-color")
	})

	t.Run("resources command shows examples", func(t *testing.T) {
		cmd := newIntrospectResourcesCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"--help"})

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Example")
		assert.Contains(t, strings.ToLower(output), "conduit introspect resources")
	})
}

func TestRunIntrospectResourceCommand(t *testing.T) {
	// Helper to create test metadata with a detailed Post resource
	createTestMetadataWithPost := func() *metadata.Metadata {
		return &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{
				{
					Name:          "Post",
					Documentation: "Blog post with content and categorization",
					FilePath:      "resources/post.cdt",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true, Constraints: []string{"@primary", "@auto"}},
						{Name: "title", Type: "string", Required: true, Constraints: []string{"@min(5)", "@max(200)"}},
						{Name: "slug", Type: "string", Required: true, Constraints: []string{"@unique"}},
						{Name: "content", Type: "text", Required: true, Constraints: []string{"@min(100)"}},
						{Name: "excerpt", Type: "text", Nullable: true},
						{Name: "author_id", Type: "uuid", Required: true},
					},
					Relationships: []metadata.RelationshipMetadata{
						{
							Name:           "author",
							Type:           "belongs_to",
							TargetResource: "User",
							ForeignKey:     "author_id",
							OnDelete:       "restrict",
						},
						{
							Name:           "comments",
							Type:           "has_many",
							TargetResource: "Comment",
						},
						{
							Name:           "tags",
							Type:           "has_many_through",
							TargetResource: "Tag",
							ThroughTable:   "post_tags",
						},
					},
					Hooks: []metadata.HookMetadata{
						{Type: "before_create", Transaction: true, SourceCode: "self.slug = String.slugify(self.title)"},
						{Type: "after_create", Async: true},
					},
					Constraints: []metadata.ConstraintMetadata{
						{
							Name:       "published_requires_content",
							Operations: []string{"create", "update"},
							When:       "self.status == \"published\"",
							Condition:  "String.length(self.content) >= 500",
							Error:      "Published posts need 500+ characters",
						},
					},
					Validations: []metadata.ValidationMetadata{
						{Field: "title", Type: "min", Value: "5"},
						{Field: "title", Type: "max", Value: "200"},
					},
					Middleware: map[string][]string{
						"create": {"auth", "rate_limit(5/hour)"},
						"list":   {"cache(300)"},
					},
				},
				{
					Name: "User",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true},
						{Name: "email", Type: "string", Required: true},
					},
				},
			},
			Routes: []metadata.RouteMetadata{
				{
					Method:     "GET",
					Path:       "/posts",
					Handler:    "PostHandler.List",
					Resource:   "Post",
					Operation:  "list",
					Middleware: []string{"cache(300)"},
				},
				{
					Method:     "POST",
					Path:       "/posts",
					Handler:    "PostHandler.Create",
					Resource:   "Post",
					Operation:  "create",
					Middleware: []string{"auth", "rate_limit(5/hour)"},
				},
			},
		}
	}

	t.Run("formats table output correctly", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadataWithPost()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		// Reset global flags
		outputFormat = "table"
		verbose = false
		noColor = true // Disable color for testing

		cmd := newIntrospectResourceCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{"Post"})
		require.NoError(t, err)

		output := buf.String()

		// Check header
		assert.Contains(t, output, "RESOURCE: Post")
		assert.Contains(t, output, "File: resources/post.cdt")
		assert.Contains(t, output, "Docs: Blog post with content and categorization")

		// Check schema section
		assert.Contains(t, output, "SCHEMA")
		assert.Contains(t, output, "FIELDS (6)")
		assert.Contains(t, output, "Required (5)")
		assert.Contains(t, output, "Optional (1)")

		// Check fields
		assert.Contains(t, output, "id")
		assert.Contains(t, output, "uuid")
		assert.Contains(t, output, "title")
		assert.Contains(t, output, "string")
		assert.Contains(t, output, "@min(5)")
		assert.Contains(t, output, "@max(200)")

		// Check relationships
		assert.Contains(t, output, "RELATIONSHIPS (3)")
		assert.Contains(t, output, "author")
		assert.Contains(t, output, "belongs_to User")
		assert.Contains(t, output, "Foreign key: author_id")
		assert.Contains(t, output, "On delete: restrict")

		// Check behavior section
		assert.Contains(t, output, "BEHAVIOR")
		assert.Contains(t, output, "LIFECYCLE HOOKS")
		assert.Contains(t, output, "@before_create")
		assert.Contains(t, output, "@after_create")

		// Check constraints
		assert.Contains(t, output, "CONSTRAINTS (1)")
		assert.Contains(t, output, "published_requires_content")

		// Check API endpoints
		assert.Contains(t, output, "API ENDPOINTS")
		assert.Contains(t, output, "GET /posts")
		assert.Contains(t, output, "POST /posts")
		assert.Contains(t, output, "cache(300)")
		assert.Contains(t, output, "auth")
	})

	t.Run("formats verbose table output correctly", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadataWithPost()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		// Set verbose flag
		outputFormat = "table"
		verbose = true
		noColor = true

		cmd := newIntrospectResourceCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{"Post"})
		require.NoError(t, err)

		output := buf.String()

		// In verbose mode, should show more details
		assert.Contains(t, output, "CONSTRAINTS (1)")
		assert.Contains(t, output, "Operations:")
		assert.Contains(t, output, "Condition:")
		assert.Contains(t, output, "Error:")

		assert.Contains(t, output, "MIDDLEWARE BY OPERATION")
		assert.Contains(t, output, "create:")
		assert.Contains(t, output, "list:")

		// Reset verbose flag
		verbose = false
	})

	t.Run("formats JSON output correctly", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadataWithPost()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		// Set JSON format
		outputFormat = "json"
		verbose = false
		noColor = true

		cmd := newIntrospectResourceCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{"Post"})
		require.NoError(t, err)

		// Parse JSON output
		var result metadata.ResourceMetadata
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		// Verify JSON structure
		assert.Equal(t, "Post", result.Name)
		assert.Equal(t, "resources/post.cdt", result.FilePath)
		assert.Equal(t, "Blog post with content and categorization", result.Documentation)
		assert.Len(t, result.Fields, 6)
		assert.Len(t, result.Relationships, 3)
		assert.Len(t, result.Hooks, 2)
		assert.Len(t, result.Constraints, 1)

		// Reset format
		outputFormat = "table"
	})

	t.Run("returns error for non-existent resource", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadataWithPost()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		verbose = false
		noColor = true

		cmd := newIntrospectResourceCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{"NonExistent"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource not found")

		output := buf.String()
		assert.Contains(t, output, "RESOURCE NOT FOUND")
		assert.Contains(t, output, "NonExistent")
		assert.Contains(t, output, "Available resources:")
		assert.Contains(t, output, "Post")
		assert.Contains(t, output, "User")
	})

	t.Run("suggests similar resource names on typo", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := createTestMetadataWithPost()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		verbose = false
		noColor = true

		cmd := newIntrospectResourceCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		// Try with typo "Pst" instead of "Post"
		err = cmd.RunE(cmd, []string{"Pst"})
		require.Error(t, err)

		output := buf.String()
		assert.Contains(t, output, "Did you mean:")
		assert.Contains(t, output, "Post")
	})

	// Cleanup after tests
	t.Cleanup(func() {
		metadata.Reset()
		outputFormat = "table"
		verbose = false
		noColor = false
	})
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"cat", "cat", 0},
		{"cat", "bat", 1},
		{"cat", "car", 1},
		{"cat", "cut", 1},
		{"cat", "cats", 1},
		{"kitten", "sitting", 3},
		{"Post", "Pst", 1},
		{"User", "Usr", 1},
		{"Comment", "Coment", 1},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s->%s", tt.s1, tt.s2), func(t *testing.T) {
			result := ui.LevenshteinDistance(tt.s1, tt.s2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindSimilarResourceNames(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "exact typo - one letter off",
			input:    "Pst",
			expected: []string{"Post"},
		},
		{
			name:     "close match",
			input:    "Usr",
			expected: []string{"User"},
		},
		{
			name:     "two letters off",
			input:    "Coment",
			expected: []string{"Comment"},
		},
		{
			name:     "no close matches",
			input:    "xyz",
			expected: []string{},
		},
		{
			name:     "multiple matches - sorted by distance",
			input:    "Cat",
			expected: []string{"Tag"}, // Cat->Tag=1 (Cat->Category=5 is >3 so won't match)
		},
	}

	candidates := []string{"Post", "User", "Comment", "Category", "Tag"}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ui.FindSimilar(tt.input, candidates, nil)

			// Check we got at least the expected suggestions (there may be more)
			for _, expected := range tt.expected {
				found := false
				for _, r := range result {
					if r == expected {
						found = true
						break
					}
				}
				if !found && len(tt.expected) > 0 {
					// Only assert if we expected suggestions
					assert.Contains(t, result, expected)
				}
			}
		})
	}
}

func TestHandleResourceNotFound(t *testing.T) {
	t.Run("shows error message and suggestions", func(t *testing.T) {
		// Setup test registry
		metadata.Reset()
		testMeta := &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{
				{Name: "Post"},
				{Name: "User"},
				{Name: "Comment"},
			},
		}
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		noColor = true // Disable color for testing

		buf := &bytes.Buffer{}
		err = handleResourceNotFound("Pst", buf)

		require.Error(t, err)
		output := buf.String()

		// Check for new UI error format
		assert.Contains(t, output, "RESOURCE NOT FOUND")
		assert.Contains(t, output, "Pst")
		assert.Contains(t, output, "Did you mean:")
		assert.Contains(t, output, "Post")
		// The new format shows "See all resources" command instead of listing them
		assert.Contains(t, output, "conduit introspect resources")
	})

	t.Run("returns error for resource not found", func(t *testing.T) {
		metadata.Reset()
		testMeta := &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{
				{Name: "Post"},
			},
		}
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)
		noColor = true

		buf := &bytes.Buffer{}
		err = handleResourceNotFound("NonExistent", buf)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "resource not found")
	})

	t.Cleanup(func() {
		metadata.Reset()
		noColor = false
	})
}

func TestMultipleHooksOfSameType(t *testing.T) {
	// Test the fix for Issue #1 and #2: handling multiple hooks of the same type
	createTestMetadataWithMultipleHooks := func() *metadata.Metadata {
		return &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{
				{
					Name: "TestResource",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true},
					},
					Hooks: []metadata.HookMetadata{
						{Type: "before_create", Transaction: true, SourceCode: "// Hook 1: Validate data\nself.validate()"},
						{Type: "before_create", Transaction: false, SourceCode: "// Hook 2: Set defaults\nself.setDefaults()"},
						{Type: "before_create", Async: true, SourceCode: "// Hook 3: Log creation\nLog.info('Creating resource')"},
					},
				},
			},
		}
	}

	t.Run("displays all hooks of same type with aggregated flags", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithMultipleHooks()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		verbose = false
		noColor = true

		cmd := newIntrospectResourceCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{"TestResource"})
		require.NoError(t, err)

		output := buf.String()

		// Should show hook type with aggregated flags
		assert.Contains(t, output, "@before_create")
		assert.Contains(t, output, "[async, transaction]") // Both flags should appear, sorted
	})

	t.Run("displays all hook source code in verbose mode", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithMultipleHooks()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		verbose = true
		noColor = true

		cmd := newIntrospectResourceCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{"TestResource"})
		require.NoError(t, err)

		output := buf.String()

		// All three hooks' source code should be displayed
		assert.Contains(t, output, "Hook 1:")
		assert.Contains(t, output, "Hook 2:")
		assert.Contains(t, output, "Hook 3:")
		assert.Contains(t, output, "// Hook 1: Validate data")
		assert.Contains(t, output, "// Hook 2: Set defaults")
		assert.Contains(t, output, "// Hook 3: Log creation")
	})

	t.Run("handles single hook without numbering", func(t *testing.T) {
		metadata.Reset()
		singleHookMeta := &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{
				{
					Name: "SingleHookResource",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true},
					},
					Hooks: []metadata.HookMetadata{
						{Type: "before_create", Transaction: true, SourceCode: "self.validate()"},
					},
				},
			},
		}
		data, err := json.Marshal(singleHookMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		verbose = true
		noColor = true

		cmd := newIntrospectResourceCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{"SingleHookResource"})
		require.NoError(t, err)

		output := buf.String()

		// Should NOT show "Hook 1:" prefix for single hook
		assert.NotContains(t, output, "Hook 1:")
		// But should still show the source code
		assert.Contains(t, output, "self.validate()")
	})

	t.Cleanup(func() {
		metadata.Reset()
		outputFormat = "table"
		verbose = false
		noColor = false
	})
}
