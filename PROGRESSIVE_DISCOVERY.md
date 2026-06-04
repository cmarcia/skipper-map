# Progressive Tool Discovery in MCP: A Complete Guide

**Using SkipperMCP as a reference implementation**

---

## The Problem: Context Bloat

Every tool registered with an MCP server costs tokens. Not just the name — the full schema: description, parameter names, types, constraints, and annotations. At scale, this matters.

A server with 20 tools and moderately detailed schemas can consume 2,000–4,000 tokens before the user's first message is even processed. At 200+ tools — the scale of a real enterprise integration — you are spending 20,000–40,000 tokens per turn on tool overhead alone. With a 200k-token context window, that is 10–20% of your budget gone before any work begins.

The naive fix is to reduce the number of tools or truncate descriptions. Both are wrong. Fewer tools means less capability. Shorter descriptions means the model cannot distinguish tools from each other and will misuse them or hallucinate incorrect parameters.

**Progressive Tool Discovery** is the correct fix. It is an architectural pattern — not a feature flag — that lets the model discover and use tools in layers: broad intent recognition first, precise parameter construction second, optional capability expansion third.

SkipperMCP, a Go MCP server for maritime navigation assistance, implements this pattern across nine tools spanning weather, navigation, safety, and logistics. It is small enough to read end-to-end and realistic enough to illustrate real design decisions.

---

## What Is Progressive Tool Discovery?

Progressive Tool Discovery is the practice of structuring your MCP server so that an LLM client can:

1. **Identify the right tool** from a large set using only tool names and descriptions
2. **Construct correct invocations** using typed schemas with rich metadata
3. **Expand capability** dynamically as the conversation context changes

These map to three distinct layers:

### Layer 1: Intent Recognition

The client has access to the full tool list. It uses tool names and descriptions to match user intent to a candidate tool. No parameters are constructed yet. This is purely semantic matching.

### Layer 2: Schema-Guided Invocation

Once a tool is selected, the client examines the tool's JSON Schema to construct the call. Typed fields, required constraints, and `jsonschema` annotations guide the model toward valid input without requiring the user to know the tool's interface.

### Layer 3: Capability Evolution

As the conversation progresses, the server may expose new tools or retire old ones. The client re-evaluates its tool knowledge and discovers newly available capabilities. This is signaled by the `notifications/tools/list_changed` notification.

### The Architecture

```
User Message
     |
     v
+--------------------+
|   LLM Client       |
|  (Claude Desktop)  |
|                    |
|  1. tools/list     |<--------+
|     (all tools,    |         |
|      descriptions) |         |
|                    |    notifications/
|  2. Match intent   |    tools/list_changed
|     to tool name   |         |
|                    |         |
|  3. Inspect schema |         |
|     Build params   |         |
|                    |         |
|  4. tools/call     |         |
+--------------------+         |
         |                     |
         v                     |
+--------------------+         |
|   MCP Server       |---------+
|  (SkipperMCP)      |
|                    |
|  get_forecast      |
|  get_alerts        |
|  calculate_nav..   |
|  book_harbor_slip  |
|  plan_fuel_cons..  |
|  check_system_m..  |
|  get_tide_data     |
|  calculate_anch..  |
|  get_emergency_..  |
+--------------------+
         |
         v
+--------------------+
|  External APIs     |
|  (NWS, NOAA, ...)  |
+--------------------+
```

The client never hard-codes tool names. It discovers them. The server never assumes what the client knows. It declares everything through schema.

---

## Part 1: Designing for Discovery (Server Side)

The quality of discovery is determined almost entirely by what the server declares. Description quality is not a documentation concern — it is a correctness concern. A model that cannot distinguish `get_forecast` from `get_tide_data` will call the wrong one.

### Writing Descriptions That Enable Disambiguation

Every tool description in SkipperMCP encodes two things: **when to use it** and **what makes it different from similar tools**.

