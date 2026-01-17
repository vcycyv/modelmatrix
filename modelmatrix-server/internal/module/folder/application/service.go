package application

import (
	"fmt"
	"strings"

	"modelmatrix-server/internal/module/folder/domain"
	"modelmatrix-server/internal/module/folder/repository"
	"modelmatrix-server/pkg/logger"

	"gorm.io/gorm"
)

// ModelDeleter interface for model cascade operations (to avoid circular imports)
type ModelDeleter interface {
	DeleteByFolderID(folderID string) error
	DeleteByProjectID(projectID string) error
}

// BuildDeleter interface for build cascade operations (to avoid circular imports)
type BuildDeleter interface {
	DeleteByFolderID(folderID string) error
	DeleteByProjectID(projectID string) error
}

// FolderService defines the interface for folder application service
type FolderService interface {
	// Folder operations
	CreateFolder(name, description string, parentID *string, createdBy string) (*domain.Folder, error)
	GetFolder(id string) (*domain.Folder, error)
	UpdateFolder(id, name, description string) (*domain.Folder, error)
	DeleteFolder(id string, force bool) error
	GetChildren(parentID string) ([]domain.Folder, error)
	GetRootFolders() ([]domain.Folder, error)
	GetFolderContentsCount(id string) (*domain.FolderContentsCount, error)

	// Project operations
	CreateProject(name, description string, folderID *string, createdBy string) (*domain.Project, error)
	GetProject(id string) (*domain.Project, error)
	UpdateProject(id, name, description string) (*domain.Project, error)
	DeleteProject(id string, force bool) error
	GetProjectsInFolder(folderID string) ([]domain.Project, error)
	GetRootProjects() ([]domain.Project, error)

	// Association operations
	GetBuildsInFolder(folderID string) ([]string, error)
	GetModelsInFolder(folderID string) ([]string, error)
	GetBuildsInProject(projectID string) ([]string, error)
	GetModelsInProject(projectID string) ([]string, error)
	AddBuildToFolder(buildID, folderID string) error
	AddBuildToProject(buildID, projectID string) error

	// Dependency injection for cascade delete (avoids circular imports)
	SetModelDeleter(deleter ModelDeleter)
	SetBuildDeleter(deleter BuildDeleter)
}

// FolderServiceImpl implements FolderService
type FolderServiceImpl struct {
	db           *gorm.DB
	folderRepo   repository.FolderRepository
	projectRepo  repository.ProjectRepository
	modelDeleter ModelDeleter
	buildDeleter BuildDeleter
}

// NewFolderService creates a new folder service
func NewFolderService(
	db *gorm.DB,
	folderRepo repository.FolderRepository,
	projectRepo repository.ProjectRepository,
) FolderService {
	return &FolderServiceImpl{
		db:          db,
		folderRepo:  folderRepo,
		projectRepo: projectRepo,
	}
}

// SetModelDeleter sets the model deleter (called after services are initialized)
func (s *FolderServiceImpl) SetModelDeleter(deleter ModelDeleter) {
	s.modelDeleter = deleter
}

// SetBuildDeleter sets the build deleter (called after services are initialized)
func (s *FolderServiceImpl) SetBuildDeleter(deleter BuildDeleter) {
	s.buildDeleter = deleter
}

// ==================== Folder Operations ====================

// CreateFolder creates a new folder
func (s *FolderServiceImpl) CreateFolder(name, description string, parentID *string, createdBy string) (*domain.Folder, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrFolderNameEmpty
	}

	// Check for duplicate name
	existing, err := s.folderRepo.GetByParentIDAndName(parentID, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrFolderNameExists
	}

	// Build path and depth
	var path string
	var depth int

	if parentID != nil {
		parent, err := s.folderRepo.GetByID(*parentID)
		if err != nil {
			return nil, domain.ErrInvalidParentFolder
		}
		path = parent.Path
		depth = parent.Depth + 1
	}

	folder := &domain.Folder{
		Name:        name,
		Description: description,
		ParentID:    parentID,
		Depth:       depth,
		CreatedBy:   createdBy,
	}

	if err := s.folderRepo.Create(folder); err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	// Update path with generated ID
	if parentID != nil {
		folder.Path = path + "/" + folder.ID
	} else {
		folder.Path = "/" + folder.ID
	}

	if err := s.folderRepo.UpdatePath(folder.ID, folder.Path); err != nil {
		return nil, fmt.Errorf("failed to update folder path: %w", err)
	}

	return folder, nil
}

