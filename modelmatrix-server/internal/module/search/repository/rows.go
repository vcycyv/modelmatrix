package repository

import "time"

// BuildRow is a row from the cross-entity build search query.
type BuildRow struct {
	ID                string
	Name              string
	Description       string
	Status            string
	ModelType         string
	Algorithm         string
	FolderID          *string
	ProjectID         *string
	FolderName        *string
	ProjectName       *string
	ProjectFolderName *string
	CreatedBy         string
	CreatedAt         time.Time
}

// ModelRow is a row from the model search query.
type ModelRow struct {
	ID                string
	Name              string
	Description       string
	Status            string
	ModelType         string
	Algorithm         string
	FolderID          *string
	ProjectID         *string
	FolderName        *string
	ProjectName       *string
	ProjectFolderName *string
	CreatedBy         string
	CreatedAt         time.Time
}

// ProjectRow is a row from the project search query.
type ProjectRow struct {
	ID         string
	Name       string
	Description string
	FolderID   *string
	FolderName *string
	CreatedBy  string
	CreatedAt  time.Time
}

// FolderRow is a row from the folder search query.
type FolderRow struct {
	ID          string
	Name        string
	Description string
	ParentID    *string
	ParentName  *string
	CreatedBy   string
	CreatedAt   time.Time
}
