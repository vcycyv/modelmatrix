package domain

import "time"

// Folder represents a folder in the hierarchical structure
type Folder struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ParentID    *string   `json:"parent_id,omitempty"`
	Path        string    `json:"path"`
	Depth       int       `json:"depth"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Project represents a project within a folder
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	FolderID    *string   `json:"folder_id,omitempty"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FolderContentsCount contains counts of items in a folder and its descendants
type FolderContentsCount struct {
	SubfolderCount int64 `json:"subfolder_count"`
	ProjectCount   int64 `json:"project_count"`
	ModelCount     int64 `json:"model_count"`
	BuildCount     int64 `json:"build_count"`
}
