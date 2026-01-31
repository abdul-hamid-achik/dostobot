package poster

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// BlueskyMaxLength is the maximum character count for a Bluesky post.
	BlueskyMaxLength = 300

	// TwitterMaxLength is the maximum character count for a Twitter post.
	TwitterMaxLength = 280
)

// FormatQuote formats a quote for posting.
func FormatQuote(quoteText, sourceBook, author string) string {
	// Build attribution
	attribution := fmt.Sprintf("— %s", sourceBook)
	if author != "" && author != "Narrator" {
		attribution = fmt.Sprintf("— %s, %s", author, sourceBook)
	}

	// Format: "Quote text"\n\n— Attribution
	return fmt.Sprintf("\"%s\"\n\n%s", quoteText, attribution)
}

// FormatWithTrend formats a quote with trend context (optional).
func FormatWithTrend(quoteText, sourceBook, author, trendTitle string, includeTrend bool) string {
	base := FormatQuote(quoteText, sourceBook, author)

	if includeTrend && trendTitle != "" {
		// Add subtle trend reference
		return fmt.Sprintf("%s\n\n#Dostoyevsky", base)
	}

	return base
}

// TruncateQuote truncates a quote to fit within a character limit.
func TruncateQuote(quote string, maxLen int, attribution string) string {
	// Calculate available space for quote
	// Format: "Quote text..."\n\n— Attribution
	overhead := 4 + len(attribution) + 4 // quotes (2) + ellipsis (3) + newlines (2) + em dash (2) + space (1)
	available := maxLen - overhead

	if utf8.RuneCountInString(quote) <= available+3 { // +3 because we don't need ellipsis
		return quote
	}

	// Truncate at word boundary
	runes := []rune(quote)
	if len(runes) <= available {
		return quote
	}

	truncated := string(runes[:available])

	// Find last space to avoid cutting mid-word
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > available/2 { // Only use word boundary if not too far back
		truncated = truncated[:lastSpace]
	}

	return strings.TrimRight(truncated, " .,;:!?") + "..."
}

// FitsInLimit checks if the formatted post fits within the limit.
func FitsInLimit(formatted string, limit int) bool {
	return utf8.RuneCountInString(formatted) <= limit
}

// SplitLongQuote splits a quote that's too long into multiple posts.
// Returns nil if the quote fits in one post.
func SplitLongQuote(quoteText, sourceBook, author string, limit int) []string {
	full := FormatQuote(quoteText, sourceBook, author)
	if FitsInLimit(full, limit) {
		return nil
	}

	// For very long quotes, split into parts
	attribution := fmt.Sprintf("— %s", sourceBook)
	if author != "" && author != "Narrator" {
		attribution = fmt.Sprintf("— %s, %s", author, sourceBook)
	}

	// Calculate how much text we can fit per post
	// Part 1: "Quote text...(1/2)
	// Part 2: ...continuation"\n\n— Attribution
	partIndicatorLen := 10 // " (1/2)" or similar
	quoteOverhead := 4     // quotes + space

	firstPartMax := limit - quoteOverhead - partIndicatorLen
	lastPartOverhead := quoteOverhead + partIndicatorLen + 4 + len(attribution) // newlines + attribution

	words := strings.Fields(quoteText)
	var parts []string
	var currentPart strings.Builder
	partNum := 1

	for i, word := range words {
		testAdd := word
		if currentPart.Len() > 0 {
			testAdd = " " + word
		}

		// Check if adding this word exceeds limit
		isLastWord := i == len(words)-1
		maxForThisPart := firstPartMax
		if isLastWord || (currentPart.Len()+len(testAdd) > firstPartMax) {
			// This might be the end of a part
			maxForThisPart = limit - lastPartOverhead
		}

		if currentPart.Len()+len(testAdd) > maxForThisPart && currentPart.Len() > 0 {
			// Save current part and start new one
			parts = append(parts, currentPart.String())
			currentPart.Reset()
			partNum++
			currentPart.WriteString(word)
		} else {
			if currentPart.Len() > 0 {
				currentPart.WriteString(" ")
			}
			currentPart.WriteString(word)
		}
	}

	// Add final part
	if currentPart.Len() > 0 {
		parts = append(parts, currentPart.String())
	}

	// Format parts with indicators
	totalParts := len(parts)
	formatted := make([]string, totalParts)

	for i, part := range parts {
		indicator := fmt.Sprintf(" (%d/%d)", i+1, totalParts)
		if i == 0 {
			formatted[i] = fmt.Sprintf("\"%s...%s", part, indicator)
		} else if i == totalParts-1 {
			formatted[i] = fmt.Sprintf("...%s\"%s\n\n%s", part, indicator, attribution)
		} else {
			formatted[i] = fmt.Sprintf("...%s...%s", part, indicator)
		}
	}

	return formatted
}
