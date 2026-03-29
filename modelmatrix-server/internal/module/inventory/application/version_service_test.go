package application

import (
	"errors"
	"testing"

	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"
	"modelmatrix-server/internal/module/inventory/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// mockModelRepo — function-field based mock used by model_service_test.go
// ---------------------------------------------------------------------------

type mockModelRepo struct {
	getByIDWithRelations func(id string) (*domain.Model, error)
	getByID              func(id string) (*domain.Model, error)
	update               func(m *domain.Model) error
	deleteVars           func(modelID string) error
	deleteFiles          func(modelID string) error
	createVariable       func(v *domain.ModelVariable) error
	createFile           func(f *domain.ModelFile) error
	getByBuildID         func(buildID string) (*domain.Model, error)
	createVariables      func(vs []domain.ModelVariable) error
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
func (m *mockModelRepo) DeleteVariablesByModelID(id string) error {
	if m.deleteVars != nil {
		return m.deleteVars(id)
	}
	return nil
}
func (m *mockModelRepo) DeleteFilesByModelID(id string) error {
	if m.deleteFiles != nil {
		return m.deleteFiles(id)
	}
	return nil
}
func (m *mockModelRepo) CreateVariable(v *domain.ModelVariable) error {
	if m.createVariable != nil {
		return m.createVariable(v)
	}
	return nil
}
func (m *mockModelRepo) CreateFile(f *domain.ModelFile) error {
	if m.createFile != nil {
		return m.createFile(f)
	}
	return nil
}
func (m *mockModelRepo) Create(model *domain.Model) error             { return nil }
func (m *mockModelRepo) Delete(id string) error                       { return nil }
func (m *mockModelRepo) GetByName(name string) (*domain.Model, error) { return nil, nil }
func (m *mockModelRepo) GetByBuildID(buildID string) (*domain.Model, error) {
	if m.getByBuildID != nil {
		return m.getByBuildID(buildID)
	}
	return nil, nil
}
func (m *mockModelRepo) List(offset, limit int, search, status string) ([]domain.Model, int64, error) {
	return nil, 0, nil
}
func (m *mockModelRepo) UpdateStatus(id string, status domain.ModelStatus) error { return nil }
func (m *mockModelRepo) GetIDsByFolderID(id string) ([]string, error)            { return nil, nil }
func (m *mockModelRepo) GetIDsByProjectID(id string) ([]string, error)           { return nil, nil }
func (m *mockModelRepo) CreateVariables(vs []domain.ModelVariable) error {
	if m.createVariables != nil {
		return m.createVariables(vs)
	}
	return nil
}
func (m *mockModelRepo) GetVariablesByModelID(id string) ([]domain.ModelVariable, error) {
	return nil, nil
}
func (m *mockModelRepo) CreateFiles(fs []domain.ModelFile) error { return nil }
func (m *mockModelRepo) GetFilesByModelID(id string) ([]domain.ModelFile, error) {
	return nil, nil
}
func (m *mockModelRepo) GetFileByModelIDAndType(id string, ft domain.FileType) (*domain.ModelFile, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// fakeVersionRepo — in-memory version repository for version service tests
// ---------------------------------------------------------------------------

// fakeVersionRepo stores versions in memory
type fakeVersionRepo struct {
	versions  map[string]*domain.ModelVersion // versionID → version
	byModelID map[string][]*domain.ModelVersion
	nextNum   map[string]int // modelID → next version number
	createErr error
}

func newFakeVersionRepo() *fakeVersionRepo {
	return &fakeVersionRepo{
		versions:  make(map[string]*domain.ModelVersion),
		byModelID: make(map[string][]*domain.ModelVersion),
		nextNum:   make(map[string]int),
	}
}

func (r *fakeVersionRepo) Create(v *domain.ModelVersion) error {
	if r.createErr != nil {
		return r.createErr
	}
	v.ID = "ver-" + v.ModelID + "-" + string(rune('0'+v.VersionNumber))
	r.versions[v.ID] = v
	r.byModelID[v.ModelID] = append(r.byModelID[v.ModelID], v)
	return nil
}

func (r *fakeVersionRepo) ListByModelID(modelID string, limit, offset int) ([]domain.ModelVersion, int64, error) {
	all := r.byModelID[modelID]
	total := int64(len(all))
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	if offset >= len(all) {
		return []domain.ModelVersion{}, total, nil
	}
	result := make([]domain.ModelVersion, end-offset)
	for i := range result {
		result[i] = *all[offset+i]
	}
	return result, total, nil
}

func (r *fakeVersionRepo) GetByID(id string) (*domain.ModelVersion, error) {
	v, ok := r.versions[id]
	if !ok {
		return nil, domain.ErrVersionNotFound
	}
	return v, nil
}

func (r *fakeVersionRepo) GetByModelIDAndNumber(modelID string, num int) (*domain.ModelVersion, error) {
	for _, v := range r.byModelID[modelID] {
		if v.VersionNumber == num {
			return v, nil
		}
	}
	return nil, domain.ErrVersionNotFound
}

func (r *fakeVersionRepo) GetNextVersionNumber(modelID string) (int, error) {
	r.nextNum[modelID]++
	return r.nextNum[modelID], nil
}

// fakeVersionStore records EnsureVersionedCopy calls without real I/O
type fakeVersionStore struct {
	copyErr error
}

func (s *fakeVersionStore) EnsureVersionedCopy(sourcePath, contentHash, fileExt string) (string, error) {
	if s.copyErr != nil {
		return "", s.copyErr
	}
	return "minio://bucket/versions/content/" + contentHash + fileExt, nil
}

// ensure crudModelRepo (defined in model_service_crud_test.go) satisfies repository.ModelVersionRepository
var _ repository.ModelVersionRepository = (*fakeVersionRepo)(nil)

// buildVersionSvc wires up a ModelVersionServiceImpl with in-memory fakes
func buildVersionSvc(
	modelRepo *crudModelRepo,
	versionRepo *fakeVersionRepo,
	store *fakeVersionStore,
) ModelVersionService {
	return NewModelVersionService(modelRepo, versionRepo, store)
}

// ---------------------------------------------------------------------------
// CreateVersion
// ---------------------------------------------------------------------------

func TestVersionService_CreateVersion_Success(t *testing.T) {
	model := &domain.Model{
		ID:        "m1",
		Name:      "Churn Model",
		ModelType: "classification",
		Status:    domain.ModelStatusActive,
		Files: []domain.ModelFile{
			{FileType: "model", FilePath: "minio://bucket/models/m1.pkl", FileName: "m1.pkl", Checksum: "abc123"},
		},
		Variables: []domain.ModelVariable{
			{Name: "age", Role: "input"},
		},
	}
	modelRepo := newCRUDRepo(model)
	versionRepo := newFakeVersionRepo()
	svc := buildVersionSvc(modelRepo, versionRepo, &fakeVersionStore{})

	resp, err := svc.CreateVersion("m1", "alice")
	require.NoError(t, err)
	assert.Equal(t, "m1", resp.ModelID)
	assert.Equal(t, 1, resp.VersionNumber)
	assert.Equal(t, "alice", resp.CreatedBy)
}

func TestVersionService_CreateVersion_ModelNotFound(t *testing.T) {
	modelRepo := newCRUDRepo()
	svc := buildVersionSvc(modelRepo, newFakeVersionRepo(), &fakeVersionStore{})

	_, err := svc.CreateVersion("missing", "alice")
	require.Error(t, err)
}

func TestVersionService_CreateVersion_FileCopyError(t *testing.T) {
	model := &domain.Model{
		ID:   "m2",
		Name: "Model",
		Files: []domain.ModelFile{
			{FileType: "model", FilePath: "minio://bucket/m2.pkl", FileName: "m2.pkl"},
		},
	}
	modelRepo := newCRUDRepo(model)
	store := &fakeVersionStore{copyErr: errors.New("minio unavailable")}
	svc := buildVersionSvc(modelRepo, newFakeVersionRepo(), store)

	_, err := svc.CreateVersion("m2", "bob")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "minio unavailable")
}

func TestVersionService_CreateVersion_NoFiles(t *testing.T) {
	model := &domain.Model{ID: "m3", Name: "No-file Model", Files: []domain.ModelFile{}}
	modelRepo := newCRUDRepo(model)
	svc := buildVersionSvc(modelRepo, newFakeVersionRepo(), &fakeVersionStore{})

	resp, err := svc.CreateVersion("m3", "alice")
	require.NoError(t, err)
	assert.Equal(t, 1, resp.VersionNumber)
}

// ---------------------------------------------------------------------------
// ListVersions
// ---------------------------------------------------------------------------

func TestVersionService_ListVersions_Success(t *testing.T) {
	model := &domain.Model{ID: "m1", Name: "M"}
	modelRepo := newCRUDRepo(model)
	versionRepo := newFakeVersionRepo()
	// seed two versions
	_ = versionRepo.Create(&domain.ModelVersion{ModelID: "m1", VersionNumber: 1, CreatedBy: "alice"})
	_ = versionRepo.Create(&domain.ModelVersion{ModelID: "m1", VersionNumber: 2, CreatedBy: "bob"})
	svc := buildVersionSvc(modelRepo, versionRepo, &fakeVersionStore{})

	resp, err := svc.ListVersions("m1", &dto.ListVersionsParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Versions, 2)
}

func TestVersionService_ListVersions_DefaultPagination(t *testing.T) {
	model := &domain.Model{ID: "m1", Name: "M"}
	modelRepo := newCRUDRepo(model)
	svc := buildVersionSvc(modelRepo, newFakeVersionRepo(), &fakeVersionStore{})

	resp, err := svc.ListVersions("m1", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(0), resp.Total)
}

func TestVersionService_ListVersions_ModelNotFound(t *testing.T) {
	modelRepo := newCRUDRepo()
	svc := buildVersionSvc(modelRepo, newFakeVersionRepo(), &fakeVersionStore{})

	_, err := svc.ListVersions("missing", nil)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetVersion
// ---------------------------------------------------------------------------

func TestVersionService_GetVersion_Success(t *testing.T) {
	model := &domain.Model{ID: "m1", Name: "M"}
	modelRepo := newCRUDRepo(model)
	versionRepo := newFakeVersionRepo()
	_ = versionRepo.Create(&domain.ModelVersion{ModelID: "m1", VersionNumber: 1})
	versionID := "ver-m1-1"
	svc := buildVersionSvc(modelRepo, versionRepo, &fakeVersionStore{})

	resp, err := svc.GetVersion("m1", versionID)
	require.NoError(t, err)
	assert.Equal(t, versionID, resp.ID)
}

func TestVersionService_GetVersion_WrongModel_ReturnsNotFound(t *testing.T) {
	// Version belongs to m1 but caller passes m2 — must be rejected (security check)
	versionRepo := newFakeVersionRepo()
	_ = versionRepo.Create(&domain.ModelVersion{ModelID: "m1", VersionNumber: 1})
	versionID := "ver-m1-1"
	svc := buildVersionSvc(newCRUDRepo(), versionRepo, &fakeVersionStore{})

	_, err := svc.GetVersion("m2", versionID)
	require.Error(t, err)
	assert.Equal(t, domain.ErrVersionNotFound, err)
}

func TestVersionService_GetVersion_NotFound(t *testing.T) {
	svc := buildVersionSvc(newCRUDRepo(), newFakeVersionRepo(), &fakeVersionStore{})
	_, err := svc.GetVersion("m1", "no-such-version")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// RestoreVersion
// ---------------------------------------------------------------------------

func TestVersionService_RestoreVersion_Success(t *testing.T) {
	current := &domain.Model{
		ID:   "m1",
		Name: "Stale Model",
		Variables: []domain.ModelVariable{{Name: "old", Role: "input"}},
		Files:     []domain.ModelFile{{FileType: "model", FileName: "old.pkl"}},
	}
	modelRepo := newCRUDRepo(current)
	versionRepo := newFakeVersionRepo()
	_ = versionRepo.Create(&domain.ModelVersion{
		ModelID:       "m1",
		VersionNumber: 1,
		Name:          "Snapshot Name",
		Status:        domain.ModelStatusActive,
		Variables:     []domain.ModelVariable{{Name: "feat1", Role: "input"}},
		Files:         []domain.ModelFile{{FileType: "model", FileName: "snap.pkl"}},
	})
	versionID := "ver-m1-1"
	svc := buildVersionSvc(modelRepo, versionRepo, &fakeVersionStore{})

	resp, err := svc.RestoreVersion("m1", versionID, "alice")
	require.NoError(t, err)
	assert.NotNil(t, resp)
	// Underlying model should have been updated with snapshot's name
	updated := modelRepo.models["m1"]
	assert.Equal(t, "Snapshot Name", updated.Name)
}

func TestVersionService_RestoreVersion_VersionNotFound(t *testing.T) {
	svc := buildVersionSvc(newCRUDRepo(), newFakeVersionRepo(), &fakeVersionStore{})
	_, err := svc.RestoreVersion("m1", "bad-version", "alice")
	require.Error(t, err)
}

func TestVersionService_RestoreVersion_WrongModel_ReturnsNotFound(t *testing.T) {
	versionRepo := newFakeVersionRepo()
	_ = versionRepo.Create(&domain.ModelVersion{ModelID: "m1", VersionNumber: 1})
	svc := buildVersionSvc(newCRUDRepo(), versionRepo, &fakeVersionStore{})

	_, err := svc.RestoreVersion("m2", "ver-m1-1", "alice")
	require.Error(t, err)
	assert.Equal(t, domain.ErrVersionNotFound, err)
}

func TestVersionService_RestoreVersion_ModelNotFound(t *testing.T) {
	versionRepo := newFakeVersionRepo()
	_ = versionRepo.Create(&domain.ModelVersion{ModelID: "m1", VersionNumber: 1})
	// modelRepo is empty — GetByIDWithRelations will return not found
	svc := buildVersionSvc(newCRUDRepo(), versionRepo, &fakeVersionStore{})

	_, err := svc.RestoreVersion("m1", "ver-m1-1", "alice")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// variableToResponse / fileToResponse — exercised through GetVersion
// These are pure struct-mapping functions; testing through GetVersion ensures
// every field is populated correctly without trivially-duplicating the structs.
// ---------------------------------------------------------------------------

func TestVersionService_GetVersion_VariableAndFileFieldsMapped(t *testing.T) {
	imp := 0.77
	model := &domain.Model{
		ID: "m1", Name: "M", Status: domain.ModelStatusActive,
	}
	versionRepo := newFakeVersionRepo()
	_ = versionRepo.Create(&domain.ModelVersion{
		ModelID:       "m1",
		VersionNumber: 1,
		// ModelVersion stores fields inline (no Snapshot wrapper)
		Name:      "M",
		Algorithm: "random_forest",
		Variables: []domain.ModelVariable{
			{
				ID: "v1", ModelID: "m1", Name: "amount",
				DataType:   domain.VariableDataTypeNumeric,
				Role:       domain.VariableRoleInput,
				Importance: &imp,
				Ordinal:    0,
			},
		},
		Files: []domain.ModelFile{
			{
				ID: "f1", ModelID: "m1", FileType: domain.FileTypeModel,
				FilePath: "models/m1/rf.pkl", FileName: "rf.pkl",
				Description: "Random forest model",
			},
		},
	})
	svc := buildVersionSvc(newCRUDRepo(model), versionRepo, &fakeVersionStore{})

	detail, err := svc.GetVersion("m1", "ver-m1-1")
	require.NoError(t, err)

	// variableToResponse fields
	require.Len(t, detail.Variables, 1)
	v := detail.Variables[0]
	assert.Equal(t, "v1", v.ID)
	assert.Equal(t, "amount", v.Name)
	assert.Equal(t, "numeric", v.DataType)
	assert.Equal(t, "input", v.Role)
	require.NotNil(t, v.Importance)
	assert.Equal(t, imp, *v.Importance)

	// fileToResponse fields
	require.Len(t, detail.Files, 1)
	f := detail.Files[0]
	assert.Equal(t, "f1", f.ID)
	assert.Equal(t, "model", f.FileType)
	assert.Equal(t, "models/m1/rf.pkl", f.FilePath)
	assert.Equal(t, "rf.pkl", f.FileName)
	assert.Equal(t, "Random forest model", f.Description)
}
