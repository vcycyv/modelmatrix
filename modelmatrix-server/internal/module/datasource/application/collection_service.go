package application

import (
	"modelmatrix-server/internal/infrastructure/fileservice"
	"modelmatrix-server/internal/module/datasource/domain"
	"modelmatrix-server/internal/module/datasource/dto"
	"modelmatrix-server/internal/module/datasource/repository"
	"modelmatrix-server/pkg/logger"
)

// CollectionServiceImpl implements CollectionService
type CollectionServiceImpl struct {
	collectionRepo repository.CollectionRepository
	datasourceRepo repository.DatasourceRepository
	domainService  *domain.Service
	fileService    fileservice.FileService
}

// NewCollectionService creates a new collection service
func NewCollectionService(
	collectionRepo repository.CollectionRepository,
	datasourceRepo repository.DatasourceRepository,
	domainService *domain.Service,
	fileService fileservice.FileService,
) CollectionService {
	return &CollectionServiceImpl{
		collectionRepo: collectionRepo,
		datasourceRepo: datasourceRepo,
		domainService:  domainService,
		fileService:    fileService,
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

	// Check name uniqueness before applying the update (use original name to exclude self)
	if req.Name != nil {
		originalName := collection.Name
		existingNames, err := s.collectionRepo.GetAllNames()
		if err != nil {
			return nil, err
		}
		var filteredNames []string
		for _, name := range existingNames {
			if name != originalName {
				filteredNames = append(filteredNames, name)
			}
		}
		if err := s.domainService.ValidateCollectionNameUnique(*req.Name, filteredNames); err != nil {
			return nil, err
		}
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

	// Update via repository
	if err := s.collectionRepo.Update(collection); err != nil {
		logger.Error("Failed to update collection: %v", err)
		return nil, err
	}

	// Get datasource count
	count, _ := s.collectionRepo.CountDatasources(id)

	return toCollectionResponse(collection, int(count)), nil
}

// Delete deletes a collection. If force is true, also deletes all datasources in the collection.
func (s *CollectionServiceImpl) Delete(id string, force bool) error {
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

	// If not force delete, validate that collection is empty
	if !force {
		if err := s.domainService.CanDeleteCollection(int(count)); err != nil {
			return err
		}
		// Delete the collection (no datasources)
		if err := s.collectionRepo.Delete(id); err != nil {
			logger.Error("Failed to delete collection: %v", err)
			return err
		}
	} else {
		// Force delete: first delete files from MinIO, then delete from database
		
		// Get all datasources to delete their files
		datasources, _, err := s.datasourceRepo.ListByCollection(id, 0, 10000) // Large limit to get all
		if err != nil {
			logger.Error("Failed to get datasources for file deletion: %v", err)
			return err
		}

		// Delete files from MinIO storage
		for _, ds := range datasources {
			if ds.FilePath != "" {
				if err := s.fileService.Delete(ds.FilePath); err != nil {
					logger.Warn("Failed to delete datasource file from storage: %s, error: %v", ds.FilePath, err)
					// Continue - don't fail the deletion for file cleanup errors
				} else {
					logger.Info("Deleted datasource file from storage: %s", ds.FilePath)
				}
			}
		}

		// Delete collection and all its datasources from database
		if err := s.collectionRepo.DeleteWithDatasources(id); err != nil {
			logger.Error("Failed to force delete collection: %v", err)
			return err
		}
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

