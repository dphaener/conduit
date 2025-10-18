package migrate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/conduit-lang/conduit/internal/orm/schema"
)

// ChangeType represents the type of schema change
type ChangeType int

const (
	ChangeAddResource ChangeType = iota
	ChangeDropResource
	ChangeAddField
	ChangeDropField
	ChangeModifyField
	ChangeAddRelationship
	ChangeDropRelationship
	ChangeModifyRelationship
	ChangeAddIndex
	ChangeDropIndex
)

// String returns the string representation of the change type
func (c ChangeType) String() string {
	switch c {
	case ChangeAddResource:
		return "add_resource"
	case ChangeDropResource:
		return "drop_resource"
	case ChangeAddField:
		return "add_field"
	case ChangeDropField:
		return "drop_field"
	case ChangeModifyField:
		return "modify_field"
	case ChangeAddRelationship:
		return "add_relationship"
	case ChangeDropRelationship:
		return "drop_relationship"
	case ChangeModifyRelationship:
		return "modify_relationship"
	case ChangeAddIndex:
		return "add_index"
	case ChangeDropIndex:
		return "drop_index"
	default:
		return "unknown"
	}
}

// SchemaChange represents a detected change between schemas
type SchemaChange struct {
	Type     ChangeType
	Resource string
	Field    string
	Relation string
	OldValue interface{}
	NewValue interface{}
	Breaking bool
	DataLoss bool
}

// Differ compares old and new schemas to detect changes
type Differ struct {
	oldSchemas map[string]*schema.ResourceSchema
	newSchemas map[string]*schema.ResourceSchema
}

// NewDiffer creates a new schema differ
func NewDiffer(oldSchemas, newSchemas map[string]*schema.ResourceSchema) *Differ {
	return &Differ{
		oldSchemas: oldSchemas,
		newSchemas: newSchemas,
	}
}

// ComputeDiff computes all changes between old and new schemas
func (d *Differ) ComputeDiff() []SchemaChange {
	var changes []SchemaChange

	// Get sorted resource names for deterministic ordering
	oldNames := getSortedResourceNames(d.oldSchemas)
	newNames := getSortedResourceNames(d.newSchemas)

	// Detect added resources
	for _, name := range setDifference(newNames, oldNames) {
		changes = append(changes, SchemaChange{
			Type:     ChangeAddResource,
			Resource: name,
			NewValue: d.newSchemas[name],
			Breaking: false,
			DataLoss: false,
		})
	}

	// Detect dropped resources
	for _, name := range setDifference(oldNames, newNames) {
		changes = append(changes, SchemaChange{
			Type:     ChangeDropResource,
			Resource: name,
			OldValue: d.oldSchemas[name],
			Breaking: true,
			DataLoss: true,
		})
	}

	// Detect changes in existing resources
	for _, name := range setIntersection(oldNames, newNames) {
		oldRes := d.oldSchemas[name]
		newRes := d.newSchemas[name]

		changes = append(changes, d.diffFields(name, oldRes, newRes)...)
		changes = append(changes, d.diffRelationships(name, oldRes, newRes)...)
	}

	return changes
}

// diffFields compares fields between old and new resource
func (d *Differ) diffFields(resourceName string, oldRes, newRes *schema.ResourceSchema) []SchemaChange {
	var changes []SchemaChange

	oldFields := getSortedFieldNames(oldRes.Fields)
	newFields := getSortedFieldNames(newRes.Fields)

	// Added fields
	for _, fieldName := range setDifference(newFields, oldFields) {
		newField := newRes.Fields[fieldName]
		breaking := !newField.Type.Nullable && newField.Type.Default == nil

		changes = append(changes, SchemaChange{
			Type:     ChangeAddField,
			Resource: resourceName,
			Field:    fieldName,
			NewValue: newField,
			Breaking: breaking,
			DataLoss: false,
		})
	}

	// Dropped fields
	for _, fieldName := range setDifference(oldFields, newFields) {
		changes = append(changes, SchemaChange{
			Type:     ChangeDropField,
			Resource: resourceName,
			Field:    fieldName,
			OldValue: oldRes.Fields[fieldName],
			Breaking: true,
			DataLoss: true,
		})
	}

	// Modified fields
	for _, fieldName := range setIntersection(oldFields, newFields) {
		oldField := oldRes.Fields[fieldName]
		newField := newRes.Fields[fieldName]

		if !d.fieldsEqual(oldField, newField) {
			breaking := d.isBreakingFieldChange(oldField, newField)
			dataLoss := d.causesDataLoss(oldField, newField)

			changes = append(changes, SchemaChange{
				Type:     ChangeModifyField,
				Resource: resourceName,
				Field:    fieldName,
				OldValue: oldField,
				NewValue: newField,
				Breaking: breaking,
				DataLoss: dataLoss,
			})
		}
	}

	return changes
}

