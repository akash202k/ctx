package score

import (
	"path/filepath"
	"strings"
	"unicode"

	"ctxengine/internal/graph"
)

// Seed is a starting node for the graph walk, with its initial reason.
type Seed struct {
	NodeID string
	Reason string
}

// FindSeeds resolves the entry_point (if provided) or falls back to keyword-matching
// the task description against node IDs and symbol names in the graph.
// Returns at least one seed if anything can be found; empty slice means the graph
// walk will have nothing to start from (caller should warn the user).
func FindSeeds(entryPoint string, task string, g *graph.Graph, parsed graph.ParsedRepo) []Seed {
	if entryPoint != "" {
		return resolveEntryPoint(entryPoint, g)
	}
	return keywordSeeds(task, g, parsed)
}

// resolveEntryPoint tries to match the entry_point string to a node in the graph.
// Accepts: exact repo-relative path, path#Symbol notation, or a path suffix match.
func resolveEntryPoint(ep string, g *graph.Graph) []Seed {
	// Exact match (file or symbol node).
	if _, ok := g.Nodes[ep]; ok {
		return []Seed{{NodeID: ep, Reason: "entry point given by user"}}
	}

	// Case-insensitive suffix match on file paths.
	epLow := strings.ToLower(ep)
	var matches []Seed
	for id, node := range g.Nodes {
		if node.Kind != graph.NodeFile {
			continue
		}
		if strings.HasSuffix(strings.ToLower(id), epLow) {
			matches = append(matches, Seed{
				NodeID: id,
				Reason: "entry point given by user (matched " + id + ")",
			})
		}
	}
	if len(matches) > 0 {
		return matches
	}

	// Try treating it as a bare filename.
	base := strings.ToLower(filepath.Base(ep))
	for id, node := range g.Nodes {
		if node.Kind != graph.NodeFile {
			continue
		}
		if strings.ToLower(filepath.Base(id)) == base {
			matches = append(matches, Seed{
				NodeID: id,
				Reason: "entry point given by user (matched " + id + ")",
			})
		}
	}
	return matches
}

// keywordSeeds tokenises the task string and scores each file node by how many
// task tokens appear in the node's path or declared symbols. Returns the top
// matches as seeds (up to 5), each with a reason citing the matched keywords.
func keywordSeeds(task string, g *graph.Graph, parsed graph.ParsedRepo) []Seed {
	tokens := tokenise(task)
	if len(tokens) == 0 {
		return nil
	}

	var candidates []candidate

	for id, node := range g.Nodes {
		if node.Kind != graph.NodeFile {
			continue
		}
		idLow := strings.ToLower(id)
		var matched []string
		seen := make(map[string]bool)

		for _, tok := range tokens {
			if seen[tok] {
				continue
			}
			if strings.Contains(idLow, tok) {
				matched = append(matched, tok)
				seen[tok] = true
				continue
			}
			// Also check symbol names in the parsed file.
			if pf, ok := parsed[id]; ok {
				for _, sym := range pf.Symbols {
					if strings.Contains(strings.ToLower(sym.Name), tok) {
						matched = append(matched, tok)
						seen[tok] = true
						break
					}
				}
			}
		}

		if len(matched) > 0 {
			candidates = append(candidates, candidate{id: id, matched: matched})
		}
	}

	// Sort by number of matched tokens descending.
	sortByMatchCount(candidates)

	// Return top 5.
	limit := 5
	if len(candidates) < limit {
		limit = len(candidates)
	}
	seeds := make([]Seed, limit)
	for i := 0; i < limit; i++ {
		c := candidates[i]
		seeds[i] = Seed{
			NodeID: c.id,
			Reason: "matched task keyword(s): " + strings.Join(c.matched, ", "),
		}
	}
	return seeds
}

// tokenise splits a string into lowercase alpha-numeric tokens of length ≥ 3,
// filtering common English stop-words that add noise to keyword matching.
func tokenise(s string) []string {
	stop := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"not": true, "you": true, "all": true, "can": true, "her": true,
		"was": true, "one": true, "our": true, "out": true, "fix": true,
		"bug": true, "add": true, "get": true, "set": true, "use": true,
		"how": true, "why": true, "when": true, "that": true, "this": true,
		"with": true, "from": true, "they": true, "have": true, "into": true,
		"will": true, "your": true, "what": true,
	}

	words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	seen := make(map[string]bool)
	var out []string
	for _, w := range words {
		if len(w) < 3 || stop[w] || seen[w] {
			continue
		}
		seen[w] = true
		out = append(out, w)
	}
	return out
}

type candidate struct {
	id      string
	matched []string
}

func sortByMatchCount(cs []candidate) {
	for i := 1; i < len(cs); i++ {
		for j := i; j > 0 && len(cs[j].matched) > len(cs[j-1].matched); j-- {
			cs[j], cs[j-1] = cs[j-1], cs[j]
		}
	}
}
