Implement 'conduit introspect routes' Command

🎯 Business Context & Purpose

Problem: Developers need to see all HTTP endpoints in their application, especially when debugging API issues or planning new endpoints.

Business Value: Route visibility reduces "what endpoints do we have?" questions in code review and enables automated API documentation.

User Impact: Developers can instantly see all routes, filter by method or middleware, and understand the complete API surface.

📋 Expected Behavior/Outcome

Command: conduit introspect routes

Default Output:

GET /api/posts -> Post.list [cache(300)]
GET /api/posts/:id -> Post.get [cache(600)]
POST /api/posts -> Post.create [auth, rate_limit(5/hour)]
PUT /api/posts/:id -> Post.update [auth, author_or_editor]
DELETE /api/posts/:id -> Post.delete [auth, author_or_admin]

GET /api/posts/:post_id/comments -> Comment.list []
POST /api/posts/:post_id/comments -> Comment.create [auth]

Filtering:

--method GET - Show only GET routes

--method POST - Show only POST routes

--middleware auth - Show routes using auth middleware

--resource Post - Show routes for specific resource

✅ Acceptance Criteria

Command conduit introspect routes implemented

Shows all routes in tabular format

Columns: Method, Path, Handler, Middleware

--method <METHOD> flag filters by HTTP method

--middleware <NAME> flag filters by middleware

--resource <NAME> flag filters by resource

--format json outputs structured JSON

Routes sorted by path alphabetically

Color-coded output (GET=green, POST=blue, DELETE=red)

Response time: <100ms

Unit tests for filtering logic

Integration test with real registry

Pragmatic Effort Estimate: 2 days

🔗 Dependencies & Constraints

Dependencies:

Requires CLI structure (CON-53)

Requires route metadata (previous ticket)

Requires runtime registry (CON-52)

Code Reference: See IMPLEMENTATION-RUNTIME.md:863-879
