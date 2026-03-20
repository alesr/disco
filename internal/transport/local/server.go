package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/alesr/disco/internal/review"
)

type Reviewer interface {
	ReviewDiffStream(ctx context.Context, diff string, emit func(review.ReviewEvent) error) error
}

type Server struct {
	httpServer *http.Server
	listener   net.Listener
	socketPath string
}

func NewServer(socketPath string, reviewer Reviewer) (*Server, error) {
	if reviewer == nil {
		return nil, errors.New("reviewer is nil")
	}

	if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("could not remove existing socket %q: %w", socketPath, err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("could not listen on socket %q: %w", socketPath, err)
	}

	if err := os.Chmod(socketPath, 0o600); err != nil {
		// local-only socket permissions reduce accidental exposure of review payloads
		listener.Close()
		return nil, fmt.Errorf("could not set socket permissions for %q: %w", socketPath, err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
	})

	mux.HandleFunc("/review", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
			return
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			_ = json.NewEncoder(w).Encode(review.ReviewEvent{Type: review.EventTypeError, Error: "streaming is not supported"})
			return
		}

		encoder := json.NewEncoder(w)
		emit := func(event review.ReviewEvent) error {
			if err := encoder.Encode(event); err != nil {
				return fmt.Errorf("could not encode review stream event: %w", err)
			}
			flusher.Flush()
			return nil
		}

		var req ReviewRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			_ = emit(review.ReviewEvent{Type: review.EventTypeError, Error: fmt.Sprintf("could not decode review request: %v", err)})
			return
		}

		if err := reviewer.ReviewDiffStream(r.Context(), req.Diff, emit); err != nil {
			_ = emit(review.ReviewEvent{Type: review.EventTypeError, Error: err.Error()})
			return
		}
	})

	httpServer := &http.Server{Handler: mux}
	return &Server{httpServer: httpServer, listener: listener, socketPath: socketPath}, nil
}

func (s *Server) Serve() error {
	if s == nil || s.httpServer == nil || s.listener == nil {
		return errors.New("server is not initialized")
	}

	if err := s.httpServer.Serve(s.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("could not serve local transport: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return nil
	}

	shutdownCtx := ctx
	if _, hasDeadline := shutdownCtx.Deadline(); !hasDeadline {
		// bounded shutdown prevents hanging service stop operations
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	err := s.httpServer.Shutdown(shutdownCtx)
	if removeErr := os.Remove(s.socketPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		if err == nil {
			err = removeErr
		}
	}

	if err != nil {
		return fmt.Errorf("could not shutdown local transport server: %w", err)
	}
	return nil
}
