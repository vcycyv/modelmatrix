package dto

import (
	"time"
)

// CreateBuildRequest represents request to create a model build
type CreateBuildRequest struct {
	Name         string                     `json:"name" binding:"required,min=1,max=255" example:"Sales Predictor v1"`
	Description  string                     `json:"description" binding:"max=1000" example:"Random forest model for sales prediction"`
	DatasourceID string                     `json:"datasource_id" binding:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ModelType    string                     `json:"model_type" binding:"required,oneof=classification regression clustering" example:"regression"`
	Parameters   *TrainingParametersRequest `json:"parameters,omitempty"`
}

// TrainingParametersRequest represents training parameters
type TrainingParametersRequest struct {
	Algorithm       string                 `json:"algorithm" example:"random_forest"`
	Hyperparameters map[string]interface{} `json:"hyperparameters,omitempty"`
	TrainTestSplit  float64                `json:"train_test_split" example:"0.8"`
	RandomSeed      int                    `json:"random_seed" example:"42"`
	MaxIterations   int                    `json:"max_iterations" example:"100"`
	EarlyStopRounds int                    `json:"early_stop_rounds" example:"10"`
}

// UpdateBuildRequest represents request to update a model build
type UpdateBuildRequest struct {
	Name        *string `json:"name,omitempty" binding:"omitempty,min=1,max=255" example:"Updated Name"`
	Description *string `json:"description,omitempty" binding:"omitempty,max=1000" example:"Updated description"`
}

// BuildResponse represents a model build in responses
type BuildResponse struct {
	ID             string                      `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name           string                      `json:"name" example:"Sales Predictor v1"`
	Description    string                      `json:"description" example:"Random forest model for sales prediction"`
	DatasourceID   string                      `json:"datasource_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	DatasourceName string                      `json:"datasource_name,omitempty" example:"Sales Data 2024"`
	ModelType      string                      `json:"model_type" example:"regression"`
	Status         string                      `json:"status" example:"completed"`
	Parameters     *TrainingParametersResponse `json:"parameters,omitempty"`
	Metrics        *MetricsResponse            `json:"metrics,omitempty"`
	ErrorMessage   string                      `json:"error_message,omitempty" example:""`
	StartedAt      *time.Time                  `json:"started_at,omitempty" example:"2024-01-15T10:30:00Z"`
	CompletedAt    *time.Time                  `json:"completed_at,omitempty" example:"2024-01-15T11:00:00Z"`
	CreatedBy      string                      `json:"created_by" example:"admin"`
	CreatedAt      time.Time                   `json:"created_at" example:"2024-01-15T10:30:00Z"`
	UpdatedAt      time.Time                   `json:"updated_at" example:"2024-01-15T11:00:00Z"`
}

// TrainingParametersResponse represents training parameters in responses
type TrainingParametersResponse struct {
	Algorithm       string                 `json:"algorithm" example:"random_forest"`
	Hyperparameters map[string]interface{} `json:"hyperparameters,omitempty"`
	TrainTestSplit  float64                `json:"train_test_split" example:"0.8"`
	RandomSeed      int                    `json:"random_seed" example:"42"`
	MaxIterations   int                    `json:"max_iterations" example:"100"`
	EarlyStopRounds int                    `json:"early_stop_rounds" example:"10"`
}

// MetricsResponse represents model metrics in responses
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

// BuildListResponse represents a list of model builds
type BuildListResponse struct {
	Builds []BuildResponse `json:"builds"`
	Total  int64           `json:"total"`
}

// ListParams represents common list query parameters
type ListParams struct {
	Page     int    `form:"page" binding:"omitempty,min=1" example:"1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100" example:"20"`
	Search   string `form:"search" example:"sales"`
	Status   string `form:"status" binding:"omitempty,oneof=pending running completed failed cancelled" example:"completed"`
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

