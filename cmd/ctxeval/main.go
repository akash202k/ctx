package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"ctxengine/internal/run"
)

// EvalCase is a single test case: a task + the ground-truth set of files that matter.
type EvalCase struct {
	Name       string   `json:"name"`
	Task       string   `json:"task"`
	EntryPoint string   `json:"entry_point,omitempty"`
	RepoRoot   string   `json:"repo_root"`
	GroundTruth []string `json:"ground_truth"` // repo-relative paths that were actually touched
}

// EvalResult is the result of one evaluation.
type EvalResult struct {
	EvalCase
	Precision     float64  `json:"precision"`
	Recall        float64  `json:"recall"`
	F1            float64  `json:"f1"`
	Selected      []string `json:"selected"`
	TruePositives []string `json:"true_positives"`
	FalsePositives []string `json:"false_positives"`
	FalseNegatives []string `json:"false_negatives"`
}

// BaselineResult is a simple keyword baseline: include any file whose path
// contains a token from the task description.
type BaselineResult struct {
	Precision float64
	Recall    float64
	F1        float64
}

func main() {
	casesFile := flag.String("cases", "", "path to JSON file containing []EvalCase")
	topN := flag.Int("top-n", 10, "limit selection to top N files for evaluation")
	budget := flag.Int("budget", 100000, "token budget (set high to avoid packing bias in eval)")
	flag.Parse()

	if *casesFile == "" {
		fmt.Fprintln(os.Stderr, "usage: ctxeval -cases cases.json [-top-n 10] [-budget 100000]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "cases.json format: []EvalCase — see cmd/ctxeval/README or the struct definition.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "To bootstrap: generate a cases.json from git PRs using:")
		fmt.Fprintln(os.Stderr, "  git log --merges --pretty='%H' | head -20    # find merged PR commits")
		fmt.Fprintln(os.Stderr, "  git diff --name-only HEAD~1 HEAD               # files touched in a commit")
		os.Exit(1)
	}

	data, err := os.ReadFile(*casesFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading cases file:", err)
		os.Exit(1)
	}
	var cases []EvalCase
	if err := json.Unmarshal(data, &cases); err != nil {
		fmt.Fprintln(os.Stderr, "error parsing cases file:", err)
		os.Exit(1)
	}

	fmt.Printf("Evaluating %d cases (top-%d selection, budget %d)...\n\n", len(cases), *topN, *budget)

	var results []EvalResult
	var baselineResults []BaselineResult

	for _, c := range cases {
		result := evalCase(c, *budget, *topN)
		baseline := baselineEval(c, *topN)
		results = append(results, result)
		baselineResults = append(baselineResults, baseline)
	}

	printResults(results, baselineResults)
}

func evalCase(c EvalCase, budget, topN int) EvalResult {
	inp := run.Input{
		Task:        c.Task,
		EntryPoint:  c.EntryPoint,
		TokenBudget: budget,
		RepoRoot:    c.RepoRoot,
	}

	out, err := run.Run(inp)
	if err != nil {
		return EvalResult{EvalCase: c}
	}

	// Collect selected paths (top N).
	selected := make([]string, 0, len(out.Selected))
	for _, s := range out.Selected {
		selected = append(selected, s.Path)
		if len(selected) >= topN {
			break
		}
	}

	return computeMetrics(c, selected)
}

func baselineEval(c EvalCase, topN int) BaselineResult {
	// Keyword baseline: include any file whose path contains a task token.
	tokens := tokenise(c.Task)

	// Walk the repo to get all files — use a simple git ls-files call.
	var allFiles []string
	// We'd ideally call walker.Walk but that requires a full walk — just use the
	// ground truth and the task to simulate the baseline for the eval subset.
	// For a proper baseline, all files in the repo need to be scored.
	// Here we score ground-truth files only (lower bound on baseline quality).
	for _, f := range c.GroundTruth {
		allFiles = append(allFiles, f)
	}

	var selected []string
	for _, f := range allFiles {
		fLow := strings.ToLower(f)
		for _, tok := range tokens {
			if strings.Contains(fLow, tok) {
				selected = append(selected, f)
				break
			}
		}
		if len(selected) >= topN {
			break
		}
	}

	r := computeMetrics(c, selected)
	return BaselineResult{Precision: r.Precision, Recall: r.Recall, F1: r.F1}
}

