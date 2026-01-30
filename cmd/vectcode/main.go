package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/jayzheng/vectcode/pkg/config"
	"github.com/jayzheng/vectcode/pkg/embedder"
	"github.com/jayzheng/vectcode/pkg/indexer"
	"github.com/jayzheng/vectcode/pkg/metadata"
	"github.com/jayzheng/vectcode/pkg/parser"
	"github.com/jayzheng/vectcode/pkg/query"
	"github.com/jayzheng/vectcode/pkg/vectorstore"
)

var version = "0.1.0"

var configPath string

func getConfigPath() string {
	if configPath != "" {
		return configPath
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".vectcode", "config.yaml")
}

func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("2006-01-02 15:04")
	}
}

func formatProjectList(projects []string) string {
	if len(projects) == 0 {
		return ""
	}
	if len(projects) <= 3 {
		return fmt.Sprintf("%v", projects)
	}
	return fmt.Sprintf("%s and %d more", projects[0], len(projects)-1)
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "vectcode",
		Short: "VectCode - A code knowledge base tool",
		Long: `VectCode ingests multiple code repositories and creates a queryable
vector store for LLM-powered code understanding.`,
		Version: version,
	}

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to config file (default: ~/.vectcode/config.yaml)")

	rootCmd.AddCommand(indexCmd())
	rootCmd.AddCommand(queryCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(infoCmd())
	rootCmd.AddCommand(deleteCmd())
	rootCmd.AddCommand(groupCmd())

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
		groupName   string
	)

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query the code knowledge base",
		Long:  `Search the indexed codebase using natural language`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if queryText == "" {
				return fmt.Errorf("--query is required")
			}

			// Can't specify both project and group
			if projectName != "" && groupName != "" {
				return fmt.Errorf("cannot specify both --project and --group")
			}

			// Load configuration
			cfg, err := config.LoadOrDefault(getConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			ctx := context.Background()

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
			} else if groupName != "" {
				// Get projects in the group
				metaStore, err := metadata.NewSQLiteStore(cfg.Metadata.DBPath)
				if err != nil {
					return fmt.Errorf("failed to create metadata store: %w", err)
				}
				defer metaStore.Close()

				projects, err := metaStore.GetProjectsByGroup(ctx, groupName)
				if err != nil {
					return fmt.Errorf("failed to get projects in group: %w", err)
				}

				if len(projects) == 0 {
					return fmt.Errorf("no projects found in group '%s'", groupName)
				}

				// Build list of project names
				projectNames := make([]string, len(projects))
				for i, proj := range projects {
					projectNames[i] = proj.Name
				}

				filters = map[string]interface{}{
					"projects": projectNames,
				}
				fmt.Printf("Filtering by group '%s' (%d projects: %s)\n",
					groupName, len(projectNames), formatProjectList(projectNames))
			}

			// Execute query
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
	cmd.Flags().StringVarP(&groupName, "group", "g", "", "Filter by group name (searches all projects in group)")

	return cmd
}

