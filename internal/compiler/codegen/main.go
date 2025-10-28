package codegen

import (
	"github.com/conduit-lang/conduit/internal/compiler/ast"
)

// GenerateMain generates the main.go entry point
func (g *Generator) GenerateMain(resources []*ast.ResourceNode, moduleName string) (string, error) {
	g.reset()

	// Package declaration
	g.writeLine("package main")
	g.writeLine("")

	// Imports
	g.imports["database/sql"] = true
	g.imports["fmt"] = true
	g.imports["log"] = true
	g.imports["net/http"] = true
	g.imports["os"] = true
	g.imports["github.com/go-chi/chi/v5"] = true
	g.imports["github.com/go-chi/chi/v5/middleware"] = true
	g.imports["_ github.com/jackc/pgx/v5/stdlib"] = true // PostgreSQL driver
	g.imports[moduleName+"/handlers"] = true              // Import handlers package

	g.writeImports()
	g.writeLine("")

	// Generate main function
	g.generateMainFunction(resources)

	return g.buf.String(), nil
}

// generateMainFunction generates the main() function
func (g *Generator) generateMainFunction(resources []*ast.ResourceNode) {
	g.writeLine("func main() {")
	g.indent++

	// Database connection
	g.writeLine("// Initialize database connection")
	g.writeLine("db, err := initDB()")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("log.Fatalf(\"Failed to initialize database: %v\", err)")
	g.indent--
	g.writeLine("}")
	g.writeLine("defer db.Close()")
	g.writeLine("")

	// Initialize router
	g.writeLine("// Initialize router")
	g.writeLine("r := chi.NewRouter()")
	g.writeLine("")

	// Add middleware
	g.writeLine("// Add middleware")
	g.writeLine("r.Use(middleware.Logger)")
	g.writeLine("r.Use(middleware.Recoverer)")
	g.writeLine("r.Use(middleware.RequestID)")
	g.writeLine("r.Use(middleware.RealIP)")
	g.writeLine("")

	// Register routes for each resource
	g.writeLine("// Register resource routes")
	for _, resource := range resources {
		g.writeLine("handlers.Register%sRoutes(r, db)", resource.Name)
	}
	g.writeLine("")

	// Health check endpoint
	g.writeLine("// Health check endpoint")
	g.writeLine("r.Get(\"/health\", func(w http.ResponseWriter, r *http.Request) {")
	g.indent++
	g.writeLine("w.WriteHeader(http.StatusOK)")
	g.writeLine("w.Write([]byte(\"OK\"))")
	g.indent--
	g.writeLine("})")
	g.writeLine("")

	// Start server
	g.writeLine("// Start server")
	g.writeLine("port := os.Getenv(\"PORT\")")
	g.writeLine("if port == \"\" {")
	g.indent++
	g.writeLine("port = \"8080\"")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	g.writeLine("addr := fmt.Sprintf(\":%s\", port)")
	g.writeLine("log.Printf(\"Server starting on %s\", addr)")
	g.writeLine("")

	g.writeLine("if err := http.ListenAndServe(addr, r); err != nil {")
	g.indent++
	g.writeLine("log.Fatalf(\"Server failed: %v\", err)")
	g.indent--
	g.writeLine("}")

	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Generate initDB helper function
	g.generateInitDBFunction()
}

// generateInitDBFunction generates the database initialization function
func (g *Generator) generateInitDBFunction() {
	g.writeLine("// initDB initializes the database connection")
	g.writeLine("func initDB() (*sql.DB, error) {")
	g.indent++

	// Get database URL from environment
	g.writeLine("// Get database URL from environment")
	g.writeLine("dbURL := os.Getenv(\"DATABASE_URL\")")
	g.writeLine("if dbURL == \"\" {")
	g.indent++
	g.writeLine("// Default connection string for local development")
	g.writeLine("dbURL = \"postgres://localhost/conduit_dev?sslmode=disable\"")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Open database connection
	g.writeLine("// Open database connection")
	g.writeLine("db, err := sql.Open(\"pgx\", dbURL)")
	g.writeLine("if err != nil {")
	g.indent++
	g.writeLine("return nil, fmt.Errorf(\"failed to open database: %w\", err)")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Test connection
	g.writeLine("// Test connection")
	g.writeLine("if err := db.Ping(); err != nil {")
	g.indent++
	g.writeLine("db.Close()")
	g.writeLine("return nil, fmt.Errorf(\"failed to ping database: %w\", err)")
	g.indent--
	g.writeLine("}")
	g.writeLine("")

	// Set connection pool settings
	g.writeLine("// Set connection pool settings")
	g.writeLine("db.SetMaxOpenConns(25)")
	g.writeLine("db.SetMaxIdleConns(5)")
	g.writeLine("")

	g.writeLine("log.Println(\"Database connection established\")")
	g.writeLine("return db, nil")

	g.indent--
	g.writeLine("}")
}

