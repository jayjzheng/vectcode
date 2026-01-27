package llm

import (
	"context"
	"fmt"

	"github.com/yourusername/codegraph/pkg/config"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Client defines the interface for LLM providers
type Client interface {
	// Chat sends messages to the LLM and returns the response
	Chat(ctx context.Context, messages []Message) (string, error)
}

// New creates an LLM client based on configuration
func New(cfg config.LLMConfig) (Client, error) {
	switch cfg.Provider {
	case "anthropic":
		return NewAnthropicClient(cfg.Model, cfg.APIKeyEnv)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}
