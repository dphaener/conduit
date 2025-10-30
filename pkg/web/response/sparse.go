package response

import (
	"encoding/json"
	"fmt"
)

// ApplySparseFieldsets filters the attributes of resources in a JSON:API document
// based on the provided fieldsets map. The fieldsets map uses resource types as keys
// and arrays of field names as values.
//
// JSON:API spec requires that id and type fields are always included, regardless of
// sparse fieldsets. This function preserves those fields along with relationships.
//
// Per JSON:API spec, requesting non-existent fields is not an error. Fields that
// don't exist in the resource are silently ignored. This allows clients to request
// a common set of fields across different resource types.
//
// Example usage in a handler:
//
//	func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
//	    // Parse sparse fieldsets from query params
//	    fieldsets := query.ParseFields(r)  // e.g., ?fields[users]=name,email
//
//	    // Render JSON:API response
//	    jsonData, _ := jsonapi.Marshal(users)
//
//	    // Apply sparse fieldsets post-processing
//	    filtered, err := response.ApplySparseFieldsets(jsonData, fieldsets)
//	    if err != nil {
//	        response.RenderInternalError(w, r, err)
//	        return
//	    }
//
//	    w.Header().Set("Content-Type", "application/vnd.api+json")
//	    w.Write(filtered)
//	}
//
// Parameters:
//   - jsonData: The JSON:API document as bytes
//   - fieldsets: Map of resource type to array of field names to include
//
// Returns the filtered JSON document, or the original document if fieldsets is empty/nil.
func ApplySparseFieldsets(jsonData []byte, fieldsets map[string][]string) ([]byte, error) {
	// Return original data if no fieldsets specified
	if len(fieldsets) == 0 {
		return jsonData, nil
	}

	// Parse JSON into a map
	var doc map[string]interface{}
	if err := json.Unmarshal(jsonData, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse JSON:API document: %w", err)
	}

	// Process the main data
	if data, exists := doc["data"]; exists && data != nil {
		switch v := data.(type) {
		case map[string]interface{}:
			// Single resource
			filterResource(v, fieldsets)
		case []interface{}:
			// Collection of resources
			for _, resource := range v {
				if res, ok := resource.(map[string]interface{}); ok {
					filterResource(res, fieldsets)
				}
			}
		}
	}

	// Process included resources
	if included, exists := doc["included"]; exists {
		if includedArray, ok := included.([]interface{}); ok {
			for _, resource := range includedArray {
				if res, ok := resource.(map[string]interface{}); ok {
					filterResource(res, fieldsets)
				}
			}
		}
	}

	// Marshal back to JSON
	filtered, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filtered document: %w", err)
	}

	return filtered, nil
}

// filterResource filters the attributes of a single resource object based on fieldsets.
// It preserves id, type, and relationships fields as required by JSON:API spec.
func filterResource(resource map[string]interface{}, fieldsets map[string][]string) {
	// Get the resource type
	resourceType, ok := resource["type"].(string)
	if !ok {
		return
	}

	// Check if we have fieldsets for this resource type
	fields, hasFieldsets := fieldsets[resourceType]
	if !hasFieldsets {
		// No fieldsets for this type, keep all attributes
		return
	}

	// Get the attributes object
	attributes, exists := resource["attributes"]
	if !exists {
		return
	}

	attrs, ok := attributes.(map[string]interface{})
	if !ok {
		return
	}

	// Build a set of allowed fields for O(1) lookup
	allowedFields := make(map[string]bool, len(fields))
	for _, field := range fields {
		allowedFields[field] = true
	}

	// Filter attributes to only include allowed fields
	filtered := make(map[string]interface{})
	for key, value := range attrs {
		if allowedFields[key] {
			filtered[key] = value
		}
	}

	// Replace attributes with filtered version
	resource["attributes"] = filtered
}
