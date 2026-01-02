package domain

import (
	"strings"
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

// CanDeleteModel checks if a model can be deleted
func (s *Service) CanDeleteModel(model *Model) error {
	if !model.CanBeDeleted() {
		return ErrModelCannotDelete
	}
	return nil
}

// CanActivateModel checks if a model can be activated
func (s *Service) CanActivateModel(model *Model) error {
	if !model.CanBeActivated() {
		return ErrModelCannotActivate
	}
	return nil
}

// CanDeactivateModel checks if a model can be deactivated
func (s *Service) CanDeactivateModel(model *Model) error {
	if !model.CanBeDeactivated() {
		return ErrModelCannotDeactivate
	}
	return nil
}
