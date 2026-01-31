package extractor

import (
	"bufio"
	"os"
	"strings"
	"unicode"
)

// Chunk represents a portion of text from a book.
type Chunk struct {
	Text       string
	StartLine  int
	EndLine    int
	Chapter    string
	WordCount  int
	CharCount  int
	ChunkIndex int
}

// ChunkerConfig holds configuration for the chunker.
type ChunkerConfig struct {
	// Target size for each chunk in words
	TargetWords int
	// Overlap between chunks in words
	OverlapWords int
	// Minimum words for a valid chunk
	MinWords int
}

// DefaultChunkerConfig returns sensible defaults for chunking.
func DefaultChunkerConfig() ChunkerConfig {
	return ChunkerConfig{
		TargetWords:  2000,
		OverlapWords: 200,
		MinWords:     500,
	}
}

// Chunker splits book text into overlapping chunks for processing.
type Chunker struct {
	config ChunkerConfig
}

// NewChunker creates a new chunker with the given config.
func NewChunker(config ChunkerConfig) *Chunker {
	return &Chunker{config: config}
}

// ChunkFile reads a file and splits it into chunks.
func (c *Chunker) ChunkFile(path string) ([]Chunk, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return c.ChunkLines(lines), nil
}

// ChunkText splits text into chunks.
func (c *Chunker) ChunkText(text string) []Chunk {
	lines := strings.Split(text, "\n")
	return c.ChunkLines(lines)
}

// ChunkLines splits lines into chunks.
func (c *Chunker) ChunkLines(lines []string) []Chunk {
	// First, strip Gutenberg header/footer
	lines = stripGutenbergBoilerplate(lines)

	// Build chunks
	var chunks []Chunk
	var currentChunk strings.Builder
	var currentWords int
	var startLine int
	var currentChapter string
	var chunkIndex int

	for i, line := range lines {
		// Detect chapter headings
		if chapter := detectChapter(line); chapter != "" {
			currentChapter = chapter
		}

		// Count words in this line
		lineWords := countWords(line)

		// Add line to current chunk
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n")
		}
		currentChunk.WriteString(line)
		currentWords += lineWords

		// Check if we've reached target size
		if currentWords >= c.config.TargetWords {
			// Find a good break point (paragraph boundary)
			chunkText := currentChunk.String()
			breakPoint := findBreakPoint(chunkText, c.config.TargetWords, c.config.OverlapWords)

			if breakPoint > 0 && breakPoint < len(chunkText) {
				chunk := Chunk{
					Text:       strings.TrimSpace(chunkText[:breakPoint]),
					StartLine:  startLine,
					EndLine:    i,
					Chapter:    currentChapter,
					WordCount:  countWords(chunkText[:breakPoint]),
					CharCount:  len(chunkText[:breakPoint]),
					ChunkIndex: chunkIndex,
				}

				if chunk.WordCount >= c.config.MinWords {
					chunks = append(chunks, chunk)
					chunkIndex++
				}

				// Start new chunk with overlap
				overlapText := chunkText[breakPoint:]
				currentChunk.Reset()
				currentChunk.WriteString(overlapText)
				currentWords = countWords(overlapText)
				startLine = i - countNewlines(overlapText)
			}
		}
	}

	// Add final chunk if it has enough content
	if currentWords >= c.config.MinWords {
		chunkText := currentChunk.String()
		chunk := Chunk{
			Text:       strings.TrimSpace(chunkText),
			StartLine:  startLine,
			EndLine:    len(lines) - 1,
			Chapter:    currentChapter,
			WordCount:  currentWords,
			CharCount:  len(chunkText),
			ChunkIndex: chunkIndex,
		}
		chunks = append(chunks, chunk)
	}

	return chunks
}

// stripGutenbergBoilerplate removes Project Gutenberg header and footer.
func stripGutenbergBoilerplate(lines []string) []string {
	startIdx := 0
	endIdx := len(lines)

	// Find start marker
	for i, line := range lines {
		if strings.Contains(line, "*** START OF") ||
			strings.Contains(line, "***START OF") ||
			strings.Contains(line, "*END*THE SMALL PRINT") {
			startIdx = i + 1
			break
		}
	}

	// Find end marker
	for i := len(lines) - 1; i >= startIdx; i-- {
		if strings.Contains(lines[i], "*** END OF") ||
			strings.Contains(lines[i], "***END OF") ||
			strings.Contains(lines[i], "End of Project Gutenberg") ||
			strings.Contains(lines[i], "End of the Project Gutenberg") {
			endIdx = i
			break
		}
	}

	if startIdx >= endIdx {
		return lines
	}

	return lines[startIdx:endIdx]
}

// detectChapter checks if a line is a chapter heading.
func detectChapter(line string) string {
	line = strings.TrimSpace(line)
	upper := strings.ToUpper(line)

	// Common chapter patterns
	patterns := []string{
		"CHAPTER",
		"PART",
		"BOOK",
		"EPILOGUE",
		"PROLOGUE",
	}

	for _, pattern := range patterns {
		if strings.HasPrefix(upper, pattern) {
			return line
		}
	}

	// Roman numerals alone (I, II, III, IV, V, etc.)
	if isRomanNumeral(line) && len(line) <= 10 {
		return line
	}

	return ""
}

// isRomanNumeral checks if a string is a valid Roman numeral.
func isRomanNumeral(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	validChars := "IVXLCDMivxlcdm."
	for _, c := range s {
		if !strings.ContainsRune(validChars, c) && !unicode.IsSpace(c) {
			return false
		}
	}
	return true
}

// countWords counts words in text.
func countWords(text string) int {
	return len(strings.Fields(text))
}

// countNewlines counts newlines in text.
func countNewlines(text string) int {
	return strings.Count(text, "\n")
}

// findBreakPoint finds a good place to break the text.
func findBreakPoint(text string, targetWords, overlapWords int) int {
	// Try to break at paragraph boundary (double newline)
	targetChars := estimateChars(targetWords - overlapWords)
	if targetChars >= len(text) {
		return len(text)
	}

	// Look for double newline near target
	searchStart := max(0, targetChars-500)
	searchEnd := min(len(text), targetChars+500)
	searchArea := text[searchStart:searchEnd]

	// Find paragraph break
	if idx := strings.LastIndex(searchArea, "\n\n"); idx != -1 {
		return searchStart + idx + 2
	}

	// Fall back to single newline
	if idx := strings.LastIndex(searchArea, "\n"); idx != -1 {
		return searchStart + idx + 1
	}

	// Last resort: break at space
	if idx := strings.LastIndex(searchArea, " "); idx != -1 {
		return searchStart + idx + 1
	}

	return targetChars
}

// estimateChars estimates character count from word count.
func estimateChars(words int) int {
	// Average English word is ~5 chars + space
	return words * 6
}
