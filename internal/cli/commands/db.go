package commands

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/fatih/color"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"

	"github.com/conduit-lang/conduit/internal/cli/config"
)

var (
	dbCreateURLFlag string
	dbCreateYesFlag bool
)

// NewDBCommand creates the db command
func NewDBCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
		Long: `Manage PostgreSQL databases for Conduit applications.

Commands for creating, managing, and inspecting databases.`,
		Example: `  # Create database from DATABASE_URL
  conduit db create

  # Create database with custom URL
  conduit db create --url postgresql://user:pass@localhost/mydb

  # Skip production confirmation prompts
  conduit db create --yes`,
	}

	cmd.AddCommand(newDBCreateCommand())

	return cmd
}

func newDBCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create the database if it doesn't exist",
		Long: `Create the PostgreSQL database specified in DATABASE_URL if it doesn't already exist.

This command is idempotent - it's safe to run multiple times. If the database
already exists, it will print an info message and exit successfully.

The command connects to the 'postgres' maintenance database to create the target
database. The user must have CREATEDB privilege in PostgreSQL.`,
		Example: `  # Create database from DATABASE_URL environment variable
  conduit db create

  # Create database from conduit.yml configuration
  conduit db create

  # Override with custom database URL
  conduit db create --url postgresql://user:pass@localhost/mydb

  # Skip production confirmation prompt
  conduit db create --yes`,
		RunE: runDBCreate,
	}

	cmd.Flags().StringVar(&dbCreateURLFlag, "url", "", "Override DATABASE_URL")
	cmd.Flags().BoolVarP(&dbCreateYesFlag, "yes", "y", false, "Skip production confirmation prompts")

	return cmd
}

func runDBCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Color setup
	successColor := color.New(color.FgGreen, color.Bold)
	infoColor := color.New(color.FgCyan)
	errorColor := color.New(color.FgRed, color.Bold)
	warningColor := color.New(color.FgYellow, color.Bold)

	// Get DATABASE_URL
	databaseURL := dbCreateURLFlag
	if databaseURL == "" {
		databaseURL = config.GetDatabaseURL()
	}

	if databaseURL == "" {
		errorColor.Println("✗ DATABASE_URL not set")
		fmt.Println("\nTo fix, set DATABASE_URL in one of these ways:")
		fmt.Println("  1. Environment variable:")
		fmt.Println("     export DATABASE_URL=\"postgresql://user:password@localhost:5432/dbname\"")
		fmt.Println("  2. In conduit.yml:")
		fmt.Println("     database:")
		fmt.Println("       url: postgresql://user:password@localhost:5432/dbname")
		fmt.Println("  3. Using --url flag:")
		fmt.Println("     conduit db create --url postgresql://user:password@localhost:5432/dbname")
		return fmt.Errorf("DATABASE_URL not set")
	}

	// Parse DATABASE_URL to extract database name and user
	dbName, user, maintenanceURL, err := parseDBURL(databaseURL)
	if err != nil {
		errorColor.Println("✗ Invalid DATABASE_URL format")
		fmt.Println("\nExpected format:")
		fmt.Println("  postgresql://user:password@host:port/dbname")
		fmt.Println("\nExamples:")
		fmt.Println("  postgresql://postgres:secret@localhost:5432/myapp")
		fmt.Println("  postgresql://user@localhost/myapp")
		fmt.Println("  postgres://user:pass@db.example.com:5432/production_db")
		return fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	// Check if this is a production database and confirm
	if isProductionDatabase(dbName) && !dbCreateYesFlag {
		warningColor.Printf("⚠️  Warning: '%s' appears to be a production database\n", dbName)
		fmt.Printf("Are you sure you want to create this database? (y/N): ")
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			infoColor.Println("ℹ Database creation cancelled")
			return nil
		}
	}

	// Connect to postgres maintenance DB
	conn, err := pgx.Connect(ctx, maintenanceURL)
	if err != nil {
		// Try template1 as fallback
		fallbackURL := strings.Replace(maintenanceURL, "/postgres", "/template1", 1)
		conn, err = pgx.Connect(ctx, fallbackURL)
		if err != nil {
			errorColor.Println("✗ Failed to connect to PostgreSQL")
			fmt.Println("\nPossible causes:")
			fmt.Println("  • PostgreSQL is not running")
			fmt.Println("  • Invalid credentials in DATABASE_URL")
			fmt.Println("  • Host or port is incorrect")
			fmt.Println("\nTo check if PostgreSQL is running:")
			fmt.Println("  pg_isready")
			fmt.Println("  # or")
			fmt.Println("  sudo systemctl status postgresql")
			return fmt.Errorf("failed to connect: %w", stripCredentials(err))
		}
	}
	defer conn.Close(ctx)

	// Check if database exists
	exists, err := databaseExists(ctx, conn, dbName)
	if err != nil {
		errorColor.Println("✗ Failed to check if database exists")
		return fmt.Errorf("database check failed: %w", err)
	}

	if exists {
		infoColor.Printf("ℹ Database '%s' already exists (no action needed)\n", dbName)
		return nil
	}

	// Create database using sanitized identifier
	sanitizedName := pgx.Identifier{dbName}.Sanitize()
	createSQL := fmt.Sprintf("CREATE DATABASE %s", sanitizedName)

	_, err = conn.Exec(ctx, createSQL)
	if err != nil {
		// Check for permission denied error
		if strings.Contains(strings.ToLower(err.Error()), "permission denied") ||
			strings.Contains(strings.ToLower(err.Error()), "must have createdb privilege") {
			errorColor.Println("✗ Permission denied to create database")
			fmt.Println("\nYour PostgreSQL user lacks the CREATEDB privilege.")
			fmt.Println("\nTo fix, run as PostgreSQL superuser:")
			fmt.Printf("  ALTER USER %s CREATEDB;\n", user)
			fmt.Println("\nOr grant the privilege directly:")
			fmt.Printf("  # As postgres user:\n")
			fmt.Printf("  sudo -u postgres psql -c \"ALTER USER %s CREATEDB;\"\n", user)
			return fmt.Errorf("permission denied")
		}

		// Check for invalid database name
		if strings.Contains(strings.ToLower(err.Error()), "invalid") {
			errorColor.Printf("✗ Invalid database name: %s\n", dbName)
			return fmt.Errorf("invalid database name: %w", err)
		}

		errorColor.Println("✗ Failed to create database")
		return fmt.Errorf("create database failed: %w", err)
	}

	successColor.Printf("✓ Database '%s' created successfully\n", dbName)
	return nil
}

