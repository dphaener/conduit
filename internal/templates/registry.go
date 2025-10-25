package templates

import (
	"fmt"
	"sync"
)

// Registry manages available project templates
type Registry struct {
	templates map[string]*Template
	mutex     sync.RWMutex
}

// NewRegistry creates a new template registry
func NewRegistry() *Registry {
	return &Registry{
		templates: make(map[string]*Template),
	}
}

// Register registers a template in the registry
func (r *Registry) Register(tmpl *Template) error {
	if err := tmpl.Validate(); err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.templates[tmpl.Name]; exists {
		return fmt.Errorf("template %s already registered", tmpl.Name)
	}

	r.templates[tmpl.Name] = tmpl
	return nil
}

// Get retrieves a template by name
func (r *Registry) Get(name string) (*Template, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	tmpl, exists := r.templates[name]
	if !exists {
		return nil, fmt.Errorf("template %s not found", name)
	}

	return tmpl, nil
}

// List returns all registered templates
func (r *Registry) List() []*Template {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	templates := make([]*Template, 0, len(r.templates))
	for _, tmpl := range r.templates {
		templates = append(templates, tmpl)
	}

	return templates
}

// Exists checks if a template exists
func (r *Registry) Exists(name string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.templates[name]
	return exists
}

// Unregister removes a template from the registry
func (r *Registry) Unregister(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.templates[name]; !exists {
		return fmt.Errorf("template %s not found", name)
	}

	delete(r.templates, name)
	return nil
}

// Default registry instance
var defaultRegistry = NewRegistry()

// DefaultRegistry returns the default template registry
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// SetDefaultRegistry sets the default template registry (useful for testing)
func SetDefaultRegistry(r *Registry) {
	defaultRegistry = r
}

// RegisterBuiltinTemplates registers all built-in templates
func RegisterBuiltinTemplates() error {
	templates := []*Template{
		NewAPITemplate(),
		NewWebTemplate(),
		NewMicroserviceTemplate(),
	}

	for _, tmpl := range templates {
		if err := defaultRegistry.Register(tmpl); err != nil {
			return fmt.Errorf("failed to register template %s: %w", tmpl.Name, err)
		}
	}

	return nil
}
