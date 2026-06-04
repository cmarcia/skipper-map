Most MCP servers have a subtle performance problem — and almost nobody talks about it.

When an LLM client connects to a server with many tools, it loads every tool's full JSON schema into the context window at startup. All of them. Every parameter, every description, every constraint. Even for tools the model will never call.

I built SkipperMCP — a nautical boat management agent in Go — specifically to demonstrate a pattern that fixes this: **Progressive Tool Discovery**.

The idea is simple. Instead of dumping everything upfront, you give the LLM three layers of access:

1. **Search** — tool names + one-line descriptions only (cheap, minimal tokens)
2. **Inspect** — full schema fetched on demand, only for the tool it actually needs
3. **Execute** — the call itself, with validated parameters

The result: dramatically less context bloat, lower latency, and a less confused model.

Here's what it looks like in practice. The server author controls discoverability through description quality:

```go
mcp.AddTool(server, &mcp.Tool{
    Name:        "get_emergency_checklist",
    Description: "CRITICAL SAFETY TOOL. Provides immediate step-by-step procedures for maritime emergencies like Fire, MOB, or Sinking.",
}, skipperHandler.GetEmergencyChecklist)
```

That description is doing real work. Keyword-rich, intent-clear, urgency-signaled. The LLM finds it when it needs it — and ignores it when it doesn't.

SkipperMCP ships with 9 tools covering forecasts, navigation, tides, anchoring, fuel planning, and emergency procedures. The GitHub repo includes a full technical walkthrough in `PROGRESSIVE_DISCOVERY.md` that breaks down the pattern in detail — with architecture diagrams, annotated code, and end-to-end scenarios.

If you're building MCP servers — or thinking about context window efficiency more broadly — I think this pattern is worth understanding.

Check it out here: [GitHub Repo Link]

---

#MCP #ModelContextProtocol #GoLang #LLMEngineering #AIArchitecture #OpenSource #SoftwareEngineering #ContextWindow #AIAgents #DeveloperTools