```go
// main.go:27
mcp.AddTool(server, &mcp.Tool{
    Name:        "get_forecast",
    Description: "Essential for pre-sail checks. Retrieves detailed marine-relevant weather forecasts for a specific GPS location.",
}, weatherHandler.GetForecast)

// main.go:33
mcp.AddTool(server, &mcp.Tool{
    Name:        "get_alerts",
    Description: "Safety critical. Monitors active weather alerts and storm warnings for a US state/area.",
}, weatherHandler.GetAlerts)
```

Both tools deal with weather. Without the descriptions, a model might call either one interchangeably. The descriptions establish the distinction:

- `get_forecast`: GPS coordinates, returns marine weather, use before departure
- `get_alerts`: State-level, returns storm warnings, use for safety monitoring

The leading phrases — "Essential for pre-sail checks" and "Safety critical" — are not marketing. They encode priority and context. When a user asks "is it safe to leave the harbor?", the model now has enough signal to reach for `get_alerts` first, not `get_forecast`.

The safety-critical tools push this further:

```go
// main.go:72
mcp.AddTool(server, &mcp.Tool{
    Name:        "get_emergency_checklist",
    Description: "CRITICAL SAFETY TOOL. Provides immediate step-by-step procedures for maritime emergencies like Fire, MOB, or Sinking.",
}, skipperHandler.GetEmergencyChecklist)
```

The uppercase `CRITICAL SAFETY TOOL` is deliberate. LLMs are not immune to emphasis. A model reasoning about a MAYDAY scenario should not have to infer priority from the tool name alone — the description makes it unambiguous.

### Tool Naming Conventions

SkipperMCP follows a consistent `verb_noun` pattern: `get_forecast`, `calculate_navigation`, `book_harbor_slip`, `plan_fuel_consumption`, `check_system_maintenance`. This is not cosmetic.

Consistent naming allows the model to:

- **Group by verb**: All `get_*` tools are read-only data retrieval. All `calculate_*` tools are computational. `book_*` implies a write/side-effect operation.
- **Predict existence**: If `get_forecast` exists, the model can reason that `get_alerts` probably exists too, even before seeing the full list.
- **Reduce ambiguity**: `plan_fuel_consumption` vs `calculate_navigation` — different verbs signal different roles even when both involve math.

Avoid synonyms for the same verb class. Do not mix `get_`, `fetch_`, `retrieve_`, and `query_` for conceptually identical operations. Pick one and enforce it.

### Typed Schemas with jsonschema Annotations

Schema quality determines whether Layer 2 (invocation construction) succeeds or fails. The `models/models.go` file shows the pattern:

```go
// models/models.go:9
type NavigationInput struct {
    StartLat  float64 `json:"start_lat" jsonschema:"Starting latitude"`
    StartLon  float64 `json:"start_lon" jsonschema:"Starting longitude"`
    EndLat    float64 `json:"end_lat" jsonschema:"Destination latitude"`
    EndLon    float64 `json:"end_lon" jsonschema:"Destination longitude"`
    BoatSpeed float64 `json:"boat_speed" jsonschema:"Speed in knots"`
}
```

Every field carries:
- A `json` tag that defines the wire name the model must use when constructing the call
- A `jsonschema` annotation that describes the field's semantic meaning and expected unit

The `jsonschema` annotation on `BoatSpeed` says "Speed in knots" — not just "speed". This matters. A model asked "how long will it take to sail from Newport to Block Island at 6 knots?" now knows not to convert to miles-per-hour before calling the tool.

Consider the anchor rode calculator:

```go
// models/models.go:45
type AnchorRodeInput struct {
    DepthFeet  float64 `json:"depth_feet" jsonschema:"Water depth in feet"`
    WindSpeed  float64 `json:"wind_speed" jsonschema:"Expected wind speed in knots"`
    IsAllChain bool    `json:"is_all_chain" jsonschema:"Whether using all-chain rode (true) or rope/chain combo (false)"`
}
```

The boolean `IsAllChain` with its annotation eliminates an entire class of ambiguity. Without the annotation, a model might pass `true` when the user says "I have a rope and chain setup" — because "I have a chain" sounds like `is_all_chain: true`. The annotation makes the boolean semantics explicit.

