package application

import (
	"errors"
	"testing"

	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// crudModelRepo is a more configurable mock for Create/List/GetByID/Update tests.
type crudModelRepo struct {
	models            map[string]*domain.Model
	getByNameFn       func(name string) (*domain.Model, error)
	createFn          func(model *domain.Model) error
	listFn            func(offset, limit int, search, status string) ([]domain.Model, int64, error)
	getIDsByFolderFn  func(folderID string) []string
	getIDsByProjectFn func(projectID string) []string
}

func newCRUDRepo(models ...*domain.Model) *crudModelRepo {
	r := &crudModelRepo{models: make(map[string]*domain.Model)}
	for _, m := range models {
		r.models[m.ID] = m
	}
	return r
}

func (r *crudModelRepo) Create(model *domain.Model) error {
	if r.createFn != nil {
		return r.createFn(model)
	}
	if model.ID == "" {
		model.ID = "gen-id"
	}
	r.models[model.ID] = model
	return nil
}
func (r *crudModelRepo) Update(model *domain.Model) error     { r.models[model.ID] = model; return nil }
func (r *crudModelRepo) Delete(id string) error               { delete(r.models, id); return nil }
func (r *crudModelRepo) UpdateStatus(id string, status domain.ModelStatus) error { return nil }
func (r *crudModelRepo) GetIDsByFolderID(id string) ([]string, error) {
	if r.getIDsByFolderFn != nil {
		return r.getIDsByFolderFn(id), nil
	}
	return nil, nil
}
func (r *crudModelRepo) GetIDsByProjectID(id string) ([]string, error) {
	if r.getIDsByProjectFn != nil {
		return r.getIDsByProjectFn(id), nil
	}
	return nil, nil
}
func (r *crudModelRepo) CreateVariable(v *domain.ModelVariable) error            { return nil }
func (r *crudModelRepo) CreateVariables(vs []domain.ModelVariable) error         { return nil }
func (r *crudModelRepo) GetVariablesByModelID(id string) ([]domain.ModelVariable, error) {
	return nil, nil
}
func (r *crudModelRepo) DeleteVariablesByModelID(id string) error { return nil }
func (r *crudModelRepo) CreateFile(f *domain.ModelFile) error     { return nil }
func (r *crudModelRepo) CreateFiles(fs []domain.ModelFile) error  { return nil }
func (r *crudModelRepo) GetFilesByModelID(id string) ([]domain.ModelFile, error) {
	return nil, nil
}
func (r *crudModelRepo) GetFileByModelIDAndType(id string, ft domain.FileType) (*domain.ModelFile, error) {
	return nil, nil
}
func (r *crudModelRepo) DeleteFilesByModelID(id string) error { return nil }
func (r *crudModelRepo) GetByBuildID(buildID string) (*domain.Model, error) { return nil, nil }

func (r *crudModelRepo) GetByName(name string) (*domain.Model, error) {
	if r.getByNameFn != nil {
		return r.getByNameFn(name)
	}
	for _, m := range r.models {
		if m.Name == name {
			return m, nil
		}
	}
	return nil, nil
}
func (r *crudModelRepo) GetByID(id string) (*domain.Model, error) {
	if m, ok := r.models[id]; ok {
		return m, nil
	}
	return nil, domain.ErrModelNotFound
}
func (r *crudModelRepo) GetByIDWithRelations(id string) (*domain.Model, error) {
	return r.GetByID(id)
}
func (r *crudModelRepo) List(offset, limit int, search, status string) ([]domain.Model, int64, error) {
	if r.listFn != nil {
		return r.listFn(offset, limit, search, status)
	}
	var result []domain.Model
	for _, m := range r.models {
		result = append(result, *m)
	}
	return result, int64(len(result)), nil
}

func buildCRUDSvc(repo *crudModelRepo) ModelService {
	return NewModelService(repo, domain.NewService(), &mockFileService{})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestModelServiceCRUD_Create_Valid(t *testing.T) {
	repo := newCRUDRepo()
	svc := buildCRUDSvc(repo)

	resp, err := svc.Create(&dto.CreateModelRequest{
		Name:         "My Model",
		BuildID:      "550e8400-e29b-41d4-a716-446655440001",
		DatasourceID: "550e8400-e29b-41d4-a716-446655440002",
		Algorithm:    "xgboost",
		ModelType:    "regression",
		TargetColumn: "price",
	}, "alice")

	require.NoError(t, err)
	assert.Equal(t, "My Model", resp.Name)
	assert.Equal(t, "draft", resp.Status)
	assert.Equal(t, "alice", resp.CreatedBy)
}

func TestModelServiceCRUD_Create_DuplicateName(t *testing.T) {
	existing := &domain.Model{ID: "m1", Name: "My Model", Status: domain.ModelStatusDraft}
	repo := newCRUDRepo(existing)
	svc := buildCRUDSvc(repo)

	_, err := svc.Create(&dto.CreateModelRequest{
		Name:         "My Model", // duplicate
		BuildID:      "550e8400-e29b-41d4-a716-446655440001",
		DatasourceID: "550e8400-e29b-41d4-a716-446655440002",
		Algorithm:    "xgboost",
		ModelType:    "regression",
		TargetColumn: "price",
	}, "bob")

	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelNameExists))
}

