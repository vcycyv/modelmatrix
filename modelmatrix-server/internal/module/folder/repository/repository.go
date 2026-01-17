package repository

import "modelmatrix-server/internal/module/folder/domain"

// FolderRepository defines the interface for folder data access
type FolderRepository interface {
	// CRUD operations
	Create(folder *domain.Folder) error
	GetByID(id string) (*domain.Folder, error)
	Update(folder *domain.Folder) error
	UpdatePath(id, path string) error
	Delete(id string) error

	// Hierarchy operations
	GetChildren(parentID string) ([]domain.Folder, error)
	GetRootFolders() ([]domain.Folder, error)
	GetPath(id string) ([]domain.Folder, error)
	GetDescendants(id string) ([]domain.Folder, error)
	GetByParentIDAndName(parentID *string, name string) (*domain.Folder, error)

	// Content counting
	CountChildren(id string) (int64, error)
	CountProjects(id string) (int64, error)
	GetContentsCount(id string) (*domain.FolderContentsCount, error)

	// Batch operations for cascade delete
	GetDescendantFolderIDs(pathPattern string) ([]string, error)
	DeleteDescendants(pathPattern string) error
}

// ProjectRepository defines the interface for project data access
type ProjectRepository interface {
	// CRUD operations
	Create(project *domain.Project) error
	GetByID(id string) (*domain.Project, error)
	Update(project *domain.Project) error
	Delete(id string) error

	// Listing
	GetByFolderID(folderID string) ([]domain.Project, error)
	GetRootProjects() ([]domain.Project, error)
	GetByFolderIDAndName(folderID *string, name string) (*domain.Project, error)

	// Content counting
	CountModels(id string) (int64, error)
	CountBuilds(id string) (int64, error)

	// Batch operations
	DeleteByFolderIDs(folderIDs []string) error
}
