package api

import (
	"modelmatrix-server/internal/infrastructure/auth"
	"modelmatrix-server/internal/infrastructure/folderservice"
	buildApp "modelmatrix-server/internal/module/build/application"
	invApp "modelmatrix-server/internal/module/inventory/application"
	"modelmatrix-server/pkg/response"

	"github.com/gin-gonic/gin"
)

// FolderController handles folder and project HTTP requests
type FolderController struct {
	folderService folderservice.FolderService
	buildService  buildApp.BuildService
	modelService  invApp.ModelService
}

// NewFolderController creates a new folder controller
func NewFolderController(
	folderService folderservice.FolderService,
	buildService buildApp.BuildService,
	modelService invApp.ModelService,
) *FolderController {
	return &FolderController{
		folderService: folderService,
		buildService:  buildService,
		modelService:  modelService,
	}
}

// RegisterRoutes registers folder and project routes
func (c *FolderController) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Folder routes
	folders := router.Group("/folders")
	folders.Use(authMiddleware)
	{
		folders.GET("", auth.RequireViewer(), c.ListRootFolders)
		folders.POST("", auth.RequireEditor(), c.CreateFolder)
		folders.GET("/:id", auth.RequireViewer(), c.GetFolder)
		folders.PUT("/:id", auth.RequireEditor(), c.UpdateFolder)
		folders.DELETE("/:id", auth.RequireAdmin(), c.DeleteFolder)
		folders.GET("/:id/children", auth.RequireViewer(), c.GetFolderChildren)
		folders.GET("/:id/projects", auth.RequireViewer(), c.GetProjectsInFolder)
		folders.GET("/:id/builds", auth.RequireViewer(), c.GetBuildsInFolder)
		folders.GET("/:id/models", auth.RequireViewer(), c.GetModelsInFolder)
		folders.POST("/:id/builds", auth.RequireEditor(), c.AddBuildToFolder)
	}

	// Project routes
	projects := router.Group("/projects")
	projects.Use(authMiddleware)
	{
		projects.GET("", auth.RequireViewer(), c.ListProjects)
		projects.POST("", auth.RequireEditor(), c.CreateProject)
		projects.GET("/:id", auth.RequireViewer(), c.GetProject)
		projects.PUT("/:id", auth.RequireEditor(), c.UpdateProject)
		projects.DELETE("/:id", auth.RequireAdmin(), c.DeleteProject)
		projects.GET("/:id/builds", auth.RequireViewer(), c.GetBuildsInProject)
		projects.GET("/:id/models", auth.RequireViewer(), c.GetModelsInProject)
		projects.POST("/:id/builds", auth.RequireEditor(), c.AddBuildToProject)
	}
}

// ==================== Folder Handlers ====================

// CreateFolderRequest represents the request to create a folder
type CreateFolderRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	ParentID    *string `json:"parent_id"`
}

// UpdateFolderRequest represents the request to update a folder
type UpdateFolderRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// ListRootFolders godoc
// @Summary List root folders
// @Description Returns all folders that have no parent (root level folders)
// @Tags folders
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} response.Response{data=[]folderservice.Folder}
// @Failure 401 {object} response.Response
// @Router /api/folders [get]
func (c *FolderController) ListRootFolders(ctx *gin.Context) {
	folders, err := c.folderService.GetRootFolders()
	if err != nil {
		response.InternalError(ctx, err.Error())
		return
	}
	response.Success(ctx, folders)
}

// CreateFolder godoc
// @Summary Create a folder
// @Description Creates a new folder, optionally under a parent folder
// @Tags folders
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param folder body CreateFolderRequest true "Folder data"
// @Success 201 {object} response.Response{data=folderservice.Folder}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/folders [post]
func (c *FolderController) CreateFolder(ctx *gin.Context) {
	var req CreateFolderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	username := auth.GetUsername(ctx)
	folder, err := c.folderService.CreateFolder(req.Name, req.Description, req.ParentID, username)
	if err != nil {
		handleFolderError(ctx, err)
		return
	}

	response.Created(ctx, folder)
}

