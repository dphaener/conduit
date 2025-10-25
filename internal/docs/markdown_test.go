package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownGenerator_Generate(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	config := &Config{
		ProjectName:        "TestAPI",
		ProjectVersion:     "1.0.0",
		ProjectDescription: "Test API Documentation",
		OutputDir:          tmpDir,
	}

	generator := NewMarkdownGenerator(config)

	doc := &Documentation{
		ProjectInfo: &ProjectInfo{
			Name:        "TestAPI",
			Version:     "1.0.0",
			Description: "Test API Documentation",
		},
		Resources: []*ResourceDoc{
			{
				Name:          "User",
				Documentation: "User resource for authentication",
				Fields: []*FieldDoc{
					{
						Name:     "id",
						Type:     "uuid!",
						Required: true,
						Example:  "550e8400-e29b-41d4-a716-446655440000",
					},
					{
						Name:        "email",
						Type:        "email!",
						Required:    true,
						Constraints: []string{"@unique"},
						Example:     "user@example.com",
					},
				},
				Endpoints: []*EndpointDoc{
					{
						Method:      "GET",
						Path:        "/users",
						Summary:     "List users",
						Description: "Get all users",
					},
				},
			},
		},
	}

	err := generator.Generate(doc)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify README was created
	readmePath := filepath.Join(tmpDir, "markdown", "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Fatal("README.md was not created")
	}

	// Verify resource file was created
	userPath := filepath.Join(tmpDir, "markdown", "user.md")
	if _, err := os.Stat(userPath); os.IsNotExist(err) {
		t.Fatal("user.md was not created")
	}

	// Read and verify README content
	readmeContent, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	readme := string(readmeContent)
	if !strings.Contains(readme, "TestAPI API Documentation") {
		t.Error("README should contain project title")
	}

	if !strings.Contains(readme, "v1.0.0") {
		t.Error("README should contain version")
	}

	if !strings.Contains(readme, "User") {
		t.Error("README should list User resource")
	}

	// Read and verify resource content
	userContent, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("Failed to read user.md: %v", err)
	}

	user := string(userContent)
	if !strings.Contains(user, "# User") {
		t.Error("Resource doc should have title")
	}

	if !strings.Contains(user, "User resource for authentication") {
		t.Error("Resource doc should contain documentation")
	}

	if !strings.Contains(user, "## Fields") {
		t.Error("Resource doc should have Fields section")
	}

	if !strings.Contains(user, "## Endpoints") {
		t.Error("Resource doc should have Endpoints section")
	}

	if !strings.Contains(user, "GET /users") {
		t.Error("Resource doc should list endpoints")
	}
}

func TestMarkdownGenerator_GenerateIndex(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		ProjectName:    "TestAPI",
		ProjectVersion: "2.0.0",
		OutputDir:      tmpDir,
		BaseURL:        "https://api.example.com",
	}

	generator := NewMarkdownGenerator(config)

	doc := &Documentation{
		ProjectInfo: &ProjectInfo{
			Name:        "TestAPI",
			Version:     "2.0.0",
			Description: "Test description",
		},
		Resources: []*ResourceDoc{
			{Name: "User"},
			{Name: "Post"},
		},
	}

	outputDir := filepath.Join(tmpDir, "markdown")
	os.MkdirAll(outputDir, 0755)

	err := generator.generateIndex(doc, outputDir)
	if err != nil {
		t.Fatalf("generateIndex failed: %v", err)
	}

	// Read the file
	content, err := os.ReadFile(filepath.Join(outputDir, "README.md"))
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	readme := string(content)

	// Verify content
	if !strings.Contains(readme, "TestAPI API Documentation") {
		t.Error("Should contain project name")
	}

	if !strings.Contains(readme, "2.0.0") {
		t.Error("Should contain version")
	}

	if !strings.Contains(readme, "[User](user.md)") {
		t.Error("Should link to User resource")
	}

	if !strings.Contains(readme, "[Post](post.md)") {
		t.Error("Should link to Post resource")
	}

	if !strings.Contains(readme, "https://api.example.com") {
		t.Error("Should contain base URL")
	}
}

func TestMarkdownGenerator_GenerateResourceDoc(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "markdown")
	os.MkdirAll(outputDir, 0755)

	config := &Config{
		OutputDir: tmpDir,
	}

	generator := NewMarkdownGenerator(config)

	resource := &ResourceDoc{
		Name:          "Post",
		Documentation: "Blog post resource",
		Fields: []*FieldDoc{
			{
				Name:        "title",
				Type:        "string!",
				Required:    true,
				Constraints: []string{"@min(5)", "@max(200)"},
			},
		},
		Relationships: []*RelationshipDoc{
			{
				Name:       "author",
				Type:       "User",
				Kind:       "belongs_to",
				ForeignKey: "author_id",
			},
		},
		Endpoints: []*EndpointDoc{
			{
				Method:  "GET",
				Path:    "/posts",
				Summary: "List posts",
			},
		},
		Hooks: []*HookDoc{
			{
				Timing: "before",
				Event:  "create",
			},
		},
	}

	err := generator.generateResourceDoc(resource, outputDir)
	if err != nil {
		t.Fatalf("generateResourceDoc failed: %v", err)
	}

	// Read the file
	content, err := os.ReadFile(filepath.Join(outputDir, "post.md"))
	if err != nil {
		t.Fatalf("Failed to read post.md: %v", err)
	}

	doc := string(content)

	// Verify sections
	if !strings.Contains(doc, "# Post") {
		t.Error("Should have title")
	}

	if !strings.Contains(doc, "Blog post resource") {
		t.Error("Should have documentation")
	}

	if !strings.Contains(doc, "## Fields") {
		t.Error("Should have Fields section")
	}

	if !strings.Contains(doc, "## Relationships") {
		t.Error("Should have Relationships section")
	}

	if !strings.Contains(doc, "## Endpoints") {
		t.Error("Should have Endpoints section")
	}

	if !strings.Contains(doc, "## Hooks") {
		t.Error("Should have Hooks section")
	}

	// Verify field details
	if !strings.Contains(doc, "`title`") {
		t.Error("Should contain field name")
	}

	if !strings.Contains(doc, "@min(5)") {
		t.Error("Should contain constraints")
	}

	// Verify relationship details
	if !strings.Contains(doc, "author") {
		t.Error("Should contain relationship name")
	}

	if !strings.Contains(doc, "belongs_to") {
		t.Error("Should contain relationship kind")
	}
}
