package integration

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	buildModel "modelmatrix-server/internal/module/build/model"
	"modelmatrix-server/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Package-level state shared by all suites within the integration package.
// Initialized by TestMain in main_test.go.
var (
	testDB        *gorm.DB
	testRouter    *gin.Engine    //nolint:unused
	testConfig    *config.Config //nolint:unused
	testServerURL string
	testServer    *httptest.Server //nolint:unused
)

// computeAvailable returns true if TEST_COMPUTE_URL is set and reachable.
func computeAvailable() bool {
	url := os.Getenv("TEST_COMPUTE_URL")
	if url == "" {
		return false
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url + "/compute/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	resp.Body.Close()
	return true
}

// getEnv returns the value of an environment variable or a default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// authenticate performs a real LDAP-backed login and returns the JWT token.
func authenticate(t *testing.T, client *http.Client, baseURL, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := client.Post(baseURL+"/api/auth/login", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	resp.Body.Close()
	require.Equal(t, 200, result.Code)
	require.NotEmpty(t, result.Data.Token)
	return result.Data.Token
}

// makeRequest is a helper to make authenticated HTTP requests.
func makeRequest(t *testing.T, client *http.Client, method, url, token string, body interface{}) *http.Response {
	t.Helper()
	var req *http.Request
	var err error

	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		req, err = http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		require.NoError(t, err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

// makeMultipartRequest makes an authenticated multipart/form-data request.
func makeMultipartRequest(t *testing.T, client *http.Client, method, url, token string, formData map[string]string, fileField, filePath string) *http.Response {
	t.Helper()
	fileData, err := os.ReadFile(filePath)
	require.NoError(t, err, "Failed to read file: %s", filePath)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range formData {
		require.NoError(t, writer.WriteField(key, value))
	}
	part, err := writer.CreateFormFile(fileField, filepath.Base(filePath))
	require.NoError(t, err)
	_, err = part.Write(fileData)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, err := http.NewRequest(method, url, &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

// parseResponse decodes JSON response body into result.
func parseResponse(t *testing.T, resp *http.Response, result interface{}) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(result), "Failed to parse response")
}

// HTTP status assertion helpers
func requireSuccess(t *testing.T, resp *http.Response) {
	t.Helper()
	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")
}
func requireNoContent(t *testing.T, resp *http.Response) {
	t.Helper()
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "Expected 204 No Content")
}
func requireCreated(t *testing.T, resp *http.Response) {
	t.Helper()
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Expected 201 Created")
}
func requireBadRequest(t *testing.T, resp *http.Response) {
	t.Helper()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Expected 400 Bad Request")
}
func requireUnauthorized(t *testing.T, resp *http.Response) {
	t.Helper()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Expected 401 Unauthorized")
}
func requireNotFound(t *testing.T, resp *http.Response) {
	t.Helper()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Expected 404 Not Found")
}

// truncateAllTables removes all data from all application tables.
// This is ~50× faster than DROP SCHEMA CASCADE + re-migrate.
func truncateAllTables(t *testing.T) {
	t.Helper()
	if testDB == nil {
		return
	}
	// Table names must match GORM TableName() on models (see internal/module/inventory/model/performance.go).
	err := testDB.Exec(`TRUNCATE TABLE
		model_performance_evaluations,
		model_performance_alerts,
		model_performance_records,
		model_performance_baselines,
		model_performance_thresholds,
		performance_threshold_defaults,
		model_version_files,
		model_version_variables,
		model_versions,
		model_files,
		model_variables,
		models,
		model_builds,
		datasource_columns,
		datasources,
		collections,
		projects,
		folders
		CASCADE`).Error
	if err != nil {
		t.Logf("Warning: truncate failed (first run or empty): %v", err)
	}
}

// waitForBuildStatus polls the database until the build reaches the expected status or times out.
// Use this for tests that trigger real async compute jobs.
func waitForBuildStatus(t *testing.T, buildID string, expectedStatus string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var build buildModel.ModelBuildModel
		if err := testDB.First(&build, "id = ?", buildID).Error; err == nil {
			if build.Status == expectedStatus {
				return
			}
			if build.Status == "failed" || build.Status == "cancelled" {
				t.Fatalf("build %s reached terminal status %q while waiting for %q; error: %s",
					buildID, build.Status, expectedStatus, build.ErrorMessage)
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("build %s did not reach status %q within %s", buildID, expectedStatus, timeout)
}

// getFixturePath returns the absolute path to a file in tests/testdata/.
func getFixturePath(filename string) string {
	possiblePaths := []string{
		filepath.Join("tests", "testdata", filename),
		filepath.Join("testdata", filename),
		filepath.Join("..", "testdata", filename),
		filepath.Join("..", "..", "tests", "testdata", filename),
	}
	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			abs, err := filepath.Abs(p)
			if err == nil {
				return abs
			}
			return p
		}
	}
	return filepath.Join("tests", "testdata", filename)
}

// skipIfNoCompute skips the test if the compute service is not available.
func skipIfNoCompute(t *testing.T) {
	t.Helper()
	if !computeAvailable() {
		t.Skip("compute service not available; set TEST_COMPUTE_URL to enable")
	}
}

// dbInsert is a helper to directly insert a GORM model into the test database.
func dbInsert(t *testing.T, value interface{}) {
	t.Helper()
	require.NoError(t, testDB.Create(value).Error)
}

// newAPIClient returns a fresh http.Client for tests.
func newAPIClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

// ensureTrainingColumnRoles sets one column as target (prefers "BAD" on HMEQ-style CSVs) and the rest as input
// so POST /api/builds/:id/start can resolve target/input columns.
func ensureTrainingColumnRoles(t *testing.T, client *http.Client, baseURL, token, datasourceID string) {
	t.Helper()
	resp := makeRequest(t, client, "GET", baseURL+"/api/datasources/"+datasourceID+"/columns", token, nil)
	defer resp.Body.Close()
	requireSuccess(t, resp)
	var result struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	parseResponse(t, resp, &result)
	require.NotEmpty(t, result.Data, "datasource should have columns")

	targetID := result.Data[0].ID
	for _, c := range result.Data {
		if c.Name == "BAD" {
			targetID = c.ID
			break
		}
	}
	updates := make([]map[string]string, 0, len(result.Data))
	for _, c := range result.Data {
		role := "input"
		if c.ID == targetID {
			role = "target"
		}
		updates = append(updates, map[string]string{"column_id": c.ID, "role": role})
	}
	putResp := makeRequest(t, client, "PUT", baseURL+"/api/datasources/"+datasourceID+"/columns/roles", token,
		map[string]interface{}{"columns": updates})
	defer putResp.Body.Close()
	requireSuccess(t, putResp)
}
