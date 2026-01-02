package api

import (
	"encoding/json"
	"io"

	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/internal/module/datasource/application"
	"modelmatrix-server/internal/module/datasource/dto"
	"modelmatrix-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// DatasourceController handles datasource-related HTTP requests
type DatasourceController struct {
	datasourceService application.DatasourceService
	columnService     application.ColumnService
}

// NewDatasourceController creates a new datasource controller
func NewDatasourceController(
	datasourceService application.DatasourceService,
	columnService application.ColumnService,
) *DatasourceController {
	return &DatasourceController{
		datasourceService: datasourceService,
		columnService:     columnService,
	}
}

// RegisterRoutes registers datasource routes
func (c *DatasourceController) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	datasources := router.Group("/datasources")
	datasources.Use(authMiddleware)
	{
		datasources.POST("", auth.RequireEditor(), c.Create)
		datasources.GET("", auth.RequireViewer(), c.List)
		datasources.GET("/:id", auth.RequireViewer(), c.GetByID)
		datasources.PUT("/:id", auth.RequireEditor(), c.Update)
		datasources.DELETE("/:id", auth.RequireAdmin(), c.Delete)

		// Column endpoints
		datasources.GET("/:id/columns", auth.RequireViewer(), c.GetColumns)
		datasources.POST("/:id/columns", auth.RequireEditor(), c.CreateColumns)
		datasources.PUT("/:id/columns/:column_id/role", auth.RequireEditor(), c.UpdateColumnRole)
		datasources.PUT("/:id/columns/roles", auth.RequireEditor(), c.BulkUpdateColumnRoles)
	}
}

// Create godoc
// @Summary Create a new datasource
// @Description Creates a new datasource. Two modes: (1) Database types (postgresql/mysql) - use JSON with connection_config, system fetches data and saves as CSV. (2) File types (csv/parquet) - use multipart/form-data with file upload.
// @Tags Datasources
// @Accept json
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param datasource body dto.CreateDatasourceRequest false "For database types (postgresql/mysql) - JSON body with connection_config"
// @Param collection_id formData string false "For file types (csv/parquet) - Collection ID (UUID)"
// @Param name formData string false "For file types (csv/parquet) - Datasource name"
// @Param description formData string false "For file types (csv/parquet) - Datasource description"
// @Param type formData string false "For file types (csv/parquet) - Must be 'csv' or 'parquet'"
// @Param file formData file false "For file types (csv/parquet) - Required data file"
// @Success 201 {object} response.Response{data=dto.DatasourceResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/datasources [post]
func (c *DatasourceController) Create(ctx *gin.Context) {
	// Check content type to determine if it's JSON or multipart/form-data
	contentType := ctx.GetHeader("Content-Type")

	var req dto.CreateDatasourceRequest

	if contentType == "application/json" {
		// Handle JSON request (for database types)
		if err := ctx.ShouldBindJSON(&req); err != nil {
			response.BadRequest(ctx, err.Error())
			return
		}
	} else {
		// Handle multipart/form-data
		if err := ctx.ShouldBind(&req); err != nil {
			response.BadRequest(ctx, err.Error())
			return
		}

		// Parse connection_config if provided as JSON string
		if connConfigStr := ctx.PostForm("connection_config"); connConfigStr != "" {
			var connConfig dto.ConnectionConfigRequest
			if err := json.Unmarshal([]byte(connConfigStr), &connConfig); err == nil {
				req.ConnectionConfig = &connConfig
			}
		}
	}

	username := auth.GetUsername(ctx)

	// For file-based types (csv/parquet), file is required
	if req.Type == "csv" || req.Type == "parquet" {
		file, header, fileErr := ctx.Request.FormFile("file")
		if fileErr != nil {
			response.BadRequest(ctx, "file is required for csv/parquet datasource types")
			return
		}
		defer file.Close()

		fileData, readErr := io.ReadAll(file)
		if readErr != nil {
			response.BadRequest(ctx, "failed to read file")
			return
		}

		result, err := c.datasourceService.Create(&req, header.Filename, fileData, username)
		if err != nil {
			handleError(ctx, err)
			return
		}
		response.Created(ctx, result)
		return
	}

	// For database types (postgresql/mysql), connection_config is required
	if req.Type == "postgresql" || req.Type == "mysql" {
		if req.ConnectionConfig == nil {
			response.BadRequest(ctx, "connection_config is required for database datasource types")
			return
		}

		result, err := c.datasourceService.Create(&req, "", nil, username)
		if err != nil {
			handleError(ctx, err)
			return
		}
		response.Created(ctx, result)
		return
	}

	response.BadRequest(ctx, "invalid datasource type")
}

