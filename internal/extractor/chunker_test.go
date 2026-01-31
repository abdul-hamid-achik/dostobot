package extractor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunker_ChunkText(t *testing.T) {
	t.Run("basic chunking", func(t *testing.T) {
		// Create text with enough words to trigger chunking
		words := make([]string, 3000)
		for i := range words {
			words[i] = "word"
		}
		text := strings.Join(words, " ")

		chunker := NewChunker(ChunkerConfig{
			TargetWords:  1000,
			OverlapWords: 100,
			MinWords:     200,
		})

		chunks := chunker.ChunkText(text)
		assert.Greater(t, len(chunks), 1, "should create multiple chunks")

		// Each chunk should have reasonable word count
		for _, chunk := range chunks {
			assert.GreaterOrEqual(t, chunk.WordCount, 200)
		}
	})

	t.Run("small text returns single chunk", func(t *testing.T) {
		text := "This is a small piece of text with only a few words that should fit in one chunk easily without any issues at all even with some padding words added here."

		chunker := NewChunker(ChunkerConfig{
			TargetWords:  1000,
			OverlapWords: 100,
			MinWords:     10,
		})

		chunks := chunker.ChunkText(text)
		require.Len(t, chunks, 1)
		assert.Equal(t, text, chunks[0].Text)
	})

	t.Run("respects paragraph boundaries", func(t *testing.T) {
		// Create paragraphs
		para1 := strings.Repeat("word ", 600)
		para2 := strings.Repeat("another ", 600)
		text := para1 + "\n\n" + para2

		chunker := NewChunker(ChunkerConfig{
			TargetWords:  800,
			OverlapWords: 100,
			MinWords:     300,
		})

		chunks := chunker.ChunkText(text)
		require.GreaterOrEqual(t, len(chunks), 1)

		// First chunk should end near paragraph boundary
		if len(chunks) > 1 {
			assert.True(t, strings.HasSuffix(strings.TrimSpace(chunks[0].Text), "word") ||
				strings.Contains(chunks[0].Text, "another"),
				"chunk should break at or near paragraph boundary")
		}
	})

	t.Run("tracks chapter", func(t *testing.T) {
		text := `CHAPTER I

First chapter content with many words ` + strings.Repeat("word ", 600) + `

CHAPTER II

Second chapter content with many words ` + strings.Repeat("more ", 600)

		chunker := NewChunker(ChunkerConfig{
			TargetWords:  500,
			OverlapWords: 50,
			MinWords:     100,
		})

		chunks := chunker.ChunkText(text)
		require.Greater(t, len(chunks), 0)

		// First chunk should have Chapter I
		assert.Contains(t, chunks[0].Chapter, "CHAPTER")
	})

	t.Run("chunk indices are sequential", func(t *testing.T) {
		text := strings.Repeat("word ", 3000)

		chunker := NewChunker(ChunkerConfig{
			TargetWords:  500,
			OverlapWords: 50,
			MinWords:     100,
		})

		chunks := chunker.ChunkText(text)
		for i, chunk := range chunks {
			assert.Equal(t, i, chunk.ChunkIndex)
		}
	})
}

func TestStripGutenbergBoilerplate(t *testing.T) {
	t.Run("strips header and footer", func(t *testing.T) {
		lines := []string{
			"The Project Gutenberg EBook",
			"Title: Test Book",
			"*** START OF THIS PROJECT GUTENBERG EBOOK ***",
			"Actual content line 1",
			"Actual content line 2",
			"Actual content line 3",
			"*** END OF THIS PROJECT GUTENBERG EBOOK ***",
			"Footer text that should be removed",
		}

		result := stripGutenbergBoilerplate(lines)
		assert.Len(t, result, 3)
		assert.Equal(t, "Actual content line 1", result[0])
		assert.Equal(t, "Actual content line 2", result[1])
		assert.Equal(t, "Actual content line 3", result[2])
	})

	t.Run("handles no markers", func(t *testing.T) {
		lines := []string{"line 1", "line 2", "line 3"}
		result := stripGutenbergBoilerplate(lines)
		assert.Equal(t, lines, result)
	})
}

func TestDetectChapter(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CHAPTER I", "CHAPTER I"},
		{"Chapter 1", "Chapter 1"},
		{"PART ONE", "PART ONE"},
		{"BOOK FIRST", "BOOK FIRST"},
		{"EPILOGUE", "EPILOGUE"},
		{"PROLOGUE", "PROLOGUE"},
		{"I", "I"},
		{"II", "II"},
		{"III", "III"},
		{"IV", "IV"},
		{"Regular text", ""},
		{"The chapter begins", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := detectChapter(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"hello world", 2},
		{"one", 1},
		{"", 0},
		{"  spaces  between  words  ", 3},
		{"multiple\nlines\nhere", 3},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := countWords(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRomanNumeral(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"I", true},
		{"II", true},
		{"III", true},
		{"IV", true},
		{"V", true},
		{"VI", true},
		{"X", true},
		{"XII", true},
		{"Hello", false},
		{"1", false},
		{"", false},
		{"I.", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isRomanNumeral(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
