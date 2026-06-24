package graph

// NodeKind classifies what a node represents.
type NodeKind string

const (
	NodeFile     NodeKind = "file"
	NodeFunction NodeKind = "function"
	NodeType     NodeKind = "type"
)

// EdgeKind classifies the relationship between two nodes.
type EdgeKind string

const (
	EdgeImports  EdgeKind = "imports"
	EdgeCalls    EdgeKind = "calls"
	EdgeCoChange EdgeKind = "co_change"
)

// Node is a vertex in the dependency graph.
// ID for files: the repo-relative path (e.g. "internal/payments/retry.go").
// ID for symbols: "path#SymbolName" (e.g. "internal/payments/retry.go#DoRetry").
type Node struct {
	ID   string   `json:"id"`
	Kind NodeKind `json:"kind"`
	Path string   `json:"path"` // file this node lives in (same as ID for file nodes)
}

// Edge is a directed relationship between two nodes.
type Edge struct {
	From   string   `json:"from"`
	To     string   `json:"to"`
	Kind   EdgeKind `json:"kind"`
	Weight float64  `json:"weight"` // co_change uses frequency-based weight; structural edges use 1.0
}

// Graph holds the full in-memory dependency + co-change graph for a repo.
type Graph struct {
	Nodes map[string]Node `json:"nodes"`
	Edges []Edge          `json:"edges"`

	// adjacency index built lazily by indexEdges; rebuilt on demand after mutations
	outIdx map[string][]int // node ID → indices into Edges (outbound)
	inIdx  map[string][]int // node ID → indices into Edges (inbound)
}

// New returns an empty Graph.
func New() *Graph {
	return &Graph{
		Nodes: make(map[string]Node),
	}
}

// AddNode inserts or replaces a node.
func (g *Graph) AddNode(n Node) {
	g.Nodes[n.ID] = n
	g.outIdx = nil // invalidate adjacency index
	g.inIdx = nil
}

// AddEdge appends an edge. Duplicate edges are not deduplicated here — callers
// should avoid adding the same edge twice (build.go merges co-change weights).
func (g *Graph) AddEdge(e Edge) {
	g.Edges = append(g.Edges, e)
	g.outIdx = nil
	g.inIdx = nil
}

// MergeEdge adds an edge if it does not yet exist (matched by From+To+Kind),
// otherwise adds delta to the existing edge's Weight.
func (g *Graph) MergeEdge(e Edge) {
	for i, existing := range g.Edges {
		if existing.From == e.From && existing.To == e.To && existing.Kind == e.Kind {
			g.Edges[i].Weight += e.Weight
			return
		}
	}
	g.AddEdge(e)
}

func (g *Graph) buildIndex() {
	if g.outIdx != nil {
		return
	}
	g.outIdx = make(map[string][]int, len(g.Nodes))
	g.inIdx = make(map[string][]int, len(g.Nodes))
	for i, e := range g.Edges {
		g.outIdx[e.From] = append(g.outIdx[e.From], i)
		g.inIdx[e.To] = append(g.inIdx[e.To], i)
	}
}

// Neighbors returns node IDs reachable from nodeID over edges of the given kinds.
// If direction is "out" it follows From→To; "in" follows To→From; "both" follows both.
// If no kinds are provided, all edge kinds are included.
func (g *Graph) Neighbors(nodeID string, direction string, kinds ...EdgeKind) []string {
	g.buildIndex()

	kindSet := make(map[EdgeKind]bool, len(kinds))
	for _, k := range kinds {
		kindSet[k] = true
	}
	matchKind := func(k EdgeKind) bool {
		return len(kinds) == 0 || kindSet[k]
	}

	seen := make(map[string]bool)
	var result []string

	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	if direction == "out" || direction == "both" {
		for _, idx := range g.outIdx[nodeID] {
			e := g.Edges[idx]
			if matchKind(e.Kind) {
				add(e.To)
			}
		}
	}
	if direction == "in" || direction == "both" {
		for _, idx := range g.inIdx[nodeID] {
			e := g.Edges[idx]
			if matchKind(e.Kind) {
				add(e.From)
			}
		}
	}
	return result
}

// EdgesBetween returns all edges between two nodes in either direction.
func (g *Graph) EdgesBetween(a, b string, kinds ...EdgeKind) []Edge {
	kindSet := make(map[EdgeKind]bool, len(kinds))
	for _, k := range kinds {
		kindSet[k] = true
	}
	matchKind := func(k EdgeKind) bool {
		return len(kinds) == 0 || kindSet[k]
	}

	var result []Edge
	for _, e := range g.Edges {
		if !matchKind(e.Kind) {
			continue
		}
		if (e.From == a && e.To == b) || (e.From == b && e.To == a) {
			result = append(result, e)
		}
	}
	return result
}
