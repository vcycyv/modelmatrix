package api

import (
	"fmt"
	"time"

	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ResourceType constants used in filter and results
const (
	TypeBuild   = "build"
	TypeModel   = "model"
	TypeProject = "project"
	TypeFolder  = "folder"
)

// SearchResultItem is one entry in the search response
type SearchResultItem struct {
	Type        string    `json:"type"`        // build | model | project | folder
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status,omitempty"`
	ModelType   string    `json:"model_type,omitempty"`
	Algorithm   string    `json:"algorithm,omitempty"`
	FolderID    *string   `json:"folder_id,omitempty"`
	ProjectID   *string   `json:"project_id,omitempty"`
	Breadcrumb  string    `json:"breadcrumb"` // human-readable path, e.g. "Prod / Fraud"
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// SearchResponse is the API response for GET /api/search
type SearchResponse struct {
	Query   string             `json:"query"`
	Total   int                `json:"total"`
	Results []SearchResultItem `json:"results"`
}

// SearchController handles cross-type resource search
type SearchController struct {
	db *gorm.DB
}

// NewSearchController creates a new search controller
func NewSearchController(db *gorm.DB) *SearchController {
	return &SearchController{db: db}
}

// RegisterRoutes registers search routes
func (c *SearchController) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	router.GET("/search", authMiddleware, auth.RequireViewer(), c.Search)
}

// Search godoc
// @Summary Search across resource types
// @Description Searches builds, models, projects, and folders by name/description.
//
//	Optionally scoped to a folder (with full descendant support).
//
// @Tags Search
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param q query string true "Search text"
// @Param type query string false "Resource type filter: all|build|model|project|folder (default: all)"
// @Param folder_id query string false "Scope to this folder and all descendants"
// @Param limit query int false "Max results per type (default 20)"
// @Success 200 {object} response.Response{data=SearchResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/search [get]
func (c *SearchController) Search(ctx *gin.Context) {
	q := ctx.Query("q")
	if q == "" {
		response.BadRequest(ctx, "q query parameter is required")
		return
	}

	typeFilter := ctx.DefaultQuery("type", "all")
	folderID := ctx.Query("folder_id")
	limit := 20

	like := "%" + q + "%"

	// Resolve folder path for descendant scoping
	var folderPath string
	if folderID != "" {
		if err := c.db.Raw("SELECT path FROM folders WHERE id = ?", folderID).Scan(&folderPath).Error; err != nil || folderPath == "" {
			response.BadRequest(ctx, "folder_id not found")
			return
		}
	}
	descendantPattern := folderPath + "/%"

	var results []SearchResultItem

	if typeFilter == "all" || typeFilter == TypeBuild {
		builds, err := c.searchBuilds(like, folderID, folderPath, descendantPattern, limit)
		if err != nil {
			response.InternalError(ctx, fmt.Sprintf("build search failed: %v", err))
			return
		}
		results = append(results, builds...)
	}

	if typeFilter == "all" || typeFilter == TypeModel {
		models, err := c.searchModels(like, folderID, folderPath, descendantPattern, limit)
		if err != nil {
			response.InternalError(ctx, fmt.Sprintf("model search failed: %v", err))
			return
		}
		results = append(results, models...)
	}

	if typeFilter == "all" || typeFilter == TypeProject {
		projects, err := c.searchProjects(like, folderID, folderPath, descendantPattern, limit)
		if err != nil {
			response.InternalError(ctx, fmt.Sprintf("project search failed: %v", err))
			return
		}
		results = append(results, projects...)
	}

	if typeFilter == "all" || typeFilter == TypeFolder {
		folders, err := c.searchFolders(like, folderID, folderPath, descendantPattern, limit)
		if err != nil {
			response.InternalError(ctx, fmt.Sprintf("folder search failed: %v", err))
			return
		}
		results = append(results, folders...)
	}

	response.Success(ctx, SearchResponse{
		Query:   q,
		Total:   len(results),
		Results: results,
	})
}

// --- per-type search helpers ---

type buildRow struct {
	ID               string
	Name             string
	Description      string
	Status           string
	ModelType        string
	Algorithm        string
	FolderID         *string
	ProjectID        *string
	FolderName       *string
	ProjectName      *string
	ProjectFolderName *string
	CreatedBy        string
	CreatedAt        time.Time
}

func (c *SearchController) searchBuilds(like, folderID, folderPath, descendantPattern string, limit int) ([]SearchResultItem, error) {
	query := c.db.Raw(`
		SELECT b.id, b.name, b.description, b.status, b.model_type, b.algorithm,
		       b.folder_id, b.project_id,
		       f.name  AS folder_name,
		       p.name  AS project_name,
		       pf.name AS project_folder_name,
		       b.created_by, b.created_at
		FROM model_builds b
		LEFT JOIN folders  f  ON b.folder_id  = f.id
		LEFT JOIN projects p  ON b.project_id = p.id
		LEFT JOIN folders  pf ON p.folder_id  = pf.id
		WHERE (LOWER(b.name) LIKE LOWER(?) OR LOWER(b.description) LIKE LOWER(?))
		`+folderScope("b", folderID, folderPath, descendantPattern)+`
		ORDER BY b.created_at DESC
		LIMIT ?`,
		like, like, limit)

	var rows []buildRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]SearchResultItem, len(rows))
	for i, r := range rows {
		items[i] = SearchResultItem{
			Type:        TypeBuild,
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Status:      r.Status,
			ModelType:   r.ModelType,
			Algorithm:   r.Algorithm,
			FolderID:    r.FolderID,
			ProjectID:   r.ProjectID,
			Breadcrumb:  buildBreadcrumb(r.FolderName, r.ProjectName, r.ProjectFolderName),
			CreatedBy:   r.CreatedBy,
			CreatedAt:   r.CreatedAt,
		}
	}
	return items, nil
}

