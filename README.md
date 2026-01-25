# CodeGraph

A code knowledge base tool that ingests multiple code repositories and creates a queryable vector store for LLM-powered code understanding.

## Features

- **Multi-repository indexing**: Index multiple Go projects into a unified knowledge base
- **Semantic search**: Query your codebase using natural language
- **Service mapping**: Understand how microservices interact with each other
- **Flow tracing**: Trace request flows across services
- **RAG-powered**: Uses vector embeddings and LLM for intelligent code understanding

## Installation

```bash
go install github.com/yourusername/codegraph/cmd/codegraph@latest
```

## Quick Start

### 1. Build the CLI

```bash
go build -o codegraph ./cmd/codegraph
```

### 2. Setup Configuration

```bash
mkdir -p ~/.codegraph
cp config.example.yaml ~/.codegraph/config.yaml
```

### 3. Index a Project

```bash
./codegraph index --path ~/projects/my-service --name my-service
```

### 4. Query the Codebase

```bash
./codegraph query --query "where is the user authentication handler?" --limit 5
```

### 5. List Indexed Projects

```bash
./codegraph list
```

### 6. Delete a Project

```bash
./codegraph delete --name my-service
```

### CLI Options

All commands support a `--config` flag to specify a custom config file:

```bash
./codegraph --config /path/to/config.yaml index --path . --name myproject
```

## Configuration

CodeGraph uses a configuration file at `~/.codegraph/config.yaml`:

```yaml
vector_store:
  type: chroma
  path: ~/.codegraph/db

embeddings:
  # Option 1: Ollama (local, free, recommended)
  provider: ollama
  model: bge-m3
  endpoint: http://localhost:11434

  # Option 2: OpenAI (requires API key)
  # provider: openai
  # model: text-embedding-3-small
  # api_key_env: OPENAI_API_KEY

llm:
  provider: anthropic
  model: claude-sonnet-4-5-20250929
  api_key_env: ANTHROPIC_API_KEY
```

### Setup Ollama (Recommended)

Ollama provides free, local embeddings with no API costs:

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull the BGE-M3 embedding model
ollama pull bge-m3

# Verify it's working
curl http://localhost:11434/api/embed -d '{"model": "bge-m3", "input": "test"}'
```

## Architecture

```
codegraph/
├── cmd/codegraph/      # CLI entry point
├── pkg/
│   ├── parser/         # Code parsing (AST analysis)
│   ├── chunker/        # Code chunking logic
│   ├── embedder/       # Generate embeddings
│   ├── vectorstore/    # Vector store interface and implementations
│   ├── indexer/        # Orchestrates parsing and storing
│   └── query/          # Query interface and LLM integration
```

## Roadmap

- [x] Project scaffolding
- [x] Go parser implementation (AST-based with HTTP detection)
- [x] Ollama embedder integration (BGE-M3)
- [ ] Vector store integration (Chroma)
- [ ] Basic CLI commands (index, query, list, delete)
- [ ] Service interaction mapping
- [ ] Flow tracing
- [ ] Visualization (Mermaid diagrams)
- [ ] Support for additional languages (TypeScript, Python)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT
