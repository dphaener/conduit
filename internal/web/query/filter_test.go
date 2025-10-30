package query

import (
	"strings"
	"testing"
)

func TestBuildFilterClause_EmptyFilters(t *testing.T) {
	filters := map[string]string{}
	validFields := []string{"status", "author_id"}

	whereClause, args, err := BuildFilterClause(filters, "posts", validFields)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if whereClause != "" {
		t.Errorf("Expected empty WHERE clause, got: %s", whereClause)
	}
	if args != nil {
		t.Errorf("Expected nil args, got: %v", args)
	}
}

func TestBuildFilterClause_SingleFilter(t *testing.T) {
	filters := map[string]string{
		"status": "published",
	}
	validFields := []string{"status", "author_id"}

	whereClause, args, err := BuildFilterClause(filters, "posts", validFields)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	expectedClause := "WHERE posts.status = $1"
	if whereClause != expectedClause {
		t.Errorf("Expected clause %q, got %q", expectedClause, whereClause)
	}

	if len(args) != 1 {
		t.Fatalf("Expected 1 arg, got %d", len(args))
	}
	if args[0] != "published" {
		t.Errorf("Expected arg 'published', got %v", args[0])
	}
}

func TestBuildFilterClause_MultipleFilters(t *testing.T) {
	filters := map[string]string{
		"status":    "published",
		"author_id": "123",
	}
	validFields := []string{"status", "author_id"}

	whereClause, args, err := BuildFilterClause(filters, "posts", validFields)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Since we sort keys, order should be deterministic (author_id before status)
	expectedClause := "WHERE posts.author_id = $1 AND posts.status = $2"
	if whereClause != expectedClause {
		t.Errorf("Expected clause %q, got %q", expectedClause, whereClause)
	}

	if len(args) != 2 {
		t.Fatalf("Expected 2 args, got %d", len(args))
	}

	// Check args match the sorted order
	if args[0] != "123" {
		t.Errorf("Expected first arg '123', got %v", args[0])
	}
	if args[1] != "published" {
		t.Errorf("Expected second arg 'published', got %v", args[1])
	}
}

func TestBuildFilterClause_InvalidField(t *testing.T) {
	filters := map[string]string{
		"status":        "published",
		"invalid_field": "value",
	}
	validFields := []string{"status", "author_id"}

	whereClause, args, err := BuildFilterClause(filters, "posts", validFields)

	if err == nil {
		t.Fatal("Expected error for invalid field, got nil")
	}

	expectedErrMsg := "invalid filter fields: invalid_field"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error containing %q, got: %v", expectedErrMsg, err)
	}

	if whereClause != "" {
		t.Errorf("Expected empty WHERE clause on error, got: %s", whereClause)
	}
	if args != nil {
		t.Errorf("Expected nil args on error, got: %v", args)
	}
}

func TestBuildFilterClause_MultipleInvalidFields(t *testing.T) {
	filters := map[string]string{
		"status":  "published",
		"invalid": "value",
		"bad":     "data",
	}
	validFields := []string{"status"}

	_, _, err := BuildFilterClause(filters, "posts", validFields)

	if err == nil {
		t.Fatal("Expected error for invalid fields, got nil")
	}

	// Should list both invalid fields
	errMsg := err.Error()
	if !strings.Contains(errMsg, "bad") || !strings.Contains(errMsg, "invalid") {
		t.Errorf("Expected error to list both invalid fields, got: %v", err)
	}
}

