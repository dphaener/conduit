package templates

import (
	"testing"
)

func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()

	tmpl := &Template{
		Name:    "test-template",
		Version: "1.0.0",
		Files: []*TemplateFile{
			{TargetPath: "test.txt", Content: "test"},
		},
	}

	// Register template
	err := registry.Register(tmpl)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Try to register duplicate
	err = registry.Register(tmpl)
	if err == nil {
		t.Error("Register() should fail for duplicate template")
	}
}

func TestRegistryGet(t *testing.T) {
	registry := NewRegistry()

	tmpl := &Template{
		Name:    "test-template",
		Version: "1.0.0",
		Files: []*TemplateFile{
			{TargetPath: "test.txt", Content: "test"},
		},
	}

	registry.Register(tmpl)

	// Get existing template
	got, err := registry.Get("test-template")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Name != tmpl.Name {
		t.Errorf("Get() name = %v, want %v", got.Name, tmpl.Name)
	}

	// Get non-existent template
	_, err = registry.Get("non-existent")
	if err == nil {
		t.Error("Get() should fail for non-existent template")
	}
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry()

	templates := []*Template{
		{
			Name:    "template1",
			Version: "1.0.0",
			Files:   []*TemplateFile{{TargetPath: "test.txt", Content: "test"}},
		},
		{
			Name:    "template2",
			Version: "1.0.0",
			Files:   []*TemplateFile{{TargetPath: "test.txt", Content: "test"}},
		},
		{
			Name:    "template3",
			Version: "1.0.0",
			Files:   []*TemplateFile{{TargetPath: "test.txt", Content: "test"}},
		},
	}

	for _, tmpl := range templates {
		registry.Register(tmpl)
	}

	list := registry.List()

	if len(list) != 3 {
		t.Errorf("List() returned %d templates, want 3", len(list))
	}

	// Verify all templates are in the list
	found := make(map[string]bool)
	for _, tmpl := range list {
		found[tmpl.Name] = true
	}

	for _, tmpl := range templates {
		if !found[tmpl.Name] {
			t.Errorf("Template %s not found in list", tmpl.Name)
		}
	}
}

func TestRegistryExists(t *testing.T) {
	registry := NewRegistry()

	tmpl := &Template{
		Name:    "test-template",
		Version: "1.0.0",
		Files:   []*TemplateFile{{TargetPath: "test.txt", Content: "test"}},
	}

	registry.Register(tmpl)

	if !registry.Exists("test-template") {
		t.Error("Exists() should return true for registered template")
	}

	if registry.Exists("non-existent") {
		t.Error("Exists() should return false for non-existent template")
	}
}

func TestRegistryUnregister(t *testing.T) {
	registry := NewRegistry()

	tmpl := &Template{
		Name:    "test-template",
		Version: "1.0.0",
		Files:   []*TemplateFile{{TargetPath: "test.txt", Content: "test"}},
	}

	registry.Register(tmpl)

	// Unregister existing template
	err := registry.Unregister("test-template")
	if err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	if registry.Exists("test-template") {
		t.Error("Template should not exist after unregister")
	}

	// Try to unregister non-existent template
	err = registry.Unregister("test-template")
	if err == nil {
		t.Error("Unregister() should fail for non-existent template")
	}
}

func TestRegisterBuiltinTemplates(t *testing.T) {
	// Create a fresh registry
	registry := NewRegistry()

	// Replace default registry temporarily
	oldRegistry := defaultRegistry
	defaultRegistry = registry
	defer func() {
		defaultRegistry = oldRegistry
	}()

	err := RegisterBuiltinTemplates()
	if err != nil {
		t.Fatalf("RegisterBuiltinTemplates() error = %v", err)
	}

	// Verify all built-in templates are registered
	expectedTemplates := []string{"api", "web", "microservice"}

	for _, name := range expectedTemplates {
		if !registry.Exists(name) {
			t.Errorf("Built-in template %s not registered", name)
		}

		tmpl, err := registry.Get(name)
		if err != nil {
			t.Errorf("Failed to get built-in template %s: %v", name, err)
			continue
		}

		// Validate the template
		if err := tmpl.Validate(); err != nil {
			t.Errorf("Built-in template %s is invalid: %v", name, err)
		}
	}

	// Verify the registry has exactly the expected number of templates
	list := registry.List()
	if len(list) != len(expectedTemplates) {
		t.Errorf("Registry has %d templates, want %d", len(list), len(expectedTemplates))
	}
}

func TestBuiltinTemplatesValid(t *testing.T) {
	tests := []struct {
		name     string
		template *Template
	}{
		{"API template", NewAPITemplate()},
		{"Web template", NewWebTemplate()},
		{"Microservice template", NewMicroserviceTemplate()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.template.Validate()
			if err != nil {
				t.Errorf("%s validation failed: %v", tt.name, err)
			}

			// Check basic properties
			if tt.template.Name == "" {
				t.Error("Template name is empty")
			}
			if tt.template.Version == "" {
				t.Error("Template version is empty")
			}
			if tt.template.Description == "" {
				t.Error("Template description is empty")
			}
			if len(tt.template.Files) == 0 {
				t.Error("Template has no files")
			}

			// Check that all variables have proper configuration
			for _, v := range tt.template.Variables {
				if v.Name == "" {
					t.Error("Variable has no name")
				}
				if v.Prompt == "" {
					t.Error("Variable has no prompt")
				}
				if v.Type == "" {
					t.Error("Variable has no type")
				}
			}

			// Check that all files have valid paths and content
			for _, f := range tt.template.Files {
				if f.TargetPath == "" {
					t.Error("File has no target path")
				}
				if f.Content == "" {
					t.Error("File has no content")
				}
			}
		})
	}
}

func TestTemplatesConcurrency(t *testing.T) {
	registry := NewRegistry()

	tmpl := &Template{
		Name:    "test-template",
		Version: "1.0.0",
		Files:   []*TemplateFile{{TargetPath: "test.txt", Content: "test"}},
	}

	// Register template
	registry.Register(tmpl)

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < 100; j++ {
				registry.Get("test-template")
				registry.Exists("test-template")
				registry.List()
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
