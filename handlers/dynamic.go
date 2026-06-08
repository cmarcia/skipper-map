package handlers

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"skipperMCP/models"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolEntry represents a tool in the catalog.
type ToolEntry struct {
	Tool *mcp.Tool
	Load func(s *mcp.Server)
}

// DynamicToolHandler manages the progressive discovery of tools.
type DynamicToolHandler struct {
	catalog     map[string]ToolEntry
	activeTools map[string]bool
	server      *mcp.Server
	mu          sync.RWMutex
}

func NewDynamicToolHandler(server *mcp.Server) *DynamicToolHandler {
	return &DynamicToolHandler{
		catalog:     make(map[string]ToolEntry),
		activeTools: make(map[string]bool),
		server:      server,
	}
}

// RegisterCatalogTool adds a tool to the internal catalog with a loader function.
func (h *DynamicToolHandler) RegisterCatalogTool(tool *mcp.Tool, loader func(s *mcp.Server)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.catalog[tool.Name] = ToolEntry{
		Tool: tool,
		Load: loader,
	}
}

// SearchCatalog allows the LLM to find tools based on a query.
func (h *DynamicToolHandler) SearchCatalog(ctx context.Context, req *mcp.CallToolRequest, input models.SearchCatalogInput) (*mcp.CallToolResult, any, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var results []string
	query := strings.ToLower(input.Query)

	for name, entry := range h.catalog {
		if strings.Contains(strings.ToLower(name), query) || 
		   strings.Contains(strings.ToLower(entry.Tool.Description), query) {
			status := "available"
			if h.activeTools[name] {
				status = "already loaded"
			}
			results = append(results, fmt.Sprintf("- **%s**: %s (%s)", name, entry.Tool.Description, status))
		}
	}

	if len(results) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "No tools matching your query were found in the catalog."},
			},
		}, nil, nil
	}

	responseText := "Found the following tools in the catalog. Use `load_tool` to activate any of them:\n\n" + strings.Join(results, "\n")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: responseText},
		},
	}, nil, nil
}

// LoadTool moves a tool from the catalog to the active server tools.
func (h *DynamicToolHandler) LoadTool(ctx context.Context, req *mcp.CallToolRequest, input models.LoadToolInput) (*mcp.CallToolResult, any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	entry, exists := h.catalog[input.ToolName]
	if !exists {
		return models.ErrorResponse(fmt.Sprintf("Tool '%s' not found in catalog.", input.ToolName)), nil, nil
	}

	if h.activeTools[input.ToolName] {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Tool '%s' is already loaded and ready to use.", input.ToolName)},
			},
		}, nil, nil
	}

	// Add the tool to the MCP server using the loader function
	entry.Load(h.server)
	h.activeTools[input.ToolName] = true

	// Note: The Go SDK AddTool calls changeAndNotify internally, 
	// which debounces and sends notifications/tools/list_changed automatically
	// to all connected sessions if Capabilities.Tools.ListChanged is true.

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Tool '%s' has been loaded successfully. You can now use it.", input.ToolName)},
		},
	}, nil, nil
}

// UnloadTool removes a tool from the active set to clean up context.
func (h *DynamicToolHandler) UnloadTool(ctx context.Context, req *mcp.CallToolRequest, input models.LoadToolInput) (*mcp.CallToolResult, any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.activeTools[input.ToolName] {
		return models.ErrorResponse(fmt.Sprintf("Tool '%s' is not currently loaded.", input.ToolName)), nil, nil
	}

	// Use RemoveTools method from the SDK
	h.server.RemoveTools(input.ToolName)
	delete(h.activeTools, input.ToolName)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("Tool '%s' has been unloaded.", input.ToolName)},
		},
	}, nil, nil
}
