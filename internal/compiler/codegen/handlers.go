package codegen

import (
	"fmt"
	"strings"

	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// GenerateHandlers generates HTTP handlers for all resources
func (g *Generator) GenerateHandlers(resources []*ast.ResourceNode, moduleName string) (string, error) {
	g.reset()

	// Package declaration
	g.writeLine("package handlers")
	g.writeLine("")

	// Imports
	g.imports["database/sql"] = true
	g.imports["encoding/json"] = true
	g.imports["errors"] = true
	g.imports["fmt"] = true
	g.imports["io"] = true
	g.imports["net/http"] = true
	g.imports["strconv"] = true
	g.imports["github.com/go-chi/chi/v5"] = true
	g.imports["github.com/DataDog/jsonapi"] = true
	g.imports[moduleName+"/models"] = true // Import models package
	g.imports["github.com/conduit-lang/conduit/pkg/web/response"] = true // Import response package for JSON:API support
	g.imports["github.com/conduit-lang/conduit/pkg/web/query"] = true    // Import query package for Phase 3 support

	// Pre-scan resources for additional imports (like uuid for ID types)
	for _, resource := range resources {
		if g.getIDType(resource) == "uuid" {
			g.imports["github.com/google/uuid"] = true
		}
	}

	g.writeImports()
	g.writeLine("")

	// Generate error response type and helper
	g.generateErrorHelpers()
	g.writeLine("")

	// Generate handlers for each resource
	for _, resource := range resources {
		if err := g.generateResourceHandlers(resource); err != nil {
			return "", fmt.Errorf("failed to generate handlers for %s: %w", resource.Name, err)
		}
		g.writeLine("")
	}

	return g.buf.String(), nil
}

// get IDType returns the type of the resource's ID field
func (g *Generator) getIDType(resource *ast.ResourceNode) string {
	for _, field := range resource.Fields {
		if field.Name == "id" {
			return field.Type.Name
		}
	}
	// Default to int64 if no explicit ID field
	return "int"
}

// generateErrorHelpers generates the error response type and helper function
func (g *Generator) generateErrorHelpers() {
	g.writeLine("// ErrorResponse represents a JSON error response")
	g.writeLine("type ErrorResponse struct {")
	g.indent++
	g.writeLine("Error string `json:\"error\"`")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("// respondWithError writes a JSON error response")
	g.writeLine("func respondWithError(w http.ResponseWriter, message string, statusCode int) {")
	g.indent++
	g.writeLine("w.Header().Set(\"Content-Type\", \"application/json\")")
	g.writeLine("w.WriteHeader(statusCode)")
	g.writeLine("json.NewEncoder(w).Encode(ErrorResponse{Error: message})")
	g.indent--
	g.writeLine("}")
}

// generateIDParsingCode generates code to parse ID from URL based on type
func (g *Generator) generateIDParsingCode(resource *ast.ResourceNode) {
	idType := g.getIDType(resource)

	g.writeLine("// Parse ID from URL")
	g.writeLine("idStr := chi.URLParam(r, \"id\")")

	switch idType {
	case "uuid":
		// UUID needs the uuid package imported
		g.imports["github.com/google/uuid"] = true
		g.writeLine("id, err := uuid.Parse(idStr)")
		g.writeLine("if err != nil {")
		g.indent++
		g.writeLine("respondWithError(w, \"Invalid ID\", http.StatusBadRequest)")
		g.writeLine("return")
		g.indent--
		g.writeLine("}")
	default: // int, int64, etc.
		g.writeLine("id, err := strconv.ParseInt(idStr, 10, 64)")
		g.writeLine("if err != nil {")
		g.indent++
		g.writeLine("respondWithError(w, \"Invalid ID\", http.StatusBadRequest)")
		g.writeLine("return")
		g.indent--
		g.writeLine("}")
	}
	g.writeLine("")
}

// generateResourceHandlers generates all CRUD handlers for a resource
func (g *Generator) generateResourceHandlers(resource *ast.ResourceNode) error {
	resourceLower := strings.ToLower(resource.Name)

	// List handler
	g.generateListHandler(resource)
	g.writeLine("")

	// Get handler
	g.generateGetHandler(resource)
	g.writeLine("")

	// Create handler
	g.generateCreateHandler(resource)
	g.writeLine("")

	// Update handler
	g.generateUpdateHandler(resource)
	g.writeLine("")

	// Delete handler
	g.generateDeleteHandler(resource)
	g.writeLine("")

	// Router registration helper
	g.writeLine("// Register%sRoutes registers all routes for %s", resource.Name, resourceLower)
	g.writeLine("func Register%sRoutes(r chi.Router, db *sql.DB) {", resource.Name)
	g.indent++
	tableName := g.toTableName(resource.Name)
	g.writeLine("r.Get(\"/%s\", List%sHandler(db))", tableName, resource.Name)
	g.writeLine("r.Post(\"/%s\", Create%sHandler(db))", tableName, resource.Name)
	g.writeLine("r.Get(\"/%s/{id}\", Get%sHandler(db))", tableName, resource.Name)
	g.writeLine("r.Put(\"/%s/{id}\", Update%sHandler(db))", tableName, resource.Name)
	g.writeLine("r.Delete(\"/%s/{id}\", Delete%sHandler(db))", tableName, resource.Name)
	g.indent--
	g.writeLine("}")

	return nil
}

// toSnakeCase converts a field name to snake_case
func (g *Generator) toSnakeCase(name string) string {
	var result strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// generateValidFieldsList generates code for a slice of valid field names
func (g *Generator) generateValidFieldsList(resource *ast.ResourceNode) {
	g.writeLine("validFields := []string{")
	g.indent++
	for i, field := range resource.Fields {
		// Convert field name to snake_case for database column names
		columnName := g.toSnakeCase(field.Name)
		if i < len(resource.Fields)-1 {
			g.writeLine("\"%s\",", columnName)
		} else {
			g.writeLine("\"%s\",", columnName)
		}
	}
	g.indent--
	g.writeLine("}")
}

// generateListHandler generates the LIST handler (GET /resources)
func (g *Generator) generateListHandler(resource *ast.ResourceNode) {
	resourceLower := strings.ToLower(resource.Name)
	tableName := g.toTableName(resource.Name)

	g.writeLine("// List%sHandler handles GET /%s - list all %s with pagination",
		resource.Name, tableName, resourceLower+"s")
	g.writeLine("func List%sHandler(db *sql.DB) http.HandlerFunc {", resource.Name)
	g.indent++
	g.writeLine("return func(w http.ResponseWriter, r *http.Request) {")
	g.indent++

	g.writeLine("ctx := r.Context()")
	g.writeLine("")

	// Parse query parameters for pagination
	g.writeLine("// Parse pagination parameters")
	g.writeLine("limit := 50 // default")
	g.writeLine("offset := 0")
	g.writeLine("")
	g.writeLine("if limitStr := r.URL.Query().Get(\"limit\"); limitStr != \"\" {")
	g.indent++
	g.writeLine("if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {")
	g.indent++
	g.writeLine("limit = l")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("if offsetStr := r.URL.Query().Get(\"offset\"); offsetStr != \"\" {")
	g.indent++
	g.writeLine("if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {")
	g.indent++
	g.writeLine("offset = o")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Parse JSON:API Phase 3 query parameters
	g.writeLine("// Parse JSON:API Phase 3 query parameters")
	g.writeLine("includes := query.ParseInclude(r)")
	g.writeLine("fields := query.ParseFields(r)")
	g.writeLine("filters := query.ParseFilter(r)")
	g.writeLine("sorts := query.ParseSort(r)")
	g.writeLine("")

	// TODO comment for includes support
	g.writeLine("// TODO: Phase 3 - Load relationships if includes is not empty")
	g.writeLine("// This requires implementing relationship loading in models package")
	g.writeLine("_ = includes // Silence unused variable warning")
	g.writeLine("")

	// Generate valid fields list
	g.writeLine("// Valid fields for filtering and sorting")
	g.generateValidFieldsList(resource)
	g.writeLine("")

	// Build base query with filtering and sorting
	g.writeLine("// Build base query")
	g.writeLine("baseQuery := \"SELECT * FROM %s\"", tableName)
	g.writeLine("")

	// Apply filtering
	g.writeLine("// Apply filtering")
	g.writeLine("whereClause, filterArgs, err := query.BuildFilterClause(filters, \"%s\", validFields)", tableName)
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("if response.IsJSONAPI(r) {")
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusBadRequest, err)")
	g.indent--
	g.writeLine("} else {")
	g.indent++
	g.writeLine("respondWithError(w, err.Error(), http.StatusBadRequest)")
	g.indent--
	g.writeLine("}")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("if whereClause != \"\" {")
	g.indent++
	g.writeLine("baseQuery += \" \" + whereClause")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Apply sorting
	g.writeLine("// Apply sorting")
	g.writeLine("orderByClause, err := query.BuildSortClause(sorts, \"%s\", validFields)", tableName)
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("if response.IsJSONAPI(r) {")
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusBadRequest, err)")
	g.indent--
	g.writeLine("} else {")
	g.indent++
	g.writeLine("respondWithError(w, err.Error(), http.StatusBadRequest)")
	g.indent--
	g.writeLine("}")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("if orderByClause != \"\" {")
	g.indent++
	g.writeLine("baseQuery += \" \" + orderByClause")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Apply pagination
	g.writeLine("// Apply pagination")
	g.writeLine("paramIndex := len(filterArgs) + 1")
	g.writeLine("baseQuery += fmt.Sprintf(\" LIMIT $%d OFFSET $%d\", paramIndex, paramIndex+1)")
	g.writeLine("")

	// Execute query
	g.writeLine("// Execute query")
	g.writeLine("args := append(filterArgs, limit, offset)")
	g.writeLine("rows, err := db.QueryContext(ctx, baseQuery, args...)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("if response.IsJSONAPI(r) {")
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusInternalServerError, fmt.Errorf(\"Failed to query %s: %%v\", err))", resourceLower+"s")
	g.indent--
	g.writeLine("} else {")
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Failed to query %s: %%v\", err), http.StatusInternalServerError)", resourceLower+"s")
	g.indent--
	g.writeLine("}")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("defer rows.Close()")
	g.writeLine("")

	// Scan results
	g.writeLine("// Scan results")
	g.writeLine("results := []*models.%s{}", resource.Name)
	g.writeLine("for rows.Next() {")
	g.indent++
	g.writeLine("item := &models.%s{}", resource.Name)

	// Generate scan call with all field pointers
	scanFields := g.generateScanFields(resource)
	g.writeLine("if err := rows.Scan(%s); err != nil {", scanFields)
	g.indent++
	g.writeLine("if response.IsJSONAPI(r) {")
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusInternalServerError, fmt.Errorf(\"Failed to scan %s: %%v\", err))", resourceLower)
	g.indent--
	g.writeLine("} else {")
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Failed to scan %s: %%v\", err), http.StatusInternalServerError)", resourceLower)
	g.indent--
	g.writeLine("}")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("results = append(results, item)")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("if err := rows.Err(); err != nil {")
	g.indent++
	g.writeLine("if response.IsJSONAPI(r) {")
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusInternalServerError, fmt.Errorf(\"Error iterating %s: %%v\", err))", resourceLower+"s")
	g.indent--
	g.writeLine("} else {")
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Error iterating %s: %%v\", err), http.StatusInternalServerError)", resourceLower+"s")
	g.indent--
	g.writeLine("}")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Get total count for pagination (use filtered count)
	g.writeLine("// Get total count for pagination (with filters applied)")
	g.writeLine("countQuery := \"SELECT COUNT(*) FROM %s\"", tableName)
	g.writeLine("if whereClause != \"\" {")
	g.indent++
	g.writeLine("countQuery += \" \" + whereClause")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("var total int")
	g.writeLine("err = db.QueryRowContext(ctx, countQuery, filterArgs...).Scan(&total)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("if response.IsJSONAPI(r) {")
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusInternalServerError, fmt.Errorf(\"Failed to count %s: %%v\", err))", resourceLower+"s")
	g.indent--
	g.writeLine("} else {")
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Failed to count %s: %%v\", err), http.StatusInternalServerError)", resourceLower+"s")
	g.indent--
	g.writeLine("}")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Content negotiation
	g.writeLine("// Content negotiation: JSON:API or legacy JSON")
	g.writeLine("if response.IsJSONAPI(r) {")
	g.indent++
	g.writeLine("// JSON:API format")
	g.writeLine("page := (offset / limit) + 1")
	g.writeLine("meta := map[string]interface{}{")
	g.indent++
	g.writeLine("\"page\": page,")
	g.writeLine("\"per_page\": limit,")
	g.writeLine("\"total\": total,")
	g.indent--
	g.writeLine("}")
	g.writeLine("links := response.BuildPaginationLinks(r.URL.Path, page, limit, total)")
	g.writeLine("")

	// Marshal with options
	g.writeLine("// Marshal with pagination metadata")
	g.writeLine("opts := []jsonapi.MarshalOption{")
	g.indent++
	g.writeLine("jsonapi.MarshalMeta(meta),")
	g.writeLine("jsonapi.MarshalLinks(links),")
	g.indent--
	g.writeLine("}")
	g.writeLine("")
	g.writeLine("data, err := jsonapi.Marshal(results, opts...)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusInternalServerError, fmt.Errorf(\"Failed to marshal response: %%v\", err))")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Apply sparse fieldsets
	g.writeLine("// Apply sparse fieldsets if requested")
	g.writeLine("if len(fields) > 0 {")
	g.indent++
	g.writeLine("data, err = response.ApplySparseFieldsets(data, fields)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusInternalServerError, fmt.Errorf(\"Failed to apply sparse fieldsets: %%v\", err))")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Write response
	g.writeLine("w.Header().Set(\"Content-Type\", response.JSONAPIMediaType)")
	g.writeLine("w.WriteHeader(http.StatusOK)")
	g.writeLine("w.Write(data)")

	g.indent--
	g.writeLine("} else {")
	g.indent++
	g.writeLine("// Legacy JSON format")
	g.writeLine("w.Header().Set(\"Content-Type\", \"application/json\")")
	g.writeLine("if err := json.NewEncoder(w).Encode(results); err != nil {")
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Failed to encode response: %%v\", err), http.StatusInternalServerError)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")

	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
}

