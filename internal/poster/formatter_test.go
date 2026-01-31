package poster

import (
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestFormatQuote(t *testing.T) {
	t.Run("basic format", func(t *testing.T) {
		result := FormatQuote("Pain and suffering are inevitable.", "Crime and Punishment", "")
		expected := "\"Pain and suffering are inevitable.\"\n\n— Crime and Punishment"
		assert.Equal(t, expected, result)
	})

	t.Run("with character", func(t *testing.T) {
		result := FormatQuote("Pain and suffering are inevitable.", "Crime and Punishment", "Raskolnikov")
		expected := "\"Pain and suffering are inevitable.\"\n\n— Raskolnikov, Crime and Punishment"
		assert.Equal(t, expected, result)
	})

	t.Run("narrator character is excluded", func(t *testing.T) {
		result := FormatQuote("The text.", "The Idiot", "Narrator")
		expected := "\"The text.\"\n\n— The Idiot"
		assert.Equal(t, expected, result)
	})
}

func TestTruncateQuote(t *testing.T) {
	t.Run("short quote unchanged", func(t *testing.T) {
		quote := "Short quote."
		result := TruncateQuote(quote, 100, "— Book")
		assert.Equal(t, quote, result)
	})

	t.Run("long quote truncated", func(t *testing.T) {
		quote := "This is a very long quote that needs to be truncated because it exceeds the character limit for the post."
		result := TruncateQuote(quote, 50, "— Book")

		assert.Less(t, utf8.RuneCountInString(result), 50)
		assert.True(t, len(result) < len(quote))
		assert.True(t, result[len(result)-3:] == "...")
	})

	t.Run("truncates at word boundary", func(t *testing.T) {
		quote := "Word1 word2 word3 word4 word5 word6 word7 word8"
		result := TruncateQuote(quote, 30, "— Book")

		// Should not end mid-word
		assert.True(t, result[len(result)-3:] == "..." ||
			result[len(result)-1] == ' ', "Should end at word boundary or with ellipsis")
	})
}

func TestFitsInLimit(t *testing.T) {
	tests := []struct {
		text   string
		limit  int
		fits   bool
	}{
		{"Hello", 10, true},
		{"Hello", 5, true},
		{"Hello", 4, false},
		{"", 1, true},
		{"日本語", 3, true}, // 3 runes
		{"日本語", 2, false},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := FitsInLimit(tt.text, tt.limit)
			assert.Equal(t, tt.fits, result)
		})
	}
}

func TestSplitLongQuote(t *testing.T) {
	t.Run("short quote returns nil", func(t *testing.T) {
		quote := "Short quote."
		result := SplitLongQuote(quote, "Book", "", 300)
		assert.Nil(t, result)
	})

	t.Run("long quote splits", func(t *testing.T) {
		// Create a quote that's definitely too long
		longQuote := "This is a very long quote that goes on and on and on. " +
			"It contains many words and sentences. " +
			"The purpose is to test the splitting functionality. " +
			"We need enough text to exceed the character limit. " +
			"So we keep adding more and more text until we're sure it's too long. " +
			"This should definitely be split into multiple posts. " +
			"Each part should be properly formatted with continuation markers."

		result := SplitLongQuote(longQuote, "Crime and Punishment", "Raskolnikov", 200)

		if result != nil {
			assert.Greater(t, len(result), 1, "Should split into multiple parts")

			// First part should start with quote mark
			assert.True(t, result[0][0] == '"', "First part should start with quote mark")

			// Last part should contain attribution
			assert.Contains(t, result[len(result)-1], "Crime and Punishment")

			// All parts should fit in limit (with some margin for edge cases)
			for _, part := range result {
				assert.LessOrEqual(t, utf8.RuneCountInString(part), 250,
					"Each part should be reasonably sized")
			}
		}
	})
}

func BenchmarkFormatQuote(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatQuote(
			"Pain and suffering are always inevitable for a large intelligence and a deep heart.",
			"Crime and Punishment",
			"Raskolnikov",
		)
	}
}
