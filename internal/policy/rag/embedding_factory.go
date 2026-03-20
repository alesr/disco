package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ardanlabs/kronk/sdk/kronk"
	"github.com/ardanlabs/kronk/sdk/kronk/model"
	"github.com/ardanlabs/kronk/sdk/tools/defaults"
	"github.com/ardanlabs/kronk/sdk/tools/libs"
	"github.com/ardanlabs/kronk/sdk/tools/models"
	"github.com/philippgille/chromem-go"
)

const (
	embedTimeout             = 120 * time.Second
	embedModelURL            = "https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q8_0.gguf"
	defaultEmbeddingProvider = "kronk"
	providerKronk            = "kronk"
	providerMistral          = "mistral"
	defaultMistralBaseURL    = "https://api.mistral.ai"
	defaultMistralModel      = "mistral-embed"
)

type embeddingProviderConfig struct {
	Provider          string
	MistralAPIKey     string
	MistralEmbedModel string
	MistralBaseURL    string
}

type mistralEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type mistralEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func newEmbeddingFunc(ctx context.Context) (chromem.EmbeddingFunc, func(), error) {
	cfg := loadEmbeddingProviderConfig()

	switch cfg.Provider {
	case providerKronk:
		return newKronkEmbeddingFunc(ctx)
	case providerMistral:
		return newMistralEmbeddingFunc(ctx, cfg)
	default:
		return nil, nil, fmt.Errorf("unsupported embedding provider %q", cfg.Provider)
	}
}

func newKronkEmbeddingFunc(ctx context.Context) (chromem.EmbeddingFunc, func(), error) {
	modelPath, err := resolveEmbeddingModelPath(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("could not resolve embedding model path: %w", err)
	}

	if err := kronk.Init(); err != nil {
		return nil, nil, fmt.Errorf("could not initialize kronk: %w", err)
	}

	krn, err := kronk.New(model.Config{ModelFiles: []string{modelPath}})
	if err != nil {
		return nil, nil, fmt.Errorf("could not load embedding model: %w", err)
	}

	cleanup := func() {
		if err := krn.Unload(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "could not unload embedding model: %v\n", err)
		}
	}

	embeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		embedCtx := ctx
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			embedCtx, cancel = context.WithTimeout(ctx, embedTimeout)
			defer cancel()
		}

		response, err := krn.Embeddings(
			embedCtx,
			model.D{
				"input":              text,
				"truncate":           true,
				"truncate_direction": "right",
			},
		)
		if err != nil {
			return nil, fmt.Errorf("could not create kronk embedding: %w", err)
		}

		if len(response.Data) == 0 || len(response.Data[0].Embedding) == 0 {
			return nil, errors.New("kronk embeddings: empty response")
		}

		normalized, err := normalizeVector(response.Data[0].Embedding)
		if err != nil {
			return nil, fmt.Errorf("could not normalize embedding: %w", err)
		}
		return normalized, nil
	}
	return embeddingFunc, cleanup, nil
}

func newMistralEmbeddingFunc(_ context.Context, cfg embeddingProviderConfig) (chromem.EmbeddingFunc, func(), error) {
	apiKey := strings.TrimSpace(cfg.MistralAPIKey)
	if apiKey == "" {
		return nil, nil, errors.New("mistral embedding provider requires MISTRAL_API_KEY")
	}

	modelName := strings.TrimSpace(cfg.MistralEmbedModel)
	if modelName == "" {
		modelName = defaultMistralModel
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.MistralBaseURL), "/")
	if baseURL == "" {
		baseURL = defaultMistralBaseURL
	}

	client := &http.Client{Timeout: embedTimeout}
	endpoint := baseURL + "/v1/embeddings"

	cleanup := func() {}

	embeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		requestPayload := mistralEmbeddingRequest{Model: modelName, Input: text}
		body, err := json.Marshal(requestPayload)
		if err != nil {
			return nil, fmt.Errorf("could not encode mistral embeddings request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("could not create mistral embeddings request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("could not create mistral embedding: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
			return nil, fmt.Errorf("could not create mistral embedding: status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
		}

		var parsedResponse mistralEmbeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&parsedResponse); err != nil {
			return nil, fmt.Errorf("could not decode mistral embeddings response: %w", err)
		}

		if len(parsedResponse.Data) == 0 || len(parsedResponse.Data[0].Embedding) == 0 {
			return nil, errors.New("mistral embeddings: empty response")
		}

		normalized, err := normalizeVector(parsedResponse.Data[0].Embedding)
		if err != nil {
			return nil, fmt.Errorf("could not normalize embedding: %w", err)
		}
		return normalized, nil
	}
	return embeddingFunc, cleanup, nil
}

func loadEmbeddingProviderConfig() embeddingProviderConfig {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("EMBEDDING_PROVIDER")))
	if provider == "" {
		// keep first run usable without forcing provider configuration (maybe return error later)
		provider = defaultEmbeddingProvider
	}
	return embeddingProviderConfig{
		Provider:          provider,
		MistralAPIKey:     os.Getenv("MISTRAL_API_KEY"),
		MistralEmbedModel: os.Getenv("MISTRAL_EMBED_MODEL"),
		MistralBaseURL:    os.Getenv("MISTRAL_BASE_URL"),
	}
}

func resolveEmbeddingModelPath(ctx context.Context) (string, error) {
	libraryManager, err := libs.New(libs.WithVersion(defaults.LibVersion("")))
	if err != nil {
		return "", fmt.Errorf("could not create libs manager: %w", err)
	}

	if _, err := libraryManager.Download(ctx, kronk.FmtLogger); err != nil {
		return "", fmt.Errorf("could not install llama.cpp libs: %w", err)
	}

	modelManager, managerErr := models.New()
	if managerErr != nil {
		return "", fmt.Errorf("could not create models manager: %w", managerErr)
	}

	downloadedPath, downloadErr := modelManager.Download(ctx, kronk.FmtLogger, embedModelURL, "")
	if downloadErr != nil {
		return "", fmt.Errorf("could not download embedding model from %q: %w", embedModelURL, downloadErr)
	}

	if len(downloadedPath.ModelFiles) == 0 {
		return "", fmt.Errorf("could not download embedding model from %q: no model files returned", embedModelURL)
	}
	return downloadedPath.ModelFiles[0], nil
}

func normalizeVector(vector []float32) ([]float32, error) {
	if len(vector) == 0 {
		return nil, errors.New("vector is empty")
	}

	var norm float64
	for _, value := range vector {
		norm += float64(value * value)
	}

	if norm == 0 {
		// zero norm silently breaks cosine distance so we fail fast here
		return nil, errors.New("vector norm is zero")
	}

	scale := float32(1 / math.Sqrt(norm))
	normalized := make([]float32, len(vector))

	for i, value := range vector {
		normalized[i] = value * scale
	}
	return normalized, nil
}
