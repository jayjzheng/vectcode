package embedder

import (
	"context"
	"fmt"
	"os"
)

// OpenAIEmbedder implements Embedder using OpenAI's API
type OpenAIEmbedder struct {
	config Config
	apiKey string
}

func NewOpenAIEmbedder(config Config) (*OpenAIEmbedder, error) {
	apiKey := os.Getenv(config.APIKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("API key not found in environment variable %s", config.APIKeyEnv)
	}
	
	return &OpenAIEmbedder{
		config: config,
		apiKey: apiKey,
	}, nil
}

func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (e *OpenAIEmbedder) Dimensions() int {
	if e.config.Model == "text-embedding-3-large" {
		return 3072
	}
	return 1536
}
