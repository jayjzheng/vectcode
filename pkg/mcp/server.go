package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/jayzheng/vectcode/pkg/config"
	"github.com/jayzheng/vectcode/pkg/embedder"
	"github.com/jayzheng/vectcode/pkg/query"
	"github.com/jayzheng/vectcode/pkg/vectorstore"
)

// Server implements an MCP server for VectCode
type Server struct {
	config      *config.Config
	embedder    embedder.Embedder
	vectorStore vectorstore.VectorStore
	queryEngine *query.Engine
}

// NewServer creates a new MCP server
func NewServer(configPath string) (*Server, error) {
	// Load config
	cfg, err := config.LoadOrDefault(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize embedder
	emb, err := embedder.New(cfg.Embeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// Initialize vector store
	store, err := vectorstore.New(cfg.ToVectorStoreConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	// Create query engine
	engine := query.NewEngine(emb, store)

	return &Server{
		config:      cfg,
		embedder:    emb,
		vectorStore: store,
		queryEngine: engine,
	}, nil
}

// Close closes the server resources
func (s *Server) Close() error {
	if s.vectorStore != nil {
		return s.vectorStore.Close()
	}
	return nil
}

// Run starts the MCP server and handles requests
func (s *Server) Run(input io.Reader, output io.Writer) error {
	for {
		req, err := ReadRequest(input)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			// Write error response and continue
			resp := NewErrorResponse(nil, -32700, fmt.Sprintf("Parse error: %v", err))
			WriteResponse(output, resp)
			continue
		}

		resp := s.handleRequest(req)
		// Only write response if there is one (notifications return nil)
		if resp != nil {
			if err := WriteResponse(output, resp); err != nil {
				return fmt.Errorf("failed to write response: %w", err)
			}
		}
	}
}

// handleRequest processes a JSON-RPC request
func (s *Server) handleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	// Check if this is a notification (no response needed)
	if req.ID == nil {
		// Notifications don't get responses, just handle them silently
		switch req.Method {
		case "notifications/initialized":
			// Client initialized, nothing to do
		case "notifications/cancelled":
			// Request cancelled, nothing to do
		}
		return nil
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return NewErrorResponse(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
	}
}

// InitializeResult contains server information
type InitializeResult struct {
	ProtocolVersion string      `json:"protocolVersion"`
	ServerInfo      ServerInfo  `json:"serverInfo"`
	Capabilities    interface{} `json:"capabilities"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (s *Server) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    "vectcode",
			Version: "0.1.0",
		},
		Capabilities: map[string]interface{}{
			"tools": map[string]interface{}{},
		},
	}
	return NewSuccessResponse(req.ID, result)
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

func (s *Server) handleToolsList(req *JSONRPCRequest) *JSONRPCResponse {
	tools := []Tool{
		{
			Name:        "search_code",
			Description: "Search indexed codebases using semantic search. Returns relevant code chunks with file paths, line numbers, and code content.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Natural language search query (e.g., 'function that fetches user data', 'API endpoint handlers')",
					},
					"project": map[string]interface{}{
						"type":        "string",
						"description": "Optional: filter results to a specific project name",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results to return (default: 5)",
						"default":     5,
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "list_projects",
			Description: "List all indexed projects available for search.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	return NewSuccessResponse(req.ID, map[string]interface{}{
		"tools": tools,
	})
}

// ToolCallParams represents parameters for a tool call
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

func (s *Server) handleToolsCall(req *JSONRPCRequest) *JSONRPCResponse {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, -32602, fmt.Sprintf("Invalid params: %v", err))
	}

	switch params.Name {
	case "search_code":
		return s.handleSearchCode(req.ID, params.Arguments)
	case "list_projects":
		return s.handleListProjects(req.ID)
	default:
		return NewErrorResponse(req.ID, -32601, fmt.Sprintf("Tool not found: %s", params.Name))
	}
}

func (s *Server) handleSearchCode(id interface{}, args map[string]interface{}) *JSONRPCResponse {
	// Extract query parameter
	queryText, ok := args["query"].(string)
	if !ok || queryText == "" {
		return NewErrorResponse(id, -32602, "Missing required parameter: query")
	}

	// Extract optional parameters
	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	var filters map[string]interface{}
	if project, ok := args["project"].(string); ok && project != "" {
		filters = map[string]interface{}{
			"project": project,
		}
	}

	// Execute search
	ctx := context.Background()
	results, err := s.queryEngine.Query(ctx, queryText, limit, filters)
	if err != nil {
		return NewErrorResponse(id, -32603, fmt.Sprintf("Search failed: %v", err))
	}

	// Format results
	formattedResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		chunk := result.Chunk
		formattedResults[i] = map[string]interface{}{
			"score":      result.Score,
			"project":    chunk.Project,
			"file":       chunk.FilePath,
			"line_start": chunk.LineStart,
			"line_end":   chunk.LineEnd,
			"type":       chunk.ChunkType,
			"name":       chunk.Name,
			"code":       chunk.Code,
			"doc_string": chunk.DocString,
		}
	}

	return NewSuccessResponse(id, map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": formatSearchResults(results),
			},
		},
	})
}

func (s *Server) handleListProjects(id interface{}) *JSONRPCResponse {
	ctx := context.Background()
	projects, err := s.vectorStore.ListProjects(ctx)
	if err != nil {
		return NewErrorResponse(id, -32603, fmt.Sprintf("Failed to list projects: %v", err))
	}

	var text string
	if len(projects) == 0 {
		text = "No projects indexed yet."
	} else {
		text = fmt.Sprintf("Indexed projects (%d):\n", len(projects))
		for i, project := range projects {
			text += fmt.Sprintf("%d. %s\n", i+1, project)
		}
	}

	return NewSuccessResponse(id, map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": text,
			},
		},
	})
}

func formatSearchResults(results []vectorstore.SearchResult) string {
	if len(results) == 0 {
		return "No results found."
	}

	output := fmt.Sprintf("Found %d results:\n\n", len(results))
	for i, result := range results {
		chunk := result.Chunk
		output += fmt.Sprintf("=== Result %d (Score: %.4f) ===\n", i+1, result.Score)
		output += fmt.Sprintf("Project: %s\n", chunk.Project)
		output += fmt.Sprintf("File: %s:%d-%d\n", chunk.FilePath, chunk.LineStart, chunk.LineEnd)
		output += fmt.Sprintf("Type: %s %s\n", chunk.ChunkType, chunk.Name)
		if chunk.DocString != "" {
			output += fmt.Sprintf("Documentation:\n%s\n", chunk.DocString)
		}
		output += fmt.Sprintf("\nCode:\n```%s\n%s\n```\n\n", chunk.Language, chunk.Code)
	}
	return output
}
