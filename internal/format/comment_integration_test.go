package format

import (
	"testing"
)

func TestFormatterPreservesAllCommentStyles(t *testing.T) {
	input := `# Main user resource for the application
resource User {
# Primary identifier
id: uuid! @primary @auto

# User profile information
name: string! # Full legal name
email: string! @unique # Must be unique across all users
bio: text? # Optional biography

# Account metadata
created_at: timestamp! @auto
updated_at: timestamp! @auto_update

# Relationships
# Link to the user's profile
profile: Profile?

@before create {
    # Validate email format
    validateEmail()

    # Hash password
    hashPassword()
}

@constraint valid_age {
    # User must be at least 18 years old
    self.age >= 18
}
}

# User profile resource
resource Profile {
id: uuid! @primary @auto
user: User!
avatar_url: string?
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	// Verify specific comments are preserved
	expectedComments := []string{
		"# Main user resource for the application",
		"# Primary identifier",
		"# User profile information",
		"# Full legal name",
		"# Must be unique across all users",
		"# Optional biography",
		"# Account metadata",
		"# Relationships",
		"# Link to the user's profile",
		"# Validate email format",
		"# Hash password",
		"# User must be at least 18 years old",
		"# User profile resource",
	}

	for _, comment := range expectedComments {
		if !containsComment(result, comment) {
			t.Errorf("Expected comment not found: %s\nResult:\n%s", comment, result)
		}
	}
}

func TestFormatterIdempotentWithComments(t *testing.T) {
	input := `# Main resource
resource User {
# ID field
id: uuid! @primary @auto
# Name field
name: string! # Required
}
`

	config := DefaultConfig()
	formatter := New(config)

	// Format once
	result1, err := formatter.Format(input)
	if err != nil {
		t.Fatalf("First format failed: %v", err)
	}

	// Format again
	result2, err := formatter.Format(result1)
	if err != nil {
		t.Fatalf("Second format failed: %v", err)
	}

	// Format third time
	result3, err := formatter.Format(result2)
	if err != nil {
		t.Fatalf("Third format failed: %v", err)
	}

	// All results should be identical
	if result1 != result2 {
		t.Errorf("First and second format differ:\nFirst:\n%s\nSecond:\n%s", result1, result2)
	}

	if result2 != result3 {
		t.Errorf("Second and third format differ:\nSecond:\n%s\nThird:\n%s", result2, result3)
	}

	// Verify comments are still present after multiple formats
	if !containsComment(result3, "# Main resource") {
		t.Errorf("Comment lost after multiple formats")
	}
	if !containsComment(result3, "# ID field") {
		t.Errorf("Comment lost after multiple formats")
	}
	if !containsComment(result3, "# Name field") {
		t.Errorf("Comment lost after multiple formats")
	}
	if !containsComment(result3, "# Required") {
		t.Errorf("Trailing comment lost after multiple formats")
	}
}

func containsComment(text, comment string) bool {
	// Simple substring check for comments
	return contains(text, comment)
}

func contains(text, substr string) bool {
	// Simple substring search
	for i := 0; i <= len(text)-len(substr); i++ {
		if text[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
