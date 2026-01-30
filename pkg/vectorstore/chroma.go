package vectorstore

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/jayzheng/vectcode/pkg/chunker"
)

// ChromaStore implements VectorStore for ChromaDB
type ChromaStore struct {
	config     Config
	client     chroma.Client
	collection chroma.Collection
}

// NewChromaStore creates a new ChromaDB vector store
func NewChromaStore(config Config) (*ChromaStore, error) {
	// Parse endpoint URL
	endpoint := parseEndpoint(config)

	// Create ChromaDB client
	client, err := chroma.NewHTTPClient(chroma.WithBaseURL(endpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create ChromaDB client: %w\n\nMake sure ChromaDB is running:\n  docker run -p 8000:8000 chromadb/chroma", err)
	}

	// Get collection name
	collectionName := config.Collection
	if collectionName == "" {
		collectionName = "vectcode"
	}

	// Get or create collection with cosine similarity
	// Set HNSW space to cosine in metadata
	metadata := chroma.NewMetadata(
		chroma.NewStringAttribute("hnsw:space", "cosine"),
	)

	collection, err := client.GetOrCreateCollection(
		context.Background(),
		collectionName,
		chroma.WithCollectionMetadataCreate(metadata),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create collection '%s': %w", collectionName, err)
	}

	return &ChromaStore{
		config:     config,
		client:     client,
		collection: collection,
	}, nil
}

// Insert inserts a single code chunk with its embedding
func (c *ChromaStore) Insert(ctx context.Context, chunk chunker.CodeChunk, embedding []float64) error {
	metadata := chunkToMetadata(chunk)
	emb := embeddings.NewEmbeddingFromFloat64(embedding)

	err := c.collection.Upsert(
		ctx,
		chroma.WithIDs(chroma.DocumentID(chunk.ID)),
		chroma.WithTexts(chunk.Code),
		chroma.WithMetadatas(metadata),
		chroma.WithEmbeddings(emb),
	)
	if err != nil {
		return fmt.Errorf("failed to insert chunk %s: %w", chunk.ID, err)
	}

	return nil
}

// InsertBatch inserts multiple code chunks with their embeddings in batches
func (c *ChromaStore) InsertBatch(ctx context.Context, chunks []chunker.CodeChunk, embs [][]float64) error {
	if len(chunks) != len(embs) {
		return fmt.Errorf("chunks and embeddings length mismatch: %d vs %d", len(chunks), len(embs))
	}

	if len(chunks) == 0 {
		return nil
	}

	// Process in batches of 1000 to avoid memory issues
	batchSize := 1000
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batchChunks := chunks[i:end]
		batchEmbeddings := embs[i:end]

		// Prepare batch data
		ids := make([]chroma.DocumentID, len(batchChunks))
		documents := make([]string, len(batchChunks))
		metadatas := make([]chroma.DocumentMetadata, len(batchChunks))
		embeddingsList := make([]embeddings.Embedding, len(batchChunks))

		for j, chunk := range batchChunks {
			ids[j] = chroma.DocumentID(chunk.ID)
			documents[j] = chunk.Code
			metadatas[j] = chunkToMetadata(chunk)
			embeddingsList[j] = embeddings.NewEmbeddingFromFloat64(batchEmbeddings[j])
		}

		// Insert batch (using Upsert to support re-indexing)
		err := c.collection.Upsert(
			ctx,
			chroma.WithIDs(ids...),
			chroma.WithTexts(documents...),
			chroma.WithMetadatas(metadatas...),
			chroma.WithEmbeddings(embeddingsList...),
		)
		if err != nil {
			return fmt.Errorf("failed to insert batch [%d:%d]: %w", i, end, err)
		}
	}

	return nil
}

