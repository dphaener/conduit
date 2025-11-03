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

func TestRunIntrospectRoutesCommand(t *testing.T) {
	// Helper to create test metadata with routes
	createTestMetadataWithRoutes := func() *metadata.Metadata {
		return &metadata.Metadata{
			Version:   "1.0.0",
			Generated: time.Now(),
			Resources: []metadata.ResourceMetadata{
				{
					Name: "Post",
					Fields: []metadata.FieldMetadata{
						{Name: "id", Type: "uuid", Required: true},
						{Name: "title", Type: "string", Required: true},
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
					Path:       "/api/posts",
					Handler:    "Post.list",
					Resource:   "Post",
					Operation:  "list",
					Middleware: []string{"cache(300)"},
				},
				{
					Method:     "GET",
					Path:       "/api/posts/:id",
					Handler:    "Post.get",
					Resource:   "Post",
					Operation:  "get",
					Middleware: []string{"cache(600)"},
				},
				{
					Method:     "POST",
					Path:       "/api/posts",
					Handler:    "Post.create",
					Resource:   "Post",
					Operation:  "create",
					Middleware: []string{"auth", "rate_limit(5/hour)"},
				},
				{
					Method:     "PUT",
					Path:       "/api/posts/:id",
					Handler:    "Post.update",
					Resource:   "Post",
					Operation:  "update",
					Middleware: []string{"auth", "author_or_editor"},
				},
				{
					Method:     "DELETE",
					Path:       "/api/posts/:id",
					Handler:    "Post.delete",
					Resource:   "Post",
					Operation:  "delete",
					Middleware: []string{"auth", "author_or_admin"},
				},
				{
					Method:     "GET",
					Path:       "/api/users",
					Handler:    "User.list",
					Resource:   "User",
					Operation:  "list",
					Middleware: []string{"auth"},
				},
				{
					Method:     "POST",
					Path:       "/api/users",
					Handler:    "User.create",
					Resource:   "User",
					Operation:  "create",
					Middleware: []string{"admin"},
				},
			},
		}
	}

	t.Run("formats table output correctly", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithRoutes()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		verbose = false
		noColor = true

		cmd := newIntrospectRoutesCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		output := buf.String()

		// Check all routes are present
		assert.Contains(t, output, "GET")
		assert.Contains(t, output, "POST")
		assert.Contains(t, output, "PUT")
		assert.Contains(t, output, "DELETE")
		assert.Contains(t, output, "/api/posts")
		assert.Contains(t, output, "/api/posts/:id")
		assert.Contains(t, output, "/api/users")

		// Check handlers
		assert.Contains(t, output, "Post.list")
		assert.Contains(t, output, "Post.create")
		assert.Contains(t, output, "User.list")

		// Check middleware is formatted correctly
		assert.Contains(t, output, "[cache(300)]")
		assert.Contains(t, output, "[auth, rate_limit(5/hour)]")
		assert.Contains(t, output, "[auth, author_or_editor]")

		// Verify routes are sorted by path
		lines := strings.Split(output, "\n")
		var routeLines []string
		for _, line := range lines {
			if strings.Contains(line, "/api/") {
				routeLines = append(routeLines, line)
			}
		}

		// /api/posts should come before /api/posts/:id which comes before /api/users
		postsIdx := -1
		postsIdIdx := -1
		usersIdx := -1

		for i, line := range routeLines {
			if strings.Contains(line, "/api/posts") && !strings.Contains(line, ":id") {
				if postsIdx == -1 {
					postsIdx = i
				}
			} else if strings.Contains(line, "/api/posts/:id") {
				if postsIdIdx == -1 {
					postsIdIdx = i
				}
			} else if strings.Contains(line, "/api/users") && !strings.Contains(line, ":id") {
				if usersIdx == -1 {
					usersIdx = i
				}
			}
		}

		assert.True(t, postsIdx < postsIdIdx, "routes should be sorted by path")
		assert.True(t, postsIdIdx < usersIdx, "routes should be sorted by path")
	})

	t.Run("filters by HTTP method", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithRoutes()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectRoutesCommand()
		cmd.SetArgs([]string{"--method", "GET"})
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()

		// Should only show GET routes
		assert.Contains(t, output, "GET")
		assert.NotContains(t, output, "POST")
		assert.NotContains(t, output, "PUT")
		assert.NotContains(t, output, "DELETE")

		// Should show all GET routes
		assert.Contains(t, output, "/api/posts")
		assert.Contains(t, output, "/api/posts/:id")
		assert.Contains(t, output, "/api/users")
	})

	t.Run("filters by HTTP method case-insensitive", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithRoutes()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectRoutesCommand()
		cmd.SetArgs([]string{"--method", "post"})
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()

		// Should only show POST routes
		assert.Contains(t, output, "POST")
		assert.NotContains(t, output, "GET")
		assert.NotContains(t, output, "PUT")
		assert.NotContains(t, output, "DELETE")
	})

	t.Run("filters by middleware", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithRoutes()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectRoutesCommand()
		cmd.SetArgs([]string{"--middleware", "auth"})
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()

		// Should show all routes with auth middleware
		assert.Contains(t, output, "Post.create")
		assert.Contains(t, output, "Post.update")
		assert.Contains(t, output, "Post.delete")
		assert.Contains(t, output, "User.list")

		// Should NOT show routes without auth
		assert.NotContains(t, output, "Post.list")
		assert.NotContains(t, output, "Post.get")
	})

	t.Run("filters by middleware with substring match", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithRoutes()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectRoutesCommand()
		cmd.SetArgs([]string{"--middleware", "cache"})
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()

		// Should show routes with cache middleware (cache(300) and cache(600))
		assert.Contains(t, output, "Post.list")
		assert.Contains(t, output, "Post.get")

		// Should NOT show routes without cache
		assert.NotContains(t, output, "Post.create")
		assert.NotContains(t, output, "User.list")
	})

	t.Run("filters by resource", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithRoutes()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectRoutesCommand()
		cmd.SetArgs([]string{"--resource", "Post"})
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()

		// Should show all Post routes
		assert.Contains(t, output, "Post.list")
		assert.Contains(t, output, "Post.get")
		assert.Contains(t, output, "Post.create")
		assert.Contains(t, output, "Post.update")
		assert.Contains(t, output, "Post.delete")

		// Should NOT show User routes
		assert.NotContains(t, output, "User.list")
		assert.NotContains(t, output, "User.create")
	})

	t.Run("filters with multiple criteria", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithRoutes()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectRoutesCommand()
		cmd.SetArgs([]string{"--method", "GET", "--middleware", "cache", "--resource", "Post"})
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()

		// Should show only GET routes for Post with cache middleware
		assert.Contains(t, output, "Post.list")
		assert.Contains(t, output, "Post.get")

		// Should NOT show POST routes even though they're for Post
		assert.NotContains(t, output, "Post.create")

		// Should NOT show User routes
		assert.NotContains(t, output, "User.list")
	})

	t.Run("formats JSON output correctly", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithRoutes()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "json"
		noColor = true

		cmd := newIntrospectRoutesCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		// Parse JSON output
		var result struct {
			TotalCount int `json:"total_count"`
			Routes     []struct {
				Method     string   `json:"method"`
				Path       string   `json:"path"`
				Handler    string   `json:"handler"`
				Resource   string   `json:"resource"`
				Operation  string   `json:"operation"`
				Middleware []string `json:"middleware"`
			} `json:"routes"`
		}

		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		// Verify JSON structure
		assert.Equal(t, 7, result.TotalCount)
		assert.Len(t, result.Routes, 7)

		// Find a specific route to verify structure
		var postListRoute *struct {
			Method     string   `json:"method"`
			Path       string   `json:"path"`
			Handler    string   `json:"handler"`
			Resource   string   `json:"resource"`
			Operation  string   `json:"operation"`
			Middleware []string `json:"middleware"`
		}

		for i := range result.Routes {
			if result.Routes[i].Path == "/api/posts" && result.Routes[i].Method == "GET" {
				postListRoute = &result.Routes[i]
				break
			}
		}

		require.NotNil(t, postListRoute)
		assert.Equal(t, "GET", postListRoute.Method)
		assert.Equal(t, "/api/posts", postListRoute.Path)
		assert.Equal(t, "Post.list", postListRoute.Handler)
		assert.Equal(t, "Post", postListRoute.Resource)
		assert.Equal(t, "list", postListRoute.Operation)
		assert.Equal(t, []string{"cache(300)"}, postListRoute.Middleware)

		// Reset format
		outputFormat = "table"
	})

	t.Run("handles empty registry", func(t *testing.T) {
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

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectRoutesCommand()
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.RunE(cmd, []string{})
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "No routes found")
	})

	t.Run("handles filter with no matches", func(t *testing.T) {
		metadata.Reset()
		testMeta := createTestMetadataWithRoutes()
		data, err := json.Marshal(testMeta)
		require.NoError(t, err)
		err = metadata.RegisterMetadata(data)
		require.NoError(t, err)

		outputFormat = "table"
		noColor = true

		cmd := newIntrospectRoutesCommand()
		cmd.SetArgs([]string{"--method", "PATCH"})
		buf := &bytes.Buffer{}
		cmd.SetOut(buf)

		err = cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "No routes found")
	})

	// Cleanup after tests
	t.Cleanup(func() {
		metadata.Reset()
		outputFormat = "table"
		verbose = false
		noColor = false
	})
}

