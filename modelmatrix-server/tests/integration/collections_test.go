package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// CollectionsTestSuite is a test suite for collection endpoints
type CollectionsTestSuite struct {
	suite.Suite
	client    *http.Client
	baseURL   string
	authToken string
}

// SetupSuite runs once before all tests
func (s *CollectionsTestSuite) SetupSuite() {
	s.client = newAPIClient()
	s.baseURL = testServerURL
	s.authToken = authenticate(s.T(), s.client, s.baseURL, "michael.jordan", "111222333")
}

// TearDownSuite runs once after all tests
func (s *CollectionsTestSuite) TearDownSuite() {
	truncateAllTables(s.T())
}

// SetupTest runs before each test
func (s *CollectionsTestSuite) SetupTest() {
	// Optional: Clean up specific test data before each test
}

// TestCreateCollection tests POST /api/collections
func (s *CollectionsTestSuite) TestCreateCollection() {
	req := map[string]interface{}{
		"name":        "Test Collection",
		"description": "This is a test collection",
	}

	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/collections", s.authToken, req)
	defer resp.Body.Close()

	requireCreated(s.T(), resp) // HTTP status should be 201

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)

	// Note: HTTP status is 201, but JSON response code is 200 (CodeSuccess)
	assert.Equal(s.T(), 200, result.Code)
	assert.NotEmpty(s.T(), result.Data.ID)
	assert.Equal(s.T(), "Test Collection", result.Data.Name)
	assert.Equal(s.T(), "This is a test collection", result.Data.Description)
}

// TestListCollections tests GET /api/collections
func (s *CollectionsTestSuite) TestListCollections() {
	// First, create a collection
	createReq := map[string]interface{}{
		"name":        "List Test Collection",
		"description": "For listing test",
	}
	createResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/collections", s.authToken, createReq)
	requireCreated(s.T(), createResp)
	createResp.Body.Close()

	// Now list collections
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/collections?page=1&page_size=10", s.authToken, nil)
	defer resp.Body.Close()

	requireSuccess(s.T(), resp)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Collections []struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"collections"`
			Total int `json:"total"`
			Page  int `json:"page"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)

	assert.Equal(s.T(), 200, result.Code)
	assert.GreaterOrEqual(s.T(), result.Data.Total, 1)
	assert.GreaterOrEqual(s.T(), len(result.Data.Collections), 1)
}

// TestGetCollection tests GET /api/collections/:id
func (s *CollectionsTestSuite) TestGetCollection() {
	// Create a collection first
	createReq := map[string]interface{}{
		"name":        "Get Test Collection",
		"description": "For get test",
	}
	createResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/collections", s.authToken, createReq)
	requireCreated(s.T(), createResp)

	var createResult struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	parseResponse(s.T(), createResp, &createResult)
	createResp.Body.Close()

	collectionID := createResult.Data.ID
	require.NotEmpty(s.T(), collectionID)

	// Get the collection
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/collections/"+collectionID, s.authToken, nil)
	defer resp.Body.Close()

	requireSuccess(s.T(), resp)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)

	assert.Equal(s.T(), 200, result.Code)
	assert.Equal(s.T(), collectionID, result.Data.ID)
	assert.Equal(s.T(), "Get Test Collection", result.Data.Name)
}

// TestUpdateCollection tests PUT /api/collections/:id
func (s *CollectionsTestSuite) TestUpdateCollection() {
	// Create a collection first
	createReq := map[string]interface{}{
		"name":        "Update Test Collection",
		"description": "Original description",
	}
	createResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/collections", s.authToken, createReq)
	requireCreated(s.T(), createResp)

	var createResult struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	parseResponse(s.T(), createResp, &createResult)
	createResp.Body.Close()

	collectionID := createResult.Data.ID

	// Update the collection
	updateReq := map[string]interface{}{
		"name":        "Updated Collection Name",
		"description": "Updated description",
	}
	resp := makeRequest(s.T(), s.client, "PUT", s.baseURL+"/api/collections/"+collectionID, s.authToken, updateReq)
	defer resp.Body.Close()

	requireSuccess(s.T(), resp)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)

	assert.Equal(s.T(), 200, result.Code)
	assert.Equal(s.T(), "Updated Collection Name", result.Data.Name)
	assert.Equal(s.T(), "Updated description", result.Data.Description)
}

// TestDeleteCollection tests DELETE /api/collections/:id
func (s *CollectionsTestSuite) TestDeleteCollection() {
	// Create a collection first
	createReq := map[string]interface{}{
		"name":        "Delete Test Collection",
		"description": "To be deleted",
	}
	createResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/collections", s.authToken, createReq)
	requireCreated(s.T(), createResp)

	var createResult struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	parseResponse(s.T(), createResp, &createResult)
	createResp.Body.Close()

	collectionID := createResult.Data.ID

	// Delete the collection
	resp := makeRequest(s.T(), s.client, "DELETE", s.baseURL+"/api/collections/"+collectionID, s.authToken, nil)
	defer resp.Body.Close()

	requireNoContent(s.T(), resp) // DELETE returns 204 No Content

	// Verify it's deleted by trying to get it
	getResp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/collections/"+collectionID, s.authToken, nil)
	defer getResp.Body.Close()
	requireNotFound(s.T(), getResp)
}

// TestCreateCollectionUnauthorized tests that unauthenticated requests are rejected
func (s *CollectionsTestSuite) TestCreateCollectionUnauthorized() {
	req := map[string]interface{}{
		"name":        "Unauthorized Test",
		"description": "Should fail",
	}

	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/collections", "", req)
	defer resp.Body.Close()

	requireUnauthorized(s.T(), resp)
}

// TestCollectionsSuite runs all collection tests
func TestCollectionsSuite(t *testing.T) {
	suite.Run(t, new(CollectionsTestSuite))
}
