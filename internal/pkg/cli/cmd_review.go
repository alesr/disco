package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/alesr/disco/internal/app"
	"github.com/alesr/disco/internal/review"
	"github.com/spf13/cobra"
)

type reviewOptions struct {
	diffPath    string
	repoPath    string
	baseRef     string
	headRef     string
	maxFindings int
}

type reviewState struct {
	result    *review.ReviewResult
	narrative []review.ReviewEvent
}

func (a *Application) newReviewCommand() *cobra.Command {
	options := reviewOptions{
		repoPath: ".",
		headRef:  "HEAD",
	}

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Run daemon-backed interactive review flow",
		RunE: func(_ *cobra.Command, _ []string) error {
			return a.runReview(options)
		},
	}

	cmd.Flags().StringVar(&options.diffPath, "diff", options.diffPath, "path to unified diff file")
	cmd.Flags().StringVar(&options.repoPath, "repo", options.repoPath, "path to git repository")
	cmd.Flags().StringVar(&options.baseRef, "base", options.baseRef, "base git ref for repo mode")
	cmd.Flags().StringVar(&options.headRef, "head", options.headRef, "head git ref for repo mode")
	cmd.Flags().IntVar(&options.maxFindings, "max-findings", options.maxFindings, "maximum number of findings to return")
	return cmd
}

func (a *Application) runReview(options reviewOptions) error {
	diffContent, err := resolveReviewDiff(options)
	if err != nil {
		return err
	}

	var state reviewState
	executionMode, err := a.deps.RunReviewStream(context.Background(), app.ReviewOptions{
		Diff: diffContent,
	}, newReviewEmitter(&state))
	if err != nil {
		return err
	}

	if state.result == nil {
		return errors.New("review stream completed without final result")
	}

	transcript := buildInteractiveTranscript(*state.result, state.narrative)
	transcript.ExecutionMode = executionMode
	return runInteractiveReview(transcript, os.Stdin, os.Stdout)
}

func newReviewEmitter(state *reviewState) func(review.ReviewEvent) error {
	return func(event review.ReviewEvent) error {
		if event.Type == review.EventTypeProgress {
			fmt.Fprintf(os.Stderr, "evaluating hunk %d/%d (%s:%d)\n", event.Current, event.Total, event.File, event.Line)
		}

		if event.Type == review.EventTypeResult && event.Result != nil {
			copyResult := *event.Result
			state.result = &copyResult
		}

		if event.Type == review.EventTypeNarrative {
			state.narrative = append(state.narrative, event)
		}
		return nil
	}
}

func resolveReviewDiff(options reviewOptions) (string, error) {
	if options.diffPath != "" {
		return resolveReviewDiffFromFile(options.diffPath)
	}

	if strings.TrimSpace(options.baseRef) == "" {
		return "", errors.New("base is required when diff is not provided")
	}
	return resolveReviewDiffFromGit(options.repoPath, options.baseRef, options.headRef)
}

func resolveReviewDiffFromFile(diffPath string) (string, error) {
	content, err := os.ReadFile(diffPath)
	if err != nil {
		return "", fmt.Errorf("could not read diff file %q: %w", diffPath, err)
	}

	diff := string(content)
	if strings.TrimSpace(diff) == "" {
		return "", errors.New("diff content is empty")
	}
	return diff, nil
}

func resolveReviewDiffFromGit(repoPath, baseRef, headRef string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "diff", fmt.Sprintf("%s...%s", baseRef, headRef))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("could not generate git diff from %s...%s: %w", baseRef, headRef, err)
	}

	diff := string(output)
	if strings.TrimSpace(diff) == "" {
		return "", errors.New("diff content is empty")
	}
	return diff, nil
}
