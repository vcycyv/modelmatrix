package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/internal/infrastructure/ldap"
	"modelmatrix-server/internal/module/build/domain"
	"modelmatrix-server/internal/module/build/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock BuildService
// ---------------------------------------------------------------------------

type mockBuildService struct {
	createFn   func(req *dto.CreateBuildRequest, createdBy string) (*dto.BuildResponse, error)
	getByIDFn  func(id string) (*dto.BuildResponse, error)
	listFn     func(params *dto.ListParams) (*dto.BuildListResponse, error)
	startFn    func(id string) (*dto.BuildResponse, error)
	cancelFn   func(id string) (*dto.BuildResponse, error)
	callbackFn func(req *dto.BuildCallbackRequest) error
	deleteFn   func(id string) error
	updateFn   func(id string, req *dto.UpdateBuildRequest) (*dto.BuildResponse, error)
}

func (m *mockBuildService) Create(req *dto.CreateBuildRequest, by string) (*dto.BuildResponse, error) {
	return m.createFn(req, by)
}
func (m *mockBuildService) GetByID(id string) (*dto.BuildResponse, error) {
	return m.getByIDFn(id)
}
func (m *mockBuildService) List(p *dto.ListParams) (*dto.BuildListResponse, error) {
	if m.listFn != nil {
		return m.listFn(p)
	}
	return &dto.BuildListResponse{}, nil
}
func (m *mockBuildService) Start(id string) (*dto.BuildResponse, error)   { return m.startFn(id) }
func (m *mockBuildService) Cancel(id string) (*dto.BuildResponse, error)  { return m.cancelFn(id) }
func (m *mockBuildService) HandleCallback(req *dto.BuildCallbackRequest) error {
	return m.callbackFn(req)
}
func (m *mockBuildService) Delete(id string) error { return m.deleteFn(id) }
func (m *mockBuildService) Update(id string, req *dto.UpdateBuildRequest) (*dto.BuildResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(id, req)
	}
	return &dto.BuildResponse{ID: id}, nil
}
func (m *mockBuildService) Retrain(modelID string, req *dto.RetrainRequest, by string) (*dto.BuildResponse, error) {
	return nil, nil
}
func (m *mockBuildService) DeleteByFolderID(id string) error  { return nil }
func (m *mockBuildService) DeleteByProjectID(id string) error { return nil }

// ---------------------------------------------------------------------------
// Test router helpers
// ---------------------------------------------------------------------------

// adminClaims injects admin claims so RBAC middlewares are satisfied.
func adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(auth.ContextKeyUser, &auth.Claims{
			Username: "admin",
			Groups:   []string{ldap.GroupAdmin},
		})
		c.Next()
	}
}

func setupBuildRouter(svc *mockBuildService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(adminMiddleware())
	ctrl := NewBuildController(svc)
	api := r.Group("/api")
	ctrl.RegisterRoutes(api, func(c *gin.Context) { c.Next() })
	return r
}

func doRequest(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
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

func parseBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	return result
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestBuildController_Create_Success(t *testing.T) {
	svc := &mockBuildService{
		createFn: func(req *dto.CreateBuildRequest, by string) (*dto.BuildResponse, error) {
			return &dto.BuildResponse{ID: "b1", Name: req.Name, Status: "pending"}, nil
		},
	}
	r := setupBuildRouter(svc)

	w := doRequest(r, "POST", "/api/builds", map[string]interface{}{
		"name":          "My Build",
		"datasource_id": "550e8400-e29b-41d4-a716-446655440001",
		"model_type":    "regression",
		"algorithm":     "random_forest",
	})

	assert.Equal(t, http.StatusCreated, w.Code)
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "b1", data["id"])
}

