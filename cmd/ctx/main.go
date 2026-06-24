package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/akash202k/ctx/internal/clipboard"
	"github.com/akash202k/ctx/internal/editor"
	"github.com/akash202k/ctx/internal/run"
	"github.com/akash202k/ctx/internal/snapshot"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	Version = "dev"

	// Global flags
	verbose bool

	// Read command flags
	readBasePath  string
	readOutput    string
	readIncludes  []string
	readExcludes  []string
	readGithubURL string

	// Edit command flags
	editBasePath string
	editInput    string

	// Select command flags
	selectRepo     string
	selectTask     string
	selectEntry    string
	selectBudget   int
	selectDistance int
	selectJSON     bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "ctx",
		Short:   "ctx - context tool for AI workflows",
		Long:    "A tool for generating and applying context snapshots optimized for LLM interactions.",
		Version: Version,
		RunE:    runWizard, // Default: run interactive wizard
	}

	// Read command (non-interactive with flags)
	readCmd := &cobra.Command{
		Use:   "read",
		Short: "Generate a ctx snapshot of the repository",
		Long:  "Walks the repository and generates a @@CTX format snapshot for LLM context.",
		RunE:  runRead,
	}
	readCmd.Flags().StringVarP(&readBasePath, "base-path", "b", ".", "base path for file operations")
	readCmd.Flags().StringVarP(&readOutput, "output", "o", "", "output file (default: clipboard)")
	readCmd.Flags().StringArrayVarP(&readIncludes, "include", "i", nil, "paths to include")
	readCmd.Flags().StringArrayVarP(&readExcludes, "exclude", "e", nil, "paths to exclude")
	readCmd.Flags().StringVarP(&readGithubURL, "github-url", "g", "", "GitHub URL to clone and process")
	readCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Edit command (non-interactive with flags)
	editCmd := &cobra.Command{
		Use:   "edit",
		Short: "Apply ctx format edits from clipboard or file",
		Long:  "Reads @@CTX format edits and applies them to the repository.",
		RunE:  runEdit,
	}
	editCmd.Flags().StringVarP(&editBasePath, "base-path", "b", ".", "base path for file operations")
	editCmd.Flags().StringVarP(&editInput, "input", "i", "", "input file (default: clipboard)")
	editCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Select command (AI context selection)
	selectCmd := &cobra.Command{
		Use:   "select",
		Short: "Select relevant files for a task using graph-based scoring",
		Long:  "Analyzes repository structure and selects the most relevant files for a given task.",
		RunE:  runSelect,
	}
	selectCmd.Flags().StringVarP(&selectRepo, "repo", "r", ".", "repository root path")
	selectCmd.Flags().StringVarP(&selectTask, "task", "t", "", "task description (required)")
	selectCmd.Flags().StringVarP(&selectEntry, "entry", "e", "", "entry point file or symbol")
	selectCmd.Flags().IntVarP(&selectBudget, "budget", "b", 8000, "token budget")
	selectCmd.Flags().IntVarP(&selectDistance, "distance", "d", 3, "max graph distance")
	selectCmd.Flags().BoolVarP(&selectJSON, "json", "j", false, "output as JSON")
	selectCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	selectCmd.MarkFlagRequired("task")

	rootCmd.AddCommand(readCmd, editCmd, selectCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runRead(cmd *cobra.Command, args []string) error {
	basePath, err := filepath.Abs(readBasePath)
	if err != nil {
		return fmt.Errorf("invalid base path: %w", err)
	}

	// Build filter rules
	var filterRules []snapshot.FilterRule
	for _, path := range readIncludes {
		filterRules = append(filterRules, snapshot.FilterRule{Type: "include", Path: path})
	}
	for _, path := range readExcludes {
		filterRules = append(filterRules, snapshot.FilterRule{Type: "exclude", Path: path})
	}

	// Handle default rules
	if len(filterRules) == 0 {
		// No rules - default include everything
		filterRules = []snapshot.FilterRule{{Type: "include", Path: "."}}
	} else {
		// Check if user only provided exclude rules
		hasInclude := false
		for _, rule := range filterRules {
			if rule.Type == "include" {
				hasInclude = true
				break
			}
		}
		if !hasInclude {
			// Only excludes - prepend implicit include "." so excludes work correctly
			filterRules = append([]snapshot.FilterRule{{Type: "include", Path: "."}}, filterRules...)
		}
	}

	result, err := snapshot.Generate(basePath, filterRules)
	if err != nil {
		return fmt.Errorf("generate snapshot: %w", err)
	}

	// Output
	if readOutput != "" {
		outputPath := readOutput
		if !filepath.IsAbs(outputPath) {
			outputPath = filepath.Join(basePath, outputPath)
		}
		if err := os.WriteFile(outputPath, []byte(result.Content), 0644); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		color.Green("✓ Snapshot saved to %s", outputPath)
	} else {
		if err := clipboard.Write(result.Content); err != nil {
			return fmt.Errorf("copy to clipboard: %w", err)
		}
		color.Green("✓ Snapshot copied to clipboard")
	}

	fmt.Printf("\n")
	color.Cyan("Token estimate: %d", result.TokenEstimate)

	if len(result.IgnoredFiles) > 0 && verbose {
		color.Yellow("Ignored %d files", len(result.IgnoredFiles))
	}

	return nil
}

func runEdit(cmd *cobra.Command, args []string) error {
	basePath, err := filepath.Abs(editBasePath)
	if err != nil {
		return fmt.Errorf("invalid base path: %w", err)
	}

	// Read input
	var input string
	if editInput != "" {
		data, err := os.ReadFile(editInput)
		if err != nil {
			return fmt.Errorf("read input file: %w", err)
		}
		input = string(data)
	} else {
		content, err := clipboard.Read()
		if err != nil {
			return fmt.Errorf("read clipboard: %w", err)
		}
		input = content
	}

	// Parse
	files, err := editor.Parse(input)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	// Process
	ctx := editor.Context{
		RootDir: basePath,
		Verbose: verbose,
	}
	if err := editor.Process(files, ctx); err != nil {
		return fmt.Errorf("process: %w", err)
	}

	color.Green("✓ Edits applied successfully")
	return nil
}

func runSelect(cmd *cobra.Command, args []string) error {
	repoPath, err := filepath.Abs(selectRepo)
	if err != nil {
		return fmt.Errorf("invalid repo path: %w", err)
	}

	input := run.Input{
		Task:        selectTask,
		EntryPoint:  selectEntry,
		TokenBudget: selectBudget,
		RepoRoot:    repoPath,
		MaxDistance: selectDistance,
	}

	output, err := run.Run(input)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}

	if selectJSON {
		data, _ := run.MarshalOutput(output)
		fmt.Println(string(data))
		return nil
	}

	// Human-readable output
	color.Cyan("Context Selection — %s", output.Task)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Budget: %d / %d tokens (%.0f%%)\n",
		output.TokensUsed, output.TokenBudget,
		float64(output.TokensUsed)/float64(output.TokenBudget)*100)
	fmt.Printf("Graph: %d nodes, %d edges\n\n", output.Meta.GraphNodes, output.Meta.GraphEdges)

	fmt.Printf("Selected (%d files):\n\n", len(output.Selected))
	for _, f := range output.Selected {
		color.White("  %s", f.Path)
		fmt.Printf("    Score: %.2f  |  Mode: %s  |  Tokens: %d\n", f.Score, f.Mode, f.Tokens)
		fmt.Printf("    %s\n\n", f.Reason)
	}

	if len(output.ExcludedTopCandidates) > 0 {
		color.Yellow("Excluded near-misses (%d):\n", len(output.ExcludedTopCandidates))
		for _, ex := range output.ExcludedTopCandidates {
			fmt.Printf("  %s  (score %.2f)\n    %s\n\n", ex.Path, ex.Score, ex.Reason)
		}
	}

	return nil
}

