package kronkgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHunkReviewResult(t *testing.T) {
	t.Parallel()

	t.Run("valid payload parses", func(t *testing.T) {
		t.Parallel()

		raw := `{"is_violation":true,"severity":"medium","rule":"error wrapping","message":"wrap upstream errors","confidence":0.75}`

		parsed, err := parseHunkReviewResult(raw)
		require.NoError(t, err)
		assert.True(t, parsed.IsViolation)
		assert.Equal(t, "medium", parsed.Severity)
		assert.Equal(t, "error wrapping", parsed.Rule)
	})

	t.Run("invalid severity returns error", func(t *testing.T) {
		t.Parallel()

		raw := `{"is_violation":true,"severity":"catastrophic","rule":"x","message":"y","confidence":0.8}`

		_, err := parseHunkReviewResult(raw)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not validate hunk review severity")
	})

	t.Run("missing message returns error", func(t *testing.T) {
		t.Parallel()

		raw := `{"is_violation":true,"severity":"low","rule":"x","message":"","confidence":0.8}`

		_, err := parseHunkReviewResult(raw)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing message")
	})

	t.Run("non violation accepts none severity", func(t *testing.T) {
		t.Parallel()

		raw := `{"is_violation":false,"severity":"none","rule":"","message":"","confidence":0.1}`

		parsed, err := parseHunkReviewResult(raw)
		require.NoError(t, err)
		assert.False(t, parsed.IsViolation)
		assert.Equal(t, "none", parsed.Severity)
	})

	t.Run("parses fenced json with string fields", func(t *testing.T) {
		t.Parallel()

		raw := "```json\n{\"is_violation\":\"true\",\"severity\":\"High\",\"rule\":\"RG-ERR-001\",\"message\":\"wrap the error\",\"confidence\":\"0.82\"}\n```"

		parsed, err := parseHunkReviewResult(raw)
		require.NoError(t, err)
		assert.True(t, parsed.IsViolation)
		assert.Equal(t, "high", parsed.Severity)
		assert.Equal(t, "RG-ERR-001", parsed.Rule)
		assert.InDelta(t, 0.82, parsed.Confidence, 1e-5)
	})

	t.Run("maps critical severity to high", func(t *testing.T) {
		t.Parallel()

		raw := `{"is_violation":true,"severity":"critical","rule":"x","message":"y","confidence":0.9}`

		parsed, err := parseHunkReviewResult(raw)
		require.NoError(t, err)
		assert.Equal(t, "high", parsed.Severity)
	})
}

func TestParseNarrativeResult(t *testing.T) {
	t.Parallel()

	t.Run("valid payload parses", func(t *testing.T) {
		t.Parallel()

		raw := `{"voice_skill":"Shivers","difficulty":"Medium","stance":"eerie","content":"The code hums with static."}`
		parsed, err := parseNarrativeResult(raw)
		require.NoError(t, err)
		assert.Equal(t, "Shivers", parsed.VoiceSkill)
		assert.Equal(t, "Medium", parsed.Difficulty)
		assert.Equal(t, "eerie", parsed.Stance)
		assert.NotEmpty(t, parsed.Content)
	})

	t.Run("missing content returns error", func(t *testing.T) {
		t.Parallel()

		raw := `{"voice_skill":"Shivers","difficulty":"Medium","stance":"eerie","content":""}`
		_, err := parseNarrativeResult(raw)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing content")
	})

	t.Run("accepts fenced json and defaults missing fields", func(t *testing.T) {
		t.Parallel()

		raw := "```json\n{\"content\":\"The rain needles line 42 in manager.go.\"}\n```"
		parsed, err := parseNarrativeResult(raw)
		require.NoError(t, err)
		assert.Equal(t, "Volition", parsed.VoiceSkill)
		assert.Equal(t, "Medium", parsed.Difficulty)
		assert.Equal(t, "restrained confidence", parsed.Stance)
		assert.NotEmpty(t, parsed.Content)
	})

	t.Run("strips bracket wrapped narrative content", func(t *testing.T) {
		t.Parallel()

		raw := `{"voice_skill":"Logic","difficulty":"Medium","stance":"steady","content":"[The rain needles line 42 in manager.go.]"}`
		parsed, err := parseNarrativeResult(raw)
		require.NoError(t, err)
		assert.Equal(t, "The rain needles line 42 in manager.go.", parsed.Content)
	})
}

