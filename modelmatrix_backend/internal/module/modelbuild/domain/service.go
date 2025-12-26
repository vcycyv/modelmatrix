package domain

import (
	"errors"
	"strings"
)

// Domain errors
var (
	ErrBuildNameEmpty        = errors.New("build name cannot be empty")
	ErrBuildNameExists       = errors.New("build name already exists")
	ErrInvalidModelType      = errors.New("invalid model type")
	ErrInvalidBuildStatus    = errors.New("invalid build status")
	ErrBuildNotFound         = errors.New("model build not found")
	ErrBuildAlreadyRunning   = errors.New("build is already running")
	ErrBuildNotPending       = errors.New("build is not in pending status")
	ErrBuildCannotBeCancelled = errors.New("build cannot be cancelled")
	ErrDatasourceNotReady    = errors.New("datasource is not ready for training")
)

// Service provides domain business logic for model builds (Domain Service)
type Service struct{}

// NewService creates a new domain service
func NewService() *Service {
	return &Service{}
}

// ValidateBuild validates a model build entity
func (s *Service) ValidateBuild(build *ModelBuild) error {
	if strings.TrimSpace(build.Name) == "" {
		return ErrBuildNameEmpty
	}

	if !build.ModelType.IsValid() {
		return ErrInvalidModelType
	}

	return nil
}

// ValidateBuildNameUnique checks if build name is unique
func (s *Service) ValidateBuildNameUnique(name string, existingNames []string) error {
	for _, existing := range existingNames {
		if strings.EqualFold(existing, name) {
			return ErrBuildNameExists
		}
	}
	return nil
}

// CanStartBuild checks if a build can be started
func (s *Service) CanStartBuild(build *ModelBuild) error {
	if !build.CanStart() {
		return ErrBuildNotPending
	}
	return nil
}

// CanCancelBuild checks if a build can be cancelled
func (s *Service) CanCancelBuild(build *ModelBuild) error {
	if !build.CanCancel() {
		return ErrBuildCannotBeCancelled
	}
	return nil
}

// ValidateParameters validates training parameters
func (s *Service) ValidateParameters(params *TrainingParameters) error {
	// Implement parameter validation logic here
	// For now, basic validation
	if params.TrainTestSplit <= 0 || params.TrainTestSplit >= 1 {
		params.TrainTestSplit = 0.8 // Default 80/20 split
	}
	if params.MaxIterations <= 0 {
		params.MaxIterations = 100
	}
	return nil
}

// GetDefaultParameters returns default training parameters for a model type
func (s *Service) GetDefaultParameters(modelType ModelType) TrainingParameters {
	params := TrainingParameters{
		TrainTestSplit:  0.8,
		RandomSeed:      42,
		MaxIterations:   100,
		EarlyStopRounds: 10,
		Hyperparameters: make(map[string]interface{}),
	}

	switch modelType {
	case ModelTypeClassification:
		params.Algorithm = "random_forest"
		params.Hyperparameters["n_estimators"] = 100
		params.Hyperparameters["max_depth"] = 10
	case ModelTypeRegression:
		params.Algorithm = "gradient_boosting"
		params.Hyperparameters["n_estimators"] = 100
		params.Hyperparameters["learning_rate"] = 0.1
	case ModelTypeClustering:
		params.Algorithm = "kmeans"
		params.Hyperparameters["n_clusters"] = 5
	}

	return params
}

