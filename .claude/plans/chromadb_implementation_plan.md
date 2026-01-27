# ChromaDB Vector Store Implementation Plan

## Overview
Implement ChromaDB integration in `pkg/vectorstore/chroma.go` to enable code indexing and semantic search. Use the `github.com/amikos-tech/chroma-go` Go client library.

## Key Design Decisions

1. **ChromaDB Client**: Use `github.com/amikos-tech/chroma-go` (most mature Go client)
2. **Metadata Storage**: Serialize CodeChunk fields to ChromaDB metadata
   - String arrays (HTTPEndpoints, Imports, etc.) → JSON strings
   - time.Time → RFC3339 string format
   - All CodeChunk fields stored in metadata for complete reconstruction
3. **Batch Size**: Process insertions in chunks of 1000 documents
4. **Distance Metric**: Use cosine similarity (standard for semantic search)
5. **Server Connection**: Default to `http://localhost:8000`, support custom URLs via config

## Implementation Steps

### 1. Add Dependency
```bash
go get github.com/amikos-tech/chroma-go
```

### 2. Update ChromaStore Structure
Add client and collection fields to hold ChromaDB connection:
```go
type ChromaStore struct {
    config     Config
    client     *chromago.Client
    collection *chromago.Collection
}
```

### 3. Implement NewChromaStore Constructor
- Parse endpoint URL from config (support both file paths and URLs)
- Create HTTP client connection to ChromaDB server
- Get or create collection with cosine similarity
- Return initialized store or error

### 4. Implement Metadata Serialization Helpers

**chunkToMetadata()**: Convert CodeChunk to map[string]interface{}
- Direct mapping for strings, ints (project, file_path, package, language, chunk_type, name, line_start, line_end, etc.)
- JSON serialize arrays (http_endpoints, http_calls, grpc_methods, imports)
- Format time.Time as RFC3339 string (last_modified)
- Omit empty optional fields

**metadataToChunk()**: Reconstruct CodeChunk from ChromaDB metadata
- Type-safe extraction with helper functions (getStringMeta, getIntMeta)
- JSON deserialize array fields
- Parse RFC3339 timestamp back to time.Time
- Handle missing fields gracefully

### 5. Implement Insert Methods

**Insert()**: Single chunk insertion
- Convert chunk to metadata using chunkToMetadata()
- Call collection.Add() with ID, embedding, metadata, document (code)
- Return error if insertion fails

**InsertBatch()**: Batch insertion (critical for performance)
- Validate chunks and embeddings length match
- Prepare arrays: ids, embeddings, metadatas, documents
- Process in batches of 1000 to avoid memory issues
- Use collection.Add() for each batch
- Return error with batch range if any batch fails

### 6. Implement Search Method

**buildWhereClause()**: Convert filter map to ChromaDB Where clause
- Map our filter keys (project, language, chunk_type) to metadata field names
- Simple equality filters: `{"project": "myproject"}`

**Search()**: Semantic search with optional filters
- Build where clause from filters
- Call collection.Query() with query embedding, limit, where clause
- Convert ChromaDB results to SearchResult format:
  - Extract IDs, documents, metadatas, distances from response
  - Convert each result to CodeChunk using metadataToChunk()
  - Calculate score: `score = 1.0 - distance` (for cosine)
  - Build SearchResult array with Chunk, Score, Distance
- Return results or error

### 7. Implement Delete Method
Delete all chunks for a project using metadata filter:
- Build where clause: `{"project": projectName}`
- Call collection.Delete() with where clause
- Return error if deletion fails

### 8. Implement ListProjects Method
Get unique project names from all indexed chunks:
- Call collection.Get() to retrieve all documents (metadata only)
- Extract "project" field from each metadata
- Build set of unique projects
- Sort and return as string slice

### 9. Implement GetChunk Method
Retrieve single chunk by ID:
- Call collection.Get() with ID array
- Check if result exists
- Convert first result to CodeChunk using metadataToChunk()
- Return chunk or "not found" error

### 10. Implement Close Method
ChromaDB Go client doesn't require explicit cleanup, so return nil.

