package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"modelmatrix-server/internal/module/datasource/domain"
	"modelmatrix-server/internal/module/datasource/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock services
// ---------------------------------------------------------------------------

type mockDatasourceService struct {
	getByIDFn        func(id string) (*dto.DatasourceDetailResponse, error)
	listFn           func(collectionID *string, params *dto.ListParams) (*dto.DatasourceListResponse, error)
	deleteFn         func(id string) error
	updateFn         func(id string, req *dto.UpdateDatasourceRequest) (*dto.DatasourceResponse, error)
	createFn         func(req *dto.CreateDatasourceRequest, filename string, fileData []byte, by string) (*dto.DatasourceResponse, error)
	getDataPreviewFn func(id string, limit int) (*dto.DataPreviewResponse, error)
}

func (m *mockDatasourceService) Create(req *dto.CreateDatasourceRequest, filename string, fileData []byte, by string) (*dto.DatasourceResponse, error) {
	if m.createFn != nil {
		return m.createFn(req, filename, fileData, by)
	}
	return &dto.DatasourceResponse{}, nil
}
func (m *mockDatasourceService) Update(id string, req *dto.UpdateDatasourceRequest) (*dto.DatasourceResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(id, req)
	}
	return &dto.DatasourceResponse{ID: id}, nil
}
func (m *mockDatasourceService) Delete(id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}
func (m *mockDatasourceService) GetByID(id string) (*dto.DatasourceDetailResponse, error) {
	return m.getByIDFn(id)
}
func (m *mockDatasourceService) List(collectionID *string, params *dto.ListParams) (*dto.DatasourceListResponse, error) {
	if m.listFn != nil {
		return m.listFn(collectionID, params)
	}
	return &dto.DatasourceListResponse{}, nil
}
func (m *mockDatasourceService) CreateFromExistingFile(collectionID, name, filePath string, rowCount int, by string) (*dto.DatasourceResponse, error) {
	return nil, nil
}
func (m *mockDatasourceService) GetDataPreview(id string, limit int) (*dto.DataPreviewResponse, error) {
	if m.getDataPreviewFn != nil {
		return m.getDataPreviewFn(id, limit)
	}
	return &dto.DataPreviewResponse{}, nil
}

type mockColumnServiceForDS struct {
	getByDatasourceIDFn func(datasourceID string) ([]dto.ColumnResponse, error)
	updateRoleFn        func(datasourceID, columnID, role string) (*dto.ColumnResponse, error)
	bulkUpdateRolesFn   func(datasourceID string, req *dto.BulkUpdateColumnRolesRequest) ([]dto.ColumnResponse, error)
	createColumnsFn     func(datasourceID string, req *dto.CreateColumnsRequest) ([]dto.ColumnResponse, error)
}

func (m *mockColumnServiceForDS) GetByDatasourceID(datasourceID string) ([]dto.ColumnResponse, error) {
	if m.getByDatasourceIDFn != nil {
		return m.getByDatasourceIDFn(datasourceID)
	}
	return []dto.ColumnResponse{}, nil
}
func (m *mockColumnServiceForDS) UpdateRole(datasourceID, columnID, role string) (*dto.ColumnResponse, error) {
	if m.updateRoleFn != nil {
		return m.updateRoleFn(datasourceID, columnID, role)
	}
	return &dto.ColumnResponse{}, nil
}
func (m *mockColumnServiceForDS) BulkUpdateRoles(datasourceID string, req *dto.BulkUpdateColumnRolesRequest) ([]dto.ColumnResponse, error) {
	if m.bulkUpdateRolesFn != nil {
		return m.bulkUpdateRolesFn(datasourceID, req)
	}
	return []dto.ColumnResponse{}, nil
}
func (m *mockColumnServiceForDS) CreateColumns(datasourceID string, req *dto.CreateColumnsRequest) ([]dto.ColumnResponse, error) {
	if m.createColumnsFn != nil {
		return m.createColumnsFn(datasourceID, req)
	}
	return []dto.ColumnResponse{}, nil
}

