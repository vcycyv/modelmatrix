package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CollectionModel is the GORM model for collections
type CollectionModel struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name        string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_collection_name"`
	Description string         `gorm:"type:text"`
	CreatedBy   string         `gorm:"type:varchar(255);not null"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	// Relations
	Datasources []DatasourceModel `gorm:"foreignKey:CollectionID;constraint:OnDelete:RESTRICT"`
}

// TableName returns the table name for CollectionModel
func (CollectionModel) TableName() string {
	return "collections"
}

// BeforeCreate generates UUID before creating record
func (c *CollectionModel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// DatasourceModel is the GORM model for datasources
type DatasourceModel struct {
	ID               string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CollectionID     string         `gorm:"type:uuid;not null;index:idx_datasource_collection"`
	Name             string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_datasource_name_collection,where:deleted_at IS NULL"`
	Description      string         `gorm:"type:text"`
	Type             string         `gorm:"type:varchar(50);not null"`
	FilePath         string         `gorm:"type:varchar(1024)"`
	ConnectionConfig JSONMap        `gorm:"type:jsonb"`
	CreatedBy        string         `gorm:"type:varchar(255);not null"`
	CreatedAt        time.Time      `gorm:"autoCreateTime"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`

	// Relations
	Collection CollectionModel `gorm:"foreignKey:CollectionID"`
	Columns    []ColumnModel   `gorm:"foreignKey:DatasourceID;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for DatasourceModel
func (DatasourceModel) TableName() string {
	return "datasources"
}

// BeforeCreate generates UUID before creating record
func (d *DatasourceModel) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}

// ColumnModel is the GORM model for datasource columns
type ColumnModel struct {
	ID           string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DatasourceID string         `gorm:"type:uuid;not null;index:idx_column_datasource;uniqueIndex:idx_column_name_datasource,priority:1"`
	Name         string         `gorm:"type:varchar(255);not null;uniqueIndex:idx_column_name_datasource,priority:2"`
	DataType     string         `gorm:"type:varchar(100);not null"`
	Role         string         `gorm:"type:varchar(50);not null;default:'input'"`
	Description  string         `gorm:"type:text"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	// Relations
	Datasource DatasourceModel `gorm:"foreignKey:DatasourceID"`
}

// BeforeCreate generates UUID before creating record
func (c *ColumnModel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// TableName returns the table name for ColumnModel
func (ColumnModel) TableName() string {
	return "datasource_columns"
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

