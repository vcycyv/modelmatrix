package api

import (
	"modelmatrix_backend/internal/infrastructure/auth"
	"modelmatrix_backend/internal/infrastructure/ldap"
	"modelmatrix_backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// LoginRequest represents login credentials
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"password"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token    string   `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	Username string   `json:"username" example:"admin"`
	FullName string   `json:"full_name" example:"Admin User"`
	Email    string   `json:"email" example:"admin@example.com"`
	Groups   []string `json:"groups" example:"modelmatrix_admin"`
}

// AuthController handles authentication-related HTTP requests
type AuthController struct {
	ldapClient   ldap.Client
	tokenService *auth.TokenService
}

// NewAuthController creates a new auth controller
func NewAuthController(ldapClient ldap.Client, tokenService *auth.TokenService) *AuthController {
	return &AuthController{
		ldapClient:   ldapClient,
		tokenService: tokenService,
	}
}

// RegisterRoutes registers auth routes
func (c *AuthController) RegisterRoutes(router *gin.RouterGroup) {
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", c.Login)
		authGroup.POST("/refresh", c.Refresh)
	}
}

// Login godoc
// @Summary User login
// @Description Authenticates user via LDAP and returns JWT token
// @Tags Auth
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "Login credentials"
// @Success 200 {object} response.Response{data=LoginResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/auth/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	// Authenticate via LDAP
	user, err := c.ldapClient.Authenticate(req.Username, req.Password)
	if err != nil {
		response.Unauthorized(ctx, "invalid credentials")
		return
	}

	// Generate JWT token
	token, err := c.tokenService.GenerateToken(user)
	if err != nil {
		response.InternalError(ctx, "failed to generate token")
		return
	}

	response.Success(ctx, LoginResponse{
		Token:    token,
		Username: user.UID,
		FullName: user.FullName,
		Email:    user.Email,
		Groups:   user.Groups,
	})
}

// Refresh godoc
// @Summary Refresh token
// @Description Refreshes the JWT token
// @Tags Auth
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} response.Response{data=LoginResponse}
// @Failure 401 {object} response.Response
// @Router /api/auth/refresh [post]
func (c *AuthController) Refresh(ctx *gin.Context) {
	claims, exists := auth.GetClaims(ctx)
	if !exists {
		response.Unauthorized(ctx, "invalid token")
		return
	}

	// Generate new token
	token, err := c.tokenService.RefreshToken(claims)
	if err != nil {
		response.InternalError(ctx, "failed to refresh token")
		return
	}

	response.Success(ctx, LoginResponse{
		Token:    token,
		Username: claims.Username,
		FullName: claims.FullName,
		Email:    claims.Email,
		Groups:   claims.Groups,
	})
}

