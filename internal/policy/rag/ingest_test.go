package rag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateStyleGuideMetadata(t *testing.T) {
	t.Parallel()

	t.Run("accepts valid rule and recommendation sections", func(t *testing.T) {
		t.Parallel()

		content := `## RG-ERR-001 - Wrap propagated errors
Type: rule
Enforcement: block
Taxonomy: error_handling
Skill_Primary: Logic
Difficulty_Base: Challenging

## RG-REC-001 - Prefer maintained packages
Type: recommendation
Enforcement: info
Taxonomy: readability
Skill_Primary: Rhetoric
Difficulty_Base: Easy
Difficulty_Min: Trivial
Difficulty_Max: Medium
`

		err := validateStyleGuideMetadata(content)
		require.NoError(t, err)
	})

	t.Run("fails when required metadata is missing", func(t *testing.T) {
		t.Parallel()

		content := `## RG-ERR-001 - Wrap propagated errors
Type: rule
Enforcement: block
Taxonomy: error_handling
Difficulty_Base: Challenging
`

		err := validateStyleGuideMetadata(content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required field \"skill_primary\"")
	})

	t.Run("fails when recommendation uses block enforcement", func(t *testing.T) {
		t.Parallel()

		content := `## RG-REC-001 - Prefer maintained packages
Type: recommendation
Enforcement: block
Taxonomy: readability
Skill_Primary: Rhetoric
Difficulty_Base: Easy
`

		err := validateStyleGuideMetadata(content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "recommendations cannot use block enforcement")
	})

	t.Run("fails on invalid difficulty bounds", func(t *testing.T) {
		t.Parallel()

		content := `## RG-ERR-001 - Wrap propagated errors
Type: rule
Enforcement: block
Taxonomy: error_handling
Skill_Primary: Logic
Difficulty_Base: Challenging
Difficulty_Min: Legendary
Difficulty_Max: Medium
`

		err := validateStyleGuideMetadata(content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "difficulty_min")
	})
}
