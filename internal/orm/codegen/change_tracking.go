// Package codegen provides code generation for change tracking methods
package codegen

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// ChangeTrackingGenerator generates change tracking methods for resources
type ChangeTrackingGenerator struct{}

// NewChangeTrackingGenerator creates a new change tracking generator
func NewChangeTrackingGenerator() *ChangeTrackingGenerator {
	return &ChangeTrackingGenerator{}
}

// Generate generates all change tracking methods for a resource
func (g *ChangeTrackingGenerator) Generate(resource *schema.ResourceSchema) (string, error) {
	var code strings.Builder

	code.WriteString("// Change Tracking Methods\n")
	code.WriteString("// These methods track field modifications for efficient updates and conditional hooks\n\n")

	// Generate field-specific change methods for each field
	for fieldName, field := range resource.Fields {
		// Skip internal fields
		if fieldName == "id" || fieldName == "created_at" || fieldName == "updated_at" {
			continue
		}

		code.WriteString(g.generateFieldChangedMethod(resource, fieldName))
		code.WriteString("\n\n")
		code.WriteString(g.generatePreviousValueMethod(resource, fieldName, field))
		code.WriteString("\n\n")
		code.WriteString(g.generateSetterMethod(resource, fieldName, field))
		code.WriteString("\n\n")
	}

	// Generate general change tracking methods
	code.WriteString(g.generateChangedMethod(resource))
	code.WriteString("\n\n")
	code.WriteString(g.generateChangedFieldsMethod(resource))
	code.WriteString("\n\n")
	code.WriteString(g.generateHasChangesMethod(resource))
	code.WriteString("\n\n")
	code.WriteString(g.generateReloadMethod(resource))
	code.WriteString("\n\n")
	code.WriteString(g.generateGetChangedDataMethod(resource))

	return code.String(), nil
}

// generateFieldChangedMethod generates a method to check if a specific field changed
func (g *ChangeTrackingGenerator) generateFieldChangedMethod(resource *schema.ResourceSchema, fieldName string) string {
	methodName := toPascalCase(fieldName) + "Changed"

	return fmt.Sprintf(`// %s returns true if the %s field has been modified
func (r *%s) %s() bool {
	tracker, ok := r.changeTracker()
	if !ok {
		return false
	}
	return tracker.Changed("%s")
}`, methodName, fieldName, resource.Name, methodName, fieldName)
}

// generatePreviousValueMethod generates a method to get the previous value of a field
func (g *ChangeTrackingGenerator) generatePreviousValueMethod(resource *schema.ResourceSchema, fieldName string, field *schema.Field) string {
	methodName := "Previous" + toPascalCase(fieldName)
	goType := mapTypeToGo(field.Type)

	// Generate conversion logic for the interface{} value
	var conversionCode string
	if field.Type.Nullable {
		conversionCode = fmt.Sprintf(`	if val == nil {
		return nil
	}
	if v, ok := val.(%s); ok {
		return &v
	}
	return nil`, strings.TrimPrefix(goType, "*"))
	} else {
		conversionCode = fmt.Sprintf(`	if v, ok := val.(%s); ok {
		return v
	}
	var zero %s
	return zero`, goType, goType)
	}

	return fmt.Sprintf(`// %s returns the previous value of %s before any modifications
func (r *%s) %s() %s {
	tracker, ok := r.changeTracker()
	if !ok {
		return r.%s
	}
	val := tracker.PreviousValue("%s")
%s
}`, methodName, fieldName, resource.Name, methodName, goType, toPascalCase(fieldName), fieldName, conversionCode)
}

// generateSetterMethod generates a setter method with change tracking
func (g *ChangeTrackingGenerator) generateSetterMethod(resource *schema.ResourceSchema, fieldName string, field *schema.Field) string {
	methodName := "Set" + toPascalCase(fieldName)
	goType := mapTypeToGo(field.Type)
	pascalField := toPascalCase(fieldName)

	return fmt.Sprintf(`// %s sets the %s field and tracks the change
func (r *%s) %s(value %s) {
	r.%s = value
	if tracker, ok := r.changeTracker(); ok {
		tracker.SetFieldValue("%s", value)
	}
}`, methodName, fieldName, resource.Name, methodName, goType, pascalField, fieldName)
}

// generateChangedMethod generates a general method to check if any field changed
func (g *ChangeTrackingGenerator) generateChangedMethod(resource *schema.ResourceSchema) string {
	return fmt.Sprintf(`// Changed returns true if the specified field has been modified
func (r *%s) Changed(field string) bool {
	tracker, ok := r.changeTracker()
	if !ok {
		return false
	}
	return tracker.Changed(field)
}`, resource.Name)
}

// generateChangedFieldsMethod generates a method to get all changed field names
func (g *ChangeTrackingGenerator) generateChangedFieldsMethod(resource *schema.ResourceSchema) string {
	return fmt.Sprintf(`// ChangedFields returns a list of all fields that have been modified
func (r *%s) ChangedFields() []string {
	tracker, ok := r.changeTracker()
	if !ok {
		return nil
	}
	return tracker.ChangedFields()
}`, resource.Name)
}

