package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntrospectStdlibCommand(t *testing.T) {
	t.Run("has correct usage", func(t *testing.T) {
		cmd := newIntrospectStdlibCommand()
		assert.Equal(t, "stdlib [namespace]", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)
	})

	t.Run("accepts zero or one argument", func(t *testing.T) {
		cmd := newIntrospectStdlibCommand()

		// Should accept no arguments
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)

		// Should accept one argument
		err = cmd.Args(cmd, []string{"String"})
		assert.NoError(t, err)

		// Should reject two arguments
		err = cmd.Args(cmd, []string{"String", "Time"})
		assert.Error(t, err)
	})

	t.Run("works without metadata (before build)", func(t *testing.T) {
		// The key feature: stdlib command should work without metadata
		cmd := newIntrospectStdlibCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		// This should not fail even if metadata is not loaded
		err := cmd.RunE(cmd, []string{})
		assert.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "STANDARD LIBRARY FUNCTIONS")
	})
}

func TestRunIntrospectStdlibCommand(t *testing.T) {
	t.Run("lists all namespaces when no filter", func(t *testing.T) {
		cmd := newIntrospectStdlibCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		// Reset global flags
		outputFormat = "table"
		noColor = true

		err := cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		output := buf.String()

		// Should show all namespaces
		assert.Contains(t, output, "String Functions")
		assert.Contains(t, output, "Time Functions")
		assert.Contains(t, output, "Array Functions")
		assert.Contains(t, output, "Hash Functions")
		assert.Contains(t, output, "UUID Functions")

		// Should show total count
		assert.Contains(t, output, "15 total")
	})

	t.Run("lists specific namespace when filtered", func(t *testing.T) {
		cmd := newIntrospectStdlibCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		// Reset global flags
		outputFormat = "table"
		noColor = true

		err := cmd.RunE(cmd, []string{"String"})
		require.NoError(t, err)

		output := buf.String()

		// Should show String namespace
		assert.Contains(t, output, "String Functions (7)")

		// Should NOT show other namespaces
		assert.NotContains(t, output, "Time Functions")
		assert.NotContains(t, output, "Array Functions")

		// Should NOT show total count when filtered
		assert.NotContains(t, output, "15 total")
	})

	t.Run("returns error for invalid namespace", func(t *testing.T) {
		cmd := newIntrospectStdlibCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		// Reset global flags
		outputFormat = "table"
		noColor = true

		err := cmd.RunE(cmd, []string{"InvalidNamespace"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "namespace not found")

		output := buf.String()
		assert.Contains(t, output, "Namespace 'InvalidNamespace' not found")
		assert.Contains(t, output, "Available namespaces:")
	})

	t.Run("shows function signatures", func(t *testing.T) {
		cmd := newIntrospectStdlibCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		// Reset global flags
		outputFormat = "table"
		noColor = true

		err := cmd.RunE(cmd, []string{"String"})
		require.NoError(t, err)

		output := buf.String()

		// Check for specific function signatures
		assert.Contains(t, output, "String.length(s: string!) -> int!")
		assert.Contains(t, output, "String.slugify(s: string!) -> string!")
		assert.Contains(t, output, "String.upcase(s: string!) -> string!")
		assert.Contains(t, output, "String.contains(s: string!, substr: string!) -> bool!")
	})

	t.Run("shows function descriptions", func(t *testing.T) {
		cmd := newIntrospectStdlibCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		// Reset global flags
		outputFormat = "table"
		noColor = true

		err := cmd.RunE(cmd, []string{"Time"})
		require.NoError(t, err)

		output := buf.String()

		// Check for descriptions
		assert.Contains(t, output, "Returns the current timestamp")
		assert.Contains(t, output, "Formats a timestamp as a string")
		assert.Contains(t, output, "Parses a string as a timestamp")
	})

	t.Run("shows nullable return types correctly", func(t *testing.T) {
		cmd := newIntrospectStdlibCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		// Reset global flags
		outputFormat = "table"
		noColor = true

		err := cmd.RunE(cmd, []string{"Time"})
		require.NoError(t, err)

		output := buf.String()

		// Time.parse returns timestamp? (nullable)
		assert.Contains(t, output, "parse(s: string!, layout: string!) -> timestamp?")
		assert.Contains(t, output, "returns null on error")
	})
}

func TestStdlibJSONOutput(t *testing.T) {
	t.Run("outputs valid JSON for all namespaces", func(t *testing.T) {
		cmd := newIntrospectStdlibCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		// Set JSON output format
		outputFormat = "json"

		err := cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		// Parse JSON
		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		// Verify structure
		assert.Contains(t, result, "total_count")
		assert.Contains(t, result, "namespaces")

		totalCount, ok := result["total_count"].(float64)
		require.True(t, ok)
		assert.Equal(t, float64(15), totalCount)

		namespaces, ok := result["namespaces"].([]interface{})
		require.True(t, ok)
		assert.Len(t, namespaces, 5)
	})

	t.Run("outputs valid JSON for single namespace", func(t *testing.T) {
		cmd := newIntrospectStdlibCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		// Set JSON output format
		outputFormat = "json"

		err := cmd.RunE(cmd, []string{"String"})
		require.NoError(t, err)

		// Parse JSON
		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		// Verify structure
		assert.Contains(t, result, "total_count")
		assert.Contains(t, result, "namespaces")

		totalCount, ok := result["total_count"].(float64)
		require.True(t, ok)
		assert.Equal(t, float64(7), totalCount)

		namespaces, ok := result["namespaces"].([]interface{})
		require.True(t, ok)
		assert.Len(t, namespaces, 1)

		// Verify namespace content
		namespace := namespaces[0].(map[string]interface{})
		assert.Equal(t, "String", namespace["namespace"])

		functions, ok := namespace["functions"].([]interface{})
		require.True(t, ok)
		assert.Len(t, functions, 7)

		// Verify function structure
		fn := functions[0].(map[string]interface{})
		assert.Contains(t, fn, "name")
		assert.Contains(t, fn, "signature")
		assert.Contains(t, fn, "description")
	})

	t.Run("JSON includes all required fields", func(t *testing.T) {
		cmd := newIntrospectStdlibCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		// Set JSON output format
		outputFormat = "json"

		err := cmd.RunE(cmd, []string{"UUID"})
		require.NoError(t, err)

		// Parse JSON
		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		namespaces := result["namespaces"].([]interface{})
		namespace := namespaces[0].(map[string]interface{})
		functions := namespace["functions"].([]interface{})
		fn := functions[0].(map[string]interface{})

		// Verify all fields are present and non-empty
		name, ok := fn["name"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, name)

		signature, ok := fn["signature"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, signature)

		description, ok := fn["description"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, description)
	})
}

func TestStdlibNamespaceFiltering(t *testing.T) {
	tests := []struct {
		namespace     string
		expectedCount int
		shouldError   bool
	}{
		{"String", 7, false},
		{"Time", 4, false},
		{"Array", 2, false},
		{"Hash", 1, false},
		{"UUID", 1, false},
		{"Invalid", 0, true},
		{"string", 0, true}, // Case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.namespace, func(t *testing.T) {
			cmd := newIntrospectStdlibCommand()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Set JSON output for easier parsing
			outputFormat = "json"

			err := cmd.RunE(cmd, []string{tt.namespace})

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)

				// Parse and verify count
				var result map[string]interface{}
				err = json.Unmarshal(buf.Bytes(), &result)
				require.NoError(t, err)

				totalCount, ok := result["total_count"].(float64)
				require.True(t, ok)
				assert.Equal(t, float64(tt.expectedCount), totalCount)
			}
		})
	}
}

