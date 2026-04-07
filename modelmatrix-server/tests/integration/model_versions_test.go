package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ModelVersionsTestSuite tests model version and retrain APIs
type ModelVersionsTestSuite struct {
	suite.Suite
	client    *http.Client
	baseURL   string
	authToken string
	modelID   string
}

func (s *ModelVersionsTestSuite) SetupSuite() {
	s.client = newAPIClient()
	s.baseURL = testServerURL
	s.authToken = authenticate(s.T(), s.client, s.baseURL, "michael.jordan", "111222333")
	s.modelID = s.createModelViaBuildCallback(s.T())
}

func (s *ModelVersionsTestSuite) TearDownSuite() {
	truncateAllTables(s.T())
}

// createModelViaBuildCallback creates a build, starts it, then sends a fake callback to create a model
func (s *ModelVersionsTestSuite) createModelViaBuildCallback(t *testing.T) string {
	collectionID := s.createTestCollection(t, "Versions Test Collection", "For version tests")
	datasourceID := s.createTestDatasource(t, collectionID)

	createBuildReq := map[string]interface{}{
		"name":          "Build for version test",
		"description":   "Build to produce one model",
		"datasource_id": datasourceID,
		"model_type":    "regression",
		"algorithm":     "random_forest",
	}
	resp := makeRequest(t, s.client, "POST", s.baseURL+"/api/builds", s.authToken, createBuildReq)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var buildResult struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	parseResponse(t, resp, &buildResult)
	buildID := buildResult.Data.ID
	require.NotEmpty(t, buildID)

	// Start build (mock compute returns immediately)
	startResp := makeRequest(t, s.client, "POST", s.baseURL+"/api/builds/"+buildID+"/start", s.authToken, nil)
	defer startResp.Body.Close()
	require.Equal(t, http.StatusOK, startResp.StatusCode)

	// Callback to create model (no auth)
	callbackReq := map[string]interface{}{
		"build_id": buildID,
		"job_id":   "mock-job-id",
		"status":   "completed",
		"model_path": "models/random_forest/test_model.pkl",
		"code_path":  "models/random_forest/test_train.py",
		"metrics": map[string]interface{}{
			"accuracy": 0.9,
			"r2":       0.85,
		},
		"feature_names": []string{"a", "b", "c"},
		"feature_importances": map[string]float64{"a": 0.5, "b": 0.3, "c": 0.2},
	}
	callbackResp := makeRequest(t, s.client, "POST", s.baseURL+"/api/builds/callback", "", callbackReq)
	defer callbackResp.Body.Close()
	require.Equal(t, http.StatusOK, callbackResp.StatusCode)

	// Find model by build_id
	listResp := makeRequest(t, s.client, "GET", s.baseURL+"/api/models?page=1&page_size=10", s.authToken, nil)
	defer listResp.Body.Close()
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	var listResult struct {
		Data struct {
			Models []struct {
				ID      string `json:"id"`
				BuildID string `json:"build_id"`
			} `json:"models"`
		} `json:"data"`
	}
	parseResponse(t, listResp, &listResult)
	for _, m := range listResult.Data.Models {
		if m.BuildID == buildID {
			return m.ID
		}
	}
	t.Fatal("model not found after callback")
	return ""
}

func (s *ModelVersionsTestSuite) createTestCollection(t *testing.T, name, description string) string {
	req := map[string]interface{}{
		"name":        name,
		"description": description,
	}
	resp := makeRequest(t, s.client, "POST", s.baseURL+"/api/collections", s.authToken, req)
	defer resp.Body.Close()
	requireCreated(t, resp)
	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	parseResponse(t, resp, &result)
	require.NotEmpty(t, result.Data.ID)
	return result.Data.ID
}

func (s *ModelVersionsTestSuite) createTestDatasource(t *testing.T, collectionID string) string {
	fixturePath := getFixturePath("hmeq.csv")
	require.FileExists(t, fixturePath)
	formData := map[string]string{
		"collection_id": collectionID,
		"name":          "DS for version test",
		"description":   "Datasource for version test",
		"type":          "csv",
	}
	resp := makeMultipartRequest(t, s.client, "POST", s.baseURL+"/api/datasources", s.authToken, formData, "file", fixturePath)
	defer resp.Body.Close()
	requireCreated(t, resp)
	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	parseResponse(t, resp, &result)
	ensureTrainingColumnRoles(t, s.client, s.baseURL, s.authToken, result.Data.ID)
	return result.Data.ID
}

// TestCreateVersion_WithoutUploadedArtifactsFails documents integration reality: build callbacks in these tests use
// synthetic model paths (no bytes in MinIO). CreateVersion copies from storage and correctly returns 5xx until
// artifacts exist — this is not masked as success. A future enhancement is to upload fixture bytes in tests and
// then assert 201 + GET + restore.
func (s *ModelVersionsTestSuite) TestCreateVersion_WithoutUploadedArtifactsFails() {
	if s.modelID == "" {
		s.T().Skip("no model from build callback")
		return
	}
	createResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+s.modelID+"/versions", s.authToken, nil)
	defer createResp.Body.Close()
	body, err := io.ReadAll(createResp.Body)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusInternalServerError, createResp.StatusCode,
		"expected 500 when model file keys are missing from object storage; body=%s", string(body))

	var errPayload struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	require.NoError(s.T(), json.Unmarshal(body, &errPayload))
	assert.Equal(s.T(), 500, errPayload.Code)
	assert.Contains(s.T(), errPayload.Msg, "version file")
}

func (s *ModelVersionsTestSuite) TestListVersions() {
	if s.modelID == "" {
		s.T().Skip("no model from build callback")
		return
	}
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models/"+s.modelID+"/versions?page=1&page_size=10", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)
	var result struct {
		Data struct {
			Versions []interface{} `json:"versions"`
			Total    int64          `json:"total"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.NotNil(s.T(), result.Data.Versions)
}

func (s *ModelVersionsTestSuite) TestGetVersionNotFound() {
	if s.modelID == "" {
		s.T().Skip("no model from build callback")
		return
	}
	fakeVersionID := "00000000-0000-0000-0000-000000000001"
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models/"+s.modelID+"/versions/"+fakeVersionID, s.authToken, nil)
	defer resp.Body.Close()
	requireNotFound(s.T(), resp)
}

func (s *ModelVersionsTestSuite) TestRetrain() {
	if s.modelID == "" {
		s.T().Skip("no model from build callback")
		return
	}
	body := map[string]interface{}{}
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+s.modelID+"/retrain", s.authToken, body)
	defer resp.Body.Close()
	// 202 Accepted when retrain started
	assert.Equal(s.T(), http.StatusAccepted, resp.StatusCode)
	var result struct {
		Data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.NotEmpty(s.T(), result.Data.ID)
}

func (s *ModelVersionsTestSuite) TestRetrainInvalidModel() {
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/00000000-0000-0000-0000-000000000099/retrain", s.authToken, nil)
	defer resp.Body.Close()
	requireNotFound(s.T(), resp)
}

func TestModelVersionsSuite(t *testing.T) {
	suite.Run(t, new(ModelVersionsTestSuite))
}
