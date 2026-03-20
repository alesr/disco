package rag

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/philippgille/chromem-go"
)

const (
	defaultTopK          = 6
	neighborWindow       = 1
	maxRetrievedContexts = 10
)

var ErrStyleGuideCollectionMissing = errors.New("style guide collection does not exist, run ingest first")

type (
	retrievalCollection interface {
		QueryEmbedding(ctx context.Context, queryEmbedding []float32, nResults int, where, whereDocument map[string]string) ([]chromem.Result, error)
	}

	retrievalDatabase interface {
		GetCollection(name string, embeddingFunc chromem.EmbeddingFunc) retrievalCollection
	}

	chromemRetrievalDatabase struct {
		db *chromem.DB
	}

	embeddingFuncFactory func(context.Context) (chromem.EmbeddingFunc, func(), error)

	RetrievalService struct {
		db      retrievalDatabase
		factory embeddingFuncFactory
	}

	Retriever struct {
		collection    retrievalCollection
		embeddingFunc chromem.EmbeddingFunc
		cleanup       func()
	}

	RetrievedChunk struct {
		ID          string
		Content     string
		Source      string
		HeadingPath string
		ChunkIndex  int
		Distance    float32
	}
)

func NewRetrievalService(db retrievalDatabase) *RetrievalService {
	return &RetrievalService{db: db, factory: newKronkEmbeddingFunc}
}

// NewRetrievalDatabase adapts a chromem DB to the retrieval database interface
func NewRetrievalDatabase(db *chromem.DB) retrievalDatabase {
	return chromemRetrievalDatabase{db: db}
}

func (d chromemRetrievalDatabase) GetCollection(name string, embeddingFunc chromem.EmbeddingFunc) retrievalCollection {
	return d.db.GetCollection(name, embeddingFunc)
}

func (s *RetrievalService) NewRetriever(ctx context.Context) (*Retriever, error) {
	return newRetriever(ctx, s.db, s.factory)
}

func (s *RetrievalService) RetrieveStyleGuideContext(ctx context.Context, query string) ([]RetrievedChunk, error) {
	retriever, err := s.NewRetriever(ctx)
	if err != nil {
		return nil, err
	}
	defer retriever.Close()

	return retriever.RetrieveStyleGuideContext(ctx, query)
}

func (r *Retriever) Close() {
	if r.cleanup != nil {
		r.cleanup()
	}
}

func (r *Retriever) RetrieveStyleGuideContext(ctx context.Context, query string) ([]RetrievedChunk, error) {
	return retrieveStyleGuideContext(ctx, r.collection, r.embeddingFunc, query)
}

func newRetriever(ctx context.Context, db retrievalDatabase, factory embeddingFuncFactory) (*Retriever, error) {
	collection := db.GetCollection(styleCollectionName, nil)
	if collection == nil {
		return nil, ErrStyleGuideCollectionMissing
	}

	embeddingFunc, cleanup, err := factory(ctx)
	if err != nil {
		return nil, err
	}
	return &Retriever{
		collection:    collection,
		embeddingFunc: embeddingFunc,
		cleanup:       cleanup,
	}, nil
}

func retrieveStyleGuideContext(ctx context.Context, collection retrievalCollection, embeddingFunc chromem.EmbeddingFunc, query string) ([]RetrievedChunk, error) {
	if query == "" {
		return nil, errors.New("query is empty")
	}

	if collection == nil {
		return nil, ErrStyleGuideCollectionMissing
	}

	queryEmbedding, err := embeddingFunc(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("could not embed retrieval query: %w", err)
	}

	baseResults, err := collection.QueryEmbedding(ctx, queryEmbedding, defaultTopK, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("could not query style guide embeddings: %w", err)
	}

	// no applicable style rule for this query
	if len(baseResults) == 0 {
		return nil, nil
	}

	selected := make(map[string]RetrievedChunk, len(baseResults)*(neighborWindow*2+1))
	for _, result := range baseResults {
		chunk, parseErr := resultToChunk(result)
		if parseErr != nil {
			return nil, parseErr
		}

		// include neighbors so split markdown sections do not lose rule context at chunk edges!
		for offset := -neighborWindow; offset <= neighborWindow; offset++ {
			neighborIndex := chunk.ChunkIndex + offset
			if neighborIndex < 1 {
				continue
			}

			neighborID := fmt.Sprintf("guide-%d", neighborIndex)
			if _, exists := selected[neighborID]; exists {
				continue
			}

			neighborResults, queryErr := collection.QueryEmbedding(
				ctx, queryEmbedding, 1,
				map[string]string{"chunk_index": strconv.Itoa(neighborIndex)}, nil,
			)
			if queryErr != nil {
				return nil, fmt.Errorf("could not query neighbor chunk %d: %w", neighborIndex, queryErr)
			}

			if len(neighborResults) == 0 {
				continue
			}

			neighborChunk, parseErr := resultToChunk(neighborResults[0])
			if parseErr != nil {
				return nil, parseErr
			}
			selected[neighborChunk.ID] = neighborChunk
		}
	}

	chunks := make([]RetrievedChunk, 0, len(selected))
	for _, chunk := range selected {
		chunks = append(chunks, chunk)
	}

	// keep the most relevant chunks before ordering for stable citation output
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Distance > chunks[j].Distance
	})

	if len(chunks) > maxRetrievedContexts {
		chunks = chunks[:maxRetrievedContexts]
	}

	// final ordering is by document position to preserve reading flow
	sort.Slice(chunks, func(i, j int) bool {
		if chunks[i].ChunkIndex == chunks[j].ChunkIndex {
			return chunks[i].Distance > chunks[j].Distance
		}
		return chunks[i].ChunkIndex < chunks[j].ChunkIndex
	})
	return chunks, nil
}

func resultToChunk(result chromem.Result) (RetrievedChunk, error) {
	source := result.Metadata["source"]
	if source == "" {
		return RetrievedChunk{}, fmt.Errorf("could not parse retrieval result %q: missing source metadata", result.ID)
	}

	headingPath := result.Metadata["heading_path"]
	if headingPath == "" {
		headingPath = "unknown"
	}

	chunkIndexRaw, ok := result.Metadata["chunk_index"]
	if !ok {
		return RetrievedChunk{}, fmt.Errorf("could not parse retrieval result %q: missing chunk_index metadata", result.ID)
	}

	chunkIndex, err := strconv.Atoi(chunkIndexRaw)
	if err != nil {
		return RetrievedChunk{}, fmt.Errorf("could not parse retrieval result %q chunk_index: %w", result.ID, err)
	}

	return RetrievedChunk{
		ID:          result.ID,
		Content:     result.Content,
		Source:      source,
		HeadingPath: headingPath,
		ChunkIndex:  chunkIndex,
		Distance:    result.Similarity,
	}, nil
}
