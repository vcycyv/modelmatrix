package application

import (
	"errors"
	"testing"

	"modelmatrix-server/internal/infrastructure/compute"
	"modelmatrix-server/internal/module/build/domain"
	dsDomain "modelmatrix-server/internal/module/datasource/domain"
	dsDto "modelmatrix-server/internal/module/datasource/dto"
	"modelmatrix-server/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Minimal mock compute.Client
// ---------------------------------------------------------------------------

type mockComputeClient struct {
	trainFn func(req *compute.TrainRequest) (*compute.TrainResponse, error)
}

func (m *mockComputeClient) TrainModel(req *compute.TrainRequest) (*compute.TrainResponse, error) {
	if m.trainFn != nil {
		return m.trainFn(req)
	}
	return &compute.TrainResponse{JobID: "job-123"}, nil
}
func (m *mockComputeClient) ScoreModel(req *compute.ScoreRequest) (*compute.ScoreResponse, error) {
	return nil, nil
}
func (m *mockComputeClient) EvaluatePerformance(req *compute.EvaluateRequest) (*compute.EvaluateResponse, error) {
	return nil, nil
}
func (m *mockComputeClient) GetStatus(jobID string) (*compute.JobStatusResponse, error) {
	return nil, nil
}
func (m *mockComputeClient) HealthCheck() error { return nil }

// ---------------------------------------------------------------------------
// Minimal mock DatasourceService
// ---------------------------------------------------------------------------

type mockDSService struct {
	getByIDFn func(id string) (*dsDto.DatasourceDetailResponse, error)
}

func (m *mockDSService) GetByID(id string) (*dsDto.DatasourceDetailResponse, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(id)
	}
	return nil, nil
}
func (m *mockDSService) Create(req *dsDto.CreateDatasourceRequest, filename string, fileData []byte, by string) (*dsDto.DatasourceResponse, error) {
	return nil, nil
}
func (m *mockDSService) Update(id string, req *dsDto.UpdateDatasourceRequest) (*dsDto.DatasourceResponse, error) {
	return nil, nil
}
func (m *mockDSService) Delete(id string) error { return nil }
func (m *mockDSService) List(collectionID *string, params *dsDto.ListParams) (*dsDto.DatasourceListResponse, error) {
	return nil, nil
}
func (m *mockDSService) CreateFromExistingFile(collectionID, name, filePath string, rowCount int, by string) (*dsDto.DatasourceResponse, error) {
	return nil, nil
}
func (m *mockDSService) GetDataPreview(id string, limit int) (*dsDto.DataPreviewResponse, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildSvcWithDeps(repo *fakeBuildRepo, cc *mockComputeClient, ds *mockDSService) BuildService {
	cfg := &config.Config{}
	cfg.Server.BaseURL = "http://localhost:8080"
	return NewBuildService(
		repo,
		domain.NewService(),
		cc,
		ds,
		nil, nil, nil, nil,
		cfg,
	)
}

func dsWithColumns(id string, target, input string) *dsDto.DatasourceDetailResponse {
	return &dsDto.DatasourceDetailResponse{
		DatasourceResponse: dsDto.DatasourceResponse{
			ID:       id,
			FilePath: "datasources/ds1/data.csv",
		},
		Columns: []dsDto.ColumnResponse{
			{Name: target, Role: string(dsDomain.ColumnRoleTarget)},
			{Name: input, Role: string(dsDomain.ColumnRoleInput)},
		},
	}
}

// ---------------------------------------------------------------------------
// Start tests
// ---------------------------------------------------------------------------

func TestBuildService_Start_Success(t *testing.T) {
	b := &domain.ModelBuild{
		ID:           "b1",
		Name:         "Test",
		Status:       domain.BuildStatusPending,
		DatasourceID: "ds1",
		ModelType:    domain.ModelTypeClassification,
		Algorithm:    "random_forest",
	}
	repo := newFakeBuildRepo(b)
	cc := &mockComputeClient{
		trainFn: func(req *compute.TrainRequest) (*compute.TrainResponse, error) {
			return &compute.TrainResponse{JobID: "job-abc"}, nil
		},
	}
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			return dsWithColumns(id, "churn", "age"), nil
		},
	}
	svc := buildSvcWithDeps(repo, cc, ds)

	resp, err := svc.Start("b1")
	require.NoError(t, err)
	assert.Equal(t, "running", resp.Status)
}

func TestBuildService_Start_BuildNotFound(t *testing.T) {
	svc := buildSvcWithDeps(newFakeBuildRepo(), &mockComputeClient{}, &mockDSService{})
	_, err := svc.Start("missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBuildNotFound))
}

func TestBuildService_Start_AlreadyRunning_DomainError(t *testing.T) {
	b := &domain.ModelBuild{ID: "b1", Name: "Running", Status: domain.BuildStatusRunning}
	svc := buildSvcWithDeps(newFakeBuildRepo(b), &mockComputeClient{}, &mockDSService{})
	_, err := svc.Start("b1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBuildNotPending))
}

func TestBuildService_Start_NoTargetColumn_Error(t *testing.T) {
	b := &domain.ModelBuild{
		ID: "b1", Name: "Build", Status: domain.BuildStatusPending, DatasourceID: "ds1",
	}
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			// All columns are input — no target
			return &dsDto.DatasourceDetailResponse{
				DatasourceResponse: dsDto.DatasourceResponse{ID: id, FilePath: "some/path.csv"},
				Columns: []dsDto.ColumnResponse{
					{Name: "age", Role: string(dsDomain.ColumnRoleInput)},
				},
			}, nil
		},
	}
	svc := buildSvcWithDeps(newFakeBuildRepo(b), &mockComputeClient{}, ds)
	_, err := svc.Start("b1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no target column")
}

func TestBuildService_Start_NoDatasourceFilePath_Error(t *testing.T) {
	b := &domain.ModelBuild{
		ID: "b1", Name: "Build", Status: domain.BuildStatusPending, DatasourceID: "ds1",
	}
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			return &dsDto.DatasourceDetailResponse{
				DatasourceResponse: dsDto.DatasourceResponse{ID: id, FilePath: ""}, // no file
			}, nil
		},
	}
	svc := buildSvcWithDeps(newFakeBuildRepo(b), &mockComputeClient{}, ds)
	_, err := svc.Start("b1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file path")
}

func TestBuildService_Start_ComputeClientError_MarksAsFailed(t *testing.T) {
	b := &domain.ModelBuild{
		ID:           "b1",
		Name:         "Build",
		Status:       domain.BuildStatusPending,
		DatasourceID: "ds1",
		ModelType:    domain.ModelTypeRegression,
		Algorithm:    "xgboost",
	}
	repo := newFakeBuildRepo(b)
	cc := &mockComputeClient{
		trainFn: func(req *compute.TrainRequest) (*compute.TrainResponse, error) {
			return nil, errors.New("compute service down")
		},
	}
	ds := &mockDSService{
		getByIDFn: func(id string) (*dsDto.DatasourceDetailResponse, error) {
			return dsWithColumns(id, "price", "sqft"), nil
		},
	}
	svc := buildSvcWithDeps(repo, cc, ds)
	_, err := svc.Start("b1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compute service down")

	// Build should be marked as failed in the repo
	updatedBuild := repo.builds["b1"]
	assert.Equal(t, domain.BuildStatusFailed, updatedBuild.Status)
}
