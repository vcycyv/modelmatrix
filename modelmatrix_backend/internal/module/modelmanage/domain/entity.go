package domain

import (
	"time"
)

// ModelStatus represents the status of a model version
type ModelStatus string

const (
	ModelStatusDraft      ModelStatus = "draft"
	ModelStatusActive     ModelStatus = "active"
	ModelStatusInactive   ModelStatus = "inactive"
	ModelStatusDeprecated ModelStatus = "deprecated"
)

// IsValid checks if the model status is valid
func (s ModelStatus) IsValid() bool {
	switch s {
	case ModelStatusDraft, ModelStatusActive, ModelStatusInactive, ModelStatusDeprecated:
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

// Model represents a trained ML model (Domain Entity)
type Model struct {
	ID           string
	Name         string
	Description  string
	BuildID      string // Reference to the build that created this model
	Status       ModelStatus
	ArtifactPath string // Path to model artifacts
	Metadata     *ModelMetadata
	CreatedBy    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ModelMetadata holds model-specific metadata
type ModelMetadata struct {
	Algorithm     string                 `json:"algorithm"`
	ModelType     string                 `json:"model_type"`
	InputFeatures []string               `json:"input_features"`
	TargetFeature string                 `json:"target_feature"`
	Framework     string                 `json:"framework"`
	Version       string                 `json:"version"`
	Metrics       map[string]float64     `json:"metrics"`
	Custom        map[string]interface{} `json:"custom,omitempty"`
}

// ModelVersion represents a version of a model (Domain Entity)
type ModelVersion struct {
	ID           string
	ModelID      string
	Version      string
	BuildID      string
	Status       ModelStatus
	ArtifactPath string
	Metrics      map[string]float64
	Notes        string
	CreatedBy    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
func (m *Model) Activate() {
	m.Status = ModelStatusActive
}

// Deactivate deactivates the model
func (m *Model) Deactivate() {
	m.Status = ModelStatusInactive
}

// Deprecate marks the model as deprecated
func (m *Model) Deprecate() {
	m.Status = ModelStatusDeprecated
}

