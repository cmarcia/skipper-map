package handlers

import (
	"context"
	"strings"
	"testing"

	"skipperMCP/models"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDynamicToolHandler(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	h := NewDynamicToolHandler(server)

	toolName := "test_tool"
	toolDesc := "A test tool description"
	
	// 1. Register a tool in the catalog
	h.RegisterCatalogTool(&mcp.Tool{
		Name:        toolName,
		Description: toolDesc,
	}, func(s *mcp.Server) {
		mcp.AddTool(s, &mcp.Tool{Name: toolName, Description: toolDesc}, func(ctx context.Context, req *mcp.CallToolRequest, input struct{}) (*mcp.CallToolResult, any, error) {
			return nil, nil, nil
		})
	})

	ctx := context.Background()

	// 2. Search for the tool
	searchRes, _, err := h.SearchCatalog(ctx, nil, models.SearchCatalogInput{Query: "test"})
	if err != nil {
		t.Fatalf("SearchCatalog failed: %v", err)
	}
	searchText := searchRes.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(searchText, toolName) {
		t.Errorf("Search result should contain tool name, got: %s", searchText)
	}

	// 3. Load the tool
	loadRes, _, err := h.LoadTool(ctx, nil, models.LoadToolInput{ToolName: toolName})
	if err != nil {
		t.Fatalf("LoadTool failed: %v", err)
	}
	loadText := loadRes.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(loadText, "successfully") {
		t.Errorf("Load result should indicate success, got: %s", loadText)
	}

	// 4. Verify tool is now active
	if !h.activeTools[toolName] {
		t.Error("Tool should be marked as active")
	}

	// 5. Unload the tool
	unloadRes, _, err := h.UnloadTool(ctx, nil, models.LoadToolInput{ToolName: toolName})
	if err != nil {
		t.Fatalf("UnloadTool failed: %v", err)
	}
	unloadText := unloadRes.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(unloadText, "unloaded") {
		t.Errorf("Unload result should indicate success, got: %s", unloadText)
	}

	// 6. Verify tool is no longer active
	if h.activeTools[toolName] {
		t.Error("Tool should not be marked as active after unloading")
	}
}
