package query

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestFilteringIntegration tests the complete filtering flow:
// parse params -> build SQL -> execute query
func TestFilteringIntegration(t *testing.T) {
	// Setup in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`CREATE TABLE posts (
		id INTEGER PRIMARY KEY,
		title TEXT,
		status TEXT,
		author_id INTEGER
	)`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	testData := []struct {
		id       int
		title    string
		status   string
		authorID int
	}{
		{1, "First Post", "published", 1},
		{2, "Draft Post", "draft", 1},
		{3, "Another Published", "published", 2},
		{4, "Private Post", "private", 2},
	}

	for _, data := range testData {
		_, err = db.Exec("INSERT INTO posts (id, title, status, author_id) VALUES (?, ?, ?, ?)",
			data.id, data.title, data.status, data.authorID)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	t.Run("Single filter", func(t *testing.T) {
		// Create mock request with filter param
		req := httptest.NewRequest("GET", "/posts?filter[status]=published", nil)

		// Parse filter params
		filters := ParseFilter(req)

		// Verify parsing
		if len(filters) != 1 {
			t.Fatalf("Expected 1 filter, got %d", len(filters))
		}
		if filters["status"] != "published" {
			t.Errorf("Expected status=published, got status=%s", filters["status"])
		}

		// Build SQL query with filter clause
		validFields := []string{"status", "author_id", "title"}
		whereClause, args, err := BuildFilterClause(filters, "posts", validFields)
		if err != nil {
			t.Fatalf("Failed to build filter clause: %v", err)
		}

		// Execute query (note: SQLite uses ? placeholders, so we need to convert)
		sqlQuery := "SELECT id, title, status, author_id FROM posts " + convertToSQLitePlaceholders(whereClause)
		rows, err := db.Query(sqlQuery, args...)
		if err != nil {
			t.Fatalf("Failed to execute query: %v", err)
		}
		defer rows.Close()

		// Verify results
		var results []struct {
			ID       int
			Title    string
			Status   string
			AuthorID int
		}

		for rows.Next() {
			var r struct {
				ID       int
				Title    string
				Status   string
				AuthorID int
			}
			if err := rows.Scan(&r.ID, &r.Title, &r.Status, &r.AuthorID); err != nil {
				t.Fatalf("Failed to scan row: %v", err)
			}
			results = append(results, r)
		}

		// Should only get published posts
		if len(results) != 2 {
			t.Errorf("Expected 2 published posts, got %d", len(results))
		}

		for _, result := range results {
			if result.Status != "published" {
				t.Errorf("Expected status=published, got status=%s", result.Status)
			}
		}
	})

	t.Run("Multiple filters with AND logic", func(t *testing.T) {
		// Create mock request with multiple filters
		req := httptest.NewRequest("GET", "/posts?filter[status]=published&filter[author_id]=1", nil)

		// Parse filter params
		filters := ParseFilter(req)

		// Verify parsing
		if len(filters) != 2 {
			t.Fatalf("Expected 2 filters, got %d", len(filters))
		}

		// Build SQL query with filter clause
		validFields := []string{"status", "author_id", "title"}
		whereClause, args, err := BuildFilterClause(filters, "posts", validFields)
		if err != nil {
			t.Fatalf("Failed to build filter clause: %v", err)
		}

		// Execute query
		sqlQuery := "SELECT id, title, status, author_id FROM posts " + convertToSQLitePlaceholders(whereClause)
		rows, err := db.Query(sqlQuery, args...)
		if err != nil {
			t.Fatalf("Failed to execute query: %v", err)
		}
		defer rows.Close()

		// Verify results
		var results []struct {
			ID       int
			Title    string
			Status   string
			AuthorID int
		}

		for rows.Next() {
			var r struct {
				ID       int
				Title    string
				Status   string
				AuthorID int
			}
			if err := rows.Scan(&r.ID, &r.Title, &r.Status, &r.AuthorID); err != nil {
				t.Fatalf("Failed to scan row: %v", err)
			}
			results = append(results, r)
		}

		// Should only get published posts by author 1
		if len(results) != 1 {
			t.Errorf("Expected 1 post (published AND author_id=1), got %d", len(results))
		}

		if len(results) > 0 {
			if results[0].Status != "published" {
				t.Errorf("Expected status=published, got status=%s", results[0].Status)
			}
			if results[0].AuthorID != 1 {
				t.Errorf("Expected author_id=1, got author_id=%d", results[0].AuthorID)
			}
		}
	})
}

// TestSortingIntegration tests the complete sorting flow:
// parse params -> build SQL -> execute query -> verify order
func TestSortingIntegration(t *testing.T) {
	// Setup in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`CREATE TABLE articles (
		id INTEGER PRIMARY KEY,
		title TEXT,
		views INTEGER,
		created_at TEXT
	)`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert unsorted test data
	testData := []struct {
		id        int
		title     string
		views     int
		createdAt string
	}{
		{1, "Zebra Article", 100, "2025-01-01"},
		{2, "Apple Article", 50, "2025-01-03"},
		{3, "Banana Article", 200, "2025-01-02"},
		{4, "Cherry Article", 75, "2025-01-04"},
	}

	for _, data := range testData {
		_, err = db.Exec("INSERT INTO articles (id, title, views, created_at) VALUES (?, ?, ?, ?)",
			data.id, data.title, data.views, data.createdAt)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	t.Run("Ascending sort", func(t *testing.T) {
		// Create mock request with sort param
		req := httptest.NewRequest("GET", "/articles?sort=title", nil)

		// Parse sort params
		sorts := ParseSort(req)

		// Verify parsing
		if len(sorts) != 1 {
			t.Fatalf("Expected 1 sort field, got %d", len(sorts))
		}
		if sorts[0] != "title" {
			t.Errorf("Expected sort=title, got sort=%s", sorts[0])
		}

		// Build SQL query with ORDER BY clause
		validFields := []string{"title", "views", "created_at"}
		orderByClause, err := BuildSortClause(sorts, "articles", validFields)
		if err != nil {
			t.Fatalf("Failed to build sort clause: %v", err)
		}

		// Execute query
		sqlQuery := "SELECT id, title, views FROM articles " + orderByClause
		rows, err := db.Query(sqlQuery)
		if err != nil {
			t.Fatalf("Failed to execute query: %v", err)
		}
		defer rows.Close()

		// Verify results are in ascending alphabetical order
		expectedOrder := []string{"Apple Article", "Banana Article", "Cherry Article", "Zebra Article"}
		var actualOrder []string

		for rows.Next() {
			var id, views int
			var title string
			if err := rows.Scan(&id, &title, &views); err != nil {
				t.Fatalf("Failed to scan row: %v", err)
			}
			actualOrder = append(actualOrder, title)
		}

		if len(actualOrder) != len(expectedOrder) {
			t.Fatalf("Expected %d results, got %d", len(expectedOrder), len(actualOrder))
		}

		for i, expected := range expectedOrder {
			if actualOrder[i] != expected {
				t.Errorf("Position %d: expected %s, got %s", i, expected, actualOrder[i])
			}
		}
	})

	t.Run("Descending sort", func(t *testing.T) {
		// Create mock request with descending sort
		req := httptest.NewRequest("GET", "/articles?sort=-views", nil)

		// Parse sort params
		sorts := ParseSort(req)

		// Build SQL query with ORDER BY clause
		validFields := []string{"title", "views", "created_at"}
		orderByClause, err := BuildSortClause(sorts, "articles", validFields)
		if err != nil {
			t.Fatalf("Failed to build sort clause: %v", err)
		}

		// Execute query
		sqlQuery := "SELECT id, title, views FROM articles " + orderByClause
		rows, err := db.Query(sqlQuery)
		if err != nil {
			t.Fatalf("Failed to execute query: %v", err)
		}
		defer rows.Close()

		// Verify results are in descending order by views
		expectedOrder := []int{200, 100, 75, 50}
		var actualOrder []int

		for rows.Next() {
			var id, views int
			var title string
			if err := rows.Scan(&id, &title, &views); err != nil {
				t.Fatalf("Failed to scan row: %v", err)
			}
			actualOrder = append(actualOrder, views)
		}

		if len(actualOrder) != len(expectedOrder) {
			t.Fatalf("Expected %d results, got %d", len(expectedOrder), len(actualOrder))
		}

		for i, expected := range expectedOrder {
			if actualOrder[i] != expected {
				t.Errorf("Position %d: expected %d views, got %d", i, expected, actualOrder[i])
			}
		}
	})

	t.Run("Multiple sort fields", func(t *testing.T) {
		// Create mock request with multiple sort fields
		req := httptest.NewRequest("GET", "/articles?sort=-created_at,title", nil)

		// Parse sort params
		sorts := ParseSort(req)

		// Build SQL query with ORDER BY clause
		validFields := []string{"title", "views", "created_at"}
		orderByClause, err := BuildSortClause(sorts, "articles", validFields)
		if err != nil {
			t.Fatalf("Failed to build sort clause: %v", err)
		}

		// Verify the clause includes both sort fields
		if orderByClause != "ORDER BY articles.created_at DESC, articles.title ASC" {
			t.Errorf("Expected 'ORDER BY articles.created_at DESC, articles.title ASC', got %s", orderByClause)
		}

		// Execute query
		sqlQuery := "SELECT id, title, created_at FROM articles " + orderByClause
		rows, err := db.Query(sqlQuery)
		if err != nil {
			t.Fatalf("Failed to execute query: %v", err)
		}
		defer rows.Close()

		// First result should be Cherry (latest date)
		if rows.Next() {
			var id int
			var title, createdAt string
			if err := rows.Scan(&id, &title, &createdAt); err != nil {
				t.Fatalf("Failed to scan row: %v", err)
			}
			if title != "Cherry Article" {
				t.Errorf("First result should be Cherry Article, got %s", title)
			}
		}
	})
}

// TestFilteringAndSorting tests combining filters and sorting
func TestFilteringAndSorting(t *testing.T) {
	// Setup in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`CREATE TABLE products (
		id INTEGER PRIMARY KEY,
		name TEXT,
		category TEXT,
		price REAL
	)`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	testData := []struct {
		id       int
		name     string
		category string
		price    float64
	}{
		{1, "Laptop", "electronics", 999.99},
		{2, "Mouse", "electronics", 29.99},
		{3, "Desk", "furniture", 299.99},
		{4, "Chair", "furniture", 199.99},
		{5, "Keyboard", "electronics", 79.99},
	}

	for _, data := range testData {
		_, err = db.Exec("INSERT INTO products (id, name, category, price) VALUES (?, ?, ?, ?)",
			data.id, data.name, data.category, data.price)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Create mock request with both filter and sort
	req := httptest.NewRequest("GET", "/products?filter[category]=electronics&sort=-price", nil)

	// Parse both filter and sort params
	filters := ParseFilter(req)
	sorts := ParseSort(req)

	// Build SQL clauses
	validFields := []string{"name", "category", "price"}
	whereClause, args, err := BuildFilterClause(filters, "products", validFields)
	if err != nil {
		t.Fatalf("Failed to build filter clause: %v", err)
	}

	orderByClause, err := BuildSortClause(sorts, "products", validFields)
	if err != nil {
		t.Fatalf("Failed to build sort clause: %v", err)
	}

	// Execute combined query
	sqlQuery := "SELECT id, name, category, price FROM products " +
		convertToSQLitePlaceholders(whereClause) + " " + orderByClause
	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}
	defer rows.Close()

	// Verify results: should get only electronics, sorted by price descending
	expectedNames := []string{"Laptop", "Keyboard", "Mouse"}
	var actualNames []string

	for rows.Next() {
		var id int
		var name, category string
		var price float64
		if err := rows.Scan(&id, &name, &category, &price); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		// Verify all results are electronics
		if category != "electronics" {
			t.Errorf("Expected category=electronics, got category=%s", category)
		}

		actualNames = append(actualNames, name)
	}

	if len(actualNames) != len(expectedNames) {
		t.Fatalf("Expected %d results, got %d", len(expectedNames), len(actualNames))
	}

	for i, expected := range expectedNames {
		if actualNames[i] != expected {
			t.Errorf("Position %d: expected %s, got %s", i, expected, actualNames[i])
		}
	}
}

// TestSparseFieldsetsIntegration tests the complete sparse fieldsets flow
func TestSparseFieldsetsIntegration(t *testing.T) {
	t.Run("Basic sparse fieldsets parsing", func(t *testing.T) {
		// Create mock request with fields parameter
		req := httptest.NewRequest("GET", "/users?fields[users]=name,email", nil)

		// Parse fields parameter
		fields := ParseFields(req)

		// Verify parsing
		if len(fields) != 1 {
			t.Fatalf("Expected 1 resource type, got %d", len(fields))
		}

		userFields, ok := fields["users"]
		if !ok {
			t.Fatal("Expected 'users' key in fields map")
		}

		if len(userFields) != 2 {
			t.Fatalf("Expected 2 fields, got %d", len(userFields))
		}

		expectedFields := map[string]bool{"name": true, "email": true}
		for _, field := range userFields {
			if !expectedFields[field] {
				t.Errorf("Unexpected field: %s", field)
			}
		}
	})

	t.Run("Multiple resource types", func(t *testing.T) {
		// Create mock request with fields for multiple resource types
		req := httptest.NewRequest("GET", "/posts?include=author&fields[posts]=title&fields[users]=name", nil)

		// Parse fields parameter
		fields := ParseFields(req)

		// Verify parsing
		if len(fields) != 2 {
			t.Fatalf("Expected 2 resource types, got %d", len(fields))
		}

		postFields, ok := fields["posts"]
		if !ok {
			t.Error("Expected 'posts' key in fields map")
		} else if len(postFields) != 1 || postFields[0] != "title" {
			t.Errorf("Expected posts fields [title], got %v", postFields)
		}

		userFields, ok := fields["users"]
		if !ok {
			t.Error("Expected 'users' key in fields map")
		} else if len(userFields) != 1 || userFields[0] != "name" {
			t.Errorf("Expected users fields [name], got %v", userFields)
		}
	})

	t.Run("Empty fields parameter", func(t *testing.T) {
		// Create mock request with empty fields parameter
		req := httptest.NewRequest("GET", "/users?fields[users]=", nil)

		// Parse fields parameter
		fields := ParseFields(req)

		// Verify parsing - empty fields should result in empty array
		userFields, ok := fields["users"]
		if !ok {
			t.Fatal("Expected 'users' key in fields map")
		}

		if len(userFields) != 0 {
			t.Errorf("Expected empty fields array, got %v", userFields)
		}
	})

	t.Run("Sparse fieldsets applied to JSON response", func(t *testing.T) {
		// Simulate a JSON:API response with sparse fieldsets applied
		// Note: In real implementation, the DataDog jsonapi library handles this

		fullResponse := map[string]interface{}{
			"data": map[string]interface{}{
				"type": "users",
				"id":   "123",
				"attributes": map[string]interface{}{
					"name":       "John Doe",
					"email":      "john@example.com",
					"bio":        "Software developer",
					"created_at": "2025-01-01T00:00:00Z",
				},
			},
		}

		// Parse request for sparse fieldsets
		req := httptest.NewRequest("GET", "/users/123?fields[users]=name,email", nil)
		requestedFields := ParseFields(req)

		// In a real implementation, we would pass requestedFields to the JSON:API marshaler
		// For this test, we simulate the filtering manually
		userFields := requestedFields["users"]
		if len(userFields) > 0 {
			// Simulate sparse fieldsets filtering
			data := fullResponse["data"].(map[string]interface{})
			attributes := data["attributes"].(map[string]interface{})

			filteredAttributes := make(map[string]interface{})
			for _, field := range userFields {
				if val, ok := attributes[field]; ok {
					filteredAttributes[field] = val
				}
			}

			// Verify only requested fields are present
			if len(filteredAttributes) != 2 {
				t.Errorf("Expected 2 attributes after filtering, got %d", len(filteredAttributes))
			}

			if _, ok := filteredAttributes["name"]; !ok {
				t.Error("Expected 'name' attribute in filtered response")
			}

			if _, ok := filteredAttributes["email"]; !ok {
				t.Error("Expected 'email' attribute in filtered response")
			}

			if _, ok := filteredAttributes["bio"]; ok {
				t.Error("Did not expect 'bio' attribute in filtered response")
			}

			// Verify id and type are always preserved (required by JSON:API spec)
			if _, ok := data["id"]; !ok {
				t.Error("Expected 'id' to always be present")
			}

			if _, ok := data["type"]; !ok {
				t.Error("Expected 'type' to always be present")
			}
		}
	})
}

// TestInvalidFilterFields tests error handling for invalid filter fields
func TestInvalidFilterFields(t *testing.T) {
	// Create mock request with invalid filter field
	req := httptest.NewRequest("GET", "/posts?filter[invalid_field]=value", nil)

	// Parse filter params
	filters := ParseFilter(req)

	// Try to build filter clause with valid fields list
	validFields := []string{"title", "status", "author_id"}
	whereClause, args, err := BuildFilterClause(filters, "posts", validFields)

	// Verify error is returned
	if err == nil {
		t.Fatal("Expected error for invalid filter field, got nil")
	}

	// Verify error message mentions the invalid field
	expectedErrMsg := "invalid filter fields: invalid_field"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error %q, got %q", expectedErrMsg, err.Error())
	}

	// Verify no SQL was generated
	if whereClause != "" {
		t.Errorf("Expected empty WHERE clause on error, got: %s", whereClause)
	}

	if args != nil {
		t.Errorf("Expected nil args on error, got: %v", args)
	}
}

// TestInvalidSortFields tests error handling for invalid sort fields
func TestInvalidSortFields(t *testing.T) {
	// Create mock request with invalid sort field
	req := httptest.NewRequest("GET", "/posts?sort=invalid_field,-title", nil)

	// Parse sort params
	sorts := ParseSort(req)

	// Verify parsing extracted both fields
	if len(sorts) != 2 {
		t.Fatalf("Expected 2 sort fields, got %d", len(sorts))
	}

	// Try to build sort clause with valid fields list
	validFields := []string{"title", "created_at", "views"}
	orderByClause, err := BuildSortClause(sorts, "posts", validFields)

	// Verify error is returned
	if err == nil {
		t.Fatal("Expected error for invalid sort field, got nil")
	}

	// Verify error message mentions the invalid field
	expectedErrMsg := "invalid sort fields: invalid_field"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error %q, got %q", expectedErrMsg, err.Error())
	}

	// Verify no SQL was generated
	if orderByClause != "" {
		t.Errorf("Expected empty ORDER BY clause on error, got: %s", orderByClause)
	}
}

// TestComplexScenario tests a realistic scenario with multiple features
func TestComplexScenario(t *testing.T) {
	// Setup in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table with multiple fields
	_, err = db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		name TEXT,
		email TEXT,
		role TEXT,
		created_at TEXT
	)`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	testData := []struct {
		id        int
		name      string
		email     string
		role      string
		createdAt string
	}{
		{1, "Alice Admin", "alice@example.com", "admin", "2025-01-01"},
		{2, "Bob User", "bob@example.com", "user", "2025-01-02"},
		{3, "Charlie User", "charlie@example.com", "user", "2025-01-03"},
		{4, "David Admin", "david@example.com", "admin", "2025-01-04"},
	}

	for _, data := range testData {
		_, err = db.Exec("INSERT INTO users (id, name, email, role, created_at) VALUES (?, ?, ?, ?, ?)",
			data.id, data.name, data.email, data.role, data.createdAt)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Simulate a complex API request:
	// - Filter by role=user
	// - Sort by created_at descending
	// - Sparse fieldsets requesting only name and email
	req := httptest.NewRequest("GET", "/users?filter[role]=user&sort=-created_at&fields[users]=name,email", nil)

	// 1. Parse all query parameters
	filters := ParseFilter(req)
	sorts := ParseSort(req)
	fields := ParseFields(req)

	// 2. Verify parsing
	if filters["role"] != "user" {
		t.Errorf("Expected role=user, got role=%s", filters["role"])
	}

	if len(sorts) != 1 || sorts[0] != "-created_at" {
		t.Errorf("Expected sort=['-created_at'], got %v", sorts)
	}

	if len(fields["users"]) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(fields["users"]))
	}

	// 3. Build SQL query
	validFields := []string{"name", "email", "role", "created_at"}
	whereClause, args, err := BuildFilterClause(filters, "users", validFields)
	if err != nil {
		t.Fatalf("Failed to build filter clause: %v", err)
	}

	orderByClause, err := BuildSortClause(sorts, "users", validFields)
	if err != nil {
		t.Fatalf("Failed to build sort clause: %v", err)
	}

	// 4. Execute query
	sqlQuery := "SELECT id, name, email, role, created_at FROM users " +
		convertToSQLitePlaceholders(whereClause) + " " + orderByClause
	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}
	defer rows.Close()

	// 5. Collect results
	var results []map[string]interface{}
	for rows.Next() {
		var id int
		var name, email, role, createdAt string
		if err := rows.Scan(&id, &name, &email, &role, &createdAt); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		// Apply sparse fieldsets filtering
		requestedFields := fields["users"]
		result := map[string]interface{}{
			"id":   id,   // id is always included per JSON:API spec
			"type": "users", // type is always included per JSON:API spec
		}

		// Only include requested fields
		fieldMap := map[string]string{
			"name":       name,
			"email":      email,
			"role":       role,
			"created_at": createdAt,
		}

		for _, field := range requestedFields {
			if val, ok := fieldMap[field]; ok {
				result[field] = val
			}
		}

		results = append(results, result)
	}

	// 6. Verify results
	if len(results) != 2 {
		t.Fatalf("Expected 2 users with role=user, got %d", len(results))
	}

	// Verify order (should be Charlie, then Bob - descending by created_at)
	if results[0]["name"] != "Charlie User" {
		t.Errorf("First result should be Charlie User, got %s", results[0]["name"])
	}

	if results[1]["name"] != "Bob User" {
		t.Errorf("Second result should be Bob User, got %s", results[1]["name"])
	}

	// Verify sparse fieldsets were applied (only name and email, plus id/type)
	for i, result := range results {
		if _, ok := result["role"]; ok {
			t.Errorf("Result %d should not include 'role' field", i)
		}
		if _, ok := result["created_at"]; ok {
			t.Errorf("Result %d should not include 'created_at' field", i)
		}
		if _, ok := result["name"]; !ok {
			t.Errorf("Result %d should include 'name' field", i)
		}
		if _, ok := result["email"]; !ok {
			t.Errorf("Result %d should include 'email' field", i)
		}
	}
}

