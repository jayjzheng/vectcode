package embedder

import "context"

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
