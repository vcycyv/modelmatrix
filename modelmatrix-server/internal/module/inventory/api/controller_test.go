package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/internal/infrastructure/compute"
	"modelmatrix-server/internal/infrastructure/ldap"
	"modelmatrix-server/internal/module/inventory/application"
	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"
	"modelmatrix-server/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock ModelService
// ---------------------------------------------------------------------------

type mockModelService struct {
	createFn            func(req *dto.CreateModelRequest, by string) (*dto.ModelResponse, error)
	updateFn            func(id string, req *dto.UpdateModelRequest) (*dto.ModelResponse, error)
	deleteFn            func(id string) error
	getByIDFn           func(id string) (*dto.ModelDetailResponse, error)
	listFn              func(params *dto.ListParams) (*dto.ModelListResponse, error)
	activateFn          func(id string) (*dto.ModelResponse, error)
	deactivateFn        func(id string) (*dto.ModelResponse, error)
	getFileContentFn    func(modelID, fileID string) (*dto.FileContentResponse, error)
	scoreFn             func(modelID string, req *dto.ScoreRequest, by string) (*dto.ScoreResponse, error)
	handleScoreCallbackFn func(req *dto.ScoreCallbackRequest) error
}

func (m *mockModelService) Create(req *dto.CreateModelRequest, by string) (*dto.ModelResponse, error) {
	return m.createFn(req, by)
}
func (m *mockModelService) Update(id string, req *dto.UpdateModelRequest) (*dto.ModelResponse, error) {
	if m.updateFn != nil {
		return m.updateFn(id, req)
	}
	return &dto.ModelResponse{ID: id}, nil
}
func (m *mockModelService) Delete(id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	return nil
}
func (m *mockModelService) GetByID(id string) (*dto.ModelDetailResponse, error) {
	return m.getByIDFn(id)
}
func (m *mockModelService) List(params *dto.ListParams) (*dto.ModelListResponse, error) {
	if m.listFn != nil {
		return m.listFn(params)
	}
	return &dto.ModelListResponse{}, nil
}
func (m *mockModelService) Activate(id string) (*dto.ModelResponse, error) {
	return m.activateFn(id)
}
func (m *mockModelService) Deactivate(id string) (*dto.ModelResponse, error) {
	return m.deactivateFn(id)
}
func (m *mockModelService) CreateFromBuild(req *dto.CreateModelFromBuildRequest) (*dto.ModelResponse, error) {
	return nil, nil
}
func (m *mockModelService) UpdateFromBuild(modelID string, req *dto.CreateModelFromBuildRequest) (*dto.ModelResponse, error) {
	return nil, nil
}
func (m *mockModelService) Score(modelID string, req *dto.ScoreRequest, by string) (*dto.ScoreResponse, error) {
	if m.scoreFn != nil {
		return m.scoreFn(modelID, req, by)
	}
	return nil, nil
}
func (m *mockModelService) HandleScoreCallback(req *dto.ScoreCallbackRequest) error {
	if m.handleScoreCallbackFn != nil {
		return m.handleScoreCallbackFn(req)
	}
	return nil
}
func (m *mockModelService) ConfigureScoring(computeClient compute.Client, datasourceGetter application.DatasourceGetter, datasourceCreator application.DatasourceCreator, cfg *config.Config) {
}
func (m *mockModelService) GetFileContent(modelID, fileID string) (*dto.FileContentResponse, error) {
	if m.getFileContentFn != nil {
		return m.getFileContentFn(modelID, fileID)
	}
	return nil, nil
}
func (m *mockModelService) DeleteByFolderID(id string) error  { return nil }
func (m *mockModelService) DeleteByProjectID(id string) error { return nil }

// ---------------------------------------------------------------------------
// Router helpers
// ---------------------------------------------------------------------------

func modelAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(auth.ContextKeyUser, &auth.Claims{
			Username: "admin",
			Groups:   []string{ldap.GroupAdmin},
		})
		c.Next()
	}
}

func setupModelRouter(svc *mockModelService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(modelAdminMiddleware())
	ctrl := NewModelController(svc)
	api := r.Group("/api")
	ctrl.RegisterRoutes(api, func(c *gin.Context) { c.Next() })
	return r
}

