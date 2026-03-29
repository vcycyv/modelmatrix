package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/internal/infrastructure/ldap"
	"modelmatrix-server/internal/module/datasource/domain"
	"modelmatrix-server/internal/module/datasource/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock CollectionService
// ---------------------------------------------------------------------------

type mockCollectionService struct {
	createFn  func(req *dto.CreateCollectionRequest, by string) (*dto.CollectionResponse, error)
	updateFn  func(id string, req *dto.UpdateCollectionRequest) (*dto.CollectionResponse, error)
	deleteFn  func(id string, force bool) error
	getByIDFn func(id string) (*dto.CollectionResponse, error)
	listFn    func(params *dto.ListParams) (*dto.CollectionListResponse, error)
}

func (m *mockCollectionService) Create(req *dto.CreateCollectionRequest, by string) (*dto.CollectionResponse, error) {
	return m.createFn(req, by)
}
func (m *mockCollectionService) Update(id string, req *dto.UpdateCollectionRequest) (*dto.CollectionResponse, error) {
	return m.updateFn(id, req)
}
func (m *mockCollectionService) Delete(id string, force bool) error { return m.deleteFn(id, force) }
func (m *mockCollectionService) GetByID(id string) (*dto.CollectionResponse, error) {
	return m.getByIDFn(id)
}
func (m *mockCollectionService) List(params *dto.ListParams) (*dto.CollectionListResponse, error) {
	if m.listFn != nil {
		return m.listFn(params)
	}
	return &dto.CollectionListResponse{}, nil
}

// ---------------------------------------------------------------------------
// Router helpers
// ---------------------------------------------------------------------------

func collectionAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(auth.ContextKeyUser, &auth.Claims{
			Username: "admin",
			Groups:   []string{ldap.GroupAdmin},
		})
		c.Next()
	}
}

func setupCollectionRouter(svc *mockCollectionService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(collectionAdminMiddleware())
	ctrl := NewCollectionController(svc)
	api := r.Group("/api")
	ctrl.RegisterRoutes(api, func(c *gin.Context) { c.Next() })
	return r
}

func doReq(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
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

func parseResp(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	return result
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCollectionController_Create_Success(t *testing.T) {
	svc := &mockCollectionService{
		createFn: func(req *dto.CreateCollectionRequest, by string) (*dto.CollectionResponse, error) {
			return &dto.CollectionResponse{ID: "c1", Name: req.Name, CreatedBy: by}, nil
		},
	}
	r := setupCollectionRouter(svc)

	w := doReq(r, "POST", "/api/collections", map[string]string{"name": "New Collection"})
	assert.Equal(t, http.StatusCreated, w.Code)
	body := parseResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "c1", data["id"])
}

func TestCollectionController_Create_InvalidJSON(t *testing.T) {
	r := setupCollectionRouter(&mockCollectionService{})
	req, _ := http.NewRequest("POST", "/api/collections", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCollectionController_Create_DuplicateName(t *testing.T) {
	svc := &mockCollectionService{
		createFn: func(req *dto.CreateCollectionRequest, by string) (*dto.CollectionResponse, error) {
			return nil, domain.ErrCollectionNameExists
		},
	}
	r := setupCollectionRouter(svc)
	w := doReq(r, "POST", "/api/collections", map[string]string{"name": "Dup"})
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCollectionController_Create_EmptyName_BadRequest(t *testing.T) {
	svc := &mockCollectionService{
		createFn: func(req *dto.CreateCollectionRequest, by string) (*dto.CollectionResponse, error) {
			return nil, domain.ErrCollectionNameEmpty
		},
	}
	r := setupCollectionRouter(svc)
	w := doReq(r, "POST", "/api/collections", map[string]string{"name": ""})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestCollectionController_GetByID_Found(t *testing.T) {
	svc := &mockCollectionService{
		getByIDFn: func(id string) (*dto.CollectionResponse, error) {
			return &dto.CollectionResponse{ID: id, Name: "Found"}, nil
		},
	}
	r := setupCollectionRouter(svc)
	w := doReq(r, "GET", "/api/collections/c1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "c1", data["id"])
}

func TestCollectionController_GetByID_NotFound(t *testing.T) {
	svc := &mockCollectionService{
		getByIDFn: func(id string) (*dto.CollectionResponse, error) {
			return nil, domain.ErrCollectionNotFound
		},
	}
	r := setupCollectionRouter(svc)
	w := doReq(r, "GET", "/api/collections/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestCollectionController_Update_Success(t *testing.T) {
	newName := "Updated"
	svc := &mockCollectionService{
		updateFn: func(id string, req *dto.UpdateCollectionRequest) (*dto.CollectionResponse, error) {
			return &dto.CollectionResponse{ID: id, Name: *req.Name}, nil
		},
	}
	r := setupCollectionRouter(svc)
	w := doReq(r, "PUT", "/api/collections/c1", map[string]string{"name": newName})
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "Updated", data["name"])
}

func TestCollectionController_Update_NotFound(t *testing.T) {
	svc := &mockCollectionService{
		updateFn: func(id string, req *dto.UpdateCollectionRequest) (*dto.CollectionResponse, error) {
			return nil, domain.ErrCollectionNotFound
		},
	}
	r := setupCollectionRouter(svc)
	w := doReq(r, "PUT", "/api/collections/missing", map[string]string{"name": "X"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestCollectionController_Delete_Success(t *testing.T) {
	svc := &mockCollectionService{
		deleteFn: func(id string, force bool) error { return nil },
	}
	r := setupCollectionRouter(svc)
	w := doReq(r, "DELETE", "/api/collections/c1", nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCollectionController_Delete_HasDatasources_Conflict(t *testing.T) {
	svc := &mockCollectionService{
		deleteFn: func(id string, force bool) error { return domain.ErrCollectionHasDatasources },
	}
	r := setupCollectionRouter(svc)
	w := doReq(r, "DELETE", "/api/collections/c1", nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCollectionController_Delete_ForceQueryParam(t *testing.T) {
	var capturedForce bool
	svc := &mockCollectionService{
		deleteFn: func(id string, force bool) error {
			capturedForce = force
			return nil
		},
	}
	r := setupCollectionRouter(svc)
	req, _ := http.NewRequest("DELETE", "/api/collections/c1?force=true", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, capturedForce)
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestCollectionController_List_Success(t *testing.T) {
	svc := &mockCollectionService{
		listFn: func(params *dto.ListParams) (*dto.CollectionListResponse, error) {
			return &dto.CollectionListResponse{
				Collections: []dto.CollectionResponse{{ID: "c1"}, {ID: "c2"}},
				Total:       2,
			}, nil
		},
	}
	r := setupCollectionRouter(svc)
	w := doReq(r, "GET", "/api/collections", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["total"])
}
