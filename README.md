# SkipperMCP

A production-grade **Model Context Protocol (MCP)** server written in Go. SkipperMCP is a complete reference implementation of the **Progressive Tool Discovery** pattern — an architecture for building AI agents that manage large toolsets without context window bloat.

> **Full tutorial**: [Progressive Tool Discovery: A Complete Guide](./PROGRESSIVE_DISCOVERY.md)

---

## The Problem This Solves

A naive MCP server loads every tool's full JSON schema into the LLM's context window at startup. At 20 tools, that is 2,000–4,000 wasted tokens per turn. At 100+ tools, it consumes a significant portion of your context budget before any work begins.

**Progressive Tool Discovery** solves this in three layers:

| Layer | What happens | Token cost |
|-------|-------------|------------|
| 1. Search | LLM matches user intent to a tool name using short descriptions | ~50 tokens/tool |
| 2. Inspect | LLM fetches the full schema for the one tool it needs | ~200 tokens for that tool |
| 3. Execute | LLM calls the tool with validated parameters | 0 extra tokens |

The server author controls how well layer 1 works by writing precise, keyword-rich tool descriptions. This repo demonstrates exactly how to do that in Go.

---

## Tools

SkipperMCP exposes nine tools across two domains:

**Weather (live NWS API calls)**
- `get_forecast` — Detailed marine weather forecast for a GPS location
- `get_alerts` — Active storm warnings for a US state

**Skipper Companion (navigation & logistics)**
- `calculate_navigation` — Great-circle course bearing, distance, and ETA
- `book_harbor_slip` — Marina berthing/slip logistics
- `plan_fuel_consumption` — Voyage efficiency planning
- `check_system_maintenance` — Vessel health diagnostics (engine, electrical, plumbing)
- `get_tide_data` — Tide and current information
- `calculate_anchor_rode` — Anchoring safety calculations
- `get_emergency_checklist` — Step-by-step procedures for maritime emergencies (Fire, MOB, Sinking)

---

## Quick Start

### Prerequisites

- Go 1.22 or higher
- Claude Desktop (or any MCP-compatible client)

### Build

```bash
git clone https://github.com/cmarcia/skipper-map.git
cd skipper-map
go build -o skipper-mcp main.go
```

### Connect to Claude Desktop

Open `~/Library/Application Support/Claude/claude_desktop_config.json` and add:

```json
{
  "mcpServers": {
    "skipper": {
      "command": "/absolute/path/to/skipper-mcp"
    }
  }
}
```

Restart Claude Desktop. Try asking: *"What sailing tools do you have available?"* — and watch Progressive Discovery in action.

---

## Go LLM Client — Progressive Discovery Demo

`cmd/client/` is a standalone Go MCP **client** that connects to the server via subprocess stdio, wires it to OpenAI's function-calling API, and runs all three Progressive Discovery scenarios end-to-end.

### Setup

```bash
# 1. Build the MCP server (required at ./skipper-mcp)
go build -o skipper-mcp .

# 2. Create your .env file
cp .env.example .env
# Edit .env and set OPENAI_API_KEY=sk-...

# 3. Run the client
go run ./cmd/client/
```

### What the client demonstrates

| Scenario | Discovery layers | Key behaviour |
|----------|-----------------|---------------|
| **A — MAYDAY Man Overboard** | Layer 1 + 2 | OpenAI picks `get_emergency_checklist` from its description alone, then constructs `{"situation":"Man Overboard"}` from the jsonschema annotation |
| **B — Newport → Block Island** | Layer 1 + 2 | Multi-tool chain: `calculate_navigation` → `get_forecast` → `get_alerts`, each selected through description matching |
| **C — Capability Evolution** | Layer 3 | Starts with 3 core tools (no anchor tool); shows LLM unable to help; simulates `notifications/tools/list_changed`; client refreshes and succeeds with `calculate_anchor_rode` |

The client prints each `[Layer 1]` tool selection and `[Layer 2]` argument construction inline, making the discovery process observable.

---

## Project Structure

```
skipper/
├── main.go                  # MCP server entrypoint and tool registration
├── cmd/
│   └── client/
│       └── main.go          # Go MCP client with OpenAI integration (3 scenarios)
├── models/
│   └── models.go            # Shared input types with jsonschema annotations
├── handlers/
│   ├── weather.go           # NWS weather tool handlers
│   ├── weather_test.go
│   └── skipper.go           # Navigation and logistics tool handlers
├── nws/
│   ├── client.go            # HTTP client with connection pooling and retry logic
│   └── client_test.go
├── .env.example             # API key template (copy to .env)
└── PROGRESSIVE_DISCOVERY.md # Full architectural guide
```

---

## What Makes This a Good Reference

**Discriminating descriptions** — each tool description encodes *when* to use it, not just *what* it does. This is what makes layer 1 (intent matching) reliable:

```go
// Emergency tool — uppercase prefix signals urgency to the model
mcp.AddTool(server, &mcp.Tool{
    Name:        "get_emergency_checklist",
    Description: "CRITICAL SAFETY TOOL. Provides immediate step-by-step procedures for maritime emergencies like Fire, MOB, or Sinking.",
}, skipperHandler.GetEmergencyChecklist)
```

**Typed schemas with unit-annotated fields** — the `jsonschema` tags guide the model through layer 2 (parameter construction) without ambiguity:

```go
type AnchorRodeInput struct {
    DepthFeet  float64 `json:"depth_feet" jsonschema:"Water depth in feet"`
    WindSpeed  float64 `json:"wind_speed" jsonschema:"Expected wind speed in knots"`
    IsAllChain bool    `json:"is_all_chain" jsonschema:"Whether using all-chain rode (true) or rope/chain combo (false)"`
}
```

**Production HTTP client** — the NWS client (`nws/client.go`) uses connection pooling, configurable timeouts, and exponential backoff — not a toy `http.Get`.

---

## Testing

```bash
go test ./... -v
```

---

## Learn More

- [Progressive Tool Discovery: A Complete Guide](./PROGRESSIVE_DISCOVERY.md) — full architectural breakdown with annotated walkthroughs
- [MCP Specification](https://spec.modelcontextprotocol.io/) — the protocol this server implements
- [go-sdk](https://github.com/modelcontextprotocol/go-sdk) — the Go MCP SDK used in this project
- [NWS API](https://www.weather.gov/documentation/services-web-api) — the upstream weather API (no key required)
