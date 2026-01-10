package api

import (
	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/internal/module/inventory/application"
	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"
	"modelmatrix-server/pkg/response"

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
		models.POST("/:id/score", auth.RequireEditor(), c.Score)
	}

	// Callback endpoint (no auth required, called by compute service)
	router.POST("/models/:id/score/callback", c.ScoreCallback)
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
// @Description Retrieves a model by its ID with variables and files
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

// Score godoc
// @Summary Score data using a model
// @Description Scores input data using a trained model
// @Tags Models
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param score body dto.ScoreRequest true "Score request data"
// @Success 202 {object} response.Response{data=dto.ScoreResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/score [post]
func (c *ModelController) Score(ctx *gin.Context) {
	id := ctx.Param("id")

	var req dto.ScoreRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	username := auth.GetUsername(ctx)
	result, err := c.modelService.Score(id, &req, username)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Accepted(ctx, result)
}

// ScoreCallback godoc
// @Summary Handle score callback from compute service
// @Description Receives callback from compute service when scoring completes
// @Tags Models
// @Accept json
// @Produce json
// @Param id path string true "Model ID (UUID)"
// @Param callback body dto.ScoreCallbackRequest true "Callback data"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/models/{id}/score/callback [post]
func (c *ModelController) ScoreCallback(ctx *gin.Context) {
	var req dto.ScoreCallbackRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	// Use model ID from path if not in body
	if req.ModelID == "" {
		req.ModelID = ctx.Param("id")
	}

	// Parse query params for output datasource creation
	req.CollectionID = ctx.Query("collection_id")
	req.TableName = ctx.Query("table_name")
	req.CreatedBy = ctx.Query("created_by")

	if err := c.modelService.HandleScoreCallback(&req); err != nil {
		response.InternalError(ctx, err.Error())
		return
	}

	response.Success(ctx, map[string]string{"status": "acknowledged"})
}

// handleError maps domain errors to HTTP responses
func handleError(ctx *gin.Context, err error) {
	switch err {
	case domain.ErrModelNotFound, domain.ErrVariableNotFound, domain.ErrFileNotFound:
		response.NotFound(ctx, err.Error())
	case domain.ErrModelNameExists, domain.ErrModelCannotDelete:
		response.Conflict(ctx, err.Error())
	case domain.ErrModelNameEmpty, domain.ErrInvalidModelStatus, domain.ErrModelCannotActivate,
		domain.ErrModelCannotDeactivate:
		response.BadRequest(ctx, err.Error())
	default:
		response.InternalError(ctx, err.Error())
	}
}