// Search performs semantic search with optional filters
func (c *ChromaStore) Search(ctx context.Context, queryEmbedding []float64, limit int, filters map[string]interface{}) ([]SearchResult, error) {
	// Build query options
	queryEmb := embeddings.NewEmbeddingFromFloat64(queryEmbedding)
	opts := []chroma.QueryOption{
		chroma.WithQueryEmbeddings(queryEmb),
		chroma.WithNResults(limit),
		chroma.WithIncludeQuery(chroma.IncludeMetadatas, chroma.IncludeDocuments, chroma.IncludeDistances),
	}

	// Add where clause if filters provided
	if len(filters) > 0 {
		whereClause := buildWhereClause(filters)
		if whereClause != nil {
			opts = append(opts, chroma.WithWhereQuery(whereClause))
		}
	}

	// Query the collection
	queryResults, err := c.collection.Query(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	// Convert results to SearchResult format
	results := make([]SearchResult, 0)

	// ChromaDB returns groups (one group per query embedding)
	idGroups := queryResults.GetIDGroups()
	if len(idGroups) == 0 {
		return results, nil
	}

	// We only have one query embedding, so get the first group
	ids := idGroups[0]
	documents := queryResults.GetDocumentsGroups()[0]
	metadatas := queryResults.GetMetadatasGroups()[0]
	distances := queryResults.GetDistancesGroups()[0]

	for i := 0; i < len(ids); i++ {
		// Reconstruct chunk from metadata
		chunk := metadataToChunk(metadatas[i])
		chunk.ID = string(ids[i])
		chunk.Code = documents[i].ContentString()

		// Get distance (convert from float32 to float64)
		distance := float64(distances[i])

		// Calculate score from distance (cosine similarity: score = 1 - distance)
		score := 1.0 - distance

		results = append(results, SearchResult{
			Chunk:    chunk,
			Score:    score,
			Distance: distance,
		})
	}

	return results, nil
}

// Delete deletes all chunks for a project
func (c *ChromaStore) Delete(ctx context.Context, projectName string) error {
	whereClause := chroma.EqString(chroma.K("project"), projectName)

	err := c.collection.Delete(
		ctx,
		chroma.WithWhereDelete(whereClause),
	)
	if err != nil {
		return fmt.Errorf("failed to delete project '%s': %w", projectName, err)
	}

	return nil
}

// ListProjects returns a list of all indexed projects
func (c *ChromaStore) ListProjects(ctx context.Context) ([]string, error) {
	// Get all documents (metadata only)
	results, err := c.collection.Get(
		ctx,
		chroma.WithIncludeGet(chroma.IncludeMetadatas),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}

	// Extract unique project names
	projectSet := make(map[string]bool)
	metadatas := results.GetMetadatas()
	for _, metadata := range metadatas {
		if project, ok := metadata.GetString("project"); ok && project != "" {
			projectSet[project] = true
		}
	}

	// Convert to sorted slice
	projects := make([]string, 0, len(projectSet))
	for project := range projectSet {
		projects = append(projects, project)
	}
	sort.Strings(projects)

	return projects, nil
}

// GetChunk retrieves a single chunk by ID
func (c *ChromaStore) GetChunk(ctx context.Context, id string) (*chunker.CodeChunk, error) {
	results, err := c.collection.Get(
		ctx,
		chroma.WithIDsGet(chroma.DocumentID(id)),
		chroma.WithIncludeGet(chroma.IncludeMetadatas, chroma.IncludeDocuments),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk %s: %w", id, err)
	}

	if results.Count() == 0 {
		return nil, fmt.Errorf("chunk not found: %s", id)
	}

	// Convert first result to CodeChunk
	ids := results.GetIDs()
	documents := results.GetDocuments()
	metadatas := results.GetMetadatas()

	chunk := metadataToChunk(metadatas[0])
	chunk.ID = string(ids[0])
	chunk.Code = documents[0].ContentString()

	return &chunk, nil
}

// Close closes the ChromaDB connection
func (c *ChromaStore) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Helper functions

// parseEndpoint extracts ChromaDB server URL from config
func parseEndpoint(config Config) string {
	// Check options first
	if endpoint, ok := config.Options["endpoint"]; ok && endpoint != "" {
		return endpoint
	}

	// Check if path is a URL
	if strings.HasPrefix(config.Path, "http://") || strings.HasPrefix(config.Path, "https://") {
		return config.Path
	}

	// Default to localhost
	return "http://localhost:8000"
}

// buildWhereClause converts filter map to ChromaDB Where clause
func buildWhereClause(filters map[string]interface{}) chroma.WhereFilter {
	if len(filters) == 0 {
		return nil
	}

	// Build where clauses for each filter
	var clauses []chroma.WhereClause
	for key, value := range filters {
		// Map filter keys to metadata field names
		switch key {
		case "project", "language", "chunk_type", "package", "file_path":
			if strVal, ok := value.(string); ok {
				clauses = append(clauses, chroma.EqString(chroma.K(key), strVal))
			}
		case "projects": // Multiple projects (OR)
			if projects, ok := value.([]string); ok && len(projects) > 0 {
				if len(projects) == 1 {
					clauses = append(clauses, chroma.EqString(chroma.K("project"), projects[0]))
				} else {
					// Build OR clause for multiple projects
					var projectClauses []chroma.WhereClause
					for _, proj := range projects {
						projectClauses = append(projectClauses, chroma.EqString(chroma.K("project"), proj))
					}
					clauses = append(clauses, chroma.Or(projectClauses...))
				}
			}
		}
	}

	// If multiple clauses, combine with AND
	if len(clauses) == 0 {
		return nil
	}
	if len(clauses) == 1 {
		return clauses[0]
	}

	return chroma.And(clauses...)
}

// chunkToMetadata converts CodeChunk to ChromaDB metadata
func chunkToMetadata(chunk chunker.CodeChunk) chroma.DocumentMetadata {
	metadata := chroma.NewDocumentMetadata(
		chroma.NewStringAttribute("project", chunk.Project),
		chroma.NewStringAttribute("file_path", chunk.FilePath),
		chroma.NewStringAttribute("package", chunk.Package),
		chroma.NewStringAttribute("language", chunk.Language),
		chroma.NewStringAttribute("chunk_type", string(chunk.ChunkType)),
		chroma.NewStringAttribute("name", chunk.Name),
		chroma.NewStringAttribute("line_start", fmt.Sprintf("%d", chunk.LineStart)),
		chroma.NewStringAttribute("line_end", fmt.Sprintf("%d", chunk.LineEnd)),
	)

	// Add optional string fields
	if chunk.Receiver != "" {
		metadata.SetString("receiver", chunk.Receiver)
	}
	if chunk.DocString != "" {
		metadata.SetString("doc_string", chunk.DocString)
	}
	if chunk.Comments != "" {
		metadata.SetString("comments", chunk.Comments)
	}

	// Serialize array fields to JSON
	if len(chunk.HTTPEndpoints) > 0 {
		if data, err := json.Marshal(chunk.HTTPEndpoints); err == nil {
			metadata.SetString("http_endpoints", string(data))
		}
	}
	if len(chunk.HTTPCalls) > 0 {
		if data, err := json.Marshal(chunk.HTTPCalls); err == nil {
			metadata.SetString("http_calls", string(data))
		}
	}
	if len(chunk.GRPCMethods) > 0 {
		if data, err := json.Marshal(chunk.GRPCMethods); err == nil {
			metadata.SetString("grpc_methods", string(data))
		}
	}
	if len(chunk.Imports) > 0 {
		if data, err := json.Marshal(chunk.Imports); err == nil {
			metadata.SetString("imports", string(data))
		}
	}

	// Format time as RFC3339
	if !chunk.LastModified.IsZero() {
		metadata.SetString("last_modified", chunk.LastModified.Format(time.RFC3339))
	}

	return metadata
}

// metadataToChunk reconstructs CodeChunk from ChromaDB metadata
func metadataToChunk(metadata chroma.DocumentMetadata) chunker.CodeChunk {
	chunk := chunker.CodeChunk{
		Project:   getStringMeta(metadata, "project"),
		FilePath:  getStringMeta(metadata, "file_path"),
		Package:   getStringMeta(metadata, "package"),
		Language:  getStringMeta(metadata, "language"),
		ChunkType: chunker.ChunkType(getStringMeta(metadata, "chunk_type")),
		Name:      getStringMeta(metadata, "name"),
		Receiver:  getStringMeta(metadata, "receiver"),
		DocString: getStringMeta(metadata, "doc_string"),
		Comments:  getStringMeta(metadata, "comments"),
		LineStart: getIntMeta(metadata, "line_start"),
		LineEnd:   getIntMeta(metadata, "line_end"),
	}

	// Deserialize array fields from JSON
	if httpEndpointsStr := getStringMeta(metadata, "http_endpoints"); httpEndpointsStr != "" {
		var endpoints []string
		if err := json.Unmarshal([]byte(httpEndpointsStr), &endpoints); err == nil {
			chunk.HTTPEndpoints = endpoints
		}
	}
	if httpCallsStr := getStringMeta(metadata, "http_calls"); httpCallsStr != "" {
		var calls []string
		if err := json.Unmarshal([]byte(httpCallsStr), &calls); err == nil {
			chunk.HTTPCalls = calls
		}
	}
	if grpcMethodsStr := getStringMeta(metadata, "grpc_methods"); grpcMethodsStr != "" {
		var methods []string
		if err := json.Unmarshal([]byte(grpcMethodsStr), &methods); err == nil {
			chunk.GRPCMethods = methods
		}
	}
	if importsStr := getStringMeta(metadata, "imports"); importsStr != "" {
		var imports []string
		if err := json.Unmarshal([]byte(importsStr), &imports); err == nil {
			chunk.Imports = imports
		}
	}

	// Parse timestamp
	if lastModStr := getStringMeta(metadata, "last_modified"); lastModStr != "" {
		if t, err := time.Parse(time.RFC3339, lastModStr); err == nil {
			chunk.LastModified = t
		}
	}

	return chunk
}

// getStringMeta extracts a string value from metadata
func getStringMeta(metadata chroma.DocumentMetadata, key string) string {
	if val, ok := metadata.GetString(key); ok {
		return val
	}
	return ""
}

// getIntMeta extracts an int value from metadata (stored as string)
func getIntMeta(metadata chroma.DocumentMetadata, key string) int {
	// ChromaDB stores integers as strings, so we need to parse them
	if val, ok := metadata.GetString(key); ok && val != "" {
		var result int
		fmt.Sscanf(val, "%d", &result)
		return result
	}
	return 0
}
