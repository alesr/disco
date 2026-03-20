package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/alesr/disco/internal/review"
)

const (
	ansiReset             = "\x1b[0m"
	ansiBold              = "\x1b[1m"
	ansiDim               = "\x1b[2m"
	interactiveRenderWide = 132 // 72 before
)

type interactiveTranscript struct {
	ExecutionMode string               `json:"execution_mode"`
	Summary       review.ReviewSummary `json:"summary"`
	Checks        []interactiveCheck   `json:"checks"`
	Interludes    []string             `json:"interludes"`
}

type interactiveCheck struct {
	Check     review.SkillCheck `json:"check"`
	Narrative string            `json:"narrative,omitempty"`
}

func buildInteractiveTranscript(result review.ReviewResult, narrativeEvents []review.ReviewEvent) interactiveTranscript {
	checks := make([]review.SkillCheck, 0, len(result.Findings))
	for _, finding := range result.Findings {
		checks = append(checks, review.SkillCheckFromFinding(finding))
	}

	queue := make([]review.ReviewEvent, 0, len(narrativeEvents))
	failures := make([]review.ReviewEvent, 0, len(narrativeEvents))

	for _, event := range narrativeEvents {
		if event.Type != review.EventTypeNarrative {
			continue
		}

		if strings.TrimSpace(event.Content) == "" {
			continue
		}

		queue = append(queue, event)
		if event.EventType != review.NarrativeEventSuccess {
			failures = append(failures, event)
		}
	}

	interludes := make([]string, 0, len(queue))
	items := make([]interactiveCheck, 0, len(checks))
	var queueIdx int
	usedFailures := make([]bool, len(failures))

	for _, check := range checks {
		expected := expectedNarrativeEventType(check)
		var narrative string

		for queueIdx < len(queue) {
			event := queue[queueIdx]
			queueIdx++
			if event.EventType == review.NarrativeEventSuccess {
				interludes = append(interludes, strings.TrimSpace(event.Content))
			}
		}

		if idx := selectNarrativeEvent(failures, usedFailures, check, expected); idx >= 0 {
			// avoid repeating the same beat on multiple checks
			narrative = strings.TrimSpace(failures[idx].Content)
			usedFailures[idx] = true
		}
		items = append(items, interactiveCheck{Check: check, Narrative: narrative})
	}

	return interactiveTranscript{
		Summary:    result.Summary,
		Checks:     items,
		Interludes: interludes,
	}
}

func selectNarrativeEvent(events []review.ReviewEvent, used []bool, check review.SkillCheck, expected string) int {
	for idx, event := range events {
		if used[idx] {
			continue
		}
		// keep narration tied to the exact finding when available
		if strings.TrimSpace(event.File) == strings.TrimSpace(check.File) && event.Line == check.Line {
			return idx
		}
	}

	for idx, event := range events {
		if used[idx] {
			continue
		}
		if strings.TrimSpace(event.File) == strings.TrimSpace(check.File) {
			return idx
		}
	}

	for idx, event := range events {
		if used[idx] {
			continue
		}
		if strings.TrimSpace(event.EventType) == expected {
			return idx
		}
	}

	for idx := range events {
		if !used[idx] {
			return idx
		}
	}
	return -1
}

func expectedNarrativeEventType(check review.SkillCheck) string {
	if check.FailureClass == review.FailureClassSoft {
		return review.NarrativeEventSoftFailure
	}

	if check.Blocking {
		return review.NarrativeEventHardFailure
	}
	return review.NarrativeEventWarningFailure
}

func runInteractiveReview(transcript interactiveTranscript, input io.Reader, output io.Writer) error {
	if len(transcript.Checks) == 0 {
		fmt.Fprintln(output, "No actionable checks.")
		printNoCheckSummary(output, transcript.Summary)
		return nil
	}

	fmt.Fprintf(output, "%d checks queued.\n", len(transcript.Checks))
	printContinuePrompt(output)

	reader := bufio.NewReader(input)
	if err := waitForEnter(reader); err != nil {
		return err
	}

	var interludeIdx int

	for idx, item := range transcript.Checks {
		fmt.Fprintln(output)
		fmt.Fprintf(output, "-- check %d/%d --\n", idx+1, len(transcript.Checks))

		if item.Narrative != "" {
			for _, line := range wrapText(strings.TrimSpace(item.Narrative), interactiveRenderWide) {
				fmt.Fprintln(output, line)
			}
			fmt.Fprintln(output)
		}

		printCompactCheck(output, item.Check)

		if interludeIdx < len(transcript.Interludes) {
			for _, line := range wrapText(strings.TrimSpace(transcript.Interludes[interludeIdx]), interactiveRenderWide) {
				fmt.Fprintln(output, line)
			}
			interludeIdx++
		}

		if idx < len(transcript.Checks)-1 {
			printContinuePrompt(output)
			if err := waitForEnter(reader); err != nil {
				return err
			}
		}
	}
	return nil
}

func printNoCheckSummary(output io.Writer, summary review.ReviewSummary) {
	hints := make([]string, 0, 4)

	if summary.HunksModelError > 0 {
		hints = append(hints, fmt.Sprintf("model errors=%d", summary.HunksModelError))
	}

	if summary.HunksSkippedTimeout > 0 {
		hints = append(hints, fmt.Sprintf("timeouts=%d", summary.HunksSkippedTimeout))
	}

	if summary.NoApplicableRule > 0 {
		hints = append(hints, fmt.Sprintf("no-applicable-rule=%d", summary.NoApplicableRule))
	}

	if summary.HunksFilteredLowConfidence > 0 {
		hints = append(hints, fmt.Sprintf("low-confidence-filtered=%d", summary.HunksFilteredLowConfidence))
	}

	if len(hints) == 0 {
		return
	}
	fmt.Fprintf(output, "summary: %s\n", strings.Join(hints, ", "))
}

func waitForEnter(reader *bufio.Reader) error {
	if _, err := reader.ReadString('\n'); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("could not continue interactive review: %w", err)
	}
	return nil
}
