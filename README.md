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

### Index a project
```bash
codegraph index --path ~/projects/my-service --name my-service
```

### Query the codebase
```bash
codegraph query "where is the user authentication handler?"
```

### List indexed projects
```bash
codegraph list
```

### Delete a project
```bash
codegraph delete --name my-service
```

## Configuration

CodeGraph uses a configuration file at `~/.codegraph/config.yaml`:

```yaml
vector_store:
  type: chroma
  path: ~/.codegraph/db
  
embeddings:
  provider: openai
  model: text-embedding-3-small
  api_key_env: OPENAI_API_KEY

llm:
  provider: anthropic
  model: claude-sonnet-4-5-20250929
  api_key_env: ANTHROPIC_API_KEY
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
- [ ] Go parser implementation
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
