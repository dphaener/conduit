package auth

import (
	"strings"
	"testing"
	"time"
)

func TestNewAuthService(t *testing.T) {
	secretKey := "test-secret"
	tokenTTL := time.Hour

	service := NewAuthService(secretKey, tokenTTL)

	if service == nil {
		t.Fatal("NewAuthService() returned nil")
	}

	if service.secretKey != secretKey {
		t.Errorf("AuthService.secretKey = %v, want %v", service.secretKey, secretKey)
	}

	if service.tokenTTL != tokenTTL {
		t.Errorf("AuthService.tokenTTL = %v, want %v", service.tokenTTL, tokenTTL)
	}
}

func TestAuthServiceGenerateToken(t *testing.T) {
	service := NewAuthService("test-secret-key", time.Hour)

	tests := []struct {
		name   string
		userID string
		email  string
		roles  []string
	}{
		{
			name:   "generates token with all fields",
			userID: "user-123",
			email:  "user@example.com",
			roles:  []string{"admin", "editor"},
		},
		{
			name:   "generates token with single role",
			userID: "user-456",
			email:  "test@example.com",
			roles:  []string{"viewer"},
		},
		{
			name:   "generates token with no roles",
			userID: "user-789",
			email:  "noroles@example.com",
			roles:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := service.GenerateToken(tt.userID, tt.email, tt.roles)
			if err != nil {
				t.Fatalf("GenerateToken() error = %v", err)
			}

			if token == "" {
				t.Error("GenerateToken() returned empty token")
			}

			// Token should have 3 parts separated by dots
			parts := strings.Split(token, ".")
			if len(parts) != 3 {
				t.Errorf("Token has %d parts, expected 3", len(parts))
			}

			// Validate the token
			claims, err := service.ValidateToken(token)
			if err != nil {
				t.Fatalf("ValidateToken() error = %v", err)
			}

			// Verify claims
			if claims["user_id"] != tt.userID {
				t.Errorf("Token user_id = %v, want %v", claims["user_id"], tt.userID)
			}

			if claims["email"] != tt.email {
				t.Errorf("Token email = %v, want %v", claims["email"], tt.email)
			}

			// Verify roles
			rolesInterface, ok := claims["roles"].([]interface{})
			if !ok {
				t.Fatal("Token roles claim is not []interface{}")
			}

			if len(rolesInterface) != len(tt.roles) {
				t.Errorf("Token has %d roles, want %d", len(rolesInterface), len(tt.roles))
			}

			for i, role := range tt.roles {
				if rolesInterface[i] != role {
					t.Errorf("Token roles[%d] = %v, want %v", i, rolesInterface[i], role)
				}
			}

			// Verify exp and iat claims exist
			if _, ok := claims["exp"]; !ok {
				t.Error("Token missing exp claim")
			}

			if _, ok := claims["iat"]; !ok {
				t.Error("Token missing iat claim")
			}
		})
	}
}

func TestAuthServiceValidateToken(t *testing.T) {
	service := NewAuthService("test-secret-key", time.Hour)

	// Generate a valid token
	validToken, _ := service.GenerateToken("user-123", "user@example.com", []string{"admin"})

	tests := []struct {
		name      string
		token     string
		wantError bool
	}{
		{
			name:      "validates valid token",
			token:     validToken,
			wantError: false,
		},
		{
			name:      "rejects invalid token format",
			token:     "invalid.token.format",
			wantError: true,
		},
		{
			name:      "rejects malformed token",
			token:     "notavalidtoken",
			wantError: true,
		},
		{
			name:      "rejects empty token",
			token:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateToken(tt.token)

			if tt.wantError {
				if err == nil {
					t.Error("ValidateToken() should return error for invalid token")
				}
			} else {
				if err != nil {
					t.Errorf("ValidateToken() unexpected error = %v", err)
				}

				if claims == nil {
					t.Error("ValidateToken() returned nil claims for valid token")
				}
			}
		})
	}
}

