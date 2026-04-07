package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// BuildsTestSuite tests the /api/builds endpoints.
type BuildsTestSuite struct {
	suite.Suite
	client       *http.Client
	baseURL      string
	authToken    string
	datasourceID string
}

func (s *BuildsTestSuite) SetupSuite() {
	s.client = newAPIClient()
	s.baseURL = testServerURL
	s.authToken = authenticate(s.T(), s.client, s.baseURL, "michael.jordan", "111222333")
	s.datasourceID = s.seedDatasource(s.T())
}

func (s *BuildsTestSuite) TearDownSuite() {
	truncateAllTables(s.T())
}

// seedDatasource creates a collection + CSV datasource for build tests.
func (s *BuildsTestSuite) seedDatasource(t *testing.T) string {
	colResp := makeRequest(t, s.client, "POST", s.baseURL+"/api/collections", s.authToken,
		map[string]string{"name": "Builds Suite Collection", "description": "for build tests"})
	requireCreated(t, colResp)
	var colResult struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	parseResponse(t, colResp, &colResult)

	fixturePath := getFixturePath("hmeq.csv")
	require.FileExists(t, fixturePath)
	dsResp := makeMultipartRequest(t, s.client, "POST", s.baseURL+"/api/datasources", s.authToken,
		map[string]string{
			"collection_id": colResult.Data.ID,
			"name":          "Build Suite DS",
			"description":   "for build tests",
			"type":          "csv",
		}, "file", fixturePath)
	requireCreated(t, dsResp)
	var dsResult struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	parseResponse(t, dsResp, &dsResult)
	ensureTrainingColumnRoles(t, s.client, s.baseURL, s.authToken, dsResult.Data.ID)
	return dsResult.Data.ID
}

// TestListBuilds verifies GET /api/builds returns created builds.
func (s *BuildsTestSuite) TestListBuilds() {
	buildID := s.createBuild(s.T(), "List Builds Row")

	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/builds?page=1&page_size=50", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Builds []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"builds"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	require.GreaterOrEqual(s.T(), result.Data.Total, int64(1))
	found := false
	for _, b := range result.Data.Builds {
		if b.ID == buildID {
			found = true
			break
		}
	}
	require.True(s.T(), found, "created build should appear in GET /api/builds")
}

