package score_test

import (
	"testing"

	"ctxengine/internal/graph"
	"ctxengine/internal/score"
)

func buildTestGraph() *graph.Graph {
	g := graph.New()
	for _, id := range []string{"retry.go", "gateway.go", "config.go", "webhook.go"} {
		g.AddNode(graph.Node{ID: id, Kind: graph.NodeFile, Path: id})
	}
	g.AddEdge(graph.Edge{From: "retry.go", To: "gateway.go", Kind: graph.EdgeImports, Weight: 1})
	g.AddEdge(graph.Edge{From: "retry.go", To: "config.go", Kind: graph.EdgeImports, Weight: 1})
	g.AddEdge(graph.Edge{From: "gateway.go", To: "webhook.go", Kind: graph.EdgeImports, Weight: 1})
	// co-change between retry.go and gateway.go
	g.AddEdge(graph.Edge{From: "retry.go", To: "gateway.go", Kind: graph.EdgeCoChange, Weight: 5})
	g.AddEdge(graph.Edge{From: "gateway.go", To: "retry.go", Kind: graph.EdgeCoChange, Weight: 5})
	return g
}

func TestScoreCandidatesOrdering(t *testing.T) {
	g := buildTestGraph()
	seeds := []score.Seed{{NodeID: "retry.go", Reason: "entry point given by user"}}
	history := map[string]graph.FileHistory{
		"retry.go":   {LastTouchDays: 1},
		"gateway.go": {LastTouchDays: 2},
		"config.go":  {LastTouchDays: 30},
		"webhook.go": {LastTouchDays: 60},
	}

	candidates := score.ScoreCandidates(g, seeds, 3, score.DefaultWeights(), history)

	if len(candidates) == 0 {
		t.Fatal("expected candidates, got none")
	}
	// Seed must be first.
	if candidates[0].Path != "retry.go" {
		t.Errorf("expected retry.go first, got %s", candidates[0].Path)
	}
	// Scores must be descending.
	for i := 1; i < len(candidates); i++ {
		if candidates[i].Score > candidates[i-1].Score {
			t.Errorf("not sorted: candidates[%d].Score=%f > candidates[%d].Score=%f",
				i, candidates[i].Score, i-1, candidates[i-1].Score)
		}
	}
	// webhook.go is distance 2 and has no co-change with the seed, should score lower than gateway.go.
	var gatewayScore, webhookScore float64
	for _, c := range candidates {
		switch c.Path {
		case "gateway.go":
			gatewayScore = c.Score
		case "webhook.go":
			webhookScore = c.Score
		}
	}
	if gatewayScore <= webhookScore {
		t.Errorf("expected gateway.go (%f) > webhook.go (%f)", gatewayScore, webhookScore)
	}
}
