package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// scaffoldTemplates defines the available scaffold templates
var scaffoldTemplates = map[string]struct {
	description string
	generate    func(string) error
}{
	"todo": {
		description: "Minimal CRUD resource (single Todo resource)",
		generate:    generateTodoScaffold,
	},
	"blog": {
		description: "Post + User + Comment resources (demonstrates relationships)",
		generate:    generateBlogScaffold,
	},
	"api": {
		description: "Resource with auth patterns",
		generate:    generateAPIScaffold,
	},
}

// NewScaffoldCommand creates the scaffold command
func NewScaffoldCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scaffold <template>",
		Short: "Generate working resource files from templates",
		Long: `Generate working resource files from templates to help you get started quickly.

Available templates:
  todo  - Minimal CRUD resource (single Todo resource with hooks)
  blog  - Post + User + Comment resources (demonstrates relationships)
  api   - Resource with auth patterns

The scaffold command will:
  - Verify your project is initialized (conduit.yaml exists)
  - Check for existing files to prevent overwriting
  - Create app/resources/ directory if needed
  - Generate resource files with helpful comments
  - Show next steps for building and running

Examples:
  conduit scaffold todo
  conduit scaffold blog
  conduit scaffold api`,
		Args: cobra.ExactArgs(1),
		RunE: runScaffold,
	}

	return cmd
}

func runScaffold(cmd *cobra.Command, args []string) error {
	templateName := strings.ToLower(args[0])

	successColor := color.New(color.FgGreen, color.Bold)
	infoColor := color.New(color.FgCyan)
	errorColor := color.New(color.FgRed, color.Bold)
	promptColor := color.New(color.FgYellow)

	// Check if template exists
	template, exists := scaffoldTemplates[templateName]
	if !exists {
		errorColor.Printf("✗ Unknown template: %s\n\n", templateName)
		fmt.Println("Available templates:")
		for name, tmpl := range scaffoldTemplates {
			fmt.Printf("  %-8s - %s\n", name, tmpl.description)
		}
		return fmt.Errorf("unknown template: %s", templateName)
	}

	// Verify project is initialized (conduit.yaml exists)
	if _, err := os.Stat("conduit.yaml"); os.IsNotExist(err) {
		errorColor.Println("✗ Not in a Conduit project directory")
		fmt.Println("\nThis command must be run in a directory with conduit.yaml")
		fmt.Println("Create a new project with: conduit new <project-name>")
		return fmt.Errorf("conduit.yaml not found")
	}

	// Create app/resources directory if it doesn't exist
	resourcesDir := filepath.Join("app", "resources")
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		return fmt.Errorf("failed to create resources directory: %w", err)
	}

	infoColor.Printf("Generating %s scaffold...\n\n", templateName)

	// Generate the scaffold
	if err := template.generate(resourcesDir); err != nil {
		return fmt.Errorf("failed to generate scaffold: %w", err)
	}

	// Print success message
	fmt.Println()
	successColor.Printf("✓ Generated %s scaffold successfully\n\n", templateName)

	promptColor.Println("Next steps:")
	fmt.Println("  1. Review the generated files in app/resources/")
	fmt.Println("  2. Build your application: conduit build")
	fmt.Println("  3. Run migrations: conduit migrate up")
	fmt.Println("  4. Start the server: conduit run")
	fmt.Println()

	return nil
}

// generateTodoScaffold generates a minimal Todo resource
func generateTodoScaffold(resourcesDir string) error {
	files := map[string]string{
		"todo.cdt": getTodoTemplate(),
	}

	return writeScaffoldFiles(resourcesDir, files)
}

// generateBlogScaffold generates Post, User, and Comment resources
func generateBlogScaffold(resourcesDir string) error {
	files := map[string]string{
		"user.cdt":    getUserTemplate(),
		"post.cdt":    getPostTemplate(),
		"comment.cdt": getCommentTemplate(),
	}

	return writeScaffoldFiles(resourcesDir, files)
}

// generateAPIScaffold generates API resources with auth patterns
func generateAPIScaffold(resourcesDir string) error {
	files := map[string]string{
		"user.cdt":    getAPIUserTemplate(),
		"api_key.cdt": getAPIKeyTemplate(),
	}

	return writeScaffoldFiles(resourcesDir, files)
}

