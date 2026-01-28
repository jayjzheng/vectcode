package metadata

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store using SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite metadata store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Run migrations
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// CreateGroup creates a new group
func (s *SQLiteStore) CreateGroup(ctx context.Context, name, description string) (*Group, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO groups (name, description) VALUES (?, ?)",
		name, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get group id: %w", err)
	}

	return s.getGroupByID(ctx, id)
}

// GetGroup retrieves a group by name
func (s *SQLiteStore) GetGroup(ctx context.Context, name string) (*Group, error) {
	var group Group
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, description, created_at, updated_at FROM groups WHERE name = ?",
		name).Scan(&group.ID, &group.Name, &group.Description, &group.CreatedAt, &group.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("group not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	return &group, nil
}

func (s *SQLiteStore) getGroupByID(ctx context.Context, id int64) (*Group, error) {
	var group Group
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, description, created_at, updated_at FROM groups WHERE id = ?",
		id).Scan(&group.ID, &group.Name, &group.Description, &group.CreatedAt, &group.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	return &group, nil
}

// ListGroups retrieves all groups
func (s *SQLiteStore) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, description, created_at, updated_at FROM groups ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var group Group
		if err := rows.Scan(&group.ID, &group.Name, &group.Description, &group.CreatedAt, &group.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

// UpdateGroup updates a group's description
func (s *SQLiteStore) UpdateGroup(ctx context.Context, name, description string) error {
	result, err := s.db.ExecContext(ctx,
		"UPDATE groups SET description = ?, updated_at = CURRENT_TIMESTAMP WHERE name = ?",
		description, name)
	if err != nil {
		return fmt.Errorf("failed to update group: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("group not found: %s", name)
	}

	return nil
}

// DeleteGroup deletes a group (sets projects' group_id to NULL)
func (s *SQLiteStore) DeleteGroup(ctx context.Context, name string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM groups WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("group not found: %s", name)
	}

	return nil
}

// CreateProject creates a new project
func (s *SQLiteStore) CreateProject(ctx context.Context, project *Project) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO projects (name, path, language, description, group_id, chunk_count, last_indexed_at, last_modified_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		project.Name, project.Path, project.Language, project.Description,
		project.GroupID, project.ChunkCount, project.LastIndexedAt, project.LastModifiedAt)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get project id: %w", err)
	}

	project.ID = id
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()

	return nil
}

// GetProject retrieves a project by name
func (s *SQLiteStore) GetProject(ctx context.Context, name string) (*Project, error) {
	var project Project
	var groupID sql.NullInt64
	var groupName sql.NullString
	var lastIndexedAt, lastModifiedAt sql.NullTime

	err := s.db.QueryRowContext(ctx,
		`SELECT p.id, p.name, p.path, p.language, p.description, p.group_id, g.name,
		        p.chunk_count, p.last_indexed_at, p.last_modified_at, p.created_at, p.updated_at
		 FROM projects p
		 LEFT JOIN groups g ON p.group_id = g.id
		 WHERE p.name = ?`,
		name).Scan(&project.ID, &project.Name, &project.Path, &project.Language, &project.Description,
		&groupID, &groupName, &project.ChunkCount, &lastIndexedAt, &lastModifiedAt,
		&project.CreatedAt, &project.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if groupID.Valid {
		project.GroupID = &groupID.Int64
		project.GroupName = groupName.String
	}
	if lastIndexedAt.Valid {
		project.LastIndexedAt = &lastIndexedAt.Time
	}
	if lastModifiedAt.Valid {
		project.LastModifiedAt = &lastModifiedAt.Time
	}

	return &project, nil
}

// ListProjects retrieves all projects with optional filtering
func (s *SQLiteStore) ListProjects(ctx context.Context, filter *ProjectFilter) ([]Project, error) {
	query := `SELECT p.id, p.name, p.path, p.language, p.description, p.group_id, g.name,
	                 p.chunk_count, p.last_indexed_at, p.last_modified_at, p.created_at, p.updated_at
	          FROM projects p
	          LEFT JOIN groups g ON p.group_id = g.id
	          WHERE 1=1`
	args := []interface{}{}

	if filter != nil {
		if filter.GroupID != nil {
			query += " AND p.group_id = ?"
			args = append(args, *filter.GroupID)
		}
		if filter.GroupName != "" {
			query += " AND g.name = ?"
			args = append(args, filter.GroupName)
		}
		if filter.Name != "" {
			query += " AND p.name = ?"
			args = append(args, filter.Name)
		}
	}

	query += " ORDER BY p.name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var project Project
		var groupID sql.NullInt64
		var groupName sql.NullString
		var lastIndexedAt, lastModifiedAt sql.NullTime

		if err := rows.Scan(&project.ID, &project.Name, &project.Path, &project.Language,
			&project.Description, &groupID, &groupName, &project.ChunkCount,
			&lastIndexedAt, &lastModifiedAt, &project.CreatedAt, &project.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}

		if groupID.Valid {
			project.GroupID = &groupID.Int64
			project.GroupName = groupName.String
		}
		if lastIndexedAt.Valid {
			project.LastIndexedAt = &lastIndexedAt.Time
		}
		if lastModifiedAt.Valid {
			project.LastModifiedAt = &lastModifiedAt.Time
		}

		projects = append(projects, project)
	}

	return projects, rows.Err()
}