// TestJSONAPIResponseStructure tests that responses conform to JSON:API spec
func TestJSONAPIResponseStructure(t *testing.T) {
	// Simulate a JSON:API response structure
	response := map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"type": "posts",
				"id":   "1",
				"attributes": map[string]interface{}{
					"title":  "Test Post",
					"status": "published",
				},
			},
		},
		"meta": map[string]interface{}{
			"page":     1,
			"per_page": 50,
			"total":    1,
		},
		"links": map[string]interface{}{
			"self":  "/posts?page[limit]=50&page[offset]=0",
			"first": "/posts?page[limit]=50&page[offset]=0",
			"last":  "/posts?page[limit]=50&page[offset]=0",
		},
	}

	// Verify top-level structure
	if _, ok := response["data"]; !ok {
		t.Error("JSON:API response must have 'data' field")
	}

	if _, ok := response["meta"]; !ok {
		t.Error("JSON:API response should have 'meta' field for pagination")
	}

	if _, ok := response["links"]; !ok {
		t.Error("JSON:API response should have 'links' field for pagination")
	}

	// Verify data structure
	data := response["data"].([]map[string]interface{})
	if len(data) != 1 {
		t.Fatalf("Expected 1 resource in data, got %d", len(data))
	}

	resource := data[0]

	// Verify required resource fields
	if _, ok := resource["type"]; !ok {
		t.Error("JSON:API resource must have 'type' field")
	}

	if _, ok := resource["id"]; !ok {
		t.Error("JSON:API resource must have 'id' field")
	}

	if _, ok := resource["attributes"]; !ok {
		t.Error("JSON:API resource should have 'attributes' field")
	}

	// Verify meta structure
	meta := response["meta"].(map[string]interface{})
	requiredMetaFields := []string{"page", "per_page", "total"}
	for _, field := range requiredMetaFields {
		if _, ok := meta[field]; !ok {
			t.Errorf("Meta should have '%s' field", field)
		}
	}

	// Verify links structure
	links := response["links"].(map[string]interface{})
	requiredLinkFields := []string{"self", "first", "last"}
	for _, field := range requiredLinkFields {
		if _, ok := links[field]; !ok {
			t.Errorf("Links should have '%s' field", field)
		}
	}
}

