package templates

// NewAPITemplate creates the official API template
func NewAPITemplate() *Template {
	return &Template{
		Name:        "api",
		Description: "RESTful API with resources and authentication",
		Version:     "1.0.0",
		Variables: []*TemplateVariable{
			{
				Name:        "project_name",
				Description: "Name of your API project",
				Type:        VariableTypeString,
				Required:    true,
				Prompt:      "Project name",
			},
			{
				Name:        "port",
				Description: "Server port",
				Type:        VariableTypeInt,
				Default:     3000,
				Prompt:      "Server port",
			},
			{
				Name:        "include_auth",
				Description: "Include authentication resources",
				Type:        VariableTypeConfirm,
				Default:     true,
				Prompt:      "Include authentication?",
			},
			{
				Name:        "database_url",
				Description: "Database connection URL",
				Type:        VariableTypeString,
				Default:     "",
				Prompt:      "Database URL (optional, can use env var)",
			},
		},
		Directories: []string{
			"app",
			"app/resources",
			"migrations",
			"build",
			"config",
		},
		Files: []*TemplateFile{
			{
				TargetPath: "CLAUDE.md",
				Template:   true,
				Content:    GetCLAUDEMDContent(),
			},
			{
				TargetPath: "app/resources/user.cdt",
				Template:   true,
				Content: `/// User resource for authentication
resource User {
  id: uuid! @primary @auto
  email: string! @unique @min(5) @max(255)
  password_hash: string! @min(60) @max(255)
  name: string! @min(2) @max(100)
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  @constraint valid_email {
    on: [create, update]
    condition: String.contains(self.email, "@")
    error: "Email must be valid"
  }
}
`,
				Condition: "{{.Variables.include_auth}}",
			},
			{
				TargetPath: "app/resources/post.cdt",
				Template:   true,
				Content: `/// Blog post resource
resource Post {
  id: uuid! @primary @auto
  title: string! @min(5) @max(200)
  slug: string! @unique
  content: text! @min(100)
  published: bool! @default(false)
  author_id: uuid!
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  @before create {
    self.slug = String.slugify(self.title)
  }

  @constraint published_requires_content {
    on: [create, update]
    when: self.published == true
    condition: String.length(self.content) >= 500
    error: "Published posts must have at least 500 characters"
  }
}
`,
			},
			{
				TargetPath: "conduit.yaml",
				Template:   true,
				Content: `# Conduit Configuration
project:
  name: {{.ProjectName}}
  version: 1.0.0

server:
  port: {{.Variables.port}}
  host: "0.0.0.0"

database:
  driver: postgresql
  {{- if .Variables.database_url}}
  url: "{{.Variables.database_url}}"
  {{- else}}
  # Set via DATABASE_URL environment variable
  {{- end}}
  pool_size: 10
  max_idle_time: 300

logging:
  level: info
  format: json

cors:
  enabled: true
  allowed_origins:
    - "*"
  allowed_methods:
    - GET
    - POST
    - PUT
    - PATCH
    - DELETE
  allowed_headers:
    - Content-Type
    - Authorization
`,
			},
			{
				TargetPath: ".gitignore",
				Template:   false,
				Content: `# Build output
build/
*.exe
*.dll
*.so
*.dylib

# Test binaries
*.test
*.out

# Dependencies
vendor/

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Environment
.env
.env.local

# Database
*.db
*.sqlite
`,
			},
			{
				TargetPath: "README.md",
				Template:   true,
				Content: `# {{.ProjectName}}

A Conduit API project.

## Getting Started

### Prerequisites

- Conduit CLI installed
- PostgreSQL 15+ running

### Setup

1. Set up your database:
   ` + "```bash" + `
   export DATABASE_URL="postgresql://user:password@localhost:5432/{{.ProjectName}}"
   ` + "```" + `

2. Run migrations:
   ` + "```bash" + `
   conduit migrate up
   ` + "```" + `

3. Build and run:
   ` + "```bash" + `
   conduit run
   ` + "```" + `

Your API will be available at http://localhost:{{.Variables.port}}

## API Endpoints

### Posts

- ` + "`GET /api/posts`" + ` - List all posts
- ` + "`GET /api/posts/:id`" + ` - Get a post by ID
- ` + "`POST /api/posts`" + ` - Create a new post
- ` + "`PUT /api/posts/:id`" + ` - Update a post
- ` + "`DELETE /api/posts/:id`" + ` - Delete a post
{{if .Variables.include_auth}}
### Users

- ` + "`GET /api/users`" + ` - List all users
- ` + "`GET /api/users/:id`" + ` - Get a user by ID
- ` + "`POST /api/users`" + ` - Create a new user
- ` + "`PUT /api/users/:id`" + ` - Update a user
- ` + "`DELETE /api/users/:id`" + ` - Delete a user
{{end}}
## Project Structure

- ` + "`app/resources/`" + ` - Conduit resource definitions
- ` + "`migrations/`" + ` - Database migrations
- ` + "`build/`" + ` - Compiled output (auto-generated)
- ` + "`conduit.yaml`" + ` - Project configuration

## Documentation

Learn more at https://conduit-lang.org/docs
`,
			},
			{
				TargetPath: ".env.example",
				Template:   false,
				Content: `# Database
DATABASE_URL=postgresql://user:password@localhost:5432/dbname

# Server
PORT=3000

# Application
LOG_LEVEL=info
`,
			},
		},
		Hooks: &TemplateHooks{
			AfterCreate: []string{
				"echo 'Project created successfully!'",
				"echo 'Next steps:'",
				"echo '  1. Set DATABASE_URL environment variable'",
				"echo '  2. Run: conduit migrate up'",
				"echo '  3. Run: conduit run'",
			},
		},
		Metadata: map[string]interface{}{
			"category": "backend",
			"tags":     []string{"api", "rest", "backend"},
		},
	}
}
