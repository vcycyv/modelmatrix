package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ModelModel is the GORM model for trained models
type ModelModel struct {
	ID           string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_model_name"`
	Description  string         `gorm:"type:text"`
	BuildID      string         `gorm:"type:uuid;not null;index:idx_model_build"`
	Status       string         `gorm:"type:varchar(50);not null;default:'draft';index:idx_model_status"`
	ArtifactPath string         `gorm:"type:varchar(1024)"`
	Metadata     JSONMap        `gorm:"type:jsonb"`
	CreatedBy    string         `gorm:"type:varchar(255);not null"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	// Relations
	Versions []ModelVersionModel `gorm:"foreignKey:ModelID;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for ModelModel
func (ModelModel) TableName() string {
	return "models"
}

// BeforeCreate generates UUID before creating record
func (m *ModelModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// ModelVersionModel is the GORM model for model versions
type ModelVersionModel struct {
	ID           string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ModelID      string         `gorm:"type:uuid;not null;index:idx_version_model;uniqueIndex:idx_version_model_version,priority:1"`
	Version      string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_version_model_version,priority:2"`
	BuildID      string         `gorm:"type:uuid;not null;index:idx_version_build"`
	Status       string         `gorm:"type:varchar(50);not null;default:'draft'"`
	ArtifactPath string         `gorm:"type:varchar(1024)"`
	Metrics      JSONMap        `gorm:"type:jsonb"`
	Notes        string         `gorm:"type:text"`
	CreatedBy    string         `gorm:"type:varchar(255);not null"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	// Relations
	Model ModelModel `gorm:"foreignKey:ModelID"`
}

// TableName returns the table name for ModelVersionModel
func (ModelVersionModel) TableName() string {
	return "model_versions"
}

// BeforeCreate generates UUID before creating record
func (v *ModelVersionModel) BeforeCreate(tx *gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	return nil
}

// JSONMap is a custom type for JSONB fields
type JSONMap map[string]interface{}

// Value implements driver.Valuer interface
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONMap value")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*j = result
	return nil
}
