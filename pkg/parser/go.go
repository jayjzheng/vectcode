package parser

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"time"
	
	"github.com/jayzheng/vectcode/pkg/chunker"
)

// GoParser implements Parser for Go language
type GoParser struct{}

// NewGoParser creates a new Go parser
func NewGoParser() *GoParser {
	return &GoParser{}
}

// Language returns "go"
func (p *GoParser) Language() string {
	return "go"
}

// Parse parses a Go project and extracts code chunks
func (p *GoParser) Parse(ctx context.Context, projectPath string, projectName string) ([]chunker.CodeChunk, error) {
	var chunks []chunker.CodeChunk
	
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			name := info.Name()
			// Skip vendor, node_modules, and hidden directories (but not "." or "..")
			if name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			// Skip hidden directories, but allow "." and ".."
			if len(name) > 1 && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		
		fileChunks, err := p.parseFile(path, projectName)
		if err != nil {
			fmt.Printf("Warning: failed to parse %s: %v\n", path, err)
			return nil
		}
		
		chunks = append(chunks, fileChunks...)
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to walk project directory: %w", err)
	}
	
	return chunks, nil
}

// parseFile parses a single Go file
func (p *GoParser) parseFile(filePath string, projectName string) ([]chunker.CodeChunk, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	
	var chunks []chunker.CodeChunk
	packageName := node.Name.Name
	imports := p.extractImports(node)
	
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			chunk := p.extractFunction(fset, x, filePath, projectName, packageName, imports, fileInfo.ModTime())
			chunks = append(chunks, chunk)
			
		case *ast.GenDecl:
			if x.Tok == token.TYPE {
				for _, spec := range x.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						chunk := p.extractType(fset, x, typeSpec, filePath, projectName, packageName, fileInfo.ModTime())
						if chunk != nil {
							chunks = append(chunks, *chunk)
						}
					}
				}
			}
		}
		return true
	})
	
	return chunks, nil
}

func (p *GoParser) extractFunction(fset *token.FileSet, fn *ast.FuncDecl, filePath, projectName, packageName string, imports []string, modTime time.Time) chunker.CodeChunk {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, fn)

	chunk := chunker.CodeChunk{
		ID:           generateID(projectName, filePath, fn.Name.Name),
		Project:      projectName,
		FilePath:     filePath,
		Package:      packageName,
		Language:     "go",
		Code:         buf.String(),
		Name:         fn.Name.Name,
		Imports:      imports,
		LineStart:    fset.Position(fn.Pos()).Line,
		LineEnd:      fset.Position(fn.End()).Line,
		LastModified: modTime,
	}
	
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		chunk.ChunkType = chunker.ChunkTypeMethod
		chunk.Receiver = p.extractReceiverType(fn.Recv)
	} else {
		chunk.ChunkType = chunker.ChunkTypeFunction
	}
	
	if fn.Doc != nil {
		chunk.DocString = fn.Doc.Text()
	}
	
	if fn.Body != nil {
		chunk.HTTPEndpoints = p.extractHTTPEndpoints(fn)
		chunk.HTTPCalls = p.extractHTTPCalls(fn)
	}
	
	return chunk
}

func (p *GoParser) extractType(fset *token.FileSet, genDecl *ast.GenDecl, typeSpec *ast.TypeSpec, filePath, projectName, packageName string, modTime time.Time) *chunker.CodeChunk {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, genDecl)
	
	chunk := &chunker.CodeChunk{
		ID:           generateID(projectName, filePath, typeSpec.Name.Name),
		Project:      projectName,
		FilePath:     filePath,
		Package:      packageName,
		Language:     "go",
		Code:         buf.String(),
		Name:         typeSpec.Name.Name,
		LineStart:    fset.Position(typeSpec.Pos()).Line,
		LineEnd:      fset.Position(typeSpec.End()).Line,
		LastModified: modTime,
	}
	
	switch typeSpec.Type.(type) {
	case *ast.StructType:
		chunk.ChunkType = chunker.ChunkTypeStruct
	case *ast.InterfaceType:
		chunk.ChunkType = chunker.ChunkTypeInterface
	default:
		return nil
	}
	
	if genDecl.Doc != nil {
		chunk.DocString = genDecl.Doc.Text()
	}
	
	return chunk
}

func (p *GoParser) extractImports(node *ast.File) []string {
	var imports []string
	for _, imp := range node.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, `"`)
			imports = append(imports, path)
		}
	}
	return imports
}

func (p *GoParser) extractReceiverType(recv *ast.FieldList) string {
	if len(recv.List) == 0 {
		return ""
	}
	
	field := recv.List[0]
	var buf bytes.Buffer
	printer.Fprint(&buf, token.NewFileSet(), field.Type)
	return buf.String()
}

func (p *GoParser) extractHTTPEndpoints(fn *ast.FuncDecl) []string {
	var endpoints []string
	
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				method := sel.Sel.Name
				if isHTTPMethod(method) {
					if len(call.Args) > 0 {
						if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							path := strings.Trim(lit.Value, `"`)
							endpoint := fmt.Sprintf("%s %s", method, path)
							endpoints = append(endpoints, endpoint)
						}
					}
				}
			}
		}
		return true
	})
	
	return endpoints
}

func (p *GoParser) extractHTTPCalls(fn *ast.FuncDecl) []string {
	var calls []string
	
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				method := sel.Sel.Name
				if method == "Get" || method == "Post" || method == "Put" || method == "Delete" || method == "Patch" {
					if len(call.Args) > 0 {
						if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							url := strings.Trim(lit.Value, `"`)
							calls = append(calls, url)
						}
					}
				}
			}
		}
		return true
	})
	
	return calls
}

func isHTTPMethod(s string) bool {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "Get", "Post", "Put", "Delete", "Patch", "Head", "Options"}
	for _, m := range methods {
		if s == m {
			return true
		}
	}
	return false
}

func generateID(projectName, filePath, name string) string {
	return fmt.Sprintf("%s:%s:%s", projectName, filePath, name)
}
