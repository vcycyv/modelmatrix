package domain

import (
	"errors"
	"time"
)

// Version domain errors
var (
	ErrVersionNotFound = errors.New("model version not found")
)

// ModelVersion represents an immutable snapshot of a model (metadata + variables + files)
type ModelVersion struct {
	ID           string
	ModelID      string
	VersionNumber int
	// Snapshot of model fields at version time
	Name         string
	Description  string
	BuildID      string
	DatasourceID string
	ProjectID    *string
	FolderID     *string
	Algorithm    string
	ModelType    string
	TargetColumn string
	Status       ModelStatus
	Metrics      *ModelMetrics
	CreatedBy    string
	CreatedAt    time.Time

	Variables []ModelVariable
	Files     []ModelFile
}
