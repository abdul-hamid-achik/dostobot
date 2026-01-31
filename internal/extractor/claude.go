package extractor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	claudeAPIURL     = "https://api.anthropic.com/v1/messages"
	claudeAPIVersion = "2023-06-01"
	defaultModel     = "claude-sonnet-4-20250514"
	maxTokens        = 4096
)

// ClaudeClient is a client for the Claude API.
type ClaudeClient struct {
	apiKey     string
	httpClient *http.Client
	model      string
}

// ClaudeConfig holds configuration for the Claude client.
type ClaudeConfig struct {
	APIKey string
	Model  string
}

// NewClaudeClient creates a new Claude API client.
func NewClaudeClient(config ClaudeConfig) *ClaudeClient {
	model := config.Model
	if model == "" {
		model = defaultModel
	}

	return &ClaudeClient{
		apiKey: config.APIKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		model: model,
	}
}

// Message represents a message in the conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeRequest is the request body for the Claude API.
type claudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
}

// claudeResponse is the response from the Claude API.
type claudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete sends a completion request to Claude.
func (c *ClaudeClient) Complete(ctx context.Context, system, user string) (string, error) {
	req := claudeRequest{
		Model:     c.model,
		MaxTokens: maxTokens,
		System:    system,
		Messages: []Message{
			{Role: "user", Content: user},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", claudeAPIVersion)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if claudeResp.Error != nil {
		return "", fmt.Errorf("API error: %s - %s", claudeResp.Error.Type, claudeResp.Error.Message)
	}

	if len(claudeResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return claudeResp.Content[0].Text, nil
}

// ExtractedQuote represents a quote extracted by Claude.
type ExtractedQuote struct {
	Text            string   `json:"text"`
	Character       string   `json:"character"`
	Themes          []string `json:"themes"`
	ModernRelevance string   `json:"modern_relevance"`
}

// ExtractQuotes extracts quotes from a text chunk using Claude.
func (c *ClaudeClient) ExtractQuotes(ctx context.Context, bookTitle, text string) ([]ExtractedQuote, error) {
	prompt := fmt.Sprintf(ExtractionPrompt, bookTitle, text)

	response, err := c.Complete(ctx, SystemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("complete: %w", err)
	}

	// Parse JSON response
	var quotes []ExtractedQuote
	if err := json.Unmarshal([]byte(response), &quotes); err != nil {
		// Try to extract JSON from response if it contains other text
		quotes, err = extractJSONFromResponse(response)
		if err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}
	}

	return quotes, nil
}

// extractJSONFromResponse tries to find and parse JSON array from a response that may contain other text.
func extractJSONFromResponse(response string) ([]ExtractedQuote, error) {
	// Find the start of JSON array
	start := -1
	for i, c := range response {
		if c == '[' {
			start = i
			break
		}
	}

	if start == -1 {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	// Find matching end bracket
	depth := 0
	end := -1
	for i := start; i < len(response); i++ {
		switch response[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
		if end != -1 {
			break
		}
	}

	if end == -1 {
		return nil, fmt.Errorf("malformed JSON array in response")
	}

	var quotes []ExtractedQuote
	if err := json.Unmarshal([]byte(response[start:end]), &quotes); err != nil {
		return nil, fmt.Errorf("parse extracted JSON: %w", err)
	}

	return quotes, nil
}
