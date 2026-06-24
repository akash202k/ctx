package ignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

var alwaysIgnore = map[string]bool{
	".git":      true,
	".DS_Store": true,
}

var lockFiles = map[string]bool{
	"package-lock.json": true,
	"yarn.lock":         true,
	"bun.lockb":         true,
	"Pipfile.lock":      true,
	"poetry.lock":       true,
	"Gemfile.lock":      true,
}

type Matcher struct {
	patterns []string
}

func Load(rootDir string) (*Matcher, error) {
	f, err := os.Open(filepath.Join(rootDir, ".gitignore"))
	if err != nil {
		if os.IsNotExist(err) {
			return &Matcher{}, nil
		}
		return nil, err
	}
	defer f.Close()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &Matcher{patterns: patterns}, nil
}

func (m *Matcher) matches(relPath string) bool {
	base := filepath.Base(relPath)
	for _, p := range m.patterns {
		p = strings.TrimSuffix(p, "/")
		if p == base || p == relPath {
			return true
		}
		if strings.HasPrefix(relPath, p+string(filepath.Separator)) {
			return true
		}
		if ok, _ := filepath.Match(p, base); ok {
			return true
		}
	}
	return false
}

func (m *Matcher) IsIgnored(rootDir, fullPath string) bool {
	relPath, err := filepath.Rel(rootDir, fullPath)
	if err != nil {
		return false
	}
	base := filepath.Base(fullPath)

	if alwaysIgnore[base] || lockFiles[base] {
		return true
	}
	return m.matches(relPath)
}
