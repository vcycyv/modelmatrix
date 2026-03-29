package application

import (
	"errors"
	"fmt"

	"modelmatrix-server/internal/module/search/dto"
	"modelmatrix-server/internal/module/search/repository"
)

// SearchService orchestrates cross-entity search (application layer).
type SearchService struct {
	repo repository.SearchRepository
}

// NewSearchService constructs a SearchService.
func NewSearchService(repo repository.SearchRepository) *SearchService {
	return &SearchService{repo: repo}
}

// Search runs a global or folder-scoped search across resource types.
func (s *SearchService) Search(q, typeFilter, folderID string, limit int) (*dto.SearchResponse, error) {
	if q == "" {
		return nil, errors.New("query is required")
	}

	like := "%" + q + "%"

	var descendantPattern string
	if folderID != "" {
		path, err := s.repo.GetFolderPath(folderID)
		if err != nil {
			if errors.Is(err, repository.ErrFolderNotFound) {
				return nil, repository.ErrFolderNotFound
			}
			return nil, err
		}
		descendantPattern = path + "/%"
	}

	var results []dto.SearchResultItem

	if typeFilter == "all" || typeFilter == dto.TypeBuild {
		builds, err := s.repo.SearchBuilds(like, folderID, descendantPattern, limit)
		if err != nil {
			return nil, fmt.Errorf("build search: %w", err)
		}
		for _, r := range builds {
			results = append(results, dto.SearchResultItem{
				Type:        dto.TypeBuild,
				ID:          r.ID,
				Name:        r.Name,
				Description: r.Description,
				Status:      r.Status,
				ModelType:   r.ModelType,
				Algorithm:   r.Algorithm,
				FolderID:    r.FolderID,
				ProjectID:   r.ProjectID,
				Breadcrumb:  buildBreadcrumb(r.FolderName, r.ProjectName, r.ProjectFolderName),
				CreatedBy:   r.CreatedBy,
				CreatedAt:   r.CreatedAt,
			})
		}
	}

	if typeFilter == "all" || typeFilter == dto.TypeModel {
		models, err := s.repo.SearchModels(like, folderID, descendantPattern, limit)
		if err != nil {
			return nil, fmt.Errorf("model search: %w", err)
		}
		for _, r := range models {
			results = append(results, dto.SearchResultItem{
				Type:        dto.TypeModel,
				ID:          r.ID,
				Name:        r.Name,
				Description: r.Description,
				Status:      r.Status,
				ModelType:   r.ModelType,
				Algorithm:   r.Algorithm,
				FolderID:    r.FolderID,
				ProjectID:   r.ProjectID,
				Breadcrumb:  buildBreadcrumb(r.FolderName, r.ProjectName, r.ProjectFolderName),
				CreatedBy:   r.CreatedBy,
				CreatedAt:   r.CreatedAt,
			})
		}
	}

	if typeFilter == "all" || typeFilter == dto.TypeProject {
		projects, err := s.repo.SearchProjects(like, folderID, descendantPattern, limit)
		if err != nil {
			return nil, fmt.Errorf("project search: %w", err)
		}
		for _, r := range projects {
			breadcrumb := ""
			if r.FolderName != nil {
				breadcrumb = *r.FolderName
			}
			results = append(results, dto.SearchResultItem{
				Type:        dto.TypeProject,
				ID:          r.ID,
				Name:        r.Name,
				Description: r.Description,
				FolderID:    r.FolderID,
				Breadcrumb:  breadcrumb,
				CreatedBy:   r.CreatedBy,
				CreatedAt:   r.CreatedAt,
			})
		}
	}

	if typeFilter == "all" || typeFilter == dto.TypeFolder {
		folders, err := s.repo.SearchFolders(like, folderID, descendantPattern, limit)
		if err != nil {
			return nil, fmt.Errorf("folder search: %w", err)
		}
		for _, r := range folders {
			breadcrumb := ""
			if r.ParentName != nil {
				breadcrumb = *r.ParentName
			}
			results = append(results, dto.SearchResultItem{
				Type:        dto.TypeFolder,
				ID:          r.ID,
				Name:        r.Name,
				Description: r.Description,
				FolderID:    r.ParentID,
				Breadcrumb:  breadcrumb,
				CreatedBy:   r.CreatedBy,
				CreatedAt:   r.CreatedAt,
			})
		}
	}

	return &dto.SearchResponse{
		Query:   q,
		Total:   len(results),
		Results: results,
	}, nil
}

// buildBreadcrumb builds a human-readable location string for builds/models.
func buildBreadcrumb(folderName, projectName, projectFolderName *string) string {
	if folderName != nil && *folderName != "" {
		return *folderName
	}
	if projectName != nil && *projectName != "" {
		if projectFolderName != nil && *projectFolderName != "" {
			return *projectFolderName + " / " + *projectName
		}
		return *projectName
	}
	return ""
}
