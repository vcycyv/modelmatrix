package application

// Tests for HandleCallback, Retrain, createModelFromBuild, updateModelFromBuild, and convertMetrics.
// All dependencies are interfaces — these are core business logic functions with real decisions.

import (
	"errors"
	"testing"

	"modelmatrix-server/internal/infrastructure/compute"
	"modelmatrix-server/internal/module/build/domain"
	"modelmatrix-server/internal/module/build/dto"
	dsDto "modelmatrix-server/internal/module/datasource/dto"
	invApp "modelmatrix-server/internal/module/inventory/application"
	invDomain "modelmatrix-server/internal/module/inventory/domain"
	invDto "modelmatrix-server/internal/module/inventory/dto"
	"modelmatrix-server/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock: invApp.ModelService
// ---------------------------------------------------------------------------

type mockModelSvc struct {
	createFromBuildFn func(req *invDto.CreateModelFromBuildRequest) (*invDto.ModelResponse, error)
	updateFromBuildFn func(modelID string, req *invDto.CreateModelFromBuildRequest) (*invDto.ModelResponse, error)
	getByIDFn         func(id string) (*invDto.ModelDetailResponse, error)
}

func (m *mockModelSvc) Create(req *invDto.CreateModelRequest, by string) (*invDto.ModelResponse, error) {
	return nil, nil
}
func (m *mockModelSvc) Update(id string, req *invDto.UpdateModelRequest) (*invDto.ModelResponse, error) {
	return nil, nil
}
func (m *mockModelSvc) Delete(id string) error { return nil }
func (m *mockModelSvc) GetByID(id string) (*invDto.ModelDetailResponse, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, nil
}
func (m *mockModelSvc) List(params *invDto.ListParams) (*invDto.ModelListResponse, error) {
	return nil, nil
}
func (m *mockModelSvc) Activate(id string) (*invDto.ModelResponse, error)   { return nil, nil }
func (m *mockModelSvc) Deactivate(id string) (*invDto.ModelResponse, error) { return nil, nil }
func (m *mockModelSvc) CreateFromBuild(req *invDto.CreateModelFromBuildRequest) (*invDto.ModelResponse, error) {
	if m.createFromBuildFn != nil {
		return m.createFromBuildFn(req)
	}
	return &invDto.ModelResponse{ID: "m1"}, nil
}
func (m *mockModelSvc) UpdateFromBuild(modelID string, req *invDto.CreateModelFromBuildRequest) (*invDto.ModelResponse, error) {
	if m.updateFromBuildFn != nil {
		return m.updateFromBuildFn(modelID, req)
	}
	return &invDto.ModelResponse{ID: modelID}, nil
}
func (m *mockModelSvc) Score(id string, req *invDto.ScoreRequest, by string) (*invDto.ScoreResponse, error) {
	return nil, nil
}
func (m *mockModelSvc) HandleScoreCallback(req *invDto.ScoreCallbackRequest) error { return nil }
func (m *mockModelSvc) ConfigureScoring(cc compute.Client, dg invApp.DatasourceGetter, dc invApp.DatasourceCreator, cfg *config.Config) {
}
func (m *mockModelSvc) GetFileContent(modelID, fileID string) (*invDto.FileContentResponse, error) {
	return nil, nil
}
func (m *mockModelSvc) DeleteByFolderID(id string) error  { return nil }
func (m *mockModelSvc) DeleteByProjectID(id string) error { return nil }

var _ invApp.ModelService = (*mockModelSvc)(nil)

// ---------------------------------------------------------------------------
// Mock: invApp.ModelVersionService
// ---------------------------------------------------------------------------

type mockVersionSvc struct {
	createVersionFn func(modelID, by string) (*invDto.VersionResponse, error)
}

func (m *mockVersionSvc) CreateVersion(modelID, by string) (*invDto.VersionResponse, error) {
	if m.createVersionFn != nil {
		return m.createVersionFn(modelID, by)
	}
	return &invDto.VersionResponse{}, nil
}
func (m *mockVersionSvc) ListVersions(modelID string, params *invDto.ListVersionsParams) (*invDto.VersionListResponse, error) {
	return nil, nil
}
func (m *mockVersionSvc) GetVersion(modelID, versionID string) (*invDto.VersionDetailResponse, error) {
	return nil, nil
}
func (m *mockVersionSvc) RestoreVersion(modelID, versionID, by string) (*invDto.ModelResponse, error) {
	return nil, nil
}

var _ invApp.ModelVersionService = (*mockVersionSvc)(nil)

// ---------------------------------------------------------------------------
// Mock: invApp.PerformanceService (minimal — only methods used by build service)
// ---------------------------------------------------------------------------

type mockPerfSvc struct {
	createBaselineFn func(modelID string, req *invDto.CreateBaselineRequest, by string) (*invDto.BaselinesListResponse, error)
}

func (m *mockPerfSvc) CreateBaseline(modelID string, req *invDto.CreateBaselineRequest, by string) (*invDto.BaselinesListResponse, error) {
	if m.createBaselineFn != nil {
		return m.createBaselineFn(modelID, req, by)
	}
	return nil, nil
}
func (m *mockPerfSvc) GetBaselines(modelID string) (*invDto.BaselinesListResponse, error) {
	return nil, nil
}
func (m *mockPerfSvc) RecordPerformance(modelID string, req *invDto.RecordPerformanceRequest, by string) (*invDto.PerformanceHistoryResponse, error) {
	return nil, nil
}
func (m *mockPerfSvc) GetPerformanceHistory(modelID string, params *invDto.GetPerformanceHistoryParams) (*invDto.PerformanceHistoryResponse, error) {
	return nil, nil
}
func (m *mockPerfSvc) GetMetricTimeSeries(modelID, metric string, limit int) (*invDto.MetricTimeSeriesResponse, error) {
	return nil, nil
}
func (m *mockPerfSvc) StartEvaluation(modelID string, req *invDto.EvaluatePerformanceRequest, by string) (*invDto.PerformanceEvaluationResponse, error) {
	return nil, nil
}
func (m *mockPerfSvc) HandleEvaluationCallback(req *invDto.EvaluationCallbackRequest) error {
	return nil
}
func (m *mockPerfSvc) GetEvaluations(modelID string, limit int) (*invDto.EvaluationsListResponse, error) {
	return nil, nil
}
func (m *mockPerfSvc) GetEvaluation(id string) (*invDto.PerformanceEvaluationResponse, error) {
	return nil, nil
}
func (m *mockPerfSvc) GetAlerts(modelID string, params *invDto.GetAlertsParams) (*invDto.AlertsListResponse, error) {
	return nil, nil
}
func (m *mockPerfSvc) UpdateAlert(alertID string, req *invDto.UpdateAlertRequest, by string) (*invDto.PerformanceAlertResponse, error) {
	return nil, nil
}
func (m *mockPerfSvc) GetThresholds(modelID string) (*invDto.ThresholdsListResponse, error) {
	return nil, nil
}
func (m *mockPerfSvc) UpdateThreshold(modelID string, req *invDto.UpdateThresholdRequest) (*invDto.PerformanceThresholdResponse, error) {
	return nil, nil
}
func (m *mockPerfSvc) InitializeDefaultThresholds(modelID string, taskType invDomain.TaskType) error {
	return nil
}
func (m *mockPerfSvc) GetThresholdDefaults(taskType string) (*invDto.ThresholdDefaultsListResponse, error) {
	return nil, nil
}
func (m *mockPerfSvc) UpsertThresholdDefault(req *invDto.UpdateThresholdDefaultRequest, by string) (*invDto.PerformanceThresholdDefaultResponse, error) {
	return nil, nil
}
func (m *mockPerfSvc) ConfigureCompute(cc compute.Client, dg invApp.DatasourceGetter, cfg *config.Config) {
}
func (m *mockPerfSvc) DeleteByModelID(id string) error { return nil }
func (m *mockPerfSvc) GetPerformanceSummary(modelID string) (*invDto.PerformanceSummaryResponse, error) {
	return nil, nil
}

var _ invApp.PerformanceService = (*mockPerfSvc)(nil)

// ---------------------------------------------------------------------------
// Builder helper
// ---------------------------------------------------------------------------

// buildFullSvc constructs a BuildServiceImpl with all dependencies injectable.
// Uses existing mockComputeClient and mockDSService from build_service_start_test.go.
func buildFullSvc(
	repo *fakeBuildRepo,
	cc compute.Client,
	ds *mockDSService,
	ms *mockModelSvc,
	vs *mockVersionSvc,
	ps *mockPerfSvc,
) *BuildServiceImpl {
	if repo == nil {
		repo = newFakeBuildRepo()
	}
	cfg := &config.Config{}
	cfg.Server.BaseURL = "http://localhost:8080"
	return &BuildServiceImpl{
		buildRepo:          repo,
		domainService:      domain.NewService(),
		computeClient:      cc,
		datasourceService:  ds,
		modelService:       ms,
		versionService:     vs,
		performanceService: ps,
		config:             cfg,
	}
}

// dsDetail builds a DatasourceDetailResponse with arbitrary columns.
func dsDetail(cols ...dsDto.ColumnResponse) *dsDto.DatasourceDetailResponse {
	return &dsDto.DatasourceDetailResponse{
		DatasourceResponse: dsDto.DatasourceResponse{FilePath: "datasources/ds1/data.csv"},
		Columns:            cols,
	}
}

func colResp(name, role string) dsDto.ColumnResponse {
	return dsDto.ColumnResponse{Name: name, Role: role}
}

// runningBuild creates a build in Running status.
func runningBuild(id string) *domain.ModelBuild {
	b := &domain.ModelBuild{
		ID:           id,
		Name:         "Test Build",
		DatasourceID: "ds1",
		ModelType:    domain.ModelTypeClassification,
		Algorithm:    "random_forest",
		Status:       domain.BuildStatusPending,
		CreatedBy:    "alice",
	}
	b.Start()
	return b
}

// ---------------------------------------------------------------------------
// convertMetrics — pure function, no dependencies
// ---------------------------------------------------------------------------

func TestConvertMetrics_Nil(t *testing.T) {
	assert.Nil(t, convertMetrics(nil))
}

func TestConvertMetrics_ClassificationMetrics(t *testing.T) {
	m := map[string]interface{}{
		"accuracy": 0.92, "precision": 0.89, "recall": 0.91, "f1_score": 0.90,
	}
	result := convertMetrics(m)
	require.NotNil(t, result)
	assert.Equal(t, 0.92, result.Accuracy)
	assert.Equal(t, 0.89, result.Precision)
	assert.Equal(t, 0.91, result.Recall)
	assert.Equal(t, 0.90, result.F1Score)
}

func TestConvertMetrics_RegressionMetrics(t *testing.T) {
	m := map[string]interface{}{
		"mse": 0.05, "rmse": 0.22, "mae": 0.18, "r2": 0.87,
	}
	result := convertMetrics(m)
	require.NotNil(t, result)
	assert.Equal(t, 0.05, result.MSE)
	assert.Equal(t, 0.22, result.RMSE)
	assert.Equal(t, 0.18, result.MAE)
	assert.Equal(t, 0.87, result.R2)
}

func TestConvertMetrics_WrongTypeIsZero(t *testing.T) {
	m := map[string]interface{}{"accuracy": "not-a-float", "recall": 0.85}
	result := convertMetrics(m)
	require.NotNil(t, result)
	assert.Equal(t, 0.0, result.Accuracy, "non-float64 must not panic and defaults to zero")
	assert.Equal(t, 0.85, result.Recall)
}

// ---------------------------------------------------------------------------
// HandleCallback — build state guards
// ---------------------------------------------------------------------------

func TestHandleCallback_BuildNotFound(t *testing.T) {
	svc := buildFullSvc(nil, nil, nil, nil, nil, nil)
	err := svc.HandleCallback(&dto.BuildCallbackRequest{BuildID: "missing", Status: "completed"})
	require.Error(t, err)
}

func TestHandleCallback_BuildNotRunning_IgnoresCallback(t *testing.T) {
	b := &domain.ModelBuild{ID: "b1", Name: "B", Status: domain.BuildStatusPending, DatasourceID: "ds1"}
	repo := newFakeBuildRepo(b)
	svc := buildFullSvc(repo, nil, nil, nil, nil, nil)

	err := svc.HandleCallback(&dto.BuildCallbackRequest{BuildID: "b1", Status: "completed"})
	require.NoError(t, err)
	// Build still Pending — callback was ignored
	assert.Equal(t, domain.BuildStatusPending, repo.builds["b1"].Status)
}

// ---------------------------------------------------------------------------
// HandleCallback — failed build
// ---------------------------------------------------------------------------

func TestHandleCallback_Failed_MarksFailedAndNoModelCreated(t *testing.T) {
	b := runningBuild("b1")
	repo := newFakeBuildRepo(b)
	created := false
	ms := &mockModelSvc{
		createFromBuildFn: func(req *invDto.CreateModelFromBuildRequest) (*invDto.ModelResponse, error) {
			created = true
			return &invDto.ModelResponse{ID: "m1"}, nil
		},
	}
	svc := buildFullSvc(repo, nil, nil, ms, nil, nil)
	errMsg := "GPU OOM"
	err := svc.HandleCallback(&dto.BuildCallbackRequest{BuildID: "b1", Status: "failed", Error: &errMsg})
	require.NoError(t, err)
	assert.Equal(t, domain.BuildStatusFailed, repo.builds["b1"].Status)
	assert.False(t, created, "model should NOT be created on failure")
}

// ---------------------------------------------------------------------------
// HandleCallback — completed, new build (no SourceModelID)
// ---------------------------------------------------------------------------

func TestHandleCallback_Completed_NewBuild_CreatesModel(t *testing.T) {
	b := runningBuild("b1") // no SourceModelID
	repo := newFakeBuildRepo(b)

	var capturedReq *invDto.CreateModelFromBuildRequest
	ms := &mockModelSvc{
		createFromBuildFn: func(req *invDto.CreateModelFromBuildRequest) (*invDto.ModelResponse, error) {
			capturedReq = req
			return &invDto.ModelResponse{ID: "m1"}, nil
		},
	}
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			return dsDetail(colResp("age", "input"), colResp("income", "input"), colResp("churn", "target")), nil
		},
	}
	modelPath := "models/b1/model.pkl"
	svc := buildFullSvc(repo, nil, ds, ms, nil, nil)
	err := svc.HandleCallback(&dto.BuildCallbackRequest{
		BuildID: "b1", Status: "completed", ModelPath: &modelPath,
	})
	require.NoError(t, err)
	assert.Equal(t, domain.BuildStatusCompleted, repo.builds["b1"].Status)
	require.NotNil(t, capturedReq)
	assert.Equal(t, "churn", capturedReq.TargetColumn)
	assert.ElementsMatch(t, []string{"age", "income"}, capturedReq.InputColumns)
}

