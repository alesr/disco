package rag

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const (
	difficultyTrivial     = "Trivial"
	difficultyEasy        = "Easy"
	difficultyMedium      = "Medium"
	difficultyChallenging = "Challenging"
	difficultyFormidable  = "Formidable"
	difficultyLegendary   = "Legendary"
	difficultyImpossible  = "Impossible"

	typeRule           = "rule"
	typeRecommendation = "recommendation"

	enforcementBlock = "block"
	enforcementInfo  = "info"
	enforcementWarn  = "warn"
)

var (
	ruleHeadingPattern = regexp.MustCompile(`^##\s+RG-[A-Z]+-\d+\s+-\s+`)

	validDifficulties = []string{
		difficultyTrivial,
		difficultyEasy,
		difficultyMedium,
		difficultyChallenging,
		difficultyFormidable,
		difficultyLegendary,
		difficultyImpossible,
	}

	difficultyOrder = map[string]int{
		difficultyTrivial:     0,
		difficultyEasy:        1,
		difficultyMedium:      2,
		difficultyChallenging: 3,
		difficultyFormidable:  4,
		difficultyLegendary:   5,
		difficultyImpossible:  6,
	}

	validTypes = map[string]struct{}{
		typeRule:           {},
		typeRecommendation: {},
	}

	validEnforcement = map[string]struct{}{
		enforcementBlock: {},
		enforcementWarn:  {},
		enforcementInfo:  {},
	}
)

func validateStyleGuideMetadata(content string) error {
	lines := strings.Split(content, "\n")

	type item struct {
		heading string
		line    int
		fields  map[string]string
	}

	var (
		items   []item
		current *item
	)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if ruleHeadingPattern.MatchString(trimmed) {
			items = append(items, item{heading: trimmed, line: i + 1, fields: map[string]string{}})
			current = &items[len(items)-1]
			continue
		}

		if current == nil || !strings.Contains(trimmed, ":") {
			continue
		}

		parts := strings.SplitN(trimmed, ":", 2)
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		if value == "" {
			continue
		}

		switch key {
		case "type", "enforcement", "taxonomy", "skill_primary", "difficulty_base", "difficulty_min", "difficulty_max", "id":
			current.fields[key] = value
		}
	}

	if len(items) == 0 {
		return errors.New("no rule sections found (expected headings like '## RG-...')")
	}

	for _, it := range items {
		required := []string{"type", "enforcement", "taxonomy", "skill_primary", "difficulty_base"}
		for _, key := range required {
			if strings.TrimSpace(it.fields[key]) == "" {
				return fmt.Errorf("%s (line %d): missing required field %q", it.heading, it.line, key)
			}
		}

		guidanceType := strings.ToLower(it.fields["type"])
		if _, ok := validTypes[guidanceType]; !ok {
			return fmt.Errorf("%s (line %d): invalid type %q", it.heading, it.line, it.fields["type"])
		}

		enforcement := strings.ToLower(it.fields["enforcement"])
		if _, ok := validEnforcement[enforcement]; !ok {
			return fmt.Errorf("%s (line %d): invalid enforcement %q", it.heading, it.line, it.fields["enforcement"])
		}

		if guidanceType == typeRecommendation && enforcement == enforcementBlock {
			return fmt.Errorf("%s (line %d): recommendations cannot use block enforcement", it.heading, it.line)
		}

		if !slices.Contains(validDifficulties, it.fields["difficulty_base"]) {
			return fmt.Errorf("%s (line %d): invalid difficulty_base %q", it.heading, it.line, it.fields["difficulty_base"])
		}

		if min := it.fields["difficulty_min"]; min != "" {
			if !slices.Contains(validDifficulties, min) {
				return fmt.Errorf("%s (line %d): invalid difficulty_min %q", it.heading, it.line, min)
			}
		}

		if max := it.fields["difficulty_max"]; max != "" {
			if !slices.Contains(validDifficulties, max) {
				return fmt.Errorf("%s (line %d): invalid difficulty_max %q", it.heading, it.line, max)
			}
		}

		min := it.fields["difficulty_min"]
		max := it.fields["difficulty_max"]
		if min != "" && max != "" && difficultyOrder[min] > difficultyOrder[max] {
			return fmt.Errorf("%s (line %d): difficulty_min %q is higher than difficulty_max %q", it.heading, it.line, min, max)
		}
	}
	return nil
}

func inferHeadingPath(chunk string) string {
	headings := make([]string, 0, 4)

	for line := range strings.SplitSeq(chunk, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !strings.HasPrefix(trimmed, "#") {
			continue
		}

		title := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		if title == "" {
			continue
		}

		headings = append(headings, title)
	}

	if len(headings) == 0 {
		return "unknown"
	}
	return strings.Join(headings, " > ")
}
