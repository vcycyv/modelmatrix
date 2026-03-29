package api

import (
	"net/http"
	"testing"

	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Mock ModelVersionService
// ---------------------------------------------------------------------------

type mockVersionService struct {
	createVersionFn  func(modelID, by string) (*dto.VersionResponse, error)
	listVersionsFn   func(modelID string, params *dto.ListVersionsParams) (*dto.VersionListResponse, error)
	getVersionFn     func(modelID, versionID string) (*dto.VersionDetailResponse, error)
	restoreVersionFn func(modelID, versionID, by string) (*dto.ModelResponse, error)
}

func (m *mockVersionService) CreateVersion(modelID, by string) (*dto.VersionResponse, error) {
	if m.createVersionFn != nil {
		return m.createVersionFn(modelID, by)
	}
	return &dto.VersionResponse{ID: "v1", ModelID: modelID}, nil
}
func (m *mockVersionService) ListVersions(modelID string, params *dto.ListVersionsParams) (*dto.VersionListResponse, error) {
	if m.listVersionsFn != nil {
		return m.listVersionsFn(modelID, params)
	}
	return &dto.VersionListResponse{}, nil
}
func (m *mockVersionService) GetVersion(modelID, versionID string) (*dto.VersionDetailResponse, error) {
	if m.getVersionFn != nil {
		return m.getVersionFn(modelID, versionID)
	}
	return &dto.VersionDetailResponse{}, nil
}
func (m *mockVersionService) RestoreVersion(modelID, versionID, by string) (*dto.ModelResponse, error) {
	if m.restoreVersionFn != nil {
		return m.restoreVersionFn(modelID, versionID, by)
	}
	return &dto.ModelResponse{ID: modelID}, nil
}

// ---------------------------------------------------------------------------
// Router helpers
// ---------------------------------------------------------------------------

func setupVersionRouter(svc *mockVersionService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(modelAdminMiddleware())
	ctrl := NewVersionController(svc)
	api := r.Group("/api")
	ctrl.RegisterRoutes(api, func(c *gin.Context) { c.Next() })
	return r
}

// ---------------------------------------------------------------------------
// CreateVersion
// ---------------------------------------------------------------------------

func TestVersionController_CreateVersion_Success(t *testing.T) {
	svc := &mockVersionService{
		createVersionFn: func(modelID, by string) (*dto.VersionResponse, error) {
			return &dto.VersionResponse{ID: "v1", ModelID: modelID, VersionNumber: 1}, nil
		},
	}
	r := setupVersionRouter(svc)
	w := doModelReq(r, "POST", "/api/models/m1/versions", nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	body := parseModelResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "v1", data["id"])
}

func TestVersionController_CreateVersion_ModelNotFound(t *testing.T) {
	svc := &mockVersionService{
		createVersionFn: func(modelID, by string) (*dto.VersionResponse, error) {
			return nil, domain.ErrModelNotFound
		},
	}
	r := setupVersionRouter(svc)
	w := doModelReq(r, "POST", "/api/models/missing/versions", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// ListVersions
// ---------------------------------------------------------------------------

func TestVersionController_ListVersions_Success(t *testing.T) {
	svc := &mockVersionService{
		listVersionsFn: func(modelID string, params *dto.ListVersionsParams) (*dto.VersionListResponse, error) {
			return &dto.VersionListResponse{
				Versions: []dto.VersionResponse{{ID: "v1"}, {ID: "v2"}},
				Total:    2,
			}, nil
		},
	}
	r := setupVersionRouter(svc)
	w := doModelReq(r, "GET", "/api/models/m1/versions", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseModelResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["total"])
}

func TestVersionController_ListVersions_ModelNotFound(t *testing.T) {
	svc := &mockVersionService{
		listVersionsFn: func(modelID string, params *dto.ListVersionsParams) (*dto.VersionListResponse, error) {
			return nil, domain.ErrModelNotFound
		},
	}
	r := setupVersionRouter(svc)
	w := doModelReq(r, "GET", "/api/models/missing/versions", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetVersion
// ---------------------------------------------------------------------------

func TestVersionController_GetVersion_Success(t *testing.T) {
	svc := &mockVersionService{
		getVersionFn: func(modelID, versionID string) (*dto.VersionDetailResponse, error) {
			return &dto.VersionDetailResponse{VersionResponse: dto.VersionResponse{ID: versionID}}, nil
		},
	}
	r := setupVersionRouter(svc)
	w := doModelReq(r, "GET", "/api/models/m1/versions/v1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVersionController_GetVersion_NotFound(t *testing.T) {
	svc := &mockVersionService{
		getVersionFn: func(modelID, versionID string) (*dto.VersionDetailResponse, error) {
			return nil, domain.ErrVersionNotFound
		},
	}
	r := setupVersionRouter(svc)
	w := doModelReq(r, "GET", "/api/models/m1/versions/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// RestoreVersion
// ---------------------------------------------------------------------------

func TestVersionController_RestoreVersion_Success(t *testing.T) {
	svc := &mockVersionService{
		restoreVersionFn: func(modelID, versionID, by string) (*dto.ModelResponse, error) {
			return &dto.ModelResponse{ID: modelID, Status: "draft"}, nil
		},
	}
	r := setupVersionRouter(svc)
	w := doModelReq(r, "POST", "/api/models/m1/versions/v1/restore", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestVersionController_RestoreVersion_NotFound(t *testing.T) {
	svc := &mockVersionService{
		restoreVersionFn: func(modelID, versionID, by string) (*dto.ModelResponse, error) {
			return nil, domain.ErrVersionNotFound
		},
	}
	r := setupVersionRouter(svc)
	w := doModelReq(r, "POST", "/api/models/m1/versions/missing/restore", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