// List godoc
// @Summary List datasources
// @Description Retrieves a paginated list of datasources
// @Tags Datasources
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param collection_id query string false "Filter by collection ID (UUID)"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param search query string false "Search term"
// @Success 200 {object} response.Response{data=dto.DatasourceListResponse}
// @Failure 401 {object} response.Response
// @Router /api/datasources [get]
func (c *DatasourceController) List(ctx *gin.Context) {
	var params dto.ListParams
	if err := ctx.ShouldBindQuery(&params); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	var collectionID *string
	if cidStr := ctx.Query("collection_id"); cidStr != "" {
		collectionID = &cidStr
	}

	result, err := c.datasourceService.List(collectionID, &params)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// GetByID godoc
// @Summary Get datasource by ID
// @Description Retrieves a datasource by its ID with columns
// @Tags Datasources
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Datasource ID (UUID)"
// @Success 200 {object} response.Response{data=dto.DatasourceDetailResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/datasources/{id} [get]
func (c *DatasourceController) GetByID(ctx *gin.Context) {
	id := ctx.Param("id")

	result, err := c.datasourceService.GetByID(id)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// Update godoc
// @Summary Update a datasource
// @Description Updates an existing datasource
// @Tags Datasources
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Datasource ID (UUID)"
// @Param datasource body dto.UpdateDatasourceRequest true "Datasource update data"
// @Success 200 {object} response.Response{data=dto.DatasourceResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/datasources/{id} [put]
func (c *DatasourceController) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var req dto.UpdateDatasourceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.datasourceService.Update(id, &req)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// Delete godoc
// @Summary Delete a datasource
// @Description Deletes a datasource (admin only)
// @Tags Datasources
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Datasource ID (UUID)"
// @Success 204 "No Content"
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/datasources/{id} [delete]
func (c *DatasourceController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := c.datasourceService.Delete(id); err != nil {
		handleError(ctx, err)
		return
	}

	response.NoContent(ctx)
}

// GetColumns godoc
// @Summary Get datasource columns
// @Description Retrieves all columns for a datasource
// @Tags Datasources
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Datasource ID (UUID)"
// @Success 200 {object} response.Response{data=[]dto.ColumnResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/datasources/{id}/columns [get]
func (c *DatasourceController) GetColumns(ctx *gin.Context) {
	id := ctx.Param("id")

	result, err := c.columnService.GetByDatasourceID(id)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// CreateColumns godoc
// @Summary Create columns for a datasource
// @Description Creates multiple columns for a datasource
// @Tags Datasources
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Datasource ID (UUID)"
// @Param columns body dto.CreateColumnsRequest true "Columns data"
// @Success 201 {object} response.Response{data=[]dto.ColumnResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/datasources/{id}/columns [post]
func (c *DatasourceController) CreateColumns(ctx *gin.Context) {
	id := ctx.Param("id")

	var req dto.CreateColumnsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.columnService.CreateColumns(id, &req)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Created(ctx, result)
}

// UpdateColumnRole godoc
// @Summary Update column role
// @Description Updates the role of a column
// @Tags Datasources
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Datasource ID (UUID)"
// @Param column_id path string true "Column ID (UUID)"
// @Param role body dto.UpdateColumnRoleRequest true "Role data"
// @Success 200 {object} response.Response{data=dto.ColumnResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/datasources/{id}/columns/{column_id}/role [put]
func (c *DatasourceController) UpdateColumnRole(ctx *gin.Context) {
	id := ctx.Param("id")
	columnID := ctx.Param("column_id")

	var req dto.UpdateColumnRoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.columnService.UpdateRole(id, columnID, req.Role)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}

// BulkUpdateColumnRoles godoc
// @Summary Bulk update column roles
// @Description Updates the roles of multiple columns
// @Tags Datasources
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Datasource ID (UUID)"
// @Param roles body dto.BulkUpdateColumnRolesRequest true "Roles data"
// @Success 200 {object} response.Response{data=[]dto.ColumnResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/datasources/{id}/columns/roles [put]
func (c *DatasourceController) BulkUpdateColumnRoles(ctx *gin.Context) {
	id := ctx.Param("id")

	var req dto.BulkUpdateColumnRolesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	result, err := c.columnService.BulkUpdateRoles(id, &req)
	if err != nil {
		handleError(ctx, err)
		return
	}

	response.Success(ctx, result)
}