func listCmd() *cobra.Command {
	var (
		detailed  bool
		groupName string
	)

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

			ctx := context.Background()

			// Initialize metadata store
			metaStore, err := metadata.NewSQLiteStore(cfg.Metadata.DBPath)
			if err != nil {
				return fmt.Errorf("failed to create metadata store: %w", err)
			}
			defer metaStore.Close()

			// Build filter
			var filter *metadata.ProjectFilter
			if groupName != "" {
				filter = &metadata.ProjectFilter{GroupName: groupName}
			}

			// List projects from metadata
			projects, err := metaStore.ListProjects(ctx, filter)
			if err != nil {
				return fmt.Errorf("failed to list projects: %w", err)
			}

			if len(projects) == 0 {
				if groupName != "" {
					fmt.Printf("No projects found in group '%s'.\n", groupName)
				} else {
					fmt.Println("No projects indexed yet.")
				}
				return nil
			}

			if detailed {
				// Detailed view
				fmt.Printf("Indexed projects (%d):\n\n", len(projects))
				for _, project := range projects {
					fmt.Printf("Name: %s\n", project.Name)
					fmt.Printf("  Path: %s\n", project.Path)
					fmt.Printf("  Language: %s\n", project.Language)
					if project.Description != "" {
						fmt.Printf("  Description: %s\n", project.Description)
					}
					if project.GroupName != "" {
						fmt.Printf("  Group: %s\n", project.GroupName)
					}
					fmt.Printf("  Chunks: %d\n", project.ChunkCount)
					if project.LastIndexedAt != nil {
						fmt.Printf("  Last indexed: %s\n", formatTimeAgo(*project.LastIndexedAt))
					} else {
						fmt.Printf("  Last indexed: never\n")
					}
					fmt.Println()
				}
			} else {
				// Simple view
				if groupName != "" {
					fmt.Printf("Projects in group '%s' (%d):\n", groupName, len(projects))
				} else {
					fmt.Printf("Indexed projects (%d):\n", len(projects))
				}
				for i, project := range projects {
					if project.GroupName != "" {
						fmt.Printf("  %d. %s [%s]\n", i+1, project.Name, project.GroupName)
					} else {
						fmt.Printf("  %d. %s\n", i+1, project.Name)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&detailed, "detailed", false, "Show detailed project information")
	cmd.Flags().StringVarP(&groupName, "group", "g", "", "Filter by group name")

	return cmd
}

func infoCmd() *cobra.Command {
	var projectName string

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show detailed information about a project",
		Long:  `Display comprehensive information about an indexed project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectName == "" {
				return fmt.Errorf("--name is required")
			}

			// Load configuration
			cfg, err := config.LoadOrDefault(getConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			ctx := context.Background()

			// Initialize metadata store
			metaStore, err := metadata.NewSQLiteStore(cfg.Metadata.DBPath)
			if err != nil {
				return fmt.Errorf("failed to create metadata store: %w", err)
			}
			defer metaStore.Close()

			// Get project
			project, err := metaStore.GetProject(ctx, projectName)
			if err != nil {
				return fmt.Errorf("project not found: %s", projectName)
			}

			// Display project info
			fmt.Printf("Project: %s\n", project.Name)
			fmt.Printf("  Path: %s\n", project.Path)
			fmt.Printf("  Language: %s\n", project.Language)

			if project.Description != "" {
				fmt.Printf("  Description: %s\n", project.Description)
			}

			if project.GroupName != "" {
				fmt.Printf("  Group: %s\n", project.GroupName)
			} else {
				fmt.Printf("  Group: (none)\n")
			}

			fmt.Printf("  Chunks: %d\n", project.ChunkCount)

			if project.LastIndexedAt != nil {
				fmt.Printf("  Last indexed: %s (%s)\n",
					project.LastIndexedAt.Format("2006-01-02 15:04:05"),
					formatTimeAgo(*project.LastIndexedAt))
			} else {
				fmt.Printf("  Last indexed: never\n")
			}

			fmt.Printf("  Created: %s\n", project.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Updated: %s\n", project.UpdatedAt.Format("2006-01-02 15:04:05"))

			// Get file count if available
			files, err := metaStore.ListFiles(ctx, project.ID)
			if err == nil && len(files) > 0 {
				fmt.Printf("  Files tracked: %d\n", len(files))

				// Check for stale files
				staleFiles, err := metaStore.GetStaleFiles(ctx, project.ID)
				if err == nil && len(staleFiles) > 0 {
					fmt.Printf("  ⚠ Stale files (need re-indexing): %d\n", len(staleFiles))
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&projectName, "name", "n", "", "Name of the project (required)")

	return cmd
}

func deleteCmd() *cobra.Command {
	var projectName string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a project from the index",
		Long:  `Remove all data for a project from the vector store and metadata`,
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

			ctx := context.Background()

			// Initialize metadata store
			metaStore, err := metadata.NewSQLiteStore(cfg.Metadata.DBPath)
			if err != nil {
				return fmt.Errorf("failed to create metadata store: %w", err)
			}
			defer metaStore.Close()

			// Initialize vector store
			store, err := vectorstore.New(cfg.ToVectorStoreConfig())
			if err != nil {
				return fmt.Errorf("failed to create vector store: %w", err)
			}
			defer store.Close()

			// Delete from vector store
			if err := store.Delete(ctx, projectName); err != nil {
				return fmt.Errorf("failed to delete project from vector store: %w", err)
			}

			// Delete from metadata store
			if err := metaStore.DeleteProject(ctx, projectName); err != nil {
				// Don't fail if not in metadata (might be old project)
				fmt.Printf("Note: Project metadata not found (may be from before metadata store)\n")
			}

			fmt.Printf("✓ Project '%s' deleted successfully\n", projectName)
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectName, "name", "n", "", "Name of the project to delete (required)")

	return cmd
}

func groupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Manage project groups",
		Long:  `Create, list, and delete project groups for organizing related projects`,
	}

	cmd.AddCommand(groupCreateCmd())
	cmd.AddCommand(groupListCmd())
	cmd.AddCommand(groupDeleteCmd())

	return cmd
}

func groupCreateCmd() *cobra.Command {
	var (
		name        string
		description string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new group",
		Long:  `Create a new group for organizing projects`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			// Load configuration
			cfg, err := config.LoadOrDefault(getConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			ctx := context.Background()

			// Initialize metadata store
			metaStore, err := metadata.NewSQLiteStore(cfg.Metadata.DBPath)
			if err != nil {
				return fmt.Errorf("failed to create metadata store: %w", err)
			}
			defer metaStore.Close()

			// Create group
			group, err := metaStore.CreateGroup(ctx, name, description)
			if err != nil {
				return fmt.Errorf("failed to create group: %w", err)
			}

			fmt.Printf("✓ Created group '%s'\n", group.Name)
			if description != "" {
				fmt.Printf("  Description: %s\n", description)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Group name (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Group description")

	return cmd
}

func groupListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all groups",
		Long:  `Display all project groups`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := config.LoadOrDefault(getConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			ctx := context.Background()

			// Initialize metadata store
			metaStore, err := metadata.NewSQLiteStore(cfg.Metadata.DBPath)
			if err != nil {
				return fmt.Errorf("failed to create metadata store: %w", err)
			}
			defer metaStore.Close()

			// List groups
			groups, err := metaStore.ListGroups(ctx)
			if err != nil {
				return fmt.Errorf("failed to list groups: %w", err)
			}

			if len(groups) == 0 {
				fmt.Println("No groups found.")
				return nil
			}

			fmt.Printf("Groups (%d):\n\n", len(groups))
			for _, group := range groups {
				// Get project count for this group
				projects, _ := metaStore.GetProjectsByGroup(ctx, group.Name)
				projectCount := len(projects)

				fmt.Printf("Name: %s\n", group.Name)
				if group.Description != "" {
					fmt.Printf("  Description: %s\n", group.Description)
				}
				fmt.Printf("  Projects: %d\n", projectCount)
				fmt.Printf("  Created: %s\n", formatTimeAgo(group.CreatedAt))
				fmt.Println()
			}

			return nil
		},
	}

	return cmd
}

func groupDeleteCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a group",
		Long:  `Delete a group (projects in the group will remain, just unassigned)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			// Load configuration
			cfg, err := config.LoadOrDefault(getConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			ctx := context.Background()

			// Initialize metadata store
			metaStore, err := metadata.NewSQLiteStore(cfg.Metadata.DBPath)
			if err != nil {
				return fmt.Errorf("failed to create metadata store: %w", err)
			}
			defer metaStore.Close()

			// Check how many projects are in this group
			projects, err := metaStore.GetProjectsByGroup(ctx, name)
			if err == nil && len(projects) > 0 {
				fmt.Printf("Note: %d project(s) in this group will be unassigned.\n", len(projects))
			}

			// Delete group
			if err := metaStore.DeleteGroup(ctx, name); err != nil {
				return fmt.Errorf("failed to delete group: %w", err)
			}

			fmt.Printf("✓ Group '%s' deleted successfully\n", name)
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Group name (required)")

	return cmd
}
