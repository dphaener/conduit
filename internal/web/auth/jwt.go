package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthService provides JWT token generation and validation
type AuthService struct {
	secretKey string
	tokenTTL  time.Duration
}

// NewAuthService creates a new AuthService with the given secret key and token TTL
func NewAuthService(secretKey string, tokenTTL time.Duration) *AuthService {
	return &AuthService{
		secretKey: secretKey,
		tokenTTL:  tokenTTL,
	}
}

// GenerateToken generates a JWT token with the given user ID, email, and roles
func (s *AuthService) GenerateToken(userID, email string, roles []string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"roles":   roles,
		"exp":     now.Add(s.tokenTTL).Unix(),
		"iat":     now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

// ValidateToken validates a JWT token and returns the claims
func (s *AuthService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify exact signing method to prevent algorithm confusion attacks
		if token.Method.Alg() != "HS256" {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