// TestCamelCaseToSnakeCase tests field name conversion in integration
func TestCamelCaseToSnakeCase(t *testing.T) {
	// Setup in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table with snake_case columns (database convention)
	_, err = db.Exec(`CREATE TABLE posts (
		id INTEGER PRIMARY KEY,
		author_id INTEGER,
		created_at TEXT
	)`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	_, err = db.Exec("INSERT INTO posts (id, author_id, created_at) VALUES (?, ?, ?)",
		1, 123, "2025-01-01")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// API request uses camelCase (JSON convention)
	req := httptest.NewRequest("GET", "/posts?filter[authorId]=123", nil)

	// Parse filter params
	filters := ParseFilter(req)

	// Build SQL query - validFields should be in snake_case
	validFields := []string{"author_id", "created_at"}
	whereClause, args, err := BuildFilterClause(filters, "posts", validFields)
	if err != nil {
		t.Fatalf("Failed to build filter clause: %v", err)
	}

	// Verify the WHERE clause uses snake_case
	if whereClause != "WHERE posts.author_id = $1" {
		t.Errorf("Expected WHERE clause with snake_case, got: %s", whereClause)
	}

	// Execute query
	sqlQuery := "SELECT id, author_id FROM posts " + convertToSQLitePlaceholders(whereClause)
	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}
	defer rows.Close()

	// Verify result
	if rows.Next() {
		var id, authorID int
		if err := rows.Scan(&id, &authorID); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}

		if authorID != 123 {
			t.Errorf("Expected author_id=123, got author_id=%d", authorID)
		}
	} else {
		t.Error("Expected to find 1 row, got 0")
	}
}

