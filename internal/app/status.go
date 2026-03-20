package app

import (
	"context"
	"fmt"

	"github.com/alesr/disco/internal/transport/local"
)

type StatusOptions struct{}

func RunStatus(ctx context.Context, options StatusOptions) (local.HealthResponse, error) {
	client := local.NewClient(local.DefaultSocketPath)
	status, err := client.Health(ctx)
	if err != nil {
		return local.HealthResponse{}, fmt.Errorf("could not query local daemon status: %w", err)
	}
	return status, nil
}
