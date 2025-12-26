package domain

import (
	"errors"
	"strings"
)

// Domain errors
var (
	ErrModelNameEmpty           = errors.New("model name cannot be empty")
	ErrModelNameExists          = errors.New("model name already exists")
	ErrModelNotFound            = errors.New("model not found")
	ErrModelVersionNotFound     = errors.New("model version not found")
	ErrInvalidModelStatus       = errors.New("invalid model status")
	ErrActiveModelCannotBeDeleted = errors.New("active models cannot be deleted")
	ErrModelCannotBeActivated   = errors.New("model cannot be activated in current state")
	ErrModelCannotBeDeactivated = errors.New("model cannot be deactivated in current state")
	ErrModelAlreadyActive       = errors.New("model is already active")
	ErrModelAlreadyInactive     = errors.New("model is already inactive")
	ErrVersionExists            = errors.New("version already exists for this model")
)

// Service provides domain business logic for model management (Domain Service)
type Service struct{}

// NewService creates a new domain service
func NewService() *Service {
	return &Service{}
}

// ValidateModel validates a model entity
func (s *Service) ValidateModel(model *Model) error {
	if strings.TrimSpace(model.Name) == "" {
		return ErrModelNameEmpty
	}

	if !model.Status.IsValid() {
		return ErrInvalidModelStatus
	}

	return nil
}

// ValidateModelNameUnique checks if model name is unique
func (s *Service) ValidateModelNameUnique(name string, existingNames []string) error {
	for _, existing := range existingNames {
		if strings.EqualFold(existing, name) {
			return ErrModelNameExists
		}
	}
	return nil
}

// CanDeleteModel checks if a model can be deleted
func (s *Service) CanDeleteModel(model *Model) error {
	if !model.CanBeDeleted() {
		return ErrActiveModelCannotBeDeleted
	}
	return nil
}

// CanActivateModel checks if a model can be activated
func (s *Service) CanActivateModel(model *Model) error {
	if !model.CanBeActivated() {
		if model.Status == ModelStatusActive {
			return ErrModelAlreadyActive
		}
		return ErrModelCannotBeActivated
	}
	return nil
}

// CanDeactivateModel checks if a model can be deactivated
func (s *Service) CanDeactivateModel(model *Model) error {
	if !model.CanBeDeactivated() {
		if model.Status == ModelStatusInactive || model.Status == ModelStatusDraft {
			return ErrModelAlreadyInactive
		}
		return ErrModelCannotBeDeactivated
	}
	return nil
}

// ValidateVersion validates a model version
func (s *Service) ValidateVersion(version *ModelVersion) error {
	if strings.TrimSpace(version.Version) == "" {
		return errors.New("version string cannot be empty")
	}
	return nil
}

// ValidateVersionUnique checks if version is unique for a model
func (s *Service) ValidateVersionUnique(version string, existingVersions []string) error {
	for _, existing := range existingVersions {
		if strings.EqualFold(existing, version) {
			return ErrVersionExists
		}
	}
	return nil
}

// GetNextVersion generates the next version string
func (s *Service) GetNextVersion(existingVersions []string) string {
	// Simple version incrementing (v1, v2, v3, ...)
	return "v" + string(rune(len(existingVersions)+1+'0'))
}

