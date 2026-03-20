package review

import "strings"

func passesConfidenceGate(confidence, evidenceScore float32, severity string) bool {
	if normalizeConfidence(confidence) < defaultMinConfidence {
		return false
	}

	if evidenceScore < defaultMinEvidenceScore {
		return false
	}

	if strings.EqualFold(strings.TrimSpace(severity), "high") {
		if normalizeConfidence(confidence) < highSeverityMinConfidence {
			return false
		}

		if evidenceScore < highSeverityMinEvidenceScore {
			return false
		}
	}

	return true
}

func normalizeConfidence(confidence float32) float32 {
	if confidence < 0 {
		return 0
	}

	if confidence > 1 {
		return 1
	}
	return confidence
}

func normalizeSeverity(severity string) string {
	s := strings.ToLower(strings.TrimSpace(severity))
	switch s {
	case "low", "medium", "high":
		return s
	default:
		return "medium"
	}
}

func normalizeMessage(preferred, fallback string) string {
	msg := strings.TrimSpace(preferred)
	if msg != "" {
		return msg
	}
	return strings.TrimSpace(fallback)
}
