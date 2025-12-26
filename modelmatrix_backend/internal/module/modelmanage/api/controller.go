package api

import (
	"modelmatrix_backend/internal/infrastructure/auth"
	"modelmatrix_backend/internal/module/modelmanage/application"
	"modelmatrix_backend/internal/module/modelmanage/domain"
	"modelmatrix_backend/internal/module/modelmanage/dto"
	"modelmatrix_backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// ModelController handles model management-related HTTP requests
type ModelController struct {
	modelService application.ModelService
}

// NewModelController creates a new model controller
func NewModelController(modelService application.ModelService) *ModelController {
	return &ModelController{
		modelService: modelService,
	}
}

// RegisterRoutes registers model routes
func (c *ModelController) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	models := router.Group("/models")
	models.Use(authMiddleware)
	{
		models.POST("", auth.RequireEditor(), c.Create)
		models.GET("", auth.RequireViewer(), c.List)
		models.GET("/:id", auth.RequireViewer(), c.GetByID)
		models.PUT("/:id", auth.RequireEditor(), c.Update)
		models.DELETE("/:id", auth.RequireAdmin(), c.Delete)
		models.POST("/:id/activate", auth.RequireEditor(), c.Activate)
		models.POST("/:id/deactivate", auth.RequireEditor(), c.Deactivate)

		// Version endpoints
		models.GET("/:id/versions", auth.RequireViewer(), c.GetVersions)
		models.POST("/:id/versions", auth.RequireEditor(), c.CreateVersion)
	}
}

// Create godoc
// @Summary Create a new model
// @Description Creates a new trained model
// @Tags Models
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param model body dto.CreateModelRequest true "Model data"
// @Success 201 {object} response.Response{data=dto.ModelResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/models [post]
func (c *ModelController) Create(ctx *gin.Context) {
	var req dto.CreateModelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	username := auth.GetUsername(ctx)
	result, err := c.modelService.Create(&req, username)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Created(ctx, result)
}

// List godoc
// @Summary List models
// @Description Retrieves a paginated list of models
// @Tags Models
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param search query string false "Search term"
// @Param status query string false "Filter by status"
// @Success 200 {object} response.Response{data=dto.ModelListResponse}
// @Failure 401 {object} response.Response
// @Router /api/models [get]
func (c *ModelController) List(ctx *gin.Context) {
	var params dto.ListParams
	if err := ctx.ShouldBindQuery(&params); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.modelService.List(&params)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// GetByID godoc
// @Summary Get model by ID
// @Description Retrieves a model by its ID with versions
// @Tags Models
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Success 200 {object} response.Response{data=dto.ModelDetailResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id} [get]
func (c *ModelController) GetByID(ctx *gin.Context) {
	id := ctx.Param("id")

	result, err := c.modelService.GetByID(id)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// Update godoc
// @Summary Update a model
// @Description Updates an existing model
// @Tags Models
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param model body dto.UpdateModelRequest true "Model update data"
// @Success 200 {object} response.Response{data=dto.ModelResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id} [put]
func (c *ModelController) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var req dto.UpdateModelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.modelService.Update(id, &req)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// Delete godoc
// @Summary Delete a model
// @Description Deletes a model (admin only, active models cannot be deleted)
// @Tags Models
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Success 204 "No Content"
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/models/{id} [delete]
func (c *ModelController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := c.modelService.Delete(id); err != nil {
		handleError(ctx, err)
		return
	}

	response.NoContent(ctx)
}

// Activate godoc
// @Summary Activate a model
// @Description Activates a draft or inactive model
// @Tags Models
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Success 200 {object} response.Response{data=dto.ModelResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/activate [post]
func (c *ModelController) Activate(ctx *gin.Context) {
	id := ctx.Param("id")

	result, err := c.modelService.Activate(id)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// Deactivate godoc
// @Summary Deactivate a model
// @Description Deactivates an active model
// @Tags Models
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Success 200 {object} response.Response{data=dto.ModelResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/deactivate [post]
func (c *ModelController) Deactivate(ctx *gin.Context) {
	id := ctx.Param("id")

	result, err := c.modelService.Deactivate(id)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// GetVersions godoc
// @Summary Get model versions
// @Description Retrieves all versions for a model
// @Tags Models
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Success 200 {object} response.Response{data=[]dto.VersionResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/versions [get]
func (c *ModelController) GetVersions(ctx *gin.Context) {
	id := ctx.Param("id")

	result, err := c.modelService.GetVersions(id)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// CreateVersion godoc
// @Summary Create model version
// @Description Creates a new version for a model
// @Tags Models
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param version body dto.CreateVersionRequest true "Version data"
// @Success 201 {object} response.Response{data=dto.VersionResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/models/{id}/versions [post]
func (c *ModelController) CreateVersion(ctx *gin.Context) {
	id := ctx.Param("id")

	var req dto.CreateVersionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	username := auth.GetUsername(ctx)
	result, err := c.modelService.CreateVersion(id, &req, username)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Created(ctx, result)
}

// handleError maps domain errors to HTTP responses
func handleError(ctx *gin.Context, err error) {
	switch err {
	case domain.ErrModelNotFound, domain.ErrModelVersionNotFound:
		response.NotFound(ctx, err.Error())
	case domain.ErrModelNameExists, domain.ErrActiveModelCannotBeDeleted, domain.ErrVersionExists:
		response.Conflict(ctx, err.Error())
	case domain.ErrModelNameEmpty, domain.ErrInvalidModelStatus, domain.ErrModelCannotBeActivated,
		domain.ErrModelCannotBeDeactivated, domain.ErrModelAlreadyActive, domain.ErrModelAlreadyInactive:
		response.BadRequest(ctx, err.Error())
	default:
		response.InternalError(ctx, err.Error())
	}
}
