package commands

import (
	"errors"
	"strings"
	"testing"
)

func TestParseDBURL(t *testing.T) {
	tests := []struct {
		name              string
		url               string
		expectedDBName    string
		expectedUser      string
		expectedMaintURL  string
		expectError       bool
		errorContains     string
	}{
		{
			name:             "Valid PostgreSQL URL with password",
			url:              "postgresql://user:pass@localhost:5432/mydb",
			expectedDBName:   "mydb",
			expectedUser:     "user",
			expectedMaintURL: "postgresql://user:pass@localhost:5432/postgres",
			expectError:      false,
		},
		{
			name:             "Valid postgres URL (short scheme)",
			url:              "postgres://user:pass@localhost:5432/mydb",
			expectedDBName:   "mydb",
			expectedUser:     "user",
			expectedMaintURL: "postgres://user:pass@localhost:5432/postgres",
			expectError:      false,
		},
		{
			name:             "URL without password",
			url:              "postgresql://user@localhost:5432/testdb",
			expectedDBName:   "testdb",
			expectedUser:     "user",
			expectedMaintURL: "postgresql://user@localhost:5432/postgres",
			expectError:      false,
		},
		{
			name:             "URL with default port",
			url:              "postgresql://postgres:secret@localhost/appdb",
			expectedDBName:   "appdb",
			expectedUser:     "postgres",
			expectedMaintURL: "postgresql://postgres:secret@localhost/postgres",
			expectError:      false,
		},
		{
			name:             "URL with remote host",
			url:              "postgresql://admin:pass123@db.example.com:5432/production",
			expectedDBName:   "production",
			expectedUser:     "admin",
			expectedMaintURL: "postgresql://admin:pass123@db.example.com:5432/postgres",
			expectError:      false,
		},
		{
			name:             "URL with query parameters",
			url:              "postgresql://user:pass@localhost:5432/mydb?sslmode=require",
			expectedDBName:   "mydb",
			expectedUser:     "user",
			expectedMaintURL: "postgresql://user:pass@localhost:5432/postgres?sslmode=require",
			expectError:      false,
		},
		{
			name:             "URL with special characters in database name",
			url:              "postgresql://user:pass@localhost:5432/my_app_db",
			expectedDBName:   "my_app_db",
			expectedUser:     "user",
			expectedMaintURL: "postgresql://user:pass@localhost:5432/postgres",
			expectError:      false,
		},
		{
			name:             "URL with special characters in password",
			url:              "postgresql://user:p@ss!123@localhost:5432/mydb",
			expectedDBName:   "mydb",
			expectedUser:     "user",
			expectedMaintURL: "postgresql://user:p@ss!123@localhost:5432/postgres",
			expectError:      false,
		},
		{
			name:          "Missing database name",
			url:           "postgresql://user:pass@localhost:5432/",
			expectError:   true,
			errorContains: "database name not specified",
		},
		{
			name:          "No database path at all",
			url:           "postgresql://user:pass@localhost:5432",
			expectError:   true,
			errorContains: "database name not specified",
		},
		{
			name:          "Invalid scheme",
			url:           "mysql://user:pass@localhost:5432/mydb",
			expectError:   true,
			errorContains: "unsupported scheme",
		},
		{
			name:          "Completely invalid URL",
			url:           "not a url at all",
			expectError:   true,
			errorContains: "unsupported scheme",
		},
		{
			name:          "Empty URL",
			url:           "",
			expectError:   true,
			errorContains: "unsupported scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbName, user, maintURL, err := parseDBURL(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error '%v' does not contain expected text '%s'", err, tt.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if dbName != tt.expectedDBName {
				t.Errorf("Expected database name '%s', got '%s'", tt.expectedDBName, dbName)
			}

			if user != tt.expectedUser {
				t.Errorf("Expected user '%s', got '%s'", tt.expectedUser, user)
			}

			if maintURL != tt.expectedMaintURL {
				t.Errorf("Expected maintenance URL '%s', got '%s'", tt.expectedMaintURL, maintURL)
			}
		})
	}
}