func TestHandleCallback_Completed_ModelPathStoredInHyperparameters(t *testing.T) {
	b := runningBuild("b1")
	repo := newFakeBuildRepo(b)
	modelPath := "models/b1/rf.pkl"
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			return dsDetail(colResp("x", "input"), colResp("y", "target")), nil
		},
	}
	ms := &mockModelSvc{
		createFromBuildFn: func(req *invDto.CreateModelFromBuildRequest) (*invDto.ModelResponse, error) {
			return &invDto.ModelResponse{ID: "m1"}, nil
		},
	}
	svc := buildFullSvc(repo, nil, ds, ms, nil, nil)
	err := svc.HandleCallback(&dto.BuildCallbackRequest{
		BuildID: "b1", Status: "completed", ModelPath: &modelPath,
	})
	require.NoError(t, err)
	assert.Equal(t, modelPath, repo.builds["b1"].Parameters.Hyperparameters["_model_path"])
}

func TestHandleCallback_Completed_AutoBaselineCreated_WhenMetricsPresent(t *testing.T) {
	b := runningBuild("b1")
	repo := newFakeBuildRepo(b)
	baselineCreated := false
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			return dsDetail(colResp("f1", "input"), colResp("label", "target")), nil
		},
	}
	ms := &mockModelSvc{
		createFromBuildFn: func(req *invDto.CreateModelFromBuildRequest) (*invDto.ModelResponse, error) {
			return &invDto.ModelResponse{ID: "m1"}, nil
		},
	}
	ps := &mockPerfSvc{
		createBaselineFn: func(modelID string, req *invDto.CreateBaselineRequest, by string) (*invDto.BaselinesListResponse, error) {
			baselineCreated = true
			return nil, nil
		},
	}
	svc := buildFullSvc(repo, nil, ds, ms, nil, ps)
	err := svc.HandleCallback(&dto.BuildCallbackRequest{
		BuildID: "b1", Status: "completed",
		Metrics: map[string]interface{}{"accuracy": 0.92, "f1_score": 0.88},
	})
	require.NoError(t, err)
	assert.True(t, baselineCreated, "auto-baseline should be created when metrics are present")
}

