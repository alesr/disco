package rag

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmbeddingFunc(t *testing.T) {
	t.Run("mistral provider requires api key", func(t *testing.T) {
		t.Setenv("EMBEDDING_PROVIDER", providerMistral)
		t.Setenv("MISTRAL_API_KEY", "")

		_, _, err := newEmbeddingFunc(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MISTRAL_API_KEY")
	})

	t.Run("unsupported provider returns error", func(t *testing.T) {
		t.Setenv("EMBEDDING_PROVIDER", "bogus")

		_, _, err := newEmbeddingFunc(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported embedding provider")
	})
}

func TestNewMistralEmbeddingFunc(t *testing.T) {
	t.Parallel()

	t.Run("creates normalized vector from mistral api", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "/v1/embeddings", r.URL.Path)
			require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

			var payload map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			require.Equal(t, "mistral-embed", payload["model"])
			require.Equal(t, "query text", payload["input"])

			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"data":[{"embedding":[3,4]}]}`))
			require.NoError(t, err)
		}))
		defer server.Close()

		embeddingFunc, cleanup, err := newMistralEmbeddingFunc(context.Background(), embeddingProviderConfig{
			Provider:          providerMistral,
			MistralAPIKey:     "test-key",
			MistralEmbedModel: defaultMistralModel,
			MistralBaseURL:    server.URL,
		})
		require.NoError(t, err)
		defer cleanup()

		vector, err := embeddingFunc(context.Background(), "query text")
		require.NoError(t, err)
		require.Len(t, vector, 2)
		assert.InDelta(t, 0.6, vector[0], 1e-5)
		assert.InDelta(t, 0.8, vector[1], 1e-5)
	})

	t.Run("returns error for non-success status", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, err := w.Write([]byte(`{"error":"invalid key"}`))
			require.NoError(t, err)
		}))
		defer server.Close()

		embeddingFunc, cleanup, err := newMistralEmbeddingFunc(context.Background(), embeddingProviderConfig{
			Provider:       providerMistral,
			MistralAPIKey:  "bad",
			MistralBaseURL: server.URL,
		})
		require.NoError(t, err)
		defer cleanup()

		_, err = embeddingFunc(context.Background(), "query text")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 401")
	})
}
