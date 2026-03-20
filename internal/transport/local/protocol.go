package local

import "github.com/alesr/disco/internal/review"

const DefaultSocketPath = "/tmp/disco.sock"

type (
	ReviewRequest struct {
		Diff string `json:"diff"`
	}

	ReviewEvent = review.ReviewEvent

	HealthResponse struct {
		Status string `json:"status"`
	}
)
