package api

// Tests for Score, ScoreCallback, and Retrain handlers.
// These handlers have real testable value: JSON binding validation and error mapping.

import (
	"errors"
	"net/http"
	"testing"

	buildApp "modelmatrix-server/internal/module/build/application"
	buildDto "modelmatrix-server/internal/module/build/dto"
	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Minimal mock for buildApp.BuildService needed only for Retrain
// ---------------------------------------------------------------------------

type minimalBuildService struct {
	retrainFn func(modelID string, req *buildDto.RetrainRequest, by string) (*buildDto.BuildResponse, error)
}

func (m *minimalBuildService) Create(req *buildDto.CreateBuildRequest, by string) (*buildDto.BuildResponse, error) {
	return nil, nil
}
func (m *minimalBuildService) GetByID(id string) (*buildDto.BuildResponse, error)  { return nil, nil }
func (m *minimalBuildService) List(p *buildDto.ListParams) (*buildDto.BuildListResponse, error) {
	return &buildDto.BuildListResponse{}, nil
}
func (m *minimalBuildService) Update(id string, req *buildDto.UpdateBuildRequest) (*buildDto.BuildResponse, error) {
	return nil, nil
}
func (m *minimalBuildService) Start(id string) (*buildDto.BuildResponse, error)  { return nil, nil }
func (m *minimalBuildService) Cancel(id string) (*buildDto.BuildResponse, error) { return nil, nil }
func (m *minimalBuildService) HandleCallback(req *buildDto.BuildCallbackRequest) error { return nil }
func (m *minimalBuildService) Delete(id string) error                                   { return nil }
func (m *minimalBuildService) DeleteByFolderID(id string) error                         { return nil }
func (m *minimalBuildService) DeleteByProjectID(id string) error                        { return nil }
func (m *minimalBuildService) Retrain(modelID string, req *buildDto.RetrainRequest, by string) (*buildDto.BuildResponse, error) {
	if m.retrainFn != nil {
		return m.retrainFn(modelID, req, by)
	}
	return &buildDto.BuildResponse{ID: "b1"}, nil
}

var _ buildApp.BuildService = (*minimalBuildService)(nil)

// Router with retrain support
func setupModelRouterWithRetrain(svc *mockModelService, bSvc buildApp.BuildService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(modelAdminMiddleware())
	ctrl := NewModelControllerWithRetrain(svc, bSvc)
	api := r.Group("/api")
	ctrl.RegisterRoutes(api, func(c *gin.Context) { c.Next() })
	return r
}

// ---------------------------------------------------------------------------
// Model List — service error path (missing from existing tests)
// ---------------------------------------------------------------------------

func TestModelController_List_ServiceError(t *testing.T) {
	svc := &mockModelService{
		listFn: func(params *dto.ListParams) (*dto.ModelListResponse, error) {
			return nil, errors.New("db error")
		},
	}
	r := setupModelRouter(svc)
	w := doModelReq(r, "GET", "/api/models", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// Score
// ---------------------------------------------------------------------------

func TestModelController_Score_Success(t *testing.T) {
	svc := &mockModelService{
		getByIDFn: func(id string) (*dto.ModelDetailResponse, error) { return nil, nil },
		scoreFn: func(modelID string, req *dto.ScoreRequest, by string) (*dto.ScoreResponse, error) {
			return &dto.ScoreResponse{JobID: "job-1", Status: "scoring"}, nil
		},
	}
	r := setupModelRouter(svc)
	payload := map[string]string{
		"datasource_id":        "550e8400-e29b-41d4-a716-446655440000",
		"output_collection_id": "550e8400-e29b-41d4-a716-446655440001",
	}
	w := doModelReq(r, "POST", "/api/models/m1/score", payload)
	assert.Equal(t, http.StatusAccepted, w.Code)
	body := parseModelResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "job-1", data["job_id"])
}

func TestModelController_Score_MissingRequiredFields(t *testing.T) {
	svc := &mockModelService{
		getByIDFn: func(id string) (*dto.ModelDetailResponse, error) { return nil, nil },
	}
	r := setupModelRouter(svc)
	// Missing required datasource_id and output_collection_id
	w := doModelReq(r, "POST", "/api/models/m1/score", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestModelController_Score_ModelNotFound(t *testing.T) {
	svc := &mockModelService{
		getByIDFn: func(id string) (*dto.ModelDetailResponse, error) { return nil, nil },
		scoreFn: func(modelID string, req *dto.ScoreRequest, by string) (*dto.ScoreResponse, error) {
			return nil, domain.ErrModelNotFound
		},
	}
	r := setupModelRouter(svc)
	payload := map[string]string{
		"datasource_id":        "550e8400-e29b-41d4-a716-446655440000",
		"output_collection_id": "550e8400-e29b-41d4-a716-446655440001",
	}
	w := doModelReq(r, "POST", "/api/models/m1/score", payload)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// ScoreCallback
// ---------------------------------------------------------------------------

func TestModelController_ScoreCallback_Success(t *testing.T) {
	svc := &mockModelService{
		getByIDFn: func(id string) (*dto.ModelDetailResponse, error) { return nil, nil },
		handleScoreCallbackFn: func(req *dto.ScoreCallbackRequest) error { return nil },
	}
	r := setupModelRouter(svc)
	payload := map[string]interface{}{
		"model_id": "m1",
		"job_id":   "job-1",
		"status":   "completed",
	}
	w := doModelReq(r, "POST", "/api/models/m1/score/callback", payload)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestModelController_ScoreCallback_InvalidJSON(t *testing.T) {
	svc := &mockModelService{
		getByIDFn: func(id string) (*dto.ModelDetailResponse, error) { return nil, nil },
	}
	r := setupModelRouter(svc)
	w := doModelReq(r, "POST", "/api/models/m1/score/callback", "not-json")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestModelController_ScoreCallback_ServiceError(t *testing.T) {
	svc := &mockModelService{
		getByIDFn: func(id string) (*dto.ModelDetailResponse, error) { return nil, nil },
		handleScoreCallbackFn: func(req *dto.ScoreCallbackRequest) error {
			return errors.New("callback processing failed")
		},
	}
	r := setupModelRouter(svc)
	payload := map[string]interface{}{"model_id": "m1", "job_id": "j1", "status": "failed"}
	w := doModelReq(r, "POST", "/api/models/m1/score/callback", payload)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// Retrain (uses buildService)
// ---------------------------------------------------------------------------

func TestModelController_Retrain_Success(t *testing.T) {
	modelSvc := &mockModelService{
		getByIDFn: func(id string) (*dto.ModelDetailResponse, error) { return nil, nil },
	}
	buildSvc := &minimalBuildService{
		retrainFn: func(modelID string, req *buildDto.RetrainRequest, by string) (*buildDto.BuildResponse, error) {
			return &buildDto.BuildResponse{ID: "b1", Name: "Retrain-1"}, nil
		},
	}
	r := setupModelRouterWithRetrain(modelSvc, buildSvc)
	w := doModelReq(r, "POST", "/api/models/m1/retrain", map[string]interface{}{})
	assert.Equal(t, http.StatusAccepted, w.Code)
}

func TestModelController_Retrain_ModelNotFound(t *testing.T) {
	modelSvc := &mockModelService{
		getByIDFn: func(id string) (*dto.ModelDetailResponse, error) { return nil, nil },
	}
	buildSvc := &minimalBuildService{
		retrainFn: func(modelID string, req *buildDto.RetrainRequest, by string) (*buildDto.BuildResponse, error) {
			return nil, domain.ErrModelNotFound // handleError maps this to 404
		},
	}
	r := setupModelRouterWithRetrain(modelSvc, buildSvc)
	w := doModelReq(r, "POST", "/api/models/missing/retrain", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
