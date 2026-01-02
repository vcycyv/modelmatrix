package domain

import (
	"errors"
	"time"
)

// Domain errors
var (
	ErrModelNotFound       = errors.New("model not found")
	ErrModelNameExists     = errors.New("model with this name already exists")
	ErrModelNameEmpty      = errors.New("model name cannot be empty")
	ErrModelCannotActivate = errors.New("model cannot be activated in current status")
	ErrModelCannotDeactivate = errors.New("model cannot be deactivated in current status")
	ErrModelCannotDelete   = errors.New("active model cannot be deleted")
	ErrInvalidModelStatus  = errors.New("invalid model status")
	ErrVariableNotFound    = errors.New("variable not found")
	ErrFileNotFound        = errors.New("file not found")
)

// ModelStatus represents the status of a model
type ModelStatus string

const (
	ModelStatusDraft      ModelStatus = "draft"
	ModelStatusActive     ModelStatus = "active"
	ModelStatusInactive   ModelStatus = "inactive"
	ModelStatusArchived   ModelStatus = "archived"
)

// IsValid checks if the model status is valid
func (s ModelStatus) IsValid() bool {
	switch s {
	case ModelStatusDraft, ModelStatusActive, ModelStatusInactive, ModelStatusArchived:
		return true
	default:
		return false
	}
}

// CanDelete checks if model with this status can be deleted
func (s ModelStatus) CanDelete() bool {
	return s != ModelStatusActive
}

// CanActivate checks if model with this status can be activated
func (s ModelStatus) CanActivate() bool {
	return s == ModelStatusDraft || s == ModelStatusInactive
}

// CanDeactivate checks if model with this status can be deactivated
func (s ModelStatus) CanDeactivate() bool {
	return s == ModelStatusActive
}

// VariableRole represents the role of a variable
type VariableRole string

const (
	VariableRoleInput  VariableRole = "input"
	VariableRoleTarget VariableRole = "target"
)

// VariableDataType represents the data type of a variable
type VariableDataType string

const (
	VariableDataTypeNumeric     VariableDataType = "numeric"
	VariableDataTypeCategorical VariableDataType = "categorical"
	VariableDataTypeBoolean     VariableDataType = "boolean"
	VariableDataTypeText        VariableDataType = "text"
)

// FileType represents the type of model file
type FileType string

const (
	FileTypeModel        FileType = "model"        // Main trained model (.pkl)
	FileTypePreprocessor FileType = "preprocessor" // Feature preprocessor
	FileTypeMetadata     FileType = "metadata"     // Model metadata JSON
	FileTypeFeatureNames FileType = "feature_names" // Feature names list
)

// Model represents a trained ML model (Domain Entity)
type Model struct {
	ID           string
	Name         string
	Description  string
	BuildID      string       // Reference to the build that created this model
	DatasourceID string       // Reference to the datasource used for training
	Algorithm    string       // Algorithm used (random_forest, xgboost, etc.)
	ModelType    string       // classification, regression, clustering
	TargetColumn string       // Target column name
	Status       ModelStatus
	Metrics      *ModelMetrics
	Version      int
	CreatedBy    string
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Relations (loaded separately)
	Variables []ModelVariable
	Files     []ModelFile
}

// ModelMetrics holds model training metrics
type ModelMetrics struct {
	Accuracy  float64 `json:"accuracy,omitempty"`
	Precision float64 `json:"precision,omitempty"`
	Recall    float64 `json:"recall,omitempty"`
	F1Score   float64 `json:"f1_score,omitempty"`
	MSE       float64 `json:"mse,omitempty"`
	RMSE      float64 `json:"rmse,omitempty"`
	MAE       float64 `json:"mae,omitempty"`
	R2        float64 `json:"r2,omitempty"`
}

// ModelVariable represents an input or output variable of the model
type ModelVariable struct {
	ID           string
	ModelID      string
	Name         string                 // Column name
	DataType     VariableDataType       // numeric, categorical, boolean
	Role         VariableRole           // input, target
	Importance   *float64               // Feature importance (0.0-1.0)
	Statistics   map[string]interface{} // min, max, mean, std, etc.
	EncodingInfo map[string]interface{} // For categorical: mapping info
	Ordinal      int                    // Order for prediction
	CreatedAt    time.Time
}

// ModelFile represents a file associated with the model
type ModelFile struct {
	ID          string
	ModelID     string
	FileType    FileType // model, preprocessor, metadata
	FilePath    string   // MinIO path
	FileName    string   // Original filename
	FileSize    *int64   // Size in bytes
	Checksum    string   // SHA256 for integrity
	Description string
	CreatedAt   time.Time
}

// CanBeDeleted checks if the model can be deleted
func (m *Model) CanBeDeleted() bool {
	return m.Status.CanDelete()
}

// CanBeActivated checks if the model can be activated
func (m *Model) CanBeActivated() bool {
	return m.Status.CanActivate()
}

// CanBeDeactivated checks if the model can be deactivated
func (m *Model) CanBeDeactivated() bool {
	return m.Status.CanDeactivate()
}

// Activate activates the model
func (m *Model) Activate() error {
	if !m.CanBeActivated() {
		return ErrModelCannotActivate
	}
	m.Status = ModelStatusActive
	return nil
}

// Deactivate deactivates the model
func (m *Model) Deactivate() error {
	if !m.CanBeDeactivated() {
		return ErrModelCannotDeactivate
	}
	m.Status = ModelStatusInactive
	return nil
}

// Archive marks the model as archived
func (m *Model) Archive() {
	m.Status = ModelStatusArchived
}

// GetInputVariables returns only input variables
func (m *Model) GetInputVariables() []ModelVariable {
	var inputs []ModelVariable
	for _, v := range m.Variables {
		if v.Role == VariableRoleInput {
			inputs = append(inputs, v)
		}
	}
	return inputs
}

// GetTargetVariable returns the target variable (usually one)
func (m *Model) GetTargetVariable() *ModelVariable {
	for _, v := range m.Variables {
		if v.Role == VariableRoleTarget {
			return &v
		}
	}
	return nil
}

// GetMainModelFile returns the main model file (.pkl)
func (m *Model) GetMainModelFile() *ModelFile {
	for _, f := range m.Files {
		if f.FileType == FileTypeModel {
			return &f
		}
	}
	return nil
}