// diffRelationships compares relationships between old and new resource
func (d *Differ) diffRelationships(resourceName string, oldRes, newRes *schema.ResourceSchema) []SchemaChange {
	var changes []SchemaChange

	oldRels := getSortedRelationshipNames(oldRes.Relationships)
	newRels := getSortedRelationshipNames(newRes.Relationships)

	// Added relationships
	for _, relName := range setDifference(newRels, oldRels) {
		changes = append(changes, SchemaChange{
			Type:     ChangeAddRelationship,
			Resource: resourceName,
			Relation: relName,
			NewValue: newRes.Relationships[relName],
			Breaking: false,
			DataLoss: false,
		})
	}

	// Dropped relationships
	for _, relName := range setDifference(oldRels, newRels) {
		changes = append(changes, SchemaChange{
			Type:     ChangeDropRelationship,
			Resource: resourceName,
			Relation: relName,
			OldValue: oldRes.Relationships[relName],
			Breaking: true,
			DataLoss: true,
		})
	}

	// Modified relationships
	for _, relName := range setIntersection(oldRels, newRels) {
		oldRel := oldRes.Relationships[relName]
		newRel := newRes.Relationships[relName]

		if !d.relationshipsEqual(oldRel, newRel) {
			breaking := d.isBreakingRelationshipChange(oldRel, newRel)

			changes = append(changes, SchemaChange{
				Type:     ChangeModifyRelationship,
				Resource: resourceName,
				Relation: relName,
				OldValue: oldRel,
				NewValue: newRel,
				Breaking: breaking,
				DataLoss: true, // Relationship changes can affect data
			})
		}
	}

	return changes
}

// fieldsEqual checks if two fields are equal
func (d *Differ) fieldsEqual(old, new *schema.Field) bool {
	if old.Name != new.Name {
		return false
	}

	if !d.typeSpecsEqual(old.Type, new.Type) {
		return false
	}

	// Compare constraints properly (both count and values)
	if !d.constraintsEqual(old.Constraints, new.Constraints) {
		return false
	}

	return true
}

// constraintsEqual checks if two constraint lists are equal
func (d *Differ) constraintsEqual(old, new []schema.Constraint) bool {
	if len(old) != len(new) {
		return false
	}

	// Build maps for comparison (order shouldn't matter)
	oldMap := make(map[schema.ConstraintType]schema.Constraint)
	for _, c := range old {
		oldMap[c.Type] = c
	}

	newMap := make(map[schema.ConstraintType]schema.Constraint)
	for _, c := range new {
		newMap[c.Type] = c
	}

	// Check that all old constraints exist in new with same values
	for cType, oldConstraint := range oldMap {
		newConstraint, exists := newMap[cType]
		if !exists {
			return false
		}

		// Compare constraint values
		if !d.constraintValuesEqual(&oldConstraint, &newConstraint) {
			return false
		}
	}

	// Check that all new constraints exist in old (catches additions)
	for cType := range newMap {
		if _, exists := oldMap[cType]; !exists {
			return false
		}
	}

	return true
}