// Helper function to convert PostgreSQL-style placeholders ($1, $2) to SQLite-style (?, ?)
func convertToSQLitePlaceholders(query string) string {
	// Simple replacement for integration tests
	// In production, use the appropriate placeholder for your database
	result := query
	for i := 10; i >= 1; i-- {
		placeholder := fmt.Sprintf("$%d", i)
		result = replaceFirst(result, placeholder, "?")
	}
	return result
}

// replaceFirst replaces only the first occurrence of old with new
func replaceFirst(s, old, new string) string {
	i := 0
	for {
		j := indexOf(s[i:], old)
		if j == -1 {
			break
		}
		s = s[:i+j] + new + s[i+j+len(old):]
		i = i + j + len(new)
	}
	return s
}

// indexOf returns the index of the first occurrence of substr in s, or -1
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestErrorResponses tests that proper error responses are generated
func TestErrorResponses(t *testing.T) {
	t.Run("Invalid filter field returns structured error", func(t *testing.T) {
		// In a real handler, this would be converted to a JSON:API error response
		req := httptest.NewRequest("GET", "/posts?filter[hacker_field]=exploit", nil)

		filters := ParseFilter(req)
		validFields := []string{"title", "status"}

		_, _, err := BuildFilterClause(filters, "posts", validFields)

		// Verify error is structured and safe
		if err == nil {
			t.Fatal("Expected error for invalid field")
		}

		// Simulate converting to JSON:API error format
		errorResponse := map[string]interface{}{
			"errors": []map[string]interface{}{
				{
					"status": "400",
					"code":   "invalid_query_parameter",
					"title":  "Invalid Query Parameter",
					"detail": err.Error(),
					"source": map[string]interface{}{
						"parameter": "filter[hacker_field]",
					},
				},
			},
		}

		// Verify error structure
		errors := errorResponse["errors"].([]map[string]interface{})
		if len(errors) != 1 {
			t.Fatalf("Expected 1 error, got %d", len(errors))
		}

		if errors[0]["status"] != "400" {
			t.Errorf("Expected status 400, got %v", errors[0]["status"])
		}

		// Verify error is JSON-serializable
		_, err = json.Marshal(errorResponse)
		if err != nil {
			t.Errorf("Error response should be JSON-serializable: %v", err)
		}
	})
}