func TestAuthServiceValidateTokenWithWrongSecret(t *testing.T) {
	// Create token with one secret
	service1 := NewAuthService("secret-key-1", time.Hour)
	token, _ := service1.GenerateToken("user-123", "user@example.com", []string{"admin"})

	// Try to validate with different secret
	service2 := NewAuthService("secret-key-2", time.Hour)
	claims, err := service2.ValidateToken(token)

	if err == nil {
		t.Error("ValidateToken() should reject token signed with different secret")
	}

	if claims != nil {
		t.Error("ValidateToken() should return nil claims for invalid token")
	}
}

func TestAuthServiceValidateExpiredToken(t *testing.T) {
	// Create service with negative TTL (already expired)
	service := NewAuthService("test-secret", -time.Hour)
	token, err := service.GenerateToken("user-expired", "user@example.com", []string{"admin"})
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Try to validate expired token
	claims, err := service.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() should reject expired token")
	}

	if claims != nil {
		t.Error("ValidateToken() should return nil claims for expired token")
	}
}

func TestAuthServiceTokenExpiration(t *testing.T) {
	ttl := 2 * time.Second
	service := NewAuthService("test-secret", ttl)

	token, err := service.GenerateToken("user-123", "user@example.com", []string{"admin"})
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Token should be valid immediately
	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Errorf("ValidateToken() should accept fresh token, got error: %v", err)
	}

	if claims == nil {
		t.Fatal("ValidateToken() returned nil claims for fresh token")
	}

	// Check exp claim is in the future
	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("exp claim is not a number")
	}

	expTime := time.Unix(int64(exp), 0)
	if !expTime.After(time.Now()) {
		t.Error("Token expiration should be in the future")
	}
}

func TestAuthServiceWithEmptyUserID(t *testing.T) {
	service := NewAuthService("test-secret", time.Hour)

	token, err := service.GenerateToken("", "user@example.com", []string{"admin"})
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	// Token should still be generated and valid
	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Errorf("ValidateToken() error = %v", err)
	}

	if claims["user_id"] != "" {
		t.Errorf("Expected empty user_id, got %v", claims["user_id"])
	}
}

func TestAuthServiceWithSpecialCharacters(t *testing.T) {
	service := NewAuthService("test-secret", time.Hour)

	specialInputs := []struct {
		userID string
		email  string
		roles  []string
	}{
		{"user-with-unicode-🔐", "emoji@example.com", []string{"admin"}},
		{"user@special#chars!", "special!@example.com", []string{"editor", "viewer"}},
		{"user\twith\ttabs", "tabs@example.com", []string{"admin"}},
	}

	for _, input := range specialInputs {
		t.Run(input.userID, func(t *testing.T) {
			token, err := service.GenerateToken(input.userID, input.email, input.roles)
			if err != nil {
				t.Fatalf("GenerateToken() error = %v", err)
			}

			claims, err := service.ValidateToken(token)
			if err != nil {
				t.Fatalf("ValidateToken() error = %v", err)
			}

			if claims["user_id"] != input.userID {
				t.Errorf("user_id = %v, want %v", claims["user_id"], input.userID)
			}

			if claims["email"] != input.email {
				t.Errorf("email = %v, want %v", claims["email"], input.email)
			}
		})
	}
}

func TestAuthServiceWithManyRoles(t *testing.T) {
	service := NewAuthService("test-secret", time.Hour)

	manyRoles := make([]string, 100)
	for i := 0; i < 100; i++ {
		manyRoles[i] = "role-" + string(rune('0'+i%10))
	}

	token, err := service.GenerateToken("user-many-roles", "user@example.com", manyRoles)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	rolesInterface, ok := claims["roles"].([]interface{})
	if !ok {
		t.Fatal("roles claim is not []interface{}")
	}

	if len(rolesInterface) != len(manyRoles) {
		t.Errorf("Token has %d roles, want %d", len(rolesInterface), len(manyRoles))
	}
}

func BenchmarkAuthServiceGenerateToken(b *testing.B) {
	service := NewAuthService("test-secret-key", time.Hour)
	roles := []string{"admin", "editor"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateToken("user-bench", "bench@example.com", roles)
	}
}

func BenchmarkAuthServiceValidateToken(b *testing.B) {
	service := NewAuthService("test-secret-key", time.Hour)
	token, _ := service.GenerateToken("user-bench", "bench@example.com", []string{"admin"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.ValidateToken(token)
	}
}
