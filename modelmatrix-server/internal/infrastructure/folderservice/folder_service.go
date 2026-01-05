package folderservice

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Domain errors
var (
	ErrFolderNotFound      = errors.New("folder not found")
	ErrFolderNameExists    = errors.New("folder with this name already exists in parent")
	ErrFolderNameEmpty     = errors.New("folder name cannot be empty")
	ErrInvalidParentFolder = errors.New("invalid parent folder")
	ErrFolderHasChildren   = errors.New("folder has children and cannot be deleted")
	ErrFolderHasProjects   = errors.New("folder has projects and cannot be deleted")
	ErrCircularReference   = errors.New("circular folder reference detected")

	ErrProjectNotFound   = errors.New("project not found")
	ErrProjectNameExists = errors.New("project with this name already exists in folder")
	ErrProjectNameEmpty  = errors.New("project name cannot be empty")
	ErrProjectHasModels  = errors.New("project has models and cannot be deleted")
	ErrProjectHasBuilds  = errors.New("project has builds and cannot be deleted")
)

// Folder represents a folder in the hierarchical structure
type Folder struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	ParentID    *string   `json:"parent_id,omitempty"` // nil for root folders
	Path        string    `json:"path"`                // Materialized path: /id1/id2/id3
	Depth       int       `json:"depth"`               // Depth level (0 for root)
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FolderModel is the GORM model for folders
type FolderModel struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string    `gorm:"type:varchar(255);not null"`
	Description string    `gorm:"type:text"`
	ParentID    *string   `gorm:"type:uuid;index:idx_folder_parent"`
	Path        string    `gorm:"type:varchar(1000);not null;index:idx_folder_path"` // For descendant queries
	Depth       int       `gorm:"not null;default:0"`
	CreatedBy   string    `gorm:"type:varchar(255);not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`

	// Self-referential relationship
	Parent   *FolderModel  `gorm:"foreignKey:ParentID"`
	Children []FolderModel `gorm:"foreignKey:ParentID"`
}

// TableName returns the table name for FolderModel
func (FolderModel) TableName() string {
	return "folders"
}

// BeforeCreate generates UUID before creating record
func (f *FolderModel) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	return nil
}

// Project represents a project container that holds model builds and models
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	FolderID    *string   `json:"folder_id,omitempty"` // nil for projects not in any folder
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ProjectModel is the GORM model for projects
type ProjectModel struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string    `gorm:"type:varchar(255);not null"`
	Description string    `gorm:"type:text"`
	FolderID    *string   `gorm:"type:uuid;index:idx_project_folder"`
	CreatedBy   string    `gorm:"type:varchar(255);not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`

	// Relationship
	Folder *FolderModel `gorm:"foreignKey:FolderID"`
}

// TableName returns the table name for ProjectModel
func (ProjectModel) TableName() string {
	return "projects"
}

// BeforeCreate generates UUID before creating record
func (p *ProjectModel) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

// Note: Junction tables (project_models, project_builds, folder_models, folder_builds)
// have been removed. Models and builds now have direct project_id and folder_id columns.

// FolderService defines the interface for folder operations
type FolderService interface {
	// Folder CRUD
	CreateFolder(name, description string, parentID *string, createdBy string) (*Folder, error)
	GetFolder(id string) (*Folder, error)
	UpdateFolder(id, name, description string) (*Folder, error)
	DeleteFolder(id string, force bool) error

	// Hierarchy operations
	GetChildren(parentID string) ([]Folder, error)
	GetRootFolders() ([]Folder, error)
	GetPath(id string) ([]Folder, error)             // Returns path from root to folder
	GetDescendants(id string) ([]Folder, error)      // All nested subfolders
	MoveFolder(id string, newParentID *string) error // Move folder to new parent

	// Project CRUD
	CreateProject(name, description string, folderID *string, createdBy string) (*Project, error)
	GetProject(id string) (*Project, error)
	UpdateProject(id, name, description string) (*Project, error)
	DeleteProject(id string, force bool) error
	MoveProject(id string, newFolderID *string) error

	// Project listing
	GetProjectsInFolder(folderID string) ([]Project, error)
	GetRootProjects() ([]Project, error)                         // Projects not in any folder
	GetAllDescendantProjects(folderID string) ([]Project, error) // Projects in folder and all subfolders

	// Project-Model association
	AddModelToProject(modelID, projectID string) error
	RemoveModelFromProject(modelID string) error
	GetModelProject(modelID string) (*Project, error)
	GetModelsInProject(projectID string) ([]string, error)

	// Project-Build association
	AddBuildToProject(buildID, projectID string) error
	RemoveBuildFromProject(buildID string) error
	GetBuildProject(buildID string) (*Project, error)
	GetBuildsInProject(projectID string) ([]string, error)

	// Folder-Build association (builds directly in folders)
	AddBuildToFolder(buildID, folderID string) error
	RemoveBuildFromFolder(buildID string) error
	GetBuildFolder(buildID string) (*Folder, error)
	GetBuildsInFolder(folderID string) ([]string, error)

	// Folder-Model association (models directly in folders)
	AddModelToFolder(modelID, folderID string) error
	RemoveModelFromFolder(modelID string) error
	GetModelFolder(modelID string) (*Folder, error)
	GetModelsInFolder(folderID string) ([]string, error)

	// Get all models/builds under a folder (through projects AND direct folder associations)
	GetAllDescendantModels(folderID string) ([]string, error) // Model IDs in folder, its projects, and all subfolders
	GetAllDescendantBuilds(folderID string) ([]string, error) // Build IDs in folder, its projects, and all subfolders

	// Search
	SearchFolders(query string) ([]Folder, error)
	SearchProjects(query string) ([]Project, error)
}