// UpdateProject updates a project
func (s *SQLiteStore) UpdateProject(ctx context.Context, project *Project) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE projects
		 SET path = ?, language = ?, description = ?, group_id = ?,
		     chunk_count = ?, last_indexed_at = ?, last_modified_at = ?,
		     updated_at = CURRENT_TIMESTAMP
		 WHERE name = ?`,
		project.Path, project.Language, project.Description, project.GroupID,
		project.ChunkCount, project.LastIndexedAt, project.LastModifiedAt,
		project.Name)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("project not found: %s", project.Name)
	}

	return nil
}

// DeleteProject deletes a project and all its files
func (s *SQLiteStore) DeleteProject(ctx context.Context, name string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM projects WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("project not found: %s", name)
	}

	return nil
}

// UpsertFile inserts or updates a file
func (s *SQLiteStore) UpsertFile(ctx context.Context, file *File) error {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO files (project_id, file_path, last_modified_at, last_indexed_at, chunk_count, file_hash)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(project_id, file_path) DO UPDATE SET
		     last_modified_at = excluded.last_modified_at,
		     last_indexed_at = excluded.last_indexed_at,
		     chunk_count = excluded.chunk_count,
		     file_hash = excluded.file_hash`,
		file.ProjectID, file.FilePath, file.LastModifiedAt, file.LastIndexedAt,
		file.ChunkCount, file.FileHash)
	if err != nil {
		return fmt.Errorf("failed to upsert file: %w", err)
	}

	if file.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil {
			file.ID = id
		}
	}

	return nil
}

// GetFile retrieves a file by project ID and file path
func (s *SQLiteStore) GetFile(ctx context.Context, projectID int64, filePath string) (*File, error) {
	var file File
	var lastModifiedAt, lastIndexedAt sql.NullTime

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, file_path, last_modified_at, last_indexed_at, chunk_count, file_hash
		 FROM files WHERE project_id = ? AND file_path = ?`,
		projectID, filePath).Scan(&file.ID, &file.ProjectID, &file.FilePath,
		&lastModifiedAt, &lastIndexedAt, &file.ChunkCount, &file.FileHash)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	if lastModifiedAt.Valid {
		file.LastModifiedAt = &lastModifiedAt.Time
	}
	if lastIndexedAt.Valid {
		file.LastIndexedAt = &lastIndexedAt.Time
	}

	return &file, nil
}

// ListFiles retrieves all files for a project
func (s *SQLiteStore) ListFiles(ctx context.Context, projectID int64) ([]File, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, file_path, last_modified_at, last_indexed_at, chunk_count, file_hash
		 FROM files WHERE project_id = ? ORDER BY file_path`,
		projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var file File
		var lastModifiedAt, lastIndexedAt sql.NullTime

		if err := rows.Scan(&file.ID, &file.ProjectID, &file.FilePath,
			&lastModifiedAt, &lastIndexedAt, &file.ChunkCount, &file.FileHash); err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}

		if lastModifiedAt.Valid {
			file.LastModifiedAt = &lastModifiedAt.Time
		}
		if lastIndexedAt.Valid {
			file.LastIndexedAt = &lastIndexedAt.Time
		}

		files = append(files, file)
	}

	return files, rows.Err()
}

// DeleteFile deletes a specific file
func (s *SQLiteStore) DeleteFile(ctx context.Context, projectID int64, filePath string) error {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM files WHERE project_id = ? AND file_path = ?",
		projectID, filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("file not found: %s", filePath)
	}

	return nil
}

// DeleteProjectFiles deletes all files for a project
func (s *SQLiteStore) DeleteProjectFiles(ctx context.Context, projectID int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM files WHERE project_id = ?", projectID)
	if err != nil {
		return fmt.Errorf("failed to delete project files: %w", err)
	}
	return nil
}

// GetProjectsByGroup retrieves all projects in a group
func (s *SQLiteStore) GetProjectsByGroup(ctx context.Context, groupName string) ([]Project, error) {
	return s.ListProjects(ctx, &ProjectFilter{GroupName: groupName})
}

// GetStaleFiles retrieves files that need re-indexing (modified after last index)
func (s *SQLiteStore) GetStaleFiles(ctx context.Context, projectID int64) ([]File, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, file_path, last_modified_at, last_indexed_at, chunk_count, file_hash
		 FROM files
		 WHERE project_id = ?
		   AND (last_indexed_at IS NULL OR last_modified_at > last_indexed_at)
		 ORDER BY file_path`,
		projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get stale files: %w", err)
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var file File
		var lastModifiedAt, lastIndexedAt sql.NullTime

		if err := rows.Scan(&file.ID, &file.ProjectID, &file.FilePath,
			&lastModifiedAt, &lastIndexedAt, &file.ChunkCount, &file.FileHash); err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}

		if lastModifiedAt.Valid {
			file.LastModifiedAt = &lastModifiedAt.Time
		}
		if lastIndexedAt.Valid {
			file.LastIndexedAt = &lastIndexedAt.Time
		}

		files = append(files, file)
	}

	return files, rows.Err()
}
