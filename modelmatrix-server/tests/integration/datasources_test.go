package integration

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// DatasourcesTestSuite is a test suite for datasource endpoints
type DatasourcesTestSuite struct {
	suite.Suite
	client    *http.Client
	baseURL   string
	authToken string
}

// SetupSuite runs once before all tests
func (s *DatasourcesTestSuite) SetupSuite() {
	s.client = newAPIClient()
	s.baseURL = testServerURL
	s.authToken = authenticate(s.T(), s.client, s.baseURL, "michael.jordan", "111222333")
}

// TearDownSuite runs once after all tests
func (s *DatasourcesTestSuite) TearDownSuite() {
	truncateAllTables(s.T())
}

// SetupTest runs before each test
func (s *DatasourcesTestSuite) SetupTest() {
	// Optional: Clean up specific test data before each test
}

// createTestCollection is a helper to create a collection for testing
func (s *DatasourcesTestSuite) createTestCollection(name, description string) string {
	req := map[string]interface{}{
		"name":        name,
		"description": description,
	}

	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/collections", s.authToken, req)
	defer resp.Body.Close()

	requireCreated(s.T(), resp)

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

	require.Equal(s.T(), 200, result.Code)
	require.NotEmpty(s.T(), result.Data.ID)
	return result.Data.ID
}

// getFixturePathForDatasource is a local alias for testing - use getFixturePath from setup.go
// Kept for backward compatibility with this file
func getFixturePathForDatasource(filename string) string {
	return getFixturePath(filename)
}

// TestCreateDatasourceWithCSVFile tests POST /api/datasources with multipart/form-data (CSV file)
func (s *DatasourcesTestSuite) TestCreateDatasourceWithCSVFile() {
	// First, create a collection
	collectionID := s.createTestCollection("Test Collection for Datasource", "Collection for CSV datasource test")

	// Get the path to the test fixture file
	fixturePath := getFixturePath("hmeq.csv")
	require.FileExists(s.T(), fixturePath, "Fixture file hmeq.csv should exist")

	// Prepare form data
	formData := map[string]string{
		"collection_id": collectionID,
		"name":          "HMEQ Dataset",
		"description":   "Home Equity dataset for testing",
		"type":          "csv",
	}

	// Make multipart request with file
	resp := makeMultipartRequest(s.T(), s.client, "POST", s.baseURL+"/api/datasources", s.authToken, formData, "file", fixturePath)
	defer resp.Body.Close()

	requireCreated(s.T(), resp) // HTTP status should be 201

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ID               string `json:"id"`
			CollectionID     string `json:"collection_id"`
			CollectionName   string `json:"collection_name"`
			Name             string `json:"name"`
			Description      string `json:"description"`
			Type             string `json:"type"`
			FilePath         string `json:"file_path,omitempty"`
			ConnectionConfig *struct {
				Host     string `json:"host"`
				Port     int    `json:"port"`
				Database string `json:"database"`
				Username string `json:"username"`
				Schema   string `json:"schema"`
				Table    string `json:"table"`
				SSLMode  string `json:"sslmode"`
			} `json:"connection_config,omitempty"`
			ColumnCount int `json:"column_count"`
			CreatedBy   string `json:"created_by"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)

	// Note: HTTP status is 201, but JSON response code is 200 (CodeSuccess)
	assert.Equal(s.T(), 200, result.Code)
	assert.NotEmpty(s.T(), result.Data.ID, "Datasource ID should be set")
	assert.Equal(s.T(), collectionID, result.Data.CollectionID, "Collection ID should match")
	assert.Equal(s.T(), "HMEQ Dataset", result.Data.Name)
	assert.Equal(s.T(), "Home Equity dataset for testing", result.Data.Description)
	assert.Equal(s.T(), "csv", result.Data.Type)
	assert.NotEmpty(s.T(), result.Data.FilePath, "File path should be set")
	assert.Greater(s.T(), result.Data.ColumnCount, 0, "Should have columns detected")
}

// TestCreateDatasourceWithPostgreSQLConnection tests POST /api/datasources with JSON (PostgreSQL database)
func (s *DatasourcesTestSuite) TestCreateDatasourceWithPostgreSQLConnection() {
	// First, create a collection
	collectionID := s.createTestCollection("Test Collection for PostgreSQL", "Collection for PostgreSQL datasource test")

	// Prepare JSON request with connection_config
	req := map[string]interface{}{
		"collection_id": collectionID,
		"name":          "iris",
		"description":   "iris description",
		"type":          "postgresql",
		"connection_config": map[string]interface{}{
			"host":     "localhost",
			"port":     5432,
			"database": "datasets",
			"username": "postgres",
			"password": "dayang",
			"schema":   "public",
			"table":    "iris",
			"sslmode":  "disable",
		},
	}

	// Make JSON request
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/datasources", s.authToken, req)
	defer resp.Body.Close()

	requireCreated(s.T(), resp) // HTTP status should be 201

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			ID               string `json:"id"`
			CollectionID     string `json:"collection_id"`
			CollectionName   string `json:"collection_name"`
			Name             string `json:"name"`
			Description      string `json:"description"`
			Type             string `json:"type"`
			FilePath         string `json:"file_path,omitempty"`
			ConnectionConfig *struct {
				Host     string `json:"host"`
				Port     int    `json:"port"`
				Database string `json:"database"`
				Username string `json:"username"`
				Schema   string `json:"schema"`
				Table    string `json:"table"`
				SSLMode  string `json:"sslmode"`
			} `json:"connection_config,omitempty"`
			ColumnCount int    `json:"column_count"`
			CreatedBy   string `json:"created_by"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)

	// Note: HTTP status is 201, but JSON response code is 200 (CodeSuccess)
	assert.Equal(s.T(), 200, result.Code)
	assert.NotEmpty(s.T(), result.Data.ID, "Datasource ID should be set")
	assert.Equal(s.T(), collectionID, result.Data.CollectionID, "Collection ID should match")
	assert.Equal(s.T(), "iris", result.Data.Name)
	assert.Equal(s.T(), "iris description", result.Data.Description)
	assert.Equal(s.T(), "postgresql", result.Data.Type)
	
	// Verify connection_config is present and correct
	require.NotNil(s.T(), result.Data.ConnectionConfig, "Connection config should be present")
	assert.Equal(s.T(), "localhost", result.Data.ConnectionConfig.Host)
	assert.Equal(s.T(), 5432, result.Data.ConnectionConfig.Port)
	assert.Equal(s.T(), "datasets", result.Data.ConnectionConfig.Database)
	assert.Equal(s.T(), "postgres", result.Data.ConnectionConfig.Username)
	assert.Equal(s.T(), "public", result.Data.ConnectionConfig.Schema)
	assert.Equal(s.T(), "iris", result.Data.ConnectionConfig.Table)
	assert.Equal(s.T(), "disable", result.Data.ConnectionConfig.SSLMode)
	
	// For database types, file_path should be set after data is fetched and converted to parquet
	assert.NotEmpty(s.T(), result.Data.FilePath, "File path should be set after data fetch")
	assert.Greater(s.T(), result.Data.ColumnCount, 0, "Should have columns detected from database")
}