func TestHandleCallback_Completed_NoAutoBaseline_WhenNoMetrics(t *testing.T) {
	b := runningBuild("b1")
	repo := newFakeBuildRepo(b)
	baselineCreated := false
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			return dsDetail(colResp("f1", "input"), colResp("label", "target")), nil
		},
	}
	ms := &mockModelSvc{
		createFromBuildFn: func(req *invDto.CreateModelFromBuildRequest) (*invDto.ModelResponse, error) {
			return &invDto.ModelResponse{ID: "m1"}, nil
		},
	}
	ps := &mockPerfSvc{
		createBaselineFn: func(modelID string, req *invDto.CreateBaselineRequest, by string) (*invDto.BaselinesListResponse, error) {
			baselineCreated = true
			return nil, nil
		},
	}
	svc := buildFullSvc(repo, nil, ds, ms, nil, ps)
	err := svc.HandleCallback(&dto.BuildCallbackRequest{BuildID: "b1", Status: "completed"})
	require.NoError(t, err)
	assert.False(t, baselineCreated, "no baseline when no metrics")
}

// ---------------------------------------------------------------------------
// HandleCallback — retrain path (SourceModelID set)
// ---------------------------------------------------------------------------

func TestHandleCallback_Completed_Retrain_SnapshotsVersionThenUpdatesModel(t *testing.T) {
	srcID := "source-m1"
	b := runningBuild("b1")
	b.SourceModelID = &srcID
	repo := newFakeBuildRepo(b)

	versionCreated := false
	modelUpdated := false
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			return dsDetail(colResp("feat", "input"), colResp("target", "target")), nil
		},
	}
	vs := &mockVersionSvc{
		createVersionFn: func(modelID, by string) (*invDto.VersionResponse, error) {
			versionCreated = true
			assert.Equal(t, srcID, modelID)
			return &invDto.VersionResponse{}, nil
		},
	}
	ms := &mockModelSvc{
		updateFromBuildFn: func(modelID string, req *invDto.CreateModelFromBuildRequest) (*invDto.ModelResponse, error) {
			modelUpdated = true
			assert.Equal(t, srcID, modelID)
			return &invDto.ModelResponse{ID: modelID}, nil
		},
	}
	svc := buildFullSvc(repo, nil, ds, ms, vs, nil)
	err := svc.HandleCallback(&dto.BuildCallbackRequest{BuildID: "b1", Status: "completed"})
	require.NoError(t, err)
	assert.True(t, versionCreated, "version snapshot should be taken before update")
	assert.True(t, modelUpdated, "model should be updated from retrain build")
}

