package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"ctxengine/internal/run"
	"ctxengine/internal/walker"
)

func main() {
	// Sub-commands: default is the walk/list mode (Layer 0 UI).
	// Adding -select mode switches into the full pipeline.
	root := flag.String("path", ".", "root directory to scan (walk mode)")
	showContent := flag.Bool("content", false, "print file contents in walk mode")
	selectMode := flag.Bool("select", false, "run the full context-selection pipeline")
	task := flag.String("task", "", "task description (select mode)")
	entry := flag.String("entry", "", "entry point file or path#Symbol (select mode)")
	budget := flag.Int("budget", 8000, "token budget (select mode)")
	maxDist := flag.Int("distance", 3, "max graph hop distance (select mode)")
	jsonOut := flag.Bool("json", false, "output JSON (select mode, implied by -select)")
	noContent := flag.Bool("no-content", false, "omit file content from JSON output (select mode)")
	flag.Parse()

	if *selectMode {
		runSelect(*root, *task, *entry, *budget, *maxDist, *jsonOut || true, *noContent)
		return
	}

	runWalk(*root, *showContent)
}

func runSelect(root, task, entry string, budget, maxDist int, asJSON, noContent bool) {
	absRoot, err := absPath(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error resolving path:", err)
		os.Exit(1)
	}

	inp := run.Input{
		Task:        task,
		EntryPoint:  entry,
		TokenBudget: budget,
		RepoRoot:    absRoot,
		MaxDistance: maxDist,
	}

	output, err := run.Run(inp)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if noContent {
		for i := range output.Selected {
			output.Selected[i].Content = ""
		}
	}

	if asJSON {
		b, _ := run.MarshalOutput(output)
		fmt.Println(string(b))
		return
	}

	printHuman(output)
}

func printHuman(o run.Output) {
	fmt.Printf("Context Engine — %s\n", o.Task)
	fmt.Println(line(60))
	fmt.Printf("Budget: %d / %d tokens (%.0f%%)\n",
		o.TokensUsed, o.TokenBudget,
		float64(o.TokensUsed)/float64(o.TokenBudget)*100)
	fmt.Printf("Graph: %d nodes, %d edges, built in %dms\n\n",
		o.Meta.GraphNodes, o.Meta.GraphEdges, o.Meta.GraphBuildMs)

	fmt.Printf("Selected (%d files):\n\n", len(o.Selected))
	for _, f := range o.Selected {
		fmt.Printf("  %s\n", f.Path)
		fmt.Printf("    Score: %.2f  |  Mode: %s  |  Tokens: %d\n", f.Score, f.Mode, f.Tokens)
		fmt.Printf("    %s\n", f.Reason)
		if len(f.Symbols) > 0 {
			fmt.Printf("    Exported: %s\n", joinN(f.Symbols, 3))
		}
		fmt.Println()
	}

	if len(o.ExcludedTopCandidates) > 0 {
		fmt.Printf("Excluded near-misses (%d):\n\n", len(o.ExcludedTopCandidates))
		for _, ex := range o.ExcludedTopCandidates {
			fmt.Printf("  %s  (score %.2f)\n    %s\n\n", ex.Path, ex.Score, ex.Reason)
		}
	}
	fmt.Println(line(60))
}

func runWalk(root string, showContent bool) {
	absRoot, err := absPath(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error resolving path:", err)
		os.Exit(1)
	}

	result, err := walker.Walk(absRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error walking directory:", err)
		os.Exit(1)
	}

	fmt.Printf("Scanned: %s\n\n", absRoot)
	fmt.Printf("Files included (%d):\n", len(result.Files))
	for _, f := range result.Files {
		fmt.Println(" ", f.RelPath)
	}
	fmt.Printf("\nSkipped (%d):\n", len(result.Skipped))
	for _, s := range result.Skipped {
		fmt.Println(" ", s)
	}
	if showContent {
		fmt.Println("\n--- contents ---")
		for _, f := range result.Files {
			fmt.Printf("\n# %s\n%s\n", f.RelPath, f.Content)
		}
	}
}

// readInputJSON reads a run.Input from a JSON file or stdin ("-").
func readInputJSON(path string) (run.Input, error) {
	var inp run.Input
	var data []byte
	var err error
	if path == "-" {
		data, err = os.ReadFile("/dev/stdin")
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return inp, err
	}
	err = json.Unmarshal(data, &inp)
	return inp, err
}

func line(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = '─'
	}
	return string(b)
}

func joinN(ss []string, n int) string {
	if len(ss) <= n {
		return join(ss)
	}
	return join(ss[:n]) + fmt.Sprintf(" … +%d more", len(ss)-n)
}

func join(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
