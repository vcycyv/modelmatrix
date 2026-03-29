package application

import (
	"errors"
	"io"
	"testing"

	"modelmatrix-server/internal/infrastructure/fileservice"
	"modelmatrix-server/internal/module/datasource/domain"
	"modelmatrix-server/internal/module/datasource/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockCollectionRepo struct {
	collections         map[string]*domain.Collection
	names               []string
	datasourceCountByID map[string]int64
	// error injection
	createErr              error
	deleteErr              error
	deleteWithDatasourcesErr error
}

func newMockCollectionRepo(collections ...*domain.Collection) *mockCollectionRepo {
	m := &mockCollectionRepo{
		collections:         make(map[string]*domain.Collection),
		datasourceCountByID: make(map[string]int64),
	}
	for _, c := range collections {
		m.collections[c.ID] = c
		m.names = append(m.names, c.Name)
	}
	return m
}

func (m *mockCollectionRepo) Create(c *domain.Collection) error {
	if m.createErr != nil {
		return m.createErr
	}
	if c.ID == "" {
		c.ID = "gen-id"
	}
	m.collections[c.ID] = c
	m.names = append(m.names, c.Name)
	return nil
}

func (m *mockCollectionRepo) Update(c *domain.Collection) error {
	m.collections[c.ID] = c
	return nil
}

func (m *mockCollectionRepo) Delete(id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.collections, id)
	return nil
}

func (m *mockCollectionRepo) DeleteWithDatasources(id string) error {
	if m.deleteWithDatasourcesErr != nil {
		return m.deleteWithDatasourcesErr
	}
	delete(m.collections, id)
	return nil
}

func (m *mockCollectionRepo) GetByID(id string) (*domain.Collection, error) {
	if c, ok := m.collections[id]; ok {
		return c, nil
	}
	return nil, domain.ErrCollectionNotFound
}

func (m *mockCollectionRepo) GetByName(name string) (*domain.Collection, error) {
	return nil, nil
}

func (m *mockCollectionRepo) List(offset, limit int, search string) ([]domain.Collection, int64, error) {
	var result []domain.Collection
	for _, c := range m.collections {
		result = append(result, *c)
	}
	return result, int64(len(result)), nil
}

func (m *mockCollectionRepo) GetAllNames() ([]string, error) {
	return m.names, nil
}

func (m *mockCollectionRepo) CountDatasources(collectionID string) (int64, error) {
	return m.datasourceCountByID[collectionID], nil
}

// mockDatasourceRepoSimple satisfies repository.DatasourceRepository with all no-ops.
type mockDatasourceRepoSimple struct {
	datasources []domain.Datasource
}

func (m *mockDatasourceRepoSimple) Create(ds *domain.Datasource) error     { return nil }
func (m *mockDatasourceRepoSimple) Update(ds *domain.Datasource) error     { return nil }
func (m *mockDatasourceRepoSimple) Delete(id string) error                  { return nil }
func (m *mockDatasourceRepoSimple) GetByID(id string) (*domain.Datasource, error) {
	return nil, domain.ErrDatasourceNotFound
}
func (m *mockDatasourceRepoSimple) GetByIDWithColumns(id string) (*domain.Datasource, error) {
	return nil, domain.ErrDatasourceNotFound
}
func (m *mockDatasourceRepoSimple) GetByName(collectionID, name string) (*domain.Datasource, error) {
	return nil, nil
}
func (m *mockDatasourceRepoSimple) List(collectionID *string, offset, limit int, search string) ([]domain.Datasource, int64, error) {
	return nil, 0, nil
}
func (m *mockDatasourceRepoSimple) ListByCollection(collectionID string, offset, limit int) ([]domain.Datasource, int64, error) {
	return m.datasources, int64(len(m.datasources)), nil
}
func (m *mockDatasourceRepoSimple) GetNamesInCollection(collectionID string) ([]string, error) {
	return nil, nil
}
func (m *mockDatasourceRepoSimple) UpdateFilePath(id, filePath string) error { return nil }

type mockFileServiceSimple struct{}

