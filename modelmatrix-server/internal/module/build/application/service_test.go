package application

import (
	"errors"
	"testing"

	"modelmatrix-server/internal/module/build/domain"
	"modelmatrix-server/internal/module/build/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- minimal mocks ---

type mockBuildRepo struct {
	builds   map[string]*domain.ModelBuild
	allNames []string
}

func newMockBuildRepo(builds ...*domain.ModelBuild) *mockBuildRepo {
	m := &mockBuildRepo{builds: make(map[string]*domain.ModelBuild)}
	for _, b := range builds {
		m.builds[b.ID] = b
		m.allNames = append(m.allNames, b.Name)
	}
	return m
}

func (m *mockBuildRepo) Create(b *domain.ModelBuild) error {
	if b.ID == "" {
		b.ID = "generated-id"
	}
	m.builds[b.ID] = b
	m.allNames = append(m.allNames, b.Name)
	return nil
}
func (m *mockBuildRepo) GetByID(id string) (*domain.ModelBuild, error) {
	if b, ok := m.builds[id]; ok {
		return b, nil
	}
	return nil, domain.ErrBuildNotFound
}
func (m *mockBuildRepo) Update(b *domain.ModelBuild) error {
	m.builds[b.ID] = b
	return nil
}
func (m *mockBuildRepo) Delete(id string) error              { delete(m.builds, id); return nil }
func (m *mockBuildRepo) GetAllNames() ([]string, error)      { return m.allNames, nil }
func (m *mockBuildRepo) List(params interface{}) (interface{}, error) {
	return nil, nil
}
func (m *mockBuildRepo) GetByDatasourceID(id string) ([]*domain.ModelBuild, error) { return nil, nil }
func (m *mockBuildRepo) GetByFolderID(id string) ([]*domain.ModelBuild, error)     { return nil, nil }
func (m *mockBuildRepo) GetByProjectID(id string) ([]*domain.ModelBuild, error)    { return nil, nil }
func (m *mockBuildRepo) GetIDsByFolderID(id string) ([]string, error)              { return nil, nil }
func (m *mockBuildRepo) GetIDsByProjectID(id string) ([]string, error)             { return nil, nil }

// TestBuildDomain_CanStart_Pending verifies pending build can start.
func TestBuildDomain_CanStart_Pending(t *testing.T) {
	b := &domain.ModelBuild{Status: domain.BuildStatusPending}
	assert.True(t, b.CanStart())
}

// TestBuildDomain_CanStart_AlreadyRunning verifies running build cannot start.
func TestBuildDomain_CanStart_AlreadyRunning(t *testing.T) {
	b := &domain.ModelBuild{Status: domain.BuildStatusRunning}
	assert.False(t, b.CanStart())
}

// TestBuildDomain_CanCancel_Pending verifies pending build can be cancelled.
func TestBuildDomain_CanCancel_Pending(t *testing.T) {
	b := &domain.ModelBuild{Status: domain.BuildStatusPending}
	assert.True(t, b.CanCancel())
}

// TestBuildDomain_CanCancel_Running verifies running build can be cancelled.
func TestBuildDomain_CanCancel_Running(t *testing.T) {
	b := &domain.ModelBuild{Status: domain.BuildStatusRunning}
	assert.True(t, b.CanCancel())
}

// TestBuildDomain_CanCancel_Completed verifies completed build cannot be cancelled.
func TestBuildDomain_CanCancel_Completed(t *testing.T) {
	b := &domain.ModelBuild{Status: domain.BuildStatusCompleted}
	assert.False(t, b.CanCancel())
}

// TestBuildDomain_Transitions verifies status machine transitions.
func TestBuildDomain_Transitions(t *testing.T) {
	b := &domain.ModelBuild{Status: domain.BuildStatusPending}

	b.Start()
	assert.Equal(t, domain.BuildStatusRunning, b.Status)
	assert.NotNil(t, b.StartedAt)

	b.Complete(nil)
	assert.Equal(t, domain.BuildStatusCompleted, b.Status)
	assert.NotNil(t, b.CompletedAt)
	assert.True(t, b.Status.IsTerminal())
}

// TestBuildDomain_Fail records error message.
func TestBuildDomain_Fail(t *testing.T) {
	b := &domain.ModelBuild{Status: domain.BuildStatusRunning}
	b.Fail("out of memory")
	assert.Equal(t, domain.BuildStatusFailed, b.Status)
	assert.Equal(t, "out of memory", b.ErrorMessage)
}

// TestDomainService_ValidateBuild_EmptyName verifies empty name is rejected.
func TestDomainService_ValidateBuild_EmptyName(t *testing.T) {
	svc := domain.NewService()
	b := &domain.ModelBuild{Name: "", ModelType: domain.ModelTypeRegression}
	err := svc.ValidateBuild(b)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrBuildNameEmpty))
}

// TestDomainService_ValidateBuild_InvalidModelType verifies invalid type is rejected.
func TestDomainService_ValidateBuild_InvalidModelType(t *testing.T) {
	svc := domain.NewService()
	b := &domain.ModelBuild{Name: "test", ModelType: "unknown"}
	err := svc.ValidateBuild(b)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidModelType))
}

// TestDomainService_ValidateBuildNameUnique verifies duplicate detection.
func TestDomainService_ValidateBuildNameUnique(t *testing.T) {
	svc := domain.NewService()
	existing := []string{"Build A", "Build B"}

	assert.NoError(t, svc.ValidateBuildNameUnique("Build C", existing))
	assert.ErrorIs(t, svc.ValidateBuildNameUnique("Build A", existing), domain.ErrBuildNameExists)
	// Case-insensitive
	assert.ErrorIs(t, svc.ValidateBuildNameUnique("build a", existing), domain.ErrBuildNameExists)
}

// TestDomainService_CanStartBuild checks pending/running transition rules.
func TestDomainService_CanStartBuild(t *testing.T) {
	svc := domain.NewService()

	pending := &domain.ModelBuild{Status: domain.BuildStatusPending}
	assert.NoError(t, svc.CanStartBuild(pending))

	running := &domain.ModelBuild{Status: domain.BuildStatusRunning}
	assert.ErrorIs(t, svc.CanStartBuild(running), domain.ErrBuildNotPending)
}

// TestDomainService_CanCancelBuild checks cancellable statuses.
func TestDomainService_CanCancelBuild(t *testing.T) {
	svc := domain.NewService()

	pending := &domain.ModelBuild{Status: domain.BuildStatusPending}
	assert.NoError(t, svc.CanCancelBuild(pending))

	completed := &domain.ModelBuild{Status: domain.BuildStatusCompleted}
	assert.ErrorIs(t, svc.CanCancelBuild(completed), domain.ErrBuildCannotBeCancelled)
}

// TestDomainService_GetDefaultParameters verifies defaults are populated.
func TestDomainService_GetDefaultParameters(t *testing.T) {
	svc := domain.NewService()

	params := svc.GetDefaultParameters(domain.ModelTypeRegression)
	assert.Equal(t, 0.8, params.TrainTestSplit)
	assert.Equal(t, 42, params.RandomSeed)
	assert.NotEmpty(t, params.Hyperparameters)
}

// TestBuildDTO_SetDefaults verifies ListParams defaults are applied.
func TestBuildDTO_SetDefaults(t *testing.T) {
	p := &dto.ListParams{}
	p.SetDefaults()
	assert.Equal(t, 1, p.Page)
	assert.Equal(t, 20, p.PageSize)
	assert.Equal(t, "desc", p.SortDir)
}
