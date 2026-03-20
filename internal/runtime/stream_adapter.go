package runtime

import (
	"context"
	"strings"

	"github.com/alesr/disco/internal/llm/kronkgen"
	"github.com/alesr/disco/internal/review"
)

func (m *Manager) retrieveEvidence(ctx context.Context, query string) ([]review.EvidenceChunk, error) {
	rawChunks, err := m.retriever.RetrieveStyleGuideContext(ctx, query)
	if err != nil {
		return nil, err
	}

	chunks := make([]review.EvidenceChunk, 0, len(rawChunks))
	for _, chunk := range rawChunks {
		chunks = append(chunks, review.EvidenceChunk{
			Source:      chunk.Source,
			HeadingPath: chunk.HeadingPath,
			ChunkIndex:  chunk.ChunkIndex,
			Score:       chunk.Distance,
			Content:     chunk.Content,
		})
	}

	return chunks, nil
}

func (m *Manager) evaluateHunk(ctx context.Context, input review.EvaluationInput, maxEvidence int) (review.EvaluationResult, error) {
	evidence := input.Evidence
	if len(evidence) > maxEvidence {
		evidence = evidence[:maxEvidence]
	}

	kronkEvidence := make([]kronkgen.EvidenceChunk, 0, len(evidence))
	for _, chunk := range evidence {
		kronkEvidence = append(kronkEvidence, kronkgen.EvidenceChunk{
			Source:      chunk.Source,
			HeadingPath: chunk.HeadingPath,
			ChunkIndex:  chunk.ChunkIndex,
			Score:       chunk.Score,
			Content:     chunk.Content,
		})
	}

	result, err := m.generator.ReviewHunk(ctx, input.File, input.Line, input.Hunk, kronkEvidence)
	if err != nil {
		return review.EvaluationResult{}, err
	}

	return review.EvaluationResult{
		IsViolation:      result.IsViolation,
		Severity:         result.Severity,
		Rule:             result.Rule,
		Message:          result.Message,
		TechnicalMessage: result.TechnicalMessage,
		DiscoMessage:     result.DiscoMessage,
		Taxonomy:         result.Taxonomy,
		Confidence:       result.Confidence,
	}, nil
}

func (m *Manager) progressEmitter(ctx context.Context, emit func(review.ReviewEvent) error) func(review.ReviewEvent) {
	previousNarratives := make([]string, 0, 4)

	var (
		lastProgress       review.ReviewEvent
		lastNarrativeClass string
		passStreak         int
		failStreak         int
	)

	return func(event review.ReviewEvent) {
		if event.Type == review.EventTypeProgress {
			lastProgress = event
		}

		if event.Type == review.EventTypeNarrative {
			if shouldNarrateEvent(event.EventType, lastNarrativeClass) {
				class := narrativeClass(event.EventType)
				if class == "failure" {
					failStreak++
					passStreak = 0
				}

				if class == "success" {
					passStreak++
					failStreak = 0
				}

				trend := narrativeTrend(class, lastNarrativeClass, passStreak, failStreak)
				lastNarrativeClass = narrativeClass(event.EventType)

				knownFact := strings.TrimSpace(event.Content)
				if knownFact == "" {
					knownFact = stageFact(event, lastProgress)
				}

				narrated, err := m.generator.NarrateEvent(ctx, kronkgen.NarrativeInput{
					EventType:          narrationBeat(event.EventType),
					Skill:              event.Skill,
					Difficulty:         event.Difficulty,
					Content:            knownFact,
					Trend:              trend,
					PassStreak:         passStreak,
					FailStreak:         failStreak,
					LastEventClass:     class,
					Current:            chooseInt(event.Current, lastProgress.Current),
					Total:              chooseInt(event.Total, lastProgress.Total),
					File:               chooseString(event.File, lastProgress.File),
					Line:               chooseInt(event.Line, lastProgress.Line),
					Severity:           event.Severity,
					FailureClass:       event.FailureClass,
					GuidanceType:       event.GuidanceType,
					Blocking:           event.Blocking,
					PreviousNarratives: previousNarratives,
				})
				if err == nil {
					event.Content = narrated.Content
					event.Skill = normalizeSkillLabel(narrated.VoiceSkill)
					event.Difficulty = normalizeDifficultyLabel(narrated.Difficulty)
					event.Stance = strings.ToLower(strings.TrimSpace(narrated.Stance))
					previousNarratives = appendNarrative(previousNarratives, event.Content)
				} else {
					// narrative failures should not block review delivery
					event.Content = ""
				}
			} else {
				// skipping selected beats keeps pacing readable in long diffs
				event.Content = ""
			}
		}

		if err := emit(event); err != nil {
			// stream cancellation or sink failures should stop review quickly
			// the runner API does not accept returning errors from progress callbacks
			// so we cancel via context by panic/recover is not safe; best effort only
		}
	}
}

func shouldNarrateEvent(eventType, lastClass string) bool {
	class := narrativeClass(eventType)
	if class == "" {
		return false
	}

	if class == "failure" {
		return true
	}

	if class == "success" {
		return lastClass == "failure"
	}

	return false
}

func narrativeClass(eventType string) string {
	switch strings.TrimSpace(eventType) {
	case review.NarrativeEventHardFailure, review.NarrativeEventWarningFailure, review.NarrativeEventSoftFailure,
		review.NarrativeEventTimeout, review.NarrativeEventModelError, review.NarrativeEventFiltered:
		return "failure"
	case review.NarrativeEventSuccess:
		return "success"
	default:
		return ""
	}
}

func narrativeTrend(class, lastClass string, passStreak, failStreak int) string {
	if class == "failure" {
		if lastClass == "success" {
			return "downhill"
		}

		if failStreak >= 2 {
			return "downhill"
		}

		return "pressure"
	}

	if class == "success" {
		if lastClass == "failure" {
			return "recovery"
		}

		if passStreak >= 2 {
			return "steady"
		}

		return "steady"
	}
	return "steady"
}