func TestFilterRoutes(t *testing.T) {
	routes := []metadata.RouteMetadata{
		{
			Method:     "GET",
			Path:       "/api/posts",
			Handler:    "Post.list",
			Resource:   "Post",
			Middleware: []string{"cache(300)"},
		},
		{
			Method:     "POST",
			Path:       "/api/posts",
			Handler:    "Post.create",
			Resource:   "Post",
			Middleware: []string{"auth", "rate_limit(5/hour)"},
		},
		{
			Method:     "GET",
			Path:       "/api/users",
			Handler:    "User.list",
			Resource:   "User",
			Middleware: []string{"auth"},
		},
		{
			Method:     "PUT",
			Path:       "/api/posts/:id",
			Handler:    "Post.update",
			Resource:   "Post",
			Middleware: []string{"auth", "owner"},
		},
	}

	t.Run("returns all routes when no filters", func(t *testing.T) {
		result := filterRoutes(routes, "", "", "")
		assert.Len(t, result, 4)
	})

	t.Run("filters by method", func(t *testing.T) {
		result := filterRoutes(routes, "GET", "", "")
		assert.Len(t, result, 2)
		assert.Equal(t, "GET", result[0].Method)
		assert.Equal(t, "GET", result[1].Method)
	})

	t.Run("filters by method case-insensitive", func(t *testing.T) {
		result := filterRoutes(routes, "post", "", "")
		assert.Len(t, result, 1)
		assert.Equal(t, "POST", result[0].Method)
	})

	t.Run("filters by middleware", func(t *testing.T) {
		result := filterRoutes(routes, "", "auth", "")
		assert.Len(t, result, 3)
		for _, route := range result {
			found := false
			for _, mw := range route.Middleware {
				if strings.Contains(strings.ToLower(mw), "auth") {
					found = true
					break
				}
			}
			assert.True(t, found, "each route should have auth middleware")
		}
	})

	t.Run("filters by middleware substring", func(t *testing.T) {
		result := filterRoutes(routes, "", "cache", "")
		assert.Len(t, result, 1)
		assert.Equal(t, "Post.list", result[0].Handler)
	})

	t.Run("filters by resource", func(t *testing.T) {
		result := filterRoutes(routes, "", "", "Post")
		assert.Len(t, result, 3)
		for _, route := range result {
			assert.Equal(t, "Post", route.Resource)
		}
	})

	t.Run("filters with multiple criteria", func(t *testing.T) {
		result := filterRoutes(routes, "GET", "cache", "Post")
		assert.Len(t, result, 1)
		assert.Equal(t, "GET", result[0].Method)
		assert.Equal(t, "Post.list", result[0].Handler)
		assert.Equal(t, "Post", result[0].Resource)
	})

	t.Run("returns empty when no matches", func(t *testing.T) {
		result := filterRoutes(routes, "DELETE", "", "")
		assert.Len(t, result, 0)
	})
}

