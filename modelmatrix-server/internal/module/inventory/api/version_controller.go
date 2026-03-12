package api

import (
	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/internal/module/inventory/application"
	"modelmatrix-server/internal/module/inventory/domain"
	"modelmatrix-server/internal/module/inventory/dto"
	"modelmatrix-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// VersionController handles model version HTTP requests
type VersionController struct {
	versionService application.ModelVersionService
}

// NewVersionController creates a new version controller
func NewVersionController(versionService application.ModelVersionService) *VersionController {
	return &VersionController{versionService: versionService}
}

// RegisterRoutes registers version routes under /models/:id/versions
func (c *VersionController) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	models := router.Group("/models")
	models.Use(authMiddleware)
	{
		models.POST("/:id/versions", auth.RequireEditor(), c.CreateVersion)
		models.GET("/:id/versions", auth.RequireViewer(), c.ListVersions)
		models.GET("/:id/versions/:versionId", auth.RequireViewer(), c.GetVersion)
		models.POST("/:id/versions/:versionId/restore", auth.RequireEditor(), c.RestoreVersion)
	}
}

// CreateVersion godoc
// @Summary Create a model version snapshot
// @Description Creates an immutable snapshot of the current model (metadata + content-addressable files)
// @Tags Model Versions
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Success 201 {object} response.Response{data=dto.VersionResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/versions [post]
func (c *VersionController) CreateVersion(ctx *gin.Context) {
	modelID := ctx.Param("id")
	username := auth.GetUsername(ctx)

	result, err := c.versionService.CreateVersion(modelID, username)
	if err != nil {
		handleVersionError(ctx, err)
		return
	}
	response.Created(ctx, result)
}

// ListVersions godoc
// @Summary List model versions
// @Description Returns versions for a model (newest first)
// @Tags Model Versions
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param page query int false "Page" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} response.Response{data=dto.VersionListResponse}
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/versions [get]
func (c *VersionController) ListVersions(ctx *gin.Context) {
	modelID := ctx.Param("id")
	var params dto.ListVersionsParams
	_ = ctx.ShouldBindQuery(&params)

	result, err := c.versionService.ListVersions(modelID, &params)
	if err != nil {
		handleVersionError(ctx, err)
		return
	}
	response.Success(ctx, result)
}

// GetVersion godoc
// @Summary Get a model version
// @Description Returns full version snapshot (metadata + variables + files)
// @Tags Model Versions
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param versionId path string true "Version ID (UUID)"
// @Success 200 {object} response.Response{data=dto.VersionDetailResponse}
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/versions/{versionId} [get]
func (c *VersionController) GetVersion(ctx *gin.Context) {
	modelID := ctx.Param("id")
	versionID := ctx.Param("versionId")

	result, err := c.versionService.GetVersion(modelID, versionID)
	if err != nil {
		handleVersionError(ctx, err)
		return
	}
	response.Success(ctx, result)
}

// RestoreVersion godoc
// @Summary Restore model from a version
// @Description Restores the current model from a version snapshot (metadata + variables + files)
// @Tags Model Versions
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Model ID (UUID)"
// @Param versionId path string true "Version ID (UUID)"
// @Success 200 {object} response.Response{data=dto.ModelResponse}
// @Failure 404 {object} response.Response
// @Router /api/models/{id}/versions/{versionId}/restore [post]
func (c *VersionController) RestoreVersion(ctx *gin.Context) {
	modelID := ctx.Param("id")
	versionID := ctx.Param("versionId")
	username := auth.GetUsername(ctx)

	result, err := c.versionService.RestoreVersion(modelID, versionID, username)
	if err != nil {
		handleVersionError(ctx, err)
		return
	}
	response.Success(ctx, result)
}

func handleVersionError(ctx *gin.Context, err error) {
	switch err {
	case domain.ErrVersionNotFound, domain.ErrModelNotFound:
		response.NotFound(ctx, err.Error())
	default:
		response.InternalError(ctx, err.Error())
	}
}
