package repository

import (
	"gorm.io/gorm"
)

// SearchRepository runs cross-entity search queries.
type SearchRepository interface {
	GetFolderPath(folderID string) (string, error)
	SearchBuilds(like, folderID, descendantPattern string, limit int) ([]BuildRow, error)
	SearchModels(like, folderID, descendantPattern string, limit int) ([]ModelRow, error)
	SearchProjects(like, folderID, descendantPattern string, limit int) ([]ProjectRow, error)
	SearchFolders(like, folderID, descendantPattern string, limit int) ([]FolderRow, error)
}

// GormSearchRepository implements SearchRepository using raw SQL via GORM.
type GormSearchRepository struct {
	db *gorm.DB
}

// NewGormSearchRepository creates a repository backed by the given DB handle.
func NewGormSearchRepository(db *gorm.DB) *GormSearchRepository {
	return &GormSearchRepository{db: db}
}

// GetFolderPath returns the materialized path for a folder, or ErrFolderNotFound.
func (r *GormSearchRepository) GetFolderPath(folderID string) (string, error) {
	var path string
	if err := r.db.Raw("SELECT path FROM folders WHERE id = ?", folderID).Scan(&path).Error; err != nil {
		return "", err
	}
	if path == "" {
		return "", ErrFolderNotFound
	}
	return path, nil
}

// SearchBuilds searches model_builds with optional folder/project scoping.
func (r *GormSearchRepository) SearchBuilds(like, folderID, descendantPattern string, limit int) ([]BuildRow, error) {
	sql := `
		SELECT b.id, b.name, b.description, b.status, b.model_type, b.algorithm,
		       b.folder_id, b.project_id,
		       f.name  AS folder_name,
		       p.name  AS project_name,
		       pf.name AS project_folder_name,
		       b.created_by, b.created_at
		FROM model_builds b
		LEFT JOIN folders  f  ON b.folder_id  = f.id
		LEFT JOIN projects p  ON b.project_id = p.id
		LEFT JOIN folders  pf ON p.folder_id  = pf.id
		WHERE (LOWER(b.name) LIKE LOWER(?) OR LOWER(b.description) LIKE LOWER(?))`
	args := []interface{}{like, like}
	if clause, extra := folderScopeSQL("b", folderID, descendantPattern); clause != "" {
		sql += " " + clause
		args = append(args, extra...)
	}
	sql += `
		ORDER BY b.created_at DESC
		LIMIT ?`
	args = append(args, limit)

	var rows []BuildRow
	if err := r.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// SearchModels searches models with optional folder/project scoping.
func (r *GormSearchRepository) SearchModels(like, folderID, descendantPattern string, limit int) ([]ModelRow, error) {
	sql := `
		SELECT m.id, m.name, m.description, m.status, m.model_type, m.algorithm,
		       m.folder_id, m.project_id,
		       f.name  AS folder_name,
		       p.name  AS project_name,
		       pf.name AS project_folder_name,
		       m.created_by, m.created_at
		FROM models m
		LEFT JOIN folders  f  ON m.folder_id  = f.id
		LEFT JOIN projects p  ON m.project_id = p.id
		LEFT JOIN folders  pf ON p.folder_id  = pf.id
		WHERE (LOWER(m.name) LIKE LOWER(?) OR LOWER(m.description) LIKE LOWER(?))`
	args := []interface{}{like, like}
	if clause, extra := folderScopeSQL("m", folderID, descendantPattern); clause != "" {
		sql += " " + clause
		args = append(args, extra...)
	}
	sql += `
		ORDER BY m.created_at DESC
		LIMIT ?`
	args = append(args, limit)

	var rows []ModelRow
	if err := r.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// SearchProjects searches projects with optional folder scoping.
func (r *GormSearchRepository) SearchProjects(like, folderID, descendantPattern string, limit int) ([]ProjectRow, error) {
	sql := `
		SELECT p.id, p.name, p.description, p.folder_id,
		       f.name AS folder_name,
		       p.created_by, p.created_at
		FROM projects p
		LEFT JOIN folders f ON p.folder_id = f.id
		WHERE (LOWER(p.name) LIKE LOWER(?) OR LOWER(p.description) LIKE LOWER(?))`
	args := []interface{}{like, like}
	if folderID != "" {
		sql += `
		AND (
			p.folder_id = ?
			OR p.folder_id IN (SELECT id FROM folders WHERE path LIKE ?)
		)`
		args = append(args, folderID, descendantPattern)
	}
	sql += `
		ORDER BY p.created_at DESC
		LIMIT ?`
	args = append(args, limit)

	var rows []ProjectRow
	if err := r.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// SearchFolders searches folders with optional descendant scoping under a folder path.
func (r *GormSearchRepository) SearchFolders(like, folderID, descendantPattern string, limit int) ([]FolderRow, error) {
	sql := `
		SELECT f.id, f.name, f.description, f.parent_id,
		       pf.name AS parent_name,
		       f.created_by, f.created_at
		FROM folders f
		LEFT JOIN folders pf ON f.parent_id = pf.id
		WHERE (LOWER(f.name) LIKE LOWER(?) OR LOWER(f.description) LIKE LOWER(?))`
	args := []interface{}{like, like}
	if folderID != "" {
		sql += ` AND f.path LIKE ?`
		args = append(args, descendantPattern)
	}
	sql += `
		ORDER BY f.depth ASC, f.name ASC
		LIMIT ?`
	args = append(args, limit)

	var rows []FolderRow
	if err := r.db.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// folderScopeSQL returns an AND clause and bound args for build/model search when folderID is set.
// tableAlias must be a safe identifier ("b" or "m"); values use placeholders.
func folderScopeSQL(tableAlias, folderID, descendantPattern string) (string, []interface{}) {
	if folderID == "" {
		return "", nil
	}
	clause := `AND (
		` + tableAlias + `.folder_id = ?
		OR ` + tableAlias + `.folder_id IN (SELECT id FROM folders WHERE path LIKE ?)
		OR ` + tableAlias + `.project_id IN (
			SELECT id FROM projects WHERE
				folder_id = ?
				OR folder_id IN (SELECT id FROM folders WHERE path LIKE ?)
		)
	)`
	args := []interface{}{folderID, descendantPattern, folderID, descendantPattern}
	return clause, args
}