func TestIsTooSimilar(t *testing.T) {
	t.Parallel()

	t.Run("flags near duplicate lines", func(t *testing.T) {
		t.Parallel()

		candidate := "The code hums with static, a dead wire in the night"
		previous := []string{"The code hums with static. A dead wire in the night."}
		assert.True(t, isTooSimilar(candidate, previous))
	})

	t.Run("allows distinct lines", func(t *testing.T) {
		t.Parallel()

		candidate := "Authority barks: this breach blocks the exit"
		previous := []string{"Composure whispers that the line still holds"}
		assert.False(t, isTooSimilar(candidate, previous))
	})
}

func TestAnchoredNarration(t *testing.T) {
	t.Parallel()

	t.Run("accepts file or line anchors", func(t *testing.T) {
		t.Parallel()

		assert.True(t, AnchoredNarration("Logic points at manager.go and the broken edge", "internal/runtime/manager.go", 0))
		assert.True(t, AnchoredNarration("Perception catches line 42 choking on timeout", "internal/runtime/manager.go", 42))
	})

	t.Run("requires more than one vague review word", func(t *testing.T) {
		t.Parallel()

		assert.False(t, AnchoredNarration("The check is haunted tonight", "", 0))
		assert.True(t, AnchoredNarration("The check trips on evidence and timeout", "", 0))
	})
}

func TestPassesNarrationQuality(t *testing.T) {
	t.Parallel()

	input := NarrativeInput{
		EventType:    "timeout",
		File:         "internal/runtime/manager.go",
		Line:         57,
		Content:      "REACTION SPEED: The thought outran the clock",
		Severity:     "none",
		FailureClass: "",
		GuidanceType: "",
		Blocking:     false,
	}

	t.Run("accepts grounded line", func(t *testing.T) {
		t.Parallel()

		line := "Reaction Speed slams the brakes: manager.go line 57 hit the timeout clock again."
		assert.True(t, passesNarrationQuality(line, input))
	})

	t.Run("rejects generic noir filler", func(t *testing.T) {
		t.Parallel()

		line := "The static hums in the dark and the silence whispers from the void tonight."
		assert.False(t, passesNarrationQuality(line, input))
	})
}

func TestLoadGenerationProviderConfig(t *testing.T) {
	t.Run("defaults to kronk provider", func(t *testing.T) {
		t.Setenv("GENERATION_PROVIDER", "")
		t.Setenv("MISTRAL_API_KEY", "")
		t.Setenv("MISTRAL_MODEL", "")
		t.Setenv("MISTRAL_BASE_URL", "")

		cfg := loadGenerationProviderConfig()
		assert.Equal(t, providerKronk, cfg.Provider)
	})

	t.Run("reads mistral settings", func(t *testing.T) {
		t.Setenv("GENERATION_PROVIDER", "mistral")
		t.Setenv("MISTRAL_API_KEY", "key")
		t.Setenv("MISTRAL_MODEL", "mistral-small-latest")
		t.Setenv("MISTRAL_BASE_URL", "https://api.mistral.ai")

		cfg := loadGenerationProviderConfig()
		assert.Equal(t, providerMistral, cfg.Provider)
		assert.Equal(t, "key", cfg.MistralAPIKey)
		assert.Equal(t, "mistral-small-latest", cfg.MistralModel)
	})
}

func TestExtractMistralContent(t *testing.T) {
	t.Parallel()

	t.Run("extracts content from first choice", func(t *testing.T) {
		t.Parallel()

		content, err := extractMistralContent(mistralChatResponse{
			Choices: []struct {
				Message struct {
					Content any `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content any `json:"content"`
				}{Content: `{"ok":true}`}},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, `{"ok":true}`, content)
	})

	t.Run("returns error when choice content is non-string", func(t *testing.T) {
		t.Parallel()

		_, err := extractMistralContent(mistralChatResponse{
			Choices: []struct {
				Message struct {
					Content any `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content any `json:"content"`
				}{Content: map[string]any{"a": 1}}},
			},
		})
		require.Error(t, err)
	})
}
