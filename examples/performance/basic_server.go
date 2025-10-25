package main

import (
	"log"
	"net/http"

	"github.com/conduit-lang/conduit/internal/web/middleware"
	"github.com/conduit-lang/conduit/internal/web/router"
	"github.com/conduit-lang/conduit/internal/web/server"
)

func main() {
	// Create router
	r := router.NewRouter()

	// Add middleware
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery())
	r.Use(middleware.Logging())

	// Add routes
	r.Get("/", homeHandler)
	r.Get("/health", healthHandler)

	// Create server with default config
	config := server.DefaultConfig(r)
	config.Address = ":8080"

	srv, err := server.New(config)
	if err != nil {
		log.Fatal(err)
	}

	// Start server with graceful shutdown
	log.Printf("Starting server on %s", config.Address)
	if err := server.StartWithGracefulShutdown(srv, nil); err != nil {
		log.Fatal(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<h1>Welcome to Conduit</h1>"))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
