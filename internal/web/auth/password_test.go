package auth

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "hashes simple password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "hashes complex password",
			password: "P@ssw0rd!2023#$%^&*()",
			wantErr:  false,
		},
		{
			name:     "hashes empty password",
			password: "",
			wantErr:  false,
		},
		{
			name:     "hashes long password within limit",
			password: strings.Repeat("a", 72), // bcrypt max is 72 bytes
			wantErr:  false,
		},
		{
			name:     "rejects password exceeding 72 bytes",
			password: strings.Repeat("a", 73),
			wantErr:  true,
		},
		{
			name:     "rejects very long password",
			password: strings.Repeat("a", 100),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify hash is not empty
				if hash == "" {
					t.Error("HashPassword() returned empty hash")
				}

				// Verify hash is different from password
				if hash == tt.password {
					t.Error("HashPassword() returned unhashed password")
				}

				// Verify hash starts with bcrypt prefix
				if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
					t.Error("HashPassword() returned invalid bcrypt hash")
				}

				// Verify hash can be validated with bcrypt
				err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(tt.password))
				if err != nil {
					t.Errorf("HashPassword() created invalid hash: %v", err)
				}
			}
		})
	}
}

func TestHashPasswordDifferentHashes(t *testing.T) {
	password := "samepassword"

	hash1, err1 := HashPassword(password)
	if err1 != nil {
		t.Fatalf("HashPassword() error = %v", err1)
	}

	hash2, err2 := HashPassword(password)
	if err2 != nil {
		t.Fatalf("HashPassword() error = %v", err2)
	}

	// Bcrypt should generate different hashes for the same password (salt)
	if hash1 == hash2 {
		t.Error("HashPassword() generated identical hashes for same password")
	}

	// But both should validate correctly
	if !CheckPassword(password, hash1) {
		t.Error("CheckPassword() failed for hash1")
	}
	if !CheckPassword(password, hash2) {
		t.Error("CheckPassword() failed for hash2")
	}
}

func TestCheckPassword(t *testing.T) {
	// Pre-generated hash for "testpassword"
	password := "testpassword"
	hash, _ := HashPassword(password)

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{
			name:     "validates correct password",
			password: password,
			hash:     hash,
			want:     true,
		},
		{
			name:     "rejects wrong password",
			password: "wrongpassword",
			hash:     hash,
			want:     false,
		},
		{
			name:     "rejects empty password",
			password: "",
			hash:     hash,
			want:     false,
		},
		{
			name:     "rejects invalid hash",
			password: password,
			hash:     "invalid-hash",
			want:     false,
		},
		{
			name:     "rejects empty hash",
			password: password,
			hash:     "",
			want:     false,
		},
		{
			name:     "case sensitive password check",
			password: "TestPassword",
			hash:     hash,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckPassword(tt.password, tt.hash)
			if got != tt.want {
				t.Errorf("CheckPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckPasswordWithSpecialCharacters(t *testing.T) {
	specialPasswords := []string{
		"p@ssw0rd!",
		"密码123",        // Chinese characters
		"пароль456",     // Cyrillic characters
		"emoji🔐pass",   // Emoji
		"space pass",    // Space
		"tab\tpass",     // Tab
		"newline\npass", // Newline
	}

	for _, password := range specialPasswords {
		t.Run(password, func(t *testing.T) {
			hash, err := HashPassword(password)
			if err != nil {
				t.Fatalf("HashPassword() error = %v", err)
			}

			if !CheckPassword(password, hash) {
				t.Error("CheckPassword() failed for special password")
			}

			// Verify wrong password fails
			if CheckPassword(password+"wrong", hash) {
				t.Error("CheckPassword() should reject modified password")
			}
		})
	}
}

func TestHashPasswordCost(t *testing.T) {
	password := "testpassword"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	// Verify bcrypt cost is DefaultCost
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("bcrypt.Cost() error = %v", err)
	}

	if cost != bcrypt.DefaultCost {
		t.Errorf("HashPassword() cost = %v, want %v", cost, bcrypt.DefaultCost)
	}
}

func BenchmarkHashPassword(b *testing.B) {
	password := "benchmarkpassword"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = HashPassword(password)
	}
}

func BenchmarkCheckPassword(b *testing.B) {
	password := "benchmarkpassword"
	hash, _ := HashPassword(password)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CheckPassword(password, hash)
	}
}
