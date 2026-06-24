package graph_test

import (
	"testing"

	"ctxengine/internal/graph"
)

func TestAddAndNeighbors(t *testing.T) {
	g := graph.New()

	g.AddNode(graph.Node{ID: "a.go", Kind: graph.NodeFile, Path: "a.go"})
	g.AddNode(graph.Node{ID: "b.go", Kind: graph.NodeFile, Path: "b.go"})
	g.AddNode(graph.Node{ID: "c.go", Kind: graph.NodeFile, Path: "c.go"})

	g.AddEdge(graph.Edge{From: "a.go", To: "b.go", Kind: graph.EdgeImports, Weight: 1.0})
	g.AddEdge(graph.Edge{From: "b.go", To: "c.go", Kind: graph.EdgeImports, Weight: 1.0})

	out := g.Neighbors("a.go", "out", graph.EdgeImports)
	if len(out) != 1 || out[0] != "b.go" {
		t.Fatalf("expected [b.go], got %v", out)
	}

	in := g.Neighbors("b.go", "in", graph.EdgeImports)
	if len(in) != 1 || in[0] != "a.go" {
		t.Fatalf("expected [a.go], got %v", in)
	}

	both := g.Neighbors("b.go", "both", graph.EdgeImports)
	if len(both) != 2 {
		t.Fatalf("expected 2 neighbors, got %v", both)
	}
}

func TestMergeEdge(t *testing.T) {
	g := graph.New()
	g.AddNode(graph.Node{ID: "x.go", Kind: graph.NodeFile, Path: "x.go"})
	g.AddNode(graph.Node{ID: "y.go", Kind: graph.NodeFile, Path: "y.go"})

	g.MergeEdge(graph.Edge{From: "x.go", To: "y.go", Kind: graph.EdgeCoChange, Weight: 1.0})
	g.MergeEdge(graph.Edge{From: "x.go", To: "y.go", Kind: graph.EdgeCoChange, Weight: 1.0})

	if len(g.Edges) != 1 {
		t.Fatalf("expected merged into 1 edge, got %d", len(g.Edges))
	}
	if g.Edges[0].Weight != 2.0 {
		t.Fatalf("expected weight 2.0, got %f", g.Edges[0].Weight)
	}
}