// GetFolder godoc
// @Summary Get a folder
// @Description Returns a folder by its ID
// @Tags folders
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Folder ID (UUID)"
// @Success 200 {object} response.Response{data=folderservice.Folder}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/folders/{id} [get]
func (c *FolderController) GetFolder(ctx *gin.Context) {
	id := ctx.Param("id")

	folder, err := c.folderService.GetFolder(id)
	if err != nil {
		handleFolderError(ctx, err)
		return
	}

	response.Success(ctx, folder)
}

// UpdateFolder godoc
// @Summary Update a folder
// @Description Updates a folder's name and description
// @Tags folders
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Folder ID (UUID)"
// @Param folder body UpdateFolderRequest true "Folder update data"
// @Success 200 {object} response.Response{data=folderservice.Folder}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/folders/{id} [put]
func (c *FolderController) UpdateFolder(ctx *gin.Context) {
	id := ctx.Param("id")

	var req UpdateFolderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	folder, err := c.folderService.UpdateFolder(id, req.Name, req.Description)
	if err != nil {
		handleFolderError(ctx, err)
		return
	}

	response.Success(ctx, folder)
}

// DeleteFolder godoc
// @Summary Delete a folder
// @Description Deletes a folder. Use force=true to delete folders with children.
// @Tags folders
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Folder ID (UUID)"
// @Param force query bool false "Force delete even if folder has children"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/folders/{id} [delete]
func (c *FolderController) DeleteFolder(ctx *gin.Context) {
	id := ctx.Param("id")
	force := ctx.Query("force") == "true"

	if err := c.folderService.DeleteFolder(id, force); err != nil {
		handleFolderError(ctx, err)
		return
	}

	response.Success(ctx, gin.H{"message": "folder deleted"})
}

// GetFolderChildren godoc
// @Summary Get folder children
// @Description Returns all direct child folders of a folder
// @Tags folders
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Folder ID (UUID)"
// @Success 200 {object} response.Response{data=[]folderservice.Folder}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/folders/{id}/children [get]
func (c *FolderController) GetFolderChildren(ctx *gin.Context) {
	id := ctx.Param("id")

	children, err := c.folderService.GetChildren(id)
	if err != nil {
		handleFolderError(ctx, err)
		return
	}

	response.Success(ctx, children)
}

// GetProjectsInFolder godoc
// @Summary Get projects in folder
// @Description Returns all projects directly in a folder
// @Tags folders
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Folder ID (UUID)"
// @Success 200 {object} response.Response{data=[]folderservice.Project}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/folders/{id}/projects [get]
func (c *FolderController) GetProjectsInFolder(ctx *gin.Context) {
	id := ctx.Param("id")

	projects, err := c.folderService.GetProjectsInFolder(id)
	if err != nil {
		handleFolderError(ctx, err)
		return
	}

	response.Success(ctx, projects)
}

// ==================== Project Handlers ====================

// CreateProjectRequest represents the request to create a project
type CreateProjectRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	FolderID    *string `json:"folder_id"`
}

// UpdateProjectRequest represents the request to update a project
type UpdateProjectRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// ListProjects godoc
// @Summary List projects
// @Description Returns projects. Use root=true to get only root projects (not in any folder).
// @Tags projects
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param root query bool false "If true, return only root projects"
// @Success 200 {object} response.Response{data=[]folderservice.Project}
// @Failure 401 {object} response.Response
// @Router /api/projects [get]
func (c *FolderController) ListProjects(ctx *gin.Context) {
	root := ctx.Query("root")

	if root == "true" {
		// Get root projects (not in any folder)
		projects, err := c.folderService.GetRootProjects()
		if err != nil {
			response.InternalError(ctx, err.Error())
			return
		}
		response.Success(ctx, projects)
		return
	}

	// If no filter, could return all projects or error
	// For now, return root projects
	projects, err := c.folderService.GetRootProjects()
	if err != nil {
		response.InternalError(ctx, err.Error())
		return
	}
	response.Success(ctx, projects)
}

