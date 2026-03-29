package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ModelsTestSuite tests the /api/models endpoints.
type ModelsTestSuite struct {
	suite.Suite
	client    *http.Client
	baseURL   string
	authToken string
	modelID   string // a model created by build callback for lifecycle tests
}

func (s *ModelsTestSuite) SetupSuite() {
	s.client = newAPIClient()
	s.baseURL = testServerURL
	s.authToken = authenticate(s.T(), s.client, s.baseURL, "michael.jordan", "111222333")
	s.modelID = s.seedModel(s.T())
}

func (s *ModelsTestSuite) TearDownSuite() {
	truncateAllTables(s.T())
}

// seedModel creates a model via the build callback shortcut.
func (s *ModelsTestSuite) seedModel(t *testing.T) string {
	collID := s.createCollection(t, "Models Suite Collection")
	dsID := s.createDatasource(t, collID)

	// Create and start a build
	buildResp := s.createBuild(t, dsID, "Models Suite Build")
	buildID := buildResp["id"].(string)

	startResp := makeRequest(t, s.client, "POST", s.baseURL+"/api/builds/"+buildID+"/start", s.authToken, nil)
	requireSuccess(t, startResp)
	startResp.Body.Close()

	// Inject callback to simulate compute finishing
	modelPath := "models/random_forest/test_model.pkl"
	codePath := "models/random_forest/train.py"
	callbackReq := map[string]interface{}{
		"build_id":            buildID,
		"job_id":              "mock-job-" + buildID,
		"status":              "completed",
		"model_path":          modelPath,
		"code_path":           codePath,
		"metrics":             map[string]interface{}{"r2": 0.88, "rmse": 0.15},
		"feature_names":       []string{"LOAN", "MORTDUE", "VALUE"},
		"feature_importances": map[string]float64{"LOAN": 0.5, "MORTDUE": 0.3, "VALUE": 0.2},
	}
	cbResp := makeRequest(t, s.client, "POST", s.baseURL+"/api/builds/callback", "", callbackReq)
	require.Equal(t, http.StatusOK, cbResp.StatusCode)
	cbResp.Body.Close()

	// Find the model for this build
	listResp := makeRequest(t, s.client, "GET", s.baseURL+"/api/models?page=1&page_size=50", s.authToken, nil)
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
	t.Fatal("model not found after build callback")
	return ""
}

// TestListModels verifies GET /api/models returns paginated results.
func (s *ModelsTestSuite) TestListModels() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models?page=1&page_size=10", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Models []interface{} `json:"models"`
			Total  int           `json:"total"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.GreaterOrEqual(s.T(), result.Data.Total, 1)
	assert.GreaterOrEqual(s.T(), len(result.Data.Models), 1)
}

// TestGetModel verifies GET /api/models/:id returns the model with details.
func (s *ModelsTestSuite) TestGetModel() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models/"+s.modelID, s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			ID        string `json:"id"`
			Algorithm string `json:"algorithm"`
			Status    string `json:"status"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), s.modelID, result.Data.ID)
	assert.NotEmpty(s.T(), result.Data.Algorithm)
	assert.Equal(s.T(), "draft", result.Data.Status)
}

// TestGetModel_NotFound verifies GET /api/models/:id returns 404 for unknown ID.
func (s *ModelsTestSuite) TestGetModel_NotFound() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models/00000000-0000-0000-0000-000000000099", s.authToken, nil)
	defer resp.Body.Close()
	requireNotFound(s.T(), resp)
}

// TestUpdateModel verifies PUT /api/models/:id updates name and description.
func (s *ModelsTestSuite) TestUpdateModel() {
	// Use a dedicated model so other tests are not affected
	collID := s.createCollection(s.T(), "Update Model Collection")
	dsID := s.createDatasource(s.T(), collID)
	buildResp := s.createBuild(s.T(), dsID, "Update Model Build")
	buildID := buildResp["id"].(string)
	modelID := s.completeBuild(s.T(), buildID)

	updateReq := map[string]string{"name": "Updated Model Name", "description": "Updated description"}
	resp := makeRequest(s.T(), s.client, "PUT", s.baseURL+"/api/models/"+modelID, s.authToken, updateReq)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), "Updated Model Name", result.Data.Name)
	assert.Equal(s.T(), "Updated description", result.Data.Description)
}

// TestActivateDeactivateModel verifies the model lifecycle: draft → active → inactive.
func (s *ModelsTestSuite) TestActivateDeactivateModel() {
	collID := s.createCollection(s.T(), "Lifecycle Collection")
	dsID := s.createDatasource(s.T(), collID)
	buildResp := s.createBuild(s.T(), dsID, "Lifecycle Build")
	buildID := buildResp["id"].(string)
	modelID := s.completeBuild(s.T(), buildID)

	// Activate
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+modelID+"/activate", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var activateResult struct {
		Data struct{ Status string `json:"status"` } `json:"data"`
	}
	parseResponse(s.T(), resp, &activateResult)
	assert.Equal(s.T(), "active", activateResult.Data.Status)

	// Deactivate
	resp2 := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+modelID+"/deactivate", s.authToken, nil)
	defer resp2.Body.Close()
	requireSuccess(s.T(), resp2)

	var deactivateResult struct {
		Data struct{ Status string `json:"status"` } `json:"data"`
	}
	parseResponse(s.T(), resp2, &deactivateResult)
	assert.Equal(s.T(), "inactive", deactivateResult.Data.Status)
}

