package pack

import (
	"ctxengine/internal/graph"
	"ctxengine/internal/parse"
	"ctxengine/internal/score"
)

// Mode describes how much of a file is included in the selection.
type Mode string

const (
	ModeFull          Mode = "full"
	ModeSignatureOnly Mode = "signature_only"
)

// SelectedFile is a candidate that was included in the budget.
type SelectedFile struct {
	score.Candidate
	Mode    Mode     `json:"mode"`
	Symbols []string `json:"symbols,omitempty"` // only set for signature_only
	Tokens  int      `json:"tokens"`
	Content string   `json:"content,omitempty"` // full text; omitted for signature_only
}

// SelectionResult is the output of Pack.
type SelectionResult struct {
	TokensUsed int            `json:"tokens_used"`
	Selected   []SelectedFile `json:"selected"`
	Excluded   []score.Candidate `json:"excluded_top_candidates"`
}

// ContentProvider supplies full file content by repo-relative path.
// This allows Pack to remain generic — callers supply the content index.
type ContentProvider interface {
	Content(relPath string) string
}

// MapContentProvider implements ContentProvider from a plain map.
type MapContentProvider map[string]string

func (m MapContentProvider) Content(p string) string { return m[p] }

// Pack greedily selects candidates under the given token budget.
//
// Algorithm (per spec §7, Week 4):
//  1. Sort candidates by score descending (they should already be sorted).
//  2. Walk the list; add each as "full" while running total stays under budget.
//  3. Under budget pressure: if a file is graph-load-bearing (imported by an
//     already-selected file) downgrade to "signature_only" (extract exported
//     symbol signatures only, far cheaper in tokens) before dropping entirely.
//  4. Files that still don't fit → excluded_top_candidates.
//
// tokenizer is a pluggable function that estimates token count for a string.
// Pass CharTokenizer for a fast approximation, or wrap a real tokenizer.
//
// parser is used to extract exported signatures for signature_only downgrade.
// May be nil — in that case signature_only downgrade is skipped and files are
// either included in full or excluded.
func Pack(
	candidates []score.Candidate,
	budget int,
	content ContentProvider,
	tokenizer func(string) int,
	g *graph.Graph,
	parser parse.Parser,
	parsed graph.ParsedRepo,
) SelectionResult {
	if tokenizer == nil {
		tokenizer = CharTokenizer
	}

	selectedPaths := make(map[string]bool)
	var selected []SelectedFile
	var excluded []score.Candidate
	used := 0

	for _, c := range candidates {
		body := content.Content(c.Path)
		fullTokens := tokenizer(body)

		if used+fullTokens <= budget {
			// Fits in full.
			selected = append(selected, SelectedFile{
				Candidate: c,
				Mode:      ModeFull,
				Tokens:    fullTokens,
				Content:   body,
			})
			selectedPaths[c.Path] = true
			used += fullTokens
			continue
		}

		// Budget pressure — check if load-bearing (imported by something already selected).
		if parser != nil && isLoadBearing(c.Path, selectedPaths, g) {
			syms, err := parser.ExtractSignatures(c.Path, body)
			if err == nil && len(syms) > 0 {
				sigText := symbolsText(syms)
				sigTokens := tokenizer(sigText)
				if used+sigTokens <= budget {
					sigStrings := make([]string, len(syms))
					for i, s := range syms {
						sigStrings[i] = s.Signature
					}
					selected = append(selected, SelectedFile{
						Candidate: c,
						Mode:      ModeSignatureOnly,
						Tokens:    sigTokens,
						Symbols:   sigStrings,
					})
					selectedPaths[c.Path] = true
					used += sigTokens
					continue
				}
			}
		}

		excluded = append(excluded, c)
	}

	return SelectionResult{
		TokensUsed: used,
		Selected:   selected,
		Excluded:   excluded,
	}
}

// isLoadBearing returns true if relPath is imported (directly) by any
// already-selected file, making it structurally necessary context.
func isLoadBearing(relPath string, selectedPaths map[string]bool, g *graph.Graph) bool {
	for selPath := range selectedPaths {
		for _, nb := range g.Neighbors(selPath, "out", graph.EdgeImports) {
			if nb == relPath {
				return true
			}
		}
	}
	return false
}

func symbolsText(syms []parse.Symbol) string {
	var sb []byte
	for _, s := range syms {
		sb = append(sb, s.Signature...)
		sb = append(sb, '\n')
	}
	return string(sb)
}

// CharTokenizer estimates token count as len(s)/4, matching common LLM tokenizer
// averages for English/code text. Pluggable — replace with a real tokenizer.
func CharTokenizer(s string) int {
	n := len(s) / 4
	if n == 0 && len(s) > 0 {
		return 1
	}
	return n
}