func doModelReq(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
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

func parseModelResp(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestModelController_Create_Success(t *testing.T) {
	svc := &mockModelService{
		createFn: func(req *dto.CreateModelRequest, by string) (*dto.ModelResponse, error) {
			return &dto.ModelResponse{ID: "m1", Name: req.Name, Status: "draft"}, nil
		},
	}
	r := setupModelRouter(svc)

	w := doModelReq(r, "POST", "/api/models", map[string]interface{}{
		"name":          "Churn Predictor",
		"build_id":      "550e8400-e29b-41d4-a716-446655440001",
		"datasource_id": "550e8400-e29b-41d4-a716-446655440002",
		"algorithm":     "random_forest",
		"model_type":    "classification",
		"target_column": "churn",
	})

	assert.Equal(t, http.StatusCreated, w.Code)
	body := parseModelResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "m1", data["id"])
}

func TestModelController_Create_Conflict(t *testing.T) {
	svc := &mockModelService{
		createFn: func(req *dto.CreateModelRequest, by string) (*dto.ModelResponse, error) {
			return nil, domain.ErrModelNameExists
		},
	}
	r := setupModelRouter(svc)

	w := doModelReq(r, "POST", "/api/models", map[string]interface{}{
		"name":          "Dup",
		"build_id":      "550e8400-e29b-41d4-a716-446655440001",
		"datasource_id": "550e8400-e29b-41d4-a716-446655440002",
		"algorithm":     "xgboost",
		"model_type":    "regression",
		"target_column": "price",
	})
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestModelController_Create_InvalidJSON(t *testing.T) {
	r := setupModelRouter(&mockModelService{})
	req, _ := http.NewRequest("POST", "/api/models", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

func TestModelController_GetByID_Found(t *testing.T) {
	svc := &mockModelService{
		getByIDFn: func(id string) (*dto.ModelDetailResponse, error) {
			return &dto.ModelDetailResponse{ModelResponse: dto.ModelResponse{ID: id, Name: "Found"}}, nil
		},
	}
	r := setupModelRouter(svc)
	w := doModelReq(r, "GET", "/api/models/m1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseModelResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "m1", data["id"])
}

func TestModelController_GetByID_NotFound(t *testing.T) {
	svc := &mockModelService{
		getByIDFn: func(id string) (*dto.ModelDetailResponse, error) {
			return nil, domain.ErrModelNotFound
		},
	}
	r := setupModelRouter(svc)
	w := doModelReq(r, "GET", "/api/models/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestModelController_List_Success(t *testing.T) {
	svc := &mockModelService{
		listFn: func(params *dto.ListParams) (*dto.ModelListResponse, error) {
			return &dto.ModelListResponse{
				Models: []dto.ModelResponse{{ID: "m1"}, {ID: "m2"}},
				Total:  2,
			}, nil
		},
	}
	r := setupModelRouter(svc)
	w := doModelReq(r, "GET", "/api/models", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseModelResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["total"])
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestModelController_Update_Success(t *testing.T) {
	svc := &mockModelService{
		updateFn: func(id string, req *dto.UpdateModelRequest) (*dto.ModelResponse, error) {
			return &dto.ModelResponse{ID: id, Name: *req.Name}, nil
		},
	}
	r := setupModelRouter(svc)
	w := doModelReq(r, "PUT", "/api/models/m1", map[string]string{"name": "Renamed"})
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseModelResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "Renamed", data["name"])
}

func TestModelController_Update_NotFound(t *testing.T) {
	svc := &mockModelService{
		updateFn: func(id string, req *dto.UpdateModelRequest) (*dto.ModelResponse, error) {
			return nil, domain.ErrModelNotFound
		},
	}
	r := setupModelRouter(svc)
	w := doModelReq(r, "PUT", "/api/models/missing", map[string]string{"name": "X"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestModelController_Delete_Success(t *testing.T) {
	svc := &mockModelService{deleteFn: func(id string) error { return nil }}
	r := setupModelRouter(svc)
	w := doModelReq(r, "DELETE", "/api/models/m1", nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestModelController_Delete_ActiveModel_Conflict(t *testing.T) {
	svc := &mockModelService{deleteFn: func(id string) error { return domain.ErrModelCannotDelete }}
	r := setupModelRouter(svc)
	w := doModelReq(r, "DELETE", "/api/models/m1", nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// ---------------------------------------------------------------------------
// Activate / Deactivate
// ---------------------------------------------------------------------------

func TestModelController_Activate_Success(t *testing.T) {
	svc := &mockModelService{
		activateFn: func(id string) (*dto.ModelResponse, error) {
			return &dto.ModelResponse{ID: id, Status: "active"}, nil
		},
	}
	r := setupModelRouter(svc)
	w := doModelReq(r, "POST", "/api/models/m1/activate", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseModelResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "active", data["status"])
}

func TestModelController_Activate_DomainError(t *testing.T) {
	svc := &mockModelService{
		activateFn: func(id string) (*dto.ModelResponse, error) {
			return nil, domain.ErrModelCannotActivate
		},
	}
	r := setupModelRouter(svc)
	w := doModelReq(r, "POST", "/api/models/m1/activate", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestModelController_Deactivate_Success(t *testing.T) {
	svc := &mockModelService{
		deactivateFn: func(id string) (*dto.ModelResponse, error) {
			return &dto.ModelResponse{ID: id, Status: "inactive"}, nil
		},
	}
	r := setupModelRouter(svc)
	w := doModelReq(r, "POST", "/api/models/m1/deactivate", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestModelController_Deactivate_DomainError(t *testing.T) {
	svc := &mockModelService{
		deactivateFn: func(id string) (*dto.ModelResponse, error) {
			return nil, domain.ErrModelCannotDeactivate
		},
	}
	r := setupModelRouter(svc)
	w := doModelReq(r, "POST", "/api/models/m1/deactivate", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// GetFileContent
// ---------------------------------------------------------------------------

func TestModelController_GetFileContent_Success(t *testing.T) {
	svc := &mockModelService{
		getByIDFn: func(id string) (*dto.ModelDetailResponse, error) { return nil, nil },
		getFileContentFn: func(modelID, fileID string) (*dto.FileContentResponse, error) {
			return &dto.FileContentResponse{
				FileID:      fileID,
				ContentType: "text/plain",
				Content:     "feature,label\n1,0\n2,1",
				IsText:      true,
			}, nil
		},
	}
	r := setupModelRouter(svc)
	w := doModelReq(r, "GET", "/api/models/m1/files/f1/content", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseModelResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "f1", data["file_id"])
}

func TestModelController_GetFileContent_NotFound(t *testing.T) {
	svc := &mockModelService{
		getByIDFn: func(id string) (*dto.ModelDetailResponse, error) { return nil, nil },
		getFileContentFn: func(modelID, fileID string) (*dto.FileContentResponse, error) {
			return nil, domain.ErrModelNotFound
		},
	}
	r := setupModelRouter(svc)
	w := doModelReq(r, "GET", "/api/models/missing/files/f1/content", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
