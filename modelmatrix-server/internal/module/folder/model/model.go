package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FolderModel is the GORM model for folders
type FolderModel struct {
	ID          string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string `gorm:"type:varchar(255);not null"`
	Description string `gorm:"type:text"`
	ParentID    *string `gorm:"type:uuid;index:idx_folder_parent"`
	Path        string `gorm:"type:varchar(1024);not null;index:idx_folder_path"`
	Depth       int    `gorm:"not null;default:0"`
	CreatedBy   string `gorm:"type:varchar(255);not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for FolderModel
func (FolderModel) TableName() string {
	return "folders"
}

// BeforeCreate generates UUID before creating record
func (m *FolderModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// ProjectModel is the GORM model for projects
type ProjectModel struct {
	ID          string `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string `gorm:"type:varchar(255);not null"`
	Description string `gorm:"type:text"`
	FolderID    *string `gorm:"type:uuid;index:idx_project_folder"`
	CreatedBy   string `gorm:"type:varchar(255);not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for ProjectModel
func (ProjectModel) TableName() string {
	return "projects"
}

// BeforeCreate generates UUID before creating record
func (m *ProjectModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// GetModels returns all folder module GORM models for migration
func GetModels() []interface{} {
	return []interface{}{
		&FolderModel{},
		&ProjectModel{},
	}
}
