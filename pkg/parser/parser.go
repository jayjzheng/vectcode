package parser

import (
	"context"
	
	"github.com/jayzheng/vectcode/pkg/chunker"
)

// Parser defines the interface for language-specific code parsers
type Parser interface {
	// Parse analyzes a project directory and extracts code chunks
	Parse(ctx context.Context, projectPath string, projectName string) ([]chunker.CodeChunk, error)
	
	// Language returns the programming language this parser handles
	Language() string
}
