package rag

import (
	"context"
	"testing"

	"github.com/philippgille/chromem-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRetrievalDB struct {
	collection retrievalCollection
}

func (f fakeRetrievalDB) GetCollection(_ string, _ chromem.EmbeddingFunc) retrievalCollection {
	return f.collection
}

type fakeRetrievalCollection struct {
	baseResults     []chromem.Result
	neighborResults map[string]chromem.Result
}

func fakeEmbeddingFactory(_ context.Context) (chromem.EmbeddingFunc, func(), error) {
	embed := func(_ context.Context, _ string) ([]float32, error) {
		return []float32{1}, nil
	}
	return embed, func() {}, nil
}

func (f fakeRetrievalCollection) QueryEmbedding(_ context.Context, _ []float32, _ int, where, _ map[string]string) ([]chromem.Result, error) {
	if where == nil {
		return f.baseResults, nil
	}

	idx, ok := where["chunk_index"]
	if !ok {
		return nil, nil
	}

	result, ok := f.neighborResults[idx]
	if !ok {
		return nil, nil
	}

	return []chromem.Result{result}, nil
}

func TestRetrieveStyleGuideContext(t *testing.T) {
	t.Parallel()

	t.Run("no results returns empty", func(t *testing.T) {
		t.Parallel()

		retriever, err := newRetriever(context.Background(), fakeRetrievalDB{collection: fakeRetrievalCollection{}}, fakeEmbeddingFactory)
		require.NoError(t, err)
		defer retriever.Close()

		chunks, err := retriever.RetrieveStyleGuideContext(context.Background(), "error wrapping")
		require.NoError(t, err)
		require.Len(t, chunks, 0)
	})

	t.Run("missing source metadata returns error", func(t *testing.T) {
		t.Parallel()

		retriever, err := newRetriever(context.Background(), fakeRetrievalDB{
			collection: fakeRetrievalCollection{
				baseResults: []chromem.Result{
					{
						ID:         "guide-1",
						Content:    "# Errors\nUse wrapping",
						Similarity: 0.9,
						Metadata: map[string]string{
							"chunk_index": "1",
						},
					},
				},
			},
		}, fakeEmbeddingFactory)
		require.NoError(t, err)
		defer retriever.Close()

		_, err = retriever.RetrieveStyleGuideContext(context.Background(), "error wrapping")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing source metadata")
	})

	t.Run("includes neighbors and keeps chunk order", func(t *testing.T) {
		t.Parallel()

		base := chromem.Result{
			ID:         "guide-2",
			Content:    "## Errors\nwrap errors",
			Similarity: 0.8,
			Metadata: map[string]string{
				"source":      "styleguides/disco/style.md",
				"chunk_index": "2",
			},
		}

		neighbors := map[string]chromem.Result{
			"1": {
				ID:         "guide-1",
				Content:    "# Errors",
				Similarity: 0.5,
				Metadata: map[string]string{
					"source":      "styleguides/disco/style.md",
					"chunk_index": "1",
				},
			},
			"3": {
				ID:         "guide-3",
				Content:    "### Wrapping",
				Similarity: 0.7,
				Metadata: map[string]string{
					"source":      "styleguides/disco/style.md",
					"chunk_index": "3",
				},
			},
			"2": base,
		}

		retriever, err := newRetriever(
			context.Background(),
			fakeRetrievalDB{collection: fakeRetrievalCollection{baseResults: []chromem.Result{base}, neighborResults: neighbors}},
			fakeEmbeddingFactory,
		)
		require.NoError(t, err)
		defer retriever.Close()

		chunks, err := retriever.RetrieveStyleGuideContext(context.Background(), "error wrapping")
		require.NoError(t, err)
		require.Len(t, chunks, 3)
		assert.Equal(t, 1, chunks[0].ChunkIndex)
		assert.Equal(t, 2, chunks[1].ChunkIndex)
		assert.Equal(t, 3, chunks[2].ChunkIndex)
	})

	t.Run("missing collection returns sentinel error", func(t *testing.T) {
		t.Parallel()

		_, err := newRetriever(context.Background(), fakeRetrievalDB{collection: nil}, fakeEmbeddingFactory)
		require.ErrorIs(t, err, ErrStyleGuideCollectionMissing)
	})
}
