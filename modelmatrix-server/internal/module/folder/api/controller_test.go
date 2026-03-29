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
	"modelmatrix-server/internal/module/folder/application"
	"modelmatrix-server/internal/module/folder/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock FolderService
// ---------------------------------------------------------------------------

type mockFolderService struct {
	createFolderFn         func(name, desc string, parentID *string, by string) (*domain.Folder, error)
	getFolderFn            func(id string) (*domain.Folder, error)
	updateFolderFn         func(id, name, desc string) (*domain.Folder, error)
	deleteFolderFn         func(id string, force bool) error
	getRootFoldersFn       func() ([]domain.Folder, error)
	getChildrenFn          func(parentID string) ([]domain.Folder, error)
	getContentsCountFn     func(id string) (*domain.FolderContentsCount, error)
	createProjectFn        func(name, desc string, folderID *string, by string) (*domain.Project, error)
	getProjectFn           func(id string) (*domain.Project, error)
	updateProjectFn        func(id, name, desc string) (*domain.Project, error)
	deleteProjectFn        func(id string, force bool) error
	getProjectsInFolderFn  func(folderID string) ([]domain.Project, error)
	getRootProjectsFn      func() ([]domain.Project, error)
}

func (m *mockFolderService) CreateFolder(name, desc string, parentID *string, by string) (*domain.Folder, error) {
	if m.createFolderFn != nil {
		return m.createFolderFn(name, desc, parentID, by)
	}
	return &domain.Folder{ID: "f1", Name: name}, nil
}
func (m *mockFolderService) GetFolder(id string) (*domain.Folder, error) {
	if m.getFolderFn != nil {
		return m.getFolderFn(id)
	}
	return &domain.Folder{ID: id}, nil
}
func (m *mockFolderService) UpdateFolder(id, name, desc string) (*domain.Folder, error) {
	if m.updateFolderFn != nil {
		return m.updateFolderFn(id, name, desc)
	}
	return &domain.Folder{ID: id, Name: name}, nil
}
func (m *mockFolderService) DeleteFolder(id string, force bool) error {
	if m.deleteFolderFn != nil {
		return m.deleteFolderFn(id, force)
	}
	return nil
}
func (m *mockFolderService) GetChildren(parentID string) ([]domain.Folder, error) {
	if m.getChildrenFn != nil {
		return m.getChildrenFn(parentID)
	}
	return []domain.Folder{}, nil
}
func (m *mockFolderService) GetRootFolders() ([]domain.Folder, error) {
	if m.getRootFoldersFn != nil {
		return m.getRootFoldersFn()
	}
	return []domain.Folder{}, nil
}
func (m *mockFolderService) GetFolderContentsCount(id string) (*domain.FolderContentsCount, error) {
	if m.getContentsCountFn != nil {
		return m.getContentsCountFn(id)
	}
	return &domain.FolderContentsCount{}, nil
}
func (m *mockFolderService) CreateProject(name, desc string, folderID *string, by string) (*domain.Project, error) {
	if m.createProjectFn != nil {
		return m.createProjectFn(name, desc, folderID, by)
	}
	return &domain.Project{ID: "p1", Name: name}, nil
}
func (m *mockFolderService) GetProject(id string) (*domain.Project, error) {
	if m.getProjectFn != nil {
		return m.getProjectFn(id)
	}
	return &domain.Project{ID: id}, nil
}
func (m *mockFolderService) UpdateProject(id, name, desc string) (*domain.Project, error) {
	if m.updateProjectFn != nil {
		return m.updateProjectFn(id, name, desc)
	}
	return &domain.Project{ID: id, Name: name}, nil
}
func (m *mockFolderService) DeleteProject(id string, force bool) error {
	if m.deleteProjectFn != nil {
		return m.deleteProjectFn(id, force)
	}
	return nil
}
func (m *mockFolderService) GetProjectsInFolder(folderID string) ([]domain.Project, error) {
	if m.getProjectsInFolderFn != nil {
		return m.getProjectsInFolderFn(folderID)
	}
	return []domain.Project{}, nil
}
func (m *mockFolderService) GetRootProjects() ([]domain.Project, error) {
	if m.getRootProjectsFn != nil {
		return m.getRootProjectsFn()
	}
	return []domain.Project{}, nil
}
func (m *mockFolderService) GetBuildsInFolder(id string) ([]string, error)   { return nil, nil }
func (m *mockFolderService) GetModelsInFolder(id string) ([]string, error)   { return nil, nil }
func (m *mockFolderService) GetBuildsInProject(id string) ([]string, error)  { return nil, nil }
func (m *mockFolderService) GetModelsInProject(id string) ([]string, error)  { return nil, nil }
func (m *mockFolderService) AddBuildToFolder(buildID, folderID string) error  { return nil }
func (m *mockFolderService) AddBuildToProject(buildID, projectID string) error { return nil }
func (m *mockFolderService) SetModelDeleter(d application.ModelDeleter) {}
func (m *mockFolderService) SetBuildDeleter(d application.BuildDeleter) {}

// ---------------------------------------------------------------------------
// Minimal stub build/model services (not used in most tests)
// ---------------------------------------------------------------------------

