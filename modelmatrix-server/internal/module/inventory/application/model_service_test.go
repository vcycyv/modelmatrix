package application

import (
	"errors"
	"io"
	"testing"

	"modelmatrix-server/internal/infrastructure/fileservice"
	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFileService is a no-op implementation of fileservice.FileService for unit tests.
type mockFileService struct{}

func (m *mockFileService) Save(_ string, _ io.Reader, _ int64) (*fileservice.FileInfo, error) {
	return &fileservice.FileInfo{}, nil
}
func (m *mockFileService) SaveWithPath(_, _ string, _ io.Reader, _ int64) (*fileservice.FileInfo, error) {
	return &fileservice.FileInfo{}, nil
}
func (m *mockFileService) Get(_ string) (io.ReadCloser, *fileservice.FileInfo, error) {
	return io.NopCloser(nil), nil, nil
}
func (m *mockFileService) ReadFileContent(_ string) ([]byte, *fileservice.FileInfo, error) {
	return nil, nil, nil
}
func (m *mockFileService) Delete(_ string) error                      { return nil }
func (m *mockFileService) Exists(_ string) bool                       { return false }
func (m *mockFileService) GetInfo(_ string) (*fileservice.FileInfo, error) { return nil, nil }
func (m *mockFileService) ValidateParquet(_ string) error             { return nil }
func (m *mockFileService) ValidateCSV(_ string) error                 { return nil }
func (m *mockFileService) HealthCheck() error                         { return nil }

// buildModelRepo builds a mockModelRepo with a model of the given status.
func buildModelRepo(id string, status domain.ModelStatus) *mockModelRepo {
	model := &domain.Model{
		ID:     id,
		Name:   "Test Model",
		Status: status,
	}
	return &mockModelRepo{
		getByID: func(modelID string) (*domain.Model, error) {
			if modelID == id {
				return model, nil
			}
			return nil, domain.ErrModelNotFound
		},
		getByIDWithRelations: func(modelID string) (*domain.Model, error) {
			if modelID == id {
				return model, nil
			}
			return nil, domain.ErrModelNotFound
		},
		update: func(m *domain.Model) error {
			model.Status = m.Status
			return nil
		},
	}
}

// TestActivateModel_FromDraft verifies that a draft model can be activated.
func TestActivateModel_FromDraft(t *testing.T) {
	repo := buildModelRepo("m1", domain.ModelStatusDraft)
	svc := NewModelService(repo, domain.NewService(), &mockFileService{})

	result, err := svc.Activate("m1")
	require.NoError(t, err)
	assert.Equal(t, "active", result.Status)
}

// TestActivateModel_WhenAlreadyActive verifies that activating an active model errors.
func TestActivateModel_WhenAlreadyActive(t *testing.T) {
	repo := buildModelRepo("m1", domain.ModelStatusActive)
	svc := NewModelService(repo, domain.NewService(), &mockFileService{})

	_, err := svc.Activate("m1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelCannotActivate))
}

// TestDeactivateModel_WhenActive verifies that an active model can be deactivated.
func TestDeactivateModel_WhenActive(t *testing.T) {
	repo := buildModelRepo("m1", domain.ModelStatusActive)
	svc := NewModelService(repo, domain.NewService(), &mockFileService{})

	result, err := svc.Deactivate("m1")
	require.NoError(t, err)
	assert.Equal(t, "inactive", result.Status)
}

// TestDeactivateModel_WhenNotActive verifies that deactivating a non-active model errors.
func TestDeactivateModel_WhenNotActive(t *testing.T) {
	repo := buildModelRepo("m1", domain.ModelStatusDraft)
	svc := NewModelService(repo, domain.NewService(), &mockFileService{})

	_, err := svc.Deactivate("m1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelCannotDeactivate))
}

// TestActivateModel_NotFound verifies that activating a non-existent model returns ErrModelNotFound.
func TestActivateModel_NotFound(t *testing.T) {
	repo := buildModelRepo("existing", domain.ModelStatusDraft)
	svc := NewModelService(repo, domain.NewService(), &mockFileService{})

	_, err := svc.Activate("nonexistent")
	require.Error(t, err)
}

// TestDeleteModel_WhenDraft verifies that a draft model can be deleted.
func TestDeleteModel_WhenDraft(t *testing.T) {
	repo := buildModelRepo("m1", domain.ModelStatusDraft)
	svc := NewModelService(repo, domain.NewService(), &mockFileService{})

	err := svc.Delete("m1")
	require.NoError(t, err)
}

// TestDeleteModel_WhenActive verifies that deleting an active model fails.
func TestDeleteModel_WhenActive(t *testing.T) {
	repo := buildModelRepo("m1", domain.ModelStatusActive)
	svc := NewModelService(repo, domain.NewService(), &mockFileService{})

	err := svc.Delete("m1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelCannotDelete))
}

// TestUpdateModel_EmptyNameRejected verifies that an empty name update returns an error.
func TestUpdateModel_EmptyNameRejected(t *testing.T) {
	repo := buildModelRepo("m1", domain.ModelStatusDraft)
	svc := NewModelService(repo, domain.NewService(), &mockFileService{})

	empty := ""
	_, err := svc.Update("m1", &dto.UpdateModelRequest{Name: &empty})
	require.Error(t, err)
}