type modelRow struct {
	ID               string
	Name             string
	Description      string
	Status           string
	ModelType        string
	Algorithm        string
	FolderID         *string
	ProjectID        *string
	FolderName       *string
	ProjectName      *string
	ProjectFolderName *string
	CreatedBy        string
	CreatedAt        time.Time
}

func (c *SearchController) searchModels(like, folderID, folderPath, descendantPattern string, limit int) ([]SearchResultItem, error) {
	query := c.db.Raw(`
		SELECT m.id, m.name, m.description, m.status, m.model_type, m.algorithm,
		       m.folder_id, m.project_id,
		       f.name  AS folder_name,
		       p.name  AS project_name,
		       pf.name AS project_folder_name,
		       m.created_by, m.created_at
		FROM models m
		LEFT JOIN folders  f  ON m.folder_id  = f.id
		LEFT JOIN projects p  ON m.project_id = p.id
		LEFT JOIN folders  pf ON p.folder_id  = pf.id
		WHERE (LOWER(m.name) LIKE LOWER(?) OR LOWER(m.description) LIKE LOWER(?))
		`+folderScope("m", folderID, folderPath, descendantPattern)+`
		ORDER BY m.created_at DESC
		LIMIT ?`,
		like, like, limit)

	var rows []modelRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]SearchResultItem, len(rows))
	for i, r := range rows {
		items[i] = SearchResultItem{
			Type:        TypeModel,
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Status:      r.Status,
			ModelType:   r.ModelType,
			Algorithm:   r.Algorithm,
			FolderID:    r.FolderID,
			ProjectID:   r.ProjectID,
			Breadcrumb:  buildBreadcrumb(r.FolderName, r.ProjectName, r.ProjectFolderName),
			CreatedBy:   r.CreatedBy,
			CreatedAt:   r.CreatedAt,
		}
	}
	return items, nil
}

type projectRow struct {
	ID          string
	Name        string
	Description string
	FolderID    *string
	FolderName  *string
	CreatedBy   string
	CreatedAt   time.Time
}

