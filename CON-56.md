Collect and Generate Route Metadata

🎯 Business Context & Purpose

Problem: Conduit auto-generates REST API routes from resources. Developers need visibility into what HTTP endpoints exist, what middleware is applied, and how routes map to resource operations.

Business Value: Route introspection enables API documentation generation, debugging, and security audits. Target: Generate API docs automatically from route metadata.

User Impact: Developers can see all HTTP endpoints without manually tracking them, crucial for API design and debugging.

📋 Expected Behavior/Outcome

Extend metadata collection to generate route metadata:

Extract resource-level middleware assignments

Generate standard REST routes (list, get, create, update, delete)

Include nested resource routes

Capture middleware chains per route

Store HTTP method, path pattern, handler name, middleware

Route Metadata Schema:

type RouteMetadata struct {
Method string `json:"method"` // "GET", "POST", etc.
Path string `json:"path"` // "/api/posts"
Handler string `json:"handler"` // "Post.list"
Middleware []string `json:"middleware"` // ["cache(300)"]
Resource string `json:"resource"` // "Post"
Operation string `json:"operation"` // "list"
}

✅ Acceptance Criteria

RouteMetadata struct added to schema

Metadata collector generates routes for each resource

Standard REST routes: GET /resources, GET /resources/:id, POST /resources, PUT /resources/:id, DELETE /resources/:id

Nested resource routes: GET /posts/:post_id/comments

Middleware extracted from @on <operation> annotations

Route generation respects @operations restrictions

Routes added to Metadata.Routes array

Unit tests for route generation with various configurations

Integration test: Compile example app, verify routes in metadata

Pragmatic Effort Estimate: 2 days

🔗 Dependencies & Constraints

Dependencies:

Requires metadata schema (CON-48)

Requires metadata collector (CON-49)

Code Reference: See IMPLEMENTATION-RUNTIME.md:212-214 (route generation in collector)
