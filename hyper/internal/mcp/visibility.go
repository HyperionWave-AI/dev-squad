package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolVisibility defines whether a tool should be visible in MCP listTools
type ToolVisibility string

const (
	// VisibilityPublic means the tool appears in MCP listTools
	VisibilityPublic ToolVisibility = "public"
	// VisibilityInternal means the tool is hidden from listTools but executable via execute_tool
	VisibilityInternal ToolVisibility = "internal"
)

// VisibilityConfig defines which tools should be public vs internal
type VisibilityConfig struct {
	// PublicTools is the set of tool names that should be visible in MCP listTools
	PublicTools map[string]bool
}

// NewVisibilityConfig creates a new visibility configuration
// By default, only discover_tools and execute_tool are public
func NewVisibilityConfig() *VisibilityConfig {
	return &VisibilityConfig{
		PublicTools: map[string]bool{
			"discover_tools": true,
			"execute_tool":   true,
		},
	}
}

// IsPublic checks if a tool should be visible in MCP listTools
func (vc *VisibilityConfig) IsPublic(toolName string) bool {
	return vc.PublicTools[toolName]
}

// AddPublicTool marks a tool as public (visible in listTools)
func (vc *VisibilityConfig) AddPublicTool(toolName string) {
	vc.PublicTools[toolName] = true
}

// FilteredServer wraps an MCP server to filter tools by visibility
type FilteredServer struct {
	*mcp.Server
	visibilityConfig *VisibilityConfig
}

// NewFilteredServer creates a new server wrapper with visibility filtering
func NewFilteredServer(server *mcp.Server, visibilityConfig *VisibilityConfig) *FilteredServer {
	return &FilteredServer{
		Server:           server,
		visibilityConfig: visibilityConfig,
	}
}

// ListTools overrides the default ListTools to only return public tools
func (fs *FilteredServer) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	// This is a conceptual implementation - actual implementation depends on SDK internals
	// The real filtering happens at the handler registration level
	// This struct is here for future extensibility if SDK supports custom ListTools override
	return nil, nil
}
