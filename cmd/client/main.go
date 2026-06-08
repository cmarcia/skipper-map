// Package main implements a Go MCP client that connects to the skipper-mcp
// server and uses OpenAI to demonstrate Progressive Tool Discovery.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	openai "github.com/sashabaranov/go-openai"
)

// llmModel is the OpenAI model used for all scenario runs.
const llmModel = "gpt-4o-mini"

// toolsChanged is an atomic flag set when the server notifies of a tool list change.
var toolsChanged atomic.Bool

func main() {
	// Load API key from .env (falls back to environment variable)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment")
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY must be set in .env or environment")
	}

	ctx := context.Background()

	// Ensure the server binary exists
	if _, err := os.Stat("./skipper-mcp"); err != nil {
		log.Fatal("skipper-mcp binary not found — run: go build -o skipper-mcp .")
	}

	// Create an MCP client with a handler for tool list changes.
	mcpClient := mcp.NewClient(&mcp.Implementation{
		Name:    "skipper-llm-client",
		Version: "1.1.0",
	}, &mcp.ClientOptions{
		ToolListChangedHandler: func(ctx context.Context, req *mcp.ToolListChangedRequest) {
			fmt.Println("\n  [MCP Event] notifications/tools/list_changed received")
			toolsChanged.Store(true)
		},
	})

	session, err := mcpClient.Connect(ctx, &mcp.CommandTransport{
		Command: exec.Command("./skipper-mcp"),
	}, nil)
	if err != nil {
		log.Fatalf("MCP connect: %v", err)
	}
	defer session.Close()

	oai := openai.NewClient(apiKey)

	// Initially, the server only exposes meta-tools (discovery tools).
	initialTools, err := fetchTools(ctx, session)
	if err != nil {
		log.Fatalf("list tools: %v", err)
	}

	fmt.Println("=== Skipper MCP Client — Progressive Discovery Demo ===")
	fmt.Printf("Connected to skipper-mcp server. %d tools initially available (Discovery Mode).\n", len(initialTools))
	for _, t := range initialTools {
		fmt.Printf("- %s: %s\n", t.Function.Name, t.Function.Description)
	}
	fmt.Println()

	// Run all scenarios
	scenarioA(ctx, oai, session, initialTools)
	scenarioB(ctx, oai, session, initialTools)
	scenarioC(ctx, oai, session, initialTools)
	scenarioD(ctx, oai, session, initialTools)
}

// ── Scenarios ──────────────────────────────────────────────────────────────

func scenarioA(ctx context.Context, oai *openai.Client, session *mcp.ClientSession, tools []openai.Tool) {
	printSectionHeader("Scenario A — MAYDAY: Man Overboard")
	fmt.Println("Demonstrates: Search Catalog → Load Tool → Use Tool")
	fmt.Println()

	query := "MAN OVERBOARD — we need the MOB procedure RIGHT NOW"
	fmt.Printf("User: %q\n\n", query)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a maritime safety assistant. Your toolset is dynamic. If you don't have a specific tool, use `search_catalog` to find it and `load_tool` to activate it. In an emergency, act immediately.",
		},
		{Role: openai.ChatMessageRoleUser, Content: query},
	}

	// We pass a pointer to tools slice so toolLoop can update it if list_changed occurs
	answer, err := toolLoop(ctx, oai, session, &tools, messages)
	if err != nil {
		fmt.Printf("Error: %v\n\n", err)
		return
	}
	fmt.Printf("Assistant: %s\n\n", answer)
}

func scenarioB(ctx context.Context, oai *openai.Client, session *mcp.ClientSession, tools []openai.Tool) {
	printSectionHeader("Scenario B — Passage Planning")
	fmt.Println("Demonstrates: multi-step discovery and usage")
	fmt.Println()

	query := "I'm leaving Newport heading to Block Island. How long will it take? Also search for and load any weather tools I might need."
	fmt.Printf("User: %q\n\n", query)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a maritime assistant. Use discovery tools to find and load navigation and weather capabilities as needed.",
		},
		{Role: openai.ChatMessageRoleUser, Content: query},
	}

	answer, err := toolLoop(ctx, oai, session, &tools, messages)
	if err != nil {
		fmt.Printf("Error: %v\n\n", err)
		return
	}
	fmt.Printf("Assistant: %s\n\n", answer)
}

func scenarioC(ctx context.Context, oai *openai.Client, session *mcp.ClientSession, tools []openai.Tool) {
	printSectionHeader("Scenario C — Capability Evolution")
	fmt.Println("Demonstrates: Dynamic Toolset cleanup using unload_tool")
	fmt.Println()

	query := "I want to check my engine maintenance status, then unload the maintenance tool when done."
	fmt.Printf("User: %q\n\n", query)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "You are a maritime assistant. Discover, load, use, and then UNLOAD tools to maintain a clean context window.",
		},
		{Role: openai.ChatMessageRoleUser, Content: query},
	}

	answer, err := toolLoop(ctx, oai, session, &tools, messages)
	if err != nil {
		fmt.Printf("Error: %v\n\n", err)
		return
	}
	fmt.Printf("Assistant: %s\n\n", answer)
}