// FolderServiceImpl implements FolderService using GORM
type FolderServiceImpl struct {
	db *gorm.DB
}

// NewFolderService creates a new folder service
func NewFolderService(db *gorm.DB) FolderService {
	return &FolderServiceImpl{db: db}
}

// CreateFolder creates a new folder
func (s *FolderServiceImpl) CreateFolder(name, description string, parentID *string, createdBy string) (*Folder, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrFolderNameEmpty
	}

	// Check for duplicate name in same parent
	var count int64
	query := s.db.Model(&FolderModel{}).Where("name = ?", name)
	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}
	if err := query.Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to check folder name: %w", err)
	}
	if count > 0 {
		return nil, ErrFolderNameExists
	}

	// Calculate path and depth
	var path string
	var depth int

	if parentID != nil {
		parent, err := s.getFolderModel(*parentID)
		if err != nil {
			return nil, ErrInvalidParentFolder
		}
		depth = parent.Depth + 1
		// Path will be updated after we have the ID
		path = parent.Path
	} else {
		depth = 0
		path = ""
	}

	folderModel := &FolderModel{
		Name:        name,
		Description: description,
		ParentID:    parentID,
		Path:        path, // Temporary, will update after create
		Depth:       depth,
		CreatedBy:   createdBy,
	}

	if err := s.db.Create(folderModel).Error; err != nil {
		return nil, fmt.Errorf("failed to create folder: %w", err)
	}

	// Update path to include the new folder ID
	newPath := fmt.Sprintf("%s/%s", path, folderModel.ID)
	if path == "" {
		newPath = fmt.Sprintf("/%s", folderModel.ID)
	}
	if err := s.db.Model(folderModel).Update("path", newPath).Error; err != nil {
		return nil, fmt.Errorf("failed to update folder path: %w", err)
	}
	folderModel.Path = newPath

	return s.toDomain(folderModel), nil
}

// GetFolder retrieves a folder by ID
func (s *FolderServiceImpl) GetFolder(id string) (*Folder, error) {
	folderModel, err := s.getFolderModel(id)
	if err != nil {
		return nil, err
	}
	return s.toDomain(folderModel), nil
}

// getFolderModel is an internal helper to get the GORM model
func (s *FolderServiceImpl) getFolderModel(id string) (*FolderModel, error) {
	var folderModel FolderModel
	if err := s.db.Where("id = ?", id).First(&folderModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFolderNotFound
		}
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}
	return &folderModel, nil
}

// UpdateFolder updates folder name and description
func (s *FolderServiceImpl) UpdateFolder(id, name, description string) (*Folder, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrFolderNameEmpty
	}

	folderModel, err := s.getFolderModel(id)
	if err != nil {
		return nil, err
	}

	// Check for duplicate name in same parent (excluding current folder)
	var count int64
	query := s.db.Model(&FolderModel{}).Where("name = ? AND id != ?", name, id)
	if folderModel.ParentID != nil {
		query = query.Where("parent_id = ?", *folderModel.ParentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}
	if err := query.Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to check folder name: %w", err)
	}
	if count > 0 {
		return nil, ErrFolderNameExists
	}

	folderModel.Name = name
	folderModel.Description = description
	if err := s.db.Save(folderModel).Error; err != nil {
		return nil, fmt.Errorf("failed to update folder: %w", err)
	}

	return s.toDomain(folderModel), nil
}