// runWizard implements the interactive wizard (default behavior when running `ctx` with no subcommand)
func runWizard(cmd *cobra.Command, args []string) error {
	// Welcome box (using lipgloss)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		BorderForeground(lipgloss.Color("6"))

	welcomeText := boxStyle.Render("Welcome to ctx!")
	fmt.Println(welcomeText)
	fmt.Println()
	fmt.Println("ctx helps you generate context for your project so you can")
	fmt.Println("easily ask questions about it using an LLM.")
	fmt.Println()
	fmt.Println("This interactive wizard will guide you through the options and")
	fmt.Println("provide a non-interactive shortcut for future use.")
	fmt.Println()

	// Step 1: Choose command
	var command string
	err := huh.NewSelect[string]().
		Title("Choose a command:").
		Options(
			huh.NewOption("Read files to compress", "read"),
			huh.NewOption("Edit files based on ctx syntax", "edit"),
		).
		Value(&command).
		Run()
	if err != nil {
		return err
	}

	color.Cyan("Command selected: %s", command)
	commandString := fmt.Sprintf("ctx %s", command)
	color.Yellow("Current command: %s", commandString)
	fmt.Println()

	// Step 2: Base path
	var basePath string
	cwd, _ := os.Getwd()
	err = huh.NewInput().
		Title("Enter the base path for file operations:").
		Value(&basePath).
		Placeholder(cwd).
		Run()
	if err != nil {
		return err
	}

	if basePath == "" {
		basePath = cwd
	}

	color.Cyan("Base path: %s", basePath)
	commandString = fmt.Sprintf("ctx %s --base-path %q", command, basePath)
	color.Yellow("Current command: %s", commandString)
	fmt.Println()

	if command == "read" {
		return wizardRead(basePath, commandString)
	}
	return wizardEdit(basePath, commandString)
}

