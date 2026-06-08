# Progressive Tool Discovery in MCP: Dynamic Implementation

**Using SkipperMCP as a reference implementation**

---

## The Problem: Context Bloat & Scalability

Every tool registered with an MCP server costs tokens. In a standard setup, 100 tools could consume 10-20% of an LLM's context window before the conversation even starts.

**Progressive Tool Discovery** solves this by loading tools only when they are needed. Instead of exposing 100 tools upfront, the server exposes only two "Meta-Tools":
1. `search_catalog`: Find tools based on intent.
2. `load_tool`: Activate a specific tool for the current session.

---

## How It Works (The Gateway Pattern)

SkipperMCP implements this pattern using the following architecture:

### 1. The Discovery Meta-Tools

The server starts with a minimal footprint, offering only discovery capabilities:

- **search_catalog(query)**: Searches an internal catalog of all available maritime tools.
- **load_tool(tool_name)**: Dynamically registers a tool on the server and signals the client to refresh its schemas.
- **unload_tool(tool_name)**: Removes a tool from the active context once a task is complete.

### 2. Enabling Dynamic Updates

The server must advertise the `listChanged` capability during initialization:

```go
server := mcp.NewServer(&mcp.Implementation{...}, &mcp.ServerOptions{
    Capabilities: &mcp.ServerCapabilities{
        Tools: &mcp.ToolCapabilities{
            ListChanged: true,
        },
    },
})
```

### 3. Signaling the Client

When `load_tool` is called, the server:
1. Adds the requested tool to the server's active tool list.
2. Emits a `notifications/tools/list_changed` notification.

The client, upon receiving this notification, re-fetches the tool list via `tools/list` and updates its local LLM tool definitions. This allows the model to "see" the new tool in the very next turn.

---

## Scenario Walkthrough: Dynamic Discovery

### Scenario: Man Overboard Emergency

1. **Initial State**: The Assistant sees only `search_catalog` and `load_tool`.
2. **Step 1 (Search)**:
   - *Assistant*: Calls `search_catalog(query="man overboard")`.
   - *Server*: Returns "Found **get_emergency_checklist**: Safety procedures for maritime emergencies."
3. **Step 2 (Load)**:
   - *Assistant*: Calls `load_tool(tool_name="get_emergency_checklist")`.
   - *Server*: Registers the tool and sends `list_changed` notification.
4. **Step 3 (Refresh)**:
   - *Client*: Re-fetches tools. The toolset now includes `get_emergency_checklist`.
5. **Step 4 (Execution)**:
   - *Assistant*: Calls `get_emergency_checklist(situation="Man Overboard")` and provides immediate life-saving instructions.

---

## Benefits

- **Minimal Token Overhead**: Only 2-3 tools are active initially.
- **Infinite Scalability**: You can have thousands of tools in the catalog without affecting performance.
- **Clean Context**: Use `unload_tool` to keep the model focused on the current task.

---

*SkipperMCP source code: [github.com/cmarcia/skipper-map](https://github.com/cmarcia/skipper-map)*
