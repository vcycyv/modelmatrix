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
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_model_name"`
	Description  string    `gorm:"type:text"`
	BuildID      string    `gorm:"type:uuid;not null;index:idx_model_build"`
	DatasourceID string    `gorm:"type:uuid;not null;index:idx_model_datasource"`
	Algorithm    string    `gorm:"type:varchar(100);not null"`
	ModelType    string    `gorm:"type:varchar(50);not null"` // classification, regression
	TargetColumn string    `gorm:"type:varchar(255);not null"`
	Status       string    `gorm:"type:varchar(50);not null;default:'draft';index:idx_model_status"`
	Metrics      JSONMap   `gorm:"type:jsonb"`
	Version      int       `gorm:"not null;default:1"`
	CreatedBy    string    `gorm:"type:varchar(255);not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`

	// Relations
	Variables []ModelVariableModel `gorm:"foreignKey:ModelID;constraint:OnDelete:CASCADE"`
	Files     []ModelFileModel     `gorm:"foreignKey:ModelID;constraint:OnDelete:CASCADE"`
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

// ModelVariableModel is the GORM model for model input/output variables
type ModelVariableModel struct {
	ID           string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ModelID      string    `gorm:"type:uuid;not null;index:idx_variable_model"`
	Name         string    `gorm:"type:varchar(255);not null"`
	DataType     string    `gorm:"type:varchar(50);not null"`  // numeric, categorical, boolean
	Role         string    `gorm:"type:varchar(50);not null"`  // input, target
	Importance   *float64  `gorm:"type:decimal(10,6)"`         // Feature importance (0.0-1.0)
	Statistics   JSONMap   `gorm:"type:jsonb"`                 // min, max, mean, std, etc.
	EncodingInfo JSONMap   `gorm:"type:jsonb"`                 // For categorical: mapping info
	Ordinal      int       `gorm:"not null"`                   // Order for prediction
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

// TableName returns the table name for ModelVariableModel
func (ModelVariableModel) TableName() string {
	return "model_variables"
}

// BeforeCreate generates UUID before creating record
func (v *ModelVariableModel) BeforeCreate(tx *gorm.DB) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	return nil
}

// ModelFileModel is the GORM model for model files
type ModelFileModel struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ModelID     string    `gorm:"type:uuid;not null;index:idx_file_model"`
	FileType    string    `gorm:"type:varchar(50);not null"`   // model, preprocessor, metadata, feature_names
	FilePath    string    `gorm:"type:varchar(500);not null"`  // MinIO path
	FileName    string    `gorm:"type:varchar(255);not null"`  // Original filename
	FileSize    *int64    `gorm:"type:bigint"`                 // Size in bytes
	Checksum    string    `gorm:"type:varchar(64)"`            // SHA256 for integrity
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

// TableName returns the table name for ModelFileModel
func (ModelFileModel) TableName() string {
	return "model_files"
}

// BeforeCreate generates UUID before creating record
func (f *ModelFileModel) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = uuid.New().String()
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
