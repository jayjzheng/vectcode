package main

import (
	"context"
	"fmt"
	"log"

	"github.com/yourusername/codegraph/pkg/embedder"
)

func main() {
	// Create Ollama embedder config
	config := embedder.Config{
		Provider: "ollama",
		Model:    "bge-m3",
		Endpoint: "http://localhost:11434",
	}

	// Create embedder
	emb, err := embedder.New(config)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	fmt.Printf("Using embedder with %d dimensions\n", emb.Dimensions())

	// Test single embedding
	ctx := context.Background()
	testCode := `func main() {
    fmt.Println("Hello, World!")
}`

	fmt.Printf("\nGenerating embedding for code:\n%s\n\n", testCode)

	embedding, err := emb.Embed(ctx, testCode)
	if err != nil {
		log.Fatalf("Failed to generate embedding: %v", err)
	}

	fmt.Printf("✓ Generated embedding: %d dimensions\n", len(embedding))
	fmt.Printf("  First 10 values: %v\n", embedding[:10])

	// Test batch embeddings
	texts := []string{
		"func Add(a, b int) int { return a + b }",
		"func Multiply(a, b int) int { return a * b }",
		"type User struct { Name string; Age int }",
	}

	fmt.Printf("\n\nGenerating batch embeddings for %d code snippets...\n", len(texts))

	embeddings, err := emb.EmbedBatch(ctx, texts)
	if err != nil {
		log.Fatalf("Failed to generate batch embeddings: %v", err)
	}

	fmt.Printf("✓ Generated %d embeddings\n", len(embeddings))
	for i, emb := range embeddings {
		fmt.Printf("  [%d] %d dimensions\n", i, len(emb))
	}

	fmt.Println("\n✓ All tests passed!")
}
