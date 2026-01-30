# Project Instructions for Claude Code

## IMPORTANT: Always Use Semantic Search First

**For ALL indexed projects (including this one), use semantic search before making changes.**

### Required Workflow:
1. **Semantic Search First**:
   ```bash
   ./vectcode query --query "relevant functionality" --project <project-name> --limit 5
   ```

2. **Read Specific Files** - Based on search results

3. **Make Changes** - With full context

## Currently Indexed Projects

### vectcode (this project)
- **Description**: Semantic code search tool with vector embeddings and ChromaDB
- **Group**: `coding`
- **Chunks**: 133
- **Status**: ✅ Indexed and searchable!

### nflcom
- **Description**: NFL.com data scraper for fantasy football
- **Group**: `fantasy football data`
- **Chunks**: 96

## Working on VectCode Itself

**YES, search vectcode semantically too!**

### Examples for this codebase:
```bash
# Find metadata store implementation
./vectcode query --query "metadata store SQLite" --project vectcode

# Find ChromaDB integration
./vectcode query --query "vector search ChromaDB" --project vectcode

# Find CLI command handlers
./vectcode query --query "index command implementation" --project vectcode

# Search the coding group (includes vectcode)
./vectcode query --query "configuration loading" --group coding
```

### Why search vectcode itself:
- 133 chunks indexed across all packages
- Finds relevant code by semantic meaning
- Faster than browsing directory structure
- See related implementations across the codebase

## Query Command Reference

```bash
# Search specific project
./vectcode query --query "your search" --project <name> --limit 5

# Search by group
./vectcode query --query "your search" --group <group-name>

# Get project info
./vectcode info --name <name>

# List projects
./vectcode list --detailed
./vectcode list --group <group-name>

# Manage groups
./vectcode group list
./vectcode group create --name <name> --description "..."
```

## Testing Changes

After making changes to vectcode:
1. Rebuild: `go build -o vectcode ./cmd/vectcode`
2. Test the specific feature changed
3. Commit with descriptive message
4. Consider re-indexing if major changes: `./vectcode index --path . --name vectcode --clean`

## User Preference

The user wants semantic search used for ALL projects, including vectcode itself. This provides:
- Better context discovery
- Semantic understanding (not just keyword matching)
- Faster navigation through 133+ code chunks
- Consistent workflow across all projects
