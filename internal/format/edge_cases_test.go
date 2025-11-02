package format

import (
	"strings"
	"testing"
)

func TestEdgeCases_EmptyComment(t *testing.T) {
	input := `resource User {
#
id: uuid! @primary
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "#") {
		t.Errorf("Empty comment not preserved.\nGot:\n%s", result)
	}
}

func TestEdgeCases_WhitespaceComment(t *testing.T) {
	input := `resource User {
#
id: uuid! @primary
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "#") {
		t.Errorf("Whitespace comment not preserved.\nGot:\n%s", result)
	}
}

func TestEdgeCases_ConsecutiveComments(t *testing.T) {
	input := `resource User {
# Comment 1
# Comment 2
# Comment 3
id: uuid! @primary
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "# Comment 1") {
		t.Errorf("First comment not preserved.\nGot:\n%s", result)
	}
	if !strings.Contains(result, "# Comment 2") {
		t.Errorf("Second comment not preserved.\nGot:\n%s", result)
	}
	if !strings.Contains(result, "# Comment 3") {
		t.Errorf("Third comment not preserved.\nGot:\n%s", result)
	}
}

func TestEdgeCases_MixedLeadingAndTrailing(t *testing.T) {
	input := `resource User {
# Leading
id: uuid! # Trailing
# Next leading
name: string!
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "# Leading") {
		t.Errorf("Leading comment not preserved.\nGot:\n%s", result)
	}
	if !strings.Contains(result, "# Trailing") {
		t.Errorf("Trailing comment not preserved.\nGot:\n%s", result)
	}
	if !strings.Contains(result, "# Next leading") {
		t.Errorf("Next leading comment not preserved.\nGot:\n%s", result)
	}
}

func TestEdgeCases_CommentBeforeRelationship(t *testing.T) {
	input := `resource User {
id: uuid! @primary

# User profile relationship
profile: Profile?
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "# User profile relationship") {
		t.Errorf("Relationship comment not preserved.\nGot:\n%s", result)
	}
}

func TestEdgeCases_TrailingCommentOnRelationship(t *testing.T) {
	input := `resource User {
id: uuid! @primary
profile: Profile? # Optional profile link
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "# Optional profile link") {
		t.Errorf("Trailing comment on relationship not preserved.\nGot:\n%s", result)
	}
}

func TestEdgeCases_CommentWithSpecialChars(t *testing.T) {
	input := `resource User {
# TODO: Fix this! @important (high priority)
id: uuid! @primary
}`

	config := DefaultConfig()
	formatter := New(config)
	result, err := formatter.Format(input)

	if err != nil {
		t.Fatalf("Formatting failed: %v", err)
	}

	if !strings.Contains(result, "# TODO: Fix this! @important (high priority)") {
		t.Errorf("Comment with special chars not preserved.\nGot:\n%s", result)
	}
}
