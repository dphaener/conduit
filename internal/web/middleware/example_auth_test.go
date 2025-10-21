package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/conduit-lang/conduit/internal/web/auth"
	"github.com/conduit-lang/conduit/internal/web/middleware"
	"github.com/go-chi/chi/v5"
)

// Example_authenticationAndAuthorization demonstrates a complete authentication and authorization flow
func Example_authenticationAndAuthorization() {
	// Setup: Create auth service and generate tokens
	authService := auth.NewAuthService("secret-key", time.Hour)

	// Admin user with full permissions
	adminToken, _ := authService.GenerateToken("admin-123", "admin@example.com", []string{"admin"})

	// Editor user with limited permissions
	editorToken, _ := authService.GenerateToken("editor-456", "editor@example.com", []string{"editor"})

	// Viewer user with read-only permissions
	viewerToken, _ := authService.GenerateToken("viewer-789", "viewer@example.com", []string{"viewer"})

	// Create router with authentication middleware
	r := chi.NewRouter()

	// Apply authentication middleware globally
	r.Use(middleware.Auth(authService))

	// Public endpoint (no authorization required, but authentication is still needed)
	r.Get("/api/posts", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "List of posts")
	})

	// Create posts - requires posts.create permission (admin and editor)
	r.With(middleware.RequirePermission(auth.PostsCreate)).
		Post("/api/posts", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Post created")
		})

	// Delete posts - requires posts.delete permission (admin only)
	r.With(middleware.RequirePermission(auth.PostsDelete)).
		Delete("/api/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Post deleted")
		})

	// Admin-only endpoint
	r.With(middleware.RequireRole("admin")).
		Get("/api/admin/stats", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Admin statistics")
		})

	// Test 1: Viewer can read posts
	req1 := httptest.NewRequest("GET", "/api/posts", nil)
	req1.Header.Set("Authorization", "Bearer "+viewerToken)
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, req1)
	fmt.Printf("Viewer reading posts: %d\n", rr1.Code)

	// Test 2: Editor can create posts
	req2 := httptest.NewRequest("POST", "/api/posts", nil)
	req2.Header.Set("Authorization", "Bearer "+editorToken)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	fmt.Printf("Editor creating post: %d\n", rr2.Code)

	// Test 3: Viewer cannot create posts
	req3 := httptest.NewRequest("POST", "/api/posts", nil)
	req3.Header.Set("Authorization", "Bearer "+viewerToken)
	rr3 := httptest.NewRecorder()
	r.ServeHTTP(rr3, req3)
	fmt.Printf("Viewer creating post: %d\n", rr3.Code)

	// Test 4: Editor cannot delete posts
	req4 := httptest.NewRequest("DELETE", "/api/posts/1", nil)
	req4.Header.Set("Authorization", "Bearer "+editorToken)
	rr4 := httptest.NewRecorder()
	r.ServeHTTP(rr4, req4)
	fmt.Printf("Editor deleting post: %d\n", rr4.Code)

	// Test 5: Admin can delete posts
	req5 := httptest.NewRequest("DELETE", "/api/posts/1", nil)
	req5.Header.Set("Authorization", "Bearer "+adminToken)
	rr5 := httptest.NewRecorder()
	r.ServeHTTP(rr5, req5)
	fmt.Printf("Admin deleting post: %d\n", rr5.Code)

	// Test 6: Editor cannot access admin endpoints
	req6 := httptest.NewRequest("GET", "/api/admin/stats", nil)
	req6.Header.Set("Authorization", "Bearer "+editorToken)
	rr6 := httptest.NewRecorder()
	r.ServeHTTP(rr6, req6)
	fmt.Printf("Editor accessing admin stats: %d\n", rr6.Code)

	// Test 7: Admin can access admin endpoints
	req7 := httptest.NewRequest("GET", "/api/admin/stats", nil)
	req7.Header.Set("Authorization", "Bearer "+adminToken)
	rr7 := httptest.NewRecorder()
	r.ServeHTTP(rr7, req7)
	fmt.Printf("Admin accessing admin stats: %d\n", rr7.Code)

	// Output:
	// Viewer reading posts: 200
	// Editor creating post: 200
	// Viewer creating post: 403
	// Editor deleting post: 403
	// Admin deleting post: 200
	// Editor accessing admin stats: 403
	// Admin accessing admin stats: 200
}