func TestIsProductionDatabase(t *testing.T) {
	tests := []struct {
		name       string
		dbName     string
		isProduction bool
	}{
		{
			name:         "Contains 'production'",
			dbName:       "myapp_production",
			isProduction: true,
		},
		{
			name:         "Contains 'prod'",
			dbName:       "myapp_prod",
			isProduction: true,
		},
		{
			name:         "Contains 'PRODUCTION' (uppercase)",
			dbName:       "PRODUCTION_DB",
			isProduction: true,
		},
		{
			name:         "Contains 'Prod' (mixed case)",
			dbName:       "myapp_Prod",
			isProduction: true,
		},
		{
			name:         "Development database",
			dbName:       "myapp_development",
			isProduction: false,
		},
		{
			name:         "Test database",
			dbName:       "myapp_test",
			isProduction: false,
		},
		{
			name:         "Staging database",
			dbName:       "myapp_staging",
			isProduction: false,
		},
		{
			name:         "Local database",
			dbName:       "myapp_local",
			isProduction: false,
		},
		{
			name:         "Simple name",
			dbName:       "myapp",
			isProduction: false,
		},
		{
			name:         "Word containing 'prod' but not production",
			dbName:       "product_catalog",
			isProduction: true, // This is by design - conservative approach
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isProductionDatabase(tt.dbName)
			if result != tt.isProduction {
				t.Errorf("Expected isProductionDatabase('%s') = %v, got %v", tt.dbName, tt.isProduction, result)
			}
		})
	}
}

func TestStripCredentials(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "Error with PostgreSQL URL and password",
			err:      errors.New("failed to connect to postgresql://user:secret123@localhost:5432/mydb"),
			expected: "failed to connect to postgresql://user:****@localhost:5432/mydb",
		},
		{
			name:     "Error with postgres URL and password",
			err:      errors.New("connection failed: postgres://admin:secret@db.example.com:5432/prod"),
			expected: "connection failed: postgres://admin:****@db.example.com:5432/prod",
		},
		{
			name:     "Error without credentials",
			err:      errors.New("connection timeout"),
			expected: "connection timeout",
		},
		{
			name:     "Error with URL but no password",
			err:      errors.New("failed: postgresql://user@localhost:5432/mydb"),
			expected: "failed: postgresql://user@localhost:5432/mydb",
		},
		{
			name:     "Complex error with multiple URLs",
			err:      errors.New("tried postgresql://user:pass1@host1/db and postgresql://user:pass2@host2/db"),
			expected: "tried postgresql://user:****@host1/db and postgresql://user:****@host2/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripCredentials(tt.err)

			if tt.err == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if result.Error() != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result.Error())
			}
		})
	}
}

func TestDatabaseNameExtraction(t *testing.T) {
	// Test various edge cases for database name extraction
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "Simple database name",
			url:      "postgresql://user:pass@localhost/mydb",
			expected: "mydb",
		},
		{
			name:     "Database name with underscores",
			url:      "postgresql://user:pass@localhost/my_app_db",
			expected: "my_app_db",
		},
		{
			name:     "Database name with numbers",
			url:      "postgresql://user:pass@localhost/mydb123",
			expected: "mydb123",
		},
		{
			name:     "Database name with hyphens",
			url:      "postgresql://user:pass@localhost/my-app-db",
			expected: "my-app-db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbName, _, _, err := parseDBURL(tt.url)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if dbName != tt.expected {
				t.Errorf("Expected database name '%s', got '%s'", tt.expected, dbName)
			}
		})
	}
}

func TestUserExtraction(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		expectedUser string
	}{
		{
			name:         "Explicit user",
			url:          "postgresql://myuser:pass@localhost/db",
			expectedUser: "myuser",
		},
		{
			name:         "User without password",
			url:          "postgresql://myuser@localhost/db",
			expectedUser: "myuser",
		},
		{
			name:         "Default postgres user",
			url:          "postgresql://postgres:pass@localhost/db",
			expectedUser: "postgres",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, user, _, err := parseDBURL(tt.url)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if user != tt.expectedUser {
				t.Errorf("Expected user '%s', got '%s'", tt.expectedUser, user)
			}
		})
	}
}

func TestMaintenanceURLConstruction(t *testing.T) {
	tests := []struct {
		name                 string
		url                  string
		expectedMaintenanceURL string
	}{
		{
			name:                   "Simple URL",
			url:                    "postgresql://user:pass@localhost:5432/mydb",
			expectedMaintenanceURL: "postgresql://user:pass@localhost:5432/postgres",
		},
		{
			name:                   "URL with query params",
			url:                    "postgresql://user:pass@localhost:5432/mydb?sslmode=require&connect_timeout=10",
			expectedMaintenanceURL: "postgresql://user:pass@localhost:5432/postgres?sslmode=require&connect_timeout=10",
		},
		{
			name:                   "URL without port",
			url:                    "postgresql://user:pass@localhost/mydb",
			expectedMaintenanceURL: "postgresql://user:pass@localhost/postgres",
		},
		{
			name:                   "Remote host URL",
			url:                    "postgresql://admin:secret@db.example.com:5432/production",
			expectedMaintenanceURL: "postgresql://admin:secret@db.example.com:5432/postgres",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, maintURL, err := parseDBURL(tt.url)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if maintURL != tt.expectedMaintenanceURL {
				t.Errorf("Expected maintenance URL '%s', got '%s'", tt.expectedMaintenanceURL, maintURL)
			}
		})
	}
}
