package templates

// NewWebTemplate creates the official web application template
func NewWebTemplate() *Template {
	return &Template{
		Name:        "web",
		Description: "Full-stack web application with frontend and backend",
		Version:     "1.0.0",
		Variables: []*TemplateVariable{
			{
				Name:        "project_name",
				Description: "Name of your web application",
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
				Description: "Include user authentication",
				Type:        VariableTypeConfirm,
				Default:     true,
				Prompt:      "Include authentication?",
			},
			{
				Name:        "include_admin",
				Description: "Include admin dashboard",
				Type:        VariableTypeConfirm,
				Default:     false,
				Prompt:      "Include admin dashboard?",
			},
			{
				Name:        "database_url",
				Description: "Database connection URL",
				Type:        VariableTypeString,
				Default:     "",
				Prompt:      "Database URL (optional)",
			},
		},
		Directories: []string{
			"app",
			"app/resources",
			"app/views",
			"migrations",
			"build",
			"public",
			"public/css",
			"public/js",
			"public/images",
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
				Content: `/// User resource with authentication
resource User {
  id: uuid! @primary @auto
  email: string! @unique @min(5) @max(255)
  password_hash: string! @min(60) @max(255)
  name: string! @min(2) @max(100)
  role: string! @default("user")
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  posts: [Post] {
    foreign_key: "author_id"
  }

  @constraint valid_email {
    on: [create, update]
    condition: String.contains(self.email, "@")
    error: "Email must be valid"
  }

  @constraint valid_role {
    on: [create, update]
    condition: self.role == "user" || self.role == "admin"
    error: "Role must be 'user' or 'admin'"
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
  excerpt: text?
  published: bool! @default(false)
  published_at: timestamp?
  author_id: uuid!
  view_count: int! @default(0)
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }

  comments: [Comment] {
    foreign_key: "post_id"
  }

  @before create {
    self.slug = String.slugify(self.title)
    if String.length(self.content) > 200 && self.excerpt == null {
      self.excerpt = String.substring(self.content, 0, 200)
    }
  }

  @before update {
    if self.published == true && self.published_at == null {
      self.published_at = Time.now()
    }
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
				TargetPath: "app/resources/comment.cdt",
				Template:   true,
				Content: `/// Comment resource for posts
resource Comment {
  id: uuid! @primary @auto
  post_id: uuid!
  author_id: uuid!
  content: text! @min(10) @max(2000)
  approved: bool! @default(false)
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  post: Post! {
    foreign_key: "post_id"
    on_delete: cascade
  }

  author: User! {
    foreign_key: "author_id"
    on_delete: cascade
  }

  @constraint no_spam {
    on: [create, update]
    condition: !String.contains(String.lower(self.content), "http://") && !String.contains(String.lower(self.content), "https://")
    error: "Comments cannot contain URLs"
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
  static_dir: "public"
  views_dir: "app/views"

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

session:
  secret: "change-this-in-production"
  max_age: 86400
  secure: false
  http_only: true

cors:
  enabled: true
  allowed_origins:
    - "http://localhost:{{.Variables.port}}"
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
				TargetPath: "public/css/style.css",
				Template:   false,
				Content: `/* {{.ProjectName}} Styles */

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  line-height: 1.6;
  color: #333;
  background: #f4f4f4;
}

.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

header {
  background: #333;
  color: #fff;
  padding: 1rem 0;
  margin-bottom: 2rem;
}

header h1 {
  margin: 0;
}

.post {
  background: #fff;
  padding: 20px;
  margin-bottom: 20px;
  border-radius: 5px;
  box-shadow: 0 2px 5px rgba(0,0,0,0.1);
}

.post h2 {
  color: #333;
  margin-bottom: 10px;
}

.post-meta {
  color: #666;
  font-size: 0.9em;
  margin-bottom: 15px;
}

.btn {
  display: inline-block;
  padding: 10px 20px;
  background: #333;
  color: #fff;
  text-decoration: none;
  border-radius: 3px;
  border: none;
  cursor: pointer;
}

.btn:hover {
  background: #555;
}
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

# Uploads
public/uploads/
`,
			},
			{
				TargetPath: "README.md",
				Template:   true,
				Content: `# {{.ProjectName}}

A Conduit web application.

## Features

- Blog posts with comments
{{- if .Variables.include_auth}}
- User authentication and authorization
{{- end}}
{{- if .Variables.include_admin}}
- Admin dashboard
{{- end}}
- Responsive design
- RESTful API

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

4. Visit http://localhost:{{.Variables.port}}

## Project Structure

- ` + "`app/resources/`" + ` - Conduit resource definitions
- ` + "`app/views/`" + ` - View templates
- ` + "`public/`" + ` - Static assets (CSS, JS, images)
- ` + "`migrations/`" + ` - Database migrations
- ` + "`build/`" + ` - Compiled output (auto-generated)
- ` + "`conduit.yaml`" + ` - Project configuration

## Documentation

Learn more at https://conduit-lang.org/docs
`,
			},
		},
		Hooks: &TemplateHooks{
			AfterCreate: []string{
				"echo 'Web application created successfully!'",
				"echo 'Next steps:'",
				"echo '  1. Set DATABASE_URL environment variable'",
				"echo '  2. Run: conduit migrate up'",
				"echo '  3. Run: conduit run'",
				"echo '  4. Visit http://localhost:3000'",
			},
		},
		Metadata: map[string]interface{}{
			"category": "fullstack",
			"tags":     []string{"web", "fullstack", "blog"},
		},
	}
}