func TestModelServiceCRUD_Create_EmptyName(t *testing.T) {
	svc := buildCRUDSvc(newCRUDRepo())
	_, err := svc.Create(&dto.CreateModelRequest{
		Name:         "",
		BuildID:      "550e8400-e29b-41d4-a716-446655440001",
		DatasourceID: "550e8400-e29b-41d4-a716-446655440002",
		Algorithm:    "xgboost",
		ModelType:    "regression",
		TargetColumn: "price",
	}, "alice")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelNameEmpty))
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestModelServiceCRUD_GetByID_Found(t *testing.T) {
	m := &domain.Model{
		ID:     "m1",
		Name:   "Test",
		Status: domain.ModelStatusActive,
		Variables: []domain.ModelVariable{
			{Name: "x", Role: domain.VariableRoleInput},
		},
	}
	repo := newCRUDRepo(m)
	svc := buildCRUDSvc(repo)

	detail, err := svc.GetByID("m1")
	require.NoError(t, err)
	assert.Equal(t, "m1", detail.ID)
	assert.Equal(t, "active", detail.Status)
}

func TestModelServiceCRUD_GetByID_NotFound(t *testing.T) {
	svc := buildCRUDSvc(newCRUDRepo())
	_, err := svc.GetByID("missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelNotFound))
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestModelServiceCRUD_List_ReturnsPaginated(t *testing.T) {
	models := []*domain.Model{
		{ID: "m1", Name: "Alpha", Status: domain.ModelStatusDraft},
		{ID: "m2", Name: "Beta", Status: domain.ModelStatusActive},
	}
	svc := buildCRUDSvc(newCRUDRepo(models...))

	resp, err := svc.List(&dto.ListParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Models, 2)
}

func TestModelServiceCRUD_List_Empty(t *testing.T) {
	svc := buildCRUDSvc(newCRUDRepo())
	resp, err := svc.List(&dto.ListParams{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Total)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestModelServiceCRUD_Update_Name(t *testing.T) {
	m := &domain.Model{ID: "m1", Name: "Old Name", Status: domain.ModelStatusDraft}
	repo := newCRUDRepo(m)
	svc := buildCRUDSvc(repo)

	newName := "New Name"
	resp, err := svc.Update("m1", &dto.UpdateModelRequest{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "New Name", resp.Name)
}

func TestModelServiceCRUD_Update_DuplicateName(t *testing.T) {
	m1 := &domain.Model{ID: "m1", Name: "A", Status: domain.ModelStatusDraft}
	m2 := &domain.Model{ID: "m2", Name: "B", Status: domain.ModelStatusDraft}
	repo := newCRUDRepo(m1, m2)
	svc := buildCRUDSvc(repo)

	// Try to rename m1 to "B" (already taken by m2)
	newName := "B"
	_, err := svc.Update("m1", &dto.UpdateModelRequest{Name: &newName})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelNameExists))
}

func TestModelServiceCRUD_Update_NotFound(t *testing.T) {
	svc := buildCRUDSvc(newCRUDRepo())
	name := "X"
	_, err := svc.Update("missing", &dto.UpdateModelRequest{Name: &name})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrModelNotFound))
}

func TestModelServiceCRUD_Update_ClearsDescription(t *testing.T) {
	m := &domain.Model{ID: "m1", Name: "Model", Description: "old desc", Status: domain.ModelStatusDraft}
	repo := newCRUDRepo(m)
	svc := buildCRUDSvc(repo)

	// Passing nil Description should clear it
	resp, err := svc.Update("m1", &dto.UpdateModelRequest{})
	require.NoError(t, err)
	assert.Empty(t, resp.Description)
}
