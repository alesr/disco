package local

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/alesr/disco/internal/review"
)

type Client struct {
	socketPath string
	httpClient *http.Client
}

func NewClient(socketPath string) *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", socketPath)
		},
	}

	httpClient := &http.Client{
		Transport: transport,
	}

	return &Client{
		socketPath: socketPath,
		httpClient: httpClient,
	}
}

func (c *Client) ReviewStream(ctx context.Context, req ReviewRequest, emit func(ReviewEvent) error) error {
	if emit == nil {
		return errors.New("stream emit function is nil")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("could not marshal review request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://local/review", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("could not create review request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("could not contact daemon on socket %q: %w", c.socketPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon review failed with status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	// daemon events can carry long narrative lines so scanner limit is raised defensively
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event ReviewEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return fmt.Errorf("could not decode daemon stream event: %w", err)
		}

		if err := emit(event); err != nil {
			return fmt.Errorf("could not emit daemon stream event: %w", err)
		}

		if event.Type == review.EventTypeError {
			if event.Error != "" {
				return errors.New(event.Error)
			}
			return errors.New("daemon emitted stream error event")
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("could not read daemon stream: %w", err)
	}
	return nil
}

func (c *Client) Health(ctx context.Context) (HealthResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://local/health", nil)
	if err != nil {
		return HealthResponse{}, fmt.Errorf("could not create health request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return HealthResponse{}, fmt.Errorf("could not contact daemon on socket %q: %w", c.socketPath, err)
	}
	defer resp.Body.Close()

	var payload HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return HealthResponse{}, fmt.Errorf("could not decode daemon health response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return HealthResponse{}, fmt.Errorf("daemon health failed with status %d", resp.StatusCode)
	}

	if payload.Status == "" {
		return HealthResponse{}, errors.New("daemon health response is empty")
	}
	return payload, nil
}