func TestFormatRoutesAsTable(t *testing.T) {
	routes := []metadata.RouteMetadata{
		{
			Method:     "GET",
			Path:       "/api/posts",
			Handler:    "Post.list",
			Middleware: []string{"cache(300)"},
		},
		{
			Method:     "POST",
			Path:       "/api/posts",
			Handler:    "Post.create",
			Middleware: []string{"auth", "rate_limit(5/hour)"},
		},
	}

	t.Run("formats routes correctly", func(t *testing.T) {
		buf := &bytes.Buffer{}
		noColor = true

		err := formatRoutesAsTable(routes, "", buf)
		require.NoError(t, err)

		output := buf.String()

		// Check method, path, handler are present
		assert.Contains(t, output, "GET")
		assert.Contains(t, output, "POST")
		assert.Contains(t, output, "/api/posts")
		assert.Contains(t, output, "Post.list")
		assert.Contains(t, output, "Post.create")

		// Check middleware formatting
		assert.Contains(t, output, "[cache(300)]")
		assert.Contains(t, output, "[auth, rate_limit(5/hour)]")
	})

	t.Run("handles empty routes", func(t *testing.T) {
		buf := &bytes.Buffer{}
		noColor = true

		err := formatRoutesAsTable([]metadata.RouteMetadata{}, "", buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "No routes found")
	})

	t.Run("handles routes without middleware", func(t *testing.T) {
		buf := &bytes.Buffer{}
		noColor = true

		routesNoMw := []metadata.RouteMetadata{
			{
				Method:  "GET",
				Path:    "/health",
				Handler: "Health.check",
			},
		}

		err := formatRoutesAsTable(routesNoMw, "", buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "GET")
		assert.Contains(t, output, "/health")
		assert.Contains(t, output, "Health.check")
		assert.NotContains(t, output, "[]")
	})

	t.Cleanup(func() {
		noColor = false
	})
}

