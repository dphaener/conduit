package response

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplySparseFieldsets_EmptyFieldsets(t *testing.T) {
	original := []byte(`{
		"data": {
			"type": "users",
			"id": "1",
			"attributes": {
				"name": "John Doe",
				"email": "john@example.com",
				"age": 30
			}
		}
	}`)

	// Empty fieldsets should return original data
	result, err := ApplySparseFieldsets(original, map[string][]string{})
	require.NoError(t, err)
	assert.JSONEq(t, string(original), string(result))

	// Nil fieldsets should return original data
	result, err = ApplySparseFieldsets(original, nil)
	require.NoError(t, err)
	assert.JSONEq(t, string(original), string(result))
}

func TestApplySparseFieldsets_SingleResource(t *testing.T) {
	original := []byte(`{
		"data": {
			"type": "users",
			"id": "1",
			"attributes": {
				"name": "John Doe",
				"email": "john@example.com",
				"age": 30,
				"bio": "Software developer"
			}
		}
	}`)

	fieldsets := map[string][]string{
		"users": {"name", "email"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	data := resultDoc["data"].(map[string]interface{})
	attrs := data["attributes"].(map[string]interface{})

	// Should only have name and email
	assert.Len(t, attrs, 2)
	assert.Equal(t, "John Doe", attrs["name"])
	assert.Equal(t, "john@example.com", attrs["email"])
	assert.NotContains(t, attrs, "age")
	assert.NotContains(t, attrs, "bio")

	// Should preserve id and type
	assert.Equal(t, "1", data["id"])
	assert.Equal(t, "users", data["type"])
}

func TestApplySparseFieldsets_Collection(t *testing.T) {
	original := []byte(`{
		"data": [
			{
				"type": "users",
				"id": "1",
				"attributes": {
					"name": "John Doe",
					"email": "john@example.com",
					"age": 30
				}
			},
			{
				"type": "users",
				"id": "2",
				"attributes": {
					"name": "Jane Smith",
					"email": "jane@example.com",
					"age": 25
				}
			}
		]
	}`)

	fieldsets := map[string][]string{
		"users": {"name"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	data := resultDoc["data"].([]interface{})
	require.Len(t, data, 2)

	// Check first resource
	res1 := data[0].(map[string]interface{})
	attrs1 := res1["attributes"].(map[string]interface{})
	assert.Len(t, attrs1, 1)
	assert.Equal(t, "John Doe", attrs1["name"])
	assert.NotContains(t, attrs1, "email")
	assert.NotContains(t, attrs1, "age")

	// Check second resource
	res2 := data[1].(map[string]interface{})
	attrs2 := res2["attributes"].(map[string]interface{})
	assert.Len(t, attrs2, 1)
	assert.Equal(t, "Jane Smith", attrs2["name"])
	assert.NotContains(t, attrs2, "email")
	assert.NotContains(t, attrs2, "age")
}

func TestApplySparseFieldsets_WithIncluded(t *testing.T) {
	original := []byte(`{
		"data": {
			"type": "articles",
			"id": "1",
			"attributes": {
				"title": "JSON:API paints my bikeshed!",
				"body": "The shortest article. Ever.",
				"created": "2015-05-22T14:56:29.000Z"
			},
			"relationships": {
				"author": {
					"data": { "type": "users", "id": "42" }
				}
			}
		},
		"included": [
			{
				"type": "users",
				"id": "42",
				"attributes": {
					"name": "John Doe",
					"email": "john@example.com",
					"twitter": "@johndoe",
					"age": 30
				}
			}
		]
	}`)

	fieldsets := map[string][]string{
		"articles": {"title"},
		"users":    {"name", "email"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	// Check main resource
	data := resultDoc["data"].(map[string]interface{})
	attrs := data["attributes"].(map[string]interface{})
	assert.Len(t, attrs, 1)
	assert.Equal(t, "JSON:API paints my bikeshed!", attrs["title"])
	assert.NotContains(t, attrs, "body")
	assert.NotContains(t, attrs, "created")

	// Relationships should be preserved
	assert.Contains(t, data, "relationships")

	// Check included resource
	included := resultDoc["included"].([]interface{})
	require.Len(t, included, 1)

	includedUser := included[0].(map[string]interface{})
	includedAttrs := includedUser["attributes"].(map[string]interface{})
	assert.Len(t, includedAttrs, 2)
	assert.Equal(t, "John Doe", includedAttrs["name"])
	assert.Equal(t, "john@example.com", includedAttrs["email"])
	assert.NotContains(t, includedAttrs, "twitter")
	assert.NotContains(t, includedAttrs, "age")
}

func TestApplySparseFieldsets_MultipleResourceTypes(t *testing.T) {
	original := []byte(`{
		"data": [
			{
				"type": "users",
				"id": "1",
				"attributes": {
					"name": "John Doe",
					"email": "john@example.com",
					"age": 30
				}
			},
			{
				"type": "articles",
				"id": "1",
				"attributes": {
					"title": "Test Article",
					"body": "Article body",
					"created": "2015-05-22T14:56:29.000Z"
				}
			}
		]
	}`)

	fieldsets := map[string][]string{
		"users":    {"name"},
		"articles": {"title", "created"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	data := resultDoc["data"].([]interface{})

	// Check user resource
	user := data[0].(map[string]interface{})
	userAttrs := user["attributes"].(map[string]interface{})
	assert.Len(t, userAttrs, 1)
	assert.Equal(t, "John Doe", userAttrs["name"])

	// Check article resource
	article := data[1].(map[string]interface{})
	articleAttrs := article["attributes"].(map[string]interface{})
	assert.Len(t, articleAttrs, 2)
	assert.Equal(t, "Test Article", articleAttrs["title"])
	assert.Equal(t, "2015-05-22T14:56:29.000Z", articleAttrs["created"])
	assert.NotContains(t, articleAttrs, "body")
}

func TestApplySparseFieldsets_PreservesIdAndType(t *testing.T) {
	original := []byte(`{
		"data": {
			"type": "users",
			"id": "123",
			"attributes": {
				"name": "John Doe",
				"email": "john@example.com"
			}
		}
	}`)

	fieldsets := map[string][]string{
		"users": {"name"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	data := resultDoc["data"].(map[string]interface{})

	// ID and type should always be present
	assert.Equal(t, "123", data["id"])
	assert.Equal(t, "users", data["type"])

	// Only name attribute should be present
	attrs := data["attributes"].(map[string]interface{})
	assert.Len(t, attrs, 1)
	assert.Equal(t, "John Doe", attrs["name"])
}

func TestApplySparseFieldsets_ResourceTypeNotInFieldsets(t *testing.T) {
	original := []byte(`{
		"data": {
			"type": "comments",
			"id": "1",
			"attributes": {
				"body": "Great article!",
				"created": "2015-05-22T14:56:29.000Z"
			}
		}
	}`)

	// Fieldsets for a different resource type
	fieldsets := map[string][]string{
		"users": {"name", "email"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	// Should keep all attributes when resource type not in fieldsets
	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	data := resultDoc["data"].(map[string]interface{})
	attrs := data["attributes"].(map[string]interface{})
	assert.Len(t, attrs, 2)
	assert.Equal(t, "Great article!", attrs["body"])
	assert.Equal(t, "2015-05-22T14:56:29.000Z", attrs["created"])
}

func TestApplySparseFieldsets_FieldNotInResource(t *testing.T) {
	original := []byte(`{
		"data": {
			"type": "users",
			"id": "1",
			"attributes": {
				"name": "John Doe",
				"email": "john@example.com"
			}
		}
	}`)

	// Request fields that don't exist in the resource
	fieldsets := map[string][]string{
		"users": {"name", "age", "twitter"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	data := resultDoc["data"].(map[string]interface{})
	attrs := data["attributes"].(map[string]interface{})

	// Should only include name (the only field that exists and is requested)
	assert.Len(t, attrs, 1)
	assert.Equal(t, "John Doe", attrs["name"])
	assert.NotContains(t, attrs, "email")
	assert.NotContains(t, attrs, "age")
	assert.NotContains(t, attrs, "twitter")
}

func TestApplySparseFieldsets_EmptyAttributes(t *testing.T) {
	original := []byte(`{
		"data": {
			"type": "users",
			"id": "1",
			"attributes": {}
		}
	}`)

	fieldsets := map[string][]string{
		"users": {"name", "email"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	data := resultDoc["data"].(map[string]interface{})
	attrs := data["attributes"].(map[string]interface{})
	assert.Empty(t, attrs)
}

func TestApplySparseFieldsets_NoAttributes(t *testing.T) {
	original := []byte(`{
		"data": {
			"type": "users",
			"id": "1"
		}
	}`)

	fieldsets := map[string][]string{
		"users": {"name", "email"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	data := resultDoc["data"].(map[string]interface{})
	assert.NotContains(t, data, "attributes")
}

func TestApplySparseFieldsets_NullValues(t *testing.T) {
	original := []byte(`{
		"data": {
			"type": "users",
			"id": "1",
			"attributes": {
				"name": "John Doe",
				"email": "john@example.com",
				"bio": null,
				"avatar": null
			}
		}
	}`)

	fieldsets := map[string][]string{
		"users": {"name", "bio"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	data := resultDoc["data"].(map[string]interface{})
	attrs := data["attributes"].(map[string]interface{})

	assert.Len(t, attrs, 2)
	assert.Equal(t, "John Doe", attrs["name"])
	assert.Nil(t, attrs["bio"])
	assert.NotContains(t, attrs, "email")
	assert.NotContains(t, attrs, "avatar")
}

func TestApplySparseFieldsets_MalformedJSON(t *testing.T) {
	malformed := []byte(`{invalid json}`)

	fieldsets := map[string][]string{
		"users": {"name"},
	}

	_, err := ApplySparseFieldsets(malformed, fieldsets)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON:API document")
}

func TestApplySparseFieldsets_WithRelationships(t *testing.T) {
	original := []byte(`{
		"data": {
			"type": "articles",
			"id": "1",
			"attributes": {
				"title": "JSON:API paints my bikeshed!",
				"body": "The shortest article. Ever.",
				"created": "2015-05-22T14:56:29.000Z"
			},
			"relationships": {
				"author": {
					"data": { "type": "users", "id": "42" }
				},
				"comments": {
					"data": [
						{ "type": "comments", "id": "5" },
						{ "type": "comments", "id": "12" }
					]
				}
			}
		}
	}`)

	fieldsets := map[string][]string{
		"articles": {"title"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	data := resultDoc["data"].(map[string]interface{})

	// Attributes should be filtered
	attrs := data["attributes"].(map[string]interface{})
	assert.Len(t, attrs, 1)
	assert.Equal(t, "JSON:API paints my bikeshed!", attrs["title"])

	// Relationships should be preserved unchanged
	relationships := data["relationships"].(map[string]interface{})
	assert.Contains(t, relationships, "author")
	assert.Contains(t, relationships, "comments")
}

func TestApplySparseFieldsets_NullData(t *testing.T) {
	original := []byte(`{
		"data": null
	}`)

	fieldsets := map[string][]string{
		"users": {"name"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	assert.Nil(t, resultDoc["data"])
}

func TestApplySparseFieldsets_EmptyCollection(t *testing.T) {
	original := []byte(`{
		"data": []
	}`)

	fieldsets := map[string][]string{
		"users": {"name"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	data := resultDoc["data"].([]interface{})
	assert.Empty(t, data)
}

func TestApplySparseFieldsets_PreservesMetaAndLinks(t *testing.T) {
	original := []byte(`{
		"data": {
			"type": "users",
			"id": "1",
			"attributes": {
				"name": "John Doe",
				"email": "john@example.com"
			}
		},
		"meta": {
			"total": 1
		},
		"links": {
			"self": "http://example.com/users/1"
		}
	}`)

	fieldsets := map[string][]string{
		"users": {"name"},
	}

	result, err := ApplySparseFieldsets(original, fieldsets)
	require.NoError(t, err)

	var resultDoc map[string]interface{}
	err = json.Unmarshal(result, &resultDoc)
	require.NoError(t, err)

	// Meta and links should be preserved
	assert.Contains(t, resultDoc, "meta")
	assert.Contains(t, resultDoc, "links")

	meta := resultDoc["meta"].(map[string]interface{})
	assert.Equal(t, float64(1), meta["total"])

	links := resultDoc["links"].(map[string]interface{})
	assert.Equal(t, "http://example.com/users/1", links["self"])
}
