package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTMLGenerator_Generate(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	config := &Config{
		ProjectName:        "TestAPI",
		ProjectVersion:     "1.0.0",
		ProjectDescription: "Test API",
		OutputDir:          tmpDir,
		BaseURL:            "https://api.example.com",
	}

	generator := NewHTMLGenerator(config)

	doc := &Documentation{
		ProjectInfo: &ProjectInfo{
			Name:        "TestAPI",
			Version:     "1.0.0",
			Description: "Test API",
		},
		Resources: []*ResourceDoc{
			{
				Name:          "User",
				Documentation: "User resource",
				Fields: []*FieldDoc{
					{
						Name:     "id",
						Type:     "uuid!",
						Required: true,
					},
				},
				Endpoints: []*EndpointDoc{
					{
						Method:  "GET",
						Path:    "/users",
						Summary: "List users",
					},
				},
			},
		},
	}

	err := generator.Generate(doc)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	htmlDir := filepath.Join(tmpDir, "html")

	// Verify index.html was created
	indexPath := filepath.Join(htmlDir, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Fatal("index.html was not created")
	}

	// Verify resource HTML was created
	userPath := filepath.Join(htmlDir, "user.html")
	if _, err := os.Stat(userPath); os.IsNotExist(err) {
		t.Fatal("user.html was not created")
	}

	// Verify CSS was created
	cssPath := filepath.Join(htmlDir, "styles.css")
	if _, err := os.Stat(cssPath); os.IsNotExist(err) {
		t.Fatal("styles.css was not created")
	}

	// Verify JS was created
	jsPath := filepath.Join(htmlDir, "script.js")
	if _, err := os.Stat(jsPath); os.IsNotExist(err) {
		t.Fatal("script.js was not created")
	}

	// Read and verify index content
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Failed to read index.html: %v", err)
	}

	index := string(indexContent)
	if !strings.Contains(index, "TestAPI") {
		t.Error("index.html should contain project name")
	}

	if !strings.Contains(index, "User") {
		t.Error("index.html should list User resource")
	}

	// Read and verify resource content
	userContent, err := os.ReadFile(userPath)
	if err != nil {
		t.Fatalf("Failed to read user.html: %v", err)
	}

	user := string(userContent)
	if !strings.Contains(user, "<h1>User</h1>") {
		t.Error("user.html should have resource title")
	}

	if !strings.Contains(user, "GET") {
		t.Error("user.html should list endpoints")
	}

	// Verify CSS content
	cssContent, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("Failed to read styles.css: %v", err)
	}

	css := string(cssContent)
	if !strings.Contains(css, ".sidebar") {
		t.Error("CSS should contain sidebar styles")
	}

	// Verify JS content
	jsContent, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatalf("Failed to read script.js: %v", err)
	}

	js := string(jsContent)
	if !strings.Contains(js, "search") {
		t.Error("JS should contain search functionality")
	}
}

func TestHTMLGenerator_LoadTemplates(t *testing.T) {
	config := &Config{}
	generator := NewHTMLGenerator(config)

	err := generator.loadTemplates()
	if err != nil {
		t.Fatalf("loadTemplates failed: %v", err)
	}

	if generator.templates == nil {
		t.Error("Templates should be loaded")
	}

	// Verify templates exist
	if generator.templates.Lookup("index") == nil {
		t.Error("index template should be loaded")
	}

	if generator.templates.Lookup("resource") == nil {
		t.Error("resource template should be loaded")
	}
}

func TestHTMLGenerator_HelperFunctions(t *testing.T) {
	generator := &HTMLGenerator{}

	// Test toJSON
	data := map[string]interface{}{"key": "value"}
	json := generator.toJSON(data)
	if !strings.Contains(json, "key") || !strings.Contains(json, "value") {
		t.Error("toJSON should serialize data")
	}

	// Test toJSONPretty
	pretty := generator.toJSONPretty(data)
	prettyStr := string(pretty)
	if !strings.Contains(prettyStr, "key") {
		t.Error("toJSONPretty should serialize data")
	}
}
