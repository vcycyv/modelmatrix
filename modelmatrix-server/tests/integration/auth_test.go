package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// AuthTestSuite tests authentication endpoints.
type AuthTestSuite struct {
	suite.Suite
	client  *http.Client
	baseURL string
}

func (s *AuthTestSuite) SetupSuite() {
	s.client = newAPIClient()
	s.baseURL = testServerURL
}

func (s *AuthTestSuite) TearDownSuite() {
	truncateAllTables(s.T())
}

// TestLogin_ValidCredentials verifies that correct credentials return a JWT.
func (s *AuthTestSuite) TestLogin_ValidCredentials() {
	req := map[string]string{"username": "michael.jordan", "password": "111222333"}
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/auth/login", "", req)
	defer resp.Body.Close()

	requireSuccess(s.T(), resp)

	var result struct {
		Code int `json:"code"`
		Data struct {
			Token    string   `json:"token"`
			Username string   `json:"username"`
			Groups   []string `json:"groups"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), 200, result.Code)
	assert.NotEmpty(s.T(), result.Data.Token)
	assert.Equal(s.T(), "michael.jordan", result.Data.Username)
	assert.Contains(s.T(), result.Data.Groups, "modelmatrix_admin")
}

// TestLogin_WrongPassword verifies that bad credentials return 401.
func (s *AuthTestSuite) TestLogin_WrongPassword() {
	req := map[string]string{"username": "michael.jordan", "password": "wrongpassword"}
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/auth/login", "", req)
	defer resp.Body.Close()
	requireUnauthorized(s.T(), resp)
}

// TestLogin_UnknownUser verifies that an unknown user returns 401.
func (s *AuthTestSuite) TestLogin_UnknownUser() {
	req := map[string]string{"username": "nobody.exists", "password": "whatever"}
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/auth/login", "", req)
	defer resp.Body.Close()
	requireUnauthorized(s.T(), resp)
}

// TestLogin_MissingFields verifies that missing fields return 400.
func (s *AuthTestSuite) TestLogin_MissingFields() {
	req := map[string]string{"username": "michael.jordan"} // missing password
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/auth/login", "", req)
	defer resp.Body.Close()
	requireBadRequest(s.T(), resp)
}

// TestRefreshToken verifies that a valid token can be refreshed.
func (s *AuthTestSuite) TestRefreshToken() {
	token := authenticate(s.T(), s.client, s.baseURL, "michael.jordan", "111222333")

	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/auth/refresh", token, nil)
	defer resp.Body.Close()

	requireSuccess(s.T(), resp)
	var result struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), 200, result.Code)
	assert.NotEmpty(s.T(), result.Data.Token)
}

// TestRefreshToken_NoAuth verifies that refresh without a token returns 401.
func (s *AuthTestSuite) TestRefreshToken_NoAuth() {
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/auth/refresh", "", nil)
	defer resp.Body.Close()
	requireUnauthorized(s.T(), resp)
}

func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}
