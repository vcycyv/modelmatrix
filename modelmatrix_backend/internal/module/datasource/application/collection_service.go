package application

import (
	"modelmatrix_backend/internal/module/datasource/domain"
	"modelmatrix_backend/internal/module/datasource/dto"
	"modelmatrix_backend/internal/module/datasource/repository"
	"modelmatrix_backend/pkg/logger"
)

// CollectionServiceImpl implements CollectionService
type CollectionServiceImpl struct {
	collectionRepo repository.CollectionRepository
	domainService  *domain.Service
}

// NewCollectionService creates a new collection service
func NewCollectionService(
	collectionRepo repository.CollectionRepository,
	domainService *domain.Service,
) CollectionService {
	return &CollectionServiceImpl{
		collectionRepo: collectionRepo,
		domainService:  domainService,
	}
}

// Create creates a new collection
func (s *CollectionServiceImpl) Create(req *dto.CreateCollectionRequest, createdBy string) (*dto.CollectionResponse, error) {
	// Convert DTO to domain entity
	collection := &domain.Collection{
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   createdBy,
	}

	// Validate using domain service
	if err := s.domainService.ValidateCollection(collection); err != nil {
		return nil, err
	}

	// Check name uniqueness
	existingNames, err := s.collectionRepo.GetAllNames()
	if err != nil {
		logger.Error("Failed to get collection names: %v", err)
		return nil, err
	}

	if err := s.domainService.ValidateCollectionNameUnique(collection.Name, existingNames); err != nil {
		return nil, err
	}

	// Create via repository
	if err := s.collectionRepo.Create(collection); err != nil {
		logger.Error("Failed to create collection: %v", err)
		return nil, err
	}

	logger.Audit(createdBy, "create", "collection", collection.ID, "success", nil)

	return toCollectionResponse(collection, 0), nil
}

// Update updates an existing collection
func (s *CollectionServiceImpl) Update(id string, req *dto.UpdateCollectionRequest) (*dto.CollectionResponse, error) {
	// Get existing collection
	collection, err := s.collectionRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.Name != nil {
		collection.Name = *req.Name
	}
	if req.Description != nil {
		collection.Description = *req.Description
	}

	// Validate using domain service
	if err := s.domainService.ValidateCollection(collection); err != nil {
		return nil, err
	}

	// Check name uniqueness if name changed
	if req.Name != nil {
		existingNames, err := s.collectionRepo.GetAllNames()
		if err != nil {
			return nil, err
		}
		// Filter out current collection's name
		var filteredNames []string
		for _, name := range existingNames {
			if name != collection.Name {
				filteredNames = append(filteredNames, name)
			}
		}
		if err := s.domainService.ValidateCollectionNameUnique(*req.Name, filteredNames); err != nil {
			return nil, err
		}
	}

	// Update via repository
	if err := s.collectionRepo.Update(collection); err != nil {
		logger.Error("Failed to update collection: %v", err)
		return nil, err
	}

	// Get datasource count
	count, _ := s.collectionRepo.CountDatasources(id)

	return toCollectionResponse(collection, int(count)), nil
}

// Delete deletes a collection
func (s *CollectionServiceImpl) Delete(id string) error {
	// Check if collection exists
	_, err := s.collectionRepo.GetByID(id)
	if err != nil {
		return err
	}

	// Check if collection has datasources
	count, err := s.collectionRepo.CountDatasources(id)
	if err != nil {
		return err
	}

	// Validate deletion using domain service
	if err := s.domainService.CanDeleteCollection(int(count)); err != nil {
		return err
	}

	// Delete the collection
	if err := s.collectionRepo.Delete(id); err != nil {
		logger.Error("Failed to delete collection: %v", err)
		return err
	}
	return nil
}

// GetByID retrieves a collection by ID
func (s *CollectionServiceImpl) GetByID(id string) (*dto.CollectionResponse, error) {
	collection, err := s.collectionRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	count, _ := s.collectionRepo.CountDatasources(id)

	return toCollectionResponse(collection, int(count)), nil
}

// List retrieves collections with pagination
func (s *CollectionServiceImpl) List(params *dto.ListParams) (*dto.CollectionListResponse, error) {
	params.SetDefaults()

	collections, total, err := s.collectionRepo.List(params.Offset(), params.PageSize, params.Search)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.CollectionResponse, len(collections))
	for i, col := range collections {
		count, _ := s.collectionRepo.CountDatasources(col.ID)
		responses[i] = *toCollectionResponse(&col, int(count))
	}

	return &dto.CollectionListResponse{
		Collections: responses,
		Total:       total,
	}, nil
}

