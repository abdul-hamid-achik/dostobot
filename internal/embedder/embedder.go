package embedder

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"time"
)

const (
	defaultModel    = "nomic-embed-text"
	embeddingDim    = 768
	defaultBatchSize = 10
)

// Embedder generates embeddings using Ollama.
type Embedder struct {
	host       string
	model      string
	httpClient *http.Client
}

// Config holds configuration for the embedder.
type Config struct {
	Host  string
	Model string
}

// New creates a new Embedder.
func New(cfg Config) *Embedder {
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}

	return &Embedder{
		host:  cfg.Host,
		model: model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ollamaRequest is the request body for Ollama embedding API.
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaResponse is the response from Ollama embedding API.
type ollamaResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Embed generates an embedding for the given text.
func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	req := ollamaRequest{
		Model:  e.model,
		Prompt: text,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/embeddings", e.host)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(ollamaResp.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	// Convert float64 to float32
	embedding := make([]float32, len(ollamaResp.Embedding))
	for i, v := range ollamaResp.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// EmbedBatch generates embeddings for multiple texts.
func (e *Embedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		embedding, err := e.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("embed text %d: %w", i, err)
		}
		embeddings[i] = embedding

		// Small delay to avoid overwhelming Ollama
		if i < len(texts)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return embeddings, nil
}

// Ping checks if Ollama is available and has the required model.
func (e *Embedder) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/tags", e.host)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	// Check if the model is available
	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return fmt.Errorf("decode tags response: %w", err)
	}

	for _, model := range tagsResp.Models {
		if model.Name == e.model || model.Name == e.model+":latest" {
			slog.Debug("found embedding model", "model", model.Name)
			return nil
		}
	}

	return fmt.Errorf("model %s not found in Ollama (run: ollama pull %s)", e.model, e.model)
}

// EmbeddingToBytes converts an embedding to bytes for storage.
func EmbeddingToBytes(embedding []float32) []byte {
	buf := new(bytes.Buffer)
	for _, v := range embedding {
		binary.Write(buf, binary.LittleEndian, v)
	}
	return buf.Bytes()
}

// BytesToEmbedding converts bytes back to an embedding.
func BytesToEmbedding(data []byte) ([]float32, error) {
	if len(data) == 0 {
		return nil, nil
	}

	if len(data)%4 != 0 {
		return nil, fmt.Errorf("invalid embedding data length: %d", len(data))
	}

	embedding := make([]float32, len(data)/4)
	reader := bytes.NewReader(data)
	for i := range embedding {
		if err := binary.Read(reader, binary.LittleEndian, &embedding[i]); err != nil {
			return nil, fmt.Errorf("read float: %w", err)
		}
	}

	return embedding, nil
}

// CosineSimilarity computes the cosine similarity between two embeddings.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}

// Normalize normalizes an embedding to unit length.
func Normalize(embedding []float32) []float32 {
	var norm float64
	for _, v := range embedding {
		norm += float64(v) * float64(v)
	}

	if norm == 0 {
		return embedding
	}

	norm = math.Sqrt(norm)
	result := make([]float32, len(embedding))
	for i, v := range embedding {
		result[i] = float32(float64(v) / norm)
	}

	return result
}