func (c *SearchController) searchProjects(like, folderID, folderPath, descendantPattern string, limit int) ([]SearchResultItem, error) {
	scopeClause := ""
	if folderID != "" {
		scopeClause = fmt.Sprintf(`AND (
			p.folder_id = '%s'
			OR p.folder_id IN (SELECT id FROM folders WHERE path LIKE '%s')
		)`, folderID, descendantPattern)
	}

	query := c.db.Raw(`
		SELECT p.id, p.name, p.description, p.folder_id,
		       f.name AS folder_name,
		       p.created_by, p.created_at
		FROM projects p
		LEFT JOIN folders f ON p.folder_id = f.id
		WHERE (LOWER(p.name) LIKE LOWER(?) OR LOWER(p.description) LIKE LOWER(?))
		`+scopeClause+`
		ORDER BY p.created_at DESC
		LIMIT ?`,
		like, like, limit)

	var rows []projectRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]SearchResultItem, len(rows))
	for i, r := range rows {
		breadcrumb := ""
		if r.FolderName != nil {
			breadcrumb = *r.FolderName
		}
		items[i] = SearchResultItem{
			Type:        TypeProject,
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			FolderID:    r.FolderID,
			Breadcrumb:  breadcrumb,
			CreatedBy:   r.CreatedBy,
			CreatedAt:   r.CreatedAt,
		}
	}
	return items, nil
}

type folderRow struct {
	ID          string
	Name        string
	Description string
	ParentID    *string
	ParentName  *string
	CreatedBy   string
	CreatedAt   time.Time
}

func (c *SearchController) searchFolders(like, folderID, folderPath, descendantPattern string, limit int) ([]SearchResultItem, error) {
	scopeClause := ""
	if folderID != "" {
		scopeClause = fmt.Sprintf(`AND (
			f.path LIKE '%s'
			OR f.path LIKE '%s'
		)`, folderPath+"/"+"%", descendantPattern)
		// match direct children and all descendants; exclude the scope folder itself
		scopeClause = fmt.Sprintf(`AND f.path LIKE '%s'`, descendantPattern)
	}

	query := c.db.Raw(`
		SELECT f.id, f.name, f.description, f.parent_id,
		       pf.name AS parent_name,
		       f.created_by, f.created_at
		FROM folders f
		LEFT JOIN folders pf ON f.parent_id = pf.id
		WHERE (LOWER(f.name) LIKE LOWER(?) OR LOWER(f.description) LIKE LOWER(?))
		`+scopeClause+`
		ORDER BY f.depth ASC, f.name ASC
		LIMIT ?`,
		like, like, limit)

	var rows []folderRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]SearchResultItem, len(rows))
	for i, r := range rows {
		breadcrumb := ""
		if r.ParentName != nil {
			breadcrumb = *r.ParentName
		}
		items[i] = SearchResultItem{
			Type:        TypeFolder,
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			FolderID:    r.ParentID,
			Breadcrumb:  breadcrumb,
			CreatedBy:   r.CreatedBy,
			CreatedAt:   r.CreatedAt,
		}
	}
	return items, nil
}

// folderScope returns an extra SQL AND clause for folder-scoped search on builds/models.
// tableAlias is "b" or "m".
func folderScope(tableAlias, folderID, folderPath, descendantPattern string) string {
	if folderID == "" {
		return ""
	}
	return fmt.Sprintf(`AND (
		%s.folder_id = '%s'
		OR %s.folder_id IN (SELECT id FROM folders WHERE path LIKE '%s')
		OR %s.project_id IN (
			SELECT id FROM projects WHERE
				folder_id = '%s'
				OR folder_id IN (SELECT id FROM folders WHERE path LIKE '%s')
		)
	)`, tableAlias, folderID,
		tableAlias, descendantPattern,
		tableAlias, folderID, descendantPattern)
}

// buildBreadcrumb builds a human-readable location string
func buildBreadcrumb(folderName, projectName, projectFolderName *string) string {
	if folderName != nil && *folderName != "" {
		return *folderName
	}
	if projectName != nil && *projectName != "" {
		if projectFolderName != nil && *projectFolderName != "" {
			return *projectFolderName + " / " + *projectName
		}
		return *projectName
	}
	return ""
}
