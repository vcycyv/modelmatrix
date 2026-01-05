package dto

import (
	"time"
)

// CreateModelRequest represents request to create a model (usually from build completion)
type CreateModelRequest struct {
	Name         string                 `json:"name" binding:"required,min=1,max=255" example:"Sales Predictor"`
	Description  string                 `json:"description" binding:"max=1000" example:"Random forest model for sales prediction"`
	BuildID      string                 `json:"build_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	DatasourceID string                 `json:"datasource_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440001"`
	Algorithm    string                 `json:"algorithm" binding:"required" example:"random_forest"`
	ModelType    string                 `json:"model_type" binding:"required,oneof=classification regression clustering" example:"classification"`
	TargetColumn string                 `json:"target_column" binding:"required" example:"BAD"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	Variables    []CreateVariableRequest `json:"variables,omitempty"`
	Files        []CreateFileRequest     `json:"files,omitempty"`
}

// CreateVariableRequest represents a variable to create with the model
type CreateVariableRequest struct {
	Name         string                 `json:"name" binding:"required" example:"LOAN"`
	DataType     string                 `json:"data_type" binding:"required,oneof=numeric categorical boolean text" example:"numeric"`
	Role         string                 `json:"role" binding:"required,oneof=input target" example:"input"`
	Importance   *float64               `json:"importance,omitempty" example:"0.25"`
	Statistics   map[string]interface{} `json:"statistics,omitempty"`
	EncodingInfo map[string]interface{} `json:"encoding_info,omitempty"`
	Ordinal      int                    `json:"ordinal" example:"1"`
}

// CreateFileRequest represents a file to create with the model
type CreateFileRequest struct {
	FileType    string `json:"file_type" binding:"required,oneof=model preprocessor metadata feature_names" example:"model"`
	FilePath    string `json:"file_path" binding:"required" example:"minio://modelmatrix/models/random_forest/abc123.pkl"`
	FileName    string `json:"file_name" binding:"required" example:"random_forest_v1.pkl"`
	FileSize    *int64 `json:"file_size,omitempty" example:"1024000"`
	Checksum    string `json:"checksum,omitempty" example:"sha256:abc123..."`
	Description string `json:"description,omitempty" example:"Main trained model file"`
}

// UpdateModelRequest represents request to update a model
type UpdateModelRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=255" example:"Updated Name"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=1000" example:"Updated description"`
}

// ModelResponse represents a model in responses
type ModelResponse struct {
	ID           string                 `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name         string                 `json:"name" example:"Sales Predictor"`
	Description  string                 `json:"description" example:"Random forest model for sales prediction"`
	BuildID      string                 `json:"build_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	DatasourceID string                 `json:"datasource_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	ProjectID    *string                `json:"project_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440003"`
	FolderID     *string                `json:"folder_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440004"`
	Algorithm    string                 `json:"algorithm" example:"random_forest"`
	ModelType    string                 `json:"model_type" example:"classification"`
	TargetColumn string                 `json:"target_column" example:"BAD"`
	Status       string                 `json:"status" example:"active"`
	Metrics      *MetricsResponse       `json:"metrics,omitempty"`
	Version      int                    `json:"version" example:"1"`
	CreatedBy    string                 `json:"created_by" example:"admin"`
	CreatedAt    time.Time              `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt    time.Time              `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// MetricsResponse represents model training metrics
type MetricsResponse struct {
	Accuracy  float64 `json:"accuracy,omitempty" example:"0.95"`
	Precision float64 `json:"precision,omitempty" example:"0.94"`
	Recall    float64 `json:"recall,omitempty" example:"0.93"`
	F1Score   float64 `json:"f1_score,omitempty" example:"0.935"`
	MSE       float64 `json:"mse,omitempty" example:"0.05"`
	RMSE      float64 `json:"rmse,omitempty" example:"0.22"`
	MAE       float64 `json:"mae,omitempty" example:"0.15"`
	R2        float64 `json:"r2,omitempty" example:"0.92"`
}

// ModelDetailResponse includes variables and files
type ModelDetailResponse struct {
	ModelResponse
	Variables []VariableResponse `json:"variables"`
	Files     []FileResponse     `json:"files"`
}

// VariableResponse represents a model variable in responses
type VariableResponse struct {
	ID           string                 `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ModelID      string                 `json:"model_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name         string                 `json:"name" example:"LOAN"`
	DataType     string                 `json:"data_type" example:"numeric"`
	Role         string                 `json:"role" example:"input"`
	Importance   *float64               `json:"importance,omitempty" example:"0.25"`
	Statistics   map[string]interface{} `json:"statistics,omitempty"`
	EncodingInfo map[string]interface{} `json:"encoding_info,omitempty"`
	Ordinal      int                    `json:"ordinal" example:"1"`
	CreatedAt    time.Time              `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// FileResponse represents a model file in responses
type FileResponse struct {
	ID          string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ModelID     string    `json:"model_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	FileType    string    `json:"file_type" example:"model"`
	FilePath    string    `json:"file_path" example:"minio://modelmatrix/models/random_forest/abc123.pkl"`
	FileName    string    `json:"file_name" example:"random_forest_v1.pkl"`
	FileSize    *int64    `json:"file_size,omitempty" example:"1024000"`
	Checksum    string    `json:"checksum,omitempty" example:"sha256:abc123..."`
	Description string    `json:"description,omitempty" example:"Main trained model file"`
	CreatedAt   time.Time `json:"created_at" example:"2024-01-15T10:30:00Z"`
}

// ModelListResponse represents a list of models
type ModelListResponse struct {
	Models []ModelResponse `json:"models"`
	Total  int64           `json:"total"`
}

// ListParams represents common list query parameters
type ListParams struct {
	Page     int    `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100" example:"20"`
	Search   string `form:"search" example:"sales"`
	Status   string `form:"status" binding:"omitempty,oneof=draft active inactive archived" example:"active"`
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

// CreateModelFromBuildRequest is used internally when creating a model from a completed build
type CreateModelFromBuildRequest struct {
	BuildID       string
	Name          string
	Description   string
	DatasourceID  string
	ProjectID     *string // Inherits from build
	FolderID      *string // Inherits from build
	Algorithm     string
	ModelType     string
	TargetColumn  string
	InputColumns  []string
	ModelFilePath string
	Metrics       map[string]interface{}
	CreatedBy     string
}
