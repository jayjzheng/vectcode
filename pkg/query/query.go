package query

import (
	"context"
	"fmt"
	
	"github.com/jayzheng/vectcode/pkg/embedder"
	"github.com/jayzheng/vectcode/pkg/vectorstore"
)

// Engine handles queries against the code knowledge base
type Engine struct {
	embedder    embedder.Embedder
	vectorStore vectorstore.VectorStore
	llmConfig   LLMConfig
}

// LLMConfig holds LLM configuration
type LLMConfig struct {
	Provider  string `yaml:"provider"`
	Model     string `yaml:"model"`
	APIKeyEnv string `yaml:"api_key_env"`
}

func New(e embedder.Embedder, vs vectorstore.VectorStore, llmConfig LLMConfig) *Engine {
	return &Engine{
		embedder:    e,
		vectorStore: vs,
		llmConfig:   llmConfig,
	}
}

// NewEngine creates a query engine without LLM config (for basic queries)
func NewEngine(e embedder.Embedder, vs vectorstore.VectorStore) *Engine {
	return &Engine{
		embedder:    e,
		vectorStore: vs,
	}
}

func (q *Engine) Query(ctx context.Context, queryText string, limit int, filters map[string]interface{}) ([]vectorstore.SearchResult, error) {
	queryEmbedding, err := q.embedder.Embed(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	
	results, err := q.vectorStore.Search(ctx, queryEmbedding, limit, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector store: %w", err)
	}
	
	return results, nil
}

func (q *Engine) QueryWithLLM(ctx context.Context, queryText string, limit int, filters map[string]interface{}) (string, error) {
	results, err := q.Query(ctx, queryText, limit, filters)
	if err != nil {
		return "", err
	}
	
	if len(results) == 0 {
		return "No relevant code found for your query.", nil
	}
	
	response := "Found relevant code:\n\n"
	for i, result := range results {
		response += fmt.Sprintf("--- Result %d (Score: %.2f) ---\n", i+1, result.Score)
		response += fmt.Sprintf("Project: %s\n", result.Chunk.Project)
		response += fmt.Sprintf("File: %s\n", result.Chunk.FilePath)
		response += fmt.Sprintf("Name: %s\n", result.Chunk.Name)
		response += fmt.Sprintf("\n%s\n\n", result.Chunk.Code)
	}
	
	return response, nil
}
