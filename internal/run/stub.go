package run

import (
	"encoding/json"
	"fmt"
)

// Input is the input contract for context selection.
type Input struct {
	Task        string `json:"task"`
	EntryPoint  string `json:"entry_point,omitempty"`
	TokenBudget int    `json:"token_budget"`
	RepoRoot    string `json:"repo_root"`
	MaxDistance int    `json:"max_distance,omitempty"`
}

// SelectedFileOutput is the output shape for a selected file.
type SelectedFileOutput struct {
	Path    string   `json:"path"`
	Mode    string   `json:"mode"`
	Score   float64  `json:"score"`
	Reason  string   `json:"reason"`
	Tokens  int      `json:"tokens"`
	Symbols []string `json:"symbols,omitempty"`
	Content string   `json:"content,omitempty"`
}

// ExcludedFileOutput is the output shape for a near-miss exclusion.
type ExcludedFileOutput struct {
	Path   string  `json:"path"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

// Meta holds diagnostic metadata about the run.
type Meta struct {
	GraphNodes   int   `json:"graph_nodes"`
	GraphEdges   int   `json:"graph_edges"`
	GraphBuildMs int64 `json:"graph_build_ms"`
	TotalFiles   int   `json:"total_files"`
}

// Output is the output contract.
type Output struct {
	Task                  string                   `json:"task"`
	TokenBudget           int                      `json:"token_budget"`
	TokensUsed            int                      `json:"tokens_used"`
	Selected              []SelectedFileOutput     `json:"selected"`
	ExcludedTopCandidates []ExcludedFileOutput     `json:"excluded_top_candidates"`
	Meta                  Meta                     `json:"meta"`
}

// Run executes the context selection pipeline.
// This is a simplified stub - the full implementation would use the graph/score/pack pipeline.
func Run(input Input) (Output, error) {
	// For now, return an empty result with a note that the full graph engine
	// needs to be integrated from the existing ctxengine packages.
	return Output{
		Task:        input.Task,
		TokenBudget: input.TokenBudget,
		TokensUsed:  0,
		Selected:    []SelectedFileOutput{},
		ExcludedTopCandidates: []ExcludedFileOutput{},
		Meta: Meta{
			GraphNodes: 0,
			GraphEdges: 0,
			TotalFiles: 0,
		},
	}, fmt.Errorf("context selection not yet implemented - integrate graph/score/pack from existing ctxengine")
}

// MarshalOutput serializes an Output to indented JSON.
func MarshalOutput(o Output) ([]byte, error) {
	return json.MarshalIndent(o, "", "  ")
}
