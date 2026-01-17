package domain

import "errors"

// Domain errors
var (
	ErrFolderNotFound      = errors.New("folder not found")
	ErrFolderNameExists    = errors.New("folder with this name already exists in parent")
	ErrFolderNameEmpty     = errors.New("folder name cannot be empty")
	ErrInvalidParentFolder = errors.New("invalid parent folder")
	ErrFolderHasChildren   = errors.New("folder has children and cannot be deleted")
	ErrFolderHasProjects   = errors.New("folder has projects and cannot be deleted")
	ErrCircularReference   = errors.New("circular folder reference detected")

	ErrProjectNotFound   = errors.New("project not found")
	ErrProjectNameExists = errors.New("project with this name already exists in folder")
	ErrProjectNameEmpty  = errors.New("project name cannot be empty")
	ErrProjectHasModels  = errors.New("project has models and cannot be deleted")
	ErrProjectHasBuilds  = errors.New("project has builds and cannot be deleted")
)
