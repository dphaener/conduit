package router

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRouterCode(t *testing.T) {
	schemas := []ResourceSchema{
		{
			Name:           "Post",
			PluralName:     "Posts",
			TableName:      "posts",
			PrimaryKey:     "id",
			PrimaryKeyType: "uuid",
			Operations:     []CRUDOperation{OpList, OpCreate, OpShow},
		},
		{
			Name:           "Comment",
			PluralName:     "Comments",
			TableName:      "comments",
			PrimaryKey:     "id",
			PrimaryKeyType: "int",
			Operations:     []CRUDOperation{OpList, OpShow, OpDelete},
		},
	}

	code := GenerateRouterCode(schemas)

	// Verify code structure
	assert.Contains(t, code, "package generated")
	assert.Contains(t, code, "func RegisterRoutes")
	assert.Contains(t, code, "Post")
	assert.Contains(t, code, "Comment")
	assert.Contains(t, code, "router.NewResourceDefinition")
	assert.Contains(t, code, "router.OpList")
	assert.Contains(t, code, "router.OpCreate")
	assert.Contains(t, code, "router.OpShow")
	assert.Contains(t, code, "router.OpDelete")
}

func TestGenerateHandlerStubs(t *testing.T) {
	schemas := []ResourceSchema{
		{
			Name:           "Post",
			PluralName:     "Posts",
			PrimaryKeyType: "uuid",
			Operations:     []CRUDOperation{OpList, OpCreate, OpShow, OpUpdate, OpPatch, OpDelete},
		},
	}

	code := GenerateHandlerStubs(schemas)

	// Verify code structure
	assert.Contains(t, code, "package generated")
	assert.Contains(t, code, "HandlePostList")
	assert.Contains(t, code, "HandlePostCreate")
	assert.Contains(t, code, "HandlePostShow")
	assert.Contains(t, code, "HandlePostUpdate")
	assert.Contains(t, code, "HandlePostPatch")
	assert.Contains(t, code, "HandlePostDelete")

	// Verify handler signatures
	assert.Contains(t, code, "func HandlePostList(w http.ResponseWriter, r *http.Request)")
	assert.Contains(t, code, "router.NewParamExtractor(r)")
}

func TestGenerateHandlerStubsListOperation(t *testing.T) {
	schemas := []ResourceSchema{
		{
			Name:           "Post",
			PrimaryKeyType: "uuid",
			Operations:     []CRUDOperation{OpList},
		},
	}

	code := GenerateHandlerStubs(schemas)

	// List operation should have pagination
	assert.Contains(t, code, "ExtractPagination")
	assert.Contains(t, code, "pagination.Page")
	assert.Contains(t, code, "pagination.PerPage")
	assert.Contains(t, code, `"total": 0`)
}

func TestGenerateHandlerStubsCreateOperation(t *testing.T) {
	schemas := []ResourceSchema{
		{
			Name:           "Post",
			PrimaryKeyType: "uuid",
			Operations:     []CRUDOperation{OpCreate},
		},
	}

	code := GenerateHandlerStubs(schemas)

	// Create operation should have TODO comments
	assert.Contains(t, code, "TODO: Parse request body")
	assert.Contains(t, code, "TODO: Validate input")
	assert.Contains(t, code, "TODO: Create resource")
	assert.Contains(t, code, "http.StatusCreated")
}

func TestGenerateHandlerStubsShowOperationUUID(t *testing.T) {
	schemas := []ResourceSchema{
		{
			Name:           "Post",
			PrimaryKeyType: "uuid",
			Operations:     []CRUDOperation{OpShow},
		},
	}

	code := GenerateHandlerStubs(schemas)

	// Show with UUID should extract UUID parameter
	assert.Contains(t, code, "PathParamUUID")
	assert.Contains(t, code, "http.StatusBadRequest")
}

func TestGenerateHandlerStubsShowOperationString(t *testing.T) {
	schemas := []ResourceSchema{
		{
			Name:           "Post",
			PrimaryKeyType: "string",
			Operations:     []CRUDOperation{OpShow},
		},
	}

	code := GenerateHandlerStubs(schemas)

	// Show with string should use PathParam
	assert.Contains(t, code, "PathParam")
	assert.NotContains(t, code, "PathParamUUID")
}

func TestGenerateHandlerStubsUpdateOperation(t *testing.T) {
	schemas := []ResourceSchema{
		{
			Name:           "Post",
			PrimaryKeyType: "uuid",
			Operations:     []CRUDOperation{OpUpdate},
		},
	}

	code := GenerateHandlerStubs(schemas)

	// Update should parse body and extract ID
	assert.Contains(t, code, "TODO: Parse request body")
	assert.Contains(t, code, "TODO: Validate input")
	assert.Contains(t, code, "TODO: Update resource")
	assert.Contains(t, code, "PathParamUUID")
}

func TestGenerateHandlerStubsPatchOperation(t *testing.T) {
	schemas := []ResourceSchema{
		{
			Name:           "Post",
			PrimaryKeyType: "uuid",
			Operations:     []CRUDOperation{OpPatch},
		},
	}

	code := GenerateHandlerStubs(schemas)

	// Patch should be similar to update
	assert.Contains(t, code, "TODO: Parse request body")
	assert.Contains(t, code, "TODO: Update resource")
	assert.Contains(t, code, "PathParamUUID")
}

