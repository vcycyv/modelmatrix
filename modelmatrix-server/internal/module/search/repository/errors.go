package repository

import "errors"

// ErrFolderNotFound is returned when folder_id does not match a folder row.
var ErrFolderNotFound = errors.New("folder not found")
