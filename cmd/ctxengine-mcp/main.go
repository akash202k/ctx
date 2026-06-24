package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"ctxengine/internal/run"
)

func main() {
	s := server.NewMCPServer(
		"context-engine",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	tool := mcp.NewTool("select_context",
		mcp.WithDescription(
			"Given a task description and a repo, returns the minimal set of files "+
				"an engineer needs to read to do the task — ranked by relevance with "+
				"plain-English reasons. Uses structural import/call graph + git co-change "+
				"history. NO embedding/LLM inside; pure graph algorithm."),
		mcp.WithString("task",
			mcp.Required(),
			mcp.Description("The task or bug description, as you would type to a coding agent.")),
		mcp.WithString("repo_root",
			mcp.Required(),
			mcp.Description("Absolute path to the root of the repository to analyze.")),
		mcp.WithString("entry_point",
			mcp.Description("Optional: repo-relative file path or path#Symbol that anchors the graph walk. "+
				"Strongly recommended when available — dramatically improves precision.")),
		mcp.WithNumber("token_budget",
			mcp.DefaultNumber(8000),
			mcp.Description("Maximum total tokens across all selected file contents. Default: 8000.")),
		mcp.WithNumber("max_distance",
			mcp.DefaultNumber(3),
			mcp.Description("Maximum graph hops from the entry point. Default: 3.")),
	)

	s.AddTool(tool, handleSelectContext)

	log.SetFlags(0) // suppress timestamps in stderr for cleaner MCP output
	if err := server.ServeStdio(s); err != nil {
		log.Fatal(err)
	}
}

func handleSelectContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	task, err := req.RequireString("task")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: task"), nil
	}
	repoRoot, err := req.RequireString("repo_root")
	if err != nil {
		return mcp.NewToolResultError("missing required parameter: repo_root"), nil
	}

	entry, _ := req.RequireString("entry_point") // optional
	budget := int(req.GetFloat("token_budget", 8000))
	maxDist := int(req.GetFloat("max_distance", 3))

	inp := run.Input{
		Task:        task,
		EntryPoint:  entry,
		TokenBudget: budget,
		RepoRoot:    repoRoot,
		MaxDistance: maxDist,
	}

	output, err := run.Run(inp)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("pipeline error: %v", err)), nil
	}

	// Serialize as JSON — the agent reads this as structured text.
	b, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal error: " + err.Error()), nil
	}

	return mcp.NewToolResultText(string(b)), nil
}

