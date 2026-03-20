package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	StyleGuideDir      string
	EmbeddingProvider  string
	GenerationProvider string
	MistralAPIKey      string
	MistralModel       string
	MistralEmbedModel  string
	MistralBaseURL     string
}

type envConfig struct {
	StyleGuideDir      string `env:"STYLE_GUIDE_DIR" envDefault:"./styleguides"`
	EmbeddingProvider  string `env:"EMBEDDING_PROVIDER" envDefault:"kronk"`
	GenerationProvider string `env:"GENERATION_PROVIDER" envDefault:"kronk"`
	MistralAPIKey      string `env:"MISTRAL_API_KEY"`
	MistralModel       string `env:"MISTRAL_MODEL" envDefault:"mistral-small-latest"`
	MistralEmbedModel  string `env:"MISTRAL_EMBED_MODEL" envDefault:"mistral-embed"`
	MistralBaseURL     string `env:"MISTRAL_BASE_URL" envDefault:"https://api.mistral.ai"`
}

func Load() (Config, error) {
	var envCfg envConfig
	if err := env.Parse(&envCfg); err != nil {
		return Config{}, fmt.Errorf("could not parse environment config: %w", err)
	}

	return Config{
		StyleGuideDir:      envCfg.StyleGuideDir,
		EmbeddingProvider:  envCfg.EmbeddingProvider,
		GenerationProvider: envCfg.GenerationProvider,
		MistralAPIKey:      envCfg.MistralAPIKey,
		MistralModel:       envCfg.MistralModel,
		MistralEmbedModel:  envCfg.MistralEmbedModel,
		MistralBaseURL:     envCfg.MistralBaseURL,
	}, nil
}
