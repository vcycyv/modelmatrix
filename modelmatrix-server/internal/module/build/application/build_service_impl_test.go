package application

import (
	"errors"
	"testing"

	"modelmatrix-server/internal/module/build/domain"
	"modelmatrix-server/internal/module/build/dto"
	"modelmatrix-server/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// fakeBuildRepo — full implementation of repository.BuildRepository
// ---------------------------------------------------------------------------

type fakeBuildRepo struct {
	builds     map[string]*domain.ModelBuild
	allNames   []string
	createErr  error
	folderIDs  map[string][]string  // folderID -> buildIDs
	projectIDs map[string][]string // projectID -> buildIDs
}

func newFakeBuildRepo(builds ...*domain.ModelBuild) *fakeBuildRepo {
	m := &fakeBuildRepo{builds: make(map[string]*domain.ModelBuild)}
	for _, b := range builds {
		m.builds[b.ID] = b
		m.allNames = append(m.allNames, b.Name)
	}
	return m
}

func (m *fakeBuildRepo) Create(b *domain.ModelBuild) error {
	if m.createErr != nil {
		return m.createErr
	}
	if b.ID == "" {
		b.ID = "fake-id"
	}
	m.builds[b.ID] = b
	m.allNames = append(m.allNames, b.Name)
	return nil
}

func (m *fakeBuildRepo) Update(b *domain.ModelBuild) error          { m.builds[b.ID] = b; return nil }
func (m *fakeBuildRepo) Delete(id string) error                      { delete(m.builds, id); return nil }
func (m *fakeBuildRepo) GetByName(name string) (*domain.ModelBuild, error) { return nil, nil }
func (m *fakeBuildRepo) GetAllNames() ([]string, error)              { return m.allNames, nil }
func (m *fakeBuildRepo) UpdateStatus(id string, status domain.BuildStatus, errMsg string) error {
	return nil
}
func (m *fakeBuildRepo) GetIDsByFolderID(id string) ([]string, error) {
	if m.folderIDs != nil {
		return m.folderIDs[id], nil
	}
	return nil, nil
}
func (m *fakeBuildRepo) GetIDsByProjectID(id string) ([]string, error) {
	if m.projectIDs != nil {
		return m.projectIDs[id], nil
	}
	return nil, nil
}

func (m *fakeBuildRepo) GetByID(id string) (*domain.ModelBuild, error) {
	if b, ok := m.builds[id]; ok {
		return b, nil
	}
	return nil, domain.ErrBuildNotFound
}

func (m *fakeBuildRepo) List(offset, limit int, search, status string) ([]domain.ModelBuild, int64, error) {
	var result []domain.ModelBuild
	for _, b := range m.builds {
		result = append(result, *b)
	}
	return result, int64(len(result)), nil
}

// ---------------------------------------------------------------------------
// Helper: build a minimal BuildServiceImpl for testing
// ---------------------------------------------------------------------------

func buildSvcImpl(repo *fakeBuildRepo) BuildService {
	return NewBuildService(
		repo,
		domain.NewService(),
		nil, // computeClient — not used for Create/Cancel/Delete
		nil, // datasourceService
		nil, // modelService
		nil, // versionService
		nil, // folderService
		nil, // performanceService
		&config.Config{},
	)
}

func validCreateReq(name string) *dto.CreateBuildRequest {
	dsID := "550e8400-e29b-41d4-a716-446655440001"
	return &dto.CreateBuildRequest{
		Name:         name,
		DatasourceID: dsID,
		ModelType:    "regression",
		Algorithm:    "random_forest",
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestBuildServiceImpl_Create_Valid(t *testing.T) {
	repo := newFakeBuildRepo()
	svc := buildSvcImpl(repo)

	resp, err := svc.Create(validCreateReq("Sales Model"), "alice")
	require.NoError(t, err)
	assert.Equal(t, "Sales Model", resp.Name)
	assert.Equal(t, "pending", resp.Status)
	assert.Equal(t, "alice", resp.CreatedBy)
}

func TestBuildServiceImpl_Create_DuplicateName(t *testing.T) {
	existing := &domain.ModelBuild{ID: "b1", Name: "Duplicate", ModelType: domain.ModelTypeRegression, Status: domain.BuildStatusPending}
	repo := newFakeBuildRepo(existing)
	svc := buildSvcImpl(repo)

	_, err := svc.Create(validCreateReq("Duplicate"), "bob")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBuildNameExists))
}

func TestBuildServiceImpl_Create_EmptyName_RejectedByBinding(t *testing.T) {
	repo := newFakeBuildRepo()
	svc := buildSvcImpl(repo)

	// Domain validation rejects empty name
	req := validCreateReq("")
	_, err := svc.Create(req, "alice")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBuildNameEmpty))
}