func (m *mockFileServiceSimple) Save(_ string, _ io.Reader, _ int64) (*fileservice.FileInfo, error) {
	return &fileservice.FileInfo{}, nil
}
func (m *mockFileServiceSimple) SaveWithPath(_, _ string, _ io.Reader, _ int64) (*fileservice.FileInfo, error) {
	return &fileservice.FileInfo{}, nil
}
func (m *mockFileServiceSimple) Get(_ string) (io.ReadCloser, *fileservice.FileInfo, error) {
	return io.NopCloser(nil), nil, nil
}
func (m *mockFileServiceSimple) ReadFileContent(_ string) ([]byte, *fileservice.FileInfo, error) {
	return nil, nil, nil
}
func (m *mockFileServiceSimple) Delete(_ string) error                          { return nil }
func (m *mockFileServiceSimple) Exists(_ string) bool                           { return false }
func (m *mockFileServiceSimple) GetInfo(_ string) (*fileservice.FileInfo, error) { return nil, nil }
func (m *mockFileServiceSimple) ValidateParquet(_ string) error                 { return nil }
func (m *mockFileServiceSimple) ValidateCSV(_ string) error                     { return nil }
func (m *mockFileServiceSimple) HealthCheck() error                             { return nil }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildCollectionSvc(colRepo *mockCollectionRepo, dsRepo *mockDatasourceRepoSimple) CollectionService {
	return NewCollectionService(colRepo, dsRepo, domain.NewService(), &mockFileServiceSimple{})
}

func sampleCollection(id, name string) *domain.Collection {
	return &domain.Collection{ID: id, Name: name}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCollectionService_Create_Valid(t *testing.T) {
	colRepo := newMockCollectionRepo()
	svc := buildCollectionSvc(colRepo, &mockDatasourceRepoSimple{})

	req := &dto.CreateCollectionRequest{Name: "My Collection", Description: "desc"}
	resp, err := svc.Create(req, "alice")

	require.NoError(t, err)
	assert.Equal(t, "My Collection", resp.Name)
	assert.Equal(t, "alice", resp.CreatedBy)
}

func TestCollectionService_Create_EmptyName(t *testing.T) {
	svc := buildCollectionSvc(newMockCollectionRepo(), &mockDatasourceRepoSimple{})
	_, err := svc.Create(&dto.CreateCollectionRequest{Name: "   "}, "alice")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrCollectionNameEmpty))
}

func TestCollectionService_Create_DuplicateName(t *testing.T) {
	colRepo := newMockCollectionRepo(sampleCollection("c1", "Sales Data"))
	svc := buildCollectionSvc(colRepo, &mockDatasourceRepoSimple{})

	_, err := svc.Create(&dto.CreateCollectionRequest{Name: "Sales Data"}, "bob")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrCollectionNameExists))
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestCollectionService_GetByID_Found(t *testing.T) {
	colRepo := newMockCollectionRepo(sampleCollection("c1", "Prod"))
	svc := buildCollectionSvc(colRepo, &mockDatasourceRepoSimple{})

	resp, err := svc.GetByID("c1")
	require.NoError(t, err)
	assert.Equal(t, "c1", resp.ID)
}

func TestCollectionService_GetByID_NotFound(t *testing.T) {
	svc := buildCollectionSvc(newMockCollectionRepo(), &mockDatasourceRepoSimple{})
	_, err := svc.GetByID("nonexistent")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrCollectionNotFound))
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestCollectionService_Update_Name(t *testing.T) {
	colRepo := newMockCollectionRepo(sampleCollection("c1", "Old Name"))
	svc := buildCollectionSvc(colRepo, &mockDatasourceRepoSimple{})

	newName := "New Name"
	resp, err := svc.Update("c1", &dto.UpdateCollectionRequest{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "New Name", resp.Name)
}

func TestCollectionService_Update_EmptyName(t *testing.T) {
	colRepo := newMockCollectionRepo(sampleCollection("c1", "Some Name"))
	svc := buildCollectionSvc(colRepo, &mockDatasourceRepoSimple{})

	empty := ""
	_, err := svc.Update("c1", &dto.UpdateCollectionRequest{Name: &empty})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrCollectionNameEmpty))
}

func TestCollectionService_Update_NotFound(t *testing.T) {
	svc := buildCollectionSvc(newMockCollectionRepo(), &mockDatasourceRepoSimple{})
	name := "X"
	_, err := svc.Update("missing", &dto.UpdateCollectionRequest{Name: &name})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrCollectionNotFound))
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestCollectionService_Delete_EmptyCollection(t *testing.T) {
	colRepo := newMockCollectionRepo(sampleCollection("c1", "Empty"))
	svc := buildCollectionSvc(colRepo, &mockDatasourceRepoSimple{})

	err := svc.Delete("c1", false)
	require.NoError(t, err)
	// Collection should be gone
	_, err2 := svc.GetByID("c1")
	assert.True(t, errors.Is(err2, domain.ErrCollectionNotFound))
}

func TestCollectionService_Delete_NonEmptyWithoutForce_Rejected(t *testing.T) {
	colRepo := newMockCollectionRepo(sampleCollection("c1", "Has Data"))
	colRepo.datasourceCountByID["c1"] = 3
	svc := buildCollectionSvc(colRepo, &mockDatasourceRepoSimple{})

	err := svc.Delete("c1", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrCollectionHasDatasources))
}

