package walker

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/akash202k/ctx/internal/fileinfo"
	"github.com/akash202k/ctx/internal/ignore"
)

type FileEntry struct {
	RelPath string
	Content string
}

type Result struct {
	Files   []FileEntry
	Skipped []string
}

// Walk walks the directory tree and returns all non-binary, non-ignored files.
// It tries to use git ls-files for git repositories (100% spec-compliant gitignore),
// falling back to filesystem walk for non-git directories.
func Walk(rootDir string) (*Result, error) {
	// Try git-based walk first
	if result, err := gitWalk(rootDir); err == nil {
		return result, nil
	}
	// Fall back to filesystem walk
	return fsWalk(rootDir)
}

// gitWalk uses git ls-files to list files, respecting .gitignore rules.
// Returns error if git is not available or rootDir is not a git repository.
func gitWalk(rootDir string) (*Result, error) {
	// Check if git is available and this is a git repo
	cmd := exec.Command("git", "-C", rootDir, "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return nil, err // Not a git repo or git not available
	}

	// Get list of files from git
	cmd = exec.Command("git", "-C", rootDir,
		"ls-files", "--cached", "--others", "--exclude-standard")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	result := &Result{}
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		relPath := strings.TrimSpace(scanner.Text())
		if relPath == "" {
			continue
		}

		fullPath := filepath.Join(rootDir, relPath)

		// Check if it's a file (git ls-files includes directories sometimes)
		info, err := os.Stat(fullPath)
		if err != nil {
			// File might have been deleted after git ls-files ran
			continue
		}
		if info.IsDir() {
			continue
		}

		// Check if binary
		isBin, err := fileinfo.IsBinary(fullPath)
		if err != nil {
			continue
		}
		if isBin {
			result.Skipped = append(result.Skipped, relPath)
			continue
		}

		// Read content
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		result.Files = append(result.Files, FileEntry{
			RelPath: relPath,
			Content: string(content),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Slice(result.Files, func(i, j int) bool {
		return result.Files[i].RelPath < result.Files[j].RelPath
	})

	return result, nil
}

// fsWalk is the fallback filesystem walker for non-git directories.
func fsWalk(rootDir string) (*Result, error) {
	matcher, err := ignore.Load(rootDir)
	if err != nil {
		return nil, err
	}

	result := &Result{}

	err = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == rootDir {
			return nil
		}

		relPath, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return relErr
		}

		if matcher.IsIgnored(rootDir, path) {
			result.Skipped = append(result.Skipped, relPath)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		isBin, err := fileinfo.IsBinary(path)
		if err != nil {
			return err
		}
		if isBin {
			result.Skipped = append(result.Skipped, relPath)
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		result.Files = append(result.Files, FileEntry{
			RelPath: relPath,
			Content: string(content),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(result.Files, func(i, j int) bool {
		return result.Files[i].RelPath < result.Files[j].RelPath
	})

	return result, nil
}
