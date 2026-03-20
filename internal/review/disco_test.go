package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillCheckFromFinding(t *testing.T) {
	t.Parallel()

	t.Run("uses skill primary and computes adjusted difficulty", func(t *testing.T) {
		t.Parallel()

		f := Finding{
			File:           "main.go",
			Line:           10,
			Severity:       "high",
			Rule:           "error handling",
			Message:        "missing wrapped error",
			Confidence:     0.92,
			EvidenceScore:  0.92,
			SkillPrimary:   "Logic",
			DifficultyBase: "Challenging",
			FailureClass:   FailureClassHard,
		}

		check := SkillCheckFromFinding(f)
		assert.Equal(t, "Legendary", check.Difficulty)
		assert.Equal(t, "Logic", check.Skill)
		assert.Equal(t, "INTELLECT", check.Category)
		assert.False(t, check.Success)
	})

	t.Run("maps skill primary to category", func(t *testing.T) {
		t.Parallel()

		f := Finding{
			Severity:       "medium",
			Rule:           "auth validation",
			Message:        "potential security bypass",
			Confidence:     0.66,
			EvidenceScore:  0.81,
			SkillPrimary:   "Half Light",
			DifficultyBase: "Medium",
		}

		check := SkillCheckFromFinding(f)
		assert.Equal(t, "Half Light", check.Skill)
		assert.Equal(t, "PHYSIQUE", check.Category)
		require.NotEmpty(t, check.Difficulty)
	})

	t.Run("soft failure difficulty is capped at medium", func(t *testing.T) {
		t.Parallel()

		f := Finding{
			Severity:       "high",
			Rule:           "strconv over fmt",
			Message:        "prefer strconv",
			Confidence:     0.95,
			EvidenceScore:  0.95,
			SkillPrimary:   "Perception",
			DifficultyBase: "Legendary",
			FailureClass:   FailureClassSoft,
		}

		check := SkillCheckFromFinding(f)
		assert.Equal(t, "Medium", check.Difficulty)
		assert.False(t, check.Success)
	})

	t.Run("keeps disco content empty when disco message is missing", func(t *testing.T) {
		t.Parallel()

		f := Finding{
			Severity:         "medium",
			Rule:             "ctx first",
			Message:          "context is not first argument",
			TechnicalMessage: "context is not first argument",
			DiscoMessage:     "",
			Confidence:       0.9,
			EvidenceScore:    0.9,
			SkillPrimary:     "Interfacing",
			DifficultyBase:   "Easy",
		}

		check := SkillCheckFromFinding(f)
		assert.Equal(t, "", check.Content)
		assert.Equal(t, "context is not first argument", check.TechnicalMessage)
	})

	t.Run("keeps disco message as provided without guardrails", func(t *testing.T) {
		t.Parallel()

		f := Finding{
			Severity:         "high",
			Rule:             "RG-ERR-001",
			Message:          "missing context when returning error",
			TechnicalMessage: "missing context when returning error",
			DiscoMessage:     "This code violates RG-ERR-001 and should wrap the error",
			Confidence:       0.9,
			EvidenceScore:    0.9,
			SkillPrimary:     "Logic",
			DifficultyBase:   "Challenging",
		}

		check := SkillCheckFromFinding(f)
		assert.Equal(t, "This code violates RG-ERR-001 and should wrap the error", check.Content)
		assert.Equal(t, "missing context when returning error", check.TechnicalMessage)
	})
}
