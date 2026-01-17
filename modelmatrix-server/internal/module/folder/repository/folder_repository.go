package repository

import (
	"fmt"

	"modelmatrix-server/internal/module/folder/domain"
	"modelmatrix-server/internal/module/folder/model"

	"gorm.io/gorm"
)

// FolderRepositoryImpl implements FolderRepository
type FolderRepositoryImpl struct {
	db *gorm.DB
}

// NewFolderRepository creates a new folder repository
func NewFolderRepository(db *gorm.DB) FolderRepository {
	return &FolderRepositoryImpl{db: db}
}

// Create creates a new folder
func (r *FolderRepositoryImpl) Create(folder *domain.Folder) error {
	m := r.toModel(folder)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	folder.ID = m.ID
	folder.CreatedAt = m.CreatedAt
	folder.UpdatedAt = m.UpdatedAt
	return nil
}

// GetByID retrieves a folder by ID
func (r *FolderRepositoryImpl) GetByID(id string) (*domain.Folder, error) {
	var m model.FolderModel
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrFolderNotFound
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

// Update updates a folder's name and description
func (r *FolderRepositoryImpl) Update(folder *domain.Folder) error {
	return r.db.Model(&model.FolderModel{}).Where("id = ?", folder.ID).Updates(map[string]interface{}{
		"name":        folder.Name,
		"description": folder.Description,
	}).Error
}

// UpdatePath updates a folder's path (used after creation)
func (r *FolderRepositoryImpl) UpdatePath(id, path string) error {
	return r.db.Model(&model.FolderModel{}).Where("id = ?", id).Update("path", path).Error
}

// Delete deletes a folder
func (r *FolderRepositoryImpl) Delete(id string) error {
	return r.db.Delete(&model.FolderModel{}, "id = ?", id).Error
}

// GetChildren retrieves direct children of a folder
func (r *FolderRepositoryImpl) GetChildren(parentID string) ([]domain.Folder, error) {
	var models []model.FolderModel
	if err := r.db.Where("parent_id = ?", parentID).Order("name ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get children: %w", err)
	}
	return r.toDomainSlice(models), nil
}

// GetRootFolders retrieves all root folders
func (r *FolderRepositoryImpl) GetRootFolders() ([]domain.Folder, error) {
	var models []model.FolderModel
	if err := r.db.Where("parent_id IS NULL").Order("name ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get root folders: %w", err)
	}
	return r.toDomainSlice(models), nil
}