// ---------------------------------------------------------------------------
// Retrain — business logic decisions
// ---------------------------------------------------------------------------

func TestRetrain_ModelNotFound(t *testing.T) {
	ms := &mockModelSvc{
		getByIDFn: func(id string) (*invDto.ModelDetailResponse, error) {
			return nil, errors.New("model not found")
		},
	}
	svc := buildFullSvc(nil, nil, nil, ms, nil, nil)
	_, err := svc.Retrain("missing", nil, "alice")
	require.Error(t, err)
}

func TestRetrain_NoInputVariables_Rejected(t *testing.T) {
	ms := &mockModelSvc{
		getByIDFn: func(id string) (*invDto.ModelDetailResponse, error) {
			return &invDto.ModelDetailResponse{
				ModelResponse: invDto.ModelResponse{ID: id, Name: "M", ModelType: "classification"},
				Variables:     []invDto.VariableResponse{{Name: "label", Role: "target"}},
			}, nil
		},
	}
	svc := buildFullSvc(nil, nil, nil, ms, nil, nil)
	_, err := svc.Retrain("m1", nil, "alice")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no input variables")
}

func TestRetrain_DefaultNameContainsModelName(t *testing.T) {
	dsID := "ds1"
	ms := &mockModelSvc{
		getByIDFn: func(id string) (*invDto.ModelDetailResponse, error) {
			return &invDto.ModelDetailResponse{
				ModelResponse: invDto.ModelResponse{
					ID: id, Name: "FraudDetector", ModelType: "classification",
					Algorithm: "rf", DatasourceID: dsID,
				},
				Variables: []invDto.VariableResponse{{Name: "amount", Role: "input"}},
			}, nil
		},
	}
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			return dsDetail(colResp("amount", "input"), colResp("is_fraud", "target")), nil
		},
	}
	repo := newFakeBuildRepo()
	svc := buildFullSvc(repo, &mockComputeClient{}, ds, ms, nil, nil)
	_, err := svc.Retrain("m1", nil, "alice")
	require.NoError(t, err)

	for _, b := range repo.builds {
		assert.Contains(t, b.Name, "FraudDetector")
	}
}