// parseDBURL extracts the database name, user, and constructs a maintenance URL
// Returns: dbName, user, maintenanceURL, error
func parseDBURL(databaseURL string) (string, string, string, error) {
	// Parse the URL
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse URL: %w", err)
	}

	// Validate scheme
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return "", "", "", fmt.Errorf("unsupported scheme '%s' (expected 'postgres' or 'postgresql')", u.Scheme)
	}

	// Extract database name
	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		return "", "", "", fmt.Errorf("database name not specified in URL")
	}

	// Extract user
	user := u.User.Username()
	if user == "" {
		user = "postgres" // Default user
	}

	// Build maintenance URL by replacing database name with 'postgres'
	maintenanceURL := strings.Replace(databaseURL, "/"+dbName, "/postgres", 1)

	// Also handle query parameters by ensuring they're preserved
	if u.RawQuery != "" {
		// The database name might be followed by query params
		maintenanceURL = strings.Replace(databaseURL, "/"+dbName+"?", "/postgres?", 1)
	}

	return dbName, user, maintenanceURL, nil
}

// databaseExists checks if a database exists
func databaseExists(ctx context.Context, conn *pgx.Conn, dbName string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
	err := conn.QueryRow(ctx, query, dbName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to query database existence: %w", err)
	}
	return exists, nil
}

// isProductionDatabase checks if a database name suggests it's a production database
func isProductionDatabase(dbName string) bool {
	lowerName := strings.ToLower(dbName)
	return strings.Contains(lowerName, "prod") || strings.Contains(lowerName, "production")
}

// stripCredentials removes credentials from an error message
func stripCredentials(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// Try to find and replace password in connection strings
	// Pattern: postgresql://user:password@host/db
	// Replace with: postgresql://user:****@host/db
	if strings.Contains(errStr, "://") {
		parts := strings.Split(errStr, "://")
		if len(parts) >= 2 {
			// Find the part between scheme and @
			for i := 1; i < len(parts); i++ {
				if strings.Contains(parts[i], "@") {
					beforeAt := strings.Split(parts[i], "@")[0]
					if strings.Contains(beforeAt, ":") {
						// Has password
						userParts := strings.Split(beforeAt, ":")
						if len(userParts) >= 2 {
							parts[i] = strings.Replace(parts[i], ":"+userParts[1]+"@", ":****@", 1)
						}
					}
				}
			}
			errStr = strings.Join(parts, "://")
		}
	}

	return fmt.Errorf("%s", errStr)
}
