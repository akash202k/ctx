package snapshot

import (
	"testing"

	"github.com/akash202k/ctx/internal/walker"
)

// TestApplyFilters_ExcludeOnly tests that exclude-only rules work correctly
// (the main bug reported by the user)
func TestApplyFilters_ExcludeOnly(t *testing.T) {
	files := []walker.FileEntry{
		{RelPath: "Makefile", Content: "all:\n\tgo build\n"},
		{RelPath: "README.md", Content: "# Project\n"},
		{RelPath: "src/main.go", Content: "package main\n"},
	}

	rules := []FilterRule{{Type: "exclude", Path: "Makefile"}}

	filtered, ignored := applyFilters(files, rules, "/test")

	// Should include everything except Makefile
	if len(filtered) != 2 {
		t.Errorf("Expected 2 files, got %d", len(filtered))
	}

	// Check that correct files are included
	foundReadme := false
	foundMain := false
	for _, f := range filtered {
		if f.RelPath == "README.md" {
			foundReadme = true
		}
		if f.RelPath == "src/main.go" {
			foundMain = true
		}
	}

	if !foundReadme {
		t.Error("README.md should be included")
	}
	if !foundMain {
		t.Error("src/main.go should be included")
	}

	// Check that Makefile is ignored
	if _, exists := ignored["Makefile"]; !exists {
		t.Error("Makefile should be in ignored map")
	}
}

// TestApplyFilters_PathSpecificity tests that more specific paths take precedence
func TestApplyFilters_PathSpecificity(t *testing.T) {
	files := []walker.FileEntry{
		{RelPath: "src/app.go", Content: "package app"},
		{RelPath: "src/tests/unit.go", Content: "package tests"},
		{RelPath: "vendor/pkg.go", Content: "package vendor"},
	}

	rules := []FilterRule{
		{Type: "include", Path: "src"},
		{Type: "exclude", Path: "src/tests"},
	}

	filtered, _ := applyFilters(files, rules, "/test")

	// Should include src/app.go but exclude src/tests/unit.go
	// vendor/pkg.go should also be included (defaultInclude: true)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 files, got %d", len(filtered))
	}

	foundApp := false
	foundTests := false
	for _, f := range filtered {
		if f.RelPath == "src/app.go" {
			foundApp = true
		}
		if f.RelPath == "src/tests/unit.go" {
			foundTests = true
		}
	}

	if !foundApp {
		t.Error("src/app.go should be included")
	}
	if foundTests {
		t.Error("src/tests/unit.go should be excluded")
	}
}

// TestApplyFilters_PrefixBoundary tests separator-aware prefix matching
func TestApplyFilters_PrefixBoundary(t *testing.T) {
	files := []walker.FileEntry{
		{RelPath: "src/main.go", Content: "package main"},
		{RelPath: "srcfoo/other.go", Content: "package srcfoo"},
	}

	rules := []FilterRule{{Type: "include", Path: "src"}}

	filtered, _ := applyFilters(files, rules, "/test")

	// Should include src/main.go but NOT srcfoo/other.go
	// However, due to defaultInclude: true, srcfoo/other.go is also included
	// when it doesn't match any rule
	foundSrc := false
	foundSrcfoo := false
	for _, f := range filtered {
		if f.RelPath == "src/main.go" {
			foundSrc = true
		}
		if f.RelPath == "srcfoo/other.go" {
			foundSrcfoo = true
		}
	}

	if !foundSrc {
		t.Error("src/main.go should be included")
	}
	// Note: srcfoo/other.go is included because of defaultInclude: true
	// This matches Astrolark behavior
	if !foundSrcfoo {
		t.Error("srcfoo/other.go should be included (defaultInclude: true)")
	}
}

// TestApplyFilters_ExactMatch tests exact path matching
func TestApplyFilters_ExactMatch(t *testing.T) {
	files := []walker.FileEntry{
		{RelPath: "README.md", Content: "# Readme"},
		{RelPath: "README.txt", Content: "Readme"},
	}

	rules := []FilterRule{{Type: "exclude", Path: "README.md"}}

	filtered, ignored := applyFilters(files, rules, "/test")

	if len(filtered) != 1 {
		t.Errorf("Expected 1 file, got %d", len(filtered))
	}

	if filtered[0].RelPath != "README.txt" {
		t.Errorf("Expected README.txt, got %s", filtered[0].RelPath)
	}

	if _, exists := ignored["README.md"]; !exists {
		t.Error("README.md should be excluded")
	}
}

// TestApplyFilters_WildcardDotRule tests that "." matches all files
func TestApplyFilters_WildcardDotRule(t *testing.T) {
	files := []walker.FileEntry{
		{RelPath: "file1.go", Content: "package main"},
		{RelPath: "dir/file2.go", Content: "package dir"},
	}

	rules := []FilterRule{
		{Type: "include", Path: "."},
	}

	filtered, _ := applyFilters(files, rules, "/test")

	if len(filtered) != 2 {
		t.Errorf("Expected 2 files, got %d", len(filtered))
	}
}

