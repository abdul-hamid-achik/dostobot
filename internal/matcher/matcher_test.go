package matcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean json",
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "json with preamble",
			input:    `Here is the result:\n{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			name:     "json with suffix",
			input:    `{"key": "value"}\n\nHope this helps!`,
			expected: `{"key": "value"}`,
		},
		{
			name: "nested json",
			input: `{"outer": {"inner": "value"}}`,
			expected: `{"outer": {"inner": "value"}}`,
		},
		{
			name:     "no json",
			input:    "Just plain text",
			expected: "",
		},
		{
			name:     "incomplete json",
			input:    `{"key": "value"`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNew(t *testing.T) {
	m := New(Config{
		APIKey: "test-key",
	})

	assert.NotNil(t, m)
	assert.Equal(t, float32(0.5), m.minSimilarity)
	assert.Equal(t, float64(0.6), m.minRelevance)
	assert.Equal(t, 10, m.candidateCount)
}

func TestNew_CustomConfig(t *testing.T) {
	m := New(Config{
		APIKey:         "test-key",
		MinSimilarity:  0.7,
		MinRelevance:   0.8,
		CandidateCount: 20,
	})

	assert.Equal(t, float32(0.7), m.minSimilarity)
	assert.Equal(t, float64(0.8), m.minRelevance)
	assert.Equal(t, 20, m.candidateCount)
}
