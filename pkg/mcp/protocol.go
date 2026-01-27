package mcp

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ReadRequest reads a JSON-RPC request from an io.Reader
func ReadRequest(r io.Reader) (*JSONRPCRequest, error) {
	decoder := json.NewDecoder(r)
	var req JSONRPCRequest
	if err := decoder.Decode(&req); err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)
	}
	return &req, nil
}

// WriteResponse writes a JSON-RPC response to an io.Writer
func WriteResponse(w io.Writer, resp *JSONRPCResponse) error {
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(resp); err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}
	return nil
}

// NewErrorResponse creates an error response
func NewErrorResponse(id interface{}, code int, message string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
}

// NewSuccessResponse creates a success response
func NewSuccessResponse(id interface{}, result interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}
