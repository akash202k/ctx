package editor

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ChunkType identifies whether a chunk is an edit or no-change marker.
type ChunkType string

const (
	ChunkEdit     ChunkType = "edit"
	ChunkNoChange ChunkType = "no-change"
)

// Chunk represents a section of file content.
type Chunk struct {
	Type    ChunkType
	Content []string // Lines of content (empty for no-change chunks)
	BlockID string
}

// FileObject represents a parsed file with its chunks.
type FileObject struct {
	Path   string
	Chunks []Chunk
}

// Parser errors
var (
	ErrMissingPath            = errors.New("FILE tag is missing path attribute")
	ErrNestedFileSection      = errors.New("nested FILE tags are not allowed")
	ErrUnexpectedFileEnd      = errors.New("unexpected FILE end tag")
	ErrUnclosedFileSection    = errors.New("FILE section is not closed")
)

var (
	fileStartRegex = regexp.MustCompile(`^@@CTX<FILE\s+path="([^"]+)">`)
	fileEndRegex   = regexp.MustCompile(`^@@CTX</FILE>`)
	noChangeRegex  = regexp.MustCompile(`^@@CTX<NO-CHANGE\s*/>`)
)

// Parse reads ctx format input and returns parsed file objects.
func Parse(input string) ([]FileObject, error) {
	var files []FileObject
	var currentFile *FileObject
	var currentChunk *Chunk
	blockCounter := 0

	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check for file start
		if matches := fileStartRegex.FindStringSubmatch(trimmed); matches != nil {
			if currentFile != nil {
				return nil, ErrNestedFileSection
			}
			currentFile = &FileObject{
				Path:   matches[1],
				Chunks: []Chunk{},
			}
			blockCounter = 0
			continue
		}

		// Check for file end
		if fileEndRegex.MatchString(trimmed) {
			if currentFile == nil {
				return nil, ErrUnexpectedFileEnd
			}
			// Flush current chunk if any
			if currentChunk != nil {
				currentFile.Chunks = append(currentFile.Chunks, *currentChunk)
				currentChunk = nil
			}
			files = append(files, *currentFile)
			currentFile = nil
			continue
		}

		// Only process content if we're inside a file section
		if currentFile == nil {
			continue
		}

		// Check for no-change marker
		if noChangeRegex.MatchString(trimmed) {
			// Flush current edit chunk if any
			if currentChunk != nil {
				currentFile.Chunks = append(currentFile.Chunks, *currentChunk)
			}
			blockCounter++
			currentFile.Chunks = append(currentFile.Chunks, Chunk{
				Type:    ChunkNoChange,
				BlockID: fmt.Sprintf("block%d", blockCounter),
			})
			currentChunk = nil
			continue
		}

		// Regular content line
		if currentChunk == nil {
			blockCounter++
			currentChunk = &Chunk{
				Type:    ChunkEdit,
				Content: []string{},
				BlockID: fmt.Sprintf("block%d", blockCounter),
			}
		}
		currentChunk.Content = append(currentChunk.Content, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}

	if currentFile != nil {
		return nil, ErrUnclosedFileSection
	}

	return files, nil
}