// TestCreateDatasourceWithPostgreSQLConnectionMissingConfig tests that connection_config is required for database types
func (s *DatasourcesTestSuite) TestCreateDatasourceWithPostgreSQLConnectionMissingConfig() {
	// First, create a collection
	collectionID := s.createTestCollection("Test Collection for Missing Config", "Collection for missing config test")

	// Prepare JSON request without connection_config
	req := map[string]interface{}{
		"collection_id": collectionID,
		"name":          "Missing Config Dataset",
		"description":   "Should fail without connection_config",
		"type":          "postgresql",
	}

	// Make JSON request
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/datasources", s.authToken, req)
	defer resp.Body.Close()

	requireBadRequest(s.T(), resp)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	parseResponse(s.T(), resp, &result)

	assert.Equal(s.T(), 400, result.Code)
	assert.Contains(s.T(), result.Msg, "connection_config is required", "Error message should mention connection_config is required")
}

// TestCreateDatasourceWithCSVFileMissingFile tests that file is required for CSV type
func (s *DatasourcesTestSuite) TestCreateDatasourceWithCSVFileMissingFile() {
	// First, create a collection
	collectionID := s.createTestCollection("Test Collection for Missing File", "Collection for missing file test")

	// Prepare form data without file
	formData := map[string]string{
		"collection_id": collectionID,
		"name":          "Missing File Dataset",
		"description":   "Should fail without file",
		"type":          "csv",
	}

	// Make multipart request without file (this will fail)
	// Create a minimal multipart request without a file
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for key, value := range formData {
		err := writer.WriteField(key, value)
		require.NoError(s.T(), err)
	}
	err := writer.Close()
	require.NoError(s.T(), err)

	req, err := http.NewRequest("POST", s.baseURL+"/api/datasources", &body)
	require.NoError(s.T(), err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	resp, err := s.client.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	requireBadRequest(s.T(), resp)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	parseResponse(s.T(), resp, &result)

	assert.Equal(s.T(), 400, result.Code)
	assert.Contains(s.T(), result.Msg, "file is required", "Error message should mention file is required")
}

// TestCreateDatasourceWithCSVFileInvalidCollection tests with invalid collection ID
func (s *DatasourcesTestSuite) TestCreateDatasourceWithCSVFileInvalidCollection() {
	// Get the path to the test fixture file
	fixturePath := getFixturePath("hmeq.csv")
	require.FileExists(s.T(), fixturePath, "Fixture file hmeq.csv should exist")

	// Prepare form data with invalid collection ID
	formData := map[string]string{
		"collection_id": "invalid-uuid",
		"name":          "Invalid Collection Dataset",
		"description":   "Should fail with invalid collection",
		"type":          "csv",
	}

	// Make multipart request with file
	resp := makeMultipartRequest(s.T(), s.client, "POST", s.baseURL+"/api/datasources", s.authToken, formData, "file", fixturePath)
	defer resp.Body.Close()

	requireBadRequest(s.T(), resp)
}

// TestCreateDatasourceWithCSVFileUnauthorized tests that unauthenticated requests are rejected
func (s *DatasourcesTestSuite) TestCreateDatasourceWithCSVFileUnauthorized() {
	// Get the path to the test fixture file
	fixturePath := getFixturePath("hmeq.csv")
	require.FileExists(s.T(), fixturePath, "Fixture file hmeq.csv should exist")

	// Create a collection first
	collectionID := s.createTestCollection("Unauthorized Test Collection", "For unauthorized test")

	// Prepare form data
	formData := map[string]string{
		"collection_id": collectionID,
		"name":          "Unauthorized Dataset",
		"description":   "Should fail without auth",
		"type":          "csv",
	}

	// Make multipart request without token
	resp := makeMultipartRequest(s.T(), s.client, "POST", s.baseURL+"/api/datasources", "", formData, "file", fixturePath)
	defer resp.Body.Close()

	requireUnauthorized(s.T(), resp)
}

// TestDatasourceHTTPLifecycle exercises list, get, column role, preview, update, and delete for a CSV datasource end-to-end.
func (s *DatasourcesTestSuite) TestDatasourceHTTPLifecycle() {
	collectionID := s.createTestCollection("HTTP Lifecycle Collection", "datasource CRUD integration")
	fixturePath := getFixturePath("hmeq.csv")
	require.FileExists(s.T(), fixturePath)

	formData := map[string]string{
		"collection_id": collectionID,
		"name":          "Lifecycle HMEQ",
		"description":   "integration lifecycle",
		"type":          "csv",
	}
	createResp := makeMultipartRequest(s.T(), s.client, "POST", s.baseURL+"/api/datasources", s.authToken, formData, "file", fixturePath)
	requireCreated(s.T(), createResp)

	var created struct {
		Data struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	parseResponse(s.T(), createResp, &created)
	dsID := created.Data.ID
	require.NotEmpty(s.T(), dsID)

	listURL := fmt.Sprintf("%s/api/datasources?collection_id=%s&page=1&page_size=20", s.baseURL, collectionID)
	listResp := makeRequest(s.T(), s.client, "GET", listURL, s.authToken, nil)
	defer listResp.Body.Close()
	requireSuccess(s.T(), listResp)
	var listResult struct {
		Data struct {
			Datasources []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"datasources"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	parseResponse(s.T(), listResp, &listResult)
	require.GreaterOrEqual(s.T(), listResult.Data.Total, int64(1))
	found := false
	for _, d := range listResult.Data.Datasources {
		if d.ID == dsID {
			found = true
			assert.Equal(s.T(), "Lifecycle HMEQ", d.Name)
			break
		}
	}
	require.True(s.T(), found, "created datasource should appear in collection-scoped list")

	getResp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/datasources/"+dsID, s.authToken, nil)
	defer getResp.Body.Close()
	requireSuccess(s.T(), getResp)
	var detail struct {
		Data struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			ColumnCount int    `json:"column_count"`
			Columns     []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Role string `json:"role"`
			} `json:"columns"`
		} `json:"data"`
	}
	parseResponse(s.T(), getResp, &detail)
	assert.Equal(s.T(), dsID, detail.Data.ID)
	assert.Greater(s.T(), detail.Data.ColumnCount, 0)
	require.NotEmpty(s.T(), detail.Data.Columns)

	colID := detail.Data.Columns[0].ID
	putRole := makeRequest(s.T(), s.client, "PUT", s.baseURL+"/api/datasources/"+dsID+"/columns/"+colID+"/role", s.authToken,
		map[string]string{"role": "input"})
	defer putRole.Body.Close()
	requireSuccess(s.T(), putRole)

	prevResp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/datasources/"+dsID+"/preview?limit=5", s.authToken, nil)
	defer prevResp.Body.Close()
	requireSuccess(s.T(), prevResp)
	var preview struct {
		Data struct {
			Columns []string                 `json:"columns"`
			Rows    []map[string]interface{} `json:"rows"`
		} `json:"data"`
	}
	parseResponse(s.T(), prevResp, &preview)
	assert.NotEmpty(s.T(), preview.Data.Columns)
	assert.NotEmpty(s.T(), preview.Data.Rows)

	newName := "Lifecycle HMEQ Renamed"
	upResp := makeRequest(s.T(), s.client, "PUT", s.baseURL+"/api/datasources/"+dsID, s.authToken,
		map[string]string{"name": newName})
	defer upResp.Body.Close()
	requireSuccess(s.T(), upResp)
	var updated struct {
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	parseResponse(s.T(), upResp, &updated)
	assert.Equal(s.T(), newName, updated.Data.Name)

	delResp := makeRequest(s.T(), s.client, "DELETE", s.baseURL+"/api/datasources/"+dsID, s.authToken, nil)
	defer delResp.Body.Close()
	requireNoContent(s.T(), delResp)

	goneResp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/datasources/"+dsID, s.authToken, nil)
	defer goneResp.Body.Close()
	requireNotFound(s.T(), goneResp)
}

// TestDatasourcesSuite runs all datasource tests
func TestDatasourcesSuite(t *testing.T) {
	suite.Run(t, new(DatasourcesTestSuite))
}

