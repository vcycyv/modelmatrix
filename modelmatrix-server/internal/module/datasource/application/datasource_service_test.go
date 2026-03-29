package application

import (
	"errors"
	"testing"

	"modelmatrix-server/internal/module/datasource/domain"
	"modelmatrix-server/internal/module/datasource/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// fakeColumnRepo — satisfies repository.ColumnRepository
// ---------------------------------------------------------------------------

type fakeColumnRepo struct {
	columns       map[string]*domain.Column     // id -> column
	byDatasource  map[string][]domain.Column    // datasourceID -> columns
	updateRoleErr error
}

func newFakeColumnRepo(byDS map[string][]domain.Column) *fakeColumnRepo {
	r := &fakeColumnRepo{
		columns:      make(map[string]*domain.Column),
		byDatasource: byDS,
	}
	if byDS == nil {
		r.byDatasource = make(map[string][]domain.Column)
	}
	// populate columns map
	for _, cols := range r.byDatasource {
		for i := range cols {
			r.columns[cols[i].ID] = &cols[i]
		}
	}
	return r
}

func (r *fakeColumnRepo) Create(col *domain.Column) error        { return nil }
func (r *fakeColumnRepo) CreateBatch(cols []domain.Column) error { return nil }
func (r *fakeColumnRepo) Update(col *domain.Column) error        { return nil }
func (r *fakeColumnRepo) Delete(id string) error                  { return nil }
func (r *fakeColumnRepo) DeleteByDatasourceID(datasourceID string) error { return nil }
func (r *fakeColumnRepo) GetByDatasourceID(datasourceID string) ([]domain.Column, error) {
	return r.byDatasource[datasourceID], nil
}
func (r *fakeColumnRepo) GetByID(id string) (*domain.Column, error) {
	if col, ok := r.columns[id]; ok {
		return col, nil
	}
	return nil, domain.ErrColumnNotFound
}
func (r *fakeColumnRepo) UpdateRole(id string, role domain.ColumnRole) error {
	if r.updateRoleErr != nil {
		return r.updateRoleErr
	}
	if col, ok := r.columns[id]; ok {
		col.Role = role
	}
	return nil
}

// ---------------------------------------------------------------------------
// fakeDSRepo — satisfies repository.DatasourceRepository for datasource service
// ---------------------------------------------------------------------------

type fakeDSRepo struct {
	datasources map[string]*domain.Datasource
	names       map[string][]string // collectionID -> names
}

func newFakeDSRepo(datasources ...*domain.Datasource) *fakeDSRepo {
	r := &fakeDSRepo{
		datasources: make(map[string]*domain.Datasource),
		names:       make(map[string][]string),
	}
	for _, ds := range datasources {
		r.datasources[ds.ID] = ds
		r.names[ds.CollectionID] = append(r.names[ds.CollectionID], ds.Name)
	}
	return r
}

func (r *fakeDSRepo) Create(ds *domain.Datasource) error           { return nil }
func (r *fakeDSRepo) Update(ds *domain.Datasource) error           { return nil }
func (r *fakeDSRepo) Delete(id string) error                        { delete(r.datasources, id); return nil }
func (r *fakeDSRepo) UpdateFilePath(id, fp string) error            { return nil }
func (r *fakeDSRepo) GetNamesInCollection(colID string) ([]string, error) {
	return r.names[colID], nil
}
func (r *fakeDSRepo) GetByID(id string) (*domain.Datasource, error) {
	if ds, ok := r.datasources[id]; ok {
		return ds, nil
	}
	return nil, domain.ErrDatasourceNotFound
}
func (r *fakeDSRepo) GetByIDWithColumns(id string) (*domain.Datasource, error) {
	return r.GetByID(id)
}
func (r *fakeDSRepo) GetByName(colID, name string) (*domain.Datasource, error) { return nil, nil }
func (r *fakeDSRepo) List(collectionID *string, offset, limit int, search string) ([]domain.Datasource, int64, error) {
	var result []domain.Datasource
	for _, ds := range r.datasources {
		if collectionID != nil && ds.CollectionID != *collectionID {
			continue
		}
		result = append(result, *ds)
	}
	return result, int64(len(result)), nil
}
func (r *fakeDSRepo) ListByCollection(collectionID string, offset, limit int) ([]domain.Datasource, int64, error) {
	return nil, 0, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func buildDSSvc(dsRepo *fakeDSRepo, colRepo *mockCollectionRepo, colRepo2 *fakeColumnRepo) DatasourceService {
	return NewDatasourceService(
		nil, // *gorm.DB — not used by GetByID/List/Update
		dsRepo,
		colRepo,
		colRepo2,
		domain.NewService(),
		&mockFileServiceSimple{},
		nil, // ExternalDBConnector — not used by tested methods
	)
}

func sampleDS(id, colID, name string) *domain.Datasource {
	return &domain.Datasource{ID: id, CollectionID: colID, Name: name, Type: domain.DatasourceTypeCSV}
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestDatasourceService_GetByID_Found(t *testing.T) {
	colRepo := newMockCollectionRepo(sampleCollection("c1", "Sales"))
	dsRepo := newFakeDSRepo(sampleDS("ds1", "c1", "Sales CSV"))
	svc := buildDSSvc(dsRepo, colRepo, newFakeColumnRepo(nil))

	resp, err := svc.GetByID("ds1")
	require.NoError(t, err)
	assert.Equal(t, "ds1", resp.ID)
	assert.Equal(t, "Sales CSV", resp.Name)
	assert.Equal(t, "Sales", resp.CollectionName)
}

func TestDatasourceService_GetByID_NotFound(t *testing.T) {
	colRepo := newMockCollectionRepo()
	svc := buildDSSvc(newFakeDSRepo(), colRepo, newFakeColumnRepo(nil))

	_, err := svc.GetByID("missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrDatasourceNotFound))
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestDatasourceService_List_All(t *testing.T) {
	colRepo := newMockCollectionRepo(
		sampleCollection("c1", "A"),
		sampleCollection("c2", "B"),
	)
	dsRepo := newFakeDSRepo(
		sampleDS("ds1", "c1", "D1"),
		sampleDS("ds2", "c1", "D2"),
		sampleDS("ds3", "c2", "D3"),
	)
	svc := buildDSSvc(dsRepo, colRepo, newFakeColumnRepo(nil))

	resp, err := svc.List(nil, &dto.ListParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(3), resp.Total)
}

func TestDatasourceService_List_FilterByCollection(t *testing.T) {
	colRepo := newMockCollectionRepo(sampleCollection("c1", "A"))
	dsRepo := newFakeDSRepo(
		sampleDS("ds1", "c1", "D1"),
		sampleDS("ds2", "c1", "D2"),
		sampleDS("ds3", "c2", "D3"),
	)
	svc := buildDSSvc(dsRepo, colRepo, newFakeColumnRepo(nil))

	colID := "c1"
	resp, err := svc.List(&colID, &dto.ListParams{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.Total)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestDatasourceService_Update_Name(t *testing.T) {
	colRepo := newMockCollectionRepo(sampleCollection("c1", "MyCol"))
	dsRepo := newFakeDSRepo(sampleDS("ds1", "c1", "Old Name"))
	svc := buildDSSvc(dsRepo, colRepo, newFakeColumnRepo(nil))

	newName := "New Name"
	resp, err := svc.Update("ds1", &dto.UpdateDatasourceRequest{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "New Name", resp.Name)
}

func TestDatasourceService_Update_DuplicateName(t *testing.T) {
	colRepo := newMockCollectionRepo(sampleCollection("c1", "MyCol"))
	dsRepo := newFakeDSRepo(
		sampleDS("ds1", "c1", "A"),
		sampleDS("ds2", "c1", "B"),
	)
	svc := buildDSSvc(dsRepo, colRepo, newFakeColumnRepo(nil))

	newName := "B"
	_, err := svc.Update("ds1", &dto.UpdateDatasourceRequest{Name: &newName})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrDatasourceNameExists))
}

func TestDatasourceService_Update_NotFound(t *testing.T) {
	svc := buildDSSvc(newFakeDSRepo(), newMockCollectionRepo(), newFakeColumnRepo(nil))
	name := "X"
	_, err := svc.Update("missing", &dto.UpdateDatasourceRequest{Name: &name})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrDatasourceNotFound))
}

// ---------------------------------------------------------------------------
// ColumnService: GetByDatasourceID
// ---------------------------------------------------------------------------

func buildColSvc(dsRepo *fakeDSRepo, colRepo *fakeColumnRepo) ColumnService {
	return NewColumnService(nil, colRepo, dsRepo, domain.NewService())
}

func TestColumnService_GetByDatasourceID_Found(t *testing.T) {
	ds := sampleDS("ds1", "c1", "CSV")
	colRepo := newFakeColumnRepo(map[string][]domain.Column{
		"ds1": {
			{ID: "col1", DatasourceID: "ds1", Name: "age", Role: domain.ColumnRoleInput},
			{ID: "col2", DatasourceID: "ds1", Name: "churn", Role: domain.ColumnRoleTarget},
		},
	})
	svc := buildColSvc(newFakeDSRepo(ds), colRepo)

	cols, err := svc.GetByDatasourceID("ds1")
	require.NoError(t, err)
	assert.Len(t, cols, 2)
}

func TestColumnService_GetByDatasourceID_DatasourceNotFound(t *testing.T) {
	svc := buildColSvc(newFakeDSRepo(), newFakeColumnRepo(nil))
	_, err := svc.GetByDatasourceID("missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrDatasourceNotFound))
}

// ---------------------------------------------------------------------------
// ColumnService: UpdateRole
// ---------------------------------------------------------------------------

func TestColumnService_UpdateRole_InputToTarget(t *testing.T) {
	ds := &domain.Datasource{
		ID: "ds1", CollectionID: "c1", Type: domain.DatasourceTypeCSV,
		Columns: []domain.Column{
			{ID: "col1", DatasourceID: "ds1", Name: "age", Role: domain.ColumnRoleInput},
			{ID: "col2", DatasourceID: "ds1", Name: "churn", Role: domain.ColumnRoleIgnore},
		},
	}
	colRepo := newFakeColumnRepo(map[string][]domain.Column{
		"ds1": ds.Columns,
	})
	dsRepo := &fakeDSRepo{
		datasources: map[string]*domain.Datasource{"ds1": ds},
		names:       make(map[string][]string),
	}
	svc := buildColSvc(dsRepo, colRepo)

	resp, err := svc.UpdateRole("ds1", "col2", "target")
	require.NoError(t, err)
	assert.Equal(t, "target", resp.Role)
}

func TestColumnService_UpdateRole_InvalidRole(t *testing.T) {
	ds := sampleDS("ds1", "c1", "CSV")
	col := domain.Column{ID: "col1", DatasourceID: "ds1", Name: "x", Role: domain.ColumnRoleInput}
	colRepo := newFakeColumnRepo(map[string][]domain.Column{"ds1": {col}})
	dsRepo := &fakeDSRepo{datasources: map[string]*domain.Datasource{"ds1": ds}, names: make(map[string][]string)}
	svc := buildColSvc(dsRepo, colRepo)

	_, err := svc.UpdateRole("ds1", "col1", "invalid-role")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidColumnRole))
}

// ---------------------------------------------------------------------------
// ColumnService: UpdateRole additional error paths
// ---------------------------------------------------------------------------

func TestColumnService_UpdateRole_DatasourceNotFound(t *testing.T) {
	svc := buildColSvc(newFakeDSRepo(), newFakeColumnRepo(nil))
	_, err := svc.UpdateRole("missing", "col1", "target")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrDatasourceNotFound))
}

func TestColumnService_UpdateRole_ColumnNotFound(t *testing.T) {
	ds := sampleDS("ds1", "c1", "CSV")
	svc := buildColSvc(newFakeDSRepo(ds), newFakeColumnRepo(nil))
	_, err := svc.UpdateRole("ds1", "no-such-col", "input")
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrColumnNotFound))
}

