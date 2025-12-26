package api

import (
	"modelmatrix_backend/internal/infrastructure/auth"
	"modelmatrix_backend/internal/module/modelbuild/application"
	"modelmatrix_backend/internal/module/modelbuild/domain"
	"modelmatrix_backend/internal/module/modelbuild/dto"
	"modelmatrix_backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// BuildController handles model build-related HTTP requests
type BuildController struct {
	buildService application.BuildService
}

// NewBuildController creates a new build controller
func NewBuildController(buildService application.BuildService) *BuildController {
	return &BuildController{
		buildService: buildService,
	}
}

// RegisterRoutes registers build routes
func (c *BuildController) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	builds := router.Group("/builds")
	builds.Use(authMiddleware)
	{
		builds.POST("", auth.RequireEditor(), c.Create)
		builds.GET("", auth.RequireViewer(), c.List)
		builds.GET("/:id", auth.RequireViewer(), c.GetByID)
		builds.PUT("/:id", auth.RequireEditor(), c.Update)
		builds.DELETE("/:id", auth.RequireAdmin(), c.Delete)
		builds.POST("/:id/start", auth.RequireEditor(), c.Start)
		builds.POST("/:id/cancel", auth.RequireEditor(), c.Cancel)
	}
}

// Create godoc
// @Summary Create a new model build
// @Description Creates a new model training job
// @Tags Model Builds
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param build body dto.CreateBuildRequest true "Build data"
// @Success 201 {object} response.Response{data=dto.BuildResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/builds [post]
func (c *BuildController) Create(ctx *gin.Context) {
	var req dto.CreateBuildRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	username := auth.GetUsername(ctx)
	result, err := c.buildService.Create(&req, username)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Created(ctx, result)
}

// List godoc
// @Summary List model builds
// @Description Retrieves a paginated list of model builds
// @Tags Model Builds
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param search query string false "Search term"
// @Param status query string false "Filter by status"
// @Success 200 {object} response.Response{data=dto.BuildListResponse}
// @Failure 401 {object} response.Response
// @Router /api/builds [get]
func (c *BuildController) List(ctx *gin.Context) {
	var params dto.ListParams
	if err := ctx.ShouldBindQuery(&params); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.buildService.List(&params)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// GetByID godoc
// @Summary Get model build by ID
// @Description Retrieves a model build by its ID
// @Tags Model Builds
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Build ID (UUID)"
// @Success 200 {object} response.Response{data=dto.BuildResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/builds/{id} [get]
func (c *BuildController) GetByID(ctx *gin.Context) {
	id := ctx.Param("id")

	result, err := c.buildService.GetByID(id)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// Update godoc
// @Summary Update a model build
// @Description Updates an existing model build
// @Tags Model Builds
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Build ID (UUID)"
// @Param build body dto.UpdateBuildRequest true "Build update data"
// @Success 200 {object} response.Response{data=dto.BuildResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/builds/{id} [put]
func (c *BuildController) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var req dto.UpdateBuildRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.buildService.Update(id, &req)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// Delete godoc
// @Summary Delete a model build
// @Description Deletes a model build (admin only)
// @Tags Model Builds
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Build ID (UUID)"
// @Success 204 "No Content"
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/builds/{id} [delete]
func (c *BuildController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := c.buildService.Delete(id); err != nil {
		handleError(ctx, err)
		return
	}

	response.NoContent(ctx)
}

// Start godoc
// @Summary Start a model build
// @Description Starts a pending model build
// @Tags Model Builds
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Build ID (UUID)"
// @Success 200 {object} response.Response{data=dto.BuildResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/builds/{id}/start [post]
func (c *BuildController) Start(ctx *gin.Context) {
	id := ctx.Param("id")

	result, err := c.buildService.Start(id)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// Cancel godoc
// @Summary Cancel a model build
// @Description Cancels a pending or running model build
// @Tags Model Builds
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Build ID (UUID)"
// @Success 200 {object} response.Response{data=dto.BuildResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/builds/{id}/cancel [post]
func (c *BuildController) Cancel(ctx *gin.Context) {
	id := ctx.Param("id")

	result, err := c.buildService.Cancel(id)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// handleError maps domain errors to HTTP responses
func handleError(ctx *gin.Context, err error) {
	switch err {
	case domain.ErrBuildNotFound:
		response.NotFound(ctx, err.Error())
	case domain.ErrBuildNameExists:
		response.Conflict(ctx, err.Error())
	case domain.ErrBuildAlreadyRunning, domain.ErrBuildNotPending, domain.ErrBuildCannotBeCancelled,
		domain.ErrBuildNameEmpty, domain.ErrInvalidModelType, domain.ErrInvalidBuildStatus:
		response.BadRequest(ctx, err.Error())
	default:
		response.InternalError(ctx, err.Error())
	}
}
