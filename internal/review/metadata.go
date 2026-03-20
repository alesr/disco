package review

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
)

var (
	metadataLinePattern = regexp.MustCompile(`(?i)^\s*(type|enforcement|taxonomy|skill_primary|difficulty_base|difficulty_min|difficulty_max)\s*:\s*(.+?)\s*$`)
	ruleIDPattern       = regexp.MustCompile(`RG-[A-Z]+-[0-9]{3}`)
)

type guideMetadata struct {
	GuidanceType   string
	Enforcement    string
	Taxonomy       string
	SkillPrimary   string
	DifficultyBase string
	DifficultyMin  string
	DifficultyMax  string
}

func classifyGuidance(evidence []EvidenceChunk, top EvidenceChunk) (guideMetadata, error) {
	candidates := make([]EvidenceChunk, 0, len(evidence))
	candidates = append(candidates, top)

	for _, chunk := range evidence {
		if chunk.Source != top.Source {
			continue
		}

		if chunk.ChunkIndex == top.ChunkIndex && chunk.Content == top.Content {
			continue
		}
		candidates = append(candidates, chunk)
	}

	var metadata guideMetadata
	seen := map[string]bool{}

	for _, chunk := range candidates {
		applyMetadataLines(strings.Split(chunk.Content, "\n"), &metadata, seen)
	}

	if !hasRequiredMetadata(seen) {
		if err := fillMetadataFromSource(evidence, top, &metadata, seen); err != nil {
			return guideMetadata{}, err
		}
	}

	if !hasRequiredMetadata(seen) {
		for _, key := range []string{"type", "enforcement", "taxonomy", "skill_primary", "difficulty_base"} {
			if !seen[key] {
				return guideMetadata{}, fmt.Errorf("missing required metadata field %q in retrieved evidence", key)
			}
		}
	}

	if metadata.GuidanceType == GuidanceTypeRecommendation && metadata.Enforcement == EnforcementBlock {
		return guideMetadata{}, errors.New("invalid metadata: recommendations cannot use block enforcement")
	}

	if metadata.DifficultyMin != "" && !slices.Contains(ValidDiscoDifficulties, metadata.DifficultyMin) {
		return guideMetadata{}, fmt.Errorf("invalid difficulty_min %q", metadata.DifficultyMin)
	}

	if metadata.DifficultyMax != "" && !slices.Contains(ValidDiscoDifficulties, metadata.DifficultyMax) {
		return guideMetadata{}, fmt.Errorf("invalid difficulty_max %q", metadata.DifficultyMax)
	}

	if metadata.DifficultyMin != "" && metadata.DifficultyMax != "" {
		if difficultyIndex(metadata.DifficultyMin) > difficultyIndex(metadata.DifficultyMax) {
			return guideMetadata{}, fmt.Errorf("invalid difficulty bounds: min %q is higher than max %q", metadata.DifficultyMin, metadata.DifficultyMax)
		}
	}
	return metadata, nil
}

func hasRequiredMetadata(seen map[string]bool) bool {
	for _, key := range []string{"type", "enforcement", "taxonomy", "skill_primary", "difficulty_base"} {
		if !seen[key] {
			return false
		}
	}
	return true
}

func applyMetadataLines(lines []string, metadata *guideMetadata, seen map[string]bool) {
	for _, line := range lines {
		matches := metadataLinePattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) != 3 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(matches[1]))
		if seen[key] {
			continue
		}

		value := strings.TrimSpace(matches[2])

		switch key {
		case "type":
			v := strings.ToLower(value)
			if v == GuidanceTypeRule || v == GuidanceTypeRecommendation {
				metadata.GuidanceType = v
				seen[key] = true
			}
		case "enforcement":
			v := strings.ToLower(value)
			if v == EnforcementBlock || v == EnforcementWarn || v == EnforcementInfo {
				metadata.Enforcement = v
				seen[key] = true
			}
		case "taxonomy":
			if value != "" {
				metadata.Taxonomy = strings.ToLower(value)
				seen[key] = true
			}
		case "skill_primary":
			if value != "" {
				metadata.SkillPrimary = value
				seen[key] = true
			}
		case "difficulty_base":
			if slices.Contains(ValidDiscoDifficulties, value) {
				metadata.DifficultyBase = value
				seen[key] = true
			}
		case "difficulty_min":
			if value != "" {
				metadata.DifficultyMin = value
				seen[key] = true
			}
		case "difficulty_max":
			if value != "" {
				metadata.DifficultyMax = value
				seen[key] = true
			}
		}
	}
}

func fillMetadataFromSource(evidence []EvidenceChunk, top EvidenceChunk, metadata *guideMetadata, seen map[string]bool) error {
	ruleID := ruleIDPattern.FindString(top.HeadingPath)
	sourcePath := strings.TrimSpace(top.Source)

	if ruleID == "" {
		for _, chunk := range evidence {
			candidateRuleID := ruleIDPattern.FindString(chunk.HeadingPath)
			if candidateRuleID == "" {
				continue
			}

			ruleID = candidateRuleID
			if sourcePath == "" {
				sourcePath = strings.TrimSpace(chunk.Source)
			}
			break
		}
	}

	if ruleID == "" || sourcePath == "" {
		return nil
	}

	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	headingPrefix := "## " + ruleID + " - "
	start := -1

	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), headingPrefix) {
			start = i + 1
			break
		}
	}

	if start == -1 {
		return nil
	}

	sectionLines := make([]string, 0, 32)
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "## RG-") {
			break
		}
		sectionLines = append(sectionLines, lines[i])
	}

	applyMetadataLines(sectionLines, metadata, seen)
	return nil
}

func difficultyIndex(level string) int {
	for i, v := range ValidDiscoDifficulties {
		if v == level {
			return i
		}
	}
	return -1
}