func TestBuildController_Create_InvalidJSON(t *testing.T) {
	svc := &mockBuildService{}
	r := setupBuildRouter(svc)

	req, _ := http.NewRequest("POST", "/api/builds", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBuildController_Create_ServiceError_Conflict(t *testing.T) {
	svc := &mockBuildService{
		createFn: func(req *dto.CreateBuildRequest, by string) (*dto.BuildResponse, error) {
			return nil, domain.ErrBuildNameExists
		},
	}
	r := setupBuildRouter(svc)

	w := doRequest(r, "POST", "/api/builds", map[string]interface{}{
		"name":          "Dupe",
		"datasource_id": "550e8400-e29b-41d4-a716-446655440001",
		"model_type":    "regression",
		"algorithm":     "random_forest",
	})

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestBuildController_GetByID_Found(t *testing.T) {
	svc := &mockBuildService{
		getByIDFn: func(id string) (*dto.BuildResponse, error) {
			return &dto.BuildResponse{ID: id, Name: "Found"}, nil
		},
	}
	r := setupBuildRouter(svc)

	w := doRequest(r, "GET", "/api/builds/b1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "b1", data["id"])
}

func TestBuildController_GetByID_NotFound(t *testing.T) {
	svc := &mockBuildService{
		getByIDFn: func(id string) (*dto.BuildResponse, error) {
			return nil, domain.ErrBuildNotFound
		},
	}
	r := setupBuildRouter(svc)

	w := doRequest(r, "GET", "/api/builds/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestBuildController_List_Success(t *testing.T) {
	svc := &mockBuildService{
		listFn: func(p *dto.ListParams) (*dto.BuildListResponse, error) {
			return &dto.BuildListResponse{
				Builds: []dto.BuildResponse{{ID: "b1"}, {ID: "b2"}},
				Total:  2,
			}, nil
		},
	}
	r := setupBuildRouter(svc)

	w := doRequest(r, "GET", "/api/builds", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["total"])
}

// ---------------------------------------------------------------------------
// Start
// ---------------------------------------------------------------------------

func TestBuildController_Start_Success(t *testing.T) {
	svc := &mockBuildService{
		startFn: func(id string) (*dto.BuildResponse, error) {
			return &dto.BuildResponse{ID: id, Status: "running"}, nil
		},
	}
	r := setupBuildRouter(svc)

	w := doRequest(r, "POST", "/api/builds/b1/start", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "running", data["status"])
}

func TestBuildController_Start_BuildNotFound(t *testing.T) {
	svc := &mockBuildService{
		startFn: func(id string) (*dto.BuildResponse, error) {
			return nil, domain.ErrBuildNotFound
		},
	}
	r := setupBuildRouter(svc)

	w := doRequest(r, "POST", "/api/builds/missing/start", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestBuildController_Start_DomainError_BadRequest(t *testing.T) {
	svc := &mockBuildService{
		startFn: func(id string) (*dto.BuildResponse, error) {
			return nil, domain.ErrBuildAlreadyRunning
		},
	}
	r := setupBuildRouter(svc)

	w := doRequest(r, "POST", "/api/builds/b1/start", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Cancel
// ---------------------------------------------------------------------------

func TestBuildController_Cancel_Success(t *testing.T) {
	svc := &mockBuildService{
		cancelFn: func(id string) (*dto.BuildResponse, error) {
			return &dto.BuildResponse{ID: id, Status: "cancelled"}, nil
		},
	}
	r := setupBuildRouter(svc)

	w := doRequest(r, "POST", "/api/builds/b1/cancel", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBuildController_Cancel_CannotCancel(t *testing.T) {
	svc := &mockBuildService{
		cancelFn: func(id string) (*dto.BuildResponse, error) {
			return nil, domain.ErrBuildCannotBeCancelled
		},
	}
	r := setupBuildRouter(svc)

	w := doRequest(r, "POST", "/api/builds/b1/cancel", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Callback (no auth required)
// ---------------------------------------------------------------------------

const validCallbackBuildID = "550e8400-e29b-41d4-a716-446655440001"

func TestBuildController_Callback_Success(t *testing.T) {
	svc := &mockBuildService{
		callbackFn: func(req *dto.BuildCallbackRequest) error { return nil },
	}
	r := setupBuildRouter(svc)

	w := doRequest(r, "POST", "/api/builds/callback", map[string]interface{}{
		"build_id": validCallbackBuildID,
		"job_id":   "job-001",
		"status":   "completed",
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBuildController_Callback_InvalidJSON(t *testing.T) {
	svc := &mockBuildService{}
	r := setupBuildRouter(svc)

	req, _ := http.NewRequest("POST", "/api/builds/callback", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBuildController_Callback_ServiceError(t *testing.T) {
	svc := &mockBuildService{
		callbackFn: func(req *dto.BuildCallbackRequest) error {
			return domain.ErrBuildNotFound
		},
	}
	r := setupBuildRouter(svc)

	w := doRequest(r, "POST", "/api/builds/callback", map[string]interface{}{
		"build_id": validCallbackBuildID,
		"job_id":   "job-002",
		"status":   "failed",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestBuildController_Update_Success(t *testing.T) {
	name := "Renamed Build"
	svc := &mockBuildService{
		updateFn: func(id string, req *dto.UpdateBuildRequest) (*dto.BuildResponse, error) {
			return &dto.BuildResponse{ID: id, Name: *req.Name}, nil
		},
	}
	r := setupBuildRouter(svc)
	w := doRequest(r, "PUT", "/api/builds/b1", map[string]interface{}{"name": name})
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, name, data["name"])
}

func TestBuildController_Update_NotFound(t *testing.T) {
	svc := &mockBuildService{
		updateFn: func(id string, req *dto.UpdateBuildRequest) (*dto.BuildResponse, error) {
			return nil, domain.ErrBuildNotFound
		},
	}
	r := setupBuildRouter(svc)
	w := doRequest(r, "PUT", "/api/builds/missing", map[string]interface{}{"name": "X"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestBuildController_Delete_Success(t *testing.T) {
	svc := &mockBuildService{
		deleteFn: func(id string) error { return nil },
	}
	r := setupBuildRouter(svc)

	w := doRequest(r, "DELETE", "/api/builds/b1", nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ---------------------------------------------------------------------------
// List — error path and filter params
// ---------------------------------------------------------------------------

func TestBuildController_List_ServiceError(t *testing.T) {
	svc := &mockBuildService{
		listFn: func(p *dto.ListParams) (*dto.BuildListResponse, error) {
			return nil, errors.New("internal error")
		},
	}
	r := setupBuildRouter(svc)
	w := doRequest(r, "GET", "/api/builds", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestBuildController_List_WithSearchParam(t *testing.T) {
	var capturedSearch string
	svc := &mockBuildService{
		listFn: func(p *dto.ListParams) (*dto.BuildListResponse, error) {
			capturedSearch = p.Search
			return &dto.BuildListResponse{}, nil
		},
	}
	r := setupBuildRouter(svc)
	w := doRequest(r, "GET", "/api/builds?search=churn", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "churn", capturedSearch)
}

// ---------------------------------------------------------------------------
// Delete — error path
// ---------------------------------------------------------------------------

func TestBuildController_Delete_NotFound(t *testing.T) {
	svc := &mockBuildService{
		deleteFn: func(id string) error { return domain.ErrBuildNotFound },
	}
	r := setupBuildRouter(svc)
	w := doRequest(r, "DELETE", "/api/builds/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestBuildController_Delete_RunningBuild_BadRequest(t *testing.T) {
	svc := &mockBuildService{
		deleteFn: func(id string) error { return domain.ErrBuildAlreadyRunning },
	}
	r := setupBuildRouter(svc)
	w := doRequest(r, "DELETE", "/api/builds/running-build", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
