package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OllamaEmbedder implements Embedder using Ollama's local API
type OllamaEmbedder struct {
	config     Config
	httpClient *http.Client
	endpoint   string
	model      string
}

// ollamaEmbedRequest represents the request to Ollama's embed API
type ollamaEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// ollamaEmbedResponse represents the response from Ollama's embed API
type ollamaEmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

func NewOllamaEmbedder(config Config) (*OllamaEmbedder, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	model := config.Model
	if model == "" {
		model = "bge-m3"
	}

	return &OllamaEmbedder{
		config:     config,
		httpClient: &http.Client{},
		endpoint:   endpoint,
		model:      model,
	}, nil
}

func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	reqBody := ollamaEmbedRequest{
		Model: e.model,
		Input: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/embed", e.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var embedResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embedResp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned from Ollama")
	}

	return embedResp.Embeddings[0], nil
}

func (e *OllamaEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))

	for i, text := range texts {
		embedding, err := e.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text at index %d: %w", i, err)
		}
		embeddings[i] = embedding
	}

	return embeddings, nil
}

func (e *OllamaEmbedder) Dimensions() int {
	// BGE-M3 produces 1024-dimensional embeddings
	// This could be made configurable for other models
	switch e.model {
	case "bge-m3":
		return 1024
	case "mxbai-embed-large":
		return 1024
	case "nomic-embed-text":
		return 768
	default:
		// Default to BGE-M3 dimensions
		return 1024
	}
}
