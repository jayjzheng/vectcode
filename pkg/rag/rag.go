package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/yourusername/codegraph/pkg/embedder"
	"github.com/yourusername/codegraph/pkg/llm"
	"github.com/yourusername/codegraph/pkg/vectorstore"
)

// Engine orchestrates RAG: retrieval from vector store + generation from LLM
type Engine struct {
	embedder    embedder.Embedder
	vectorStore vectorstore.VectorStore
	llm         llm.Client
}

// New creates a new RAG engine
func New(emb embedder.Embedder, store vectorstore.VectorStore, llmClient llm.Client) *Engine {
	return &Engine{
		embedder:    emb,
		vectorStore: store,
		llm:         llmClient,
	}
}

// Ask answers a question using RAG
func (e *Engine) Ask(ctx context.Context, question string, options AskOptions) (string, error) {
	// Step 1: Retrieve relevant code chunks
	fmt.Println("Searching codebase for relevant context...")

	// Embed the question
	questionEmbedding, err := e.embedder.Embed(ctx, question)
	if err != nil {
		return "", fmt.Errorf("failed to embed question: %w", err)
	}

	// Search vector store
	filters := make(map[string]interface{})
	if options.Project != "" {
		filters["project"] = options.Project
	}

	results, err := e.vectorStore.Search(ctx, questionEmbedding, options.TopK, filters)
	if err != nil {
		return "", fmt.Errorf("failed to search: %w", err)
	}

	if len(results) == 0 {
		return "No relevant code found in the indexed codebase.", nil
	}

	fmt.Printf("Found %d relevant code chunks\n", len(results))

	// Step 2: Build context from retrieved chunks
	context := e.buildContext(results, options.MaxContextChunks)

	// Step 3: Build prompt
	prompt := e.buildPrompt(question, context)

	// Step 4: Send to LLM
	fmt.Println("Generating answer with LLM...")
	messages := []llm.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	answer, err := e.llm.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to get LLM response: %w", err)
	}

	return answer, nil
}

// AskOptions configures the RAG request
type AskOptions struct {
	Project          string // Filter by project
	TopK             int    // Number of chunks to retrieve
	MaxContextChunks int    // Maximum chunks to include in context
}

// DefaultAskOptions returns sensible defaults
func DefaultAskOptions() AskOptions {
	return AskOptions{
		TopK:             10,
		MaxContextChunks: 5,
	}
}

// buildContext creates a formatted context string from search results
func (e *Engine) buildContext(results []vectorstore.SearchResult, maxChunks int) string {
	var sb strings.Builder

	limit := maxChunks
	if len(results) < limit {
		limit = len(results)
	}

	for i := 0; i < limit; i++ {
		result := results[i]
		chunk := result.Chunk

		sb.WriteString(fmt.Sprintf("\n--- Code Chunk %d (Score: %.4f) ---\n", i+1, result.Score))
		sb.WriteString(fmt.Sprintf("File: %s:%d-%d\n", chunk.FilePath, chunk.LineStart, chunk.LineEnd))
		sb.WriteString(fmt.Sprintf("Type: %s %s\n", chunk.ChunkType, chunk.Name))

		if chunk.DocString != "" {
			sb.WriteString(fmt.Sprintf("Documentation:\n%s\n", chunk.DocString))
		}

		sb.WriteString(fmt.Sprintf("\nCode:\n```%s\n%s\n```\n", chunk.Language, chunk.Code))
	}

	return sb.String()
}

// buildPrompt creates the final prompt for the LLM
func (e *Engine) buildPrompt(question, context string) string {
	return fmt.Sprintf(`You are a helpful assistant that answers questions about a codebase. Use the following code snippets as context to answer the user's question.

CONTEXT FROM CODEBASE:
%s

USER QUESTION:
%s

Please provide a clear, concise answer based on the code context above. If the code doesn't contain enough information to fully answer the question, say so. Include relevant code references (file paths and line numbers) in your answer.`, context, question)
}
