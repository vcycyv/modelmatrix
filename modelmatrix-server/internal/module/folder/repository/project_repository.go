package repository

import (
	"modelmatrix-server/internal/module/folder/domain"
	"modelmatrix-server/internal/module/folder/model"

	"gorm.io/gorm"
)

// ProjectRepositoryImpl implements ProjectRepository
type ProjectRepositoryImpl struct {
	db *gorm.DB
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &ProjectRepositoryImpl{db: db}
}

// Create creates a new project
func (r *ProjectRepositoryImpl) Create(project *domain.Project) error {
	m := r.toModel(project)
	if err := r.db.Create(m).Error; err != nil {
		return err
	}
	project.ID = m.ID
	project.CreatedAt = m.CreatedAt
	project.UpdatedAt = m.UpdatedAt
	return nil
}

// GetByID retrieves a project by ID
func (r *ProjectRepositoryImpl) GetByID(id string) (*domain.Project, error) {
	var m model.ProjectModel
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrProjectNotFound
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

// Update updates a project
func (r *ProjectRepositoryImpl) Update(project *domain.Project) error {
	return r.db.Model(&model.ProjectModel{}).Where("id = ?", project.ID).Updates(map[string]interface{}{
		"name":        project.Name,
		"description": project.Description,
	}).Error
}

// Delete deletes a project
func (r *ProjectRepositoryImpl) Delete(id string) error {
	return r.db.Delete(&model.ProjectModel{}, "id = ?", id).Error
}

// GetByFolderID retrieves all projects in a folder
func (r *ProjectRepositoryImpl) GetByFolderID(folderID string) ([]domain.Project, error) {
	var models []model.ProjectModel
	if err := r.db.Where("folder_id = ?", folderID).Order("name ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	return r.toDomainSlice(models), nil
}

// GetRootProjects retrieves all root projects (not in any folder)
func (r *ProjectRepositoryImpl) GetRootProjects() ([]domain.Project, error) {
	var models []model.ProjectModel
	if err := r.db.Where("folder_id IS NULL").Order("name ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	return r.toDomainSlice(models), nil
}

// GetByFolderIDAndName finds a project by folder ID and name
func (r *ProjectRepositoryImpl) GetByFolderIDAndName(folderID *string, name string) (*domain.Project, error) {
	var m model.ProjectModel
	query := r.db.Where("name = ?", name)
	if folderID == nil {
		query = query.Where("folder_id IS NULL")
	} else {
		query = query.Where("folder_id = ?", *folderID)
	}
	if err := query.First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.toDomain(&m), nil
}

// CountModels counts models in a project
func (r *ProjectRepositoryImpl) CountModels(id string) (int64, error) {
	var count int64
	if err := r.db.Table("models").Where("project_id = ?", id).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountBuilds counts builds in a project
func (r *ProjectRepositoryImpl) CountBuilds(id string) (int64, error) {
	var count int64
	if err := r.db.Table("model_builds").Where("project_id = ?", id).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteByFolderIDs deletes projects in the given folders
func (r *ProjectRepositoryImpl) DeleteByFolderIDs(folderIDs []string) error {
	if len(folderIDs) == 0 {
		return nil
	}
	return r.db.Where("folder_id IN ?", folderIDs).Delete(&model.ProjectModel{}).Error
}

// toModel converts domain entity to GORM model
func (r *ProjectRepositoryImpl) toModel(project *domain.Project) *model.ProjectModel {
	return &model.ProjectModel{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		FolderID:    project.FolderID,
		CreatedBy:   project.CreatedBy,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
	}
}

// toDomain converts GORM model to domain entity
func (r *ProjectRepositoryImpl) toDomain(m *model.ProjectModel) *domain.Project {
	return &domain.Project{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		FolderID:    m.FolderID,
		CreatedBy:   m.CreatedBy,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// toDomainSlice converts a slice of GORM models to domain entities
func (r *ProjectRepositoryImpl) toDomainSlice(models []model.ProjectModel) []domain.Project {
	result := make([]domain.Project, len(models))
	for i, m := range models {
		result[i] = *r.toDomain(&m)
	}
	return result
}
