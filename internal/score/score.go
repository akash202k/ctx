package score

import (
	"fmt"
	"math"
	"sort"

	"ctxengine/internal/graph"
)

// ScoreWeights controls the relative importance of each scoring signal.
type ScoreWeights struct {
	GraphDistance float64 // weight for distance-based score (default 0.5)
	CoChange      float64 // weight for co-change frequency score (default 0.3)
	Recency       float64 // weight for file recency score (default 0.2)
}

// DefaultWeights returns the spec-specified defaults.
func DefaultWeights() ScoreWeights {
	return ScoreWeights{
		GraphDistance: 0.5,
		CoChange:      0.3,
		Recency:       0.2,
	}
}

// Candidate is a file node scored for relevance to the task.
type Candidate struct {
	Path   string  `json:"path"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
	// Component scores — retained for debugging and weight tuning.
	DistanceScore float64 `json:"distance_score"`
	CoChangeScore float64 `json:"co_change_score"`
	RecencyScore  float64 `json:"recency_score"`
}

// ScoreCandidates performs a bounded BFS from the given seeds over the graph,
// scores each reachable file node, and returns a sorted (desc) slice of candidates.
//
// The BFS follows EdgeImports, EdgeCalls (in both directions) and EdgeCoChange up
// to maxDistance hops. Seeds score 1.0 with their seed reason.
//
// Per-node score = w.GraphDistance*distanceScore + w.CoChange*coChangeScore + w.Recency*recencyScore
// where:
//   - distanceScore = 1 / (1 + hops)
//   - coChangeScore = min(1, totalCoChangeWeight / 10)
//   - recencyScore  = 1 / (1 + lastTouchDays / 30)
func ScoreCandidates(
	g *graph.Graph,
	seeds []Seed,
	maxDistance int,
	weights ScoreWeights,
	history map[string]graph.FileHistory,
) []Candidate {

	if maxDistance <= 0 {
		maxDistance = 3
	}

	// BFS state: node → minimum hop distance from any seed.
	dist := make(map[string]int)
	// BFS state: node → which seed(s) reached it and how.
	reachReason := make(map[string]string)
	queue := make([]string, 0, 64)

	for _, seed := range seeds {
		if _, exists := g.Nodes[seed.NodeID]; !exists {
			continue
		}
		dist[seed.NodeID] = 0
		reachReason[seed.NodeID] = seed.Reason
		queue = append(queue, seed.NodeID)
	}

	edgeKinds := []graph.EdgeKind{graph.EdgeImports, graph.EdgeCalls, graph.EdgeCoChange}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		curDist := dist[cur]
		if curDist >= maxDistance {
			continue
		}
		neighbors := g.Neighbors(cur, "both", edgeKinds...)
		for _, nb := range neighbors {
			if _, seen := dist[nb]; seen {
				continue
			}
			dist[nb] = curDist + 1
			reachReason[nb] = reachReasonFor(g, cur, nb, curDist+1, reachReason[cur])
			queue = append(queue, nb)
		}
	}

	// Aggregate co-change weights per file node (sum of co_change edge weights to/from seeds).
	coChangeWeight := make(map[string]float64)
	for _, seed := range seeds {
		for _, e := range g.Edges {
			if e.Kind != graph.EdgeCoChange {
				continue
			}
			if e.From == seed.NodeID {
				coChangeWeight[e.To] += e.Weight
			} else if e.To == seed.NodeID {
				coChangeWeight[e.From] += e.Weight
			}
		}
	}

	// Track which node IDs are seeds for special-casing.
	seedIDs := make(map[string]bool, len(seeds))
	for _, s := range seeds {
		seedIDs[s.NodeID] = true
	}

	// Build candidate list from all reached file nodes.
	var candidates []Candidate
	for nodeID, hops := range dist {
		node, ok := g.Nodes[nodeID]
		if !ok || node.Kind != graph.NodeFile {
			continue
		}

		// Seeds always score 1.0 (spec §7, Week 3: "Seeds score ~1.0").
		if seedIDs[nodeID] {
			candidates = append(candidates, Candidate{
				Path:          nodeID,
				Score:         1.0,
				Reason:        reachReason[nodeID],
				DistanceScore: 1.0,
				CoChangeScore: math.Min(1.0, coChangeWeight[nodeID]/10.0),
				RecencyScore:  func() float64 {
					if h, ok := history[nodeID]; ok {
						return 1.0 / (1.0 + h.LastTouchDays/30.0)
					}
					return 0
				}(),
			})
			continue
		}

		distScore := 1.0 / (1.0 + float64(hops))
		coScore := math.Min(1.0, coChangeWeight[nodeID]/10.0)

		recScore := 0.0
		if h, ok := history[nodeID]; ok {
			recScore = 1.0 / (1.0 + h.LastTouchDays/30.0)
		}

		finalScore := weights.GraphDistance*distScore +
			weights.CoChange*coScore +
			weights.Recency*recScore

		reason := dominantReason(distScore, coScore, recScore, weights, hops, reachReason[nodeID])

		candidates = append(candidates, Candidate{
			Path:          nodeID,
			Score:         math.Round(finalScore*1000) / 1000,
			Reason:        reason,
			DistanceScore: math.Round(distScore*1000) / 1000,
			CoChangeScore: math.Round(coScore*1000) / 1000,
			RecencyScore:  math.Round(recScore*1000) / 1000,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	return candidates
}

// reachReasonFor generates a reason string for a newly reached node.
func reachReasonFor(g *graph.Graph, from, to string, hops int, fromReason string) string {
	// Check which edge kind connected them.
	fromNode := g.Nodes[from]
	toNode := g.Nodes[to]
	edges := g.EdgesBetween(from, to)

	_ = fromNode
	_ = toNode

	for _, e := range edges {
		switch e.Kind {
		case graph.EdgeImports:
			if e.From == from {
				return fmt.Sprintf("imported by %s (distance %d)", from, hops)
			}
			return fmt.Sprintf("imports %s (distance %d)", from, hops)
		case graph.EdgeCalls:
			if e.From == from {
				return fmt.Sprintf("called from %s (distance %d)", from, hops)
			}
			return fmt.Sprintf("calls into %s (distance %d)", from, hops)
		case graph.EdgeCoChange:
			return fmt.Sprintf("frequently co-changed with %s (distance %d)", from, hops)
		}
	}
	return fmt.Sprintf("reachable from %s (distance %d)", from, hops)
}

// dominantReason explains the primary scoring signal for a candidate.
func dominantReason(dist, coChange, recency float64, w ScoreWeights, hops int, reachReason string) string {
	dContrib := w.GraphDistance * dist
	cContrib := w.CoChange * coChange
	rContrib := w.Recency * recency

	switch {
	case dContrib >= cContrib && dContrib >= rContrib:
		return reachReason
	case cContrib >= dContrib && cContrib >= rContrib:
		return "frequent co-change with seed file; " + reachReason
	default:
		return "recently modified; " + reachReason
	}
}
