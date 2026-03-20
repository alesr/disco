package runtime

import (
	"fmt"
	"strings"

	"github.com/alesr/disco/internal/review"
)

var knownSkillLabels = []string{
	"Logic", "Encyclopedia", "Rhetoric", "Drama", "Conceptualization", "Visual Calculus",
	"Volition", "Inland Empire", "Empathy", "Authority", "Esprit de Corps", "Suggestion",
	"Endurance", "Pain Threshold", "Physical Instrument", "Electrochemistry", "Shivers", "Half Light",
	"Hand/Eye Coordination", "Perception", "Reaction Speed", "Savoir Faire", "Interfacing", "Composure",
}

func stageFact(event review.ReviewEvent, fallback review.ReviewEvent) string {
	file := chooseString(event.File, fallback.File)
	line := chooseInt(event.Line, fallback.Line)
	current := chooseInt(event.Current, fallback.Current)
	total := chooseInt(event.Total, fallback.Total)

	if file == "" {
		file = "active-hunk"
	}

	if line <= 0 {
		line = 1
	}

	if current <= 0 {
		current = 1
	}

	if total < current {
		total = current
	}
	return fmt.Sprintf("checkpoint at %s:%d (%d/%d)", file, line, current, total)
}

func narrationBeat(eventType string) string {
	switch strings.TrimSpace(eventType) {
	case review.NarrativeEventHardFailure:
		return "pressure_spike"
	case review.NarrativeEventWarningFailure:
		return "hairline_crack"
	case review.NarrativeEventSoftFailure:
		return "unease"
	case review.NarrativeEventTimeout:
		return "signal_fade"
	case review.NarrativeEventModelError:
		return "static"
	case review.NarrativeEventFiltered:
		return "fog"
	case review.NarrativeEventSuccess:
		return "breath"
	case review.NarrativeEventNoRule:
		return "silence"
	default:
		return "checkpoint"
	}
}

func containsDiagnosticLanguage(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return true
	}

	markers := []string{
		"violates", "violation", "rule", "style guide", "finding", "severity",
		"blocking", "fix", "rg-",
	}

	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func appendNarrative(existing []string, line string) []string {
	trimmed := chooseString(line, "")
	if trimmed == "" {
		return existing
	}

	existing = append(existing, trimmed)
	if len(existing) <= 4 {
		return existing
	}

	// short rolling mem balances continuity against repetitive prompt bloat
	return existing[len(existing)-4:]
}

func chooseInt(primary, fallback int) int {
	if primary > 0 {
		return primary
	}
	return fallback
}

func chooseString(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func normalizeSkillLabel(skill string) string {
	trimmed := strings.TrimSpace(skill)

	for _, known := range knownSkillLabels {
		if strings.EqualFold(trimmed, known) {
			return known
		}
	}
	return trimmed
}

func normalizeDifficultyLabel(level string) string {
	trimmed := strings.TrimSpace(level)

	for _, known := range review.ValidDiscoDifficulties {
		if strings.EqualFold(trimmed, known) {
			return known
		}
	}
	return trimmed
}
