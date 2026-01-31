package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("uses default model", func(t *testing.T) {
		e := New(Config{Host: "http://localhost:11434"})
		assert.Equal(t, defaultModel, e.model)
	})

	t.Run("uses custom model", func(t *testing.T) {
		e := New(Config{
			Host:  "http://localhost:11434",
			Model: "custom-model",
		})
		assert.Equal(t, "custom-model", e.model)
	})
}

func TestEmbedder_Embed(t *testing.T) {
	t.Run("successful embedding", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/embeddings", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			var req ollamaRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "test text", req.Prompt)

			// Return a fake embedding
			embedding := make([]float64, 768)
			for i := range embedding {
				embedding[i] = float64(i) / 768.0
			}

			json.NewEncoder(w).Encode(ollamaResponse{Embedding: embedding})
		}))
		defer server.Close()

		e := New(Config{Host: server.URL})
		embedding, err := e.Embed(context.Background(), "test text")

		require.NoError(t, err)
		assert.Len(t, embedding, 768)
	})

	t.Run("handles error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		}))
		defer server.Close()

		e := New(Config{Host: server.URL})
		_, err := e.Embed(context.Background(), "test text")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 500")
	})

	t.Run("handles empty embedding", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(ollamaResponse{Embedding: []float64{}})
		}))
		defer server.Close()

		e := New(Config{Host: server.URL})
		_, err := e.Embed(context.Background(), "test text")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty embedding")
	})
}

func TestEmbedder_Ping(t *testing.T) {
	t.Run("model found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/tags", r.URL.Path)

			response := struct {
				Models []struct {
					Name string `json:"name"`
				} `json:"models"`
			}{
				Models: []struct {
					Name string `json:"name"`
				}{
					{Name: "nomic-embed-text:latest"},
					{Name: "llama2:latest"},
				},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		e := New(Config{Host: server.URL})
		err := e.Ping(context.Background())

		assert.NoError(t, err)
	})

	t.Run("model not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := struct {
				Models []struct {
					Name string `json:"name"`
				} `json:"models"`
			}{
				Models: []struct {
					Name string `json:"name"`
				}{
					{Name: "llama2:latest"},
				},
			}
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		e := New(Config{Host: server.URL})
		err := e.Ping(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestEmbeddingToBytes(t *testing.T) {
	embedding := []float32{1.0, 2.0, 3.0, 4.0}
	data := EmbeddingToBytes(embedding)

	assert.Len(t, data, 16) // 4 floats × 4 bytes
}

func TestBytesToEmbedding(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		original := []float32{1.0, 2.5, 3.7, 4.2}
		data := EmbeddingToBytes(original)
		result, err := BytesToEmbedding(data)

		require.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("empty data", func(t *testing.T) {
		result, err := BytesToEmbedding([]byte{})
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("invalid length", func(t *testing.T) {
		_, err := BytesToEmbedding([]byte{1, 2, 3}) // Not divisible by 4
		assert.Error(t, err)
	})
}

func TestCosineSimilarity(t *testing.T) {
	t.Run("identical vectors", func(t *testing.T) {
		a := []float32{1, 2, 3}
		b := []float32{1, 2, 3}
		sim := CosineSimilarity(a, b)
		assert.InDelta(t, 1.0, float64(sim), 0.0001)
	})

	t.Run("orthogonal vectors", func(t *testing.T) {
		a := []float32{1, 0, 0}
		b := []float32{0, 1, 0}
		sim := CosineSimilarity(a, b)
		assert.InDelta(t, 0.0, float64(sim), 0.0001)
	})

	t.Run("opposite vectors", func(t *testing.T) {
		a := []float32{1, 0, 0}
		b := []float32{-1, 0, 0}
		sim := CosineSimilarity(a, b)
		assert.InDelta(t, -1.0, float64(sim), 0.0001)
	})

	t.Run("different lengths", func(t *testing.T) {
		a := []float32{1, 2}
		b := []float32{1, 2, 3}
		sim := CosineSimilarity(a, b)
		assert.Equal(t, float32(0), sim)
	})

	t.Run("zero vector", func(t *testing.T) {
		a := []float32{0, 0, 0}
		b := []float32{1, 2, 3}
		sim := CosineSimilarity(a, b)
		assert.Equal(t, float32(0), sim)
	})
}

func TestNormalize(t *testing.T) {
	t.Run("normalizes to unit length", func(t *testing.T) {
		v := []float32{3, 4}
		normalized := Normalize(v)

		// Check length is 1
		var length float64
		for _, x := range normalized {
			length += float64(x) * float64(x)
		}
		assert.InDelta(t, 1.0, math.Sqrt(length), 0.0001)
	})

	t.Run("zero vector unchanged", func(t *testing.T) {
		v := []float32{0, 0, 0}
		normalized := Normalize(v)
		assert.Equal(t, v, normalized)
	})
}

func BenchmarkCosineSimilarity(b *testing.B) {
	// Create realistic 768-dimensional embeddings
	a := make([]float32, 768)
	bb := make([]float32, 768)
	for i := range a {
		a[i] = float32(i) / 768.0
		bb[i] = float32(768-i) / 768.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CosineSimilarity(a, bb)
	}
}

func BenchmarkEmbeddingRoundTrip(b *testing.B) {
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = float32(i) / 768.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := EmbeddingToBytes(embedding)
		BytesToEmbedding(data)
	}
}

// Test that our binary format is compatible with reading
func TestBinaryFormatCompatibility(t *testing.T) {
	original := []float32{1.5, -2.5, 3.14159, 0.0, -0.0}
	data := EmbeddingToBytes(original)

	// Verify we can read it back using standard library
	reader := bytes.NewReader(data)
	for _, expected := range original {
		buf := make([]byte, 4)
		_, readErr := reader.Read(buf)
		require.NoError(t, readErr)

		// Manual float32 parsing (little endian)
		bits := uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
		val := math.Float32frombits(bits)

		assert.Equal(t, expected, val)
	}
}