// DeleteFolder deletes a folder
func (s *FolderServiceImpl) DeleteFolder(id string, force bool) error {
	folderModel, err := s.getFolderModel(id)
	if err != nil {
		return err
	}

	// Check for children
	var childCount int64
	if err := s.db.Model(&FolderModel{}).Where("parent_id = ?", id).Count(&childCount).Error; err != nil {
		return fmt.Errorf("failed to check children: %w", err)
	}
	if childCount > 0 && !force {
		return ErrFolderHasChildren
	}

	// Check for projects in folder
	var projectCount int64
	if err := s.db.Model(&ProjectModel{}).Where("folder_id = ?", id).Count(&projectCount).Error; err != nil {
		return fmt.Errorf("failed to check projects: %w", err)
	}
	if projectCount > 0 && !force {
		return ErrFolderHasProjects
	}

	// If force delete, remove all descendants and their projects, unlink models/builds
	if force {
		pathPattern := folderModel.Path + "/%"

		// Unlink models from projects in descendant folders
		if err := s.db.Exec(`
			UPDATE models SET project_id = NULL 
			WHERE project_id IN (SELECT id FROM projects WHERE folder_id IN (SELECT id FROM folders WHERE path LIKE ?))
		`, pathPattern).Error; err != nil {
			return fmt.Errorf("failed to unlink models from descendant projects: %w", err)
		}

		// Unlink builds from projects in descendant folders
		if err := s.db.Exec(`
			UPDATE model_builds SET project_id = NULL 
			WHERE project_id IN (SELECT id FROM projects WHERE folder_id IN (SELECT id FROM folders WHERE path LIKE ?))
		`, pathPattern).Error; err != nil {
			return fmt.Errorf("failed to unlink builds from descendant projects: %w", err)
		}

		// Unlink models directly in descendant folders
		if err := s.db.Exec(`
			UPDATE models SET folder_id = NULL 
			WHERE folder_id IN (SELECT id FROM folders WHERE path LIKE ?)
		`, pathPattern).Error; err != nil {
			return fmt.Errorf("failed to unlink models from descendant folders: %w", err)
		}

		// Unlink builds directly in descendant folders
		if err := s.db.Exec(`
			UPDATE model_builds SET folder_id = NULL 
			WHERE folder_id IN (SELECT id FROM folders WHERE path LIKE ?)
		`, pathPattern).Error; err != nil {
			return fmt.Errorf("failed to unlink builds from descendant folders: %w", err)
		}

		// Delete projects in descendant folders
		if err := s.db.Exec(`
			DELETE FROM projects WHERE folder_id IN (SELECT id FROM folders WHERE path LIKE ?)
		`, pathPattern).Error; err != nil {
			return fmt.Errorf("failed to delete descendant projects: %w", err)
		}

		// Delete all descendant folders
		if err := s.db.Where("path LIKE ?", pathPattern).Delete(&FolderModel{}).Error; err != nil {
			return fmt.Errorf("failed to delete descendants: %w", err)
		}

		// Unlink models from projects in this folder
		if err := s.db.Exec(`
			UPDATE models SET project_id = NULL 
			WHERE project_id IN (SELECT id FROM projects WHERE folder_id = ?)
		`, id).Error; err != nil {
			return fmt.Errorf("failed to unlink models from folder projects: %w", err)
		}

		// Unlink builds from projects in this folder
		if err := s.db.Exec(`
			UPDATE model_builds SET project_id = NULL 
			WHERE project_id IN (SELECT id FROM projects WHERE folder_id = ?)
		`, id).Error; err != nil {
			return fmt.Errorf("failed to unlink builds from folder projects: %w", err)
		}

		// Unlink models directly in this folder
		if err := s.db.Exec("UPDATE models SET folder_id = NULL WHERE folder_id = ?", id).Error; err != nil {
			return fmt.Errorf("failed to unlink models from folder: %w", err)
		}

		// Unlink builds directly in this folder
		if err := s.db.Exec("UPDATE model_builds SET folder_id = NULL WHERE folder_id = ?", id).Error; err != nil {
			return fmt.Errorf("failed to unlink builds from folder: %w", err)
		}

		// Delete projects in this folder
		if err := s.db.Where("folder_id = ?", id).Delete(&ProjectModel{}).Error; err != nil {
			return fmt.Errorf("failed to delete projects: %w", err)
		}
	}

	// Delete the folder itself
	if err := s.db.Delete(folderModel).Error; err != nil {
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	return nil
}

// GetChildren retrieves direct children of a folder
func (s *FolderServiceImpl) GetChildren(parentID string) ([]Folder, error) {
	var folderModels []FolderModel
	if err := s.db.Where("parent_id = ?", parentID).Order("name ASC").Find(&folderModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get children: %w", err)
	}
	return s.toDomainSlice(folderModels), nil
}

// GetRootFolders retrieves all root folders (folders without parent)
func (s *FolderServiceImpl) GetRootFolders() ([]Folder, error) {
	var folderModels []FolderModel
	if err := s.db.Where("parent_id IS NULL").Order("name ASC").Find(&folderModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get root folders: %w", err)
	}
	return s.toDomainSlice(folderModels), nil
}

// GetPath returns the path from root to the specified folder
func (s *FolderServiceImpl) GetPath(id string) ([]Folder, error) {
	folder, err := s.getFolderModel(id)
	if err != nil {
		return nil, err
	}

	// Extract IDs from path
	pathParts := strings.Split(strings.Trim(folder.Path, "/"), "/")
	if len(pathParts) == 0 || (len(pathParts) == 1 && pathParts[0] == "") {
		return []Folder{}, nil
	}

	var folderModels []FolderModel
	if err := s.db.Where("id IN ?", pathParts).Find(&folderModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get path folders: %w", err)
	}

	// Sort by depth to maintain order
	result := make([]Folder, len(folderModels))
	folderMap := make(map[string]*FolderModel)
	for i := range folderModels {
		folderMap[folderModels[i].ID] = &folderModels[i]
	}

	for i, partID := range pathParts {
		if fm, ok := folderMap[partID]; ok {
			result[i] = *s.toDomain(fm)
		}
	}

	return result, nil
}

// GetDescendants retrieves all descendant folders
func (s *FolderServiceImpl) GetDescendants(id string) ([]Folder, error) {
	folder, err := s.getFolderModel(id)
	if err != nil {
		return nil, err
	}

	var folderModels []FolderModel
	pathPattern := folder.Path + "/%"
	if err := s.db.Where("path LIKE ?", pathPattern).Order("depth ASC, name ASC").Find(&folderModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get descendants: %w", err)
	}

	return s.toDomainSlice(folderModels), nil
}

// MoveFolder moves a folder to a new parent
func (s *FolderServiceImpl) MoveFolder(id string, newParentID *string) error {
	folder, err := s.getFolderModel(id)
	if err != nil {
		return err
	}

	oldPath := folder.Path

	// Calculate new path and depth
	var newPath string
	var newDepth int

	if newParentID != nil {
		// Check that we're not moving to a descendant (circular reference)
		newParent, err := s.getFolderModel(*newParentID)
		if err != nil {
			return ErrInvalidParentFolder
		}

		if strings.HasPrefix(newParent.Path, folder.Path) {
			return ErrCircularReference
		}

		newDepth = newParent.Depth + 1
		newPath = fmt.Sprintf("%s/%s", newParent.Path, folder.ID)
	} else {
		newDepth = 0
		newPath = fmt.Sprintf("/%s", folder.ID)
	}

	// Check for duplicate name in new parent
	var count int64
	query := s.db.Model(&FolderModel{}).Where("name = ? AND id != ?", folder.Name, id)
	if newParentID != nil {
		query = query.Where("parent_id = ?", *newParentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check folder name: %w", err)
	}
	if count > 0 {
		return ErrFolderNameExists
	}

	// Update the folder
	depthDelta := newDepth - folder.Depth
	if err := s.db.Model(folder).Updates(map[string]interface{}{
		"parent_id": newParentID,
		"path":      newPath,
		"depth":     newDepth,
	}).Error; err != nil {
		return fmt.Errorf("failed to move folder: %w", err)
	}

	// Update all descendants' paths and depths
	if err := s.db.Exec(`
		UPDATE folders 
		SET path = REPLACE(path, ?, ?),
		    depth = depth + ?
		WHERE path LIKE ?
	`, oldPath, newPath, depthDelta, oldPath+"/%").Error; err != nil {
		return fmt.Errorf("failed to update descendant paths: %w", err)
	}

	return nil
}

// ==================== Project CRUD ====================

// CreateProject creates a new project
func (s *FolderServiceImpl) CreateProject(name, description string, folderID *string, createdBy string) (*Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrProjectNameEmpty
	}

	// Check for duplicate name in same folder
	var count int64
	query := s.db.Model(&ProjectModel{}).Where("name = ?", name)
	if folderID != nil {
		// Verify folder exists
		if _, err := s.getFolderModel(*folderID); err != nil {
			return nil, ErrFolderNotFound
		}
		query = query.Where("folder_id = ?", *folderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	if err := query.Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to check project name: %w", err)
	}
	if count > 0 {
		return nil, ErrProjectNameExists
	}

	projectModel := &ProjectModel{
		Name:        name,
		Description: description,
		FolderID:    folderID,
		CreatedBy:   createdBy,
	}

	if err := s.db.Create(projectModel).Error; err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	return s.projectToDomain(projectModel), nil
}

// GetProject retrieves a project by ID
func (s *FolderServiceImpl) GetProject(id string) (*Project, error) {
	projectModel, err := s.getProjectModel(id)
	if err != nil {
		return nil, err
	}
	return s.projectToDomain(projectModel), nil
}

// getProjectModel is an internal helper to get the GORM project model
func (s *FolderServiceImpl) getProjectModel(id string) (*ProjectModel, error) {
	var projectModel ProjectModel
	if err := s.db.Where("id = ?", id).First(&projectModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	return &projectModel, nil
}

// UpdateProject updates project name and description
func (s *FolderServiceImpl) UpdateProject(id, name, description string) (*Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrProjectNameEmpty
	}

	projectModel, err := s.getProjectModel(id)
	if err != nil {
		return nil, err
	}

	// Check for duplicate name in same folder (excluding current project)
	var count int64
	query := s.db.Model(&ProjectModel{}).Where("name = ? AND id != ?", name, id)
	if projectModel.FolderID != nil {
		query = query.Where("folder_id = ?", *projectModel.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	if err := query.Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to check project name: %w", err)
	}
	if count > 0 {
		return nil, ErrProjectNameExists
	}

	projectModel.Name = name
	projectModel.Description = description
	if err := s.db.Save(projectModel).Error; err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return s.projectToDomain(projectModel), nil
}

// DeleteProject deletes a project
func (s *FolderServiceImpl) DeleteProject(id string, force bool) error {
	_, err := s.getProjectModel(id)
	if err != nil {
		return err
	}

	// Check for models in project
	var modelCount int64
	if err := s.db.Table("models").Where("project_id = ?", id).Count(&modelCount).Error; err != nil {
		return fmt.Errorf("failed to check models: %w", err)
	}
	if modelCount > 0 && !force {
		return ErrProjectHasModels
	}

	// Check for builds in project
	var buildCount int64
	if err := s.db.Table("model_builds").Where("project_id = ?", id).Count(&buildCount).Error; err != nil {
		return fmt.Errorf("failed to check builds: %w", err)
	}
	if buildCount > 0 && !force {
		return ErrProjectHasBuilds
	}

	// If force delete, set project_id to NULL on associated models and builds
	if force {
		if err := s.db.Exec("UPDATE models SET project_id = NULL WHERE project_id = ?", id).Error; err != nil {
			return fmt.Errorf("failed to unlink models: %w", err)
		}
		if err := s.db.Exec("UPDATE model_builds SET project_id = NULL WHERE project_id = ?", id).Error; err != nil {
			return fmt.Errorf("failed to unlink builds: %w", err)
		}
	}

	// Delete the project
	if err := s.db.Delete(&ProjectModel{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// MoveProject moves a project to a new folder
func (s *FolderServiceImpl) MoveProject(id string, newFolderID *string) error {
	project, err := s.getProjectModel(id)
	if err != nil {
		return err
	}

	// Verify new folder exists if specified
	if newFolderID != nil {
		if _, err := s.getFolderModel(*newFolderID); err != nil {
			return ErrFolderNotFound
		}
	}

	// Check for duplicate name in new folder
	var count int64
	query := s.db.Model(&ProjectModel{}).Where("name = ? AND id != ?", project.Name, id)
	if newFolderID != nil {
		query = query.Where("folder_id = ?", *newFolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check project name: %w", err)
	}
	if count > 0 {
		return ErrProjectNameExists
	}

	// Update the project's folder
	if err := s.db.Model(project).Update("folder_id", newFolderID).Error; err != nil {
		return fmt.Errorf("failed to move project: %w", err)
	}

	return nil
}

// ==================== Project Listing ====================

// GetProjectsInFolder gets all projects directly in a folder
func (s *FolderServiceImpl) GetProjectsInFolder(folderID string) ([]Project, error) {
	var projectModels []ProjectModel
	if err := s.db.Where("folder_id = ?", folderID).Order("name ASC").Find(&projectModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get projects in folder: %w", err)
	}
	return s.projectsToDomainSlice(projectModels), nil
}

// GetRootProjects gets all projects not in any folder
func (s *FolderServiceImpl) GetRootProjects() ([]Project, error) {
	var projectModels []ProjectModel
	if err := s.db.Where("folder_id IS NULL").Order("name ASC").Find(&projectModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get root projects: %w", err)
	}
	return s.projectsToDomainSlice(projectModels), nil
}

// GetAllDescendantProjects gets all projects in a folder and its descendants
func (s *FolderServiceImpl) GetAllDescendantProjects(folderID string) ([]Project, error) {
	folder, err := s.getFolderModel(folderID)
	if err != nil {
		return nil, err
	}

	var projectModels []ProjectModel
	if err := s.db.Raw(`
		SELECT p.* FROM projects p
		INNER JOIN folders f ON f.id = p.folder_id
		WHERE f.path LIKE ? OR f.id = ?
		ORDER BY p.name ASC
	`, folder.Path+"/%", folderID).Scan(&projectModels).Error; err != nil {
		return nil, fmt.Errorf("failed to get descendant projects: %w", err)
	}

	return s.projectsToDomainSlice(projectModels), nil
}

// ==================== Project-Model Association ====================

// AddModelToProject adds a model to a project
func (s *FolderServiceImpl) AddModelToProject(modelID, projectID string) error {
	// Verify project exists
	if _, err := s.getProjectModel(projectID); err != nil {
		return err
	}

	// Update model's project_id (also clear folder_id as model can only be in one place)
	if err := s.db.Exec("UPDATE models SET project_id = ?, folder_id = NULL WHERE id = ?", projectID, modelID).Error; err != nil {
		return fmt.Errorf("failed to add model to project: %w", err)
	}

	return nil
}

// RemoveModelFromProject removes a model from its project
func (s *FolderServiceImpl) RemoveModelFromProject(modelID string) error {
	if err := s.db.Exec("UPDATE models SET project_id = NULL WHERE id = ?", modelID).Error; err != nil {
		return fmt.Errorf("failed to remove model from project: %w", err)
	}
	return nil
}

// GetModelProject gets the project a model belongs to
func (s *FolderServiceImpl) GetModelProject(modelID string) (*Project, error) {
	var projectID *string
	if err := s.db.Table("models").Where("id = ?", modelID).Pluck("project_id", &projectID).Error; err != nil {
		return nil, fmt.Errorf("failed to get model project: %w", err)
	}

	if projectID == nil {
		return nil, nil // Model has no project
	}

	return s.GetProject(*projectID)
}

// GetModelsInProject gets all model IDs in a project
func (s *FolderServiceImpl) GetModelsInProject(projectID string) ([]string, error) {
	var modelIDs []string
	if err := s.db.Table("models").Where("project_id = ?", projectID).Pluck("id", &modelIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to get models in project: %w", err)
	}
	return modelIDs, nil
}

// ==================== Project-Build Association ====================

// AddBuildToProject adds a build to a project
func (s *FolderServiceImpl) AddBuildToProject(buildID, projectID string) error {
	// Verify project exists
	if _, err := s.getProjectModel(projectID); err != nil {
		return err
	}

	// Update build's project_id (also clear folder_id as build can only be in one place)
	if err := s.db.Exec("UPDATE model_builds SET project_id = ?, folder_id = NULL WHERE id = ?", projectID, buildID).Error; err != nil {
		return fmt.Errorf("failed to add build to project: %w", err)
	}

	return nil
}

// RemoveBuildFromProject removes a build from its project
func (s *FolderServiceImpl) RemoveBuildFromProject(buildID string) error {
	if err := s.db.Exec("UPDATE model_builds SET project_id = NULL WHERE id = ?", buildID).Error; err != nil {
		return fmt.Errorf("failed to remove build from project: %w", err)
	}
	return nil
}

// GetBuildProject gets the project a build belongs to
func (s *FolderServiceImpl) GetBuildProject(buildID string) (*Project, error) {
	var projectID *string
	if err := s.db.Table("model_builds").Where("id = ?", buildID).Pluck("project_id", &projectID).Error; err != nil {
		return nil, fmt.Errorf("failed to get build project: %w", err)
	}

	if projectID == nil {
		return nil, nil // Build has no project
	}

	return s.GetProject(*projectID)
}

// GetBuildsInProject gets all build IDs in a project
func (s *FolderServiceImpl) GetBuildsInProject(projectID string) ([]string, error) {
	var buildIDs []string
	if err := s.db.Table("model_builds").Where("project_id = ?", projectID).Pluck("id", &buildIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to get builds in project: %w", err)
	}
	return buildIDs, nil
}

// ==================== Folder-Build Association ====================

// AddBuildToFolder adds a build directly to a folder
func (s *FolderServiceImpl) AddBuildToFolder(buildID, folderID string) error {
	// Verify folder exists
	if _, err := s.getFolderModel(folderID); err != nil {
		return err
	}

	// Update build's folder_id (also clear project_id as build can only be in one place)
	if err := s.db.Exec("UPDATE model_builds SET folder_id = ?, project_id = NULL WHERE id = ?", folderID, buildID).Error; err != nil {
		return fmt.Errorf("failed to add build to folder: %w", err)
	}
	return nil
}

// RemoveBuildFromFolder removes a build from its folder
func (s *FolderServiceImpl) RemoveBuildFromFolder(buildID string) error {
	if err := s.db.Exec("UPDATE model_builds SET folder_id = NULL WHERE id = ?", buildID).Error; err != nil {
		return fmt.Errorf("failed to remove build from folder: %w", err)
	}
	return nil
}

// GetBuildFolder gets the folder a build belongs to (if directly in a folder)
func (s *FolderServiceImpl) GetBuildFolder(buildID string) (*Folder, error) {
	var folderID *string
	if err := s.db.Table("model_builds").Where("id = ?", buildID).Pluck("folder_id", &folderID).Error; err != nil {
		return nil, fmt.Errorf("failed to get build folder: %w", err)
	}

	if folderID == nil {
		return nil, nil // Build has no folder
	}

	return s.GetFolder(*folderID)
}

// GetBuildsInFolder gets all build IDs directly in a folder
func (s *FolderServiceImpl) GetBuildsInFolder(folderID string) ([]string, error) {
	var buildIDs []string
	if err := s.db.Table("model_builds").Where("folder_id = ?", folderID).Pluck("id", &buildIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to get builds in folder: %w", err)
	}
	return buildIDs, nil
}

// ==================== Folder-Model Association ====================

// AddModelToFolder adds a model directly to a folder
func (s *FolderServiceImpl) AddModelToFolder(modelID, folderID string) error {
	// Verify folder exists
	if _, err := s.getFolderModel(folderID); err != nil {
		return err
	}

	// Update model's folder_id (also clear project_id as model can only be in one place)
	if err := s.db.Exec("UPDATE models SET folder_id = ?, project_id = NULL WHERE id = ?", folderID, modelID).Error; err != nil {
		return fmt.Errorf("failed to add model to folder: %w", err)
	}
	return nil
}

// RemoveModelFromFolder removes a model from its folder
func (s *FolderServiceImpl) RemoveModelFromFolder(modelID string) error {
	if err := s.db.Exec("UPDATE models SET folder_id = NULL WHERE id = ?", modelID).Error; err != nil {
		return fmt.Errorf("failed to remove model from folder: %w", err)
	}
	return nil
}

// GetModelFolder gets the folder a model belongs to (if directly in a folder)
func (s *FolderServiceImpl) GetModelFolder(modelID string) (*Folder, error) {
	var folderID *string
	if err := s.db.Table("models").Where("id = ?", modelID).Pluck("folder_id", &folderID).Error; err != nil {
		return nil, fmt.Errorf("failed to get model folder: %w", err)
	}

	if folderID == nil {
		return nil, nil // Model has no folder
	}

	return s.GetFolder(*folderID)
}

// GetModelsInFolder gets all model IDs directly in a folder
func (s *FolderServiceImpl) GetModelsInFolder(folderID string) ([]string, error) {
	var modelIDs []string
	if err := s.db.Table("models").Where("folder_id = ?", folderID).Pluck("id", &modelIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to get models in folder: %w", err)
	}
	return modelIDs, nil
}

// ==================== Descendant Queries ====================

// GetAllDescendantModels gets all model IDs in a folder (direct and through projects) and all subfolders
func (s *FolderServiceImpl) GetAllDescendantModels(folderID string) ([]string, error) {
	folder, err := s.getFolderModel(folderID)
	if err != nil {
		return nil, err
	}

	// Get models in projects within this folder and descendants, plus models directly in folders
	var modelIDs []string
	if err := s.db.Raw(`
		SELECT m.id FROM models m
		WHERE m.project_id IN (
			SELECT p.id FROM projects p
		INNER JOIN folders f ON f.id = p.folder_id
		WHERE f.path LIKE ? OR f.id = ?
		)
		OR m.folder_id IN (
			SELECT f.id FROM folders f
			WHERE f.path LIKE ? OR f.id = ?
		)
	`, folder.Path+"/%", folderID, folder.Path+"/%", folderID).Scan(&modelIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to get descendant models: %w", err)
	}

	return modelIDs, nil
}

// GetAllDescendantBuilds gets all build IDs in a folder (direct and through projects) and all subfolders
func (s *FolderServiceImpl) GetAllDescendantBuilds(folderID string) ([]string, error) {
	folder, err := s.getFolderModel(folderID)
	if err != nil {
		return nil, err
	}

	// Get builds in projects within this folder and descendants, plus builds directly in folders
	var buildIDs []string
	if err := s.db.Raw(`
		SELECT mb.id FROM model_builds mb
		WHERE mb.project_id IN (
			SELECT p.id FROM projects p
		INNER JOIN folders f ON f.id = p.folder_id
		WHERE f.path LIKE ? OR f.id = ?
		)
		OR mb.folder_id IN (
			SELECT f.id FROM folders f
			WHERE f.path LIKE ? OR f.id = ?
		)
	`, folder.Path+"/%", folderID, folder.Path+"/%", folderID).Scan(&buildIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to get descendant builds: %w", err)
	}

	return buildIDs, nil
}

// ==================== Search ====================

// SearchFolders searches folders by name
func (s *FolderServiceImpl) SearchFolders(query string) ([]Folder, error) {
	var folderModels []FolderModel
	if err := s.db.Where("name ILIKE ?", "%"+query+"%").Order("path ASC").Find(&folderModels).Error; err != nil {
		return nil, fmt.Errorf("failed to search folders: %w", err)
	}
	return s.toDomainSlice(folderModels), nil
}

// SearchProjects searches projects by name
func (s *FolderServiceImpl) SearchProjects(query string) ([]Project, error) {
	var projectModels []ProjectModel
	if err := s.db.Where("name ILIKE ?", "%"+query+"%").Order("name ASC").Find(&projectModels).Error; err != nil {
		return nil, fmt.Errorf("failed to search projects: %w", err)
	}
	return s.projectsToDomainSlice(projectModels), nil
}

// ==================== Converters ====================

// toDomain converts a FolderModel to a Folder domain object
func (s *FolderServiceImpl) toDomain(fm *FolderModel) *Folder {
	return &Folder{
		ID:          fm.ID,
		Name:        fm.Name,
		Description: fm.Description,
		ParentID:    fm.ParentID,
		Path:        fm.Path,
		Depth:       fm.Depth,
		CreatedBy:   fm.CreatedBy,
		CreatedAt:   fm.CreatedAt,
		UpdatedAt:   fm.UpdatedAt,
	}
}

// toDomainSlice converts a slice of FolderModels to Folders
func (s *FolderServiceImpl) toDomainSlice(fms []FolderModel) []Folder {
	folders := make([]Folder, len(fms))
	for i, fm := range fms {
		folders[i] = *s.toDomain(&fm)
	}
	return folders
}

// projectToDomain converts a ProjectModel to a Project domain object
func (s *FolderServiceImpl) projectToDomain(pm *ProjectModel) *Project {
	return &Project{
		ID:          pm.ID,
		Name:        pm.Name,
		Description: pm.Description,
		FolderID:    pm.FolderID,
		CreatedBy:   pm.CreatedBy,
		CreatedAt:   pm.CreatedAt,
		UpdatedAt:   pm.UpdatedAt,
	}
}

// projectsToDomainSlice converts a slice of ProjectModels to Projects
func (s *FolderServiceImpl) projectsToDomainSlice(pms []ProjectModel) []Project {
	projects := make([]Project, len(pms))
	for i, pm := range pms {
		projects[i] = *s.projectToDomain(&pm)
	}
	return projects
}

// GetModels returns the GORM models for migrations
func GetModels() []interface{} {
	return []interface{}{
		&FolderModel{},
		&ProjectModel{},
	}
}




