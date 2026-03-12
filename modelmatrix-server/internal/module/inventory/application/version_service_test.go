package application

import (
	"errors"
	"testing"

	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockModelRepo struct {
	getByIDWithRelations func(id string) (*domain.Model, error)
	getByID              func(id string) (*domain.Model, error)
	update               func(m *domain.Model) error
	deleteVars           func(modelID string) error
	deleteFiles          func(modelID string) error
	createVariable       func(v *domain.ModelVariable) error
	createFile           func(f *domain.ModelFile) error
}

func (m *mockModelRepo) GetByIDWithRelations(id string) (*domain.Model, error) {
	if m.getByIDWithRelations != nil {
		return m.getByIDWithRelations(id)
	}
	return nil, domain.ErrModelNotFound
}
func (m *mockModelRepo) GetByID(id string) (*domain.Model, error) {
	if m.getByID != nil {
		return m.getByID(id)
	}
	return nil, domain.ErrModelNotFound
}
func (m *mockModelRepo) Update(model *domain.Model) error {
	if m.update != nil {
		return m.update(model)
	}
	return nil
}
func (m *mockModelRepo) DeleteVariablesByModelID(modelID string) error {
	if m.deleteVars != nil {
		return m.deleteVars(modelID)
	}
	return nil
}
func (m *mockModelRepo) DeleteFilesByModelID(modelID string) error {
	if m.deleteFiles != nil {
		return m.deleteFiles(modelID)
	}
	return nil
}

// Satisfy remaining ModelRepository interface with no-op or panic as needed
func (m *mockModelRepo) Create(model *domain.Model) error { return nil }
func (m *mockModelRepo) Delete(id string) error          { return nil }
func (m *mockModelRepo) GetByName(name string) (*domain.Model, error) {
	return nil, nil
}
func (m *mockModelRepo) GetByBuildID(buildID string) (*domain.Model, error) {
	return nil, nil
}
func (m *mockModelRepo) List(offset, limit int, search, status string) ([]domain.Model, int64, error) {
	return nil, 0, nil
}
func (m *mockModelRepo) UpdateStatus(id string, status domain.ModelStatus) error {
	return nil
}
func (m *mockModelRepo) CreateVariable(variable *domain.ModelVariable) error {
	if m.createVariable != nil {
		return m.createVariable(variable)
	}
	return nil
}
func (m *mockModelRepo) CreateVariables(variables []domain.ModelVariable) error {
	return nil
}
func (m *mockModelRepo) GetVariablesByModelID(modelID string) ([]domain.ModelVariable, error) {
	return nil, nil
}
func (m *mockModelRepo) CreateFile(file *domain.ModelFile) error {
	if m.createFile != nil {
		return m.createFile(file)
	}
	return nil
}
func (m *mockModelRepo) CreateFiles(files []domain.ModelFile) error {
	return nil
}
func (m *mockModelRepo) GetFilesByModelID(modelID string) ([]domain.ModelFile, error) {
	return nil, nil
}
func (m *mockModelRepo) GetFileByModelIDAndType(modelID string, fileType domain.FileType) (*domain.ModelFile, error) {
	return nil, domain.ErrFileNotFound
}
func (m *mockModelRepo) GetIDsByFolderID(folderID string) ([]string, error) {
	return nil, nil
}
func (m *mockModelRepo) GetIDsByProjectID(projectID string) ([]string, error) {
	return nil, nil
}

type mockVersionRepo struct {
	getByID     func(versionID string) (*domain.ModelVersion, error)
	list        func(modelID string, limit, offset int) ([]domain.ModelVersion, int64, error)
	create      func(version *domain.ModelVersion) error
	nextVersion func(modelID string) (int, error)
}

func (m *mockVersionRepo) GetByID(versionID string) (*domain.ModelVersion, error) {
	if m.getByID != nil {
		return m.getByID(versionID)
	}
	return nil, domain.ErrVersionNotFound
}
func (m *mockVersionRepo) ListByModelID(modelID string, limit, offset int) ([]domain.ModelVersion, int64, error) {
	if m.list != nil {
		return m.list(modelID, limit, offset)
	}
	return nil, 0, nil
}
func (m *mockVersionRepo) Create(version *domain.ModelVersion) error {
	if m.create != nil {
		return m.create(version)
	}
	return nil
}
func (m *mockVersionRepo) GetNextVersionNumber(modelID string) (int, error) {
	if m.nextVersion != nil {
		return m.nextVersion(modelID)
	}
	return 1, nil
}
func (m *mockVersionRepo) GetByModelIDAndNumber(modelID string, versionNumber int) (*domain.ModelVersion, error) {
	return nil, domain.ErrVersionNotFound
}

type mockVersionStore struct {
	ensureCopy func(sourcePath, contentHash, fileExt string) (string, error)
}

func (m *mockVersionStore) EnsureVersionedCopy(sourcePath, contentHash, fileExt string) (string, error) {
	if m.ensureCopy != nil {
		return m.ensureCopy(sourcePath, contentHash, fileExt)
	}
	return "minio://bucket/versions/content/abc", nil
}

func TestGetVersion_WrongModelID(t *testing.T) {
	versionID := "v1"
	modelID := "m1"
	otherModelID := "m2"
	svc := NewModelVersionService(
		&mockModelRepo{},
		&mockVersionRepo{
			getByID: func(id string) (*domain.ModelVersion, error) {
				return &domain.ModelVersion{
					ID:      versionID,
					ModelID: otherModelID,
				}, nil
			},
		},
		&mockVersionStore{},
	)
	_, err := svc.GetVersion(modelID, versionID)
	require.True(t, errors.Is(err, domain.ErrVersionNotFound))
}

func TestListVersions_ModelNotFound(t *testing.T) {
	svc := NewModelVersionService(
		&mockModelRepo{
			getByID: func(id string) (*domain.Model, error) {
				return nil, domain.ErrModelNotFound
			},
		},
		&mockVersionRepo{},
		&mockVersionStore{},
	)
	_, err := svc.ListVersions("nonexistent", &dto.ListVersionsParams{})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelNotFound))
}

func TestRestoreVersion_WrongModelID(t *testing.T) {
	svc := NewModelVersionService(
		&mockModelRepo{},
		&mockVersionRepo{
			getByID: func(id string) (*domain.ModelVersion, error) {
				return &domain.ModelVersion{
					ID:      "v1",
					ModelID: "other-model",
				}, nil
			},
		},
		&mockVersionStore{},
	)
	_, err := svc.RestoreVersion("my-model", "v1", "user")
	require.True(t, errors.Is(err, domain.ErrVersionNotFound))
}
