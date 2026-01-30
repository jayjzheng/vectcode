package indexer

import (
	"context"
	"fmt"
	
	"github.com/jayzheng/vectcode/pkg/chunker"
	"github.com/jayzheng/vectcode/pkg/embedder"
	"github.com/jayzheng/vectcode/pkg/parser"
	"github.com/jayzheng/vectcode/pkg/vectorstore"
)

// Indexer orchestrates the indexing process
type Indexer struct {
	parser      parser.Parser
	embedder    embedder.Embedder
	vectorStore vectorstore.VectorStore
}

func New(p parser.Parser, e embedder.Embedder, vs vectorstore.VectorStore) *Indexer {
	return &Indexer{
		parser:      p,
		embedder:    e,
		vectorStore: vs,
	}
}

func (i *Indexer) IndexProject(ctx context.Context, projectPath string, projectName string) (int, error) {
	fmt.Printf("Parsing project: %s\n", projectName)

	chunks, err := i.parser.Parse(ctx, projectPath, projectName)
	if err != nil {
		return 0, fmt.Errorf("failed to parse project: %w", err)
	}

	if len(chunks) == 0 {
		return 0, fmt.Errorf("no code chunks found in project")
	}

	fmt.Printf("Found %d code chunks\n", len(chunks))
	fmt.Printf("Generating embeddings...\n")

	embeddings, err := i.generateEmbeddings(ctx, chunks)
	if err != nil {
		return 0, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	fmt.Printf("Storing in vector database...\n")
	err = i.vectorStore.InsertBatch(ctx, chunks, embeddings)
	if err != nil {
		return 0, fmt.Errorf("failed to store chunks: %w", err)
	}

	fmt.Printf("Successfully indexed project: %s\n", projectName)
	return len(chunks), nil
}

func (i *Indexer) DeleteProject(ctx context.Context, projectName string) error {
	return i.vectorStore.Delete(ctx, projectName)
}

func (i *Indexer) ListProjects(ctx context.Context) ([]string, error) {
	return i.vectorStore.ListProjects(ctx)
}

func (i *Indexer) generateEmbeddings(ctx context.Context, chunks []chunker.CodeChunk) ([][]float64, error) {
	texts := make([]string, len(chunks))
	for idx, chunk := range chunks {
		texts[idx] = chunk.ToText()
	}
	
	return i.embedder.EmbedBatch(ctx, texts)
}