// GetFolder retrieves a folder by ID
func (s *FolderServiceImpl) GetFolder(id string) (*domain.Folder, error) {
	return s.folderRepo.GetByID(id)
}

// UpdateFolder updates a folder's name and description
func (s *FolderServiceImpl) UpdateFolder(id, name, description string) (*domain.Folder, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrFolderNameEmpty
	}

	folder, err := s.folderRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Check for duplicate name if name changed
	if folder.Name != name {
		existing, err := s.folderRepo.GetByParentIDAndName(folder.ParentID, name)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != id {
			return nil, domain.ErrFolderNameExists
		}
	}

	folder.Name = name
	folder.Description = description

	if err := s.folderRepo.Update(folder); err != nil {
		return nil, fmt.Errorf("failed to update folder: %w", err)
	}

	return folder, nil
}

// DeleteFolder deletes a folder
func (s *FolderServiceImpl) DeleteFolder(id string, force bool) error {
	folder, err := s.folderRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Check for children
	childCount, err := s.folderRepo.CountChildren(id)
	if err != nil {
		return fmt.Errorf("failed to check children: %w", err)
	}
	if childCount > 0 && !force {
		return domain.ErrFolderHasChildren
	}

	// Check for projects
	projectCount, err := s.folderRepo.CountProjects(id)
	if err != nil {
		return fmt.Errorf("failed to check projects: %w", err)
	}
	if projectCount > 0 && !force {
		return domain.ErrFolderHasProjects
	}

	if force {
		if err := s.cascadeDeleteFolder(folder); err != nil {
			return err
		}
	}

	// Delete the folder itself
	if err := s.folderRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	return nil
}