func scenarioD(ctx context.Context, oai *openai.Client, session *mcp.ClientSession, tools []openai.Tool) {
	printSectionHeader("Scenario D — Harbor Arrival Briefing")
	fmt.Println("Demonstrates: Discovery combined with Resources & Prompts")
	fmt.Println()

	fmt.Println("Action: Fetching 'arrival-briefing' prompt from server...")
	prompt, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "arrival-briefing",
		Arguments: map[string]string{
			"marina_name": "Newport Harbor",
			"vessel_name": "SV Discovery",
		},
	})
	if err != nil {
		fmt.Printf("Error fetching prompt: %v\n\n", err)
		return
	}

	messages := make([]openai.ChatCompletionMessage, 0, len(prompt.Messages))
	for _, m := range prompt.Messages {
		role := openai.ChatMessageRoleUser
		if m.Role == mcp.Role("assistant") {
			role = openai.ChatMessageRoleAssistant
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: m.Content.(*mcp.TextContent).Text,
		})
	}

	// Add resource tool
	resourceTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "read_resource",
			Description: "Reads the content of a specific MCP resource URI (e.g., vessel://..., marina://...).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"uri": map[string]any{
						"type":        "string",
						"description": "The URI of the resource to read.",
					},
				},
				"required": []string{"uri"},
			},
		},
	}
	currentTools := append(tools, resourceTool)

	answer, err := toolLoop(ctx, oai, session, &currentTools, messages)
	if err != nil {
		fmt.Printf("Error: %v\n\n", err)
		return
	}
	fmt.Printf("Assistant: %s\n\n", answer)
}

// ── Core loop ──────────────────────────────────────────────────────────────

func toolLoop(
	ctx context.Context,
	oai *openai.Client,
	session *mcp.ClientSession,
	tools *[]openai.Tool,
	messages []openai.ChatCompletionMessage,
) (string, error) {
	for {
		resp, err := oai.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    llmModel,
			Messages: messages,
			Tools:    *tools,
		})
		if err != nil {
			return "", fmt.Errorf("OpenAI API: %w", err)
		}

		msg := resp.Choices[0].Message

		if len(msg.ToolCalls) == 0 {
			return msg.Content, nil
		}

		messages = append(messages, msg)

		for _, tc := range msg.ToolCalls {
			var content string
			if tc.Function.Name == "read_resource" {
				var args struct {
					URI string `json:"uri"`
				}
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				fmt.Printf("  [MCP] Resource requested: %s\n", args.URI)
				
				res, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: args.URI})
				if err != nil {
					content = fmt.Sprintf("resource error: %v", err)
				} else {
					content = res.Contents[0].Text
				}
				fmt.Printf("  [MCP] Resource Content:  %s\n\n", truncate(content, 100))
			} else {
				fmt.Printf("  [MCP] Calling tool: %s\n", tc.Function.Name)
				fmt.Printf("  [MCP] Arguments:    %s\n", tc.Function.Arguments)

				var args map[string]any
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					args = map[string]any{}
				}

				result, err := session.CallTool(ctx, &mcp.CallToolParams{
					Name:      tc.Function.Name,
					Arguments: args,
				})

				if err != nil {
					content = fmt.Sprintf("tool error: %v", err)
				} else {
					content = extractText(result)
					if result.IsError {
						content = "tool returned error: " + content
					}
				}
				fmt.Printf("  [MCP] Result:       %s\n\n", truncate(content, 300))

				// Check if we need to refresh the tools list
				if toolsChanged.Swap(false) {
					fmt.Println("  [Client] Refreshing tool schemas due to list_changed notification...")
					newTools, err := fetchTools(ctx, session)
					if err == nil {
						// Preserve 'read_resource' if it was manually added
						hasResource := false
						for _, t := range *tools {
							if t.Function.Name == "read_resource" {
								hasResource = true
								break
							}
						}
						if hasResource {
							resourceTool := (*tools)[len(*tools)-1] // Assuming it's at the end
							newTools = append(newTools, resourceTool)
						}
						*tools = newTools
						fmt.Printf("  [Client] Toolset updated: %d tools now available\n\n", len(*tools))
					}
				}
			}

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    content,
				ToolCallID: tc.ID,
			})
		}
	}
}

// ── MCP ↔ OpenAI adapters ──────────────────────────────────────────────────

func fetchTools(ctx context.Context, session *mcp.ClientSession) ([]openai.Tool, error) {
	resp, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	result := make([]openai.Tool, 0, len(resp.Tools))
	for _, t := range resp.Tools {
		result = append(result, mcpToolToOpenAI(t))
	}
	return result, nil
}

func mcpToolToOpenAI(t *mcp.Tool) openai.Tool {
	var params any = map[string]any{"type": "object", "properties": map[string]any{}}
	if t.InputSchema != nil {
		if raw, err := json.Marshal(t.InputSchema); err == nil {
			var m map[string]any
			if json.Unmarshal(raw, &m) == nil {
				params = m
			}
		}
	}
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  params,
		},
	}
}

func extractText(r *mcp.CallToolResult) string {
	parts := make([]string, 0, len(r.Content))
	for _, c := range r.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// ── Helpers ────────────────────────────────────────────────────────────────

func printSectionHeader(title string) {
	bar := strings.Repeat("─", len(title)+4)
	fmt.Printf("\n┌%s┐\n│  %s  │\n└%s┘\n\n", bar, title, bar)
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
