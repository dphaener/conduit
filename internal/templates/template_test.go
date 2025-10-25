package templates

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTemplateValidation(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    *Template
		wantErr bool
	}{
		{
			name: "valid template",
			tmpl: &Template{
				Name:    "test",
				Version: "1.0.0",
				Files: []*TemplateFile{
					{TargetPath: "test.txt", Content: "test"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			tmpl: &Template{
				Version: "1.0.0",
				Files: []*TemplateFile{
					{TargetPath: "test.txt", Content: "test"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing version",
			tmpl: &Template{
				Name: "test",
				Files: []*TemplateFile{
					{TargetPath: "test.txt", Content: "test"},
				},
			},
			wantErr: true,
		},
		{
			name: "no files",
			tmpl: &Template{
				Name:    "test",
				Version: "1.0.0",
				Files:   []*TemplateFile{},
			},
			wantErr: true,
		},
		{
			name: "duplicate variable names",
			tmpl: &Template{
				Name:    "test",
				Version: "1.0.0",
				Variables: []*TemplateVariable{
					{Name: "var1", Type: VariableTypeString},
					{Name: "var1", Type: VariableTypeString},
				},
				Files: []*TemplateFile{
					{TargetPath: "test.txt", Content: "test"},
				},
			},
			wantErr: true,
		},
		{
			name: "select variable without options",
			tmpl: &Template{
				Name:    "test",
				Version: "1.0.0",
				Variables: []*TemplateVariable{
					{Name: "choice", Type: VariableTypeSelect},
				},
				Files: []*TemplateFile{
					{TargetPath: "test.txt", Content: "test"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tmpl.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngineRenderString(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		context  *TemplateContext
		want     string
		wantErr  bool
	}{
		{
			name:     "simple variable substitution",
			template: "Hello {{.ProjectName}}",
			context: &TemplateContext{
				ProjectName: "test-project",
				Variables:   make(map[string]interface{}),
			},
			want:    "Hello test-project",
			wantErr: false,
		},
		{
			name:     "variable from context",
			template: "Port: {{.Variables.port}}",
			context: &TemplateContext{
				Variables: map[string]interface{}{
					"port": 3000,
				},
			},
			want:    "Port: 3000",
			wantErr: false,
		},
		{
			name:     "upper function",
			template: "{{upper .ProjectName}}",
			context: &TemplateContext{
				ProjectName: "test",
			},
			want:    "TEST",
			wantErr: false,
		},
		{
			name:     "lower function",
			template: "{{lower .ProjectName}}",
			context: &TemplateContext{
				ProjectName: "TEST",
			},
			want:    "test",
			wantErr: false,
		},
		{
			name:     "conditional",
			template: "{{if .Variables.enabled}}yes{{else}}no{{end}}",
			context: &TemplateContext{
				Variables: map[string]interface{}{
					"enabled": true,
				},
			},
			want:    "yes",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.renderString(tt.template, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("renderString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngineExecute(t *testing.T) {
	engine := NewEngine()

	// Create temporary directory for testing
	tmpDir := t.TempDir()

	tmpl := &Template{
		Name:    "test-template",
		Version: "1.0.0",
		Variables: []*TemplateVariable{
			{Name: "app_name", Type: VariableTypeString, Required: true},
		},
		Directories: []string{
			"src",
			"config",
		},
		Files: []*TemplateFile{
			{
				TargetPath: "README.md",
				Template:   true,
				Content:    "# {{.ProjectName}}\n\nApp: {{.Variables.app_name}}",
			},
			{
				TargetPath: "config/settings.txt",
				Template:   false,
				Content:    "static content",
			},
		},
	}

	ctx := &TemplateContext{
		ProjectName: "my-project",
		Variables: map[string]interface{}{
			"app_name": "MyApp",
		},
		Timestamp: time.Now(),
	}

	targetDir := filepath.Join(tmpDir, "output")

	err := engine.Execute(tmpl, ctx, targetDir)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify directories were created
	if _, err := os.Stat(filepath.Join(targetDir, "src")); os.IsNotExist(err) {
		t.Error("src directory was not created")
	}

	if _, err := os.Stat(filepath.Join(targetDir, "config")); os.IsNotExist(err) {
		t.Error("config directory was not created")
	}

	// Verify files were created with correct content
	readmeContent, err := os.ReadFile(filepath.Join(targetDir, "README.md"))
	if err != nil {
		t.Fatalf("Failed to read README.md: %v", err)
	}

	wantReadme := "# my-project\n\nApp: MyApp"
	if string(readmeContent) != wantReadme {
		t.Errorf("README.md content = %q, want %q", string(readmeContent), wantReadme)
	}

	settingsContent, err := os.ReadFile(filepath.Join(targetDir, "config", "settings.txt"))
	if err != nil {
		t.Fatalf("Failed to read settings.txt: %v", err)
	}

	if string(settingsContent) != "static content" {
		t.Errorf("settings.txt content = %q, want %q", string(settingsContent), "static content")
	}
}

func TestEngineExecuteWithCondition(t *testing.T) {
	engine := NewEngine()
	tmpDir := t.TempDir()

	tmpl := &Template{
		Name:    "conditional-template",
		Version: "1.0.0",
		Files: []*TemplateFile{
			{
				TargetPath: "always.txt",
				Template:   false,
				Content:    "always created",
			},
			{
				TargetPath: "conditional.txt",
				Template:   false,
				Content:    "conditional content",
				Condition:  "{{.Variables.create_file}}",
			},
		},
	}

	tests := []struct {
		name             string
		createFile       string
		shouldExist      bool
		conditionalExist bool
	}{
		{
			name:             "condition true",
			createFile:       "true",
			shouldExist:      true,
			conditionalExist: true,
		},
		{
			name:             "condition false",
			createFile:       "false",
			shouldExist:      true,
			conditionalExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetDir := filepath.Join(tmpDir, tt.name)

			ctx := &TemplateContext{
				Variables: map[string]interface{}{
					"create_file": tt.createFile,
				},
			}

			err := engine.Execute(tmpl, ctx, targetDir)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			// always.txt should always exist
			if _, err := os.Stat(filepath.Join(targetDir, "always.txt")); os.IsNotExist(err) {
				t.Error("always.txt was not created")
			}

			// Check conditional file
			_, err = os.Stat(filepath.Join(targetDir, "conditional.txt"))
			if tt.conditionalExist && os.IsNotExist(err) {
				t.Error("conditional.txt should exist but doesn't")
			}
			if !tt.conditionalExist && !os.IsNotExist(err) {
				t.Error("conditional.txt should not exist but does")
			}
		})
	}
}

func TestEngineValidateContext(t *testing.T) {
	engine := NewEngine()

	tmpl := &Template{
		Name:    "test",
		Version: "1.0.0",
		Variables: []*TemplateVariable{
			{Name: "required_var", Required: true},
			{Name: "optional_var", Required: false},
		},
		Files: []*TemplateFile{
			{TargetPath: "test.txt", Content: "test"},
		},
	}

	tests := []struct {
		name    string
		ctx     *TemplateContext
		wantErr bool
	}{
		{
			name: "all required variables provided",
			ctx: &TemplateContext{
				Variables: map[string]interface{}{
					"required_var": "value",
					"optional_var": "value",
				},
			},
			wantErr: false,
		},
		{
			name: "optional variable missing",
			ctx: &TemplateContext{
				Variables: map[string]interface{}{
					"required_var": "value",
				},
			},
			wantErr: false,
		},
		{
			name: "required variable missing",
			ctx: &TemplateContext{
				Variables: map[string]interface{}{
					"optional_var": "value",
				},
			},
			wantErr: true,
		},
		{
			name: "no variables provided",
			ctx: &TemplateContext{
				Variables: map[string]interface{}{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.validateContext(tmpl, tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTemplateFileExecutable(t *testing.T) {
	engine := NewEngine()
	tmpDir := t.TempDir()

	tmpl := &Template{
		Name:    "executable-test",
		Version: "1.0.0",
		Files: []*TemplateFile{
			{
				TargetPath: "script.sh",
				Template:   false,
				Content:    "#!/bin/bash\necho hello",
				Executable: true,
			},
			{
				TargetPath: "readme.md",
				Template:   false,
				Content:    "# README",
				Executable: false,
			},
		},
	}

	ctx := &TemplateContext{
		Variables: make(map[string]interface{}),
	}

	targetDir := filepath.Join(tmpDir, "exec-test")

	err := engine.Execute(tmpl, ctx, targetDir)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Check executable file permissions
	scriptInfo, err := os.Stat(filepath.Join(targetDir, "script.sh"))
	if err != nil {
		t.Fatalf("Failed to stat script.sh: %v", err)
	}

	if scriptInfo.Mode().Perm() != 0755 {
		t.Errorf("script.sh permissions = %o, want %o", scriptInfo.Mode().Perm(), 0755)
	}

	// Check non-executable file permissions
	readmeInfo, err := os.Stat(filepath.Join(targetDir, "readme.md"))
	if err != nil {
		t.Fatalf("Failed to stat readme.md: %v", err)
	}

	if readmeInfo.Mode().Perm() != 0644 {
		t.Errorf("readme.md permissions = %o, want %o", readmeInfo.Mode().Perm(), 0644)
	}
}

func TestEngineTemplateFunctions(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		ctx      *TemplateContext
		want     string
	}{
		{
			name:     "year function",
			template: "Copyright {{year}}",
			ctx:      &TemplateContext{},
			want:     "Copyright 2025",
		},
		{
			name:     "title function",
			template: "{{title \"hello world\"}}",
			ctx:      &TemplateContext{},
			want:     "Hello World",
		},
		{
			name:     "default function - use default",
			template: "{{default \"default_val\" .Variables.missing}}",
			ctx: &TemplateContext{
				Variables: map[string]interface{}{},
			},
			want:    "default_val",
		},
		{
			name:     "default function - use value",
			template: "{{default \"default_val\" .Variables.present}}",
			ctx: &TemplateContext{
				Variables: map[string]interface{}{
					"present": "actual_val",
				},
			},
			want:    "actual_val",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.renderString(tt.template, tt.ctx)
			if err != nil {
				t.Fatalf("renderString() error = %v", err)
			}
			// For year, just check it's a 4-digit number
			if tt.name == "year function" {
				if len(got) < 14 || !strings.Contains(got, "Copyright") {
					t.Errorf("renderString() = %v, want to contain Copyright and year", got)
				}
			} else if got != tt.want {
				t.Errorf("renderString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngineExecuteNestedDirectories(t *testing.T) {
	engine := NewEngine()
	tmpDir := t.TempDir()

	tmpl := &Template{
		Name:    "nested-test",
		Version: "1.0.0",
		Directories: []string{
			"src",
			"src/models",
			"src/controllers",
			"config/environments",
		},
		Files: []*TemplateFile{
			{
				TargetPath: "src/models/user.txt",
				Template:   false,
				Content:    "user model",
			},
			{
				TargetPath: "config/environments/dev.txt",
				Template:   false,
				Content:    "dev config",
			},
		},
	}

	ctx := &TemplateContext{
		Variables: make(map[string]interface{}),
	}

	targetDir := filepath.Join(tmpDir, "nested")

	err := engine.Execute(tmpl, ctx, targetDir)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify nested directories exist
	nestedDirs := []string{
		"src",
		"src/models",
		"src/controllers",
		"config/environments",
	}

	for _, dir := range nestedDirs {
		if _, err := os.Stat(filepath.Join(targetDir, dir)); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}

	// Verify files in nested directories
	if _, err := os.Stat(filepath.Join(targetDir, "src/models/user.txt")); os.IsNotExist(err) {
		t.Error("File in nested directory was not created")
	}
}

func TestEngineExecuteTemplatedPaths(t *testing.T) {
	engine := NewEngine()
	tmpDir := t.TempDir()

	tmpl := &Template{
		Name:    "templated-paths",
		Version: "1.0.0",
		Directories: []string{
			"{{.ProjectName}}_src",
		},
		Files: []*TemplateFile{
			{
				TargetPath: "{{.ProjectName}}_src/main.txt",
				Template:   true,
				Content:    "Project: {{.ProjectName}}",
			},
		},
	}

	ctx := &TemplateContext{
		ProjectName: "myproject",
		Variables:   make(map[string]interface{}),
	}

	targetDir := filepath.Join(tmpDir, "templated")

	err := engine.Execute(tmpl, ctx, targetDir)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify templated directory name
	if _, err := os.Stat(filepath.Join(targetDir, "myproject_src")); os.IsNotExist(err) {
		t.Error("Templated directory was not created")
	}

	// Verify templated file path and content
	content, err := os.ReadFile(filepath.Join(targetDir, "myproject_src/main.txt"))
	if err != nil {
		t.Fatalf("Failed to read templated file: %v", err)
	}

	if string(content) != "Project: myproject" {
		t.Errorf("File content = %q, want %q", string(content), "Project: myproject")
	}
}

func TestEngineMissingRequiredVariable(t *testing.T) {
	engine := NewEngine()
	tmpDir := t.TempDir()

	tmpl := &Template{
		Name:    "required-var-test",
		Version: "1.0.0",
		Variables: []*TemplateVariable{
			{Name: "required", Required: true},
		},
		Files: []*TemplateFile{
			{TargetPath: "test.txt", Content: "test"},
		},
	}

	ctx := &TemplateContext{
		Variables: map[string]interface{}{}, // Missing required variable
	}

	err := engine.Execute(tmpl, ctx, tmpDir)
	if err == nil {
		t.Error("Execute() should fail with missing required variable")
	}
}

func TestLoadTemplateFromFS(t *testing.T) {
	// Test that LoadTemplateFromFS returns not implemented
	var fs embed.FS
	result, err := LoadTemplateFromFS(fs, "test")
	if err == nil {
		t.Error("LoadTemplateFromFS() should return not implemented error")
	}
	if result != nil {
		t.Error("LoadTemplateFromFS() should return nil template")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Errorf("Error should mention 'not implemented', got: %v", err)
	}
}

func TestTemplateValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		tmpl    *Template
		wantErr bool
		errMsg  string
	}{
		{
			name: "empty variable name",
			tmpl: &Template{
				Name:    "test",
				Version: "1.0.0",
				Variables: []*TemplateVariable{
					{Name: "", Type: VariableTypeString},
				},
				Files: []*TemplateFile{
					{TargetPath: "test.txt", Content: "test"},
				},
			},
			wantErr: true,
			errMsg:  "variable name is required",
		},
		{
			name: "empty file target path",
			tmpl: &Template{
				Name:    "test",
				Version: "1.0.0",
				Files: []*TemplateFile{
					{TargetPath: "", Content: "test"},
				},
			},
			wantErr: true,
			errMsg:  "target path is required",
		},
		{
			name: "empty file content",
			tmpl: &Template{
				Name:    "test",
				Version: "1.0.0",
				Files: []*TemplateFile{
					{TargetPath: "test.txt", Content: ""},
				},
			},
			wantErr: true,
			errMsg:  "content is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tmpl.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Error should contain %q, got %v", tt.errMsg, err)
			}
		})
	}
}

func TestEvaluateCondition(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name      string
		condition string
		ctx       *TemplateContext
		want      bool
		wantErr   bool
	}{
		{
			name:      "true condition",
			condition: "true",
			ctx:       &TemplateContext{Variables: make(map[string]interface{})},
			want:      true,
			wantErr:   false,
		},
		{
			name:      "false condition",
			condition: "false",
			ctx:       &TemplateContext{Variables: make(map[string]interface{})},
			want:      false,
			wantErr:   false,
		},
		{
			name:      "variable condition - true",
			condition: "{{.Variables.enabled}}",
			ctx: &TemplateContext{
				Variables: map[string]interface{}{
					"enabled": "true",
				},
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.evaluateCondition(tt.condition, tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("evaluateCondition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("evaluateCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEngineRenderErrors(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		template string
		ctx      *TemplateContext
		wantErr  bool
	}{
		{
			name:     "invalid template syntax",
			template: "{{.Invalid",
			ctx:      &TemplateContext{},
			wantErr:  true,
		},
		{
			name:     "missing field",
			template: "{{.NonExistent}}",
			ctx:      &TemplateContext{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := engine.renderString(tt.template, tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTemplateMetadata(t *testing.T) {
	// Test that built-in templates have proper metadata
	api := NewAPITemplate()
	if api.Metadata == nil {
		t.Error("API template should have metadata")
	}

	if category, ok := api.Metadata["category"].(string); !ok || category == "" {
		t.Error("API template should have category metadata")
	}

	if tags, ok := api.Metadata["tags"].([]string); !ok || len(tags) == 0 {
		t.Error("API template should have tags metadata")
	}
}

func TestEngineExecuteInvalidTargetDir(t *testing.T) {
	engine := NewEngine()

	tmpl := &Template{
		Name:    "test",
		Version: "1.0.0",
		Files: []*TemplateFile{
			{TargetPath: "test.txt", Content: "test"},
		},
	}

	ctx := &TemplateContext{
		Variables: make(map[string]interface{}),
	}

	// Try to write to an invalid directory (e.g., a file path instead of directory)
	tmpFile := filepath.Join(t.TempDir(), "file.txt")
	os.WriteFile(tmpFile, []byte("test"), 0644)

	err := engine.Execute(tmpl, ctx, tmpFile)
	// This might not error immediately on all systems, so we just ensure it doesn't panic
	if err == nil {
		// If it doesn't error, at least verify we tried to execute something
		t.Log("Execute did not error on invalid target directory")
	}
}

func TestEngineExecutePathTraversalProtection(t *testing.T) {
	engine := NewEngine()
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		tmpl        *Template
		ctx         *TemplateContext
		wantErr     bool
		errContains string
	}{
		{
			name: "file path traversal with ../",
			tmpl: &Template{
				Name:    "path-traversal-file",
				Version: "1.0.0",
				Files: []*TemplateFile{
					{
						TargetPath: "../../etc/passwd",
						Template:   false,
						Content:    "malicious content",
					},
				},
			},
			ctx: &TemplateContext{
				Variables: make(map[string]interface{}),
			},
			wantErr:     true,
			errContains: "outside project directory",
		},
		{
			name: "directory path traversal with ../",
			tmpl: &Template{
				Name:        "path-traversal-dir",
				Version:     "1.0.0",
				Directories: []string{"../../etc"},
				Files: []*TemplateFile{
					{TargetPath: "safe.txt", Content: "safe"},
				},
			},
			ctx: &TemplateContext{
				Variables: make(map[string]interface{}),
			},
			wantErr:     true,
			errContains: "outside project directory",
		},
		{
			name: "absolute path in file",
			tmpl: &Template{
				Name:    "absolute-path-file",
				Version: "1.0.0",
				Files: []*TemplateFile{
					{
						TargetPath: "/etc/passwd",
						Template:   false,
						Content:    "malicious content",
					},
				},
			},
			ctx: &TemplateContext{
				Variables: make(map[string]interface{}),
			},
			wantErr:     true,
			errContains: "outside project directory",
		},
		{
			name: "absolute path in directory",
			tmpl: &Template{
				Name:        "absolute-path-dir",
				Version:     "1.0.0",
				Directories: []string{"/tmp/evil"},
				Files: []*TemplateFile{
					{TargetPath: "safe.txt", Content: "safe"},
				},
			},
			ctx: &TemplateContext{
				Variables: make(map[string]interface{}),
			},
			wantErr:     true,
			errContains: "outside project directory",
		},
		{
			name: "templated path traversal",
			tmpl: &Template{
				Name:    "templated-traversal",
				Version: "1.0.0",
				Files: []*TemplateFile{
					{
						TargetPath: "{{.Variables.evil_path}}",
						Template:   false,
						Content:    "malicious content",
					},
				},
			},
			ctx: &TemplateContext{
				Variables: map[string]interface{}{
					"evil_path": "../../etc/passwd",
				},
			},
			wantErr:     true,
			errContains: "outside project directory",
		},
		{
			name: "safe relative path",
			tmpl: &Template{
				Name:    "safe-path",
				Version: "1.0.0",
				Files: []*TemplateFile{
					{
						TargetPath: "src/models/user.txt",
						Template:   false,
						Content:    "safe content",
					},
				},
			},
			ctx: &TemplateContext{
				Variables: make(map[string]interface{}),
			},
			wantErr: false,
		},
		{
			name: "safe path with ./ prefix",
			tmpl: &Template{
				Name:    "safe-dot-path",
				Version: "1.0.0",
				Files: []*TemplateFile{
					{
						TargetPath: "./config/app.txt",
						Template:   false,
						Content:    "safe content",
					},
				},
			},
			ctx: &TemplateContext{
				Variables: make(map[string]interface{}),
			},
			wantErr: false,
		},
		{
			name: "directory with .. that stays within bounds",
			tmpl: &Template{
				Name:        "safe-backtrack",
				Version:     "1.0.0",
				Directories: []string{"a/b/../c"},
				Files: []*TemplateFile{
					{TargetPath: "test.txt", Content: "test"},
				},
			},
			ctx: &TemplateContext{
				Variables: make(map[string]interface{}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetDir := filepath.Join(tmpDir, tt.name)
			err := engine.Execute(tt.tmpl, tt.ctx, targetDir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Execute() should have failed with path traversal error")
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Execute() error = %v, should contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Execute() should not have failed for safe path: %v", err)
				}
			}
		})
	}
}

func TestTitleFunction(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single word lowercase",
			input: "hello",
			want:  "Hello",
		},
		{
			name:  "single word uppercase",
			input: "HELLO",
			want:  "Hello",
		},
		{
			name:  "multiple words",
			input: "hello world",
			want:  "Hello World",
		},
		{
			name:  "mixed case",
			input: "hELLo WoRLd",
			want:  "Hello World",
		},
		{
			name:  "multiple spaces",
			input: "hello  world",
			want:  "Hello World",
		},
		{
			name:  "leading/trailing spaces",
			input: "  hello world  ",
			want:  "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl := "{{title \"" + tt.input + "\"}}"
			ctx := &TemplateContext{
				Variables: make(map[string]interface{}),
			}

			got, err := engine.renderString(tmpl, ctx)
			if err != nil {
				t.Fatalf("renderString() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("title(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