// cascadeDeleteFolder handles cascade deletion of folder contents
// Each service (Model, Build) handles its own cleanup including MinIO files
func (s *FolderServiceImpl) cascadeDeleteFolder(folder *domain.Folder) error {
	pathPattern := folder.Path + "/%"

	// Get all descendant folder IDs
	descendantFolderIDs, err := s.folderRepo.GetDescendantFolderIDs(pathPattern)
	if err != nil {
		return fmt.Errorf("failed to get descendant folder IDs: %w", err)
	}

	// Get project IDs in descendant folders
	var descendantProjectIDs []string
	for _, folderID := range descendantFolderIDs {
		projects, err := s.projectRepo.GetByFolderID(folderID)
		if err != nil {
			logger.Warn("Failed to get projects in folder %s: %v", folderID, err)
			continue
		}
		for _, p := range projects {
			descendantProjectIDs = append(descendantProjectIDs, p.ID)
		}
	}

	// Delete models and builds in descendant folders (each service handles its own MinIO cleanup)
	for _, folderID := range descendantFolderIDs {
		if s.modelDeleter != nil {
			if err := s.modelDeleter.DeleteByFolderID(folderID); err != nil {
				logger.Warn("Failed to delete models in folder %s: %v", folderID, err)
			}
		}
		if s.buildDeleter != nil {
			if err := s.buildDeleter.DeleteByFolderID(folderID); err != nil {
				logger.Warn("Failed to delete builds in folder %s: %v", folderID, err)
			}
		}
	}

	// Delete models and builds in descendant projects
	for _, projectID := range descendantProjectIDs {
		if s.modelDeleter != nil {
			if err := s.modelDeleter.DeleteByProjectID(projectID); err != nil {
				logger.Warn("Failed to delete models in project %s: %v", projectID, err)
			}
		}
		if s.buildDeleter != nil {
			if err := s.buildDeleter.DeleteByProjectID(projectID); err != nil {
				logger.Warn("Failed to delete builds in project %s: %v", projectID, err)
			}
		}
	}

	// Delete projects in descendant folders
	if err := s.projectRepo.DeleteByFolderIDs(descendantFolderIDs); err != nil {
		return fmt.Errorf("failed to delete projects in descendant folders: %w", err)
	}

	// Delete descendant folders
	if err := s.folderRepo.DeleteDescendants(pathPattern); err != nil {
		return fmt.Errorf("failed to delete descendant folders: %w", err)
	}

	// Now handle this folder's direct content
	projects, err := s.projectRepo.GetByFolderID(folder.ID)
	if err != nil {
		return fmt.Errorf("failed to get projects in folder: %w", err)
	}

	// Delete models and builds in this folder
	if s.modelDeleter != nil {
		if err := s.modelDeleter.DeleteByFolderID(folder.ID); err != nil {
			logger.Warn("Failed to delete models in folder %s: %v", folder.ID, err)
		}
	}
	if s.buildDeleter != nil {
		if err := s.buildDeleter.DeleteByFolderID(folder.ID); err != nil {
			logger.Warn("Failed to delete builds in folder %s: %v", folder.ID, err)
		}
	}

	// Delete models and builds in this folder's projects
	for _, p := range projects {
		if s.modelDeleter != nil {
			if err := s.modelDeleter.DeleteByProjectID(p.ID); err != nil {
				logger.Warn("Failed to delete models in project %s: %v", p.ID, err)
			}
		}
		if s.buildDeleter != nil {
			if err := s.buildDeleter.DeleteByProjectID(p.ID); err != nil {
				logger.Warn("Failed to delete builds in project %s: %v", p.ID, err)
			}
		}
	}

	// Delete projects in this folder
	if err := s.projectRepo.DeleteByFolderIDs([]string{folder.ID}); err != nil {
		return fmt.Errorf("failed to delete projects in folder: %w", err)
	}

	return nil
}

// GetChildren retrieves direct children of a folder
func (s *FolderServiceImpl) GetChildren(parentID string) ([]domain.Folder, error) {
	return s.folderRepo.GetChildren(parentID)
}

// GetRootFolders retrieves all root folders
func (s *FolderServiceImpl) GetRootFolders() ([]domain.Folder, error) {
	return s.folderRepo.GetRootFolders()
}

// GetFolderContentsCount returns counts of items in a folder and its descendants
func (s *FolderServiceImpl) GetFolderContentsCount(id string) (*domain.FolderContentsCount, error) {
	return s.folderRepo.GetContentsCount(id)
}

// ==================== Project Operations ====================

// CreateProject creates a new project
func (s *FolderServiceImpl) CreateProject(name, description string, folderID *string, createdBy string) (*domain.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrProjectNameEmpty
	}

	// Validate folder exists if specified
	if folderID != nil {
		if _, err := s.folderRepo.GetByID(*folderID); err != nil {
			return nil, domain.ErrFolderNotFound
		}
	}

	// Check for duplicate name
	existing, err := s.projectRepo.GetByFolderIDAndName(folderID, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrProjectNameExists
	}

	project := &domain.Project{
		Name:        name,
		Description: description,
		FolderID:    folderID,
		CreatedBy:   createdBy,
	}

	if err := s.projectRepo.Create(project); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return project, nil
}

// GetProject retrieves a project by ID
func (s *FolderServiceImpl) GetProject(id string) (*domain.Project, error) {
	return s.projectRepo.GetByID(id)
}

