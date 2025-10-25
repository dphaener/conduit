package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/conduit-lang/conduit/internal/web/middleware"
	"github.com/conduit-lang/conduit/internal/web/profiling"
	"github.com/conduit-lang/conduit/internal/web/router"
	"github.com/conduit-lang/conduit/internal/web/server"
	"github.com/conduit-lang/conduit/internal/web/static"
	"github.com/conduit-lang/conduit/internal/web/stream"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	// Setup database connection
	db, err := setupDatabase()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create router
	r := router.NewRouter()

	// Add middleware stack
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery())
	r.Use(middleware.Logging())
	r.Use(middleware.Compression())
	r.Use(middleware.Timeout(30 * time.Second))

	// Register profiling endpoints
	profiling.EnableProfilingHTTP(profiling.WrapRouter(func(pattern string, handler http.HandlerFunc) {
		r.Get(pattern, handler)
	}))

	// API routes
	r.Get("/api/posts", listPostsHandler(db))
	r.Get("/api/posts/stream", streamPostsHandler(db))
	r.Get("/health", healthHandler)

	// Static file serving
	staticHandler := static.NewFileServer("./public", "/static")
	r.Get("/static/*", func(w http.ResponseWriter, r *http.Request) {
		staticHandler.ServeHTTP(w, r)
	})

	// Create optimized server configuration
	config := &server.Config{
		Address:           ":8080",
		Handler:           r,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
		EnableHTTP2:       true,
		Database: &server.DatabaseConfig{
			DB:              db,
			MaxOpenConns:    100,
			MaxIdleConns:    10,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
		},
	}

	srv, err := server.New(config)
	if err != nil {
		log.Fatal(err)
	}

	// Setup graceful shutdown with cleanup hooks
	shutdownConfig := server.DefaultShutdownConfig()
	gs := server.NewGracefulShutdown(srv, shutdownConfig)

	// Register cleanup hooks
	gs.RegisterHook(func(ctx context.Context) error {
		log.Println("Closing database connections...")
		return db.Close()
	})

	// Start server
	log.Printf("Starting optimized server on %s", config.Address)
	log.Printf("Profiling available at http://localhost:8080/debug/pprof/")
	if err := gs.Start(); err != nil {
		log.Fatal(err)
	}
}

func setupDatabase() (*sql.DB, error) {
	db, err := sql.Open("pgx", "postgres://localhost/conduit?sslmode=disable")
	if err != nil {
		return nil, err
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func listPostsHandler(db *sql.DB) http.HandlerFunc {
	type Post struct {
		ID      int    `json:"id"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, title, content FROM posts LIMIT 100")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		posts := make([]Post, 0)
		for rows.Next() {
			var post Post
			if err := rows.Scan(&post.ID, &post.Title, &post.Content); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			posts = append(posts, post)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(posts)
	}
}

func streamPostsHandler(db *sql.DB) http.HandlerFunc {
	type Post struct {
		ID      int    `json:"id"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		streamer, err := stream.NewJSON(w)
		if err != nil {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		rows, err := db.Query("SELECT id, title, content FROM posts")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Create channel for streaming
		posts := make(chan interface{}, 10)

		go func() {
			defer close(posts)
			for rows.Next() {
				var post Post
				if err := rows.Scan(&post.ID, &post.Title, &post.Content); err != nil {
					return
				}
				posts <- post
			}
		}()

		// Stream JSON array
		streamer.WriteJSONArray(posts)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
}