**Naming fields with units in the name** (`depth_feet`, `wind_speed`, `distance_nm`) is a defensive practice. When the field name encodes the unit, a mismatch between the model's assumption and the API's expectation is harder to introduce silently.

---

## Part 2: How Clients Use Discovery

Understanding what the client does with your declarations is essential to designing them well. The following describes how MCP-compatible clients like Claude Desktop perform discovery in practice.

### Phase 1: Initial Enumeration (tools/list)

When the MCP session is established, the client calls `tools/list`. The server returns the complete list of tool objects — names, descriptions, and full JSON schemas. The client caches this.

For SkipperMCP, this means the client receives all nine tools upfront. The client does not call any of them yet. It reads and indexes the descriptions.

At this point, a query like "what's the weather forecast for 41.3°N, 71.5°W?" maps cleanly to `get_forecast` because:
- The description says "GPS location"
- The input schema has `latitude` and `longitude` fields
- No other tool has GPS coordinates in its schema

### Phase 2: Semantic Matching and Schema Inspection

When the user's message arrives, the client does not scan the full schema of every tool. It uses the descriptions — much shorter — to narrow candidates. Once a candidate is selected, it inspects that tool's schema to construct the call.

This is why description quality matters for Layer 1 and schema quality matters for Layer 2. They serve different phases of the same discovery process.

For the query above, the client:
1. Reads descriptions, identifies `get_forecast` as the match
2. Inspects `ForecastInput`: `latitude float64`, `longitude float64`
3. Extracts 41.3 and -71.5 from the user message
4. Constructs and sends the `tools/call`

### Phase 3: Disambiguation Under Ambiguity

When multiple tools could plausibly match, the client uses description content to differentiate. Consider the user asking: "are there any storm warnings near Connecticut?"

Both `get_forecast` and `get_alerts` deal with weather. The client must choose. The descriptions encode the distinction:

- `get_forecast`: "specific GPS location"
- `get_alerts`: "US state/area", "storm warnings"

"Storm warnings" appears only in `get_alerts`. The state-level scope ("Connecticut") matches `get_alerts`'s `State` parameter (`json:"state"`, annotated as "Two-letter US state code"). The model constructs `{"state": "CT"}` and calls `get_alerts`.

Without the description content, this disambiguation requires either more conversation turns (asking the user clarifying questions) or a wrong tool call.

---

## Part 3: Complete SkipperMCP Walkthrough

### Scenario A: MAYDAY — Man Overboard

**User message**: "MAN OVERBOARD — we need the MOB procedure RIGHT NOW"

**Layer 1 — Intent Recognition:**

The client scans tool descriptions. The phrase "MOB" appears nowhere, but "Man Overboard" is explicitly called out in `get_emergency_checklist`:

```
"CRITICAL SAFETY TOOL. Provides immediate step-by-step procedures for maritime emergencies like Fire, MOB, or Sinking."
```

The uppercase prefix signals urgency. The model does not evaluate any other tool. `get_emergency_checklist` is the match.

**Layer 2 — Schema-Guided Invocation:**

```go
// models/models.go:51
type EmergencyInput struct {
    Situation string `json:"situation" jsonschema:"Emergency type (e.g., Man Overboard, Fire, Engine Failure, Taking Water)"`
}
```

The `jsonschema` annotation provides enumeration hints. The model maps "MAN OVERBOARD" to the canonical value "Man Overboard" from the examples. The call is constructed as `{"situation": "Man Overboard"}`.

**Layer 3 — Capability Evolution:**

After delivering the MOB checklist, the server might surface additional context — tide information for search and rescue calculations, weather alerts for the current area. If the server supports `notifications/tools/list_changed`, it can push updated tool visibility at this point (see Part 4). Even without dynamic changes, the client can now propose follow-up calls to `get_forecast` or `get_tide_data` based on the conversation state.

**Total discovery steps: 2 (description match, schema annotation match). Zero clarifying questions needed.**

---

### Scenario B: Passage Planning — Newport to Block Island

