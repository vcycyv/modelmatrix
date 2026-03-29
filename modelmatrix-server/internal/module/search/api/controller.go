package api

import (
	"errors"
	"fmt"

	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/internal/module/search/application"
	searchdto "modelmatrix-server/internal/module/search/dto"
	"modelmatrix-server/internal/module/search/repository"
	"modelmatrix-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// SearchController handles cross-type resource search (HTTP only).
type SearchController struct {
	svc *application.SearchService
}

// NewSearchController creates a controller backed by SearchService.
func NewSearchController(svc *application.SearchService) *SearchController {
	return &SearchController{svc: svc}
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

	resp, err := c.svc.Search(q, typeFilter, folderID, limit)
	if err != nil {
		if errors.Is(err, repository.ErrFolderNotFound) {
			response.BadRequest(ctx, "folder_id not found")
			return
		}
		response.InternalError(ctx, fmt.Sprintf("search failed: %v", err))
		return
	}

	response.Success(ctx, resp)
}

// Re-export DTO types for Swagger / external references that imported api package.
type (
	SearchResultItem = searchdto.SearchResultItem
	SearchResponse   = searchdto.SearchResponse
)

const (
	TypeBuild   = searchdto.TypeBuild
	TypeModel   = searchdto.TypeModel
	TypeProject = searchdto.TypeProject
	TypeFolder  = searchdto.TypeFolder
)