func computeMetrics(c EvalCase, selected []string) EvalResult {
	truthSet := make(map[string]bool, len(c.GroundTruth))
	for _, f := range c.GroundTruth {
		truthSet[f] = true
	}

	selectedSet := make(map[string]bool, len(selected))
	for _, f := range selected {
		selectedSet[f] = true
	}

	var tp, fp, fn []string
	for _, f := range selected {
		if truthSet[f] {
			tp = append(tp, f)
		} else {
			fp = append(fp, f)
		}
	}
	for _, f := range c.GroundTruth {
		if !selectedSet[f] {
			fn = append(fn, f)
		}
	}

	precision := 0.0
	if len(selected) > 0 {
		precision = float64(len(tp)) / float64(len(selected))
	}
	recall := 0.0
	if len(c.GroundTruth) > 0 {
		recall = float64(len(tp)) / float64(len(c.GroundTruth))
	}
	f1 := 0.0
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}

	return EvalResult{
		EvalCase:       c,
		Precision:      precision,
		Recall:         recall,
		F1:             f1,
		Selected:       selected,
		TruePositives:  tp,
		FalsePositives: fp,
		FalseNegatives: fn,
	}
}

func printResults(results []EvalResult, baselines []BaselineResult) {
	fmt.Printf("%-30s  %6s  %6s  %6s  |  %-6s  %-6s  %-6s  (baseline)\n",
		"Case", "P", "R", "F1", "P", "R", "F1")
	fmt.Println(strings.Repeat("─", 90))

	var sumP, sumR, sumF1, bSumP, bSumR, bSumF1 float64
	for i, r := range results {
		b := baselines[i]
		name := r.Name
		if len(name) > 30 {
			name = name[:27] + "..."
		}
		fmt.Printf("%-30s  %6.2f  %6.2f  %6.2f  |  %6.2f  %6.2f  %6.2f\n",
			name, r.Precision, r.Recall, r.F1,
			b.Precision, b.Recall, b.F1)
		sumP += r.Precision
		sumR += r.Recall
		sumF1 += r.F1
		bSumP += b.Precision
		bSumR += b.Recall
		bSumF1 += b.F1
	}

	n := float64(len(results))
	fmt.Println(strings.Repeat("─", 90))
	fmt.Printf("%-30s  %6.2f  %6.2f  %6.2f  |  %6.2f  %6.2f  %6.2f\n",
		"AVERAGE",
		sumP/n, sumR/n, sumF1/n,
		bSumP/n, bSumR/n, bSumF1/n)

	fmt.Printf("\nEngine avg F1: %.3f vs Keyword baseline avg F1: %.3f  (delta: %+.3f)\n",
		sumF1/n, bSumF1/n, sumF1/n-bSumF1/n)

	// Print per-case details.
	fmt.Println("\n--- Per-case breakdown ---")
	sort.Slice(results, func(i, j int) bool { return results[i].F1 < results[j].F1 })
	for _, r := range results {
		fmt.Printf("\n[%s]\n  Task: %s\n  Precision: %.2f  Recall: %.2f  F1: %.2f\n",
			r.Name, r.Task, r.Precision, r.Recall, r.F1)
		if len(r.TruePositives) > 0 {
			fmt.Printf("  TP: %s\n", strings.Join(r.TruePositives, ", "))
		}
		if len(r.FalseNegatives) > 0 {
			fmt.Printf("  Missed: %s\n", strings.Join(r.FalseNegatives, ", "))
		}
		if len(r.FalsePositives) > 0 {
			fmt.Printf("  Extra: %s\n", strings.Join(r.FalsePositives, ", "))
		}
	}
}

func tokenise(s string) []string {
	stop := map[string]bool{
		"the": true, "and": true, "for": true, "fix": true, "bug": true,
		"add": true, "get": true, "use": true, "how": true, "why": true,
	}
	words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == ',' || r == '.' || r == ':' || r == '-'
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
