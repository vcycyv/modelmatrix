package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ModelBuildModel is the GORM model for model builds
type ModelBuildModel struct {
	ID           string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_build_name"`
	Description  string         `gorm:"type:text"`
	DatasourceID string         `gorm:"type:uuid;not null;index:idx_build_datasource"`
	ModelType    string         `gorm:"type:varchar(50);not null"`
	Status       string         `gorm:"type:varchar(50);not null;default:'pending';index:idx_build_status"`
	Parameters   JSONMap        `gorm:"type:jsonb"`
	Metrics      JSONMap        `gorm:"type:jsonb"`
	ErrorMessage string         `gorm:"type:text"`
	StartedAt    *time.Time
	CompletedAt  *time.Time
	CreatedBy string    `gorm:"type:varchar(255);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for ModelBuildModel
func (ModelBuildModel) TableName() string {
	return "model_builds"
}

// BeforeCreate generates UUID before creating record
func (m *ModelBuildModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
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
	bytes, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
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