func TestColumnService_UpdateRole_ColumnWrongDatasource(t *testing.T) {
	// col1 belongs to ds2, not ds1
	col := domain.Column{ID: "col1", DatasourceID: "ds2", Name: "x", Role: domain.ColumnRoleInput}
	ds1 := &domain.Datasource{ID: "ds1", CollectionID: "c1", Type: domain.DatasourceTypeCSV, Columns: []domain.Column{}}
	ds2 := sampleDS("ds2", "c1", "Other")
	colRepo := newFakeColumnRepo(map[string][]domain.Column{"ds2": {col}})
	dsRepo := &fakeDSRepo{
		datasources: map[string]*domain.Datasource{"ds1": ds1, "ds2": ds2},
		names:       make(map[string][]string),
	}
	svc := buildColSvc(dsRepo, colRepo)
	_, err := svc.UpdateRole("ds1", "col1", "input")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong")
}

// ---------------------------------------------------------------------------
// ColumnService: BulkUpdateRoles — validation paths (pre-transaction)
// ---------------------------------------------------------------------------

func TestColumnService_BulkUpdateRoles_DatasourceNotFound(t *testing.T) {
	svc := buildColSvc(newFakeDSRepo(), newFakeColumnRepo(nil))
	req := &dto.BulkUpdateColumnRolesRequest{
		Columns: []dto.ColumnRoleUpdate{{ColumnID: "col1", Role: "input"}},
	}
	_, err := svc.BulkUpdateRoles("missing", req)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrDatasourceNotFound))
}