func TestFormatRoutesAsJSON(t *testing.T) {
	routes := []metadata.RouteMetadata{
		{
			Method:     "GET",
			Path:       "/api/posts",
			Handler:    "Post.list",
			Resource:   "Post",
			Operation:  "list",
			Middleware: []string{"cache(300)"},
		},
		{
			Method:     "POST",
			Path:       "/api/posts",
			Handler:    "Post.create",
			Resource:   "Post",
			Operation:  "create",
			Middleware: []string{"auth"},
		},
	}

	t.Run("formats JSON correctly", func(t *testing.T) {
		buf := &bytes.Buffer{}

		err := formatRoutesAsJSON(routes, "", buf)
		require.NoError(t, err)

		// Parse JSON
		var result struct {
			TotalCount int                      `json:"total_count"`
			Routes     []metadata.RouteMetadata `json:"routes"`
		}

		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, 2, result.TotalCount)
		assert.Len(t, result.Routes, 2)

		assert.Equal(t, "GET", result.Routes[0].Method)
		assert.Equal(t, "/api/posts", result.Routes[0].Path)
		assert.Equal(t, "Post.list", result.Routes[0].Handler)
		assert.Equal(t, "Post", result.Routes[0].Resource)
		assert.Equal(t, []string{"cache(300)"}, result.Routes[0].Middleware)
	})

	t.Run("handles empty routes", func(t *testing.T) {
		buf := &bytes.Buffer{}

		err := formatRoutesAsJSON([]metadata.RouteMetadata{}, "", buf)
		require.NoError(t, err)

		var result struct {
			TotalCount int                      `json:"total_count"`
			Routes     []metadata.RouteMetadata `json:"routes"`
		}

		err = json.Unmarshal(buf.Bytes(), &result)
		require.NoError(t, err)

		assert.Equal(t, 0, result.TotalCount)
		assert.Len(t, result.Routes, 0)
	})
}

// BenchmarkIntrospectRoutesCommand benchmarks the routes command performance
func BenchmarkIntrospectRoutesCommand(b *testing.B) {
	// Setup test registry with realistic data
	testMeta := &metadata.Metadata{
		Version:   "1.0.0",
		Generated: time.Now(),
		Routes:    make([]metadata.RouteMetadata, 0, 100),
	}

	// Create 100 routes to simulate a realistic application
	resources := []string{"User", "Post", "Comment", "Category", "Tag"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	operations := []string{"list", "get", "create", "update", "delete"}

	for i := 0; i < 100; i++ {
		resource := resources[i%len(resources)]
		method := methods[i%len(methods)]
		operation := operations[i%len(operations)]

		route := metadata.RouteMetadata{
			Method:     method,
			Path:       fmt.Sprintf("/api/%s/%d", strings.ToLower(resource), i),
			Handler:    fmt.Sprintf("%s.%s", resource, operation),
			Resource:   resource,
			Operation:  operation,
			Middleware: []string{"auth", "rate_limit"},
		}
		testMeta.Routes = append(testMeta.Routes, route)
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
	noColor = true

	cmd := newIntrospectRoutesCommand()
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