func TestCollectionService_Delete_NonEmptyWithForce_Succeeds(t *testing.T) {
	colRepo := newMockCollectionRepo(sampleCollection("c1", "Has Data"))
	colRepo.datasourceCountByID["c1"] = 2
	dsRepo := &mockDatasourceRepoSimple{
		datasources: []domain.Datasource{
			{ID: "ds1", CollectionID: "c1", FilePath: "path/file.parquet"},
			{ID: "ds2", CollectionID: "c1", FilePath: ""},
		},
	}
	svc := buildCollectionSvc(colRepo, dsRepo)

	err := svc.Delete("c1", true)
	require.NoError(t, err)
}

func TestCollectionService_Delete_NotFound(t *testing.T) {
	svc := buildCollectionSvc(newMockCollectionRepo(), &mockDatasourceRepoSimple{})
	err := svc.Delete("missing", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrCollectionNotFound))
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestCollectionService_List_ReturnsPaginated(t *testing.T) {
	colRepo := newMockCollectionRepo(
		sampleCollection("c1", "Alpha"),
		sampleCollection("c2", "Beta"),
	)
	svc := buildCollectionSvc(colRepo, &mockDatasourceRepoSimple{})

	params := &dto.ListParams{Page: 1, PageSize: 10}
	resp, err := svc.List(params)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
	assert.Len(t, resp.Collections, 2)
}

func TestCollectionService_List_Defaults(t *testing.T) {
	svc := buildCollectionSvc(newMockCollectionRepo(), &mockDatasourceRepoSimple{})
	// Empty params — SetDefaults should be called internally
	resp, err := svc.List(&dto.ListParams{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

// ---------------------------------------------------------------------------
// Additional error path coverage
// ---------------------------------------------------------------------------

func TestCollectionService_Create_RepositoryError(t *testing.T) {
	colRepo := newMockCollectionRepo()
	colRepo.createErr = errors.New("db write error")
	svc := buildCollectionSvc(colRepo, &mockDatasourceRepoSimple{})

	_, err := svc.Create(&dto.CreateCollectionRequest{Name: "New"}, "alice")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db write error")
}

func TestCollectionService_Update_DescriptionOnly(t *testing.T) {
	col := sampleCollection("c1", "Alpha")
	colRepo := newMockCollectionRepo(col)
	svc := buildCollectionSvc(colRepo, &mockDatasourceRepoSimple{})
	desc := "new desc"

	resp, err := svc.Update("c1", &dto.UpdateCollectionRequest{Description: &desc})
	require.NoError(t, err)
	assert.Equal(t, "new desc", resp.Description)
}

func TestCollectionService_Update_DuplicateName(t *testing.T) {
	alpha := sampleCollection("c1", "Alpha")
	beta := sampleCollection("c2", "Beta")
	colRepo := newMockCollectionRepo(alpha, beta)
	svc := buildCollectionSvc(colRepo, &mockDatasourceRepoSimple{})
	newName := "Beta" // already exists

	_, err := svc.Update("c1", &dto.UpdateCollectionRequest{Name: &newName})
	require.Error(t, err)
}

func TestCollectionService_Delete_CountDatasourcesError(t *testing.T) {
	col := sampleCollection("c1", "Alpha")
	colRepo := &mockCollectionRepoWithCountErr{mockCollectionRepo: newMockCollectionRepo(col)}
	svc := NewCollectionService(colRepo, &mockDatasourceRepoSimple{}, domain.NewService(), &mockFileServiceSimple{})

	err := svc.Delete("c1", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "count error")
}

// mockCollectionRepoWithCountErr injects an error on CountDatasources
type mockCollectionRepoWithCountErr struct {
	*mockCollectionRepo
}

func (m *mockCollectionRepoWithCountErr) CountDatasources(id string) (int64, error) {
	return 0, errors.New("count error")
}

func TestCollectionService_Delete_ForceDeleteDBError(t *testing.T) {
	col := sampleCollection("c1", "Alpha")
	colRepo := newMockCollectionRepo(col)
	colRepo.datasourceCountByID["c1"] = 3
	colRepo.deleteWithDatasourcesErr = errors.New("db error")
	svc := buildCollectionSvc(colRepo, &mockDatasourceRepoSimple{})

	err := svc.Delete("c1", true)
	require.Error(t, err)
}

func TestCollectionService_Delete_EmptyDelete_RepositoryError(t *testing.T) {
	col := sampleCollection("c1", "Alpha")
	colRepo := newMockCollectionRepo(col)
	colRepo.deleteErr = errors.New("delete failed")
	svc := buildCollectionSvc(colRepo, &mockDatasourceRepoSimple{})

	err := svc.Delete("c1", false)
	require.Error(t, err)
}
