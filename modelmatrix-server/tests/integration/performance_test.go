package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PerformanceTestSuite tests /api/models/:id/performance/* endpoints.
type PerformanceTestSuite struct {
	suite.Suite
	client       *http.Client
	baseURL      string
	authToken    string
	modelID      string
	datasourceID string
}

func (s *PerformanceTestSuite) SetupSuite() {
	s.client = newAPIClient()
	s.baseURL = testServerURL
	s.authToken = authenticate(s.T(), s.client, s.baseURL, "michael.jordan", "111222333")
	s.datasourceID, s.modelID = s.seedModelAndDatasource(s.T())
}

func (s *PerformanceTestSuite) TearDownSuite() {
	truncateAllTables(s.T())
}

// seedModelAndDatasource creates a model via build callback for performance tests.
func (s *PerformanceTestSuite) seedModelAndDatasource(t *testing.T) (dsID, modelID string) {
	colResp := makeRequest(t, s.client, "POST", s.baseURL+"/api/collections", s.authToken,
		map[string]string{"name": "Perf Suite Collection", "description": "for perf tests"})
	requireCreated(t, colResp)
	var colResult struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	parseResponse(t, colResp, &colResult)

	fixturePath := getFixturePath("hmeq.csv")
	require.FileExists(t, fixturePath)
	dsResp := makeMultipartRequest(t, s.client, "POST", s.baseURL+"/api/datasources", s.authToken,
		map[string]string{"collection_id": colResult.Data.ID, "name": "Perf DS", "description": "for perf tests", "type": "csv"},
		"file", fixturePath)
	requireCreated(t, dsResp)
	var dsResult struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	parseResponse(t, dsResp, &dsResult)
	dsID = dsResult.Data.ID
	ensureTrainingColumnRoles(t, s.client, s.baseURL, s.authToken, dsID)

	buildResp := makeRequest(t, s.client, "POST", s.baseURL+"/api/builds", s.authToken, map[string]interface{}{
		"name": "Perf Suite Build", "datasource_id": dsID, "model_type": "regression", "algorithm": "random_forest",
	})
	requireCreated(t, buildResp)
	var buildResult struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	parseResponse(t, buildResp, &buildResult)
	buildID := buildResult.Data.ID

	startResp := makeRequest(t, s.client, "POST", s.baseURL+"/api/builds/"+buildID+"/start", s.authToken, nil)
	requireSuccess(t, startResp)
	startResp.Body.Close()

	modelPath := "models/rf/model.pkl"
	cbResp := makeRequest(t, s.client, "POST", s.baseURL+"/api/builds/callback", "", map[string]interface{}{
		"build_id": buildID, "job_id": "perf-mock-job", "status": "completed",
		"model_path":          modelPath,
		"metrics":             map[string]interface{}{"r2": 0.88, "rmse": 0.12},
		"feature_names":       []string{"LOAN", "MORTDUE"},
		"feature_importances": map[string]float64{"LOAN": 0.6, "MORTDUE": 0.4},
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
			modelID = m.ID
			return
		}
	}
	t.Fatal("model not found after build callback in perf setup")
	return
}

// TestGetPerformanceSummary verifies GET /api/models/:id/performance returns a summary.
func (s *PerformanceTestSuite) TestGetPerformanceSummary() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models/"+s.modelID+"/performance", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			ModelID string `json:"model_id"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), s.modelID, result.Data.ModelID)
}

// TestCreateBaseline verifies POST /api/models/:id/performance/baselines creates a baseline.
func (s *PerformanceTestSuite) TestCreateBaseline() {
	req := map[string]interface{}{
		"metrics":      map[string]float64{"r2": 0.85, "rmse": 0.15},
		"sample_count": 500,
		"description":  "Initial production baseline",
	}
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+s.modelID+"/performance/baselines", s.authToken, req)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Baselines []struct {
				ModelID     string  `json:"model_id"`
				MetricName  string  `json:"metric_name"`
				MetricValue float64 `json:"metric_value"`
			} `json:"baselines"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.NotEmpty(s.T(), result.Data.Baselines)
	assert.Equal(s.T(), s.modelID, result.Data.Baselines[0].ModelID)
}

// TestGetBaselines verifies GET /api/models/:id/performance/baselines returns baselines.
func (s *PerformanceTestSuite) TestGetBaselines() {
	// Seed a baseline first
	seedReq := map[string]interface{}{
		"metrics":      map[string]float64{"r2": 0.9},
		"sample_count": 200,
	}
	seedResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+s.modelID+"/performance/baselines", s.authToken, seedReq)
	requireSuccess(s.T(), seedResp)
	seedResp.Body.Close()

	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models/"+s.modelID+"/performance/baselines", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Baselines []interface{} `json:"baselines"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.NotEmpty(s.T(), result.Data.Baselines)
}

// TestRecordPerformance verifies POST /api/models/:id/performance/record saves a record.
func (s *PerformanceTestSuite) TestRecordPerformance() {
	req := map[string]interface{}{
		"datasource_id": s.datasourceID,
		"metrics":       map[string]float64{"r2": 0.82, "rmse": 0.18},
		"sample_count":  300,
	}
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+s.modelID+"/performance/record", s.authToken, req)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Records []interface{} `json:"records"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.NotEmpty(s.T(), result.Data.Records)
}

// TestGetPerformanceHistory verifies GET /api/models/:id/performance/history returns records.
func (s *PerformanceTestSuite) TestGetPerformanceHistory() {
	// Seed a performance record
	seedReq := map[string]interface{}{
		"datasource_id": s.datasourceID,
		"metrics":       map[string]float64{"r2": 0.80},
		"sample_count":  150,
	}
	seedResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+s.modelID+"/performance/record", s.authToken, seedReq)
	requireSuccess(s.T(), seedResp)
	seedResp.Body.Close()

	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models/"+s.modelID+"/performance/history", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Records    []interface{} `json:"records"`
			TotalCount int           `json:"total_count"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.GreaterOrEqual(s.T(), result.Data.TotalCount, 1)
}

// TestGetAlerts verifies GET /api/models/:id/performance/alerts returns the alert list.
func (s *PerformanceTestSuite) TestGetAlerts() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models/"+s.modelID+"/performance/alerts", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Alerts     []interface{} `json:"alerts"`
			TotalCount int           `json:"total_count"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.NotNil(s.T(), result.Data.Alerts)
}

// TestGetThresholds verifies GET /api/models/:id/performance/thresholds returns thresholds.
func (s *PerformanceTestSuite) TestGetThresholds() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models/"+s.modelID+"/performance/thresholds", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Thresholds []interface{} `json:"thresholds"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.NotNil(s.T(), result.Data.Thresholds)
}

// TestGetGlobalThresholdDefaults verifies GET /api/performance/threshold-defaults.
func (s *PerformanceTestSuite) TestGetGlobalThresholdDefaults() {
	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/performance/threshold-defaults?task_type=regression", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			TaskType string        `json:"task_type"`
			Defaults []interface{} `json:"defaults"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), "regression", result.Data.TaskType)
}

// TestStartEvaluation verifies POST /api/models/:id/performance/evaluate starts an evaluation.
func (s *PerformanceTestSuite) TestStartEvaluation() {
	req := map[string]interface{}{
		"datasource_id": s.datasourceID,
		"actual_column": "BAD",
	}
	resp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+s.modelID+"/performance/evaluate", s.authToken, req)
	defer resp.Body.Close()

	var result struct {
		Data struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	// 201 or 202 depending on implementation
	assert.True(s.T(), resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted,
		"expected 201 or 202, got %d", resp.StatusCode)
	parseResponse(s.T(), resp, &result)
	assert.NotEmpty(s.T(), result.Data.ID)
}

// TestGetEvaluations verifies GET /api/models/:id/performance/evaluations returns evaluations.
func (s *PerformanceTestSuite) TestGetEvaluations() {
	resp := makeRequest(s.T(), s.client, "GET", fmt.Sprintf("%s/api/models/%s/performance/evaluations", s.baseURL, s.modelID), s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)

	var result struct {
		Data struct {
			Evaluations []interface{} `json:"evaluations"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.NotNil(s.T(), result.Data.Evaluations)
}

// TestGetEvaluationByID verifies GET /api/models/:id/performance/evaluations/:evaluationId after starting an evaluation.
func (s *PerformanceTestSuite) TestGetEvaluationByID() {
	req := map[string]interface{}{
		"datasource_id": s.datasourceID,
		"actual_column": "BAD",
	}
	startResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+s.modelID+"/performance/evaluate", s.authToken, req)
	defer startResp.Body.Close()
	require.True(s.T(), startResp.StatusCode == http.StatusCreated || startResp.StatusCode == http.StatusAccepted,
		"start evaluation: status %d", startResp.StatusCode)

	var started struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	parseResponse(s.T(), startResp, &started)
	evalID := started.Data.ID
	require.NotEmpty(s.T(), evalID)

	getResp := makeRequest(s.T(), s.client, "GET",
		fmt.Sprintf("%s/api/models/%s/performance/evaluations/%s", s.baseURL, s.modelID, evalID), s.authToken, nil)
	defer getResp.Body.Close()
	requireSuccess(s.T(), getResp)
	var detail struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	parseResponse(s.T(), getResp, &detail)
	assert.Equal(s.T(), evalID, detail.Data.ID)
}

// TestGetMetricTimeSeries verifies GET .../metrics/:metricName/series after recording metrics.
func (s *PerformanceTestSuite) TestGetMetricTimeSeries() {
	seedReq := map[string]interface{}{
		"datasource_id": s.datasourceID,
		"metrics":       map[string]float64{"r2": 0.81},
		"sample_count":  100,
	}
	seedResp := makeRequest(s.T(), s.client, "POST", s.baseURL+"/api/models/"+s.modelID+"/performance/record", s.authToken, seedReq)
	requireSuccess(s.T(), seedResp)
	seedResp.Body.Close()

	resp := makeRequest(s.T(), s.client, "GET", s.baseURL+"/api/models/"+s.modelID+"/performance/metrics/r2/series?limit=10", s.authToken, nil)
	defer resp.Body.Close()
	requireSuccess(s.T(), resp)
	var result struct {
		Data struct {
			MetricName string `json:"metric_name"`
		} `json:"data"`
	}
	parseResponse(s.T(), resp, &result)
	assert.Equal(s.T(), "r2", result.Data.MetricName)
}

func TestPerformanceSuite(t *testing.T) {
	suite.Run(t, new(PerformanceTestSuite))
}
