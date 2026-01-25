package embedder

import (
	"context"
	"fmt"
)

// Embedder defines the interface for generating embeddings
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)
	Dimensions() int
}

// Config holds embedder configuration
type Config struct {
	Provider  string `yaml:"provider"`
	Model     string `yaml:"model"`
	APIKeyEnv string `yaml:"api_key_env"`
	Endpoint  string `yaml:"endpoint"`
}

// New creates an embedder based on the provider in the config
func New(config Config) (Embedder, error) {
	switch config.Provider {
	case "ollama":
		return NewOllamaEmbedder(config)
	case "openai":
		return NewOpenAIEmbedder(config)
	default:
		return nil, fmt.Errorf("unsupported embedder provider: %s", config.Provider)
	}
}
