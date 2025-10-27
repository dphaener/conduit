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

	t.Run("returns error when registry not initialized", func(t *testing.T) {
		// Reset registry to ensure it's empty
		metadata.Reset()

		cmd := newIntrospectResourcesCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err := cmd.RunE(cmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registry not initialized")
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
		assert.Contains(t, output, "4 fields")  // User has 4 fields
		assert.Contains(t, output, "2 relationships")  // User has 2 relationships
		assert.Contains(t, output, "1 hook")  // User has 1 hook

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

	t.Run("returns not implemented error", func(t *testing.T) {
		cmd := newIntrospectResourceCommand()
		err := cmd.RunE(cmd, []string{"Post"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
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

	t.Run("returns not implemented error", func(t *testing.T) {
		cmd := newIntrospectRoutesCommand()
		err := cmd.RunE(cmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
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

	t.Run("returns not implemented error", func(t *testing.T) {
		cmd := newIntrospectDepsCommand()
		err := cmd.RunE(cmd, []string{"Post"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
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

	t.Run("returns not implemented error", func(t *testing.T) {
		cmd := newIntrospectPatternsCommand()
		err := cmd.RunE(cmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
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
			"zebra": "z",
			"apple": "a",
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
		// Expected to fail with "registry not initialized" since we reset it
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registry not initialized")

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