// CreateProject godoc
// @Summary Create a project
// @Description Creates a new project, optionally in a folder
// @Tags projects
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param project body CreateProjectRequest true "Project data"
// @Success 201 {object} response.Response{data=folderservice.Project}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/projects [post]
func (c *FolderController) CreateProject(ctx *gin.Context) {
	var req CreateProjectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	username := auth.GetUsername(ctx)
	project, err := c.folderService.CreateProject(req.Name, req.Description, req.FolderID, username)
	if err != nil {
		handleFolderError(ctx, err)
		return
	}

	response.Created(ctx, project)
}

// GetProject godoc
// @Summary Get a project
// @Description Returns a project by its ID
// @Tags projects
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Project ID (UUID)"
// @Success 200 {object} response.Response{data=folderservice.Project}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/projects/{id} [get]
func (c *FolderController) GetProject(ctx *gin.Context) {
	id := ctx.Param("id")

	project, err := c.folderService.GetProject(id)
	if err != nil {
		handleFolderError(ctx, err)
		return
	}

	response.Success(ctx, project)
}

// UpdateProject godoc
// @Summary Update a project
// @Description Updates a project's name and description
// @Tags projects
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Project ID (UUID)"
// @Param project body UpdateProjectRequest true "Project update data"
// @Success 200 {object} response.Response{data=folderservice.Project}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/projects/{id} [put]
func (c *FolderController) UpdateProject(ctx *gin.Context) {
	id := ctx.Param("id")

	var req UpdateProjectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	project, err := c.folderService.UpdateProject(id, req.Name, req.Description)
	if err != nil {
		handleFolderError(ctx, err)
		return
	}

	response.Success(ctx, project)
}

// DeleteProject godoc
// @Summary Delete a project
// @Description Deletes a project. Use force=true to delete projects with builds/models.
// @Tags projects
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Project ID (UUID)"
// @Param force query bool false "Force delete even if project has builds/models"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/projects/{id} [delete]
func (c *FolderController) DeleteProject(ctx *gin.Context) {
	id := ctx.Param("id")
	force := ctx.Query("force") == "true"

	if err := c.folderService.DeleteProject(id, force); err != nil {
		handleFolderError(ctx, err)
		return
	}

	response.Success(ctx, gin.H{"message": "project deleted"})
}

// GetBuildsInProject godoc
// @Summary Get builds in project
// @Description Returns all model builds in a project
// @Tags projects
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Project ID (UUID)"
// @Success 200 {object} response.Response{data=[]dto.BuildResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/projects/{id}/builds [get]
func (c *FolderController) GetBuildsInProject(ctx *gin.Context) {
	id := ctx.Param("id")

	// Get build IDs from folder service
	buildIDs, err := c.folderService.GetBuildsInProject(id)
	if err != nil {
		handleFolderError(ctx, err)
		return
	}

	// Fetch full build objects - initialize as empty slice, not nil
	builds := make([]interface{}, 0)
	for _, buildID := range buildIDs {
		build, err := c.buildService.GetByID(buildID)
		if err == nil && build != nil {
			builds = append(builds, build)
		}
	}

	response.Success(ctx, builds)
}

// GetModelsInProject godoc
// @Summary Get models in project
// @Description Returns all models in a project
// @Tags projects
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Project ID (UUID)"
// @Success 200 {object} response.Response{data=[]dto.ModelResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/projects/{id}/models [get]
func (c *FolderController) GetModelsInProject(ctx *gin.Context) {
	id := ctx.Param("id")

	// Get model IDs from folder service
	modelIDs, err := c.folderService.GetModelsInProject(id)
	if err != nil {
		handleFolderError(ctx, err)
		return
	}

	// Fetch full model objects - initialize as empty slice, not nil
	models := make([]interface{}, 0)
	for _, modelID := range modelIDs {
		model, err := c.modelService.GetByID(modelID)
		if err == nil && model != nil {
			models = append(models, model)
		}
	}

	response.Success(ctx, models)
}

// AddBuildToProjectRequest represents the request to add a build to a project
type AddBuildToProjectRequest struct {
	BuildID string `json:"build_id" binding:"required"`
}