// constraintValuesEqual checks if two constraint values are equal
func (d *Differ) constraintValuesEqual(old, new *schema.Constraint) bool {
	if old.Type != new.Type {
		return false
	}

	// Handle nil values
	if old.Value == nil && new.Value == nil {
		return true
	}
	if old.Value == nil || new.Value == nil {
		return false
	}

	// Type-specific comparisons
	switch old.Type {
	case schema.ConstraintMin, schema.ConstraintMax:
		// Numeric constraints
		oldVal, oldOk := old.Value.(int)
		newVal, newOk := new.Value.(int)
		if !oldOk || !newOk {
			// Try float conversion
			oldFloat, oldFloatOk := old.Value.(float64)
			newFloat, newFloatOk := new.Value.(float64)
			if oldFloatOk && newFloatOk {
				return oldFloat == newFloat
			}
			return false
		}
		return oldVal == newVal

	case schema.ConstraintPattern:
		// String pattern constraints
		oldStr, oldOk := old.Value.(string)
		newStr, newOk := new.Value.(string)
		if !oldOk || !newOk {
			return false
		}
		return oldStr == newStr

	case schema.ConstraintUnique, schema.ConstraintPrimary, schema.ConstraintAuto:
		// Boolean-like constraints (presence matters, not value)
		return true

	default:
		// Generic equality check
		return old.Value == new.Value
	}
}

// typeSpecsEqual checks if two type specs are equal
func (d *Differ) typeSpecsEqual(old, new *schema.TypeSpec) bool {
	if old.BaseType != new.BaseType {
		return false
	}

	if old.Nullable != new.Nullable {
		return false
	}

	// Check length parameters
	if !intPtrEqual(old.Length, new.Length) {
		return false
	}

	if !intPtrEqual(old.Precision, new.Precision) {
		return false
	}

	if !intPtrEqual(old.Scale, new.Scale) {
		return false
	}

	return true
}

// relationshipsEqual checks if two relationships are equal
func (d *Differ) relationshipsEqual(old, new *schema.Relationship) bool {
	if old.Type != new.Type {
		return false
	}

	if old.TargetResource != new.TargetResource {
		return false
	}

	if old.ForeignKey != new.ForeignKey {
		return false
	}

	if old.OnDelete != new.OnDelete || old.OnUpdate != new.OnUpdate {
		return false
	}

	return true
}

// isBreakingFieldChange determines if a field change is breaking
func (d *Differ) isBreakingFieldChange(old, new *schema.Field) bool {
	// Nullability change: optional -> required
	if old.Type.Nullable && !new.Type.Nullable {
		return true
	}

	// Type change
	if old.Type.BaseType != new.Type.BaseType {
		return true
	}

	// Stricter constraints
	if d.hasStricterConstraints(old, new) {
		return true
	}

	return false
}

// causesDataLoss determines if a field change may cause data loss
func (d *Differ) causesDataLoss(old, new *schema.Field) bool {
	// Type change may cause data loss
	if old.Type.BaseType != new.Type.BaseType {
		return true
	}

	// String length reduction
	if old.Type.BaseType == schema.TypeString {
		oldLen := d.getLength(old.Type)
		newLen := d.getLength(new.Type)
		if newLen > 0 && (oldLen == 0 || newLen < oldLen) {
			return true
		}
	}

	// Precision/scale reduction for decimal
	if old.Type.BaseType == schema.TypeDecimal {
		oldPrec := d.getPrecision(old.Type)
		newPrec := d.getPrecision(new.Type)
		if newPrec > 0 && (oldPrec == 0 || newPrec < oldPrec) {
			return true
		}
	}

	return false
}

// isBreakingRelationshipChange determines if a relationship change is breaking
func (d *Differ) isBreakingRelationshipChange(old, new *schema.Relationship) bool {
	// Type change
	if old.Type != new.Type {
		return true
	}

	// Target resource change
	if old.TargetResource != new.TargetResource {
		return true
	}

	// Cascade behavior change
	if old.OnDelete != new.OnDelete || old.OnUpdate != new.OnUpdate {
		return true
	}

	return false
}

// hasStricterConstraints checks if new field has stricter constraints
func (d *Differ) hasStricterConstraints(old, new *schema.Field) bool {
	// Check for new unique constraint
	hasOldUnique := d.hasConstraintType(old, schema.ConstraintUnique)
	hasNewUnique := d.hasConstraintType(new, schema.ConstraintUnique)
	if !hasOldUnique && hasNewUnique {
		return true
	}

	// Check for stricter min/max constraints
	oldMin := d.getMinConstraint(old)
	newMin := d.getMinConstraint(new)
	if newMin > oldMin {
		return true
	}

	oldMax := d.getMaxConstraint(old)
	newMax := d.getMaxConstraint(new)
	if newMax > 0 && (oldMax == 0 || newMax < oldMax) {
		return true
	}

	return false
}