type stubBuildSvc struct{}
type stubModelSvc struct{}

// ---------------------------------------------------------------------------
// Router setup
// ---------------------------------------------------------------------------

func folderAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(auth.ContextKeyUser, &auth.Claims{
			Username: "admin",
			Groups:   []string{ldap.GroupAdmin},
		})
		c.Next()
	}
}

func setupFolderRouter(svc *mockFolderService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(folderAdminMiddleware())
	ctrl := NewFolderController(svc, nil, nil)
	api := r.Group("/api")
	ctrl.RegisterRoutes(api, func(c *gin.Context) { c.Next() })
	return r
}

func doFolderReq(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
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

func parseFolderResp(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out
}

// ---------------------------------------------------------------------------
// Folder: ListRootFolders
// ---------------------------------------------------------------------------

func TestFolderController_ListRootFolders_Success(t *testing.T) {
	svc := &mockFolderService{
		getRootFoldersFn: func() ([]domain.Folder, error) {
			return []domain.Folder{{ID: "f1", Name: "Root"}}, nil
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "GET", "/api/folders", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseFolderResp(t, w)
	data := body["data"].([]interface{})
	assert.Len(t, data, 1)
}

// ---------------------------------------------------------------------------
// Folder: CreateFolder
// ---------------------------------------------------------------------------

func TestFolderController_CreateFolder_Success(t *testing.T) {
	r := setupFolderRouter(&mockFolderService{})
	w := doFolderReq(r, "POST", "/api/folders", map[string]string{"name": "Analytics"})
	assert.Equal(t, http.StatusCreated, w.Code)
	body := parseFolderResp(t, w)
	data := body["data"].(map[string]interface{})
	assert.Equal(t, "Analytics", data["name"])
}

func TestFolderController_CreateFolder_InvalidJSON(t *testing.T) {
	r := setupFolderRouter(&mockFolderService{})
	req, _ := http.NewRequest("POST", "/api/folders", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFolderController_CreateFolder_DuplicateName(t *testing.T) {
	svc := &mockFolderService{
		createFolderFn: func(name, desc string, parentID *string, by string) (*domain.Folder, error) {
			return nil, domain.ErrFolderNameExists
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "POST", "/api/folders", map[string]string{"name": "Dup"})
	assert.Equal(t, http.StatusConflict, w.Code)
}

// ---------------------------------------------------------------------------
// Folder: GetFolder
// ---------------------------------------------------------------------------

func TestFolderController_GetFolder_Found(t *testing.T) {
	svc := &mockFolderService{
		getFolderFn: func(id string) (*domain.Folder, error) {
			return &domain.Folder{ID: id, Name: "Prod"}, nil
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "GET", "/api/folders/f1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFolderController_GetFolder_NotFound(t *testing.T) {
	svc := &mockFolderService{
		getFolderFn: func(id string) (*domain.Folder, error) {
			return nil, domain.ErrFolderNotFound
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "GET", "/api/folders/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Folder: UpdateFolder
// ---------------------------------------------------------------------------

func TestFolderController_UpdateFolder_Success(t *testing.T) {
	r := setupFolderRouter(&mockFolderService{})
	w := doFolderReq(r, "PUT", "/api/folders/f1", map[string]string{"name": "Renamed"})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFolderController_UpdateFolder_NotFound(t *testing.T) {
	svc := &mockFolderService{
		updateFolderFn: func(id, name, desc string) (*domain.Folder, error) {
			return nil, domain.ErrFolderNotFound
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "PUT", "/api/folders/missing", map[string]string{"name": "X"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Folder: DeleteFolder
// ---------------------------------------------------------------------------

func TestFolderController_DeleteFolder_Success(t *testing.T) {
	r := setupFolderRouter(&mockFolderService{})
	w := doFolderReq(r, "DELETE", "/api/folders/f1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFolderController_DeleteFolder_HasChildren(t *testing.T) {
	svc := &mockFolderService{
		deleteFolderFn: func(id string, force bool) error {
			return domain.ErrFolderHasChildren
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "DELETE", "/api/folders/f1", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFolderController_DeleteFolder_ForceQueryParam(t *testing.T) {
	var capturedForce bool
	svc := &mockFolderService{
		deleteFolderFn: func(id string, force bool) error {
			capturedForce = force
			return nil
		},
	}
	r := setupFolderRouter(svc)
	req, _ := http.NewRequest("DELETE", "/api/folders/f1?force=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, capturedForce)
}

// ---------------------------------------------------------------------------
// Folder: GetFolderChildren
// ---------------------------------------------------------------------------

func TestFolderController_GetFolderChildren_Success(t *testing.T) {
	svc := &mockFolderService{
		getChildrenFn: func(parentID string) ([]domain.Folder, error) {
			return []domain.Folder{{ID: "c1"}, {ID: "c2"}}, nil
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "GET", "/api/folders/f1/children", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseFolderResp(t, w)
	data := body["data"].([]interface{})
	assert.Len(t, data, 2)
}

// ---------------------------------------------------------------------------
// Project: ListProjects
// ---------------------------------------------------------------------------

func TestFolderController_ListProjects_Success(t *testing.T) {
	r := setupFolderRouter(&mockFolderService{})
	w := doFolderReq(r, "GET", "/api/projects", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Project: CreateProject
// ---------------------------------------------------------------------------

func TestFolderController_CreateProject_Success(t *testing.T) {
	r := setupFolderRouter(&mockFolderService{})
	w := doFolderReq(r, "POST", "/api/projects", map[string]string{"name": "Fraud Model"})
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestFolderController_CreateProject_DuplicateName(t *testing.T) {
	svc := &mockFolderService{
		createProjectFn: func(name, desc string, folderID *string, by string) (*domain.Project, error) {
			return nil, domain.ErrProjectNameExists
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "POST", "/api/projects", map[string]string{"name": "Dup"})
	assert.Equal(t, http.StatusConflict, w.Code)
}

// ---------------------------------------------------------------------------
// Project: GetProject / UpdateProject / DeleteProject
// ---------------------------------------------------------------------------

func TestFolderController_GetProject_Found(t *testing.T) {
	r := setupFolderRouter(&mockFolderService{})
	w := doFolderReq(r, "GET", "/api/projects/p1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFolderController_GetProject_NotFound(t *testing.T) {
	svc := &mockFolderService{
		getProjectFn: func(id string) (*domain.Project, error) {
			return nil, domain.ErrProjectNotFound
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "GET", "/api/projects/missing", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestFolderController_UpdateProject_Success(t *testing.T) {
	r := setupFolderRouter(&mockFolderService{})
	w := doFolderReq(r, "PUT", "/api/projects/p1", map[string]string{"name": "Renamed"})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFolderController_DeleteProject_Success(t *testing.T) {
	r := setupFolderRouter(&mockFolderService{})
	w := doFolderReq(r, "DELETE", "/api/projects/p1", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFolderController_DeleteProject_HasModels(t *testing.T) {
	svc := &mockFolderService{
		deleteProjectFn: func(id string, force bool) error {
			return domain.ErrProjectHasModels
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "DELETE", "/api/projects/p1", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// GetProjectsInFolder
// ---------------------------------------------------------------------------

func TestFolderController_GetProjectsInFolder_Success(t *testing.T) {
	svc := &mockFolderService{
		getProjectsInFolderFn: func(folderID string) ([]domain.Project, error) {
			return []domain.Project{{ID: "p1"}, {ID: "p2"}}, nil
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "GET", "/api/folders/f1/projects", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseFolderResp(t, w)
	data := body["data"].([]interface{})
	assert.Len(t, data, 2)
}

// ---------------------------------------------------------------------------
// GetFolderChildren — error path
// ---------------------------------------------------------------------------

func TestFolderController_GetFolderChildren_NotFound(t *testing.T) {
	svc := &mockFolderService{
		getChildrenFn: func(parentID string) ([]domain.Folder, error) {
			return nil, domain.ErrFolderNotFound
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "GET", "/api/folders/missing/children", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// GetProjectsInFolder — error path
// ---------------------------------------------------------------------------

func TestFolderController_GetProjectsInFolder_FolderNotFound(t *testing.T) {
	svc := &mockFolderService{
		getProjectsInFolderFn: func(folderID string) ([]domain.Project, error) {
			return nil, domain.ErrFolderNotFound
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "GET", "/api/folders/missing/projects", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// ListProjects — root=true branch and error path
// ---------------------------------------------------------------------------

func TestFolderController_ListProjects_RootTrue_Success(t *testing.T) {
	svc := &mockFolderService{
		getRootProjectsFn: func() ([]domain.Project, error) {
			return []domain.Project{{ID: "p1"}, {ID: "p2"}}, nil
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "GET", "/api/projects?root=true", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFolderController_ListProjects_ServiceError(t *testing.T) {
	svc := &mockFolderService{
		getRootProjectsFn: func() ([]domain.Project, error) {
			return nil, errors.New("db error")
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "GET", "/api/projects", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// ListRootFolders — error path
// ---------------------------------------------------------------------------

func TestFolderController_ListRootFolders_Error(t *testing.T) {
	svc := &mockFolderService{
		getRootFoldersFn: func() ([]domain.Folder, error) {
			return nil, errors.New("db error")
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "GET", "/api/folders", nil)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---------------------------------------------------------------------------
// handleFolderError — covers Conflict and BadRequest error types
// ---------------------------------------------------------------------------

func TestFolderController_DeleteFolder_HasProjects_Conflict(t *testing.T) {
	svc := &mockFolderService{
		deleteFolderFn: func(id string, force bool) error {
			return domain.ErrFolderHasProjects
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "DELETE", "/api/folders/f1", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFolderController_CreateFolder_DuplicateName_Conflict(t *testing.T) {
	svc := &mockFolderService{
		createFolderFn: func(name, desc string, parentID *string, by string) (*domain.Folder, error) {
			return nil, domain.ErrFolderNameExists
		},
	}
	r := setupFolderRouter(svc)
	w := doFolderReq(r, "POST", "/api/folders", map[string]string{"name": "Dup"})
	assert.Equal(t, http.StatusConflict, w.Code)
}