**User message**: "I'm leaving Newport (41.49°N, 71.31°W) heading to Block Island (41.17°N, 71.58°W) at 7 knots. How long will it take and should I check the weather first?"

This is a compound intent. The user is asking for two things: a navigation calculation and a weather recommendation. The client must call multiple tools.

**Step 1 — Navigation Calculation:**

The client matches the first part of the query to `calculate_navigation`:

```
"Computes course bearing, distance, and estimated voyage time between two nautical coordinates."
```

Schema inspection reveals `NavigationInput`:

```go
// models/models.go:9
type NavigationInput struct {
    StartLat  float64 `json:"start_lat" jsonschema:"Starting latitude"`
    StartLon  float64 `json:"start_lon" jsonschema:"Starting longitude"`
    EndLat    float64 `json:"end_lat" jsonschema:"Destination latitude"`
    EndLon    float64 `json:"end_lon" jsonschema:"Destination longitude"`
    BoatSpeed float64 `json:"boat_speed" jsonschema:"Speed in knots"`
}
```

The annotations resolve every ambiguity: latitude vs longitude ordering, start vs end, and the unit for speed. The call is:

```json
{
  "start_lat": 41.49,
  "start_lon": -71.31,
  "end_lat": 41.17,
  "end_lon": -71.58,
  "boat_speed": 7.0
}
```

The handler computes the result using the haversine formula:

```go
// handlers/skipper.go
dist := haversine(input.StartLat, input.StartLon, input.EndLat, input.EndLon)
time := dist / input.BoatSpeed
result := fmt.Sprintf("Course Analysis:\nDistance: %.2f nautical miles\nEstimated Time at En-route: %.1f hours\nInitial Bearing: %.0f°",
    dist, time, calculateBearing(input.StartLat, input.StartLon, input.EndLat, input.EndLon))
```

**Step 2 — Weather Check:**

The user asked "should I check the weather first?" — this is a recommendation request, not a direct command. But the model, having just established a departure location, now has enough context to call `get_forecast` proactively. The GPS coordinates from Step 1 can be reused directly as `latitude`/`longitude` for the forecast call.

Simultaneously or sequentially, the client may call `get_alerts` for the Rhode Island area (`{"state": "RI"}`), since `get_alerts` is described as "Safety critical" and covers "storm warnings" — exactly the kind of pre-departure check the user implied.

**Step 3 — Optional Tide Context:**

The route passes through a harbor entrance. The client, having now established a destination at Block Island, can propose `get_tide_data`:

```
"Retrieves localized tide and current information. Critical for navigating narrow channels and harbor entrances."
```

"Harbor entrances" is the trigger phrase. The description provides the semantic link between the navigation context (going to Block Island) and this tool's utility.

**Total tools called: 2 mandatory (navigation, forecast), 1 contextually relevant (alerts), 1 optionally suggested (tides). All selected through description matching and schema-guided invocation — no hard-coding, no brittle conditional logic.**

---

## Part 4: Handling Dynamic Toolsets

Not all MCP servers have a static tool list. A server might expose different tools based on:

- User authentication level (read vs read-write access)
- Subscription tier (basic vs premium features)
- Conversation state (onboarding mode vs expert mode)
- External system availability (tools hidden when upstream APIs are down)

The MCP specification provides `notifications/tools/list_changed` for exactly this case. When the server's tool set changes, it sends this notification to all connected clients. Clients that receive it should re-issue `tools/list` to refresh their cached tool knowledge before the next turn.

For SkipperMCP, a real-world extension might work as follows:

- On session start, expose only the six core tools (weather, navigation, anchoring)
- After the user completes a harbor booking, unlock `check_system_maintenance` and `plan_fuel_consumption` as contextually relevant next steps
- If a MAYDAY situation is detected, immediately push `get_emergency_checklist` to the top of the tool list

The notification pattern keeps the client's view of available tools current without requiring it to poll. The server drives capability disclosure based on state it owns; the client remains stateless about which tools exist.

**Design principle**: The notification should be sent eagerly — before the client needs the new tools — not lazily after the client has already attempted and failed a call for a non-existent tool.

