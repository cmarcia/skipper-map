package handlers

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSkipperHandler_Resources(t *testing.T) {
	handler := NewSkipperHandler()
	ctx := context.Background()

	t.Run("Read Vessel Logs", func(t *testing.T) {
		req := &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{URI: "vessel://status/logs"},
		}
		result, err := handler.GetResource(ctx, req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(result.Contents) != 1 {
			t.Fatalf("Expected 1 content block, got %d", len(result.Contents))
		}
		if !strings.Contains(result.Contents[0].Text, "VESSEL LOGBOOK") {
			t.Errorf("Expected logbook content, got: %s", result.Contents[0].Text)
		}
	})

	t.Run("Read Marina Regulations", func(t *testing.T) {
		req := &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{URI: "marina://newport/regulations"},
		}
		result, err := handler.GetResource(ctx, req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !strings.Contains(result.Contents[0].Text, "REGULATIONS FOR MARINA: NEWPORT") {
			t.Errorf("Expected Newport regulations, got: %s", result.Contents[0].Text)
		}
	})

	t.Run("Resource Not Found", func(t *testing.T) {
		req := &mcp.ReadResourceRequest{
			Params: &mcp.ReadResourceParams{URI: "vessel://unknown"},
		}
		_, err := handler.GetResource(ctx, req)
		if err == nil {
			t.Fatal("Expected error for unknown URI")
		}
	})
}

func TestSkipperHandler_Prompts(t *testing.T) {
	handler := NewSkipperHandler()
	ctx := context.Background()

	t.Run("Get Arrival Briefing", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "arrival-briefing",
				Arguments: map[string]string{
					"marina_name": "Newport Harbor",
					"vessel_name": "SV Discovery",
				},
			},
		}
		result, err := handler.GetPrompt(ctx, req)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if result.Description == "" {
			t.Error("Expected prompt description")
		}
		if len(result.Messages) != 2 {
			t.Fatalf("Expected 2 messages, got %d", len(result.Messages))
		}
		
		userMsg := result.Messages[1].Content.(*mcp.TextContent).Text
		if !strings.Contains(userMsg, "SV Discovery") || !strings.Contains(userMsg, "Newport Harbor") {
			t.Errorf("Prompt message missing arguments: %s", userMsg)
		}
	})

	t.Run("Prompt Not Found", func(t *testing.T) {
		req := &mcp.GetPromptRequest{
			Params: &mcp.GetPromptParams{
				Name: "unknown-prompt",
			},
		}
		_, err := handler.GetPrompt(ctx, req)
		if err == nil {
			t.Fatal("Expected error for unknown prompt")
		}
	})
}
