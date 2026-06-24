package graph

import (
	"bufio"
	"fmt"
	"math"
	"os/exec"
	"strings"
	"time"
)

// CommitInfo holds the set of files touched in a single commit and when.
type CommitInfo struct {
	Hash  string
	Files []string
	When  time.Time
}

// FileHistory captures per-file metadata derived from git log.
type FileHistory struct {
	// LastTouchDays is the number of days since this file was last committed.
	LastTouchDays float64
}

// BuildCoChange runs `git log` against repoRoot and returns:
//   - co-change edges (any two files in the same commit get/strengthen an edge)
//   - per-file history metadata (recency)
//
// sinceCommits caps how many commits are analysed (0 = all).
func BuildCoChange(repoRoot string, sinceCommits int) ([]Edge, map[string]FileHistory, error) {
	commits, err := parseGitLog(repoRoot, sinceCommits)
	if err != nil {
		return nil, nil, fmt.Errorf("git log: %w", err)
	}

	now := time.Now()

	// Count raw co-occurrences and track most-recent commit per pair.
	type pairKey struct{ A, B string }
	type pairVal struct {
		count   float64
		lastDay float64
	}
	pairs := make(map[pairKey]*pairVal)
	fileRecency := make(map[string]float64) // file → min days since last touch

	for _, commit := range commits {
		ageDays := now.Sub(commit.When).Hours() / 24
		if ageDays < 0 {
			ageDays = 0
		}

		for _, f := range commit.Files {
			if cur, ok := fileRecency[f]; !ok || ageDays < cur {
				fileRecency[f] = ageDays
			}
		}

		// All pairs within this commit.
		for i := 0; i < len(commit.Files); i++ {
			for j := i + 1; j < len(commit.Files); j++ {
				a, b := commit.Files[i], commit.Files[j]
				if a > b {
					a, b = b, a
				}
				k := pairKey{a, b}
				if _, ok := pairs[k]; !ok {
					pairs[k] = &pairVal{}
				}
				pairs[k].count++
				// Keep track of the most-recent co-change for decay.
				if pairs[k].lastDay == 0 || ageDays < pairs[k].lastDay {
					pairs[k].lastDay = ageDays
				}
			}
		}
	}

	// Convert to edges with recency-decayed weights.
	// weight = frequency * exp(-lastDays / 180)   (180-day half-life)
	edges := make([]Edge, 0, len(pairs))
	for k, v := range pairs {
		decayFactor := math.Exp(-v.lastDay / 180.0)
		weight := v.count * decayFactor
		edges = append(edges, Edge{
			From:   k.A,
			To:     k.B,
			Kind:   EdgeCoChange,
			Weight: weight,
		})
		// Co-change is undirected: add both directions so Neighbors() works symmetrically.
		edges = append(edges, Edge{
			From:   k.B,
			To:     k.A,
			Kind:   EdgeCoChange,
			Weight: weight,
		})
	}

	history := make(map[string]FileHistory, len(fileRecency))
	for f, days := range fileRecency {
		history[f] = FileHistory{LastTouchDays: days}
	}

	return edges, history, nil
}

// parseGitLog shells out to git and parses commit metadata + file lists.
func parseGitLog(repoRoot string, sinceCommits int) ([]CommitInfo, error) {
	args := []string{
		"-C", repoRoot,
		"log",
		"--name-only",
		"--pretty=format:COMMIT %H %ai", // hash + author-date ISO
		"--diff-filter=ACDMR",           // only real file changes, not renames
	}
	if sinceCommits > 0 {
		args = append(args, fmt.Sprintf("-n%d", sinceCommits))
	}

	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, err
	}

	var commits []CommitInfo
	var current *CommitInfo

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "COMMIT ") {
			if current != nil && len(current.Files) > 0 {
				commits = append(commits, *current)
			}
			parts := strings.SplitN(line, " ", 3)
			if len(parts) < 3 {
				current = &CommitInfo{Hash: parts[1]}
				continue
			}
			// parts[2] is the ISO date string; parse it.
			t, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[2])
			current = &CommitInfo{Hash: parts[1], When: t}
		} else if current != nil {
			current.Files = append(current.Files, line)
		}
	}
	if current != nil && len(current.Files) > 0 {
		commits = append(commits, *current)
	}

	return commits, scanner.Err()
}
