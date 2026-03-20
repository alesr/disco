package rag

import (
	"context"
	"fmt"
	"runtime"

	"github.com/philippgille/chromem-go"
	"github.com/tmc/langchaingo/textsplitter"
)

const styleCollectionName = "style-guide"

type (
	embeddingFactory func(context.Context) (chromem.EmbeddingFunc, func(), error)

	IngestService struct {
		db             database
		resolvePath    func(string) (string, error)
		readGuide      func(string) (string, error)
		validateGuide  func(string) error
		newEmbedder    embeddingFactory
		splitter       func(string) ([]string, error)
		inferHeadings  func(string) string
		concurrencyCap int
	}

	database interface {
		GetCollection(name string, embeddingFunc chromem.EmbeddingFunc) *chromem.Collection
		GetOrCreateCollection(name string, metadata map[string]string, embeddingFunc chromem.EmbeddingFunc) (*chromem.Collection, error)
		DeleteCollection(name string) error
	}
)

func NewIngestService(db database) *IngestService {
	return &IngestService{
		db:             db,
		resolvePath:    resolveStyleGuidePath,
		readGuide:      readStyleGuide,
		validateGuide:  validateStyleGuideMetadata,
		newEmbedder:    newKronkEmbeddingFunc,
		splitter:       splitStyleGuide,
		inferHeadings:  inferHeadingPath,
		concurrencyCap: runtime.NumCPU(),
	}
}

func (s *IngestService) Ingest(ctx context.Context, styleGuideDir string) error {
	selectedStyleGuide, err := s.resolvePath(styleGuideDir)
	if err != nil {
		return fmt.Errorf("could not resolve style guide path: %w", err)
	}

	content, err := s.readAndValidateStyleGuide(selectedStyleGuide)
	if err != nil {
		return fmt.Errorf("could not read and validate style guide: %w", err)
	}

	embeddingFunc, cleanup, err := s.newEmbedder(ctx)
	if err != nil {
		return fmt.Errorf("could not create embedding function: %w", err)
	}
	// embedder resources can be heavy and must be released on every ingest path
	defer cleanup()

	chunks, err := s.splitter(content)
	if err != nil {
		return fmt.Errorf("could not split style guide content: %w", err)
	}

	if err := s.resetStyleCollectionIfExists(embeddingFunc); err != nil {
		return fmt.Errorf("could not reset style collection: %w", err)
	}

	collection, err := s.db.GetOrCreateCollection(styleCollectionName, nil, embeddingFunc)
	if err != nil {
		return fmt.Errorf("could not create collection %s: %w", styleCollectionName, err)
	}

	docs, err := s.buildStyleGuideDocuments(ctx, embeddingFunc, chunks, selectedStyleGuide)
	if err != nil {
		return fmt.Errorf("could not build style guide documents: %w", err)
	}

	if err := collection.AddDocuments(ctx, docs, s.concurrencyCap); err != nil {
		return fmt.Errorf("could not add style guide documents: %w", err)
	}
	return nil
}

func (s *IngestService) readAndValidateStyleGuide(path string) (string, error) {
	content, err := s.readGuide(path)
	if err != nil {
		return "", err
	}

	if err := s.validateGuide(content); err != nil {
		return "", fmt.Errorf("could not validate style guide %q: %w", path, err)
	}
	return content, nil
}

func splitStyleGuide(content string) ([]string, error) {
	splitter := textsplitter.NewMarkdownTextSplitter(
		textsplitter.WithChunkSize(1000),
		textsplitter.WithChunkOverlap(150),
		textsplitter.WithHeadingHierarchy(true),
		textsplitter.WithCodeBlocks(true),
	)

	chunks, err := splitter.SplitText(content)
	if err != nil {
		return nil, fmt.Errorf("could not split style guide: %w", err)
	}
	return chunks, nil
}

func (s *IngestService) resetStyleCollectionIfExists(embeddingFunc chromem.EmbeddingFunc) error {
	if existing := s.db.GetCollection(styleCollectionName, embeddingFunc); existing == nil {
		return nil
	}

	// replacing the collection keeps embedding spaces consistent across provider/model changes
	if err := s.db.DeleteCollection(styleCollectionName); err != nil {
		return fmt.Errorf("could not reset collection %s: %w", styleCollectionName, err)
	}
	return nil
}

func (s *IngestService) buildStyleGuideDocuments(ctx context.Context, embeddingFunc chromem.EmbeddingFunc, chunks []string, selectedStyleGuide string) ([]chromem.Document, error) {
	docs := make([]chromem.Document, len(chunks))

	for i, chunk := range chunks {
		embedding, err := embeddingFunc(ctx, chunk)
		if err != nil {
			return nil, fmt.Errorf("could not embed chunk %d: %w", i+1, err)
		}

		headingPath := s.inferHeadings(chunk)

		docs[i] = chromem.Document{
			ID:        fmt.Sprintf("guide-%d", i+1),
			Content:   chunk,
			Embedding: embedding,
			Metadata: map[string]string{
				"source":       selectedStyleGuide,
				"chunk_index":  fmt.Sprintf("%d", i+1),
				"heading_path": headingPath,
			},
		}
	}
	return docs, nil
}