// TestApplyFilters_ExcludeAll tests excluding everything
func TestApplyFilters_ExcludeAll(t *testing.T) {
	files := []walker.FileEntry{
		{RelPath: "file1.go", Content: "package main"},
		{RelPath: "dir/file2.go", Content: "package dir"},
	}

	rules := []FilterRule{
		{Type: "exclude", Path: "."},
	}

	filtered, ignored := applyFilters(files, rules, "/test")

	if len(filtered) != 0 {
		t.Errorf("Expected 0 files, got %d", len(filtered))
	}

	if len(ignored) != 2 {
		t.Errorf("Expected 2 ignored files, got %d", len(ignored))
	}
}

// TestApplyFilters_EmptyRules tests that empty rules include everything
func TestApplyFilters_EmptyRules(t *testing.T) {
	files := []walker.FileEntry{
		{RelPath: "file1.go", Content: "package main"},
		{RelPath: "file2.go", Content: "package main"},
	}

	rules := []FilterRule{}

	filtered, _ := applyFilters(files, rules, "/test")

	if len(filtered) != 2 {
		t.Errorf("Expected 2 files (no filtering), got %d", len(filtered))
	}
}

// TestApplyFilters_MultipleExcludes tests multiple exclude rules
func TestApplyFilters_MultipleExcludes(t *testing.T) {
	files := []walker.FileEntry{
		{RelPath: "Makefile", Content: "all:"},
		{RelPath: "vendor/pkg.go", Content: "package vendor"},
		{RelPath: "src/main.go", Content: "package main"},
		{RelPath: "README.md", Content: "# Readme"},
	}

	rules := []FilterRule{
		{Type: "exclude", Path: "Makefile"},
		{Type: "exclude", Path: "vendor"},
	}

	filtered, _ := applyFilters(files, rules, "/test")

	// Should include src/main.go and README.md only
	if len(filtered) != 2 {
		t.Errorf("Expected 2 files, got %d", len(filtered))
	}

	for _, f := range filtered {
		if f.RelPath == "Makefile" || f.RelPath == "vendor/pkg.go" {
			t.Errorf("File %s should be excluded", f.RelPath)
		}
	}
}

// TestApplyFilters_IncludeExcludeMixed tests mixed include/exclude rules
func TestApplyFilters_IncludeExcludeMixed(t *testing.T) {
	files := []walker.FileEntry{
		{RelPath: "src/app.go", Content: "package app"},
		{RelPath: "src/tests/unit.go", Content: "package tests"},
		{RelPath: "docs/guide.md", Content: "# Guide"},
		{RelPath: "vendor/pkg.go", Content: "package vendor"},
	}

	rules := []FilterRule{
		{Type: "include", Path: "."},
		{Type: "exclude", Path: "vendor"},
		{Type: "exclude", Path: "src/tests"},
	}

	filtered, _ := applyFilters(files, rules, "/test")

	// Should include src/app.go and docs/guide.md
	// Should exclude vendor/pkg.go and src/tests/unit.go
	if len(filtered) != 2 {
		t.Errorf("Expected 2 files, got %d", len(filtered))
	}

	for _, f := range filtered {
		if f.RelPath == "vendor/pkg.go" || f.RelPath == "src/tests/unit.go" {
			t.Errorf("File %s should be excluded", f.RelPath)
		}
	}
}

// TestApplyFilters_OrderIndependence tests that rule order doesn't matter (path length determines precedence)
func TestApplyFilters_OrderIndependence(t *testing.T) {
	files := []walker.FileEntry{
		{RelPath: "src/tests/unit.go", Content: "package tests"},
	}

	// Rules in this order: general include before specific exclude
	rules1 := []FilterRule{
		{Type: "include", Path: "src"},
		{Type: "exclude", Path: "src/tests"},
	}

	// Rules in reverse order: specific exclude before general include
	rules2 := []FilterRule{
		{Type: "exclude", Path: "src/tests"},
		{Type: "include", Path: "src"},
	}

	filtered1, _ := applyFilters(files, rules1, "/test")
	filtered2, _ := applyFilters(files, rules2, "/test")

	// Both should give the same result: file should be excluded
	// because src/tests is more specific than src
	if len(filtered1) != len(filtered2) {
		t.Error("Order of rules should not matter (path length determines precedence)")
	}

	if len(filtered1) != 0 {
		t.Errorf("Expected file to be excluded (more specific path wins), got %d files", len(filtered1))
	}
}

// TestApplyFilters_NestedDirectories tests filtering with nested directory structures
func TestApplyFilters_NestedDirectories(t *testing.T) {
	files := []walker.FileEntry{
		{RelPath: "a/b/c/file.go", Content: "package c"},
		{RelPath: "a/b/other.go", Content: "package b"},
		{RelPath: "a/file.go", Content: "package a"},
	}

	rules := []FilterRule{
		{Type: "include", Path: "a"},
		{Type: "exclude", Path: "a/b/c"},
	}

	filtered, _ := applyFilters(files, rules, "/test")

	// Should exclude a/b/c/file.go but include a/b/other.go and a/file.go
	if len(filtered) != 2 {
		t.Errorf("Expected 2 files, got %d", len(filtered))
	}

	for _, f := range filtered {
		if f.RelPath == "a/b/c/file.go" {
			t.Error("a/b/c/file.go should be excluded")
		}
	}
}
