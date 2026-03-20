package app

import (
	"context"
	"fmt"

	"github.com/alesr/disco/internal/review"
	"github.com/alesr/disco/internal/transport/local"
)

type ReviewOptions struct {
	Diff string
}

func RunReviewStream(ctx context.Context, options ReviewOptions, emit func(review.ReviewEvent) error) (string, error) {
	if emit == nil {
		return "", fmt.Errorf("could not run review stream: emit callback is nil")
	}

	client := local.NewClient(local.DefaultSocketPath)

	if err := client.ReviewStream(ctx, local.ReviewRequest{Diff: options.Diff}, emit); err != nil {
		return "", fmt.Errorf("could not run daemon review (start daemon with `disco serve`): %w", err)
	}
	return "remote", nil
}
