package metadata

import (
	"context"
	"time"
)

// Group represents a logical grouping of projects
type Group struct {
	ID          int64
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Project represents an indexed code project
type Project struct {
	ID             int64
	Name           string
	Path           string
	Language       string
	Description    string
	GroupID        *int64     // NULL if not in a group
	GroupName      string     // Populated when joining with groups
	ChunkCount     int
	LastIndexedAt  *time.Time // NULL if never indexed
	LastModifiedAt *time.Time // NULL if unknown
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// File represents a source file in a project
type File struct {
	ID             int64
	ProjectID      int64
	FilePath       string // Relative to project root
	LastModifiedAt *time.Time
	LastIndexedAt  *time.Time
	ChunkCount     int
	FileHash       string // SHA256 hash
}

// ProjectFilter for querying projects
type ProjectFilter struct {
	GroupID   *int64
	GroupName string
	Name      string
}

// Store is the interface for metadata storage
type Store interface {
	// Close closes the metadata store
	Close() error

	// Groups
	CreateGroup(ctx context.Context, name, description string) (*Group, error)
	GetGroup(ctx context.Context, name string) (*Group, error)
	ListGroups(ctx context.Context) ([]Group, error)
	UpdateGroup(ctx context.Context, name, description string) error
	DeleteGroup(ctx context.Context, name string) error

	// Projects
	CreateProject(ctx context.Context, project *Project) error
	GetProject(ctx context.Context, name string) (*Project, error)
	ListProjects(ctx context.Context, filter *ProjectFilter) ([]Project, error)
	UpdateProject(ctx context.Context, project *Project) error
	DeleteProject(ctx context.Context, name string) error

	// Files
	UpsertFile(ctx context.Context, file *File) error
	GetFile(ctx context.Context, projectID int64, filePath string) (*File, error)
	ListFiles(ctx context.Context, projectID int64) ([]File, error)
	DeleteFile(ctx context.Context, projectID int64, filePath string) error
	DeleteProjectFiles(ctx context.Context, projectID int64) error

	// Helpers
	GetProjectsByGroup(ctx context.Context, groupName string) ([]Project, error)
	GetStaleFiles(ctx context.Context, projectID int64) ([]File, error) // Files where last_modified_at > last_indexed_at
}
