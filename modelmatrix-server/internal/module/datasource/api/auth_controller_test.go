package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/internal/infrastructure/ldap"
	"modelmatrix-server/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock LDAP client
// ---------------------------------------------------------------------------

type mockLDAPClient struct {
	authenticateFn func(username, password string) (*ldap.User, error)
}

func (m *mockLDAPClient) Authenticate(username, password string) (*ldap.User, error) {
	return m.authenticateFn(username, password)
}
func (m *mockLDAPClient) GetUserGroups(userDN string) ([]string, error) { return nil, nil }
func (m *mockLDAPClient) GetUser(username string) (*ldap.User, error)   { return nil, nil }
func (m *mockLDAPClient) HealthCheck() error                             { return nil }
func (m *mockLDAPClient) Close()                                         {}

// ---------------------------------------------------------------------------
// Router helpers
// ---------------------------------------------------------------------------

func setupAuthRouter(ldapClient ldap.Client, tokenSvc *auth.TokenService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := NewAuthController(ldapClient, tokenSvc)
	api := r.Group("/api")
	ctrl.RegisterRoutes(api)
	return r
}

func newTokenService() *auth.TokenService {
	return auth.NewTokenService(&config.JWTConfig{
		Secret:          "test-secret-key",
		ExpirationHours: 1,
	})
}

func sendAuthReq(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestAuthController_Login_Success(t *testing.T) {
	ldapClient := &mockLDAPClient{
		authenticateFn: func(username, password string) (*ldap.User, error) {
			if username == "alice" && password == "secret" {
				return &ldap.User{
					UID:      "alice",
					FullName: "Alice Smith",
					Email:    "alice@example.org",
					Groups:   []string{ldap.GroupAdmin},
				}, nil
			}
			return nil, errors.New("invalid credentials")
		},
	}
	tokenSvc := newTokenService()
	r := setupAuthRouter(ldapClient, tokenSvc)

	w := sendAuthReq(r, "POST", "/api/auth/login", map[string]string{
		"username": "alice",
		"password": "secret",
	})

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "alice", data["username"])
	assert.NotEmpty(t, data["token"])
}

func TestAuthController_Login_InvalidCredentials(t *testing.T) {
	ldapClient := &mockLDAPClient{
		authenticateFn: func(username, password string) (*ldap.User, error) {
			return nil, errors.New("invalid credentials")
		},
	}
	r := setupAuthRouter(ldapClient, newTokenService())

	w := sendAuthReq(r, "POST", "/api/auth/login", map[string]string{
		"username": "hacker",
		"password": "wrong",
	})

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthController_Login_MissingFields(t *testing.T) {
	r := setupAuthRouter(&mockLDAPClient{}, newTokenService())

	// Missing password — binding:"required" should reject it
	w := sendAuthReq(r, "POST", "/api/auth/login", map[string]string{
		"username": "alice",
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_Login_InvalidJSON(t *testing.T) {
	r := setupAuthRouter(&mockLDAPClient{}, newTokenService())

	req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Refresh
// ---------------------------------------------------------------------------

func TestAuthController_Refresh_ValidToken(t *testing.T) {
	tokenSvc := newTokenService()

	// Generate a real token first
	user := &ldap.User{UID: "bob", FullName: "Bob", Email: "bob@example.org", Groups: []string{ldap.GroupViewer}}
	token, err := tokenSvc.GenerateToken(user)
	require.NoError(t, err)

	// Build router with auth middleware so claims are injected for Refresh
	gin.SetMode(gin.TestMode)
	r2 := gin.New()
	r2.Use(auth.Middleware(tokenSvc))
	ctrl := NewAuthController(&mockLDAPClient{}, tokenSvc)
	api := r2.Group("/api")
	ctrl.RegisterRoutes(api)

	req, _ := http.NewRequest("POST", "/api/auth/refresh", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r2.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["token"])
}

func TestAuthController_Refresh_MissingClaims(t *testing.T) {
	// No auth middleware → no claims in context → 401
	r := setupAuthRouter(&mockLDAPClient{}, newTokenService())

	req, _ := http.NewRequest("POST", "/api/auth/refresh", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