// Helper functions

func (d *Differ) hasConstraintType(field *schema.Field, cType schema.ConstraintType) bool {
	for i := range field.Constraints {
		if field.Constraints[i].Type == cType {
			return true
		}
	}
	return false
}

func (d *Differ) getMinConstraint(field *schema.Field) int {
	for i := range field.Constraints {
		if field.Constraints[i].Type == schema.ConstraintMin {
			if val, ok := field.Constraints[i].Value.(int); ok {
				return val
			}
		}
	}
	return 0
}

func (d *Differ) getMaxConstraint(field *schema.Field) int {
	for i := range field.Constraints {
		if field.Constraints[i].Type == schema.ConstraintMax {
			if val, ok := field.Constraints[i].Value.(int); ok {
				return val
			}
		}
	}
	return 0
}

func (d *Differ) getLength(t *schema.TypeSpec) int {
	if t.Length != nil {
		return *t.Length
	}
	return 0
}

func (d *Differ) getPrecision(t *schema.TypeSpec) int {
	if t.Precision != nil {
		return *t.Precision
	}
	return 0
}

// Set operations
func setDifference(a, b []string) []string {
	mb := make(map[string]bool)
	for _, x := range b {
		mb[x] = true
	}

	var diff []string
	for _, x := range a {
		if !mb[x] {
			diff = append(diff, x)
		}
	}
	return diff
}

func setIntersection(a, b []string) []string {
	mb := make(map[string]bool)
	for _, x := range b {
		mb[x] = true
	}

	var inter []string
	for _, x := range a {
		if mb[x] {
			inter = append(inter, x)
		}
	}
	return inter
}

func getSortedResourceNames(schemas map[string]*schema.ResourceSchema) []string {
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func getSortedFieldNames(fields map[string]*schema.Field) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func getSortedRelationshipNames(rels map[string]*schema.Relationship) []string {
	names := make([]string, 0, len(rels))
	for name := range rels {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// GenerateMigrationName creates a descriptive name for the migration
func GenerateMigrationName(changes []SchemaChange) string {
	if len(changes) == 0 {
		return "no_changes"
	}

	// Categorize changes
	var added, dropped, modified []string
	var resources, fields int

	for _, change := range changes {
		switch change.Type {
		case ChangeAddResource:
			added = append(added, fmt.Sprintf("resource_%s", change.Resource))
			resources++
		case ChangeDropResource:
			dropped = append(dropped, fmt.Sprintf("resource_%s", change.Resource))
			resources++
		case ChangeAddField:
			added = append(added, fmt.Sprintf("%s.%s", change.Resource, change.Field))
			fields++
		case ChangeDropField:
			dropped = append(dropped, fmt.Sprintf("%s.%s", change.Resource, change.Field))
			fields++
		case ChangeModifyField:
			modified = append(modified, fmt.Sprintf("%s.%s", change.Resource, change.Field))
			fields++
		}
	}

	// Build name components
	var parts []string

	if len(added) > 0 {
		if len(added) <= 3 {
			parts = append(parts, "add_"+strings.Join(added, "_"))
		} else {
			parts = append(parts, fmt.Sprintf("add_%d_items", len(added)))
		}
	}

	if len(dropped) > 0 {
		if len(dropped) <= 3 {
			parts = append(parts, "drop_"+strings.Join(dropped, "_"))
		} else {
			parts = append(parts, fmt.Sprintf("drop_%d_items", len(dropped)))
		}
	}

	if len(modified) > 0 {
		if len(modified) <= 3 {
			parts = append(parts, "modify_"+strings.Join(modified, "_"))
		} else {
			parts = append(parts, fmt.Sprintf("modify_%d_fields", len(modified)))
		}
	}

	if len(parts) == 0 {
		return "schema_changes"
	}

	name := strings.Join(parts, "_and_")

	// Limit name length
	if len(name) > 200 {
		return fmt.Sprintf("schema_changes_%d_resources_%d_fields", resources, fields)
	}

	return name
}
