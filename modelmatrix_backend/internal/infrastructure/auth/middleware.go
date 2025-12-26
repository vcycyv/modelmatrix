package auth

import (
	"strings"

	"modelmatrix_backend/internal/infrastructure/ldap"
	"modelmatrix_backend/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	// ContextKeyUser is the key for user claims in gin context
	ContextKeyUser = "user"
)

// Middleware creates an authentication middleware
func Middleware(tokenService *TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Unauthorized(c, "invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := tokenService.ValidateToken(tokenString)
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		// Store claims in context
		c.Set(ContextKeyUser, claims)
		c.Next()
	}
}

// RequireAdmin creates a middleware that requires admin role
func RequireAdmin() gin.HandlerFunc {
	return RequireGroups([]string{ldap.GroupAdmin})
}

// RequireEditor creates a middleware that requires editor role (or admin)
func RequireEditor() gin.HandlerFunc {
	return RequireGroups([]string{ldap.GroupAdmin, ldap.GroupEditor})
}

// RequireViewer creates a middleware that requires viewer role (or editor/admin)
func RequireViewer() gin.HandlerFunc {
	return RequireGroups([]string{ldap.GroupAdmin, ldap.GroupEditor, ldap.GroupViewer})
}

// RequireGroups creates a middleware that requires any of the specified groups
func RequireGroups(groups []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := GetClaims(c)
		if !exists {
			response.Unauthorized(c, "user not authenticated")
			c.Abort()
			return
		}

		if !claims.HasAnyGroup(groups) {
			response.Forbidden(c, "insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetClaims retrieves user claims from context
func GetClaims(c *gin.Context) (*Claims, bool) {
	value, exists := c.Get(ContextKeyUser)
	if !exists {
		return nil, false
	}

	claims, ok := value.(*Claims)
	return claims, ok
}

// GetUserID retrieves user ID from context
func GetUserID(c *gin.Context) string {
	claims, exists := GetClaims(c)
	if !exists {
		return ""
	}
	return claims.UserID
}

// GetUsername retrieves username from context
func GetUsername(c *gin.Context) string {
	claims, exists := GetClaims(c)
	if !exists {
		return ""
	}
	return claims.Username
}

