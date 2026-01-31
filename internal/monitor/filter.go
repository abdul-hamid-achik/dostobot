package monitor

import (
	"strings"
)

// SensitiveTopics that should be filtered out to avoid controversy.
var SensitiveTopics = []string{
	// Political figures (too divisive)
	"trump", "biden", "obama", "clinton", "putin", "xi jinping",
	"maga", "democrat", "republican", "liberal", "conservative",

	// Hot-button political issues
	"abortion", "pro-life", "pro-choice",
	"gun control", "second amendment", "2nd amendment",
	"immigration", "border", "deportation",
	"lgbtq", "transgender", "gay rights",

	// Tragedy/violence
	"shooting", "massacre", "terrorist", "terrorism",
	"murder", "killed", "death toll", "casualties",
	"suicide", "self-harm",

	// Religion (avoid proselytizing appearance)
	"atheism", "christian", "muslim", "jewish", "religion debate",

	// Explicit content
	"nsfw", "porn", "sex", "nude",

	// Hate speech related
	"racist", "racism", "nazi", "white supremac", "hate crime",

	// Current wars/conflicts
	"ukraine", "russia war", "gaza", "israel", "hamas",

	// Conspiracy theories
	"qanon", "deep state", "illuminati", "flat earth",
	"anti-vax", "plandemic",
}

// Filter checks trends for sensitive content.
type Filter struct {
	sensitiveTerms []string
	minScore       int
}

// FilterConfig holds filter configuration.
type FilterConfig struct {
	AdditionalTerms []string
	MinScore        int
}

// NewFilter creates a new filter.
func NewFilter(cfg FilterConfig) *Filter {
	terms := make([]string, len(SensitiveTopics))
	copy(terms, SensitiveTopics)

	// Add any additional terms
	terms = append(terms, cfg.AdditionalTerms...)

	// Lowercase all terms for case-insensitive matching
	for i, term := range terms {
		terms[i] = strings.ToLower(term)
	}

	return &Filter{
		sensitiveTerms: terms,
		minScore:       cfg.MinScore,
	}
}

// FilterResult contains the filter decision.
type FilterResult struct {
	Pass   bool
	Reason string
}

// Check examines a trend and returns whether it should be processed.
func (f *Filter) Check(trend Trend) FilterResult {
	// Check minimum score
	if f.minScore > 0 && trend.Score < f.minScore {
		return FilterResult{
			Pass:   false,
			Reason: "score below threshold",
		}
	}

	// Check for sensitive content
	text := strings.ToLower(trend.Title + " " + trend.Description)

	for _, term := range f.sensitiveTerms {
		if strings.Contains(text, term) {
			return FilterResult{
				Pass:   false,
				Reason: "contains sensitive topic: " + term,
			}
		}
	}

	return FilterResult{Pass: true}
}

// FilterTrends filters a list of trends, returning only those that pass.
func (f *Filter) FilterTrends(trends []Trend) []Trend {
	result := make([]Trend, 0, len(trends))

	for _, trend := range trends {
		if check := f.Check(trend); check.Pass {
			result = append(result, trend)
		}
	}

	return result
}
