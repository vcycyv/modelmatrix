package auth

import (
	"fmt"
	"time"

	"modelmatrix_backend/internal/infrastructure/ldap"
	"modelmatrix_backend/pkg/config"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents JWT claims
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	FullName string   `json:"full_name"`
	Groups   []string `json:"groups"`
	jwt.RegisteredClaims
}

// TokenService handles JWT token operations
type TokenService struct {
	secret          []byte
	expirationHours int
}

// NewTokenService creates a new TokenService
func NewTokenService(cfg *config.JWTConfig) *TokenService {
	return &TokenService{
		secret:          []byte(cfg.Secret),
		expirationHours: cfg.ExpirationHours,
	}
}

// GenerateToken generates a JWT token for a user
func (s *TokenService) GenerateToken(user *ldap.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   user.UID,
		Username: user.UID,
		Email:    user.Email,
		FullName: user.FullName,
		Groups:   user.Groups,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.expirationHours) * time.Hour)),
			Issuer:    "modelmatrix",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ValidateToken validates a JWT token and returns claims
func (s *TokenService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// RefreshToken creates a new token with extended expiration
func (s *TokenService) RefreshToken(claims *Claims) (string, error) {
	now := time.Now()
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(s.expirationHours) * time.Hour))

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// HasGroup checks if claims contain a specific group
func (c *Claims) HasGroup(group string) bool {
	for _, g := range c.Groups {
		if g == group {
			return true
		}
	}
	return false
}

// HasAnyGroup checks if claims contain any of the specified groups
func (c *Claims) HasAnyGroup(groups []string) bool {
	for _, g := range groups {
		if c.HasGroup(g) {
			return true
		}
	}
	return false
}

// IsAdmin checks if user is an admin
func (c *Claims) IsAdmin() bool {
	return c.HasGroup(ldap.GroupAdmin)
}

// IsEditor checks if user is an editor
func (c *Claims) IsEditor() bool {
	return c.HasAnyGroup([]string{ldap.GroupAdmin, ldap.GroupEditor})
}

// IsViewer checks if user is a viewer
func (c *Claims) IsViewer() bool {
	return c.HasAnyGroup([]string{ldap.GroupAdmin, ldap.GroupEditor, ldap.GroupViewer})
}

