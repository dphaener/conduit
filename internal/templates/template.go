package templates

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// VariableType represents the type of template variable
type VariableType string

const (
	VariableTypeString  VariableType = "string"
	VariableTypeInt     VariableType = "int"
	VariableTypeBool    VariableType = "bool"
	VariableTypeSelect  VariableType = "select"
	VariableTypeConfirm VariableType = "confirm"
)

// Template represents a project template
type Template struct {
	Name        string
	Description string
	Version     string
	Variables   []*TemplateVariable
	Files       []*TemplateFile
	Directories []string
	Hooks       *TemplateHooks
	Metadata    map[string]interface{}
}

// TemplateVariable represents a configurable variable in a template
type TemplateVariable struct {
	Name        string
	Description string
	Type        VariableType
	Default     interface{}
	Required    bool
	Options     []string
	Prompt      string
}

// TemplateFile represents a file in a template
type TemplateFile struct {
	SourcePath  string
	TargetPath  string
	Content     string
	Template    bool // Use template engine
	Executable  bool
	Condition   string // Conditional file generation
}

// TemplateHooks contains lifecycle hooks for template execution
type TemplateHooks struct {
	BeforeCreate []string
	AfterCreate  []string
}

// TemplateContext contains all data for template execution
type TemplateContext struct {
	ProjectName string
	Variables   map[string]interface{}
	Timestamp   time.Time
}

// Engine is the template rendering engine
type Engine struct {
	funcs template.FuncMap
}

// NewEngine creates a new template engine
func NewEngine() *Engine {
	return &Engine{
		funcs: template.FuncMap{
			"upper": func(s string) string { return string(bytes.ToUpper([]byte(s))) },
			"lower": func(s string) string { return string(bytes.ToLower([]byte(s))) },
			"title": func(s string) string {
				if s == "" {
					return s
				}
				words := strings.Fields(s)
				for i, word := range words {
					if len(word) > 0 {
						words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
					}
				}
				return strings.Join(words, " ")
			},
			"now":     time.Now,
			"year":    func() int { return time.Now().Year() },
			"default": func(def, val interface{}) interface{} { if val == nil { return def }; return val },
		},
	}
}

// Execute executes a template and creates a project
func (e *Engine) Execute(tmpl *Template, ctx *TemplateContext, targetDir string) error {
	// Validate context
	if err := e.validateContext(tmpl, ctx); err != nil {
		return fmt.Errorf("invalid template context: %w", err)
	}

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Create directories
	for _, dir := range tmpl.Directories {
		dirPath, err := e.renderString(dir, ctx)
		if err != nil {
			return fmt.Errorf("failed to render directory path %s: %w", dir, err)
		}

		// Sanitize path to prevent directory traversal
		dirPath = filepath.Clean(dirPath)

		// Reject absolute paths
		if filepath.IsAbs(dirPath) {
			return fmt.Errorf("invalid directory path: %s attempts to write outside project directory", dir)
		}

		fullPath := filepath.Join(targetDir, dirPath)
		cleanFullPath := filepath.Clean(fullPath)
		cleanTargetDir := filepath.Clean(targetDir) + string(filepath.Separator)

		// Ensure the resolved path is still within targetDir
		if !strings.HasPrefix(cleanFullPath+string(filepath.Separator), cleanTargetDir) {
			return fmt.Errorf("invalid directory path: %s attempts to write outside project directory", dir)
		}

		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
		}
	}

	// Process files
	for _, file := range tmpl.Files {
		// Check condition
		if file.Condition != "" {
			shouldCreate, err := e.evaluateCondition(file.Condition, ctx)
			if err != nil {
				return fmt.Errorf("failed to evaluate condition for %s: %w", file.TargetPath, err)
			}
			if !shouldCreate {
				continue
			}
		}

		// Render target path
		targetPath, err := e.renderString(file.TargetPath, ctx)
		if err != nil {
			return fmt.Errorf("failed to render target path %s: %w", file.TargetPath, err)
		}

		// Sanitize path to prevent directory traversal
		targetPath = filepath.Clean(targetPath)

		// Reject absolute paths
		if filepath.IsAbs(targetPath) {
			return fmt.Errorf("invalid target path: %s attempts to write outside project directory", file.TargetPath)
		}

		fullPath := filepath.Join(targetDir, targetPath)
		cleanFullPath := filepath.Clean(fullPath)
		cleanTargetDir := filepath.Clean(targetDir) + string(filepath.Separator)

		// Ensure the resolved path is still within targetDir
		if !strings.HasPrefix(cleanFullPath+string(filepath.Separator), cleanTargetDir) {
			return fmt.Errorf("invalid target path: %s attempts to write outside project directory", file.TargetPath)
		}

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", fullPath, err)
		}

		// Render content
		var content string
		if file.Template {
			content, err = e.renderString(file.Content, ctx)
			if err != nil {
				return fmt.Errorf("failed to render template %s: %w", file.TargetPath, err)
			}
		} else {
			content = file.Content
		}

		// Write file
		mode := os.FileMode(0644)
		if file.Executable {
			mode = 0755
		}
		if err := os.WriteFile(fullPath, []byte(content), mode); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return nil
}

// renderString renders a template string with the given context
func (e *Engine) renderString(tmplStr string, ctx *TemplateContext) (string, error) {
	tmpl, err := template.New("").Funcs(e.funcs).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// validateContext validates that all required variables are provided
func (e *Engine) validateContext(tmpl *Template, ctx *TemplateContext) error {
	for _, v := range tmpl.Variables {
		if v.Required {
			if _, ok := ctx.Variables[v.Name]; !ok {
				return fmt.Errorf("required variable %s not provided", v.Name)
			}
		}
	}
	return nil
}

// evaluateCondition evaluates a simple boolean condition
func (e *Engine) evaluateCondition(condition string, ctx *TemplateContext) (bool, error) {
	// For MVP, support simple variable checks: {{.Variables.name}}
	result, err := e.renderString(condition, ctx)
	if err != nil {
		return false, err
	}
	return result == "true", nil
}

// LoadTemplateFromFS loads a template from an embedded filesystem
func LoadTemplateFromFS(fs embed.FS, name string) (*Template, error) {
	// This is a placeholder for loading from YAML/JSON metadata
	// For now, we'll construct templates programmatically
	return nil, fmt.Errorf("not implemented")
}

// Validate validates a template structure
func (t *Template) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if t.Version == "" {
		return fmt.Errorf("template version is required")
	}
	if len(t.Files) == 0 {
		return fmt.Errorf("template must have at least one file")
	}

	// Validate variables
	varNames := make(map[string]bool)
	for _, v := range t.Variables {
		if v.Name == "" {
			return fmt.Errorf("variable name is required")
		}
		if varNames[v.Name] {
			return fmt.Errorf("duplicate variable name: %s", v.Name)
		}
		varNames[v.Name] = true

		if v.Type == VariableTypeSelect && len(v.Options) == 0 {
			return fmt.Errorf("select variable %s must have options", v.Name)
		}
	}

	// Validate files
	for _, f := range t.Files {
		if f.TargetPath == "" {
			return fmt.Errorf("file target path is required")
		}
		if f.Content == "" {
			return fmt.Errorf("file content is required for %s", f.TargetPath)
		}
	}

	return nil
}
