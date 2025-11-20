# MCP Tool Registration Pattern

## Overview

Standard pattern for registering Model Context Protocol (MCP) tools with proper schema definition, parameter validation, and handler functions in Hyperion services.

## Technology

- Model Context Protocol (MCP) SDK
- JSON Schema for parameter validation
- Go context-aware handlers

## Use Case

Use this pattern when implementing new MCP tools that need to be exposed through the MCP server. This pattern ensures consistent tool registration with proper schema validation and error handling.

## Implementation

### Tool Handler Structure

**File Reference**: `hyper/internal/mcp/handlers/qdrant_tools.go:39-67, 70-117`

```go
// 1. Create tool handler
type QdrantToolHandler struct {
    qdrantClient      storage.QdrantClientInterface
    knowledgeStorage  storage.KnowledgeStorage
    metadataRegistry  *ToolMetadataRegistry
}

// 2. Register all tools
func (h *QdrantToolHandler) RegisterQdrantTools(server *mcp.Server) error {
    if err := h.registerKnowledgeFind(server); err != nil {
        return fmt.Errorf("failed to register knowledge_find tool: %w", err)
    }
    // ... register other tools
    return nil
}

// 3. Register individual tool
func (h *QdrantToolHandler) registerKnowledgeFind(server *mcp.Server) error {
    tool := &mcp.Tool{
        Name:        "knowledge_find",
        Description: "Search for knowledge by semantic similarity...",
        InputSchema: &jsonschema.Schema{
            Type: "object",
            Properties: map[string]*jsonschema.Schema{
                "collectionName": {
                    Type:        "string",
                    Description: "Collection name to search",
                },
                "query": {
                    Type:        "string",
                    Description: "Search query text",
                },
            },
            Required: []string{"collectionName", "query"},
        },
    }

    // 4. Add tool with handler function
    server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        args, err := extractArguments(req)
        if err != nil {
            return createErrorResult(err.Error()), nil
        }
        return h.handleKnowledgeFind(args)
    })

    return nil
}
```

## Key Points

### Tool Definition Components

1. **Tool Name**: snake_case identifier (e.g., `knowledge_find`, `task_create`)
2. **Description**: Clear, concise description of what the tool does
3. **Input Schema**: JSON Schema defining required and optional parameters
4. **Handler Function**: Context-aware function that processes tool invocations

### Schema Definition Best Practices

```go
InputSchema: &jsonschema.Schema{
    Type: "object",
    Properties: map[string]*jsonschema.Schema{
        "requiredParam": {
            Type:        "string",
            Description: "Clear description of parameter purpose",
        },
        "optionalParam": {
            Type:        "number",
            Description: "Optional parameter with default behavior",
        },
    },
    Required: []string{"requiredParam"}, // List required parameters
}
```

### Handler Function Pattern

```go
server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // 1. Extract and validate arguments
    args, err := extractArguments(req)
    if err != nil {
        return createErrorResult(err.Error()), nil
    }

    // 2. Execute business logic
    result, err := h.handleToolLogic(args)
    if err != nil {
        return createErrorResult(err.Error()), nil
    }

    // 3. Return formatted result
    return &mcp.CallToolResult{
        Content: []interface{}{
            mcp.TextContent{
                Type: "text",
                Text: result,
            },
        },
    }, nil
})
```

### Error Handling

```go
func createErrorResult(errMsg string) *mcp.CallToolResult {
    return &mcp.CallToolResult{
        IsError: true,
        Content: []interface{}{
            mcp.TextContent{
                Type: "text",
                Text: errMsg,
            },
        },
    }
}
```

### Registration Flow

1. **Handler Struct**: Create handler struct with dependencies (storage, clients)
2. **Batch Registration**: Implement `RegisterTools()` method that registers all related tools
3. **Individual Registration**: Each tool has its own register method for clarity
4. **Server Connection**: Use `server.AddTool()` to connect tool definition to handler

### Naming Conventions

- **Tool names**: snake_case (e.g., `knowledge_find`, `coordinator_create_task`)
- **Parameters**: camelCase in JSON (e.g., `collectionName`, `vectorSize`)
- **Go structs**: PascalCase (e.g., `QdrantToolHandler`, `TaskStorage`)

## Metadata

- **Domain**: protocol
- **Language**: go
- **Pattern**: tool-registration
- **Technology**: mcp