// generateGetHandler generates the GET handler (GET /resources/:id)
func (g *Generator) generateGetHandler(resource *ast.ResourceNode) {
	resourceLower := strings.ToLower(resource.Name)
	tableName := g.toTableName(resource.Name)

	g.writeLine("// Get%sHandler handles GET /%s/{id} - get a single %s",
		resource.Name, tableName, resourceLower)
	g.writeLine("func Get%sHandler(db *sql.DB) http.HandlerFunc {", resource.Name)
	g.indent++
	g.writeLine("return func(w http.ResponseWriter, r *http.Request) {")
	g.indent++

	g.writeLine("ctx := r.Context()")
	g.writeLine("")

	// Parse ID from URL
	g.generateIDParsingCode(resource)

	// Call FindByID function
	g.writeLine("result, err := models.Find%sByID(ctx, db, id)", resource.Name)
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("if err == sql.ErrNoRows {")
	g.indent++
	g.writeLine("respondWithError(w, \"Not found\", http.StatusNotFound)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Failed to get %s: %%v\", err), http.StatusInternalServerError)", resourceLower)
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Content negotiation
	g.writeLine("// Content negotiation: JSON:API or legacy JSON")
	g.writeLine("if response.IsJSONAPI(r) {")
	g.indent++
	g.writeLine("// JSON:API format")
	g.writeLine("if err := response.RenderJSONAPI(w, http.StatusOK, result); err != nil {")
	g.indent++
	g.writeLine("respondWithError(w, \"Failed to encode response\", http.StatusInternalServerError)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("} else {")
	g.indent++
	g.writeLine("// Legacy JSON format")
	g.writeLine("w.Header().Set(\"Content-Type\", \"application/json\")")
	g.writeLine("if err := json.NewEncoder(w).Encode(result); err != nil {")
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Failed to encode response: %v\", err), http.StatusInternalServerError)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")

	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
}

