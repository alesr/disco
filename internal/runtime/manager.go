package runtime

import (
	"context"
	"errors"
	"fmt"

	"github.com/alesr/disco/internal/llm/kronkgen"
	"github.com/alesr/disco/internal/policy/rag"
	"github.com/alesr/disco/internal/review"
)

const (
	maxEvidencePerHunk = 4
)

type Manager struct {
	retriever *rag.Retriever
	generator *kronkgen.Generator
}

func (m *Manager) ReviewDiff(ctx context.Context, diff string, maxFindings int) (review.ReviewResult, error) {
	var final *review.ReviewResult
	err := m.ReviewDiffStream(ctx, diff, func(event review.ReviewEvent) error {
		if event.Type == review.EventTypeResult && event.Result != nil {
			resultCopy := *event.Result
			final = &resultCopy
		}
		return nil
	})
	if err != nil {
		return review.ReviewResult{}, err
	}

	if final == nil {
		return review.ReviewResult{}, errors.New("review stream completed without final result")
	}
	return *final, nil
}

func (m *Manager) ReviewDiffStream(ctx context.Context, diff string, emit func(review.ReviewEvent) error) error {
	if emit == nil {
		return errors.New("stream emit function is nil")
	}

	engine := review.NewEngine(
		m.retrieveEvidence,
		func(ctx context.Context, input review.EvaluationInput) (review.EvaluationResult, error) {
			return m.evaluateHunk(ctx, input, maxEvidencePerHunk)
		},
		m.progressEmitter(ctx, emit),
	)

	result, err := engine.ReviewDiff(ctx, diff)
	if err != nil {
		return err
	}

	if err := emit(review.ReviewEvent{Type: review.EventTypeResult, Result: &result}); err != nil {
		return fmt.Errorf("could not emit final review result: %w", err)
	}
	return nil
}