// UpdateProject updates a project's name and description
func (s *FolderServiceImpl) UpdateProject(id, name, description string) (*domain.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrProjectNameEmpty
	}

	project, err := s.projectRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Check for duplicate name if name changed
	if project.Name != name {
		existing, err := s.projectRepo.GetByFolderIDAndName(project.FolderID, name)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != id {
			return nil, domain.ErrProjectNameExists
		}
	}

	project.Name = name
	project.Description = description

	if err := s.projectRepo.Update(project); err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return project, nil
}

// DeleteProject deletes a project
func (s *FolderServiceImpl) DeleteProject(id string, force bool) error {
	_, err := s.projectRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Check for models
	modelCount, err := s.projectRepo.CountModels(id)
	if err != nil {
		return fmt.Errorf("failed to check models: %w", err)
	}
	if modelCount > 0 && !force {
		return domain.ErrProjectHasModels
	}

	// Check for builds
	buildCount, err := s.projectRepo.CountBuilds(id)
	if err != nil {
		return fmt.Errorf("failed to check builds: %w", err)
	}
	if buildCount > 0 && !force {
		return domain.ErrProjectHasBuilds
	}

	if force {
		// Delete models (service handles MinIO cleanup)
		if s.modelDeleter != nil {
			if err := s.modelDeleter.DeleteByProjectID(id); err != nil {
				logger.Warn("Failed to delete models in project %s: %v", id, err)
			}
		}

		// Delete builds
		if s.buildDeleter != nil {
			if err := s.buildDeleter.DeleteByProjectID(id); err != nil {
				logger.Warn("Failed to delete builds in project %s: %v", id, err)
			}
		}
	}

	// Delete the project
	if err := s.projectRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// GetProjectsInFolder retrieves all projects in a folder
func (s *FolderServiceImpl) GetProjectsInFolder(folderID string) ([]domain.Project, error) {
	return s.projectRepo.GetByFolderID(folderID)
}

// GetRootProjects retrieves all root projects
func (s *FolderServiceImpl) GetRootProjects() ([]domain.Project, error) {
	return s.projectRepo.GetRootProjects()
}

// ==================== Association Operations ====================

// GetBuildsInFolder returns build IDs in a folder
func (s *FolderServiceImpl) GetBuildsInFolder(folderID string) ([]string, error) {
	var ids []string
	err := s.db.Table("model_builds").Where("folder_id = ?", folderID).Pluck("id", &ids).Error
	return ids, err
}

// GetModelsInFolder returns model IDs in a folder
func (s *FolderServiceImpl) GetModelsInFolder(folderID string) ([]string, error) {
	var ids []string
	err := s.db.Table("models").Where("folder_id = ?", folderID).Pluck("id", &ids).Error
	return ids, err
}

// GetBuildsInProject returns build IDs in a project
func (s *FolderServiceImpl) GetBuildsInProject(projectID string) ([]string, error) {
	var ids []string
	err := s.db.Table("model_builds").Where("project_id = ?", projectID).Pluck("id", &ids).Error
	return ids, err
}

// GetModelsInProject returns model IDs in a project
func (s *FolderServiceImpl) GetModelsInProject(projectID string) ([]string, error) {
	var ids []string
	err := s.db.Table("models").Where("project_id = ?", projectID).Pluck("id", &ids).Error
	return ids, err
}

// AddBuildToFolder associates a build with a folder
func (s *FolderServiceImpl) AddBuildToFolder(buildID, folderID string) error {
	// Verify folder exists
	if _, err := s.folderRepo.GetByID(folderID); err != nil {
		return err
	}
	// Update build to set folder_id
	return s.db.Exec("UPDATE model_builds SET folder_id = ?, project_id = NULL WHERE id = ?", folderID, buildID).Error
}

// AddBuildToProject associates a build with a project
func (s *FolderServiceImpl) AddBuildToProject(buildID, projectID string) error {
	// Verify project exists
	if _, err := s.projectRepo.GetByID(projectID); err != nil {
		return err
	}
	// Update build to set project_id
	return s.db.Exec("UPDATE model_builds SET project_id = ?, folder_id = NULL WHERE id = ?", projectID, buildID).Error
}