func TestBuildServiceImpl_Create_InvalidModelType(t *testing.T) {
	repo := newFakeBuildRepo()
	svc := buildSvcImpl(repo)

	req := &dto.CreateBuildRequest{
		Name:         "Bad Type",
		DatasourceID: "550e8400-e29b-41d4-a716-446655440001",
		ModelType:    "magic",
		Algorithm:    "random_forest",
	}
	_, err := svc.Create(req, "alice")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidModelType))
}

// ---------------------------------------------------------------------------
// Cancel
// ---------------------------------------------------------------------------

func TestBuildServiceImpl_Cancel_PendingBuild(t *testing.T) {
	b := &domain.ModelBuild{ID: "b1", Name: "Pending Build", Status: domain.BuildStatusPending}
	repo := newFakeBuildRepo(b)
	svc := buildSvcImpl(repo)

	resp, err := svc.Cancel("b1")
	require.NoError(t, err)
	assert.Equal(t, "cancelled", resp.Status)
}

func TestBuildServiceImpl_Cancel_RunningBuild(t *testing.T) {
	b := &domain.ModelBuild{ID: "b1", Name: "Running Build", Status: domain.BuildStatusRunning}
	repo := newFakeBuildRepo(b)
	svc := buildSvcImpl(repo)

	resp, err := svc.Cancel("b1")
	require.NoError(t, err)
	assert.Equal(t, "cancelled", resp.Status)
}

func TestBuildServiceImpl_Cancel_CompletedBuild(t *testing.T) {
	b := &domain.ModelBuild{ID: "b1", Name: "Done Build", Status: domain.BuildStatusCompleted}
	repo := newFakeBuildRepo(b)
	svc := buildSvcImpl(repo)

	_, err := svc.Cancel("b1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBuildCannotBeCancelled))
}

func TestBuildServiceImpl_Cancel_NotFound(t *testing.T) {
	svc := buildSvcImpl(newFakeBuildRepo())
	_, err := svc.Cancel("nonexistent")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBuildNotFound))
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestBuildServiceImpl_Delete_PendingBuild(t *testing.T) {
	b := &domain.ModelBuild{ID: "b1", Name: "Old Build", Status: domain.BuildStatusPending}
	repo := newFakeBuildRepo(b)
	svc := buildSvcImpl(repo)

	err := svc.Delete("b1")
	require.NoError(t, err)
	_, err2 := svc.GetByID("b1")
	assert.True(t, errors.Is(err2, domain.ErrBuildNotFound))
}

