package templates

import (
	"strings"
	"testing"
	"time"
)

func TestGetCLAUDEMDContent(t *testing.T) {
	content := GetCLAUDEMDContent()

	// Basic sanity checks
	if content == "" {
		t.Fatal("CLAUDE.md content should not be empty")
	}

	// Check that it's valid markdown (starts with # heading)
	if !strings.HasPrefix(content, "# CLAUDE.md") {
		t.Error("CLAUDE.md should start with '# CLAUDE.md' heading")
	}

	// Verify key sections exist
	requiredSections := []string{
		"## Quick Context",
		"## Bootstrap (First Resource)",
		"## Discovery Mechanisms",
		"## How to Learn Conduit",
		"## Project Structure",
		"## Development Workflow",
		"## Critical Safety Rules",
	}

	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			t.Errorf("CLAUDE.md missing required section: %s", section)
		}
	}
}

func TestCLAUDEMDIntrospectionCommands(t *testing.T) {
	content := GetCLAUDEMDContent()

	// Verify introspection commands are prominently featured
	introspectionCommands := []string{
		"conduit introspect schema",
		"conduit introspect relationships",
		"conduit introspect hooks",
		"conduit introspect validators",
		"conduit introspect patterns",
	}

	for _, cmd := range introspectionCommands {
		if !strings.Contains(content, cmd) {
			t.Errorf("CLAUDE.md missing introspection command: %s", cmd)
		}
	}

	// Check that introspection appears in first 50 lines (UX-3)
	lines := strings.Split(content, "\n")
	first50Lines := strings.Join(lines[:min(50, len(lines))], "\n")

	if !strings.Contains(first50Lines, "introspect") {
		t.Error("Introspection should appear prominently in first 50 lines (UX-3)")
	}
}

func TestCLAUDEMDDiscoveryFirstApproach(t *testing.T) {
	content := GetCLAUDEMDContent()

	// Verify discovery-first language
	discoveryKeywords := []string{
		"DISCOVER",
		"discovery",
		"introspection",
		"metadata",
		"build/app.meta.json",
	}

	for _, keyword := range discoveryKeywords {
		if !strings.Contains(content, keyword) {
			t.Errorf("CLAUDE.md should emphasize discovery-first approach, missing: %s", keyword)
		}
	}

	// Verify DISCOVER → LEARN → APPLY → VERIFY pattern
	if !strings.Contains(content, "DISCOVER") ||
	   !strings.Contains(content, "LEARN") ||
	   !strings.Contains(content, "APPLY") ||
	   !strings.Contains(content, "VERIFY") {
		t.Error("CLAUDE.md should include DISCOVER → LEARN → APPLY → VERIFY pattern (UX-4)")
	}
}

func TestCLAUDEMDSafetyRules(t *testing.T) {
	content := GetCLAUDEMDContent()

	// Verify critical safety warnings exist
	safetyTopics := []string{
		"gitignored",
		"NEVER edit files in `build/`",
		"NEVER modify migrations after running",
		"immutable",
	}

	for _, topic := range safetyTopics {
		if !strings.Contains(content, topic) {
			t.Errorf("CLAUDE.md missing critical safety topic: %s", topic)
		}
	}
}

func TestCLAUDEMDTemplateVariables(t *testing.T) {
	content := GetCLAUDEMDContent()

	// Verify template variables are present
	templateVars := []string{
		"{{.ProjectName}}",
		"{{.Variables.port}}",
		"{{.Variables.project_name}}",
	}

	for _, tmplVar := range templateVars {
		if !strings.Contains(content, tmplVar) {
			t.Errorf("CLAUDE.md missing template variable: %s", tmplVar)
		}
	}
}

func TestCLAUDEMDFileSize(t *testing.T) {
	content := GetCLAUDEMDContent()

	// UX-1: File should be <12KB for fast LLM parsing
	sizeInBytes := len(content)
	maxSizeBytes := 12 * 1024 // 12KB

	if sizeInBytes > maxSizeBytes {
		t.Errorf("CLAUDE.md is %d bytes, exceeds 12KB limit (%d bytes) (UX-1)",
			sizeInBytes, maxSizeBytes)
	}

	t.Logf("CLAUDE.md size: %d bytes (%.2f KB)", sizeInBytes, float64(sizeInBytes)/1024)
}

func TestCLAUDEMDBootstrapGuide(t *testing.T) {
	content := GetCLAUDEMDContent()

	// FC-4: Should include step-by-step "starting from scratch" guide
	bootstrapSteps := []string{
		"Step 1",
		"Step 2",
		"Step 3",
		"Step 4",
		"Step 5",
		"Step 6",
	}

	for _, step := range bootstrapSteps {
		if !strings.Contains(content, step) {
			t.Errorf("Bootstrap guide missing: %s", step)
		}
	}
}

func TestCLAUDEMDMetadataFileReferences(t *testing.T) {
	content := GetCLAUDEMDContent()

	// FC-7: Should include links to metadata files and introspection commands
	metadataRefs := []string{
		"build/app.meta.json",
		"build/generated/",
		"migrations/",
	}

	for _, ref := range metadataRefs {
		if !strings.Contains(content, ref) {
			t.Errorf("CLAUDE.md should reference metadata file/directory: %s", ref)
		}
	}
}