func TestStdlibCommandIntegration(t *testing.T) {
	t.Run("is registered as subcommand of introspect", func(t *testing.T) {
		rootCmd := NewIntrospectCommand()

		// Find stdlib subcommand
		subCmd, _, err := rootCmd.Find([]string{"stdlib"})
		require.NoError(t, err)
		assert.Equal(t, "stdlib", subCmd.Name())
	})

	t.Run("inherits format flag from parent", func(t *testing.T) {
		rootCmd := NewIntrospectCommand()

		// Set format flag on parent
		err := rootCmd.PersistentFlags().Set("format", "json")
		require.NoError(t, err)

		// Execute stdlib subcommand
		var buf bytes.Buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetArgs([]string{"stdlib", "UUID"})

		err = rootCmd.Execute()
		require.NoError(t, err)

		// Should output JSON
		var result map[string]interface{}
		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)
	})

	t.Run("works without metadata file", func(t *testing.T) {
		// This is the key feature - stdlib should work before build
		rootCmd := NewIntrospectCommand()

		var buf bytes.Buffer
		rootCmd.SetOut(&buf)
		rootCmd.SetArgs([]string{"stdlib"})

		// Should not require metadata loading
		err := rootCmd.Execute()
		assert.NoError(t, err)

		output := buf.String()
		assert.NotEmpty(t, output)
		assert.Contains(t, output, "String Functions")
	})
}

func TestStdlibOutputFormatting(t *testing.T) {
	t.Run("uses namespaced function names", func(t *testing.T) {
		cmd := newIntrospectStdlibCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		outputFormat = "table"
		noColor = true

		err := cmd.RunE(cmd, []string{"String"})
		require.NoError(t, err)

		output := buf.String()

		// All functions should be prefixed with namespace
		assert.Contains(t, output, "String.length")
		assert.Contains(t, output, "String.slugify")
		assert.Contains(t, output, "String.upcase")

		// Should NOT contain bare function names
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			// Skip header and description lines
			if !strings.Contains(line, "(") || !strings.Contains(line, "->") {
				continue
			}
			// Function signature lines must have namespace prefix
			if strings.Contains(line, "length") || strings.Contains(line, "slugify") {
				assert.Contains(t, line, "String.")
			}
		}
	})

	t.Run("shows correct parameter counts", func(t *testing.T) {
		cmd := newIntrospectStdlibCommand()
		var buf bytes.Buffer
		cmd.SetOut(&buf)

		outputFormat = "table"
		noColor = true

		err := cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		output := buf.String()

		// No-parameter functions
		assert.Contains(t, output, "now()")
		assert.Contains(t, output, "generate()")

		// Multi-parameter functions
		assert.Contains(t, output, "contains(s: string!, substr: string!)")
		assert.Contains(t, output, "replace(s: string!, old: string!, new: string!)")
	})
}