// GetPath retrieves the path from root to the given folder
func (r *FolderRepositoryImpl) GetPath(id string) ([]domain.Folder, error) {
	folder, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	var result []domain.Folder
	current := folder
	for current != nil {
		result = append([]domain.Folder{*current}, result...)
		if current.ParentID == nil {
			break
		}
		current, err = r.GetByID(*current.ParentID)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// GetDescendants retrieves all descendant folders
func (r *FolderRepositoryImpl) GetDescendants(id string) ([]domain.Folder, error) {
	folder, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	pathPattern := folder.Path + "/%"
	var models []model.FolderModel
	if err := r.db.Where("path LIKE ?", pathPattern).Order("depth ASC, name ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get descendants: %w", err)
	}
	return r.toDomainSlice(models), nil
}

// GetByParentIDAndName finds a folder by parent ID and name
func (r *FolderRepositoryImpl) GetByParentIDAndName(parentID *string, name string) (*domain.Folder, error) {
	var m model.FolderModel
	query := r.db.Where("name = ?", name)
	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}
	if err := query.First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

// CountChildren counts direct child folders
func (r *FolderRepositoryImpl) CountChildren(id string) (int64, error) {
	var count int64
	if err := r.db.Model(&model.FolderModel{}).Where("parent_id = ?", id).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountProjects counts projects directly in a folder
func (r *FolderRepositoryImpl) CountProjects(id string) (int64, error) {
	var count int64
	if err := r.db.Model(&model.ProjectModel{}).Where("folder_id = ?", id).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetContentsCount returns counts of items in a folder and its descendants
func (r *FolderRepositoryImpl) GetContentsCount(id string) (*domain.FolderContentsCount, error) {
	folder, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	pathPattern := folder.Path + "/%"

	// Count direct subfolders
	var subfolderCount int64
	if err := r.db.Model(&model.FolderModel{}).Where("parent_id = ?", id).Count(&subfolderCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count subfolders: %w", err)
	}

	// Count all descendant subfolders
	var totalDescendantFolders int64
	if err := r.db.Model(&model.FolderModel{}).Where("path LIKE ?", pathPattern).Count(&totalDescendantFolders).Error; err != nil {
		return nil, fmt.Errorf("failed to count descendant folders: %w", err)
	}

	// Count projects directly in this folder
	var directProjectCount int64
	if err := r.db.Model(&model.ProjectModel{}).Where("folder_id = ?", id).Count(&directProjectCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count direct projects: %w", err)
	}

	// Count projects in descendant folders
	var descendantProjectCount int64
	if err := r.db.Raw(`
		SELECT COUNT(*) FROM projects WHERE folder_id IN (SELECT id FROM folders WHERE path LIKE ?)
	`, pathPattern).Scan(&descendantProjectCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count descendant projects: %w", err)
	}

	// Count models
	var directModelCount, projectModelCount, descendantModelCount int64
	r.db.Table("models").Where("folder_id = ?", id).Count(&directModelCount)
	r.db.Raw(`SELECT COUNT(*) FROM models WHERE project_id IN (SELECT id FROM projects WHERE folder_id = ?)`, id).Scan(&projectModelCount)
	r.db.Raw(`
		SELECT COUNT(*) FROM models WHERE 
			folder_id IN (SELECT id FROM folders WHERE path LIKE ?) OR
			project_id IN (SELECT id FROM projects WHERE folder_id IN (SELECT id FROM folders WHERE path LIKE ?))
	`, pathPattern, pathPattern).Scan(&descendantModelCount)

	// Count builds
	var directBuildCount, projectBuildCount, descendantBuildCount int64
	r.db.Table("model_builds").Where("folder_id = ?", id).Count(&directBuildCount)
	r.db.Raw(`SELECT COUNT(*) FROM model_builds WHERE project_id IN (SELECT id FROM projects WHERE folder_id = ?)`, id).Scan(&projectBuildCount)
	r.db.Raw(`
		SELECT COUNT(*) FROM model_builds WHERE 
			folder_id IN (SELECT id FROM folders WHERE path LIKE ?) OR
			project_id IN (SELECT id FROM projects WHERE folder_id IN (SELECT id FROM folders WHERE path LIKE ?))
	`, pathPattern, pathPattern).Scan(&descendantBuildCount)

	return &domain.FolderContentsCount{
		SubfolderCount: subfolderCount + totalDescendantFolders,
		ProjectCount:   directProjectCount + descendantProjectCount,
		ModelCount:     directModelCount + projectModelCount + descendantModelCount,
		BuildCount:     directBuildCount + projectBuildCount + descendantBuildCount,
	}, nil
}

// GetDescendantFolderIDs returns IDs of all descendant folders matching the path pattern
func (r *FolderRepositoryImpl) GetDescendantFolderIDs(pathPattern string) ([]string, error) {
	var ids []string
	if err := r.db.Model(&model.FolderModel{}).Where("path LIKE ?", pathPattern).Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// DeleteDescendants deletes all descendant folders matching the path pattern
func (r *FolderRepositoryImpl) DeleteDescendants(pathPattern string) error {
	return r.db.Where("path LIKE ?", pathPattern).Delete(&model.FolderModel{}).Error
}

// toModel converts domain entity to GORM model
func (r *FolderRepositoryImpl) toModel(folder *domain.Folder) *model.FolderModel {
	return &model.FolderModel{
		ID:          folder.ID,
		Name:        folder.Name,
		Description: folder.Description,
		ParentID:    folder.ParentID,
		Path:        folder.Path,
		Depth:       folder.Depth,
		CreatedBy:   folder.CreatedBy,
		CreatedAt:   folder.CreatedAt,
		UpdatedAt:   folder.UpdatedAt,
	}
}

// toDomain converts GORM model to domain entity
func (r *FolderRepositoryImpl) toDomain(m *model.FolderModel) *domain.Folder {
	return &domain.Folder{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		ParentID:    m.ParentID,
		Path:        m.Path,
		Depth:       m.Depth,
		CreatedBy:   m.CreatedBy,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// toDomainSlice converts a slice of GORM models to domain entities
func (r *FolderRepositoryImpl) toDomainSlice(models []model.FolderModel) []domain.Folder {
	result := make([]domain.Folder, len(models))
	for i, m := range models {
		result[i] = *r.toDomain(&m)
	}
	return result
}