// ---------------------------------------------------------------------------
// Router setup
// ---------------------------------------------------------------------------

func setupDatasourceRouter(dsSvc *mockDatasourceService, colSvc *mockColumnServiceForDS) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(dsAdminMiddleware())
	ctrl := NewDatasourceController(dsSvc, colSvc)
	api := r.Group("/api")
	ctrl.RegisterRoutes(api, func(c *gin.Context) { c.Next() })
	return r
}

// reuse admin middleware from existing auth_controller_test.go
func dsAdminMiddleware() gin.HandlerFunc {
	return collectionAdminMiddleware()
}

func doDsReq(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	req, _ := http.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseDsResp(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestDatasourceController_GetByID_Found(t *testing.T) {
	dsSvc := &mockDatasourceService{
		getByIDFn: func(id string) (*dto.DatasourceDetailResponse, error) {
			return &dto.DatasourceDetailResponse{
				DatasourceResponse: dto.DatasourceResponse{ID: id, Name: "Sales CSV"},
			}, nil
		},
	}
	r := setupDatasourceRouter(dsSvc, &mockColumnServiceForDS{})
	w := doDsReq(r, "GET", "/api/datasources/ds1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseDsResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "ds1", data["id"])
}

func TestDatasourceController_GetByID_NotFound(t *testing.T) {
	dsSvc := &mockDatasourceService{
		getByIDFn: func(id string) (*dto.DatasourceDetailResponse, error) {
			return nil, domain.ErrDatasourceNotFound
		},
	}
	r := setupDatasourceRouter(dsSvc, &mockColumnServiceForDS{})
	w := doDsReq(r, "GET", "/api/datasources/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestDatasourceController_List_Success(t *testing.T) {
	dsSvc := &mockDatasourceService{
		listFn: func(collectionID *string, params *dto.ListParams) (*dto.DatasourceListResponse, error) {
			return &dto.DatasourceListResponse{
				Datasources: []dto.DatasourceResponse{{ID: "ds1"}, {ID: "ds2"}},
				Total:       2,
			}, nil
		},
	}
	r := setupDatasourceRouter(dsSvc, &mockColumnServiceForDS{})
	w := doDsReq(r, "GET", "/api/datasources", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseDsResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["total"])
}

func TestDatasourceController_List_WithCollectionFilter(t *testing.T) {
	var capturedCollectionID *string
	dsSvc := &mockDatasourceService{
		listFn: func(collectionID *string, params *dto.ListParams) (*dto.DatasourceListResponse, error) {
			capturedCollectionID = collectionID
			return &dto.DatasourceListResponse{}, nil
		},
	}
	r := setupDatasourceRouter(dsSvc, &mockColumnServiceForDS{})
	req, _ := http.NewRequest("GET", "/api/datasources?collection_id=c1", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, capturedCollectionID)
	assert.Equal(t, "c1", *capturedCollectionID)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestDatasourceController_Delete_Success(t *testing.T) {
	dsSvc := &mockDatasourceService{
		deleteFn: func(id string) error { return nil },
	}
	r := setupDatasourceRouter(dsSvc, &mockColumnServiceForDS{})
	w := doDsReq(r, "DELETE", "/api/datasources/ds1", nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDatasourceController_Delete_NotFound(t *testing.T) {
	dsSvc := &mockDatasourceService{
		deleteFn: func(id string) error { return domain.ErrDatasourceNotFound },
	}
	r := setupDatasourceRouter(dsSvc, &mockColumnServiceForDS{})
	w := doDsReq(r, "DELETE", "/api/datasources/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetColumns
// ---------------------------------------------------------------------------

func TestDatasourceController_GetColumns_Success(t *testing.T) {
	colSvc := &mockColumnServiceForDS{
		getByDatasourceIDFn: func(datasourceID string) ([]dto.ColumnResponse, error) {
			return []dto.ColumnResponse{
				{ID: "col1", Name: "age", Role: "input"},
				{ID: "col2", Name: "churn", Role: "target"},
			}, nil
		},
	}
	r := setupDatasourceRouter(&mockDatasourceService{getByIDFn: func(id string) (*dto.DatasourceDetailResponse, error) { return nil, nil }}, colSvc)
	w := doDsReq(r, "GET", "/api/datasources/ds1/columns", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseDsResp(t, w)
	data := body["data"].([]interface{})
	assert.Len(t, data, 2)
}

func TestDatasourceController_GetColumns_NotFound(t *testing.T) {
	colSvc := &mockColumnServiceForDS{
		getByDatasourceIDFn: func(datasourceID string) ([]dto.ColumnResponse, error) {
			return nil, domain.ErrDatasourceNotFound
		},
	}
	r := setupDatasourceRouter(&mockDatasourceService{getByIDFn: func(id string) (*dto.DatasourceDetailResponse, error) { return nil, nil }}, colSvc)
	w := doDsReq(r, "GET", "/api/datasources/missing/columns", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// UpdateColumnRole
// ---------------------------------------------------------------------------

func TestDatasourceController_UpdateColumnRole_Success(t *testing.T) {
	colSvc := &mockColumnServiceForDS{
		updateRoleFn: func(datasourceID, columnID, role string) (*dto.ColumnResponse, error) {
			return &dto.ColumnResponse{ID: columnID, Role: role}, nil
		},
	}
	r := setupDatasourceRouter(&mockDatasourceService{getByIDFn: func(id string) (*dto.DatasourceDetailResponse, error) { return nil, nil }}, colSvc)
	w := doDsReq(r, "PUT", "/api/datasources/ds1/columns/col1/role", map[string]string{"role": "target"})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDatasourceController_UpdateColumnRole_InvalidRole(t *testing.T) {
	colSvc := &mockColumnServiceForDS{
		updateRoleFn: func(datasourceID, columnID, role string) (*dto.ColumnResponse, error) {
			return nil, domain.ErrInvalidColumnRole
		},
	}
	r := setupDatasourceRouter(&mockDatasourceService{getByIDFn: func(id string) (*dto.DatasourceDetailResponse, error) { return nil, nil }}, colSvc)
	w := doDsReq(r, "PUT", "/api/datasources/ds1/columns/col1/role", map[string]string{"role": "invalid"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// BulkUpdateColumnRoles
// ---------------------------------------------------------------------------

func TestDatasourceController_BulkUpdateColumnRoles_Success(t *testing.T) {
	colSvc := &mockColumnServiceForDS{
		bulkUpdateRolesFn: func(datasourceID string, req *dto.BulkUpdateColumnRolesRequest) ([]dto.ColumnResponse, error) {
			return []dto.ColumnResponse{
				{ID: req.Columns[0].ColumnID, Role: req.Columns[0].Role},
			}, nil
		},
	}
	r := setupDatasourceRouter(&mockDatasourceService{}, colSvc)
	payload := map[string]interface{}{
		"columns": []map[string]string{
			{"column_id": "550e8400-e29b-41d4-a716-446655440000", "role": "input"},
		},
	}
	w := doDsReq(r, "PUT", "/api/datasources/ds1/columns/roles", payload)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDatasourceController_BulkUpdateColumnRoles_InvalidJSON(t *testing.T) {
	r := setupDatasourceRouter(&mockDatasourceService{}, &mockColumnServiceForDS{})
	w := doDsReq(r, "PUT", "/api/datasources/ds1/columns/roles", "bad-payload")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Update (DatasourceController)
// ---------------------------------------------------------------------------

func TestDatasourceController_Update_Success(t *testing.T) {
	name := "Renamed DS"
	dsSvc := &mockDatasourceService{
		getByIDFn: func(id string) (*dto.DatasourceDetailResponse, error) { return nil, nil },
		updateFn: func(id string, req *dto.UpdateDatasourceRequest) (*dto.DatasourceResponse, error) {
			return &dto.DatasourceResponse{ID: id, Name: *req.Name}, nil
		},
	}
	r := setupDatasourceRouter(dsSvc, &mockColumnServiceForDS{})
	w := doDsReq(r, "PUT", "/api/datasources/ds1", map[string]interface{}{"name": name})
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseDsResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, name, data["name"])
}

func TestDatasourceController_Update_NotFound(t *testing.T) {
	dsSvc := &mockDatasourceService{
		getByIDFn: func(id string) (*dto.DatasourceDetailResponse, error) { return nil, nil },
		updateFn: func(id string, req *dto.UpdateDatasourceRequest) (*dto.DatasourceResponse, error) {
			return nil, domain.ErrDatasourceNotFound
		},
	}
	r := setupDatasourceRouter(dsSvc, &mockColumnServiceForDS{})
	w := doDsReq(r, "PUT", "/api/datasources/missing", map[string]interface{}{"name": "X"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDatasourceController_Update_InvalidJSON(t *testing.T) {
	dsSvc := &mockDatasourceService{
		getByIDFn: func(id string) (*dto.DatasourceDetailResponse, error) { return nil, nil },
	}
	r := setupDatasourceRouter(dsSvc, &mockColumnServiceForDS{})
	w := doDsReq(r, "PUT", "/api/datasources/ds1", "bad")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// GetDataPreview
// ---------------------------------------------------------------------------

func TestDatasourceController_GetDataPreview_Success(t *testing.T) {
	dsSvc := &mockDatasourceService{
		getByIDFn: func(id string) (*dto.DatasourceDetailResponse, error) { return nil, nil },
		getDataPreviewFn: func(id string, limit int) (*dto.DataPreviewResponse, error) {
			return &dto.DataPreviewResponse{
				Columns:    []string{"age", "income"},
				TotalRows:  200,
				PreviewMax: limit,
			}, nil
		},
	}
	r := setupDatasourceRouter(dsSvc, &mockColumnServiceForDS{})
	w := doDsReq(r, "GET", "/api/datasources/ds1/preview?limit=50", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDatasourceController_GetDataPreview_NotFound(t *testing.T) {
	dsSvc := &mockDatasourceService{
		getByIDFn: func(id string) (*dto.DatasourceDetailResponse, error) { return nil, nil },
		getDataPreviewFn: func(id string, limit int) (*dto.DataPreviewResponse, error) {
			return nil, domain.ErrDatasourceNotFound
		},
	}
	r := setupDatasourceRouter(dsSvc, &mockColumnServiceForDS{})
	w := doDsReq(r, "GET", "/api/datasources/missing/preview", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// CreateColumns
// ---------------------------------------------------------------------------

func TestDatasourceController_CreateColumns_Success(t *testing.T) {
	colSvc := &mockColumnServiceForDS{
		createColumnsFn: func(datasourceID string, req *dto.CreateColumnsRequest) ([]dto.ColumnResponse, error) {
			return []dto.ColumnResponse{{ID: "c1", Name: req.Columns[0].Name}}, nil
		},
	}
	r := setupDatasourceRouter(&mockDatasourceService{getByIDFn: func(id string) (*dto.DatasourceDetailResponse, error) { return nil, nil }}, colSvc)
	payload := map[string]interface{}{
		"columns": []map[string]string{
			{"name": "revenue", "data_type": "float64", "role": "input"},
		},
	}
	w := doDsReq(r, "POST", "/api/datasources/ds1/columns", payload)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestDatasourceController_CreateColumns_InvalidJSON(t *testing.T) {
	r := setupDatasourceRouter(&mockDatasourceService{getByIDFn: func(id string) (*dto.DatasourceDetailResponse, error) { return nil, nil }}, &mockColumnServiceForDS{})
	w := doDsReq(r, "POST", "/api/datasources/ds1/columns", "bad")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