func TestColumnService_BulkUpdateRoles_ColumnNotInDatasource(t *testing.T) {
	// Datasource exists but has no columns matching the requested column ID
	ds := &domain.Datasource{
		ID: "ds1", CollectionID: "c1", Type: domain.DatasourceTypeCSV,
		Columns: []domain.Column{
			{ID: "col1", DatasourceID: "ds1", Name: "age", Role: domain.ColumnRoleInput},
		},
	}
	dsRepo := &fakeDSRepo{datasources: map[string]*domain.Datasource{"ds1": ds}, names: make(map[string][]string)}
	svc := buildColSvc(dsRepo, newFakeColumnRepo(nil))
	req := &dto.BulkUpdateColumnRolesRequest{
		Columns: []dto.ColumnRoleUpdate{{ColumnID: "no-such-col", Role: "input"}},
	}
	_, err := svc.BulkUpdateRoles("ds1", req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in datasource")
}

func TestColumnService_BulkUpdateRoles_InvalidRole(t *testing.T) {
	ds := &domain.Datasource{
		ID: "ds1", CollectionID: "c1", Type: domain.DatasourceTypeCSV,
		Columns: []domain.Column{
			{ID: "col1", DatasourceID: "ds1", Name: "age", Role: domain.ColumnRoleInput},
		},
	}
	dsRepo := &fakeDSRepo{datasources: map[string]*domain.Datasource{"ds1": ds}, names: make(map[string][]string)}
	svc := buildColSvc(dsRepo, newFakeColumnRepo(nil))
	req := &dto.BulkUpdateColumnRolesRequest{
		Columns: []dto.ColumnRoleUpdate{{ColumnID: "col1", Role: "invalid-role"}},
	}
	_, err := svc.BulkUpdateRoles("ds1", req)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidColumnRole))
}

