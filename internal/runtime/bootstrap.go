package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alesr/disco/internal/llm/kronkgen"
	"github.com/alesr/disco/internal/policy/rag"
	"github.com/philippgille/chromem-go"
)

const (
	vectorDBPath          = "./data/vector_db"
	generatorInitTimeout  = 25 * time.Minute
	generatorCloseTimeout = 30 * time.Second
)

type Dependencies struct {
	Retriever *rag.Retriever
	Generator *kronkgen.Generator
}

func NewManager(ctx context.Context) (*Manager, error) {
	retriever, err := newRetriever(ctx)
	if err != nil {
		return nil, err
	}

	generator, err := newGenerator(ctx)
	if err != nil {
		retriever.Close()
		return nil, err
	}
	return NewManagerWithDependencies(Dependencies{
		Retriever: retriever,
		Generator: generator,
	})
}

func NewManagerWithDependencies(deps Dependencies) (*Manager, error) {
	if deps.Retriever == nil {
		return nil, errors.New("retriever dependency is nil")
	}

	if deps.Generator == nil {
		return nil, errors.New("generator dependency is nil")
	}
	return &Manager{retriever: deps.Retriever, generator: deps.Generator}, nil
}

func newRetriever(ctx context.Context) (*rag.Retriever, error) {
	db, err := chromem.NewPersistentDB(vectorDBPath, false)
	if err != nil {
		return nil, fmt.Errorf("could not create persistent DB: %w", err)
	}

	retrievalDB := rag.NewRetrievalDatabase(db)
	retriever, err := rag.NewRetrievalService(retrievalDB).NewRetriever(ctx)
	if err != nil {
		if errors.Is(err, rag.ErrStyleGuideCollectionMissing) {
			return nil, errors.New("style guide index is missing, run `disco ingest` first")
		}

		return nil, fmt.Errorf("could not initialize style retrieval: %w", err)
	}
	return retriever, nil
}

func newGenerator(ctx context.Context) (*kronkgen.Generator, error) {
	initCtx := ctx
	if _, hasDeadline := initCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		initCtx, cancel = context.WithTimeout(ctx, generatorInitTimeout)
		defer cancel()
	}
	return kronkgen.NewGenerator(initCtx)
}

func (m *Manager) Close(ctx context.Context) error {
	if m == nil {
		return nil
	}

	if m.retriever != nil {
		m.retriever.Close()
	}

	if m.generator == nil {
		return nil
	}

	closeCtx := ctx
	if _, hasDeadline := closeCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		closeCtx, cancel = context.WithTimeout(ctx, generatorCloseTimeout)
		defer cancel()
	}

	if err := m.generator.Close(closeCtx); err != nil {
		return fmt.Errorf("could not close generation runtime: %w", err)
	}
	return nil
}