### 11. Add Helper Functions
- `parseEndpoint(config)`: Extract ChromaDB server URL from config
  - Check options["endpoint"] first
  - Check if path starts with "http://" or "https://"
  - Default to "http://localhost:8000"
- `getStringMeta(metadata, key)`: Type-safe string extraction
- `getIntMeta(metadata, key)`: Type-safe int extraction (handle float64)

### 12. Error Handling
- Connection failures: Clear error message with setup instructions
- Validation: Check embedding dimensions, chunk/embedding length match
- Graceful degradation: Log warnings for malformed metadata, continue processing
- Context support: Respect context cancellation in all operations

## Critical Files

1. **`/Users/jayzheng/projects/codegraph/pkg/vectorstore/chroma.go`** - Main implementation file
2. **`/Users/jayzheng/projects/codegraph/pkg/chunker/chunker.go`** - CodeChunk structure reference
3. **`/Users/jayzheng/projects/codegraph/pkg/vectorstore/vectorstore.go`** - Interface contract
4. **`/Users/jayzheng/projects/codegraph/config.example.yaml`** - Update with ChromaDB setup instructions

## Configuration Updates

Update `config.example.yaml` to document ChromaDB options:
```yaml
vector_store:
  type: chroma
  path: ~/.codegraph/db  # Local persistence (not used for HTTP client)
  collection: codegraph
  options:
    endpoint: http://localhost:8000  # ChromaDB server URL (optional)
```

## Testing Plan

### Prerequisites
Start ChromaDB server:
```bash
docker run -p 8000:8000 chromadb/chroma
```

### Test Cases

1. **Connection Test**: Verify ChromaStore initialization connects successfully
2. **Insert Test**: Index a single code chunk and retrieve it
3. **Batch Insert Test**: Index 100+ chunks from a real Go project
4. **Search Test**: Query indexed code and verify relevant results returned
5. **Filter Test**: Search with project filter, verify only matching project returned
6. **Delete Test**: Delete a project, verify all chunks removed
7. **ListProjects Test**: Index multiple projects, verify all listed
8. **Metadata Round-trip Test**: Verify all CodeChunk fields preserved after insert/retrieve
9. **Array Fields Test**: Verify HTTPEndpoints, Imports correctly serialized/deserialized
10. **Error Test**: Try connecting to non-running ChromaDB, verify clear error message

### End-to-End Verification

```bash
# 1. Start ChromaDB
docker run -p 8000:8000 chromadb/chroma

# 2. Build CLI
go build -o codegraph ./cmd/codegraph

# 3. Index a project
./codegraph index --path . --name codegraph

# Expected output:
# Indexing project: codegraph from path: .
# Initializing embedder...
# Initializing vector store...
# Initializing parser...
# Parsing project: codegraph
# Found X code chunks
# Generating embeddings...
# Storing in vector database...
# Successfully indexed project: codegraph

# 4. Query the codebase
./codegraph query --query "embedder implementation" --limit 5

# Expected output:
# Found 5 results:
# === Result 1 (Score: 0.85) ===
# Project: codegraph
# File: pkg/embedder/ollama.go:27-45
# Type: function Embed
# [code shown]

# 5. List projects
./codegraph list

# Expected output:
# Indexed projects (1):
#   1. codegraph

# 6. Delete project
./codegraph delete --name codegraph

# Expected output:
# Deleting project: codegraph
# ✓ Project 'codegraph' deleted successfully
```

## Potential Issues & Mitigations

1. **ChromaDB not running**: Add clear error with setup instructions
2. **Large metadata**: Validate chunk size, warn if >100KB
3. **Duplicate IDs**: Document that re-indexing same project requires delete first
4. **Memory usage**: Batch processing at 1000 chunks prevents OOM
5. **JSON serialization errors**: Log warnings, use empty arrays as fallback

## Future Enhancements

After basic implementation works:
- Add retry logic for transient network errors
- Support upsert semantics for re-indexing
- Add ChromaDB authentication support
- Implement pagination for large result sets
- Add telemetry/metrics for operations