// TestCreateBuild verifies POST /api/builds creates a pending build.
func (s *BuildsTestSuite) TestCreateBuild() {
	req := map[string]interface{}{
		"name":          "Test Build",
		"description":   "integration test build",
		"datasource_id": s.datasourceID,
		"model_type":    "regression",
		"algorithm":     "random_forest",
	}
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/builds", s.authToken, req)
	defer resp.Body.Close()
	requireCreated(s.T(), resp)

	var result struct {
		Data struct {
			ID           string `json:"id"`
			Status       string `json:"status"`
			DatasourceID string `json:"datasource_id"`
			Algorithm    string `json:"algorithm"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.NotEmpty(s.T(), result.Data.ID)
	assert.Equal(s.T(), "pending", result.Data.Status)
	assert.Equal(s.T(), s.datasourceID, result.Data.DatasourceID)
	assert.Equal(s.T(), "random_forest", result.Data.Algorithm)
}

// TestCreateBuild_MissingName verifies that a missing name returns 400.
func (s *BuildsTestSuite) TestCreateBuild_MissingName() {
	req := map[string]interface{}{
		"datasource_id": s.datasourceID,
		"model_type":    "regression",
		"algorithm":     "random_forest",
	}
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/builds", s.authToken, req)
	defer resp.Body.Close()
	requireBadRequest(s.T(), resp)
}

// TestCreateBuild_InvalidDatasource verifies that a non-existent datasource returns 400/404.
func (s *BuildsTestSuite) TestCreateBuild_InvalidDatasource() {
	req := map[string]interface{}{
		"name":          "Bad DS Build",
		"datasource_id": "00000000-0000-0000-0000-000000000001",
		"model_type":    "regression",
		"algorithm":     "random_forest",
	}
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/builds", s.authToken, req)
	defer resp.Body.Close()
	requireNotFound(s.T(), resp)
}

// TestGetBuild verifies GET /api/builds/:id returns build details.
func (s *BuildsTestSuite) TestGetBuild() {
	buildID := s.createBuild(s.T(), "Get Build Test")

	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/builds/"+buildID, s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), buildID, result.Data.ID)
}

// TestGetBuild_NotFound verifies that unknown ID returns 404.
func (s *BuildsTestSuite) TestGetBuild_NotFound() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/builds/00000000-0000-0000-0000-000000000099", s.authToken, nil)
	defer resp.Body.Close()
	requireNotFound(s.T(), resp)
}

// TestUpdateBuild verifies PUT /api/builds/:id updates name/description.
func (s *BuildsTestSuite) TestUpdateBuild() {
	buildID := s.createBuild(s.T(), "Update Build Test")
	newName := "Updated Build Name"
	newDesc := "Updated description"

	resp := makeRequest(s.T(), s.client, "PUT", s.baseURL+"/api/builds/"+buildID, s.authToken, map[string]string{
		"name":        newName,
		"description": newDesc,
	})
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), newName, result.Data.Name)
}

// TestStartBuild_MockCallback verifies start + manual callback flow (no real compute needed).
func (s *BuildsTestSuite) TestStartBuild_MockCallback() {
	buildID := s.createBuild(s.T(), "Start Callback Build")

	// Start
	startResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/builds/"+buildID+"/start", s.authToken, nil)
	defer startResp.Body.Close()
	requireSuccess(s.T(), startResp)

	// Verify status changed to running
	getResp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/builds/"+buildID, s.authToken, nil)
	defer getResp.Body.Close()
	var getResult struct {
		Data struct{ Status string `json:"status"` } `json:"data"`
	}
	parseResponse(s.T(), getResp, &getResult)
	assert.Equal(s.T(), "running", getResult.Data.Status)

	// Inject callback
	modelPath := "models/rf/model.pkl"
	cbResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/builds/callback", "", map[string]interface{}{
		"build_id":            buildID,
		"job_id":              "mock-job",
		"status":              "completed",
		"model_path":          modelPath,
		"metrics":             map[string]interface{}{"r2": 0.9},
		"feature_names":       []string{"LOAN", "MORTDUE"},
		"feature_importances": map[string]float64{"LOAN": 0.6, "MORTDUE": 0.4},
	})
	defer cbResp.Body.Close()
	require.Equal(s.T(), http.StatusOK, cbResp.StatusCode)

	// Verify build is completed
	finalResp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/builds/"+buildID, s.authToken, nil)
	defer finalResp.Body.Close()
	var finalResult struct {
		Data struct{ Status string `json:"status"` } `json:"data"`
	}
	parseResponse(s.T(), finalResp, &finalResult)
	assert.Equal(s.T(), "completed", finalResult.Data.Status)
}

// TestStartBuild_RealCompute verifies that with a real compute service the full async flow works.
// This test is skipped if TEST_COMPUTE_URL is not set.
func (s *BuildsTestSuite) TestStartBuild_RealCompute() {
	skipIfNoCompute(s.T())

	buildID := s.createBuild(s.T(), "Real Compute Build")
	startResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/builds/"+buildID+"/start", s.authToken, nil)
	requireSuccess(s.T(), startResp)
	startResp.Body.Close()

	// Wait for compute to callback and build to complete (up to 2 minutes)
	waitForBuildStatus(s.T(), buildID, "completed", 120e9)

	// Verify a model was created for this build
	listResp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models?page=1&page_size=100", s.authToken, nil)
	defer listResp.Body.Close()
	var listResult struct {
		Data struct {
			Models []struct {
				ID      string `json:"id"`
				BuildID string `json:"build_id"`
			} `json:"models"`
		} `json:"data"`
	}
	parseResponse(s.T(), listResp, &listResult)
	found := false
	for _, m := range listResult.Data.Models {
		if m.BuildID == buildID {
			found = true
			break
		}
	}
	assert.True(s.T(), found, "model should be created after real compute finishes")
}

// TestCancelBuild verifies POST /api/builds/:id/cancel transitions a pending build.
func (s *BuildsTestSuite) TestCancelBuild() {
	buildID := s.createBuild(s.T(), "Cancel Build Test")

	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/builds/"+buildID+"/cancel", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct{ Status string `json:"status"` } `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), "cancelled", result.Data.Status)
}

