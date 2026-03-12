package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ModelVersionModel is the GORM model for immutable model version snapshots
type ModelVersionModel struct {
	ID            string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ModelID       string    `gorm:"type:uuid;not null;index:idx_version_model"`
	VersionNumber int       `gorm:"not null"`
	Name          string    `gorm:"type:varchar(255);not null"`
	Description   string    `gorm:"type:text"`
	BuildID       string    `gorm:"type:uuid;not null"`
	DatasourceID  string    `gorm:"type:uuid;not null"`
	ProjectID     *string   `gorm:"type:uuid"`
	FolderID      *string   `gorm:"type:uuid"`
	Algorithm     string    `gorm:"type:varchar(100);not null"`
	ModelType     string    `gorm:"type:varchar(50);not null"`
	TargetColumn  string    `gorm:"type:varchar(255);not null"`
	Status        string    `gorm:"type:varchar(50);not null"`
	Metrics       JSONMap   `gorm:"type:jsonb"`
	CreatedBy     string    `gorm:"type:varchar(255);not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`

	Variables []ModelVersionVariableModel `gorm:"foreignKey:ModelVersionID;constraint:OnDelete:CASCADE"`
	Files     []ModelVersionFileModel    `gorm:"foreignKey:ModelVersionID;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name
func (ModelVersionModel) TableName() string {
	return "model_versions"
}

// BeforeCreate generates UUID before creating record
func (m *ModelVersionModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// ModelVersionVariableModel is the GORM model for variables in a version snapshot
type ModelVersionVariableModel struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ModelVersionID string    `gorm:"type:uuid;not null;index:idx_version_var_version"`
	Name           string    `gorm:"type:varchar(255);not null"`
	DataType       string    `gorm:"type:varchar(50);not null"`
	Role           string    `gorm:"type:varchar(50);not null"`
	Importance     *float64  `gorm:"type:decimal(10,6)"`
	Statistics     JSONMap   `gorm:"type:jsonb"`
	EncodingInfo   JSONMap   `gorm:"type:jsonb"`
	Ordinal        int       `gorm:"not null"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

// TableName returns the table name
func (ModelVersionVariableModel) TableName() string {
	return "model_version_variables"
}

// BeforeCreate generates UUID before creating record
func (v *ModelVersionVariableModel) BeforeCreate(tx *gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	return nil
}

// ModelVersionFileModel is the GORM model for files in a version snapshot (path = content-addressable store)
type ModelVersionFileModel struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ModelVersionID string    `gorm:"type:uuid;not null;index:idx_version_file_version"`
	FileType       string    `gorm:"type:varchar(50);not null"`
	FilePath       string    `gorm:"type:varchar(500);not null"` // minio://bucket/versions/content/{hash}
	FileName       string    `gorm:"type:varchar(255);not null"`
	FileSize       *int64    `gorm:"type:bigint"`
	Checksum       string    `gorm:"type:varchar(64)"`
	Description    string    `gorm:"type:text"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

// TableName returns the table name
func (ModelVersionFileModel) TableName() string {
	return "model_version_files"
}

// BeforeCreate generates UUID before creating record
func (f *ModelVersionFileModel) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
	}
	return nil
}