// generateCreateHandler generates the CREATE handler (POST /resources)
func (g *Generator) generateCreateHandler(resource *ast.ResourceNode) {
	resourceLower := strings.ToLower(resource.Name)
	tableName := g.toTableName(resource.Name)
	receiverName := strings.ToLower(resource.Name[0:1])

	g.writeLine("// Create%sHandler handles POST /%s - create a new %s",
		resource.Name, tableName, resourceLower)
	g.writeLine("func Create%sHandler(db *sql.DB) http.HandlerFunc {", resource.Name)
	g.indent++
	g.writeLine("return func(w http.ResponseWriter, r *http.Request) {")
	g.indent++

	g.writeLine("ctx := r.Context()")
	g.writeLine("")

	// Branch on content negotiation
	g.writeLine("// Check if JSON:API format is requested")
	g.writeLine("if response.IsJSONAPI(r) {")
	g.indent++

	// JSON:API path
	g.writeLine("// Validate Content-Type")
	g.writeLine("if !response.ValidateJSONAPIContentType(w, r) {")
	g.indent++
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("// Limit request body size to prevent DoS attacks (10MB default)")
	g.writeLine("r.Body = http.MaxBytesReader(w, r.Body, 10<<20)")
	g.writeLine("")
	g.writeLine("// Read request body")
	g.writeLine("body, err := io.ReadAll(r.Body)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("var maxBytesError *http.MaxBytesError")
	g.writeLine("if errors.As(err, &maxBytesError) {")
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusRequestEntityTooLarge, fmt.Errorf(\"Request body too large (max 10MB)\"))")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("response.RenderJSONAPIError(w, http.StatusBadRequest, fmt.Errorf(\"Failed to read request body: %%v\", err))")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("// Unmarshal JSON:API request")
	g.writeLine("var %s models.%s", receiverName, resource.Name)
	g.writeLine("if err := jsonapi.Unmarshal(body, &%s); err != nil {", receiverName)
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusBadRequest, fmt.Errorf(\"Invalid JSON:API request: %%v\", err))")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("// Create %s (includes validation and hooks)", resourceLower)
	g.writeLine("if err := %s.Create(ctx, db); err != nil {", receiverName)
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusUnprocessableEntity, err)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("// Set Location header")
	g.writeLine("w.Header().Set(\"Location\", fmt.Sprintf(\"/api/%s/%%s\", %s.ID))", tableName, receiverName)
	g.writeLine("")

	g.writeLine("// Render JSON:API response")
	g.writeLine("if err := response.RenderJSONAPI(w, http.StatusCreated, &%s); err != nil {", receiverName)
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusInternalServerError, fmt.Errorf(\"Failed to encode response: %%v\", err))")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")

	g.indent--
	g.writeLine("} else {")
	g.indent++

	// Legacy JSON path
	g.writeLine("// Legacy JSON format")
	g.writeLine("var %s models.%s", receiverName, resource.Name)
	g.writeLine("if err := json.NewDecoder(r.Body).Decode(&%s); err != nil {", receiverName)
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Invalid request body: %%v\", err), http.StatusBadRequest)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("// Create %s (includes validation and hooks)", resourceLower)
	g.writeLine("if err := %s.Create(ctx, db); err != nil {", receiverName)
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Failed to create %s: %%v\", err), http.StatusUnprocessableEntity)", resourceLower)
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("w.Header().Set(\"Content-Type\", \"application/json\")")
	g.writeLine("w.WriteHeader(http.StatusCreated)")
	g.writeLine("if err := json.NewEncoder(w).Encode(%s); err != nil {", receiverName)
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Failed to encode response: %%v\", err), http.StatusInternalServerError)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")

	g.indent--
	g.writeLine("}")

	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
}

