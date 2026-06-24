package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BlockType describes how a block should be processed.
type BlockType string

const (
	BlockFull   BlockType = "full"
	BlockTop    BlockType = "top"
	BlockMiddle BlockType = "middle"
	BlockBottom BlockType = "bottom"
)

// ProcessedChunk extends Chunk with block type and anchor information.
type ProcessedChunk struct {
	Chunk
	BlockType    BlockType
	TopAnchor    string
	BottomAnchor string
}

// ProcessedFile extends FileObject with processed chunks.
type ProcessedFile struct {
	Path   string
	Chunks []ProcessedChunk
}

// Context holds configuration for the processor.
type Context struct {
	RootDir string
	Verbose bool
}

// Process applies all four phases to the parsed file objects.
func Process(files []FileObject, ctx Context) error {
	// Phase 1: Remove consecutive no-change chunks
	phase1Files := phase1(files)

	// Phase 2: Identify block types and assign anchors
	phase2Files, err := phase2(phase1Files)
	if err != nil {
		return fmt.Errorf("phase2: %w", err)
	}

	// Phase 3: Insert anchors into original files
	if err := phase3(phase2Files, ctx); err != nil {
		return fmt.Errorf("phase3: %w", err)
	}

	// Phase 4: Replace content between anchors
	if err := phase4(phase2Files, ctx); err != nil {
		return fmt.Errorf("phase4: %w", err)
	}

	return nil
}

// phase1 removes consecutive no-change chunks.
func phase1(files []FileObject) []FileObject {
	result := make([]FileObject, len(files))
	for i, file := range files {
		var filtered []Chunk
		var lastWasNoChange bool

		for _, chunk := range file.Chunks {
			if chunk.Type == ChunkNoChange {
				if !lastWasNoChange {
					filtered = append(filtered, chunk)
					lastWasNoChange = true
				}
			} else {
				filtered = append(filtered, chunk)
				lastWasNoChange = false
			}
		}

		result[i] = FileObject{
			Path:   file.Path,
			Chunks: filtered,
		}
	}
	return result
}

// phase2 assigns block types and generates anchor names.
func phase2(files []FileObject) ([]ProcessedFile, error) {
	result := make([]ProcessedFile, len(files))

	for i, file := range files {
		processed := ProcessedFile{
			Path:   file.Path,
			Chunks: make([]ProcessedChunk, len(file.Chunks)),
		}

		// Special case: single edit chunk = full file replacement
		if len(file.Chunks) == 1 && file.Chunks[0].Type == ChunkEdit {
			processed.Chunks[0] = ProcessedChunk{
				Chunk:        file.Chunks[0],
				BlockType:    BlockFull,
				TopAnchor:    fmt.Sprintf("@@CTX_%s_ANCHOR_TOP", file.Chunks[0].BlockID),
				BottomAnchor: fmt.Sprintf("@@CTX_%s_ANCHOR_BOTTOM", file.Chunks[0].BlockID),
			}
			result[i] = processed
			continue
		}

		// Multi-chunk: assign top/middle/bottom
		for j, chunk := range file.Chunks {
			pc := ProcessedChunk{Chunk: chunk}

			if chunk.Type == ChunkNoChange {
				// No anchors for no-change blocks
				processed.Chunks[j] = pc
				continue
			}

			// Edit chunk - determine position
			isFirst := j == 0
			isLast := j == len(file.Chunks)-1
			hasPrevNoChange := j > 0 && file.Chunks[j-1].Type == ChunkNoChange
			hasNextNoChange := j < len(file.Chunks)-1 && file.Chunks[j+1].Type == ChunkNoChange

			if isFirst {
				pc.BlockType = BlockTop
			} else if isLast {
				pc.BlockType = BlockBottom
			} else if hasPrevNoChange && hasNextNoChange {
				pc.BlockType = BlockMiddle
			} else {
				return nil, fmt.Errorf("misplaced edit block at index %d in file %s", j, file.Path)
			}

			pc.TopAnchor = fmt.Sprintf("@@CTX_%s_ANCHOR_TOP", chunk.BlockID)
			pc.BottomAnchor = fmt.Sprintf("@@CTX_%s_ANCHOR_BOTTOM", chunk.BlockID)
			processed.Chunks[j] = pc
		}

		result[i] = processed
	}

	return result, nil
}