func TestColumnService_BulkUpdateRoles_MultipleTargets_ValidationFails(t *testing.T) {
	// Two columns both set to "target" → domain validation should reject
	ds := &domain.Datasource{
		ID: "ds1", CollectionID: "c1", Type: domain.DatasourceTypeCSV,
		Columns: []domain.Column{
			{ID: "col1", DatasourceID: "ds1", Name: "a", Role: domain.ColumnRoleInput},
			{ID: "col2", DatasourceID: "ds1", Name: "b", Role: domain.ColumnRoleInput},
		},
	}
	dsRepo := &fakeDSRepo{datasources: map[string]*domain.Datasource{"ds1": ds}, names: make(map[string][]string)}
	svc := buildColSvc(dsRepo, newFakeColumnRepo(nil))
	req := &dto.BulkUpdateColumnRolesRequest{
		Columns: []dto.ColumnRoleUpdate{
			{ColumnID: "col1", Role: "target"},
			{ColumnID: "col2", Role: "target"},
		},
	}
	_, err := svc.BulkUpdateRoles("ds1", req)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// ColumnService.CreateColumns
// ---------------------------------------------------------------------------

func TestColumnService_CreateColumns_Success(t *testing.T) {
	ds := &domain.Datasource{
		ID: "ds1", CollectionID: "c1", Type: domain.DatasourceTypeCSV,
		Columns: []domain.Column{},
	}
	dsRepo := &fakeDSRepo{datasources: map[string]*domain.Datasource{"ds1": ds}, names: make(map[string][]string)}
	svc := buildColSvc(dsRepo, newFakeColumnRepo(nil))
	req := &dto.CreateColumnsRequest{
		Columns: []dto.CreateColumnRequest{
			{Name: "age", DataType: "int64", Role: "input"},
			{Name: "churn", DataType: "bool", Role: "target"},
		},
	}
	result, err := svc.CreateColumns("ds1", req)
	require.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestColumnService_CreateColumns_DatasourceNotFound(t *testing.T) {
	dsRepo := &fakeDSRepo{datasources: make(map[string]*domain.Datasource), names: make(map[string][]string)}
	svc := buildColSvc(dsRepo, newFakeColumnRepo(nil))
	req := &dto.CreateColumnsRequest{
		Columns: []dto.CreateColumnRequest{{Name: "age", DataType: "int64", Role: "input"}},
	}
	_, err := svc.CreateColumns("missing", req)
	require.Error(t, err)
}

func TestColumnService_CreateColumns_InvalidRole(t *testing.T) {
	ds := &domain.Datasource{
		ID: "ds1", CollectionID: "c1", Type: domain.DatasourceTypeCSV,
		Columns: []domain.Column{},
	}
	dsRepo := &fakeDSRepo{datasources: map[string]*domain.Datasource{"ds1": ds}, names: make(map[string][]string)}
	svc := buildColSvc(dsRepo, newFakeColumnRepo(nil))
	req := &dto.CreateColumnsRequest{
		Columns: []dto.CreateColumnRequest{{Name: "age", DataType: "int64", Role: "bad-role"}},
	}
	_, err := svc.CreateColumns("ds1", req)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidColumnRole))
}

