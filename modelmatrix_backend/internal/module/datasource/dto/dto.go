package dto

import (
	"time"
)

// ===============================
// Collection DTOs
// ===============================

// CreateCollectionRequest represents request to create a collection
type CreateCollectionRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255" example:"Training Data"`
	Description string `json:"description" binding:"max=1000" example:"Collection for ML training datasets"`
}

// UpdateCollectionRequest represents request to update a collection
type UpdateCollectionRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=255" example:"Updated Name"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=1000" example:"Updated description"`
}

// CollectionResponse represents a collection in responses
type CollectionResponse struct {
	ID              string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name            string    `json:"name" example:"Training Data"`
	Description     string    `json:"description" example:"Collection for ML training datasets"`
	DatasourceCount int       `json:"datasource_count" example:"5"`
	CreatedBy       string    `json:"created_by" example:"admin"`
	CreatedAt       time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt       time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// CollectionListResponse represents a list of collections
type CollectionListResponse struct {
	Collections []CollectionResponse `json:"collections"`
	Total       int64                `json:"total"`
}

// ===============================
// Datasource DTOs
// ===============================

// CreateDatasourceRequest represents request to create a datasource
type CreateDatasourceRequest struct {
	CollectionID     string                   `json:"collection_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name             string                   `json:"name" binding:"required,min=1,max=255" example:"Sales Data 2024"`
	Description      string                   `json:"description" binding:"max=1000" example:"Sales dataset for 2024"`
	Type             string                   `json:"type" binding:"required,oneof=postgresql mysql csv parquet" example:"csv"`
	ConnectionConfig *ConnectionConfigRequest `json:"connection_config,omitempty"`
}

// ConnectionConfigRequest represents database connection configuration
type ConnectionConfigRequest struct {
	Host     string `json:"host" binding:"required" example:"localhost"`
	Port     int    `json:"port" binding:"required" example:"5432"`
	Database string `json:"database" binding:"required" example:"sales_db"`
	Username string `json:"username" binding:"required" example:"reader"`
	Password string `json:"password" binding:"required" example:"secret"`
	Schema   string `json:"schema" example:"public"`
	Table    string `json:"table" binding:"required" example:"sales"`
	SSLMode  string `json:"sslmode" example:"disable"`
}

// UpdateDatasourceRequest represents request to update a datasource
type UpdateDatasourceRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=255" example:"Updated Name"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=1000" example:"Updated description"`
}

// DatasourceResponse represents a datasource in responses
type DatasourceResponse struct {
	ID               string                    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CollectionID     string                    `json:"collection_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	CollectionName   string                    `json:"collection_name" example:"Training Data"`
	Name             string                    `json:"name" example:"Sales Data 2024"`
	Description      string                    `json:"description" example:"Sales dataset for 2024"`
	Type             string                    `json:"type" example:"csv"`
	FilePath         string                    `json:"file_path,omitempty" example:"datasources/sales_2024.csv"`
	ConnectionConfig *ConnectionConfigResponse `json:"connection_config,omitempty"`
	ColumnCount      int                       `json:"column_count" example:"10"`
	CreatedBy        string                    `json:"created_by" example:"admin"`
	CreatedAt        time.Time                 `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt        time.Time                 `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// ConnectionConfigResponse represents database connection configuration in responses
type ConnectionConfigResponse struct {
	Host     string `json:"host" example:"localhost"`
	Port     int    `json:"port" example:"5432"`
	Database string `json:"database" example:"sales_db"`
	Username string `json:"username" example:"reader"`
	Schema   string `json:"schema" example:"public"`
	Table    string `json:"table" example:"sales"`
	SSLMode  string `json:"sslmode" example:"disable"`
}

// DatasourceListResponse represents a list of datasources
type DatasourceListResponse struct {
	Datasources []DatasourceResponse `json:"datasources"`
	Total       int64                `json:"total"`
}

// DatasourceDetailResponse includes columns
type DatasourceDetailResponse struct {
	DatasourceResponse
	Columns []ColumnResponse `json:"columns"`
}

// ===============================
// Column DTOs
// ===============================

// ColumnResponse represents a column in responses
type ColumnResponse struct {
	ID          string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name        string    `json:"name" example:"price"`
	DataType    string    `json:"data_type" example:"float64"`
	Role        string    `json:"role" example:"input"`
	Description string    `json:"description" example:"Product price"`
	CreatedAt   time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// UpdateColumnRoleRequest represents request to update a column's role
type UpdateColumnRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=target input output ignore" example:"target"`
}

// BulkUpdateColumnRolesRequest represents request to update multiple column roles
type BulkUpdateColumnRolesRequest struct {
	Columns []ColumnRoleUpdate `json:"columns" binding:"required,dive"`
}

// ColumnRoleUpdate represents a single column role update
type ColumnRoleUpdate struct {
	ColumnID string `json:"column_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Role     string `json:"role" binding:"required,oneof=target input output ignore" example:"input"`
}

// CreateColumnsRequest represents request to add columns to a datasource
type CreateColumnsRequest struct {
	Columns []CreateColumnRequest `json:"columns" binding:"required,dive"`
}

// CreateColumnRequest represents a single column creation
type CreateColumnRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255" example:"price"`
	DataType    string `json:"data_type" binding:"required" example:"float64"`
	Role        string `json:"role" binding:"required,oneof=target input output ignore" example:"input"`
	Description string `json:"description" binding:"max=500" example:"Product price"`
}

// ===============================
// File Upload DTOs
// ===============================

// FileUploadResponse represents the response after file upload
type FileUploadResponse struct {
	DatasourceID string           `json:"datasource_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	FilePath     string           `json:"file_path" example:"datasources/sales_2024.csv"`
	FileSize     int64            `json:"file_size" example:"1024000"`
	Columns      []ColumnResponse `json:"columns"`
}

// ===============================
// Query Parameters
// ===============================

// ListParams represents common list query parameters
type ListParams struct {
	Page     int    `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100" example:"20"`
	Search   string `form:"search" example:"sales"`
	SortBy   string `form:"sort_by" example:"created_at"`
	SortDir  string `form:"sort_dir" binding:"omitempty,oneof=asc desc" example:"desc"`
}

// SetDefaults sets default values for list params
func (p *ListParams) SetDefaults() {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PageSize == 0 {
		p.PageSize = 20
	}
	if p.SortDir == "" {
		p.SortDir = "desc"
	}
	if p.SortBy == "" {
		p.SortBy = "created_at"
	}
}

// Offset calculates the offset for pagination
func (p *ListParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

