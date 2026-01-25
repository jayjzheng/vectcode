package vectorstore

import (
	"context"
	"fmt"

	"github.com/yourusername/codegraph/pkg/chunker"
)

// SearchResult represents a search result from the vector store
type SearchResult struct {
	Chunk    chunker.CodeChunk `json:"chunk"`
	Score    float64            `json:"score"`
	Distance float64            `json:"distance"`
}

// VectorStore defines the interface for vector storage backends
type VectorStore interface {
	Insert(ctx context.Context, chunk chunker.CodeChunk, embedding []float64) error
	InsertBatch(ctx context.Context, chunks []chunker.CodeChunk, embeddings [][]float64) error
	Search(ctx context.Context, queryEmbedding []float64, limit int, filters map[string]interface{}) ([]SearchResult, error)
	Delete(ctx context.Context, projectName string) error
	ListProjects(ctx context.Context) ([]string, error)
	GetChunk(ctx context.Context, id string) (*chunker.CodeChunk, error)
	Close() error
}

// Config holds vector store configuration
type Config struct {
	Type       string            `yaml:"type"`
	Path       string            `yaml:"path"`
	Collection string            `yaml:"collection"`
	Options    map[string]string `yaml:"options"`
}

// New creates a vector store based on the type in the config
func New(config Config) (VectorStore, error) {
	switch config.Type {
	case "chroma":
		return NewChromaStore(config)
	default:
		return nil, fmt.Errorf("unsupported vector store type: %s", config.Type)
	}
}