// writeScaffoldFiles writes the scaffold files to disk, checking for conflicts
func writeScaffoldFiles(resourcesDir string, files map[string]string) error {
	infoColor := color.New(color.FgCyan)
	warningColor := color.New(color.FgYellow)

	// First, check for existing files
	var conflicts []string
	for filename := range files {
		filePath := filepath.Join(resourcesDir, filename)
		if _, err := os.Stat(filePath); err == nil {
			conflicts = append(conflicts, filename)
		}
	}

	if len(conflicts) > 0 {
		warningColor.Println("⚠ The following files already exist:")
		for _, filename := range conflicts {
			fmt.Printf("    %s\n", filename)
		}
		fmt.Println("\nTo avoid overwriting, scaffold generation cancelled.")
		fmt.Println("Remove or rename existing files before running scaffold.")
		return fmt.Errorf("files already exist: %v", conflicts)
	}

	// Write all files
	for filename, content := range files {
		filePath := filepath.Join(resourcesDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
		infoColor.Printf("  ✓ Created %s\n", filepath.Join("app", "resources", filename))
	}

	return nil
}

// getTodoTemplate returns the Todo resource template
func getTodoTemplate() string {
	return `// Generated by: conduit scaffold todo
// This is a minimal CRUD resource demonstrating the basics of Conduit

/// Todo item with validation and lifecycle hooks
///
/// This resource demonstrates:
/// - Field validation with @min/@max
/// - Default values with @default
/// - Optional fields with ?
/// - Lifecycle hooks for automatic slug generation
/// - Auto-updating timestamps
resource Todo {
  // Primary key - auto-generated UUID
  // @primary marks this as the primary key
  // @auto tells Conduit to generate the value automatically
  id: uuid! @primary @auto

  // Title with length validation
  // Must be between 3 and 200 characters
  // The ! means required (cannot be null)
  title: string! @min(3) @max(200)

  // Unique slug for URL-friendly identifiers
  // Automatically generated from title in @before hook below
  // @unique ensures no two todos can have the same slug
  slug: string! @unique

  // Optional description
  // The ? means nullable (can be omitted)
  description: string?

  // Status field with default value
  // Defaults to "pending" for new todos
  status: string! @default("pending")

  // Priority level (1-5, where 5 is highest)
  // Defaults to 3 (medium priority)
  priority: int! @default(3)

  // Completed flag
  // Defaults to false for new todos
  completed: bool! @default(false)

  // Optional due date
  // Can be null if todo has no deadline
  due_date: timestamp?

  // Automatic timestamps
  // created_at: Set once when record is created
  // updated_at: Automatically updated every time the record changes
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  // Lifecycle hook - runs before creating a new todo
  // Automatically generates a URL-friendly slug from the title
  // Example: "Write Documentation" becomes "write-documentation"
  //
  // IMPORTANT: All standard library functions are namespaced
  // Use String.slugify(), NOT slugify()
  @before create {
    self.slug = String.slugify(self.title)
  }

  // Lifecycle hook - runs before updating a todo
  // Updates the slug if the title changed
  @before update {
    self.slug = String.slugify(self.title)
  }
}
`
}

// getUserTemplate returns the User resource template
func getUserTemplate() string {
	return `// Generated by: conduit scaffold blog
// User resource for blog authors

/// User resource representing blog authors
///
/// This resource demonstrates:
/// - Email validation and uniqueness
/// - Password storage (hash only)
/// - Relationships (one user has many posts)
/// - Simple resource without lifecycle hooks
resource User {
  // Primary key
  id: uuid! @primary @auto

  // Email must be unique across all users
  // @unique creates a database index and enforces uniqueness
  email: string! @unique @min(5) @max(255)

  // Store password hash, never plain text
  // In production, hash with bcrypt before storing
  password_hash: string! @min(60) @max(255)

  // User's display name
  name: string! @min(2) @max(100)

  // Optional bio/profile text
  bio: text?

  // Timestamps
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update
}
`
}

// getPostTemplate returns the Post resource template
func getPostTemplate() string {
	return `// Generated by: conduit scaffold blog
// Post resource with author relationship

/// Blog post resource
///
/// This resource demonstrates:
/// - belongs_to relationship (Post belongs to User)
/// - URL-friendly slugs
/// - Published/draft state
/// - Content length validation
resource Post {
  // Primary key
  id: uuid! @primary @auto

  // Post title with length constraints
  title: string! @min(5) @max(200)

  // URL-friendly slug (auto-generated from title)
  slug: string! @unique

  // Post content (long text)
  content: text! @min(100)

  // Published state (defaults to draft)
  published: bool! @default(false)

  // Foreign key to User (author)
  // Note: We store the UUID directly
  author_id: uuid!

  // Timestamps
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  // Relationship: Post belongs to User
  // This creates an association between Post and User
  // foreign_key: specifies the field holding the user ID
  // on_delete: restrict prevents deleting users with posts
  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }
}
`
}

// getCommentTemplate returns the Comment resource template
func getCommentTemplate() string {
	return `// Generated by: conduit scaffold blog
// Comment resource with relationships

/// Comment resource
///
/// This resource demonstrates:
/// - Multiple belongs_to relationships
/// - Comments belong to both Post and User
/// - Optional relationships (nullable)
resource Comment {
  // Primary key
  id: uuid! @primary @auto

  // Comment text
  content: text! @min(1) @max(2000)

  // Foreign keys
  post_id: uuid!
  author_id: uuid!

  // Timestamps
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  // Relationship: Comment belongs to Post
  post: Post! {
    foreign_key: "post_id"
    on_delete: cascade
  }

  // Relationship: Comment belongs to User (author)
  author: User! {
    foreign_key: "author_id"
    on_delete: restrict
  }
}
`
}

// getAPIUserTemplate returns the User resource for API template
func getAPIUserTemplate() string {
	return `// Generated by: conduit scaffold api
// User resource with API authentication patterns

/// User resource for API authentication
///
/// This resource demonstrates:
/// - Email validation and uniqueness
/// - Password hashing (store hash only)
/// - Account status tracking
/// - Rate limiting fields
resource User {
  // Primary key
  id: uuid! @primary @auto

  // Email address (unique login identifier)
  email: string! @unique @min(5) @max(255)

  // Password hash (never store plain text passwords)
  // Hash with bcrypt/argon2 before storing
  password_hash: string! @min(60) @max(255)

  // User's display name
  name: string! @min(2) @max(100)

  // Account status
  // Values: "active", "suspended", "deleted"
  status: string! @default("active")

  // Email verification
  email_verified: bool! @default(false)
  email_verified_at: timestamp?

  // API rate limiting
  // Track requests for rate limiting
  last_api_call: timestamp?
  api_calls_count: int! @default(0)

  // Timestamps
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  // Reset API call counter daily
  @before update {
    // In a real app, you'd check if 24 hours passed
    // and reset api_calls_count if needed
  }
}
`
}

// getAPIKeyTemplate returns the APIKey resource template
func getAPIKeyTemplate() string {
	return `// Generated by: conduit scaffold api
// API Key resource for token-based authentication

/// API Key resource for authentication
///
/// This resource demonstrates:
/// - API key management
/// - Token expiration
/// - Usage tracking
/// - Scopes/permissions
resource APIKey {
  // Primary key
  id: uuid! @primary @auto

  // The actual API key (generate with UUID or random string)
  // This should be hashed in production for security
  key_hash: string! @unique @min(64) @max(255)

  // Human-readable name for the key
  name: string! @min(3) @max(100)

  // Foreign key to User who owns this key
  user_id: uuid!

  // Key status
  // Values: "active", "revoked", "expired"
  status: string! @default("active")

  // Scopes define what this key can access
  // Example: "read:posts,write:posts,read:users"
  scopes: string! @default("read:*")

  // Expiration tracking
  expires_at: timestamp?
  last_used_at: timestamp?

  // Usage statistics
  request_count: int! @default(0)

  // Timestamps
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  // Relationship: API Key belongs to User
  user: User! {
    foreign_key: "user_id"
    on_delete: cascade
  }

  // Note: last_used_at should be updated in your API middleware
  // when the key is actually used, not in the model hooks
}
`
}
