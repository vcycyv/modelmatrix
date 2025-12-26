package api

import (
	"modelmatrix_backend/internal/infrastructure/auth"
	"modelmatrix_backend/internal/module/datasource/application"
	"modelmatrix_backend/internal/module/datasource/domain"
	"modelmatrix_backend/internal/module/datasource/dto"
	"modelmatrix_backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// CollectionController handles collection-related HTTP requests
type CollectionController struct {
	collectionService application.CollectionService
}

// NewCollectionController creates a new collection controller
func NewCollectionController(collectionService application.CollectionService) *CollectionController {
	return &CollectionController{
		collectionService: collectionService,
	}
}

// RegisterRoutes registers collection routes
func (c *CollectionController) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	collections := router.Group("/collections")
	collections.Use(authMiddleware)
	{
		collections.POST("", auth.RequireEditor(), c.Create)
		collections.GET("", auth.RequireViewer(), c.List)
		collections.GET("/:id", auth.RequireViewer(), c.GetByID)
		collections.PUT("/:id", auth.RequireEditor(), c.Update)
		collections.DELETE("/:id", auth.RequireAdmin(), c.Delete)
	}
}

// Create godoc
// @Summary Create a new collection
// @Description Creates a new datasource collection
// @Tags Collections
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param collection body dto.CreateCollectionRequest true "Collection data"
// @Success 201 {object} response.Response{data=dto.CollectionResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/collections [post]
func (c *CollectionController) Create(ctx *gin.Context) {
	var req dto.CreateCollectionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	username := auth.GetUsername(ctx)
	result, err := c.collectionService.Create(&req, username)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Created(ctx, result)
}

// List godoc
// @Summary List collections
// @Description Retrieves a paginated list of collections
// @Tags Collections
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param search query string false "Search term"
// @Success 200 {object} response.Response{data=dto.CollectionListResponse}
// @Failure 401 {object} response.Response
// @Router /api/collections [get]
func (c *CollectionController) List(ctx *gin.Context) {
	var params dto.ListParams
	if err := ctx.ShouldBindQuery(&params); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.collectionService.List(&params)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// GetByID godoc
// @Summary Get collection by ID
// @Description Retrieves a collection by its ID
// @Tags Collections
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Collection ID (UUID)"
// @Success 200 {object} response.Response{data=dto.CollectionResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/collections/{id} [get]
func (c *CollectionController) GetByID(ctx *gin.Context) {
	id := ctx.Param("id")

	result, err := c.collectionService.GetByID(id)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// Update godoc
// @Summary Update a collection
// @Description Updates an existing collection
// @Tags Collections
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Collection ID (UUID)"
// @Param collection body dto.UpdateCollectionRequest true "Collection update data"
// @Success 200 {object} response.Response{data=dto.CollectionResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/collections/{id} [put]
func (c *CollectionController) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var req dto.UpdateCollectionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.collectionService.Update(id, &req)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// Delete godoc
// @Summary Delete a collection
// @Description Deletes a collection (admin only)
// @Tags Collections
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Collection ID (UUID)"
// @Success 204 "No Content"
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/collections/{id} [delete]
func (c *CollectionController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := c.collectionService.Delete(id); err != nil {
		handleError(ctx, err)
		return
	}

	response.NoContent(ctx)
}

// handleError maps domain errors to HTTP responses
func handleError(ctx *gin.Context, err error) {
	switch err {
	case domain.ErrCollectionNotFound, domain.ErrDatasourceNotFound, domain.ErrColumnNotFound:
		response.NotFound(ctx, err.Error())
	case domain.ErrCollectionNameExists, domain.ErrDatasourceNameExists, domain.ErrCollectionHasDatasources:
		response.Conflict(ctx, err.Error())
	case domain.ErrMultipleTargetColumns, domain.ErrInvalidColumnRole, domain.ErrInvalidDatasourceType,
		domain.ErrCollectionNameEmpty, domain.ErrDatasourceNameEmpty, domain.ErrFilePathRequired, domain.ErrConnectionConfigRequired:
		response.BadRequest(ctx, err.Error())
	default:
		response.InternalError(ctx, err.Error())
	}
}

