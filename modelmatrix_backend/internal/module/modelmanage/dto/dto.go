package dto

import (
	"time"
)

// CreateModelRequest represents request to create a model
type CreateModelRequest struct {
	Name        string                `json:"name" binding:"required,min=1,max=255" example:"Sales Predictor"`
	Description string                `json:"description" binding:"max=1000" example:"Production sales prediction model"`
	BuildID     string                `json:"build_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Metadata    *ModelMetadataRequest `json:"metadata,omitempty"`
}

// ModelMetadataRequest represents model metadata
type ModelMetadataRequest struct {
	Algorithm     string                 `json:"algorithm" example:"random_forest"`
	ModelType     string                 `json:"model_type" example:"regression"`
	InputFeatures []string               `json:"input_features" example:"price,quantity"`
	TargetFeature string                 `json:"target_feature" example:"sales"`
	Framework     string                 `json:"framework" example:"scikit-learn"`
	Version       string                 `json:"version" example:"1.0.0"`
	Custom        map[string]interface{} `json:"custom,omitempty"`
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
	Description  string                 `json:"description" example:"Production sales prediction model"`
	BuildID      string                 `json:"build_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Status       string                 `json:"status" example:"active"`
	ArtifactPath string                 `json:"artifact_path,omitempty" example:"models/1/artifact.pkl"`
	Metadata     *ModelMetadataResponse `json:"metadata,omitempty"`
	VersionCount int                    `json:"version_count" example:"3"`
	CreatedBy    string                 `json:"created_by" example:"admin"`
	CreatedAt    time.Time              `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt    time.Time              `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// ModelMetadataResponse represents model metadata in responses
type ModelMetadataResponse struct {
	Algorithm     string                 `json:"algorithm" example:"random_forest"`
	ModelType     string                 `json:"model_type" example:"regression"`
	InputFeatures []string               `json:"input_features" example:"price,quantity"`
	TargetFeature string                 `json:"target_feature" example:"sales"`
	Framework     string                 `json:"framework" example:"scikit-learn"`
	Version       string                 `json:"version" example:"1.0.0"`
	Metrics       map[string]float64     `json:"metrics,omitempty"`
	Custom        map[string]interface{} `json:"custom,omitempty"`
}

// ModelListResponse represents a list of models
type ModelListResponse struct {
	Models []ModelResponse `json:"models"`
	Total  int64           `json:"total"`
}

// ModelDetailResponse includes versions
type ModelDetailResponse struct {
	ModelResponse
	Versions []VersionResponse `json:"versions"`
}

// CreateVersionRequest represents request to create a model version
type CreateVersionRequest struct {
	Version      string             `json:"version" binding:"required,min=1,max=50" example:"v1.0.0"`
	BuildID      string             `json:"build_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ArtifactPath string             `json:"artifact_path" example:"models/1/v1/artifact.pkl"`
	Notes        string             `json:"notes" binding:"max=1000" example:"Initial release"`
	Metrics      map[string]float64 `json:"metrics,omitempty"`
}

// VersionResponse represents a model version in responses
type VersionResponse struct {
	ID           string             `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ModelID      string             `json:"model_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Version      string             `json:"version" example:"v1.0.0"`
	BuildID      string             `json:"build_id" example:"550e8400-e29b-41d4-a716-446655440002"`
	Status       string             `json:"status" example:"active"`
	ArtifactPath string             `json:"artifact_path,omitempty" example:"models/1/v1/artifact.pkl"`
	Metrics      map[string]float64 `json:"metrics,omitempty"`
	Notes        string             `json:"notes,omitempty" example:"Initial release"`
	CreatedBy    string             `json:"created_by" example:"admin"`
	CreatedAt    time.Time          `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt    time.Time          `json:"updated_at" example:"2024-01-15T10:30:00Z"`
}

// ListParams represents common list query parameters
type ListParams struct {
	Page     int    `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100" example:"20"`
	Search   string `form:"search" example:"sales"`
	Status   string `form:"status" binding:"omitempty,oneof=draft active inactive deprecated" example:"active"`
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
