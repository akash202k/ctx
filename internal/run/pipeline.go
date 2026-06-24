package run

import (
	"encoding/json"
	"fmt"
	"time"

	"ctxengine/internal/graph"
	"ctxengine/internal/pack"
	"ctxengine/internal/parse"
	"ctxengine/internal/score"
	"ctxengine/internal/walker"
)

// Input is the §4 input contract.
type Input struct {
	Task        string           `json:"task"`
	EntryPoint  string           `json:"entry_point,omitempty"`
	TokenBudget int              `json:"token_budget"`
	RepoRoot    string           `json:"repo_root"`
	MaxDistance int              `json:"max_distance,omitempty"` // default 3
	Weights     *score.ScoreWeights `json:"weights,omitempty"`
	// SinceCommits caps how many git commits are scanned for co-change (0 = all).
	SinceCommits int `json:"since_commits,omitempty"`
}

// SelectedFileOutput is the §4 output shape for a selected file.
type SelectedFileOutput struct {
	Path    string   `json:"path"`
	Mode    string   `json:"mode"`
	Score   float64  `json:"score"`
	Reason  string   `json:"reason"`
	Tokens  int      `json:"tokens"`
	Symbols []string `json:"symbols,omitempty"`
	Content string   `json:"content,omitempty"`
}

// ExcludedFileOutput is the §4 output shape for a near-miss exclusion.
type ExcludedFileOutput struct {
	Path   string  `json:"path"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

// Output is the §4 output contract.
type Output struct {
	Task              string               `json:"task"`
	TokenBudget       int                  `json:"token_budget"`
	TokensUsed        int                  `json:"tokens_used"`
	Selected          []SelectedFileOutput `json:"selected"`
	ExcludedTopCandidates []ExcludedFileOutput `json:"excluded_top_candidates"`
	// Meta carries diagnostic info (build time, graph size, etc.)
	Meta Meta `json:"meta"`
}

// Meta holds diagnostic metadata about the run.
type Meta struct {
	GraphNodes    int    `json:"graph_nodes"`
	GraphEdges    int    `json:"graph_edges"`
	GraphBuildMs  int64  `json:"graph_build_ms"`
	TotalFiles    int    `json:"total_files"`
}

// Run executes the full pipeline: walk → build graph → score → pack.
// It is the single entry point called by CLI, HTTP server, and MCP server.
func Run(input Input) (Output, error) {
	if input.TokenBudget <= 0 {
		input.TokenBudget = 8000
	}
	if input.MaxDistance <= 0 {
		input.MaxDistance = 3
	}
	if input.SinceCommits <= 0 {
		input.SinceCommits = 500
	}
	weights := score.DefaultWeights()
	if input.Weights != nil {
		weights = *input.Weights
	}

	// ── Layer 0: file enumeration ────────────────────────────────────────────
	walkResult, err := walker.Walk(input.RepoRoot)
	if err != nil {
		return Output{}, fmt.Errorf("walk: %w", err)
	}

	// ── Layer 1: build dependency graph ─────────────────────────────────────
	graphStart := time.Now()
	parsers := []parse.Parser{parse.NewGoParser()}

	g, parsed, err := graph.Build(walkResult, parsers)
	if err != nil {
		return Output{}, fmt.Errorf("graph build: %w", err)
	}

	// Merge co-change edges into the graph.
	coEdges, history, coErr := graph.BuildCoChange(input.RepoRoot, input.SinceCommits)
	if coErr == nil {
		for _, e := range coEdges {
			g.MergeEdge(e)
		}
	}
	// Non-fatal: if git log fails (e.g. no git repo) we continue without co-change.

	graphMs := time.Since(graphStart).Milliseconds()

	// ── Layer 2: seed selection + relevance scoring ──────────────────────────
	seeds := score.FindSeeds(input.EntryPoint, input.Task, g, parsed)
	if len(seeds) == 0 {
		return Output{
			Task:        input.Task,
			TokenBudget: input.TokenBudget,
			Meta: Meta{
				GraphNodes:   len(g.Nodes),
				GraphEdges:   len(g.Edges),
				GraphBuildMs: graphMs,
				TotalFiles:   len(walkResult.Files),
			},
		}, nil
	}

	candidates := score.ScoreCandidates(g, seeds, input.MaxDistance, weights, history)

	// ── Layer 3: budget-aware packing ────────────────────────────────────────
	// Build content map for the packer.
	contentMap := make(pack.MapContentProvider, len(walkResult.Files))
	for _, f := range walkResult.Files {
		contentMap[f.RelPath] = f.Content
	}

	goParser := parse.NewGoParser()
	result := pack.Pack(candidates, input.TokenBudget, contentMap, pack.CharTokenizer, g, goParser, parsed)

	// ── Assemble output ───────────────────────────────────────────────────────
	out := Output{
		Task:        input.Task,
		TokenBudget: input.TokenBudget,
		TokensUsed:  result.TokensUsed,
		Meta: Meta{
			GraphNodes:   len(g.Nodes),
			GraphEdges:   len(g.Edges),
			GraphBuildMs: graphMs,
			TotalFiles:   len(walkResult.Files),
		},
	}

	for _, sf := range result.Selected {
		out.Selected = append(out.Selected, SelectedFileOutput{
			Path:    sf.Path,
			Mode:    string(sf.Mode),
			Score:   sf.Score,
			Reason:  sf.Reason,
			Tokens:  sf.Tokens,
			Symbols: sf.Symbols,
			Content: sf.Content,
		})
	}
	for _, ex := range result.Excluded {
		out.ExcludedTopCandidates = append(out.ExcludedTopCandidates, ExcludedFileOutput{
			Path:   ex.Path,
			Score:  ex.Score,
			Reason: ex.Reason,
		})
	}

	return out, nil
}

// MarshalOutput serialises an Output to indented JSON.
func MarshalOutput(o Output) ([]byte, error) {
	return json.MarshalIndent(o, "", "  ")
}
