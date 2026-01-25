# Embedder Package

This package provides embeddings generation for code chunks.

## Supported Providers

### Ollama (Recommended)

Local, free embeddings using Ollama with the BGE-M3 model.

**Setup:**
```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull the BGE-M3 model
ollama pull bge-m3

# Verify it's running
curl http://localhost:11434/api/embed \
  -d '{"model": "bge-m3", "input": "test"}'
```

**Configuration:**
```yaml
embeddings:
  provider: ollama
  model: bge-m3
  endpoint: http://localhost:11434  # optional, defaults to localhost:11434
```

**Usage:**
```go
config := embedder.Config{
    Provider: "ollama",
    Model:    "bge-m3",
    Endpoint: "http://localhost:11434",
}

emb, err := embedder.New(config)
if err != nil {
    log.Fatal(err)
}

// Single embedding
embedding, err := emb.Embed(ctx, "func main() { ... }")

// Batch embeddings
embeddings, err := emb.EmbedBatch(ctx, []string{"text1", "text2"})

// Get dimensions
dims := emb.Dimensions() // 1024 for BGE-M3
```

**Supported Ollama Models:**
- `bge-m3` - 1024 dimensions (recommended for code)
- `mxbai-embed-large` - 1024 dimensions
- `nomic-embed-text` - 768 dimensions

### OpenAI

Cloud-based embeddings using OpenAI's API (requires API key and costs money).

**Configuration:**
```yaml
embeddings:
  provider: openai
  model: text-embedding-3-small
  api_key_env: OPENAI_API_KEY
```

**Setup:**
```bash
export OPENAI_API_KEY=sk-...
```

**Usage:**
```go
config := embedder.Config{
    Provider:  "openai",
    Model:     "text-embedding-3-small",
    APIKeyEnv: "OPENAI_API_KEY",
}

emb, err := embedder.New(config)
```

## Comparison

| Feature | Ollama (BGE-M3) | OpenAI |
|---------|-----------------|--------|
| Cost | Free | ~$0.02/1M tokens |
| Privacy | Local (private) | Cloud (sent to OpenAI) |
| Speed | Fast (local) | Network dependent |
| Dimensions | 1024 | 1536 (small) / 3072 (large) |
| Context | 8192 tokens | 8191 tokens |
| Setup | Requires local install | API key only |
| Multilingual | 100+ languages | Good multilingual support |