// TestCancelBuild_AfterComplete verifies that a completed build cannot be cancelled.
func (s *BuildsTestSuite) TestCancelBuild_AfterComplete() {
	buildID := s.createBuild(s.T(), "Cancel Completed Build")

	startResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/builds/"+buildID+"/start", s.authToken, nil)
	requireSuccess(s.T(), startResp)
	startResp.Body.Close()

	// Complete via callback
	modelPath := "models/rf/m.pkl"
	cbResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/builds/callback", "", map[string]interface{}{
		"build_id": buildID, "job_id": "j", "status": "completed",
		"model_path": modelPath, "feature_names": []string{"A"}, "feature_importances": map[string]float64{"A": 1.0},
	})
	require.Equal(s.T(), http.StatusOK, cbResp.StatusCode)
	cbResp.Body.Close()

	// Cancel should fail
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/builds/"+buildID+"/cancel", s.authToken, nil)
	defer resp.Body.Close()
	assert.NotEqual(s.T(), http.StatusOK, resp.StatusCode)
}

// TestDeleteBuild verifies DELETE /api/builds/:id removes the build.
func (s *BuildsTestSuite) TestDeleteBuild() {
	buildID := s.createBuild(s.T(), "Delete Build Test")
	resp := makeRequest(s.T(), s.client, "DELETE", s.baseURL+"/api/builds/"+buildID, s.authToken, nil)
	defer resp.Body.Close()
	requireNoContent(s.T(), resp)

	getResp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/builds/"+buildID, s.authToken, nil)
	defer getResp.Body.Close()
	requireNotFound(s.T(), getResp)
}

// TestCallback_FailedJob verifies that a failed callback marks the build as failed.
func (s *BuildsTestSuite) TestCallback_FailedJob() {
	buildID := s.createBuild(s.T(), "Callback Failed Build")

	startResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/builds/"+buildID+"/start", s.authToken, nil)
	requireSuccess(s.T(), startResp)
	startResp.Body.Close()

	errorMsg := "out of memory"
	cbResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/builds/callback", "", map[string]interface{}{
		"build_id": buildID,
		"job_id":   "mock-fail-job",
		"status":   "failed",
		"error":    errorMsg,
	})
	defer cbResp.Body.Close()
	require.Equal(s.T(), http.StatusOK, cbResp.StatusCode)

	getResp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/builds/"+buildID, s.authToken, nil)
	defer getResp.Body.Close()
	var getResult struct {
		Data struct {
			Status       string `json:"status"`
			ErrorMessage string `json:"error_message"`
		} `json:"data"`
	}
	parseResponse(s.T(), getResp, &getResult)
	assert.Equal(s.T(), "failed", getResult.Data.Status)
	assert.Contains(s.T(), getResult.Data.ErrorMessage, errorMsg)
}

func (s *BuildsTestSuite) createBuild(t *testing.T, name string) string {
	resp := makeRequest(t, s.client, "POST", s.baseURL+"/api/builds", s.authToken, map[string]interface{}{
		"name":          fmt.Sprintf("%s-%d", name, len(name)),
		"datasource_id": s.datasourceID,
		"model_type":    "regression",
		"algorithm":     "random_forest",
	})
	defer resp.Body.Close()
	requireCreated(t, resp)
	var r struct {
		Data struct{ ID string `json:"id"` } `json:"data"`
	}
	parseResponse(t, resp, &r)
	require.NotEmpty(t, r.Data.ID)
	return r.Data.ID
}

func TestBuildsSuite(t *testing.T) {
	suite.Run(t, new(BuildsTestSuite))
}
