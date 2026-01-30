# VectCode

A semantic code search tool that indexes code repositories into a vector database, enabling natural language search across your codebase.

## Features

- **Multi-repository indexing**: Index multiple Go projects into a unified knowledge base
- **Semantic search**: Query your codebase using natural language via vector embeddings
- **ChromaDB integration**: Fast vector storage and retrieval
- **MCP Server**: Use VectCode with Claude Desktop and other LLM clients via Model Context Protocol
- **Ollama embeddings**: Free, local embeddings with BGE-M3 model (or OpenAI alternative)

## Installation

### Prerequisites

1. **ChromaDB** - Vector database server:
   ```bash
   docker run -d -p 8000:8000 chromadb/chroma
   ```

2. **Ollama** (recommended) - Local embedding model:
   ```bash
   # Install Ollama
   curl -fsSL https://ollama.com/install.sh | sh

   # Pull the BGE-M3 embedding model
   ollama pull bge-m3
   ```

### Build from Source

```bash
# Build CLI tool
go build -o vectcode ./cmd/vectcode

# Build MCP server (optional, for Claude Desktop integration)
go build -o vectcode-mcp-server ./cmd/mcp-server
```

## Quick Start

### 1. Setup Configuration

```bash
mkdir -p ~/.vectcode
cp config.example.yaml ~/.vectcode/config.yaml
```

Edit `~/.vectcode/config.yaml` to configure ChromaDB and Ollama endpoints.

### 2. Index a Project

```bash
./vectcode index --path ~/projects/my-service --name my-service
```

**Re-indexing with clean slate:**
```bash
# Use --clean to delete existing data first (removes orphaned chunks from deleted code)
./vectcode index --path ~/projects/my-service --name my-service --clean
```

### 3. Query the Codebase

```bash
./vectcode query --query "where is the user authentication handler?" --limit 5
```

### 4. List Indexed Projects

```bash
./vectcode list
```

### 5. Delete a Project

```bash
./vectcode delete --name my-service
```

## MCP Server (Claude Desktop Integration)

VectCode can be used as an MCP (Model Context Protocol) server, allowing Claude Desktop and other LLM clients to search your indexed codebases during conversations.

See [MCP_SETUP.md](MCP_SETUP.md) for detailed setup instructions.

**Quick setup for Claude Desktop (macOS)**:

1. Build the MCP server:
   ```bash
   go build -o vectcode-mcp-server ./cmd/mcp-server
   sudo cp vectcode-mcp-server /usr/local/bin/
   ```

2. Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:
   ```json
   {
     "mcpServers": {
       "vectcode": {
         "command": "/usr/local/bin/vectcode-mcp-server"
       }
     }
   }
   ```

3. Restart Claude Desktop and start searching your code!

### CLI Options

All commands support a `--config` flag to specify a custom config file:

```bash
./vectcode --config /path/to/config.yaml index --path . --name myproject
```

## Configuration

VectCode uses a configuration file at `~/.vectcode/config.yaml`:

```yaml
vector_store:
  type: chroma
  collection: vectcode
  options:
    endpoint: http://localhost:8000

embeddings:
  # Option 1: Ollama (local, free, recommended)
  provider: ollama
  model: bge-m3
  endpoint: http://localhost:11434

  # Option 2: OpenAI (requires API key)
  # provider: openai
  # model: text-embedding-3-small
  # api_key_env: OPENAI_API_KEY
```

## Architecture

```
vectcode/
├── cmd/
│   ├── vectcode/      # CLI entry point
│   └── mcp-server/     # MCP server for LLM integration
├── pkg/
│   ├── parser/         # Code parsing (AST analysis)
│   ├── chunker/        # Code chunking logic
│   ├── embedder/       # Generate embeddings (Ollama/OpenAI)
│   ├── vectorstore/    # Vector store interface and ChromaDB implementation
│   ├── indexer/        # Orchestrates parsing and storing
│   ├── query/          # Query engine for semantic search
│   ├── config/         # Configuration management
│   └── mcp/            # MCP protocol and server implementation
```

## How It Works

1. **Parsing**: VectCode parses Go source files using AST analysis to extract:
   - Functions and methods
   - Struct and interface definitions
   - Constants and global variables
   - Documentation strings

2. **Chunking**: Each code element (function, struct, etc.) is extracted as a separate chunk with:
   - Code content
   - File path and line numbers
   - Documentation
   - Type information

3. **Embedding**: Code chunks are converted to vector embeddings using:
   - Ollama with BGE-M3 model (local, free)
   - Or OpenAI's text-embedding models

4. **Storage**: Embeddings and metadata are stored in ChromaDB for fast similarity search

5. **Querying**: Natural language queries are embedded and matched against stored code chunks using cosine similarity

## Supported Languages

Currently supported:
- **Go**: Full AST-based parsing with function, method, struct, and interface extraction

## Use Cases

- **Code Discovery**: Find relevant code examples across multiple repositories
- **Onboarding**: Help new team members understand codebase structure
- **Refactoring**: Find all usages and similar patterns
- **Documentation**: Locate functions and their documentation
- **LLM Integration**: Use with Claude Desktop for AI-powered code assistance

## Re-indexing Behavior

When you re-index a project, VectCode uses deterministic IDs (based on `project:file:name`) to handle updates:

**Without `--clean` flag:**
- Existing code chunks are **updated** (upsert behavior)
- New code chunks are **added**
- ⚠️ **Orphaned chunks remain**: If you delete code from your project, those chunks stay in the database

**With `--clean` flag:**
- All existing project data is **deleted first**
- Then indexes from scratch
- ✅ **No orphaned chunks**: Ensures database exactly matches current code state

**When to use `--clean`:**
- After deleting or renaming files/functions
- When you want to ensure a fresh, accurate index
- Troubleshooting stale search results

## Roadmap

- [x] Project scaffolding
- [x] Go parser implementation (AST-based)
- [x] Ollama embedder integration (BGE-M3)
- [x] ChromaDB vector store integration
- [x] Basic CLI commands (index, query, list, delete)
- [x] MCP server for Claude Desktop integration
- [ ] Support for additional languages (TypeScript, Python, Rust)
- [ ] Incremental indexing (detect and index only changed files)
- [ ] Multi-language project support
- [ ] Enhanced metadata filtering

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT
