# Project Instructions for Claude Code

## IMPORTANT: Always Use Semantic Search First

**For ALL indexed projects (including this one), use semantic search before making changes.**

### Required Workflow:
1. **Semantic Search First**:
   ```bash
   ./codegraph query --query "relevant functionality" --project <project-name> --limit 5
   ```

2. **Read Specific Files** - Based on search results

3. **Make Changes** - With full context

## Currently Indexed Projects

### codegraph (this project)
- **Description**: Semantic code search tool with vector embeddings and ChromaDB
- **Group**: `coding`
- **Chunks**: 133
- **Status**: ✅ Indexed and searchable!

### nflcom
- **Description**: NFL.com data scraper for fantasy football
- **Group**: `fantasy football data`
- **Chunks**: 96

## Working on CodeGraph Itself

**YES, search codegraph semantically too!**

### Examples for this codebase:
```bash
# Find metadata store implementation
./codegraph query --query "metadata store SQLite" --project codegraph

# Find ChromaDB integration
./codegraph query --query "vector search ChromaDB" --project codegraph

# Find CLI command handlers
./codegraph query --query "index command implementation" --project codegraph

# Search the coding group (includes codegraph)
./codegraph query --query "configuration loading" --group coding
```

### Why search codegraph itself:
- 133 chunks indexed across all packages
- Finds relevant code by semantic meaning
- Faster than browsing directory structure
- See related implementations across the codebase

## Query Command Reference

```bash
# Search specific project
./codegraph query --query "your search" --project <name> --limit 5

# Search by group
./codegraph query --query "your search" --group <group-name>

# Get project info
./codegraph info --name <name>

# List projects
./codegraph list --detailed
./codegraph list --group <group-name>

# Manage groups
./codegraph group list
./codegraph group create --name <name> --description "..."
```

## Testing Changes

After making changes to codegraph:
1. Rebuild: `go build -o codegraph ./cmd/codegraph`
2. Test the specific feature changed
3. Commit with descriptive message
4. Consider re-indexing if major changes: `./codegraph index --path . --name codegraph --clean`

## User Preference

The user wants semantic search used for ALL projects, including codegraph itself. This provides:
- Better context discovery
- Semantic understanding (not just keyword matching)
- Faster navigation through 133+ code chunks
- Consistent workflow across all projects
