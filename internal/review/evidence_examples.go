package review

import (
	"os"
	"strings"
)

func extractGoodExample(evidence []EvidenceChunk, rule string) string {
	ruleID := strings.ToUpper(strings.TrimSpace(ruleIDPattern.FindString(rule)))
	bestScore := float32(-1)

	var best, bestSource string

	for _, chunk := range evidence {
		heading := strings.TrimSpace(chunk.HeadingPath)
		if heading == "" {
			continue
		}

		if !strings.Contains(strings.ToLower(heading), "> good") {
			continue
		}

		if ruleID != "" && !strings.Contains(strings.ToUpper(heading), ruleID) {
			continue
		}

		example := sanitizeGoodExample(chunk.Content)
		if example == "" {
			continue
		}

		if chunk.Score > bestScore {
			bestScore = chunk.Score
			best = example
			bestSource = strings.TrimSpace(chunk.Source)
		}
	}

	if best != "" {
		// ranked retrieval evidence is preferred because it already matches the active hunk context
		return best
	}

	if ruleID == "" {
		ruleID = inferRuleIDFromEvidence(evidence)
	}

	if bestSource == "" {
		bestSource = inferSourcePath(evidence)
	}

	if ruleID == "" || bestSource == "" {
		return ""
	}

	// source fallback recovers examples when retrieval returns heading-only chunks
	return extractGoodExampleFromSource(bestSource, ruleID)
}

func inferRuleIDFromEvidence(evidence []EvidenceChunk) string {
	for _, chunk := range evidence {
		ruleID := strings.ToUpper(strings.TrimSpace(ruleIDPattern.FindString(chunk.HeadingPath)))
		if ruleID != "" {
			return ruleID
		}
	}
	return ""
}

func inferSourcePath(evidence []EvidenceChunk) string {
	for _, chunk := range evidence {
		source := strings.TrimSpace(chunk.Source)
		if source != "" {
			return source
		}
	}
	return ""
}

func extractGoodExampleFromSource(sourcePath, ruleID string) string {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	ruleHeadingPrefix := "## " + ruleID + " - "
	sectionStart := -1
	for idx, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), ruleHeadingPrefix) {
			sectionStart = idx + 1
			break
		}
	}

	if sectionStart == -1 {
		return ""
	}

	goodStart := -1
	for idx := sectionStart; idx < len(lines); idx++ {
		trimmed := strings.TrimSpace(lines[idx])
		if strings.HasPrefix(trimmed, "## RG-") {
			break
		}

		if strings.EqualFold(trimmed, "### Good") {
			goodStart = idx + 1
			break
		}
	}

	if goodStart == -1 {
		return ""
	}

	collected := make([]string, 0, 16)
	for idx := goodStart; idx < len(lines); idx++ {
		trimmed := strings.TrimSpace(lines[idx])
		if strings.HasPrefix(trimmed, "### ") || strings.HasPrefix(trimmed, "## RG-") {
			break
		}
		collected = append(collected, lines[idx])
	}

	if len(collected) == 0 {
		return ""
	}
	return sanitizeGoodExample(strings.Join(collected, "\n"))
}

func sanitizeGoodExample(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}

	if fenced := extractFencedCodeBlock(trimmed); fenced != "" {
		return fenced
	}

	trimmed = strings.TrimPrefix(trimmed, "### Good")
	trimmed = strings.TrimSpace(trimmed)

	lines := strings.Split(trimmed, "\n")
	cleaned := make([]string, 0, len(lines))

	for _, line := range lines {
		stripped := strings.TrimSpace(line)
		if stripped == "" {
			continue
		}

		if strings.HasPrefix(stripped, "#") {
			continue
		}

		if strings.EqualFold(stripped, "good") || strings.EqualFold(stripped, "bad") {
			continue
		}

		cleaned = append(cleaned, line)
	}

	if len(cleaned) == 0 {
		return ""
	}

	if len(cleaned) > 6 {
		cleaned = cleaned[:6]
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func extractFencedCodeBlock(content string) string {
	lines := strings.Split(content, "\n")
	start := -1
	end := -1

	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if start == -1 {
				start = idx
				continue
			}

			end = idx
			break
		}
	}

	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines[start:end+1], "\n"))
}