func TestBuildFilterClause_SQLInjectionPrevention(t *testing.T) {
	// Attempt SQL injection through filter values
	filters := map[string]string{
		"status": "published'; DROP TABLE posts; --",
	}
	validFields := []string{"status"}

	whereClause, args, err := BuildFilterClause(filters, "posts", validFields)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify the malicious value is treated as a parameter, not inline SQL
	expectedClause := "WHERE posts.status = $1"
	if whereClause != expectedClause {
		t.Errorf("Expected clause %q, got %q", expectedClause, whereClause)
	}

	// The dangerous string should be in args (safe parameterization)
	if len(args) != 1 {
		t.Fatalf("Expected 1 arg, got %d", len(args))
	}
	if args[0] != "published'; DROP TABLE posts; --" {
		t.Errorf("Expected dangerous string as parameter, got %v", args[0])
	}

	// Verify no inline SQL execution in WHERE clause
	if strings.Contains(whereClause, "DROP") {
		t.Error("WHERE clause contains dangerous SQL - parameterization failed!")
	}
}

func TestBuildFilterClause_CamelCaseToSnakeCase(t *testing.T) {
	filters := map[string]string{
		"authorId": "123",
	}
	// Valid fields should be in snake_case (database column format)
	validFields := []string{"author_id"}

	whereClause, args, err := BuildFilterClause(filters, "posts", validFields)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should convert authorId to author_id
	expectedClause := "WHERE posts.author_id = $1"
	if whereClause != expectedClause {
		t.Errorf("Expected clause %q, got %q", expectedClause, whereClause)
	}

	if len(args) != 1 || args[0] != "123" {
		t.Errorf("Expected args [123], got %v", args)
	}
}

func TestBuildFilterClause_TableNamePrefix(t *testing.T) {
	filters := map[string]string{
		"status": "draft",
	}
	validFields := []string{"status"}

	whereClause, args, err := BuildFilterClause(filters, "articles", validFields)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Should use 'articles' as table prefix
	if !strings.Contains(whereClause, "articles.status") {
		t.Errorf("Expected table prefix 'articles', got: %s", whereClause)
	}

	if len(args) != 1 || args[0] != "draft" {
		t.Errorf("Expected args [draft], got %v", args)
	}
}

func TestValidateFilterFields_AllValid(t *testing.T) {
	filters := map[string]string{
		"status":    "published",
		"author_id": "123",
	}
	validFields := []string{"status", "author_id", "title"}

	err := ValidateFilterFields(filters, validFields)

	if err != nil {
		t.Errorf("Expected no error for valid fields, got: %v", err)
	}
}

func TestValidateFilterFields_SomeInvalid(t *testing.T) {
	filters := map[string]string{
		"status":  "published",
		"invalid": "value",
	}
	validFields := []string{"status", "author_id"}

	err := ValidateFilterFields(filters, validFields)

	if err == nil {
		t.Fatal("Expected error for invalid field, got nil")
	}

	if !strings.Contains(err.Error(), "invalid") {
		t.Errorf("Expected error to mention 'invalid', got: %v", err)
	}
}

func TestValidateFilterFields_EmptyFilters(t *testing.T) {
	filters := map[string]string{}
	validFields := []string{"status", "author_id"}

	err := ValidateFilterFields(filters, validFields)

	if err != nil {
		t.Errorf("Expected no error for empty filters, got: %v", err)
	}
}

func TestValidateFilterFields_EmptyValidFields(t *testing.T) {
	filters := map[string]string{
		"status": "published",
	}
	validFields := []string{}

	err := ValidateFilterFields(filters, validFields)

	if err == nil {
		t.Fatal("Expected error when no valid fields defined, got nil")
	}
}

func TestBuildFilterClause_ParameterIndexing(t *testing.T) {
	// Test that parameters increment correctly
	filters := map[string]string{
		"status":    "published",
		"author_id": "123",
		"category":  "tech",
	}
	validFields := []string{"status", "author_id", "category"}

	whereClause, args, err := BuildFilterClause(filters, "posts", validFields)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check for $1, $2, $3 in sorted order
	if !strings.Contains(whereClause, "$1") {
		t.Error("Expected $1 parameter in WHERE clause")
	}
	if !strings.Contains(whereClause, "$2") {
		t.Error("Expected $2 parameter in WHERE clause")
	}
	if !strings.Contains(whereClause, "$3") {
		t.Error("Expected $3 parameter in WHERE clause")
	}

	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}
}
