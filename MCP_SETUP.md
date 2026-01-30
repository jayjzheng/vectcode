# VectCode MCP Server Setup

This guide shows you how to use VectCode as an MCP (Model Context Protocol) server with Claude Desktop and other LLM clients.

## What is MCP?

MCP (Model Context Protocol) allows LLMs like Claude to access external tools and data sources during conversations. With VectCode as an MCP server, Claude can search through your indexed codebases in real-time.

## Prerequisites

1. **VectCode with indexed projects**:
   ```bash
   # Make sure you've indexed some projects
   ./vectcode list
   ```

2. **MCP server binary**:
   ```bash
   # Build the MCP server
   go build -o vectcode-mcp-server ./cmd/mcp-server

   # Move to a permanent location (optional)
   sudo cp vectcode-mcp-server /usr/local/bin/
   ```

3. **Running services**:
   - ChromaDB at `http://localhost:8000`
   - Ollama at `http://localhost:11434` with `bge-m3` model

## Setup for Claude Desktop

### 1. Find Claude Desktop Config Location

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`

**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

**Linux**: `~/.config/Claude/claude_desktop_config.json`

### 2. Configure the MCP Server

Edit your `claude_desktop_config.json` file:

```json
{
  "mcpServers": {
    "vectcode": {
      "command": "/path/to/vectcode-mcp-server",
      "env": {
        "VECTCODE_CONFIG": "/Users/yourusername/.vectcode/config.yaml"
      }
    }
  }
}
```

**Important**: Replace `/path/to/vectcode-mcp-server` with the actual path to your binary.

#### Example (macOS):

```json
{
  "mcpServers": {
    "vectcode": {
      "command": "/usr/local/bin/vectcode-mcp-server"
    }
  }
}
```

Or if using the binary from your project directory:

```json
{
  "mcpServers": {
    "vectcode": {
      "command": "/Users/jayzheng/projects/vectcode/vectcode-mcp-server"
    }
  }
}
```

### 3. Restart Claude Desktop

Close and reopen Claude Desktop for the changes to take effect.

### 4. Verify Connection

In Claude Desktop, you should see a 🔨 (hammer) icon indicating MCP tools are available. Click it to see the VectCode tools:

- **search_code**: Search indexed codebases
- **list_projects**: List all indexed projects

## Using VectCode in Claude Conversations

Once configured, you can ask Claude to search your code:

### Example Conversations:

**List projects**:
> "What projects are indexed in VectCode?"

**Search code**:
> "Search for functions that fetch team data"
> "Find the API client implementation"
> "Show me SQL insert operations"

**Project-specific search**:
> "Search the nflcom project for database operations"

**Code understanding**:
> "How does the team data fetching work in my codebase?"

Claude will use the `search_code` tool to find relevant code chunks and provide detailed answers based on your actual code.

## Available MCP Tools

### 1. search_code

Searches indexed codebases using semantic search.

**Parameters**:
- `query` (required): Natural language search query
- `project` (optional): Filter to specific project name
- `limit` (optional): Max results to return (default: 5)

**Returns**: Code chunks with file paths, line numbers, documentation, and code content.

### 2. list_projects

Lists all indexed projects available for search.

**Parameters**: None

**Returns**: List of project names.

## Troubleshooting

### MCP Server Not Showing in Claude Desktop

1. Check the config file path is correct
2. Verify the binary path is absolute (not relative)
3. Check Claude Desktop logs:
   - macOS: `~/Library/Logs/Claude/`
   - Windows: `%APPDATA%\Claude\logs\`

### "Connection failed" Error

1. Ensure ChromaDB is running:
   ```bash
   curl http://localhost:8000/api/v1/heartbeat
   ```

2. Ensure Ollama is running:
   ```bash
   curl http://localhost:11434/api/tags
   ```

3. Test MCP server manually:
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./vectcode-mcp-server
   ```

### No Projects Found

Index some projects first:
```bash
./vectcode index --path ~/projects/myproject --name myproject
./vectcode list
```

## Configuration

The MCP server uses the same config as the CLI tool: `~/.vectcode/config.yaml`

You can override the config path with the `VECTCODE_CONFIG` environment variable in your Claude Desktop config:

```json
{
  "mcpServers": {
    "vectcode": {
      "command": "/path/to/vectcode-mcp-server",
      "env": {
        "VECTCODE_CONFIG": "/custom/path/to/config.yaml"
      }
    }
  }
}
```

## Example Claude Desktop Config (Complete)

```json
{
  "mcpServers": {
    "vectcode": {
      "command": "/usr/local/bin/vectcode-mcp-server",
      "env": {
        "VECTCODE_CONFIG": "/Users/jayzheng/.vectcode/config.yaml"
      }
    }
  }
}
```

## Next Steps

Once configured, Claude can:
- Search your codebases during conversations
- Find relevant code examples
- Answer questions about your code architecture
- Help with debugging by finding similar implementations
- Provide context-aware coding suggestions

Try asking Claude: "Search my code for API authentication logic" or "Show me how database connections are handled in the nflcom project."