func TestBuildServiceImpl_Delete_RunningBuild_Rejected(t *testing.T) {
	b := &domain.ModelBuild{ID: "b1", Name: "Active Build", Status: domain.BuildStatusRunning}
	repo := newFakeBuildRepo(b)
	svc := buildSvcImpl(repo)

	err := svc.Delete("b1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBuildAlreadyRunning))
}

func TestBuildServiceImpl_Delete_NotFound(t *testing.T) {
	svc := buildSvcImpl(newFakeBuildRepo())
	err := svc.Delete("missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBuildNotFound))
}

// ---------------------------------------------------------------------------
// GetByID / List
// ---------------------------------------------------------------------------

func TestBuildServiceImpl_GetByID_Found(t *testing.T) {
	b := &domain.ModelBuild{ID: "b1", Name: "Model 1", Status: domain.BuildStatusPending}
	svc := buildSvcImpl(newFakeBuildRepo(b))

	resp, err := svc.GetByID("b1")
	require.NoError(t, err)
	assert.Equal(t, "b1", resp.ID)
}

func TestBuildServiceImpl_GetByID_NotFound(t *testing.T) {
	svc := buildSvcImpl(newFakeBuildRepo())
	_, err := svc.GetByID("missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBuildNotFound))
}

func TestBuildServiceImpl_List_ReturnsPaginated(t *testing.T) {
	builds := []*domain.ModelBuild{
		{ID: "b1", Name: "A", Status: domain.BuildStatusPending},
		{ID: "b2", Name: "B", Status: domain.BuildStatusCompleted},
	}
	svc := buildSvcImpl(newFakeBuildRepo(builds...))

	resp, err := svc.List(&dto.ListParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
}

// ---------------------------------------------------------------------------
// Create — custom parameters and repository error paths
// ---------------------------------------------------------------------------

func TestBuildServiceImpl_Create_WithCustomParameters(t *testing.T) {
	repo := &fakeBuildRepo{builds: make(map[string]*domain.ModelBuild)}
	svc := buildSvcImpl(repo)
	params := &dto.TrainingParametersRequest{
		TrainTestSplit:  0.7,
		RandomSeed:      99,
		MaxIterations:   200,
		EarlyStopRounds: 20,
		Hyperparameters: map[string]interface{}{"n_estimators": 200},
	}
	req := &dto.CreateBuildRequest{
		Name:         "Custom Params Build",
		DatasourceID: "ds1",
		ModelType:    "regression",
		Algorithm:    "xgboost",
		Parameters:   params,
	}
	resp, err := svc.Create(req, "alice")
	require.NoError(t, err)
	assert.Equal(t, "Custom Params Build", resp.Name)
	assert.Equal(t, 0.7, resp.Parameters.TrainTestSplit)
}

func TestBuildServiceImpl_Create_RepositoryError(t *testing.T) {
	repo := &fakeBuildRepo{
		builds:    make(map[string]*domain.ModelBuild),
		createErr: errors.New("db write error"),
	}
	svc := buildSvcImpl(repo)
	req := &dto.CreateBuildRequest{
		Name: "X", DatasourceID: "ds1", ModelType: "classification", Algorithm: "random_forest",
	}
	_, err := svc.Create(req, "alice")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db write error")
}

// ---------------------------------------------------------------------------
// Delete — running build rejected
// ---------------------------------------------------------------------------

func TestBuildServiceImpl_Delete_RunningBuildRejected(t *testing.T) {
	b := &domain.ModelBuild{ID: "b1", Name: "Running", Status: domain.BuildStatusRunning}
	repo := &fakeBuildRepo{builds: map[string]*domain.ModelBuild{"b1": b}}
	svc := buildSvcImpl(repo)

	err := svc.Delete("b1")
	require.Error(t, err)
	assert.Equal(t, domain.ErrBuildAlreadyRunning, err)
}

// ---------------------------------------------------------------------------
// DeleteByProjectID — empty and with builds
// ---------------------------------------------------------------------------

func TestBuildServiceImpl_DeleteByProjectID_Empty(t *testing.T) {
	repo := &fakeBuildRepo{builds: make(map[string]*domain.ModelBuild), projectIDs: make(map[string][]string)}
	svc := buildSvcImpl(repo)
	require.NoError(t, svc.DeleteByProjectID("p1"))
}

func TestBuildServiceImpl_DeleteByProjectID_WithBuilds(t *testing.T) {
	b1 := &domain.ModelBuild{ID: "b1", Name: "B1", Status: domain.BuildStatusPending}
	repo := &fakeBuildRepo{
		builds:     map[string]*domain.ModelBuild{"b1": b1},
		projectIDs: map[string][]string{"p1": {"b1"}},
	}
	svc := buildSvcImpl(repo)
	require.NoError(t, svc.DeleteByProjectID("p1"))
	assert.Empty(t, repo.builds)
}
