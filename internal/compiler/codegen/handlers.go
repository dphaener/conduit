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
	g.imports["fmt"] = true
	g.imports["net/http"] = true
	g.imports["strconv"] = true
	g.imports["github.com/go-chi/chi/v5"] = true
	g.imports[moduleName+"/models"] = true // Import models package

	// Pre-scan resources for additional imports (like uuid for ID types)
	for _, resource := range resources {
		if g.getIDType(resource) == "uuid" {
			g.imports["github.com/google/uuid"] = true
		}
	}

	g.writeImports()
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
		g.writeLine("http.Error(w, \"Invalid ID\", http.StatusBadRequest)")
		g.writeLine("return")
		g.indent--
		g.writeLine("}")
	default: // int, int64, etc.
		g.writeLine("id, err := strconv.ParseInt(idStr, 10, 64)")
		g.writeLine("if err != nil {")
		g.indent++
		g.writeLine("http.Error(w, \"Invalid ID\", http.StatusBadRequest)")
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

	// Call FindAll function
	g.writeLine("results, err := models.FindAll%s(ctx, db, limit, offset)", resource.Name)
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("http.Error(w, fmt.Sprintf(\"Failed to list %s: %%v\", err), http.StatusInternalServerError)", resourceLower+"s")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Write JSON response
	g.writeLine("w.Header().Set(\"Content-Type\", \"application/json\")")
	g.writeLine("if err := json.NewEncoder(w).Encode(results); err != nil {")
	g.indent++
	g.writeLine("http.Error(w, fmt.Sprintf(\"Failed to encode response: %%v\", err), http.StatusInternalServerError)")
	g.writeLine("return")
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
	g.writeLine("http.Error(w, \"Not found\", http.StatusNotFound)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("http.Error(w, fmt.Sprintf(\"Failed to get %s: %%v\", err), http.StatusInternalServerError)", resourceLower)
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Write JSON response
	g.writeLine("w.Header().Set(\"Content-Type\", \"application/json\")")
	g.writeLine("if err := json.NewEncoder(w).Encode(result); err != nil {")
	g.indent++
	g.writeLine("http.Error(w, fmt.Sprintf(\"Failed to encode response: %%v\", err), http.StatusInternalServerError)")
	g.writeLine("return")
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

	// Decode JSON request body
	g.writeLine("// Decode request body")
	g.writeLine("var %s models.%s", receiverName, resource.Name)
	g.writeLine("if err := json.NewDecoder(r.Body).Decode(&%s); err != nil {", receiverName)
	g.indent++
	g.writeLine("http.Error(w, fmt.Sprintf(\"Invalid request body: %%v\", err), http.StatusBadRequest)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Call Create method (includes hooks and validation)
	g.writeLine("// Create %s (includes validation and hooks)", resourceLower)
	g.writeLine("if err := %s.Create(ctx, db); err != nil {", receiverName)
	g.indent++
	g.writeLine("http.Error(w, fmt.Sprintf(\"Failed to create %s: %%v\", err), http.StatusUnprocessableEntity)", resourceLower)
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Write JSON response with 201 Created
	g.writeLine("w.Header().Set(\"Content-Type\", \"application/json\")")
	g.writeLine("w.WriteHeader(http.StatusCreated)")
	g.writeLine("if err := json.NewEncoder(w).Encode(%s); err != nil {", receiverName)
	g.indent++
	g.writeLine("http.Error(w, fmt.Sprintf(\"Failed to encode response: %%v\", err), http.StatusInternalServerError)")
	g.writeLine("return")
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

	// Decode JSON request body
	g.writeLine("// Decode request body")
	g.writeLine("var %s models.%s", receiverName, resource.Name)
	g.writeLine("if err := json.NewDecoder(r.Body).Decode(&%s); err != nil {", receiverName)
	g.indent++
	g.writeLine("http.Error(w, fmt.Sprintf(\"Invalid request body: %%v\", err), http.StatusBadRequest)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Set ID from URL
	g.writeLine("// Set ID from URL")
	g.writeLine("%s.ID = id", receiverName)
	g.writeLine("")

	// Call Update method (includes hooks and validation)
	g.writeLine("// Update %s (includes validation and hooks)", resourceLower)
	g.writeLine("if err := %s.Update(ctx, db); err != nil {", receiverName)
	g.indent++
	g.writeLine("http.Error(w, fmt.Sprintf(\"Failed to update %s: %%v\", err), http.StatusUnprocessableEntity)", resourceLower)
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Write JSON response
	g.writeLine("w.Header().Set(\"Content-Type\", \"application/json\")")
	g.writeLine("if err := json.NewEncoder(w).Encode(%s); err != nil {", receiverName)
	g.indent++
	g.writeLine("http.Error(w, fmt.Sprintf(\"Failed to encode response: %%v\", err), http.StatusInternalServerError)")
	g.writeLine("return")
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
	g.writeLine("http.Error(w, \"Not found\", http.StatusNotFound)")
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("http.Error(w, fmt.Sprintf(\"Failed to find %s: %%v\", err), http.StatusInternalServerError)", resourceLower)
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Call Delete method (includes hooks)
	g.writeLine("// Delete %s (includes hooks)", resourceLower)
	g.writeLine("if err := %s.Delete(ctx, db); err != nil {", receiverName)
	g.indent++
	g.writeLine("http.Error(w, fmt.Sprintf(\"Failed to delete %s: %%v\", err), http.StatusInternalServerError)", resourceLower)
	g.writeLine("return")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Write 204 No Content response
	g.writeLine("w.WriteHeader(http.StatusNoContent)")

	g.indent--
	g.writeLine("}")
	g.indent--
	g.writeLine("}")
}