func TestRetrain_OverridesParametersFromRequest(t *testing.T) {
	dsID := "ds1"
	ms := &mockModelSvc{
		getByIDFn: func(id string) (*invDto.ModelDetailResponse, error) {
			return &invDto.ModelDetailResponse{
				ModelResponse: invDto.ModelResponse{
					ID: id, Name: "M", ModelType: "classification",
					Algorithm: "xgboost", DatasourceID: dsID,
				},
				Variables: []invDto.VariableResponse{{Name: "f1", Role: "input"}},
			}, nil
		},
	}
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			return dsDetail(colResp("f1", "input"), colResp("y", "target")), nil
		},
	}
	repo := newFakeBuildRepo()
	svc := buildFullSvc(repo, &mockComputeClient{}, ds, ms, nil, nil)
	split := 0.85
	_, err := svc.Retrain("m1", &dto.RetrainRequest{
		Parameters: &dto.TrainingParametersRequest{TrainTestSplit: split},
	}, "alice")
	require.NoError(t, err)
	for _, b := range repo.builds {
		assert.Equal(t, split, b.Parameters.TrainTestSplit)
	}
}

func TestRetrain_CustomName(t *testing.T) {
	dsID := "ds1"
	ms := &mockModelSvc{
		getByIDFn: func(id string) (*invDto.ModelDetailResponse, error) {
			return &invDto.ModelDetailResponse{
				ModelResponse: invDto.ModelResponse{
					ID: id, Name: "M", ModelType: "classification",
					Algorithm: "rf", DatasourceID: dsID,
				},
				Variables: []invDto.VariableResponse{{Name: "f1", Role: "input"}},
			}, nil
		},
	}
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			return dsDetail(colResp("f1", "input"), colResp("y", "target")), nil
		},
	}
	repo := newFakeBuildRepo()
	customName := "Q4 Retrain"
	svc := buildFullSvc(repo, &mockComputeClient{}, ds, ms, nil, nil)
	_, err := svc.Retrain("m1", &dto.RetrainRequest{Name: &customName}, "alice")
	require.NoError(t, err)
	for _, b := range repo.builds {
		assert.Equal(t, customName, b.Name)
	}
}

