package walker

import (
	"os"
	"path/filepath"
	"sort"

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

func Walk(rootDir string) (*Result, error) {
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
