.DEFAULT_GOAL := help

DISCO_BIN ?= $(CURDIR)/.bin/disco
FISH_COMPLETION_DIR ?= $(HOME)/.config/fish/completions
FISH_COMPLETION_FILE ?= $(FISH_COMPLETION_DIR)/disco.fish

.PHONY: help
help:
	@grep -E '^[a-zA-Z0-9_/%\-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort

.PHONY: all
all: fmt test vet vulncheck ## Run core quality gates

.PHONY: fmt
fmt: ## Format all Go files
	gofmt -w .

.PHONY: test
test: ## Run tests with race detector
	go test -race -count=1 -v ./...

.PHONY: vet
vet: ## Run go vet checks
	go vet ./...

.PHONY: vulncheck
vulncheck: ## Run vulnerability scan
	govulncheck ./...

.PHONY: install
install: ## Install disco CLI binary
	go install ./cmd/disco

.PHONY: review-sample-diff
review-sample-diff: ## Review sample diff
	go run ./cmd/disco review --diff sample-review.diff
