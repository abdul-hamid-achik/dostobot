package matcher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abdulachik/dostobot/internal/db"
	"github.com/abdulachik/dostobot/internal/extractor"
)

// Selector uses Claude to evaluate quote-trend matches.
type Selector struct {
	claude *extractor.ClaudeClient
}

// SelectorConfig holds configuration for the selector.
type SelectorConfig struct {
	APIKey string
}

// NewSelector creates a new selector.
func NewSelector(cfg SelectorConfig) *Selector {
	return &Selector{
		claude: extractor.NewClaudeClient(extractor.ClaudeConfig{
			APIKey: cfg.APIKey,
		}),
	}
}

// SelectionResult contains the evaluation of a quote-trend match.
type SelectionResult struct {
	RelevanceScore float64
	Reasoning      string
	Concerns       []string
	Recommendation string // "post" or "skip"
}

// Evaluate evaluates a single quote against a trend.
func (s *Selector) Evaluate(ctx context.Context, trend *db.Trend, quote *db.Quote) (*SelectionResult, error) {
	description := ""
	if trend.Description.Valid {
		description = trend.Description.String
	}

	prompt := fmt.Sprintf(SelectionPrompt,
		trend.Title,
		description,
		quote.Text,
		quote.SourceBook,
		quote.Themes,
	)

	response, err := s.claude.Complete(ctx, SelectionSystemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("claude complete: %w", err)
	}

	// Parse JSON response
	var result struct {
		RelevanceScore float64  `json:"relevance_score"`
		Reasoning      string   `json:"reasoning"`
		Concerns       []string `json:"concerns"`
		Recommendation string   `json:"recommendation"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Try to extract JSON from response
		jsonStr := extractJSON(response)
		if jsonStr == "" {
			return nil, fmt.Errorf("parse response: %w", err)
		}
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			return nil, fmt.Errorf("parse extracted json: %w", err)
		}
	}

	return &SelectionResult{
		RelevanceScore: result.RelevanceScore,
		Reasoning:      result.Reasoning,
		Concerns:       result.Concerns,
		Recommendation: result.Recommendation,
	}, nil
}

// BatchEvaluationResult contains the evaluation of multiple quotes.
type BatchEvaluationResult struct {
	BestMatchIndex int
	Evaluations    []QuoteEvaluation
	Recommendation string
}

// QuoteEvaluation contains the evaluation of a single quote in a batch.
type QuoteEvaluation struct {
	Index     int
	Score     float64
	Reasoning string
}

// EvaluateBatch evaluates multiple quotes against a trend.
func (s *Selector) EvaluateBatch(ctx context.Context, trend *db.Trend, quotes []*db.Quote) (*BatchEvaluationResult, error) {
	if len(quotes) == 0 {
		return &BatchEvaluationResult{BestMatchIndex: -1}, nil
	}

	description := ""
	if trend.Description.Valid {
		description = trend.Description.String
	}

	// Build quotes list for prompt
	var quotesList strings.Builder
	for i, q := range quotes {
		quotesList.WriteString(fmt.Sprintf("\n%d. \"%s\"\n   — From %s\n   Themes: %s\n",
			i+1, q.Text, q.SourceBook, q.Themes))
	}

	prompt := fmt.Sprintf(BatchSelectionPrompt,
		trend.Title,
		description,
		quotesList.String(),
	)

	response, err := s.claude.Complete(ctx, SelectionSystemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("claude complete: %w", err)
	}

	// Parse JSON response
	var result struct {
		BestMatchIndex int `json:"best_match_index"`
		Evaluations    []struct {
			Index     int     `json:"index"`
			Score     float64 `json:"score"`
			Reasoning string  `json:"reasoning"`
		} `json:"evaluations"`
		Recommendation string `json:"recommendation"`
	}

	jsonStr := extractJSON(response)
	if jsonStr == "" {
		jsonStr = response
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	evals := make([]QuoteEvaluation, len(result.Evaluations))
	for i, e := range result.Evaluations {
		evals[i] = QuoteEvaluation{
			Index:     e.Index,
			Score:     e.Score,
			Reasoning: e.Reasoning,
		}
	}

	return &BatchEvaluationResult{
		BestMatchIndex: result.BestMatchIndex,
		Evaluations:    evals,
		Recommendation: result.Recommendation,
	}, nil
}

// extractJSON finds and extracts a JSON object from text.
func extractJSON(text string) string {
	// Find opening brace
	start := strings.Index(text, "{")
	if start == -1 {
		return ""
	}

	// Find matching closing brace
	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}

	return ""
}
