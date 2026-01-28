package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/yourusername/codegraph/pkg/config"
	"github.com/yourusername/codegraph/pkg/embedder"
	"github.com/yourusername/codegraph/pkg/indexer"
	"github.com/yourusername/codegraph/pkg/metadata"
	"github.com/yourusername/codegraph/pkg/parser"
	"github.com/yourusername/codegraph/pkg/query"
	"github.com/yourusername/codegraph/pkg/vectorstore"
)

var version = "0.1.0"

var configPath string

func getConfigPath() string {
	if configPath != "" {
		return configPath
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codegraph", "config.yaml")
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "codegraph",
		Short: "CodeGraph - A code knowledge base tool",
		Long: `CodeGraph ingests multiple code repositories and creates a queryable
vector store for LLM-powered code understanding.`,
		Version: version,
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to config file (default: ~/.codegraph/config.yaml)")

	rootCmd.AddCommand(indexCmd())
	rootCmd.AddCommand(queryCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(deleteCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func indexCmd() *cobra.Command {
	var (
		projectPath string
		projectName string
		groupName   string
		description string
		clean       bool
	)

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Index a code project",
		Long:  `Parse and index a code project into the vector store`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectPath == "" {
				return fmt.Errorf("--path is required")
			}
			if projectName == "" {
				return fmt.Errorf("--name is required")
			}

			// Load configuration
			cfg, err := config.LoadOrDefault(getConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			fmt.Printf("Indexing project: %s from path: %s\n", projectName, projectPath)

			// Initialize metadata store
			metaStore, err := metadata.NewSQLiteStore(cfg.Metadata.DBPath)
			if err != nil {
				return fmt.Errorf("failed to create metadata store: %w", err)
			}
			defer metaStore.Close()

			// Initialize components
			fmt.Println("Initializing embedder...")
			emb, err := embedder.New(cfg.Embeddings)
			if err != nil {
				return fmt.Errorf("failed to create embedder: %w", err)
			}

			fmt.Println("Initializing vector store...")
			store, err := vectorstore.New(cfg.ToVectorStoreConfig())
			if err != nil {
				return fmt.Errorf("failed to create vector store: %w", err)
			}
			defer store.Close()

			fmt.Println("Initializing parser...")
			parser := parser.NewGoParser()

			// Create indexer
			idx := indexer.New(parser, emb, store)

			ctx := context.Background()

			// Clean re-index: delete existing project first
			if clean {
				fmt.Printf("Cleaning existing data for project: %s\n", projectName)
				if err := idx.DeleteProject(ctx, projectName); err != nil {
					// Don't fail if project doesn't exist
					fmt.Printf("Note: Could not delete existing project (may not exist): %v\n", err)
				}
				// Also delete from metadata store
				metaStore.DeleteProject(ctx, projectName)
			}

			// Run indexing
			chunkCount, err := idx.IndexProject(ctx, projectPath, projectName)
			if err != nil {
				return fmt.Errorf("indexing failed: %w", err)
			}

			// Record metadata
			now := time.Now()
			project := &metadata.Project{
				Name:          projectName,
				Path:          projectPath,
				Language:      parser.Language(),
				Description:   description,
				ChunkCount:    chunkCount,
				LastIndexedAt: &now,
			}

			// Get group ID if group specified
			if groupName != "" {
				group, err := metaStore.GetGroup(ctx, groupName)
				if err != nil {
					// Group doesn't exist, create it
					group, err = metaStore.CreateGroup(ctx, groupName, "")
					if err != nil {
						return fmt.Errorf("failed to create group: %w", err)
					}
				}
				project.GroupID = &group.ID
			}

			// Check if project exists
			existing, err := metaStore.GetProject(ctx, projectName)
			if err == nil {
				// Update existing project
				project.ID = existing.ID
				if err := metaStore.UpdateProject(ctx, project); err != nil {
					return fmt.Errorf("failed to update project metadata: %w", err)
				}
			} else {
				// Create new project
				if err := metaStore.CreateProject(ctx, project); err != nil {
					return fmt.Errorf("failed to create project metadata: %w", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&projectPath, "path", "p", "", "Path to the project directory (required)")
	cmd.Flags().StringVarP(&projectName, "name", "n", "", "Name of the project (required)")
	cmd.Flags().StringVarP(&groupName, "group", "g", "", "Group name to organize projects")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Project description")
	cmd.Flags().BoolVar(&clean, "clean", false, "Delete existing project data before indexing (ensures no orphaned chunks)")

	return cmd
}

func queryCmd() *cobra.Command {
	var (
		queryText   string
		limit       int
		projectName string
	)

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query the code knowledge base",
		Long:  `Search the indexed codebase using natural language`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if queryText == "" {
				return fmt.Errorf("--query is required")
			}

			// Load configuration
			cfg, err := config.LoadOrDefault(getConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			fmt.Printf("Querying: %s\n", queryText)

			// Initialize components
			emb, err := embedder.New(cfg.Embeddings)
			if err != nil {
				return fmt.Errorf("failed to create embedder: %w", err)
			}

			store, err := vectorstore.New(cfg.ToVectorStoreConfig())
			if err != nil {
				return fmt.Errorf("failed to create vector store: %w", err)
			}
			defer store.Close()

			// Create query engine
			engine := query.NewEngine(emb, store)

			// Build filters
			var filters map[string]interface{}
			if projectName != "" {
				filters = map[string]interface{}{
					"project": projectName,
				}
				fmt.Printf("Filtering by project: %s\n", projectName)
			}

			// Execute query
			ctx := context.Background()
			results, err := engine.Query(ctx, queryText, limit, filters)
			if err != nil {
				return fmt.Errorf("query failed: %w", err)
			}

			// Display results
			fmt.Printf("\nFound %d results:\n\n", len(results))
			for i, result := range results {
				chunk := result.Chunk
				fmt.Printf("=== Result %d (Score: %.4f) ===\n", i+1, result.Score)
				fmt.Printf("Project: %s\n", chunk.Project)
				fmt.Printf("File: %s:%d-%d\n", chunk.FilePath, chunk.LineStart, chunk.LineEnd)
				fmt.Printf("Type: %s %s\n", chunk.ChunkType, chunk.Name)
				if chunk.DocString != "" {
					fmt.Printf("Docs: %s\n", chunk.DocString)
				}
				fmt.Printf("\n%s\n\n", chunk.Code)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&queryText, "query", "q", "", "Query text (required)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 5, "Maximum number of results")
	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Filter by project name")

	return cmd
}

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all indexed projects",
		Long:  `Display all projects that have been indexed`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := config.LoadOrDefault(getConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Initialize vector store
			store, err := vectorstore.New(cfg.ToVectorStoreConfig())
			if err != nil {
				return fmt.Errorf("failed to create vector store: %w", err)
			}
			defer store.Close()

			// List projects
			ctx := context.Background()
			projects, err := store.ListProjects(ctx)
			if err != nil {
				return fmt.Errorf("failed to list projects: %w", err)
			}

			if len(projects) == 0 {
				fmt.Println("No projects indexed yet.")
				return nil
			}

			fmt.Printf("Indexed projects (%d):\n", len(projects))
			for i, project := range projects {
				fmt.Printf("  %d. %s\n", i+1, project)
			}

			return nil
		},
	}

	return cmd
}

func deleteCmd() *cobra.Command {
	var projectName string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a project from the index",
		Long:  `Remove all data for a project from the vector store`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectName == "" {
				return fmt.Errorf("--name is required")
			}

			// Load configuration
			cfg, err := config.LoadOrDefault(getConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			fmt.Printf("Deleting project: %s\n", projectName)

			// Initialize vector store
			store, err := vectorstore.New(cfg.ToVectorStoreConfig())
			if err != nil {
				return fmt.Errorf("failed to create vector store: %w", err)
			}
			defer store.Close()

			// Delete project
			ctx := context.Background()
			if err := store.Delete(ctx, projectName); err != nil {
				return fmt.Errorf("failed to delete project: %w", err)
			}

			fmt.Printf("✓ Project '%s' deleted successfully\n", projectName)
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectName, "name", "n", "", "Name of the project to delete (required)")

	return cmd
}
