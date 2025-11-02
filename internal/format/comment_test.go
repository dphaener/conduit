package format

import (
	"strings"
	"testing"
)

func TestFormatterPreservesCommentsBeforeFields(t *testing.T) {
	input := `resource User {
# This is the primary key
id: uuid! @primary @auto
# User's full name
name: string!
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	// Comments should be preserved
	if !strings.Contains(result, "# This is the primary key") {
		t.Errorf("Comment before field not preserved.\nGot:\n%s", result)
	}
	if !strings.Contains(result, "# User's full name") {
		t.Errorf("Comment before field not preserved.\nGot:\n%s", result)
	}
}

func TestFormatterPreservesCommentsAfterFields(t *testing.T) {
	input := `resource User {
id: uuid! @primary @auto # Primary identifier
name: string! # Required field
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	// Comments should be preserved
	if !strings.Contains(result, "# Primary identifier") {
		t.Errorf("Comment after field not preserved.\nGot:\n%s", result)
	}
	if !strings.Contains(result, "# Required field") {
		t.Errorf("Comment after field not preserved.\nGot:\n%s", result)
	}
}

func TestFormatterPreservesCommentsInResourceBody(t *testing.T) {
	input := `resource User {
# Primary fields
id: uuid! @primary @auto
name: string!

# Contact information
email: string! @unique
phone: string?
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	// Section comments should be preserved
	if !strings.Contains(result, "# Primary fields") {
		t.Errorf("Section comment not preserved.\nGot:\n%s", result)
	}
	if !strings.Contains(result, "# Contact information") {
		t.Errorf("Section comment not preserved.\nGot:\n%s", result)
	}
}

func TestFormatterPreservesCommentsBeforeResource(t *testing.T) {
	input := `# User account resource
resource User {
id: uuid! @primary @auto
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	// Top-level comments should be preserved
	if !strings.Contains(result, "# User account resource") {
		t.Errorf("Comment before resource not preserved.\nGot:\n%s", result)
	}
}

func TestFormatterPreservesCommentsInHookBodies(t *testing.T) {
	input := `resource User {
id: uuid! @primary @auto

@before create {
    # Validate the user
    validate()
    # Hash the password
    hashPassword()
}
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	// Comments in hook bodies should be preserved (these are raw text)
	if !strings.Contains(result, "# Validate the user") {
		t.Errorf("Comment in hook body not preserved.\nGot:\n%s", result)
	}
	if !strings.Contains(result, "# Hash the password") {
		t.Errorf("Comment in hook body not preserved.\nGot:\n%s", result)
	}
}