func wizardRead(basePath, initialCommandString string) error {
	commandString := initialCommandString

	// Step 3: Filter rules loop
	color.Cyan("Now, let's specify which files or directories to include or exclude.")
	fmt.Println()

	var filterRules []snapshot.FilterRule

	for {
		var ruleType string
		err := huh.NewSelect[string]().
			Title("Choose rule type:").
			Options(
				huh.NewOption("Finish adding rules", "finish"),
				huh.NewOption("Include path", "include"),
				huh.NewOption("Exclude path", "exclude"),
			).
			Value(&ruleType).
			Run()
		if err != nil {
			return err
		}

		if ruleType == "finish" {
			break
		}

		var rulePath string
		err = huh.NewInput().
			Title(fmt.Sprintf("Enter a file or directory path to %s:", ruleType)).
			Value(&rulePath).
			Run()
		if err != nil {
			return err
		}

		if rulePath != "" {
			filterRules = append(filterRules, snapshot.FilterRule{Type: ruleType, Path: rulePath})
			color.Cyan("Added %s rule: %s", ruleType, rulePath)
			commandString = fmt.Sprintf("%s --%s %q", commandString, ruleType, rulePath)
			color.Yellow("Current command: %s", commandString)
			fmt.Println()
		}
	}

	// Handle default rules based on what user entered
	if len(filterRules) == 0 {
		// No rules at all - default include everything
		filterRules = []snapshot.FilterRule{{Type: "include", Path: "."}}
		color.Cyan("Default include rule: .")
		commandString = fmt.Sprintf("%s --include %q", commandString, ".")
		color.Yellow("Current command: %s", commandString)
		fmt.Println()
	} else {
		// Check if user only added exclude rules
		hasInclude := false
		for _, rule := range filterRules {
			if rule.Type == "include" {
				hasInclude = true
				break
			}
		}
		if !hasInclude {
			// User only added excludes - prepend implicit include "." so excludes work correctly
			filterRules = append([]snapshot.FilterRule{{Type: "include", Path: "."}}, filterRules...)
			color.Cyan("Implicit include rule: . (all files)")
			// Update command string to show the implicit include
			commandString = fmt.Sprintf("%s --include %q", commandString, ".")
			color.Yellow("Current command: %s", commandString)
			fmt.Println()
		}
	}

	// Step 4: Output file
	color.Cyan("Let's configure the output options.")
	fmt.Println()

	var outputPath string
	err := huh.NewInput().
		Title("Enter the output file name/path (leave empty to copy to clipboard):").
		Value(&outputPath).
		Run()
	if err != nil {
		return err
	}

	if outputPath != "" {
		color.Cyan("Output: %s", outputPath)
		commandString = fmt.Sprintf("%s --output %q", commandString, outputPath)
	} else {
		color.Cyan("Output: Clipboard")
	}
	color.Yellow("Current command: %s", commandString)
	fmt.Println()

	// Step 5: Verbose
	color.Cyan("Almost done! Just a few more optional settings.")
	fmt.Println()

	var verboseOption bool
	err = huh.NewConfirm().
		Title("Enable verbose output?").
		Value(&verboseOption).
		Run()
	if err != nil {
		return err
	}

	if verboseOption {
		color.Cyan("Verbose: Yes")
		commandString = fmt.Sprintf("%s --verbose", commandString)
	} else {
		color.Cyan("Verbose: No")
	}
	color.Yellow("Current command: %s", commandString)
	fmt.Println()

	// Completion box
	completionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		BorderForeground(lipgloss.Color("2"))

	completionText := fmt.Sprintf("Wizard complete! You can use the following command to run ctx with these options:\n\n%s", commandString)
	fmt.Println(completionStyle.Render(completionText))
	fmt.Println()

	color.Cyan("Executing the command...")
	fmt.Println()

	// Execute the read operation
	absPath, _ := filepath.Abs(basePath)
	result, err := snapshot.Generate(absPath, filterRules)
	if err != nil {
		return fmt.Errorf("generate snapshot: %w", err)
	}

	// Output
	if outputPath != "" {
		outputAbsPath := outputPath
		if !filepath.IsAbs(outputAbsPath) {
			outputAbsPath = filepath.Join(absPath, outputAbsPath)
		}
		if err := os.WriteFile(outputAbsPath, []byte(result.Content), 0644); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		color.Green("✓ Snapshot saved to %s", outputAbsPath)
	} else {
		if err := clipboard.Write(result.Content); err != nil {
			return fmt.Errorf("copy to clipboard: %w", err)
		}
		color.Green("✓ Snapshot copied to clipboard")
	}

	fmt.Println()
	color.Cyan("Token estimate: %d", result.TokenEstimate)

	if len(result.IgnoredFiles) > 0 && verboseOption {
		color.Yellow("Ignored %d files", len(result.IgnoredFiles))
	}

	return nil
}