func TestGenerateHandlerStubsDeleteOperation(t *testing.T) {
	schemas := []ResourceSchema{
		{
			Name:           "Post",
			PrimaryKeyType: "uuid",
			Operations:     []CRUDOperation{OpDelete},
		},
	}

	code := GenerateHandlerStubs(schemas)

	// Delete should extract ID and return no content
	assert.Contains(t, code, "TODO: Delete resource")
	assert.Contains(t, code, "PathParamUUID")
	assert.Contains(t, code, "http.StatusNoContent")
}

func TestLowerFirst(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Post", "post"},
		{"Comment", "comment"},
		{"HTTPRequest", "hTTPRequest"},
		{"a", "a"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := lowerFirst(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCapitalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"post", "Post"},
		{"comment", "Comment"},
		{"list", "List"},
		{"a", "A"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := capitalize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateResourceRegistration(t *testing.T) {
	schema := ResourceSchema{
		Name:           "Post",
		PluralName:     "Posts",
		PrimaryKeyType: "uuid",
		Operations:     []CRUDOperation{OpList, OpCreate},
	}

	code := generateResourceRegistration(schema)

	// Verify structure
	assert.Contains(t, code, "postDef := router.NewResourceDefinition")
	assert.Contains(t, code, "postDef.PluralName = \"Posts\"")
	assert.Contains(t, code, "postDef.IDType = \"uuid\"")
	assert.Contains(t, code, "router.OpList")
	assert.Contains(t, code, "router.OpCreate")
	assert.Contains(t, code, "HandlePostList")
	assert.Contains(t, code, "HandlePostCreate")
}

func TestGenerateHandlerStub(t *testing.T) {
	schema := ResourceSchema{
		Name:           "Post",
		PrimaryKeyType: "uuid",
	}

	// Test each operation
	tests := []struct {
		operation CRUDOperation
		contains  []string
	}{
		{
			operation: OpList,
			contains:  []string{"HandlePostList", "ExtractPagination", "pagination.Page"},
		},
		{
			operation: OpCreate,
			contains:  []string{"HandlePostCreate", "TODO: Create resource", "StatusCreated"},
		},
		{
			operation: OpShow,
			contains:  []string{"HandlePostShow", "PathParamUUID", "TODO: Fetch resource"},
		},
		{
			operation: OpUpdate,
			contains:  []string{"HandlePostUpdate", "PathParamUUID", "TODO: Update resource"},
		},
		{
			operation: OpPatch,
			contains:  []string{"HandlePostPatch", "PathParamUUID", "TODO: Update resource"},
		},
		{
			operation: OpDelete,
			contains:  []string{"HandlePostDelete", "PathParamUUID", "TODO: Delete resource", "StatusNoContent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.operation.String(), func(t *testing.T) {
			code := generateHandlerStub(schema, tt.operation)
			for _, expected := range tt.contains {
				assert.Contains(t, code, expected)
			}
		})
	}
}

func TestGenerateMultipleResources(t *testing.T) {
	schemas := []ResourceSchema{
		{
			Name:           "Post",
			PluralName:     "Posts",
			PrimaryKeyType: "uuid",
			Operations:     []CRUDOperation{OpList, OpCreate},
		},
		{
			Name:           "User",
			PluralName:     "Users",
			PrimaryKeyType: "int",
			Operations:     []CRUDOperation{OpShow, OpUpdate},
		},
	}

	code := GenerateRouterCode(schemas)

	// Both resources should be registered
	assert.Contains(t, code, "postDef")
	assert.Contains(t, code, "userDef")

	// Handler stubs should be generated for both
	handlers := GenerateHandlerStubs(schemas)
	assert.Contains(t, handlers, "HandlePostList")
	assert.Contains(t, handlers, "HandlePostCreate")
	assert.Contains(t, handlers, "HandleUserShow")
	assert.Contains(t, handlers, "HandleUserUpdate")
}

func TestCodeGenerationWithFields(t *testing.T) {
	schema := ResourceSchema{
		Name:           "Post",
		PluralName:     "Posts",
		PrimaryKeyType: "uuid",
		Operations:     []CRUDOperation{OpList},
		Fields: []FieldSchema{
			{Name: "title", Type: "string", Required: true},
			{Name: "content", Type: "text", Required: true},
		},
	}

	code := GenerateHandlerStubs([]ResourceSchema{schema})

	// Code should still be generated even with fields
	assert.Contains(t, code, "HandlePostList")
	assert.Contains(t, code, "package generated")
}

func TestGeneratedCodeIsValidGo(t *testing.T) {
	schemas := []ResourceSchema{
		{
			Name:           "Post",
			PluralName:     "Posts",
			PrimaryKeyType: "uuid",
			Operations:     []CRUDOperation{OpList, OpCreate},
		},
	}

	routerCode := GenerateRouterCode(schemas)
	handlerCode := GenerateHandlerStubs(schemas)

	// Should have valid package declarations
	assert.True(t, strings.HasPrefix(routerCode, "// Code generated"))
	assert.True(t, strings.HasPrefix(handlerCode, "// Code generated"))

	// Should have proper imports
	assert.Contains(t, routerCode, "import (")
	assert.Contains(t, handlerCode, "import (")

	// Should have balanced braces
	assert.Equal(t, strings.Count(routerCode, "{"), strings.Count(routerCode, "}"))
	assert.Equal(t, strings.Count(handlerCode, "{"), strings.Count(handlerCode, "}"))
}
