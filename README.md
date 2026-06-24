# ctx — Context Engine

Selects the minimal set of files an engineer needs to read to do a task, ranked by relevance, each with a reason. Uses structural import/call graph + git co-change history. No embeddings or AI inside — pure graph algorithm.

## Build

```bash
go build -o ctxengine .
go build -o ctxengine-server ./cmd/ctxengine-server
go build -o ctxengine-mcp ./cmd/ctxengine-mcp
go build -o ctxeval ./cmd/ctxeval
```

## Use

**CLI (human output):**
```bash
./ctxengine -select -path /path/to/repo -task "fix retry bug" -entry internal/payments/retry.go
```

**CLI (JSON output):**
```bash
./ctxengine -select -path /path/to/repo -task "..." -no-content -json
```

**HTTP server:**
```bash
./ctxengine-server -port 8080
curl -X POST http://localhost:8080/select \
  -H "Content-Type: application/json" \
  -d '{"task":"fix retry bug","repo_root":"/path/to/repo","entry_point":"retry.go","token_budget":8000}'
```

**Cursor MCP (auto-called by Cursor agent):**
Add to `.cursor/mcp.json`:
```json
{ "mcpServers": { "context-engine": { "command": "/abs/path/to/ctxengine-mcp" } } }
```
Restart Cursor. The `select_context` tool is now available to the agent.

**Validation harness:**
```bash
./ctxeval -cases cmd/ctxeval/sample_cases.json
```
