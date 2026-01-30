# Getting Started with VectCode

## Prerequisites

- Go 1.21 or higher
- One of the following for embeddings:
  - OpenAI API key (for OpenAI embeddings)
  - Anthropic API key (for Claude embeddings)
- One of the following for LLM queries:
  - Anthropic API key (recommended)
  - OpenAI API key

## Installation

### 1. Clone and build

```bash
cd ~/projects/vectcode
go mod tidy
go build -o vectcode ./cmd/vectcode
```

Or use the Makefile:

```bash
make build
```

### 2. Set up configuration

Copy the example config:

```bash
mkdir -p ~/.vectcode
cp config.example.yaml ~/.vectcode/config.yaml
```

Edit `~/.vectcode/config.yaml` and set your preferences.

### 3. Set environment variables

```bash
export OPENAI_API_KEY="your-key-here"
export ANTHROPIC_API_KEY="your-key-here"
```

## Next Steps - Implementation

The scaffolding is complete! Here's what needs to be implemented:

### Phase 1: Core Functionality (Week 1-2)

1. **Vector Store Integration**
   - Implement Chroma client in `pkg/vectorstore/chroma.go`
   - Add ChromaDB Go client dependency
   - Implement Insert, Search, Delete operations

2. **Embeddings**
   - Implement OpenAI embedder in `pkg/embedder/openai.go`
   - Use official OpenAI Go SDK
   - Add batch embedding support

3. **Wire it all together**
   - Update `cmd/vectcode/main.go` to actually use the packages
   - Load config from `~/.vectcode/config.yaml`
   - Initialize components properly

### Phase 2: Testing (Week 2-3)

1. **Index a test project**
   ```bash
   ./vectcode index --path ~/projects/test-service --name test-service
   ```

2. **Query the codebase**
   ```bash
   ./vectcode query --query "where is the HTTP handler?"
   ```

3. **Test with multiple projects**
   ```bash
   ./vectcode index --path ~/projects/user-service --name user-service
   ./vectcode index --path ~/projects/property-service --name property-service
   ./vectcode query --query "show me all authentication code"
   ```

### Phase 3: Enhancements (Week 3-4)

1. **LLM Integration**
   - Implement `QueryWithLLM` in `pkg/query/query.go`
   - Use Claude API to synthesize answers from code chunks

2. **Service Flow Tracing**
   - Enhance HTTP endpoint/call detection
   - Build service dependency graph
   - Generate Mermaid diagrams

3. **Additional Features**
   - Add support for TypeScript/JavaScript
   - Add configuration management
   - Improve chunking strategies

## Project Structure

```
vectcode/
├── cmd/vectcode/          # CLI entry point
│   └── main.go
├── pkg/
│   ├── chunker/           # Code chunk data structures
│   │   └── chunker.go
│   ├── parser/            # Language parsers
│   │   ├── parser.go      # Parser interface
│   │   └── go.go          # Go AST parser (implemented)
│   ├── embedder/          # Embedding generation
│   │   ├── embedder.go    # Embedder interface
│   │   └── openai.go      # OpenAI implementation (stub)
│   ├── vectorstore/       # Vector storage
│   │   ├── vectorstore.go # VectorStore interface
│   │   └── chroma.go      # Chroma implementation (stub)
│   ├── indexer/           # Orchestration
│   │   └── indexer.go
│   └── query/             # Query engine
│       └── query.go
├── go.mod
├── README.md
├── Makefile
├── .gitignore
└── config.example.yaml
```

## What's Already Implemented

✅ **Go Parser**: Fully functional AST parser that extracts:
  - Functions and methods
  - Structs and interfaces
  - HTTP endpoints and calls
  - Import statements
  - Documentation

✅ **Project Structure**: Clean, modular architecture
✅ **CLI Framework**: Cobra-based CLI with index, query, list, delete commands
✅ **Interfaces**: Well-defined interfaces for extensibility

## What Needs Implementation

🔲 **Vector Store**: ChromaDB client integration
🔲 **Embeddings**: OpenAI API calls
🔲 **Config Loading**: YAML config file parsing
🔲 **LLM Integration**: Claude API for query synthesis
🔲 **Tests**: Unit and integration tests

## Recommended Dependencies

Add these to `go.mod`:

```go
require (
	github.com/amikos-tech/chroma-go v0.1.0  // Chroma client
	github.com/sashabaranov/go-openai v1.17.9  // OpenAI client
	gopkg.in/yaml.v3 v3.0.1  // Already included
	github.com/spf13/cobra v1.8.0  // Already included
)
```

## Quick Reference

### Index a project
```bash
vectcode index --path ~/projects/my-service --name my-service
```

### Query
```bash
vectcode query --query "authentication handler" --limit 5
vectcode query --query "HTTP endpoints" --project user-service
```

### List projects
```bash
vectcode list
```

### Delete a project
```bash
vectcode delete --name my-service
```

## Tips

1. **Start small**: Index one small project first to validate the approach
2. **Test parsing**: Run the Go parser on your codebase to see what it extracts
3. **Chunking strategy**: Adjust chunking based on your codebase characteristics
4. **Embedding costs**: Be mindful of API costs when indexing large codebases

## Support

For issues or questions, check the README.md or create an issue in the repository.