func TestCLAUDEMDCodeBlocks(t *testing.T) {
	content := GetCLAUDEMDContent()

	// Verify code blocks use proper language tags
	if !strings.Contains(content, "```bash") {
		t.Error("CLAUDE.md should have bash code blocks")
	}

	if !strings.Contains(content, "```conduit") {
		t.Error("CLAUDE.md should have conduit code blocks showing language examples")
	}

	// Count code blocks to ensure there are examples
	bashBlockCount := strings.Count(content, "```bash")
	if bashBlockCount < 10 {
		t.Errorf("CLAUDE.md should have at least 10 bash code examples, found: %d", bashBlockCount)
	}
}

func TestCLAUDEMDRendering(t *testing.T) {
	// Test that CLAUDE.md renders correctly with template variables
	engine := NewEngine()
	content := GetCLAUDEMDContent()

	ctx := &TemplateContext{
		ProjectName: "test-app",
		Variables: map[string]interface{}{
			"project_name": "api",
			"port":         3000,
			"include_auth": true,
		},
		Timestamp: time.Now(),
	}

	rendered, err := engine.renderString(content, ctx)
	if err != nil {
		t.Fatalf("Failed to render CLAUDE.md template: %v", err)
	}

	// Verify variables were substituted
	if !strings.Contains(rendered, "test-app") {
		t.Error("ProjectName should be substituted in rendered output")
	}

	if !strings.Contains(rendered, "3000") {
		t.Error("Port should be substituted in rendered output")
	}

	if strings.Contains(rendered, "{{.ProjectName}}") {
		t.Error("Template variables should be fully substituted, found: {{.ProjectName}}")
	}

	if strings.Contains(rendered, "{{.Variables.port}}") {
		t.Error("Template variables should be fully substituted, found: {{.Variables.port}}")
	}
}

func TestCLAUDEMDIntegrationInAPITemplate(t *testing.T) {
	tmpl := NewAPITemplate()

	// Verify CLAUDE.md is included in files
	foundCLAUDE := false
	for _, file := range tmpl.Files {
		if file.TargetPath == "CLAUDE.md" {
			foundCLAUDE = true

			// Verify it's configured correctly
			if !file.Template {
				t.Error("CLAUDE.md should be marked as a template file")
			}

			if file.Content == "" {
				t.Error("CLAUDE.md should have content")
			}

			break
		}
	}

	if !foundCLAUDE {
		t.Error("API template should include CLAUDE.md file")
	}
}

func TestCLAUDEMDIntegrationInWebTemplate(t *testing.T) {
	tmpl := NewWebTemplate()

	// Verify CLAUDE.md is included in files
	foundCLAUDE := false
	for _, file := range tmpl.Files {
		if file.TargetPath == "CLAUDE.md" {
			foundCLAUDE = true

			if !file.Template {
				t.Error("CLAUDE.md should be marked as a template file")
			}

			if file.Content == "" {
				t.Error("CLAUDE.md should have content")
			}

			break
		}
	}

	if !foundCLAUDE {
		t.Error("Web template should include CLAUDE.md file")
	}
}

func TestCLAUDEMDIntegrationInMicroserviceTemplate(t *testing.T) {
	tmpl := NewMicroserviceTemplate()

	// Verify CLAUDE.md is included in files
	foundCLAUDE := false
	for _, file := range tmpl.Files {
		if file.TargetPath == "CLAUDE.md" {
			foundCLAUDE = true

			if !file.Template {
				t.Error("CLAUDE.md should be marked as a template file")
			}

			if file.Content == "" {
				t.Error("CLAUDE.md should have content")
			}

			break
		}
	}

	if !foundCLAUDE {
		t.Error("Microservice template should include CLAUDE.md file")
	}
}

func TestCLAUDEMDCommonTasks(t *testing.T) {
	content := GetCLAUDEMDContent()

	// Verify common tasks are documented
	commonTasks := []string{
		"Adding a New Field",
		"Adding Validation",
		"Adding a Relationship",
		"Adding a Hook",
	}

	for _, task := range commonTasks {
		if !strings.Contains(content, task) {
			t.Errorf("CLAUDE.md should document common task: %s", task)
		}
	}
}

func TestCLAUDEMDExternalLinks(t *testing.T) {
	content := GetCLAUDEMDContent()

	// Verify external documentation links are present
	if !strings.Contains(content, "conduit-lang.org") {
		t.Error("CLAUDE.md should link to external documentation")
	}

	// Should mention other spec files for deeper learning
	specFiles := []string{
		"LANGUAGE-SPEC.md",
		"ARCHITECTURE.md",
		"IMPLEMENTATION-",
	}

	foundAny := false
	for _, spec := range specFiles {
		if strings.Contains(content, spec) {
			foundAny = true
			break
		}
	}

	if !foundAny {
		t.Error("CLAUDE.md should reference spec files for deeper learning")
	}
}

// Helper function for min (Go 1.21+ has built-in min)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
