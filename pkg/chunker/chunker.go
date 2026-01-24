package chunker

import "time"

// ChunkType represents the type of code chunk
type ChunkType string

const (
	ChunkTypeFunction  ChunkType = "function"
	ChunkTypeMethod    ChunkType = "method"
	ChunkTypeStruct    ChunkType = "struct"
	ChunkTypeInterface ChunkType = "interface"
	ChunkTypePackage   ChunkType = "package"
	ChunkTypeFile      ChunkType = "file"
)

// CodeChunk represents a parsed piece of code with metadata
type CodeChunk struct {
	// Identification
	ID       string    `json:"id"`
	Project  string    `json:"project"`
	FilePath string    `json:"file_path"`
	Package  string    `json:"package"`
	Language string    `json:"language"` // "go", "typescript", etc.
	
	// Content
	Code      string    `json:"code"`
	ChunkType ChunkType `json:"chunk_type"`
	Name      string    `json:"name"` // function/struct/interface name
	
	// For methods
	Receiver string `json:"receiver,omitempty"` // receiver type for methods
	
	// Service interaction metadata
	HTTPEndpoints []string `json:"http_endpoints,omitempty"` // e.g., "POST /api/users"
	HTTPCalls     []string `json:"http_calls,omitempty"`     // outbound HTTP calls
	GRPCMethods   []string `json:"grpc_methods,omitempty"`   // gRPC service methods
	Imports       []string `json:"imports,omitempty"`        // imported packages
	
	// Documentation
	DocString string `json:"doc_string,omitempty"` // godoc comment
	Comments  string `json:"comments,omitempty"`   // inline comments
	
	// Metadata
	LineStart    int       `json:"line_start"`
	LineEnd      int       `json:"line_end"`
	LastModified time.Time `json:"last_modified"`
}

// ToText converts the chunk to a text representation for embedding
func (c *CodeChunk) ToText() string {
	text := ""
	
	if c.DocString != "" {
		text += c.DocString + "\n\n"
	}
	
	text += "Project: " + c.Project + "\n"
	text += "Package: " + c.Package + "\n"
	text += "Type: " + string(c.ChunkType) + "\n"
	
	if c.Name != "" {
		text += "Name: " + c.Name + "\n"
	}
	
	if len(c.HTTPEndpoints) > 0 {
		text += "HTTP Endpoints: " + joinStrings(c.HTTPEndpoints) + "\n"
	}
	
	if len(c.Imports) > 0 {
		text += "Imports: " + joinStrings(c.Imports) + "\n"
	}
	
	text += "\nCode:\n" + c.Code
	
	return text
}

func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}
