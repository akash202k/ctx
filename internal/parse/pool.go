package parse

import (
	"path/filepath"
	"strings"
	"sync"
)

// job is a unit of work sent to a worker goroutine.
type job struct {
	path    string
	content string
	parser  Parser
}

// result is returned by a worker goroutine.
type result struct {
	pf  ParsedFile
	err error
}

// FileInput is a file path + content pair passed to ParseConcurrently.
type FileInput struct {
	Path    string
	Content string
}

// ParseConcurrently parses the given files using the provided parsers in parallel.
// It uses a worker pool of workerCount goroutines. Each goroutine independently
// parses one file and sends its result over a channel — no shared mutable state.
// Returns a map of repo-relative path → ParsedFile for all parseable files.
func ParseConcurrently(files []FileInput, parsers []Parser, workerCount int) map[string]ParsedFile {
	// Index parsers by extension.
	byExt := make(map[string]Parser, len(parsers)*2)
	for _, p := range parsers {
		for _, ext := range p.Extensions() {
			byExt[ext] = p
		}
	}

	// Filter to only files we have a parser for, build the job list.
	var jobs []job
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f.Path))
		p, ok := byExt[ext]
		if !ok {
			continue
		}
		jobs = append(jobs, job{path: f.Path, content: f.Content, parser: p})
	}

	if len(jobs) == 0 {
		return nil
	}

	if workerCount <= 0 {
		workerCount = 4
	}
	if workerCount > len(jobs) {
		workerCount = len(jobs)
	}

	jobCh := make(chan job, len(jobs))
	resultCh := make(chan result, len(jobs))

	// Launch workers.
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobCh {
				pf, err := j.parser.Parse(j.path, j.content)
				resultCh <- result{pf: pf, err: err}
			}
		}()
	}

	// Send all jobs.
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	// Close result channel once all workers are done.
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results — single goroutine, no mutex needed.
	out := make(map[string]ParsedFile, len(jobs))
	for r := range resultCh {
		if r.err == nil {
			out[r.pf.Path] = r.pf
		}
	}
	return out
}
