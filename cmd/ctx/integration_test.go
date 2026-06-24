package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLI_ReadExclude tests the CLI with exclude flag
func TestCLI_ReadExclude(t *testing.T) {
	// Test case 1: Exclude Makefile
	t.Run("ExcludeMakefile", func(t *testing.T) {
		// Create a temporary directory with test files
		tmpDir, err := os.MkdirTemp("", "ctx-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Create test files
		testFiles := map[string]string{
			"Makefile":     "all:\n\tgo build\n",
			"README.md":    "# Test Project\n",
			"src/main.go":  "package main\n",
			"vendor/pkg.go": "package vendor\n",
		}

		for path, content := range testFiles {
			fullPath := filepath.Join(tmpDir, path)
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
		}

		outputPath := filepath.Join(tmpDir, "output1.ctx")

		// Simulate: ctx read --base-path tmpDir --exclude Makefile --output output1.ctx
		readBasePath = tmpDir
		readOutput = outputPath
		readIncludes = nil
		readExcludes = []string{"Makefile"}
		verbose = false

		err = runRead(nil, nil)
		if err != nil {
			t.Fatalf("runRead failed: %v", err)
		}

		// Read the output
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read output: %v", err)
		}

		output := string(content)

		// Verify Makefile is excluded
		if strings.Contains(output, `@@CTX<FILE path="Makefile">`) {
			t.Error("Makefile should be excluded from output")
		}

		// Verify other files are included
		if !strings.Contains(output, `@@CTX<FILE path="README.md">`) {
			t.Error("README.md should be included in output")
		}
		if !strings.Contains(output, `@@CTX<FILE path="src/main.go">`) {
			t.Error("src/main.go should be included in output")
		}
	})

	// Test case 2: Exclude directory
	t.Run("ExcludeDirectory", func(t *testing.T) {
		// Create a fresh temporary directory
		tmpDir, err := os.MkdirTemp("", "ctx-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Create test files
		testFiles := map[string]string{
			"Makefile":      "all:\n\tgo build\n",
			"README.md":     "# Test Project\n",
			"src/main.go":   "package main\n",
			"vendor/pkg.go": "package vendor\n",
		}

		for path, content := range testFiles {
			fullPath := filepath.Join(tmpDir, path)
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
		}

		outputPath := filepath.Join(tmpDir, "output2.ctx")

		readBasePath = tmpDir
		readOutput = outputPath
		readIncludes = nil
		readExcludes = []string{"vendor"}
		verbose = false

		err = runRead(nil, nil)
		if err != nil {
			t.Fatalf("runRead failed: %v", err)
		}

		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read output: %v", err)
		}

		output := string(content)

		// Verify vendor directory is excluded
		if strings.Contains(output, `@@CTX<FILE path="vendor/pkg.go">`) {
			t.Error("vendor/pkg.go should be excluded from output")
		}

		// Verify other files are included
		if !strings.Contains(output, `@@CTX<FILE path="Makefile">`) {
			t.Error("Makefile should be included in output")
		}
	})
}

// TestCLI_ReadIncludeExclude tests mixed include/exclude rules
func TestCLI_ReadIncludeExclude(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ctx-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files with nested structure
	testFiles := map[string]string{
		"src/app.go":        "package app\n",
		"src/tests/unit.go": "package tests\n",
		"docs/guide.md":     "# Guide\n",
		"vendor/pkg.go":     "package vendor\n",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test: Include src, exclude src/tests
	outputPath := filepath.Join(tmpDir, "output.ctx")

	readBasePath = tmpDir
	readOutput = outputPath
	readIncludes = []string{"src"}
	readExcludes = []string{"src/tests"}
	verbose = false

	err = runRead(nil, nil)
	if err != nil {
		t.Fatalf("runRead failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	output := string(content)

	// Verify src/app.go is included (matches include src)
	if !strings.Contains(output, `@@CTX<FILE path="src/app.go">`) {
		t.Error("src/app.go should be included (matches include rule)")
	}

	// Verify src/tests/unit.go is excluded (more specific path)
	if strings.Contains(output, `@@CTX<FILE path="src/tests/unit.go">`) {
		t.Error("src/tests/unit.go should be excluded (more specific exclude rule)")
	}

	// Note: docs/guide.md and vendor/pkg.go will be included due to defaultInclude: true
	// This matches Astrolark behavior
}

// TestCLI_ReadMultipleExcludes tests multiple exclude rules
func TestCLI_ReadMultipleExcludes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ctx-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFiles := map[string]string{
		"Makefile":      "all:\n",
		"vendor/pkg.go": "package vendor\n",
		"src/main.go":   "package main\n",
		"README.md":     "# Readme\n",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	outputPath := filepath.Join(tmpDir, "output.ctx")

	readBasePath = tmpDir
	readOutput = outputPath
	readIncludes = nil
	readExcludes = []string{"Makefile", "vendor"}
	verbose = false

	err = runRead(nil, nil)
	if err != nil {
		t.Fatalf("runRead failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	output := string(content)

	// Verify exclusions
	if strings.Contains(output, `@@CTX<FILE path="Makefile">`) {
		t.Error("Makefile should be excluded")
	}
	if strings.Contains(output, `@@CTX<FILE path="vendor/pkg.go">`) {
		t.Error("vendor/pkg.go should be excluded")
	}

	// Verify inclusions
	if !strings.Contains(output, `@@CTX<FILE path="src/main.go">`) {
		t.Error("src/main.go should be included")
	}
	if !strings.Contains(output, `@@CTX<FILE path="README.md">`) {
		t.Error("README.md should be included")
	}
}

// TestCLI_ReadPrefixBoundary tests that prefix matching respects path separators
func TestCLI_ReadPrefixBoundary(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ctx-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFiles := map[string]string{
		"src/main.go":     "package main\n",
		"srcfoo/other.go": "package srcfoo\n",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	outputPath := filepath.Join(tmpDir, "output.ctx")

	readBasePath = tmpDir
	readOutput = outputPath
	readIncludes = []string{"src"}
	readExcludes = nil
	verbose = false

	err = runRead(nil, nil)
	if err != nil {
		t.Fatalf("runRead failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	output := string(content)

	// Verify src/main.go is included
	if !strings.Contains(output, `@@CTX<FILE path="src/main.go">`) {
		t.Error("src/main.go should be included")
	}

	// srcfoo/other.go should be excluded (include-only rule, no match)
	// This is the CORRECT behavior: include-only rules exclude unmatched files
	if strings.Contains(output, `@@CTX<FILE path="srcfoo/other.go">`) {
		t.Error("srcfoo/other.go should be excluded (include-only rule, no match)")
	}
}
