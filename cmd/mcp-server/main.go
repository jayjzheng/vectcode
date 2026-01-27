package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/yourusername/codegraph/pkg/mcp"
)

func main() {
	// Get config path from environment or use default
	configPath := os.Getenv("CODEGRAPH_CONFIG")
	if configPath == "" {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".codegraph", "config.yaml")
	}

	// Create and run MCP server
	server, err := mcp.NewServer(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}
	defer server.Close()

	// Run server (reads from stdin, writes to stdout)
	if err := server.Run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