func TestColumnService_CreateColumns_DuplicateName(t *testing.T) {
	ds := &domain.Datasource{
		ID: "ds1", CollectionID: "c1", Type: domain.DatasourceTypeCSV,
		Columns: []domain.Column{
			{ID: "c1", DatasourceID: "ds1", Name: "age", Role: domain.ColumnRoleInput},
		},
	}
	dsRepo := &fakeDSRepo{datasources: map[string]*domain.Datasource{"ds1": ds}, names: make(map[string][]string)}
	svc := buildColSvc(dsRepo, newFakeColumnRepo(nil))
	req := &dto.CreateColumnsRequest{
		Columns: []dto.CreateColumnRequest{{Name: "age", DataType: "int64", Role: "input"}},
	}
	_, err := svc.CreateColumns("ds1", req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestColumnService_CreateColumns_MultipleTargetsRejected(t *testing.T) {
	ds := &domain.Datasource{
		ID: "ds1", CollectionID: "c1", Type: domain.DatasourceTypeCSV,
		Columns: []domain.Column{},
	}
	dsRepo := &fakeDSRepo{datasources: map[string]*domain.Datasource{"ds1": ds}, names: make(map[string][]string)}
	svc := buildColSvc(dsRepo, newFakeColumnRepo(nil))
	req := &dto.CreateColumnsRequest{
		Columns: []dto.CreateColumnRequest{
			{Name: "a", DataType: "int64", Role: "target"},
			{Name: "b", DataType: "int64", Role: "target"},
		},
	}
	_, err := svc.CreateColumns("ds1", req)
	require.Error(t, err)
}
