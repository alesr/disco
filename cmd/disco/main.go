package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alesr/disco/internal/app"
	"github.com/alesr/disco/internal/config"
	"github.com/alesr/disco/internal/daemon"
	"github.com/alesr/disco/internal/pkg/cli"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	application := cli.NewApplication(cli.Dependencies{
		StyleGuideDir: cfg.StyleGuideDir,
		NewDaemonManager: func() (cli.DaemonManager, error) {
			workingDir, err := os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("could not resolve current working directory: %w", err)
			}

			styleGuideDir := cfg.StyleGuideDir
			if !filepath.IsAbs(styleGuideDir) {
				styleGuideDir = filepath.Join(workingDir, styleGuideDir)
			}
			return daemon.NewManagerWithOptions(daemon.Options{
				WorkingDirectory:   workingDir,
				StyleGuideDir:      styleGuideDir,
				EmbeddingProvider:  cfg.EmbeddingProvider,
				GenerationProvider: cfg.GenerationProvider,
				MistralAPIKey:      cfg.MistralAPIKey,
				MistralModel:       cfg.MistralModel,
				MistralEmbedModel:  cfg.MistralEmbedModel,
				MistralBaseURL:     cfg.MistralBaseURL,
			})
		},
		RunServe:        app.RunServe,
		RunStatus:       app.RunStatus,
		RunReviewStream: app.RunReviewStream,
	})

	if err := application.Execute(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