// generateUpdateHandler generates the UPDATE handler (PUT /resources/:id)
func (g *Generator) generateUpdateHandler(resource *ast.ResourceNode) {
	resourceLower := strings.ToLower(resource.Name)
	tableName := g.toTableName(resource.Name)
	receiverName := strings.ToLower(resource.Name[0:1])
	idType := g.getIDType(resource)

	g.writeLine("// Update%sHandler handles PUT /%s/{id} - update an existing %s",
		resource.Name, tableName, resourceLower)
	g.writeLine("func Update%sHandler(db *sql.DB) http.HandlerFunc {", resource.Name)
	g.indent++
	g.writeLine("return func(w http.ResponseWriter, r *http.Request) {")
	g.indent++

	g.writeLine("ctx := r.Context()")
	g.writeLine("")

	// Parse ID from URL
	g.generateIDParsingCode(resource)

	// Branch on content negotiation
	g.writeLine("// Check if JSON:API format is requested")
	g.writeLine("if response.IsJSONAPI(r) {")
	g.indent++

	// JSON:API path
	g.writeLine("// Validate Content-Type")
	g.writeLine("if !response.ValidateJSONAPIContentType(w, r) {")
	g.indent++
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("// Limit request body size to prevent DoS attacks (10MB default)")
	g.writeLine("r.Body = http.MaxBytesReader(w, r.Body, 10<<20)")
	g.writeLine("")
	g.writeLine("// Read request body")
	g.writeLine("body, err := io.ReadAll(r.Body)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("var maxBytesError *http.MaxBytesError")
	g.writeLine("if errors.As(err, &maxBytesError) {")
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusRequestEntityTooLarge, fmt.Errorf(\"Request body too large (max 10MB)\"))")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("response.RenderJSONAPIError(w, http.StatusBadRequest, fmt.Errorf(\"Failed to read request body: %%v\", err))")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("// Unmarshal JSON:API request")
	g.writeLine("var %s models.%s", receiverName, resource.Name)
	g.writeLine("if err := jsonapi.Unmarshal(body, &%s); err != nil {", receiverName)
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusBadRequest, fmt.Errorf(\"Invalid JSON:API request: %%v\", err))")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Validate ID matches URL  (allow zero UUID in create requests)
	g.writeLine("// Validate ID matches URL")
	if idType == "uuid" {
		g.writeLine("if %s.ID != uuid.Nil && %s.ID != id {", receiverName, receiverName)
	} else {
		// Ensure type compatibility for integer IDs by converting both to int64
		g.writeLine("if int64(%s.ID) != id {", receiverName)
	}
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusConflict, fmt.Errorf(\"ID in request body doesn't match URL\"))")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("// Update %s (includes validation and hooks)", resourceLower)
	g.writeLine("if err := %s.Update(ctx, db); err != nil {", receiverName)
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusUnprocessableEntity, err)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("// Render JSON:API response")
	g.writeLine("if err := response.RenderJSONAPI(w, http.StatusOK, &%s); err != nil {", receiverName)
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusInternalServerError, fmt.Errorf(\"Failed to encode response: %%v\", err))")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")

	g.indent--
	g.writeLine("} else {")
	g.indent++

	// Legacy JSON path
	g.writeLine("// Legacy JSON format")
	g.writeLine("var %s models.%s", receiverName, resource.Name)
	g.writeLine("if err := json.NewDecoder(r.Body).Decode(&%s); err != nil {", receiverName)
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Invalid request body: %%v\", err), http.StatusBadRequest)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("// Set ID from URL")
	g.writeLine("%s.ID = id", receiverName)
	g.writeLine("")

	g.writeLine("// Update %s (includes validation and hooks)", resourceLower)
	g.writeLine("if err := %s.Update(ctx, db); err != nil {", receiverName)
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Failed to update %s: %%v\", err), http.StatusUnprocessableEntity)", resourceLower)
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("w.Header().Set(\"Content-Type\", \"application/json\")")
	g.writeLine("if err := json.NewEncoder(w).Encode(%s); err != nil {", receiverName)
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Failed to encode response: %%v\", err), http.StatusInternalServerError)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")

	g.indent--
	g.writeLine("}")

	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
}