func TestRetrain_CustomDatasourceIDOverridesModel(t *testing.T) {
	origDS := "original-ds"
	newDS := "new-ds"
	ms := &mockModelSvc{
		getByIDFn: func(id string) (*invDto.ModelDetailResponse, error) {
			return &invDto.ModelDetailResponse{
				ModelResponse: invDto.ModelResponse{
					ID: id, Name: "M", ModelType: "classification",
					Algorithm: "rf", DatasourceID: origDS,
				},
				Variables: []invDto.VariableResponse{{Name: "f1", Role: "input"}},
			}, nil
		},
	}
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			return dsDetail(colResp("f1", "input"), colResp("y", "target")), nil
		},
	}
	repo := newFakeBuildRepo()
	svc := buildFullSvc(repo, &mockComputeClient{}, ds, ms, nil, nil)
	_, err := svc.Retrain("m1", &dto.RetrainRequest{DatasourceID: &newDS}, "alice")
	require.NoError(t, err)
	for _, b := range repo.builds {
		assert.Equal(t, newDS, b.DatasourceID, "custom datasourceID should override model's datasourceID")
	}
}

// ---------------------------------------------------------------------------
// createModelFromBuild — feature name fallback logic
// ---------------------------------------------------------------------------

func TestCreateModelFromBuild_FeatureNamesFallbackToDatasource(t *testing.T) {
	b := runningBuild("b1")
	repo := newFakeBuildRepo(b)
	var capturedReq *invDto.CreateModelFromBuildRequest
	ms := &mockModelSvc{
		createFromBuildFn: func(req *invDto.CreateModelFromBuildRequest) (*invDto.ModelResponse, error) {
			capturedReq = req
			return &invDto.ModelResponse{ID: "m1"}, nil
		},
	}
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			return dsDetail(colResp("age", "input"), colResp("zip", "input"), colResp("churn", "target")), nil
		},
	}
	svc := buildFullSvc(repo, nil, ds, ms, nil, nil)
	// Callback has no FeatureNames → should fall back to datasource input columns
	err := svc.HandleCallback(&dto.BuildCallbackRequest{BuildID: "b1", Status: "completed"})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"age", "zip"}, capturedReq.InputColumns)
}

func TestCreateModelFromBuild_FeatureNamesFromCallbackTakePriority(t *testing.T) {
	b := runningBuild("b1")
	repo := newFakeBuildRepo(b)
	var capturedReq *invDto.CreateModelFromBuildRequest
	ms := &mockModelSvc{
		createFromBuildFn: func(req *invDto.CreateModelFromBuildRequest) (*invDto.ModelResponse, error) {
			capturedReq = req
			return &invDto.ModelResponse{ID: "m1"}, nil
		},
	}
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			// Datasource has 3 input cols, but model only used 2
			return dsDetail(
				colResp("age", "input"), colResp("zip", "input"), colResp("region", "input"),
				colResp("churn", "target"),
			), nil
		},
	}
	svc := buildFullSvc(repo, nil, ds, ms, nil, nil)
	err := svc.HandleCallback(&dto.BuildCallbackRequest{
		BuildID:      "b1",
		Status:       "completed",
		FeatureNames: []string{"age", "zip"}, // subset from callback
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"age", "zip"}, capturedReq.InputColumns)
	assert.NotContains(t, capturedReq.InputColumns, "region")
}
