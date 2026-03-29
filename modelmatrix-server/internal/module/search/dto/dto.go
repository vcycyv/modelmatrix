package dto

import "time"

// Resource type constants for filter and results
const (
	TypeBuild   = "build"
	TypeModel   = "model"
	TypeProject = "project"
	TypeFolder  = "folder"
)

// SearchResultItem is one entry in the search response
type SearchResultItem struct {
	Type        string    `json:"type"` // build | model | project | folder
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status,omitempty"`
	ModelType   string    `json:"model_type,omitempty"`
	Algorithm   string    `json:"algorithm,omitempty"`
	FolderID    *string   `json:"folder_id,omitempty"`
	ProjectID   *string   `json:"project_id,omitempty"`
	Breadcrumb  string    `json:"breadcrumb"` // human-readable path, e.g. "Prod / Fraud"
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// SearchResponse is the API response for GET /api/search
type SearchResponse struct {
	Query   string             `json:"query"`
	Total   int                `json:"total"`
	Results []SearchResultItem `json:"results"`
}
