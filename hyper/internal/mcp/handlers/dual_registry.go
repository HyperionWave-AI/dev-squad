package handlers

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DualRegistry manages both MCP server registration (public tools)
// and internal tool registry (internal tools) based on tool hub mode
type DualRegistry struct {
	mcpServer            *mcp.Server
	internalToolRegistry *InternalToolRegistry
	metadataRegistry     *ToolMetadataRegistry
	toolHubMode          bool // If true, only register to internal registry (except public tools)
}

// NewDualRegistry creates a new dual registry
func NewDualRegistry(
	mcpServer *mcp.Server,
	internalToolRegistry *InternalToolRegistry,
	metadataRegistry *ToolMetadataRegistry,
	toolHubMode bool,
) *DualRegistry {
	return &DualRegistry{
		mcpServer:            mcpServer,
		internalToolRegistry: internalToolRegistry,
		metadataRegistry:     metadataRegistry,
		toolHubMode:          toolHubMode,
	}
}

// RegisterTool registers a tool to either MCP server (public) or internal registry (internal)
// based on tool hub mode and whether the tool is in the public list
func (dr *DualRegistry) RegisterTool(tool *mcp.Tool, handler mcp.ToolHandler, isPublic bool) {
	// Always register to metadata registry for indexing
	if dr.metadataRegistry != nil {
		dr.metadataRegistry.RegisterTool(
			tool.Name,
			tool.Description,
			map[string]interface{}{
				"type":        "mcp-tool",
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": tool.InputSchema,
			},
		)
	}

	// In tool hub mode, decide where to register based on isPublic flag
	if dr.toolHubMode {
		if isPublic {
			// Public tools go to MCP server (visible in listTools)
			dr.mcpServer.AddTool(tool, handler)
		} else {
			// Internal tools go to internal registry (hidden from listTools)
			if dr.internalToolRegistry != nil {
				dr.internalToolRegistry.RegisterInternalTool(tool, handler)
			}
		}
	} else {
		// Non-hub mode: all tools go to MCP server (traditional behavior)
		dr.mcpServer.AddTool(tool, handler)
	}
}

// IsToolHubMode returns whether tool hub mode is enabled
func (dr *DualRegistry) IsToolHubMode() bool {
	return dr.toolHubMode
}