// TestActivateModel_AlreadyActive verifies that activating an active model returns an error.
func (s *ModelsTestSuite) TestActivateModel_AlreadyActive() {
	collID := s.createCollection(s.T(), "Double Activate Collection")
	dsID := s.createDatasource(s.T(), collID)
	buildResp := s.createBuild(s.T(), dsID, "Double Activate Build")
	buildID := buildResp["id"].(string)
	modelID := s.completeBuild(s.T(), buildID)

	// Activate once
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+modelID+"/activate", s.authToken, nil)
	requireSuccess(s.T(), resp)
	resp.Body.Close()

	// Activate again — should fail
	resp2 := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+modelID+"/activate", s.authToken, nil)
	defer resp2.Body.Close()
	assert.Equal(s.T(), http.StatusUnprocessableEntity, resp2.StatusCode)
}

// TestDeleteModel verifies DELETE /api/models/:id removes the model.
func (s *ModelsTestSuite) TestDeleteModel() {
	collID := s.createCollection(s.T(), "Delete Model Collection")
	dsID := s.createDatasource(s.T(), collID)
	buildResp := s.createBuild(s.T(), dsID, "Delete Model Build")
	buildID := buildResp["id"].(string)
	modelID := s.completeBuild(s.T(), buildID)

	// Delete
	resp := makeRequest(s.T(), s.client, "DELETE", s.baseURL+"/api/models/"+modelID, s.authToken, nil)
	defer resp.Body.Close()
	requireNoContent(s.T(), resp)

	// Verify gone
	getResp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models/"+modelID, s.authToken, nil)
	defer getResp.Body.Close()
	requireNotFound(s.T(), getResp)
}

// TestDeleteModel_WhenActive verifies that an active model cannot be deleted.
func (s *ModelsTestSuite) TestDeleteModel_WhenActive() {
	collID := s.createCollection(s.T(), "No Delete Active Collection")
	dsID := s.createDatasource(s.T(), collID)
	buildResp := s.createBuild(s.T(), dsID, "No Delete Active Build")
	buildID := buildResp["id"].(string)
	modelID := s.completeBuild(s.T(), buildID)

	// Activate
	activateResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+modelID+"/activate", s.authToken, nil)
	requireSuccess(s.T(), activateResp)
	activateResp.Body.Close()

	// Delete should fail
	resp := makeRequest(s.T(), s.client, "DELETE", s.baseURL+"/api/models/"+modelID, s.authToken, nil)
	defer resp.Body.Close()
	assert.Equal(s.T(), http.StatusUnprocessableEntity, resp.StatusCode)
}

// TestModelsUnauthorized verifies that unauthenticated requests return 401.
func (s *ModelsTestSuite) TestModelsUnauthorized() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models", "", nil)
	defer resp.Body.Close()
	requireUnauthorized(s.T(), resp)
}

// --- helpers ---

func (s *ModelsTestSuite) createCollection(t *testing.T, name string) string {
	resp := makeRequest(t, s.client, "POST", s.baseURL+"/api/collections", s.authToken,
		map[string]string{"name": name, "description": "test"})
	defer resp.Body.Close()
	requireCreated(t, resp)
	var r struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	parseResponse(t, resp, &r)
	require.NotEmpty(t, r.Data.ID)
	return r.Data.ID
}

func (s *ModelsTestSuite) createDatasource(t *testing.T, collID string) string {
	fixturePath := getFixturePath("hmeq.csv")
	require.FileExists(t, fixturePath)
	resp := makeMultipartRequest(t, s.client, "POST", s.baseURL+"/api/datasources", s.authToken,
		map[string]string{"collection_id": collID, "name": fmt.Sprintf("DS-%s", collID[:8]), "description": "test", "type": "csv"},
		"file", fixturePath)
	defer resp.Body.Close()
	requireCreated(t, resp)
	var r struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	parseResponse(t, resp, &r)
	return r.Data.ID
}

func (s *ModelsTestSuite) createBuild(t *testing.T, dsID, name string) map[string]interface{} {
	resp := makeRequest(t, s.client, "POST", s.baseURL+"/api/builds", s.authToken, map[string]interface{}{
		"name":          name,
		"datasource_id": dsID,
		"model_type":    "regression",
		"algorithm":     "random_forest",
	})
	defer resp.Body.Close()
	requireCreated(t, resp)
	var r struct {
		Data map[string]interface{} `json:"data"`
	}
	parseResponse(t, resp, &r)
	return r.Data
}

// completeBuild starts a build then injects a successful callback, returning the model ID.
func (s *ModelsTestSuite) completeBuild(t *testing.T, buildID string) string {
	startResp := makeRequest(t, s.client, "POST", s.baseURL+"/api/builds/"+buildID+"/start", s.authToken, nil)
	requireSuccess(t, startResp)
	startResp.Body.Close()

	modelPath := "models/rf/model.pkl"
	cbResp := makeRequest(t, s.client, "POST", s.baseURL+"/api/builds/callback", "", map[string]interface{}{
		"build_id":            buildID,
		"job_id":              "mock-" + buildID,
		"status":              "completed",
		"model_path":          modelPath,
		"metrics":             map[string]interface{}{"r2": 0.9},
		"feature_names":       []string{"LOAN"},
		"feature_importances": map[string]float64{"LOAN": 1.0},
	})
	require.Equal(t, http.StatusOK, cbResp.StatusCode)
	cbResp.Body.Close()

	listResp := makeRequest(t, s.client, "GET", s.baseURL+"/api/models?page=1&page_size=100", s.authToken, nil)
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
	t.Fatal("model not found after build callback")
	return ""
}

func TestModelsSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}