---

## Setup Guide

### Prerequisites

- Go 1.21 or later
- Claude Desktop (or any MCP-compatible client)
- Access to the NWS API (public, no API key required)

### Build

```bash
git clone https://github.com/charliemarciano/skipper-mcp
cd skipper-mcp
go build -o skipper-mcp ./...
```

### Claude Desktop Configuration

Add the following to your Claude Desktop MCP configuration file. On macOS, this is located at `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "skipper-mcp": {
      "command": "/absolute/path/to/skipper-mcp",
      "args": []
    }
  }
}
```

Replace `/absolute/path/to/skipper-mcp` with the full path to the compiled binary.

Restart Claude Desktop. Open a new conversation and ask: "What sailing tools do you have available?" — the model will enumerate the registered tools using its cached `tools/list` response.

### Verify Tool Registration

At startup, the server logs:

```
Starting Skipper Companion MCP Server...
```

No tool-level startup logging is included in the current implementation. If you add verbose logging, emit tool names and descriptions at registration time — this makes debugging schema issues significantly easier.

---

## Best Practices

**1. Encode priority and context in descriptions, not just function.**
"Safety critical" and "CRITICAL SAFETY TOOL" are not emphasis for aesthetics — they influence which tool the model selects under ambiguity. Write descriptions as if you are briefing a competent but unfamiliar engineer.

**2. Use consistent verb prefixes across all tools.**
`get_`, `calculate_`, `book_`, `plan_`, `check_` each carry semantic weight. Mixing synonyms (`fetch_` vs `get_`, `compute_` vs `calculate_`) forces the model to treat conceptually equivalent operations as unrelated.

**3. Include units in both field names and jsonschema annotations.**
`depth_feet`, `wind_speed`, `distance_nm` reduce unit conversion errors. The annotation provides semantic context; the field name provides a mnemonic safety net.

**4. Name boolean fields to make `true` unambiguous.**
`IsAllChain bool` with the annotation "Whether using all-chain rode (true) or rope/chain combo (false)" is unambiguous. A field named `chain_type string` with values `"all-chain"` or `"combo"` is equally clear but requires the model to know the exact string values. Type-narrowing boolean fields with explicit annotations is often the cleaner choice.

**5. Call out explicit examples in descriptions for enum-like fields.**
`EmergencyInput.Situation` has no Go enum type — it is a free string. The `jsonschema` annotation provides examples: "Man Overboard, Fire, Engine Failure, Taking Water". These examples function as implicit enumerations. The model selects from them rather than inventing free-form values.

**6. Send `notifications/tools/list_changed` eagerly.**
If your server has dynamic tools, notify clients before they need the new capability. Reactively adding tools only after a failed call introduces a turn of latency and potentially a confusing error response.

**7. Group related tools under a common naming prefix when the set is large.**
SkipperMCP is small enough that the verb-based grouping suffices. At 20+ tools, a domain prefix (`weather_get_forecast`, `nav_calculate_route`) provides a second layer of disambiguation and makes `tools/list` responses easier to reason about as a set.

**8. Test your schemas by simulating the model's perspective.**
Given only the tool name and description, can you identify the correct tool for each query type in your domain? Given only the schema, can you construct a valid call with no documentation? If either answer is no, the descriptions or annotations need work.

---

## References

- [Model Context Protocol Specification](https://spec.modelcontextprotocol.io/)
- [go-sdk for MCP](https://github.com/modelcontextprotocol/go-sdk) — the Go SDK used in SkipperMCP
- [National Weather Service API](https://www.weather.gov/documentation/services-web-api) — the upstream API behind `get_forecast` and `get_alerts`
- [NOAA Tides and Currents](https://tidesandcurrents.noaa.gov/api/) — the data source for `get_tide_data`
- [JSON Schema Specification](https://json-schema.org/specification) — the schema standard underlying MCP tool definitions

---

*SkipperMCP source code: [github.com/charliemarciano/skipper-mcp](https://github.com/charliemarciano/skipper-mcp)*
