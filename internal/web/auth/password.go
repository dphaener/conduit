package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plain text password using bcrypt
// Returns the hashed password as a string, or an error if hashing fails
// Rejects passwords longer than 72 bytes (bcrypt's maximum)
func HashPassword(password string) (string, error) {
	if len(password) > 72 {
		return "", fmt.Errorf("password exceeds maximum length of 72 bytes")
	}
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// CheckPassword compares a plain text password with a hashed password
// Returns true if the password matches the hash, false otherwise
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
