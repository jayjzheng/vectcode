# Testing CodeGraph

This guide will help you test the CodeGraph CLI with the Ollama embedder integration.

## Prerequisites

### 1. Install Ollama

```bash
# macOS/Linux
curl -fsSL https://ollama.com/install.sh | sh

# Verify installation
ollama --version
```

### 2. Pull the BGE-M3 Model

```bash
ollama pull bge-m3
```

### 3. Start Ollama Server

```bash
# Ollama usually starts automatically, but if not:
ollama serve
```

Verify it's running:
```bash
curl http://localhost:11434/api/tags
```

## Build CodeGraph

```bash
# From the project root
go build -o codegraph ./cmd/codegraph

# Verify build
./codegraph --version
```

## Configuration

The CLI will use `~/.codegraph/config.yaml` by default. Create it:

```bash
mkdir -p ~/.codegraph
cp config.example.yaml ~/.codegraph/config.yaml
```

Default configuration uses Ollama with BGE-M3:
```yaml
embeddings:
  provider: ollama
  model: bge-m3
  endpoint: http://localhost:11434
```

## Testing the CLI

### Test 1: Index a Sample Project

Index the codegraph project itself:

```bash
./codegraph index \
  --path . \
  --name codegraph
```

**Expected Output:**
```
Indexing project: codegraph from path: .
Initializing embedder...
Initializing vector store...
Initializing parser...
Parsing project: codegraph
Found X code chunks
Generating embeddings...
Storing in vector database...
Successfully indexed project: codegraph
```

**Note:** This will currently fail at the vector store step because ChromaDB is not implemented yet. Expected error:
```
Error: failed to create vector store: not implemented yet
```

### Test 2: Test Ollama Embedder Directly

Run the example test program:

```bash
go run examples/test_ollama_embedder.go
```

**Expected Output:**
```
Using embedder with 1024 dimensions

Generating embedding for code:
func main() {
    fmt.Println("Hello, World!")
}

✓ Generated embedding: 1024 dimensions
  First 10 values: [0.123 -0.456 0.789 ...]

Generating batch embeddings for 3 code snippets...
✓ Generated 3 embeddings
  [0] 1024 dimensions
  [1] 1024 dimensions
  [2] 1024 dimensions

✓ All tests passed!
```

### Test 3: Verify CLI Commands

```bash
# Show help
./codegraph --help

# Show index command help
./codegraph index --help

# Show query command help
./codegraph query --help

# List projects (will fail - vector store not implemented)
./codegraph list
```

## Current Limitations

### ✅ Working:
- CLI framework and commands
- Configuration loading
- Go parser (AST-based)
- Ollama embedder integration
- Component initialization

### ❌ Not Yet Implemented:
- ChromaDB vector store integration
- Actual query execution (blocked by vector store)
- LLM integration for query synthesis
- Project listing (blocked by vector store)
- Project deletion (blocked by vector store)

## Next Steps

To make the full pipeline work, you need to implement the ChromaDB vector store. See `pkg/vectorstore/chroma.go`.

### Quick ChromaDB Setup

```bash
# Install ChromaDB (Python)
pip install chromadb

# Or use Docker
docker pull chromadb/chroma
docker run -p 8000:8000 chromadb/chroma

# Then update the vector store implementation to use the Chroma Go client
```

## Troubleshooting

### "Failed to create embedder: connection refused"

Ollama is not running. Start it:
```bash
ollama serve
```

### "Failed to create embedder: model not found"

Pull the BGE-M3 model:
```bash
ollama pull bge-m3
```

### "Failed to load config"

Config file doesn't exist. Create it:
```bash
mkdir -p ~/.codegraph
cp config.example.yaml ~/.codegraph/config.yaml
```

### "Not implemented yet" (vector store)

Expected! ChromaDB integration is not implemented yet. This is the next step.