// AddBuildToProject godoc
// @Summary Add build to project
// @Description Associates a model build with a project
// @Tags projects
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Project ID (UUID)"
// @Param request body AddBuildToProjectRequest true "Build ID to add"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/projects/{id}/builds [post]
func (c *FolderController) AddBuildToProject(ctx *gin.Context) {
	id := ctx.Param("id")

	var req AddBuildToProjectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	if err := c.folderService.AddBuildToProject(req.BuildID, id); err != nil {
		handleFolderError(ctx, err)
		return
	}

	response.Success(ctx, gin.H{"message": "build added to project"})
}

// ==================== Folder Builds/Models Handlers ====================

// GetBuildsInFolder godoc
// @Summary Get builds in folder
// @Description Returns all model builds directly in a folder (not in projects)
// @Tags folders
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Folder ID (UUID)"
// @Success 200 {object} response.Response{data=[]dto.BuildResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/folders/{id}/builds [get]
func (c *FolderController) GetBuildsInFolder(ctx *gin.Context) {
	id := ctx.Param("id")

	// Get build IDs from folder service
	buildIDs, err := c.folderService.GetBuildsInFolder(id)
	if err != nil {
		handleFolderError(ctx, err)
		return
	}

	// Fetch full build objects - initialize as empty slice, not nil
	builds := make([]interface{}, 0)
	for _, buildID := range buildIDs {
		build, err := c.buildService.GetByID(buildID)
		if err == nil && build != nil {
			builds = append(builds, build)
		}
	}

	response.Success(ctx, builds)
}

// GetModelsInFolder godoc
// @Summary Get models in folder
// @Description Returns all models directly in a folder (not in projects)
// @Tags folders
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Folder ID (UUID)"
// @Success 200 {object} response.Response{data=[]dto.ModelResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/folders/{id}/models [get]
func (c *FolderController) GetModelsInFolder(ctx *gin.Context) {
	id := ctx.Param("id")

	// Get model IDs from folder service
	modelIDs, err := c.folderService.GetModelsInFolder(id)
	if err != nil {
		handleFolderError(ctx, err)
		return
	}

	// Fetch full model objects - initialize as empty slice, not nil
	models := make([]interface{}, 0)
	for _, modelID := range modelIDs {
		model, err := c.modelService.GetByID(modelID)
		if err == nil && model != nil {
			models = append(models, model)
		}
	}

	response.Success(ctx, models)
}

// AddBuildToFolderRequest represents the request to add a build to a folder
type AddBuildToFolderRequest struct {
	BuildID string `json:"build_id" binding:"required"`
}

// AddBuildToFolder godoc
// @Summary Add build to folder
// @Description Associates a model build directly with a folder (not in a project)
// @Tags folders
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Folder ID (UUID)"
// @Param request body AddBuildToFolderRequest true "Build ID to add"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/folders/{id}/builds [post]
func (c *FolderController) AddBuildToFolder(ctx *gin.Context) {
	id := ctx.Param("id")

	var req AddBuildToFolderRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.BadRequest(ctx, err.Error())
		return
	}

	if err := c.folderService.AddBuildToFolder(req.BuildID, id); err != nil {
		handleFolderError(ctx, err)
		return
	}

	response.Success(ctx, gin.H{"message": "build added to folder"})
}

// handleFolderError handles folder service errors
func handleFolderError(ctx *gin.Context, err error) {
	switch err {
	case folderservice.ErrFolderNotFound:
		response.NotFound(ctx, err.Error())
	case folderservice.ErrProjectNotFound:
		response.NotFound(ctx, err.Error())
	case folderservice.ErrFolderNameExists:
		response.Conflict(ctx, err.Error())
	case folderservice.ErrProjectNameExists:
		response.Conflict(ctx, err.Error())
	case folderservice.ErrFolderNameEmpty:
		response.BadRequest(ctx, err.Error())
	case folderservice.ErrProjectNameEmpty:
		response.BadRequest(ctx, err.Error())
	case folderservice.ErrFolderHasChildren:
		response.BadRequest(ctx, err.Error())
	case folderservice.ErrFolderHasProjects:
		response.BadRequest(ctx, err.Error())
	case folderservice.ErrProjectHasModels:
		response.BadRequest(ctx, err.Error())
	case folderservice.ErrProjectHasBuilds:
		response.BadRequest(ctx, err.Error())
	default:
		response.InternalError(ctx, err.Error())
	}
}
