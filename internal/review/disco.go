package review

import "strings"

func findingToSkillCheck(f Finding) SkillCheck {
	skill := strings.TrimSpace(f.SkillPrimary)
	category := mapCategory(skill)
	difficulty := computeDifficulty(f)

	if f.FailureClass == FailureClassSoft {
		difficulty = capSoftFailureDifficulty(difficulty)
	}

	content := strings.TrimSpace(f.DiscoMessage)
	technical := strings.TrimSpace(f.TechnicalMessage)

	if technical == "" {
		technical = strings.TrimSpace(f.Message)
	}

	return SkillCheck{
		Skill:            skill,
		Category:         category,
		Difficulty:       difficulty,
		Success:          false,
		Content:          content,
		TechnicalMessage: technical,
		Rule:             f.Rule,
		Severity:         f.Severity,
		GuidanceType:     f.GuidanceType,
		Enforcement:      f.Enforcement,
		SkillPrimary:     f.SkillPrimary,
		DifficultyBase:   f.DifficultyBase,
		DifficultyMin:    f.DifficultyMin,
		DifficultyMax:    f.DifficultyMax,
		FailureClass:     f.FailureClass,
		Blocking:         f.Blocking,
		Citation:         f.Citation,
		File:             f.File,
		Line:             f.Line,
		GoodExample:      strings.TrimSpace(f.GoodExample),
	}
}

// SkillCheckFromFinding maps a review finding into Disco-style mechanics output
func SkillCheckFromFinding(f Finding) SkillCheck {
	return findingToSkillCheck(f)
}

func computeDifficulty(f Finding) string {
	idx := difficultyIndex(f.DifficultyBase)

	s := strings.ToLower(strings.TrimSpace(f.Severity))
	switch s {
	case "high":
		idx += 2
	case "medium":
		idx++
	}

	if f.Confidence >= 0.85 {
		idx++
	} else if f.Confidence < 0.65 {
		idx--
	}

	if f.EvidenceScore >= 0.85 {
		idx++
	} else if f.EvidenceScore < 0.70 {
		idx--
	}

	if f.DifficultyMin != "" {
		minIdx := difficultyIndex(f.DifficultyMin)
		if minIdx > idx {
			idx = minIdx
		}
	}

	if f.DifficultyMax != "" {
		maxIdx := difficultyIndex(f.DifficultyMax)
		if maxIdx >= 0 && idx > maxIdx {
			idx = maxIdx
		}
	} else {
		legendaryIdx := difficultyIndex("Legendary")
		if idx > legendaryIdx {
			idx = legendaryIdx
		}
	}

	if idx < 0 {
		idx = 0
	}

	if idx >= len(ValidDiscoDifficulties) {
		idx = len(ValidDiscoDifficulties) - 1
	}

	return ValidDiscoDifficulties[idx]
}

func mapCategory(skill string) string {
	switch strings.TrimSpace(skill) {
	case "Logic", "Encyclopedia", "Rhetoric", "Drama", "Conceptualization", "Visual Calculus":
		return "INTELLECT"
	case "Volition", "Inland Empire", "Empathy", "Authority", "Esprit de Corps", "Suggestion":
		return "PSYCHE"
	case "Endurance", "Pain Threshold", "Physical Instrument", "Electrochemistry", "Shivers", "Half Light":
		return "PHYSIQUE"
	case "Hand/Eye Coordination", "Perception", "Reaction Speed", "Savoir Faire", "Interfacing", "Composure":
		return "MOTORICS"
	default:
		return "INTELLECT"
	}
}

func capSoftFailureDifficulty(difficulty string) string {
	switch strings.TrimSpace(difficulty) {
	case "Trivial", "Easy", "Medium":
		return difficulty
	default:
		return "Medium"
	}
}
