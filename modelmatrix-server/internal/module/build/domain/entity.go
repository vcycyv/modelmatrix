package domain

import (
	"time"
)

// BuildStatus represents the status of a model build
type BuildStatus string

const (
	BuildStatusPending    BuildStatus = "pending"
	BuildStatusRunning    BuildStatus = "running"
	BuildStatusCompleted  BuildStatus = "completed"
	BuildStatusFailed     BuildStatus = "failed"
	BuildStatusCancelled  BuildStatus = "cancelled"
)

// IsValid checks if the build status is valid
func (s BuildStatus) IsValid() bool {
	switch s {
	case BuildStatusPending, BuildStatusRunning, BuildStatusCompleted, BuildStatusFailed, BuildStatusCancelled:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the status is terminal (cannot change)
func (s BuildStatus) IsTerminal() bool {
	return s == BuildStatusCompleted || s == BuildStatusFailed || s == BuildStatusCancelled
}

// ModelType represents the type of ML model
type ModelType string

const (
	ModelTypeClassification ModelType = "classification"
	ModelTypeRegression     ModelType = "regression"
	ModelTypeClustering     ModelType = "clustering"
)

// IsValid checks if the model type is valid
func (t ModelType) IsValid() bool {
	switch t {
	case ModelTypeClassification, ModelTypeRegression, ModelTypeClustering:
		return true
	default:
		return false
	}
}

// ModelBuild represents a model training job (Domain Entity)
type ModelBuild struct {
	ID           string
	Name         string
	Description  string
	DatasourceID string
	ModelType    ModelType
	Status       BuildStatus
	Parameters   TrainingParameters
	Metrics      *BuildMetrics
	ErrorMessage string
	StartedAt    *time.Time
	CompletedAt  *time.Time
	CreatedBy    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TrainingParameters holds ML training configuration
type TrainingParameters struct {
	Algorithm       string                 `json:"algorithm"`
	Hyperparameters map[string]interface{} `json:"hyperparameters"`
	TrainTestSplit  float64                `json:"train_test_split"`
	RandomSeed      int                    `json:"random_seed"`
	MaxIterations   int                    `json:"max_iterations"`
	EarlyStopRounds int                    `json:"early_stop_rounds"`
}

// BuildMetrics holds model evaluation metrics
type BuildMetrics struct {
	Accuracy  float64 `json:"accuracy,omitempty"`
	Precision float64 `json:"precision,omitempty"`
	Recall    float64 `json:"recall,omitempty"`
	F1Score   float64 `json:"f1_score,omitempty"`
	MSE       float64 `json:"mse,omitempty"`
	RMSE      float64 `json:"rmse,omitempty"`
	MAE       float64 `json:"mae,omitempty"`
	R2        float64 `json:"r2,omitempty"`
}

// CanStart checks if the build can be started
func (b *ModelBuild) CanStart() bool {
	return b.Status == BuildStatusPending
}

// CanCancel checks if the build can be cancelled
func (b *ModelBuild) CanCancel() bool {
	return b.Status == BuildStatusPending || b.Status == BuildStatusRunning
}

// Start transitions build to running status
func (b *ModelBuild) Start() {
	now := time.Now()
	b.Status = BuildStatusRunning
	b.StartedAt = &now
}

// Complete transitions build to completed status
func (b *ModelBuild) Complete(metrics *BuildMetrics) {
	now := time.Now()
	b.Status = BuildStatusCompleted
	b.CompletedAt = &now
	b.Metrics = metrics
}

// Fail transitions build to failed status
func (b *ModelBuild) Fail(errorMsg string) {
	now := time.Now()
	b.Status = BuildStatusFailed
	b.CompletedAt = &now
	b.ErrorMessage = errorMsg
}

// Cancel transitions build to cancelled status
func (b *ModelBuild) Cancel() {
	now := time.Now()
	b.Status = BuildStatusCancelled
	b.CompletedAt = &now
}