// phase3 inserts anchor markers into the original files.
func phase3(files []ProcessedFile, ctx Context) error {
	for _, file := range files {
		filePath := resolveFilePath(file.Path, ctx.RootDir)

		if ctx.Verbose {
			fmt.Printf("Processing file: %s\n", filePath)
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}

		// Read original file
		originalContent := ""
		isNewFile := false
		data, err := os.ReadFile(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				isNewFile = true
				if ctx.Verbose {
					fmt.Printf("  Creating new file: %s\n", filePath)
				}
			} else {
				return fmt.Errorf("read file: %w", err)
			}
		} else {
			originalContent = string(data)
		}

		// Check for existing anchors
		if strings.Contains(originalContent, "@@CTX_") {
			return fmt.Errorf("file %s already contains ctx anchors", filePath)
		}

		// Check for no-change in new files
		if isNewFile {
			for _, chunk := range file.Chunks {
				if chunk.Type == ChunkNoChange {
					return fmt.Errorf("newly created file %s contains no-change block", filePath)
				}
			}
		}

		lines := strings.Split(originalContent, "\n")
		modifiedLines := make([]string, len(lines))
		copy(modifiedLines, lines)
		offset := 0

		// Insert anchors for each edit chunk
		for _, chunk := range file.Chunks {
			if chunk.Type != ChunkEdit || chunk.BlockType == BlockFull {
				continue
			}

			// Find anchor positions using sliding window
			topPos, bottomPos := findAnchorPositions(lines, chunk.Content)

			if ctx.Verbose {
				fmt.Printf("  Block %s: top=%d, bottom=%d\n", chunk.BlockID, topPos, bottomPos)
			}

			if topPos >= 0 {
				modifiedLines = insertAt(modifiedLines, topPos+offset, chunk.TopAnchor)
				offset++
			}
			if bottomPos >= 0 {
				modifiedLines = insertAt(modifiedLines, bottomPos+offset+1, chunk.BottomAnchor)
				offset++
			}
		}

		// Write back
		if err := os.WriteFile(filePath, []byte(strings.Join(modifiedLines, "\n")), 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}

	return nil
}

// phase4 replaces content between anchors (or full file for BlockFull).
func phase4(files []ProcessedFile, ctx Context) error {
	for _, file := range files {
		filePath := resolveFilePath(file.Path, ctx.RootDir)

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}

		content := string(data)

		for _, chunk := range file.Chunks {
			if chunk.Type != ChunkEdit {
				continue
			}

			newContent := strings.Join(chunk.Content, "\n")

			if chunk.BlockType == BlockFull {
				// Replace entire file
				content = newContent
				continue
			}

			// Find and replace between anchors
			startIdx := strings.Index(content, chunk.TopAnchor)
			endIdx := strings.Index(content, chunk.BottomAnchor)

			if startIdx == -1 || endIdx == -1 {
				return fmt.Errorf("anchor not found in %s (start=%d, end=%d)", file.Path, startIdx, endIdx)
			}

			// Replace content between anchors (removing the anchors themselves)
			content = content[:startIdx] + newContent + content[endIdx+len(chunk.BottomAnchor):]
		}

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}

	return nil
}

func resolveFilePath(relPath, rootDir string) string {
	if filepath.IsAbs(relPath) {
		return relPath
	}
	return filepath.Join(rootDir, relPath)
}

func insertAt(lines []string, pos int, line string) []string {
	if pos < 0 || pos > len(lines) {
		return lines
	}
	result := make([]string, len(lines)+1)
	copy(result[:pos], lines[:pos])
	result[pos] = line
	copy(result[pos+1:], lines[pos:])
	return result
}

func findAnchorPositions(originalLines, editLines []string) (topPos, bottomPos int) {
	if len(editLines) == 0 {
		return -1, -1
	}

	// Find bottom anchor using sliding window (from end)
	bottomPos = findUniqueMatch(originalLines, editLines, false)

	// Find top anchor (from start, up to bottom position)
	if bottomPos >= 0 {
		topPos = findUniqueMatch(originalLines[:bottomPos], editLines, true)
	} else {
		topPos = -1
	}

	return topPos, bottomPos
}

func findUniqueMatch(haystack, needle []string, fromStart bool) int {
	if len(needle) == 0 || len(haystack) == 0 {
		return -1
	}

	for windowSize := 1; windowSize <= len(needle); windowSize++ {
		var searchWindow []string
		if fromStart {
			searchWindow = needle[:windowSize]
		} else {
			searchWindow = needle[len(needle)-windowSize:]
		}

		matches := findAllMatches(haystack, searchWindow)
		if len(matches) == 1 {
			if fromStart {
				return matches[0]
			}
			return matches[0] + len(searchWindow) - 1
		}
		if len(matches) == 0 {
			break
		}
	}

	return -1
}

func findAllMatches(haystack, needle []string) []int {
	var matches []int
	needleStr := strings.Join(needle, "\n")

	for i := 0; i <= len(haystack)-len(needle); i++ {
		section := strings.Join(haystack[i:i+len(needle)], "\n")
		if section == needleStr {
			matches = append(matches, i)
		}
	}

	return matches
}