// generateDeleteHandler generates the DELETE handler (DELETE /resources/:id)
func (g *Generator) generateDeleteHandler(resource *ast.ResourceNode) {
	resourceLower := strings.ToLower(resource.Name)
	tableName := g.toTableName(resource.Name)
	receiverName := strings.ToLower(resource.Name[0:1])

	g.writeLine("// Delete%sHandler handles DELETE /%s/{id} - delete a %s",
		resource.Name, tableName, resourceLower)
	g.writeLine("func Delete%sHandler(db *sql.DB) http.HandlerFunc {", resource.Name)
	g.indent++
	g.writeLine("return func(w http.ResponseWriter, r *http.Request) {")
	g.indent++

	g.writeLine("ctx := r.Context()")
	g.writeLine("")

	// Parse ID from URL
	g.generateIDParsingCode(resource)

	// Fetch existing resource
	g.writeLine("// Fetch existing %s", resourceLower)
	g.writeLine("%s, err := models.Find%sByID(ctx, db, id)", receiverName, resource.Name)
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("if err == sql.ErrNoRows {")
	g.indent++
	g.writeLine("if response.IsJSONAPI(r) {")
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusNotFound, fmt.Errorf(\"Not found\"))")
	g.indent--
	g.writeLine("} else {")
	g.indent++
	g.writeLine("respondWithError(w, \"Not found\", http.StatusNotFound)")
	g.indent--
	g.writeLine("}")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("if response.IsJSONAPI(r) {")
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusInternalServerError, fmt.Errorf(\"Failed to find %s: %%v\", err))", resourceLower)
	g.indent--
	g.writeLine("} else {")
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Failed to find %s: %%v\", err), http.StatusInternalServerError)", resourceLower)
	g.indent--
	g.writeLine("}")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Call Delete method (includes hooks)
	g.writeLine("// Delete %s (includes hooks)", resourceLower)
	g.writeLine("if err := %s.Delete(ctx, db); err != nil {", receiverName)
	g.indent++
	g.writeLine("if response.IsJSONAPI(r) {")
	g.indent++
	g.writeLine("response.RenderJSONAPIError(w, http.StatusInternalServerError, fmt.Errorf(\"Failed to delete %s: %%v\", err))", resourceLower)
	g.indent--
	g.writeLine("} else {")
	g.indent++
	g.writeLine("respondWithError(w, fmt.Sprintf(\"Failed to delete %s: %%v\", err), http.StatusInternalServerError)", resourceLower)
	g.indent--
	g.writeLine("}")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Write 204 No Content response (same for both JSON:API and legacy)
	g.writeLine("// Return 204 No Content for both JSON:API and legacy format")
	g.writeLine("w.WriteHeader(http.StatusNoContent)")

	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
}

// generateScanFields generates the list of field pointers for rows.Scan()
func (g *Generator) generateScanFields(resource *ast.ResourceNode) string {
	var scanFields []string

	// Always include ID field first if not explicitly defined
	hasID := false
	for _, field := range resource.Fields {
		if field.Name == "id" {
			hasID = true
			break
		}
	}

	if !hasID {
		scanFields = append(scanFields, "&item.ID")
	}

	// Add all other fields
	for _, field := range resource.Fields {
		fieldName := g.toGoFieldName(field.Name)
		scanFields = append(scanFields, fmt.Sprintf("&item.%s", fieldName))
	}

	return strings.Join(scanFields, ", ")
}