// generateHasChangesMethod generates a method to check if any changes exist
func (g *ChangeTrackingGenerator) generateHasChangesMethod(resource *schema.ResourceSchema) string {
	return fmt.Sprintf(`// HasChanges returns true if any fields have been modified
func (r *%s) HasChanges() bool {
	tracker, ok := r.changeTracker()
	if !ok {
		return false
	}
	return tracker.HasChanges()
}`, resource.Name)
}

// generateReloadMethod generates a method to reload the resource from database
func (g *ChangeTrackingGenerator) generateReloadMethod(resource *schema.ResourceSchema) string {
	return fmt.Sprintf(`// Reload reloads the resource from the database, discarding any unsaved changes
func (r *%s) Reload(ctx context.Context, db *sql.DB) error {
	ops := crud.NewOperations(db, resourceSchema_%s, nil, nil, nil, nil)
	data, err := ops.Find(ctx, r.ID)
	if err != nil {
		return err
	}

	// Update all fields from database
	if err := r.loadFromData(data); err != nil {
		return err
	}

	// Reset change tracking
	if tracker, ok := r.changeTracker(); ok {
		tracker.Reset()
	}

	return nil
}`, resource.Name, resource.Name)
}

// generateGetChangedDataMethod generates a method to get only changed fields as a map
func (g *ChangeTrackingGenerator) generateGetChangedDataMethod(resource *schema.ResourceSchema) string {
	return fmt.Sprintf(`// GetChangedData returns a map of only the changed fields with their new values
func (r *%s) GetChangedData() map[string]interface{} {
	tracker, ok := r.changeTracker()
	if !ok {
		return make(map[string]interface{})
	}
	return tracker.GetChangedData()
}`, resource.Name)
}

// Helper method to generate the change tracker accessor (this would be added to the main struct)
func (g *ChangeTrackingGenerator) GenerateChangeTrackerField() string {
	return `	// Internal change tracking
	__changeTracker__ *tracking.ChangeTracker
	__changeTrackerMu__ sync.RWMutex`
}

// GenerateChangeTrackerAccessor generates the accessor method for the change tracker
func (g *ChangeTrackingGenerator) GenerateChangeTrackerAccessor(resource *schema.ResourceSchema) string {
	return fmt.Sprintf(`// changeTracker returns the change tracker for this resource
func (r *%s) changeTracker() (*tracking.ChangeTracker, bool) {
	r.__changeTrackerMu__.RLock()
	defer r.__changeTrackerMu__.RUnlock()
	if r.__changeTracker__ == nil {
		return nil, false
	}
	return r.__changeTracker__, true
}

// initChangeTracker initializes the change tracker with original and current states
func (r *%s) initChangeTracker(original, current map[string]interface{}) {
	r.__changeTrackerMu__.Lock()
	defer r.__changeTrackerMu__.Unlock()
	r.__changeTracker__ = tracking.NewChangeTracker(original, current)
}

// resetChangeTracker resets the change tracker after a successful save
func (r *%s) resetChangeTracker() {
	r.__changeTrackerMu__.Lock()
	defer r.__changeTrackerMu__.Unlock()
	if r.__changeTracker__ != nil {
		r.__changeTracker__.Reset()
	}
}`, resource.Name, resource.Name, resource.Name)
}

// mapTypeToGo maps a Conduit TypeSpec to a Go type
func mapTypeToGo(typeSpec *schema.TypeSpec) string {
	if typeSpec == nil {
		return "interface{}"
	}

	var baseType string

	// Handle complex types
	if typeSpec.ArrayElement != nil {
		elemType := mapTypeToGo(typeSpec.ArrayElement)
		baseType = "[]" + elemType
	} else if typeSpec.HashKey != nil && typeSpec.HashValue != nil {
		keyType := mapTypeToGo(typeSpec.HashKey)
		valType := mapTypeToGo(typeSpec.HashValue)
		baseType = fmt.Sprintf("map[%s]%s", keyType, valType)
	} else if len(typeSpec.StructFields) > 0 {
		baseType = "map[string]interface{}"
	} else {
		// Map primitive types
		baseType = mapPrimitiveTypeToGo(typeSpec.BaseType)
	}

	// Add pointer for nullable types
	if typeSpec.Nullable {
		return "*" + baseType
	}

	return baseType
}

// mapPrimitiveTypeToGo maps a primitive type to Go
func mapPrimitiveTypeToGo(t schema.PrimitiveType) string {
	switch t {
	case schema.TypeString, schema.TypeText, schema.TypeMarkdown:
		return "string"
	case schema.TypeInt:
		return "int"
	case schema.TypeBigInt:
		return "int64"
	case schema.TypeFloat:
		return "float64"
	case schema.TypeDecimal:
		return "float64" // Use float64 for decimal in Go
	case schema.TypeBool:
		return "bool"
	case schema.TypeTimestamp, schema.TypeDate, schema.TypeTime:
		return "time.Time"
	case schema.TypeUUID:
		return "string" // UUID as string (could use google/uuid package)
	case schema.TypeULID:
		return "string"
	case schema.TypeEmail, schema.TypeURL, schema.TypePhone:
		return "string"
	case schema.TypeJSON, schema.TypeJSONB:
		return "map[string]interface{}"
	case schema.TypeEnum:
		return "string" // Enums as strings
	default:
		return "interface{}"
	}
}