func wizardEdit(basePath, initialCommandString string) error {
	commandString := initialCommandString

	// Step 3: Verbose (edit has no filter rules or output options)
	color.Cyan("Almost done! Just a few more optional settings.")
	fmt.Println()

	var verboseOption bool
	err := huh.NewConfirm().
		Title("Enable verbose output?").
		Value(&verboseOption).
		Run()
	if err != nil {
		return err
	}

	if verboseOption {
		color.Cyan("Verbose: Yes")
		commandString = fmt.Sprintf("%s --verbose", commandString)
	} else {
		color.Cyan("Verbose: No")
	}
	color.Yellow("Current command: %s", commandString)
	fmt.Println()

	// Completion box
	completionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		BorderForeground(lipgloss.Color("2"))

	completionText := fmt.Sprintf("Wizard complete! You can use the following command to run ctx with these options:\n\n%s", commandString)
	fmt.Println(completionStyle.Render(completionText))
	fmt.Println()

	color.Cyan("Executing the command...")
	fmt.Println()

	// Execute the edit operation
	content, err := clipboard.Read()
	if err != nil {
		return fmt.Errorf("read clipboard: %w", err)
	}

	files, err := editor.Parse(content)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	absPath, _ := filepath.Abs(basePath)
	ctx := editor.Context{
		RootDir: absPath,
		Verbose: verboseOption,
	}
	if err := editor.Process(files, ctx); err != nil {
		return fmt.Errorf("process: %w", err)
	}

	color.Green("✓ Edits applied successfully")
	return nil
}
