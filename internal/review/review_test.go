package review

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewDiff(t *testing.T) {
	t.Parallel()

	diff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -3,1 +3,2 @@
 return err
+fmt.Println(err)`

	t.Run("no applicable rule increments summary", func(t *testing.T) {
		t.Parallel()

		engine := NewEngine(
			func(_ context.Context, _ string) ([]EvidenceChunk, error) {
				return nil, nil
			},
			func(_ context.Context, _ EvaluationInput) (EvaluationResult, error) {
				return EvaluationResult{}, nil
			},
			nil,
		)

		result, err := engine.ReviewDiff(context.Background(), diff)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Summary.NoApplicableRule)
		require.Len(t, result.Findings, 0)
	})

	t.Run("violation includes citation", func(t *testing.T) {
		t.Parallel()

		engine := NewEngine(
			func(_ context.Context, _ string) ([]EvidenceChunk, error) {
				return []EvidenceChunk{{
					Source:      "styleguides/disco/style.md",
					HeadingPath: "errors > wrapping",
					ChunkIndex:  2,
					Score:       0.91,
					Content:     "Type: rule\nEnforcement: block\nTaxonomy: error_handling\nSkill_Primary: Logic\nDifficulty_Base: Challenging\nalways wrap errors",
				}}, nil
			},
			func(_ context.Context, _ EvaluationInput) (EvaluationResult, error) {
				return EvaluationResult{
					IsViolation: true,
					Severity:    "high",
					Rule:        "error wrapping",
					Message:     "missing context when returning error",
					Confidence:  0.9,
				}, nil
			},
			nil,
		)

		result, err := engine.ReviewDiff(context.Background(), diff)
		require.NoError(t, err)
		require.Len(t, result.Findings, 1)
		require.Len(t, result.Checks, 1)
		assert.Equal(t, "violation", result.Findings[0].Kind)
		assert.Equal(t, GuidanceTypeRule, result.Findings[0].GuidanceType)
		assert.Equal(t, FailureClassHard, result.Findings[0].FailureClass)
		assert.True(t, result.Findings[0].Blocking)
		assert.Equal(t, "high", result.Findings[0].Severity)
		assert.Equal(t, 1, result.Summary.Violations)
		assert.Equal(t, 1, result.Summary.BlockingFindings)
		assert.Equal(t, "styleguides/disco/style.md", result.Findings[0].Citation.Source)
	})

	t.Run("recommendation becomes soft advisory failure", func(t *testing.T) {
		t.Parallel()

		engine := NewEngine(
			func(_ context.Context, _ string) ([]EvidenceChunk, error) {
				return []EvidenceChunk{{
					Source:      "styleguides/disco/style.md",
					HeadingPath: "performance",
					ChunkIndex:  3,
					Score:       0.82,
					Content:     "Type: recommendation\nEnforcement: info\nTaxonomy: performance\nSkill_Primary: Perception\nDifficulty_Base: Easy\nprefer strconv over fmt",
				}}, nil
			},
			func(_ context.Context, _ EvaluationInput) (EvaluationResult, error) {
				return EvaluationResult{
					IsViolation: true,
					Severity:    "high",
					Rule:        "strconv over fmt",
					Message:     "prefer strconv for scalar conversion",
					Confidence:  0.9,
				}, nil
			},
			nil,
		)

		result, err := engine.ReviewDiff(context.Background(), diff)
		require.NoError(t, err)
		require.Len(t, result.Findings, 1)
		require.Len(t, result.Checks, 1)
		assert.Equal(t, "recommendation", result.Findings[0].Kind)
		assert.Equal(t, GuidanceTypeRecommendation, result.Findings[0].GuidanceType)
		assert.Equal(t, EnforcementInfo, result.Findings[0].Enforcement)
		assert.Equal(t, FailureClassSoft, result.Findings[0].FailureClass)
		assert.False(t, result.Findings[0].Blocking)
		assert.Equal(t, 0, result.Summary.Violations)
		assert.Equal(t, 1, result.Summary.Recommendations)
		assert.Equal(t, 1, result.Summary.SoftFailures)
		assert.Equal(t, 0, result.Summary.BlockingFindings)
	})

	t.Run("does not fallback disco message to technical message", func(t *testing.T) {
		t.Parallel()

		engine := NewEngine(
			func(_ context.Context, _ string) ([]EvidenceChunk, error) {
				return []EvidenceChunk{{
					Source:      "styleguides/disco/style.md",
					HeadingPath: "errors > wrapping",
					ChunkIndex:  2,
					Score:       0.91,
					Content:     "Type: rule\nEnforcement: block\nTaxonomy: error_handling\nSkill_Primary: Logic\nDifficulty_Base: Challenging\nalways wrap errors",
				}}, nil
			},
			func(_ context.Context, _ EvaluationInput) (EvaluationResult, error) {
				return EvaluationResult{
					IsViolation:      true,
					Severity:         "high",
					Rule:             "error wrapping",
					Message:          "missing context when returning error",
					TechnicalMessage: "missing context when returning error",
					DiscoMessage:     "",
					Confidence:       0.9,
				}, nil
			},
			nil,
		)

		result, err := engine.ReviewDiff(context.Background(), diff)
		require.NoError(t, err)
		require.Len(t, result.Findings, 1)
		assert.Equal(t, "", result.Findings[0].DiscoMessage)
		assert.Equal(t, "missing context when returning error", result.Findings[0].TechnicalMessage)
	})

	t.Run("dedupes repeated findings", func(t *testing.T) {
		t.Parallel()

		doubleDiff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,1 +1,1 @@
-a
+b
@@ -3,1 +3,1 @@
-x
+y`

		engine := NewEngine(
			func(_ context.Context, _ string) ([]EvidenceChunk, error) {
				return []EvidenceChunk{{
					Source:      "styleguides/disco/style.md",
					HeadingPath: "rules",
					ChunkIndex:  1,
					Score:       0.77,
					Content:     "Type: rule\nEnforcement: warn\nTaxonomy: readability\nSkill_Primary: Composure\nDifficulty_Base: Medium\nkeep naming clear",
				}}, nil
			},
			func(_ context.Context, _ EvaluationInput) (EvaluationResult, error) {
				return EvaluationResult{
					IsViolation: true,
					Severity:    "medium",
					Rule:        "naming",
					Message:     "use clearer identifier names",
					Confidence:  0.8,
				}, nil
			},
			nil,
		)

		result, err := engine.ReviewDiff(context.Background(), doubleDiff)
		require.NoError(t, err)
		require.Len(t, result.Findings, 1)
	})

	t.Run("evaluation error increments model error counter", func(t *testing.T) {
		t.Parallel()

		engine := NewEngine(
			func(_ context.Context, _ string) ([]EvidenceChunk, error) {
				return []EvidenceChunk{{
					Source:      "styleguides/disco/style.md",
					HeadingPath: "rules",
					ChunkIndex:  1,
					Score:       0.77,
					Content:     "Type: rule\nEnforcement: warn\nTaxonomy: readability\nSkill_Primary: Composure\nDifficulty_Base: Medium\nkeep naming clear",
				}}, nil
			},
			func(_ context.Context, _ EvaluationInput) (EvaluationResult, error) {
				return EvaluationResult{}, errors.New("model parse failure")
			},
			nil,
		)

		result, err := engine.ReviewDiff(context.Background(), diff)
		require.NoError(t, err)
		require.Len(t, result.Findings, 0)
		assert.Equal(t, 1, result.Summary.HunksModelError)
	})

	t.Run("low confidence violation is filtered", func(t *testing.T) {
		t.Parallel()

		engine := NewEngine(
			func(_ context.Context, _ string) ([]EvidenceChunk, error) {
				return []EvidenceChunk{{
					Source:      "styleguides/disco/style.md",
					HeadingPath: "rules",
					ChunkIndex:  1,
					Score:       0.90,
					Content:     "Type: rule\nEnforcement: warn\nTaxonomy: readability\nSkill_Primary: Composure\nDifficulty_Base: Medium\nkeep naming clear",
				}}, nil
			},
			func(_ context.Context, _ EvaluationInput) (EvaluationResult, error) {
				return EvaluationResult{
					IsViolation: true,
					Severity:    "medium",
					Rule:        "naming",
					Message:     "use clearer identifier names",
					Confidence:  0.30,
				}, nil
			},
			nil,
		)

		result, err := engine.ReviewDiff(context.Background(), diff)
		require.NoError(t, err)
		require.Len(t, result.Findings, 0)
		assert.Equal(t, 1, result.Summary.HunksFilteredLowConfidence)
	})

	t.Run("uses top scored evidence for gating and citation", func(t *testing.T) {
		t.Parallel()

		engine := NewEngine(
			func(_ context.Context, _ string) ([]EvidenceChunk, error) {
				return []EvidenceChunk{
					{
						Source:      "styleguides/disco/style.md",
						HeadingPath: "readability",
						ChunkIndex:  1,
						Score:       0.40,
						Content:     "Type: rule\nEnforcement: warn\nTaxonomy: readability\nSkill_Primary: Composure\nDifficulty_Base: Medium\nlower evidence",
					},
					{
						Source:      "styleguides/disco/style.md",
						HeadingPath: "errors",
						ChunkIndex:  2,
						Score:       0.91,
						Content:     "Type: rule\nEnforcement: block\nTaxonomy: error_handling\nSkill_Primary: Logic\nDifficulty_Base: Challenging\nhigher evidence",
					},
				}, nil
			},
			func(_ context.Context, _ EvaluationInput) (EvaluationResult, error) {
				return EvaluationResult{
					IsViolation: true,
					Severity:    "medium",
					Rule:        "error wrapping",
					Message:     "wrap returned errors",
					Confidence:  0.9,
				}, nil
			},
			nil,
		)

		result, err := engine.ReviewDiff(context.Background(), diff)
		require.NoError(t, err)
		require.Len(t, result.Findings, 1)
		assert.Equal(t, float32(0.91), result.Findings[0].EvidenceScore)
		assert.Equal(t, 2, result.Findings[0].Citation.ChunkIndex)
	})

	t.Run("prefers rule-section evidence over generic heading", func(t *testing.T) {
		t.Parallel()

		engine := NewEngine(
			func(_ context.Context, _ string) ([]EvidenceChunk, error) {
				return []EvidenceChunk{
					{
						Source:      "styleguides/disco/style.md",
						HeadingPath: "Disco Go Style Guide",
						ChunkIndex:  1,
						Score:       0.99,
						Content:     "Guide preamble",
					},
					{
						Source:      "styleguides/disco/style.md",
						HeadingPath: "Disco Go Style Guide > RG-ERR-001 - Wrap propagated errors with context",
						ChunkIndex:  2,
						Score:       0.90,
						Content:     "Type: rule\nEnforcement: block\nTaxonomy: error_handling\nSkill_Primary: Logic\nDifficulty_Base: Challenging",
					},
				}, nil
			},
			func(_ context.Context, _ EvaluationInput) (EvaluationResult, error) {
				return EvaluationResult{
					IsViolation: true,
					Severity:    "medium",
					Rule:        "error wrapping",
					Message:     "wrap returned errors",
					Confidence:  0.9,
				}, nil
			},
			nil,
		)

		result, err := engine.ReviewDiff(context.Background(), diff)
		require.NoError(t, err)
		require.Len(t, result.Findings, 1)
		assert.Equal(t, 2, result.Findings[0].Citation.ChunkIndex)
		assert.Equal(t, GuidanceTypeRule, result.Findings[0].GuidanceType)
		assert.Equal(t, EnforcementBlock, result.Findings[0].Enforcement)
	})

	t.Run("classifies metadata from cited top evidence only", func(t *testing.T) {
		t.Parallel()

		engine := NewEngine(
			func(_ context.Context, _ string) ([]EvidenceChunk, error) {
				return []EvidenceChunk{
					{
						Source:      "styleguides/disco/style.md",
						HeadingPath: "performance",
						ChunkIndex:  3,
						Score:       0.95,
						Content:     "Type: recommendation\nEnforcement: info\nTaxonomy: performance\nSkill_Primary: Perception\nDifficulty_Base: Easy\nprefer clear defaults",
					},
					{
						Source:      "styleguides/disco/style.md",
						HeadingPath: "errors",
						ChunkIndex:  1,
						Score:       0.40,
						Content:     "Type: rule\nEnforcement: block\nTaxonomy: error_handling\nSkill_Primary: Logic\nDifficulty_Base: Challenging\nwrap propagated errors",
					},
				}, nil
			},
			func(_ context.Context, _ EvaluationInput) (EvaluationResult, error) {
				return EvaluationResult{
					IsViolation: true,
					Severity:    "medium",
					Rule:        "consistency",
					Message:     "prefer consistency",
					Confidence:  0.9,
				}, nil
			},
			nil,
		)

		result, err := engine.ReviewDiff(context.Background(), diff)
		require.NoError(t, err)
		require.Len(t, result.Findings, 1)
		assert.Equal(t, 3, result.Findings[0].Citation.ChunkIndex)
		assert.Equal(t, GuidanceTypeRecommendation, result.Findings[0].GuidanceType)
		assert.Equal(t, EnforcementInfo, result.Findings[0].Enforcement)
		assert.Equal(t, FailureClassSoft, result.Findings[0].FailureClass)
		assert.False(t, result.Findings[0].Blocking)
	})

	t.Run("uses top heading neighbors when top evidence omits metadata", func(t *testing.T) {
		t.Parallel()

		engine := NewEngine(
			func(_ context.Context, _ string) ([]EvidenceChunk, error) {
				return []EvidenceChunk{
					{
						Source:      "styleguides/disco/style.md",
						HeadingPath: "error handling",
						ChunkIndex:  11,
						Score:       0.96,
						Content:     "### Statement\nWrap errors with context",
					},
					{
						Source:      "styleguides/disco/style.md",
						HeadingPath: "error handling",
						ChunkIndex:  10,
						Score:       0.92,
						Content:     "Type: rule\nEnforcement: block\nTaxonomy: error_handling\nSkill_Primary: Logic\nDifficulty_Base: Challenging",
					},
					{
						Source:      "styleguides/disco/style.md",
						HeadingPath: "performance",
						ChunkIndex:  3,
						Score:       0.40,
						Content:     "Type: recommendation\nEnforcement: info\nTaxonomy: performance\nSkill_Primary: Perception\nDifficulty_Base: Easy",
					},
				}, nil
			},
			func(_ context.Context, _ EvaluationInput) (EvaluationResult, error) {
				return EvaluationResult{
					IsViolation: true,
					Severity:    "medium",
					Rule:        "error wrapping",
					Message:     "wrap returned errors",
					Confidence:  0.9,
				}, nil
			},
			nil,
		)

		result, err := engine.ReviewDiff(context.Background(), diff)
		require.NoError(t, err)
		require.Len(t, result.Findings, 1)
		assert.Equal(t, 11, result.Findings[0].Citation.ChunkIndex)
		assert.Equal(t, GuidanceTypeRule, result.Findings[0].GuidanceType)
		assert.Equal(t, EnforcementBlock, result.Findings[0].Enforcement)
		assert.Equal(t, FailureClassHard, result.Findings[0].FailureClass)
		assert.True(t, result.Findings[0].Blocking)
	})

	t.Run("skips deletion-only hunk with zero target start", func(t *testing.T) {
		t.Parallel()

		deletionDiff := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -3,2 +0,0 @@
-old()
-gone()`

		engine := NewEngine(
			func(_ context.Context, _ string) ([]EvidenceChunk, error) {
				return []EvidenceChunk{{
					Source:      "styleguides/disco/style.md",
					HeadingPath: "rules",
					ChunkIndex:  1,
					Score:       0.90,
					Content:     "Type: rule\nEnforcement: warn\nTaxonomy: readability\nSkill_Primary: Composure\nDifficulty_Base: Medium\nkeep naming clear",
				}}, nil
			},
			func(_ context.Context, _ EvaluationInput) (EvaluationResult, error) {
				return EvaluationResult{IsViolation: true, Confidence: 0.9}, nil
			},
			nil,
		)

		result, err := engine.ReviewDiff(context.Background(), deletionDiff)
		require.NoError(t, err)
		assert.Equal(t, 1, result.Summary.HunksScanned)
		assert.Equal(t, 0, result.Summary.HunksEvaluated)
		require.Len(t, result.Findings, 0)
	})

	t.Run("emits narrative events during processing", func(t *testing.T) {
		t.Parallel()

		events := make([]ReviewEvent, 0, 8)
		engine := NewEngine(
			func(_ context.Context, _ string) ([]EvidenceChunk, error) {
				return []EvidenceChunk{{
					Source:      "styleguides/disco/style.md",
					HeadingPath: "errors > wrapping",
					ChunkIndex:  2,
					Score:       0.91,
					Content:     "Type: rule\nEnforcement: block\nTaxonomy: error_handling\nSkill_Primary: Logic\nDifficulty_Base: Challenging\nalways wrap errors",
				}}, nil
			},
			func(_ context.Context, _ EvaluationInput) (EvaluationResult, error) {
				return EvaluationResult{
					IsViolation: true,
					Severity:    "high",
					Rule:        "error wrapping",
					Message:     "wrap returned errors",
					Confidence:  0.92,
				}, nil
			},
			func(event ReviewEvent) {
				events = append(events, event)
			},
		)

		_, err := engine.ReviewDiff(context.Background(), diff)
		require.NoError(t, err)

		narrativeCount := 0
		for _, event := range events {
			if event.Type == EventTypeNarrative {
				narrativeCount++
				assert.True(t, event.NonBlocking)
				assert.Equal(t, "", event.Content)
			}
		}

		assert.GreaterOrEqual(t, narrativeCount, 1)
	})
}

func TestClassifyGuidance(t *testing.T) {
	t.Parallel()

	t.Run("reads strict metadata from evidence", func(t *testing.T) {
		t.Parallel()

		evidence := EvidenceChunk{
			Content: "Type: recommendation\nEnforcement: info\nTaxonomy: performance\nSkill_Primary: Perception\nDifficulty_Base: Easy\nDifficulty_Min: Trivial\nDifficulty_Max: Medium",
		}

		metadata, err := classifyGuidance([]EvidenceChunk{evidence}, evidence)
		require.NoError(t, err)
		assert.Equal(t, GuidanceTypeRecommendation, metadata.GuidanceType)
		assert.Equal(t, EnforcementInfo, metadata.Enforcement)
		assert.Equal(t, "performance", metadata.Taxonomy)
		assert.Equal(t, "Perception", metadata.SkillPrimary)
		assert.Equal(t, "Easy", metadata.DifficultyBase)
		assert.Equal(t, "Trivial", metadata.DifficultyMin)
		assert.Equal(t, "Medium", metadata.DifficultyMax)
	})

	t.Run("fails when required metadata is absent", func(t *testing.T) {
		t.Parallel()

		evidence := EvidenceChunk{Content: "no metadata here"}
		_, err := classifyGuidance([]EvidenceChunk{evidence}, evidence)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required metadata field")
	})

	t.Run("fails when recommendation uses block enforcement", func(t *testing.T) {
		t.Parallel()

		evidence := EvidenceChunk{
			Content: "Type: recommendation\nEnforcement: block\nTaxonomy: performance\nSkill_Primary: Perception\nDifficulty_Base: Easy",
		}

		_, err := classifyGuidance([]EvidenceChunk{evidence}, evidence)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "recommendations cannot use block enforcement")
	})

	t.Run("falls back to cited rule section metadata from source file", func(t *testing.T) {
		t.Parallel()

		tmp := t.TempDir()
		guidePath := filepath.Join(tmp, "style.md")
		guideContent := "# Guide\n\n## RG-ERR-001 - Wrap propagated errors with context\n\nType: rule\nEnforcement: block\nTaxonomy: error_handling\nSkill_Primary: Logic\nDifficulty_Base: Challenging\n\n### Statement\n...\n"
		require.NoError(t, os.WriteFile(guidePath, []byte(guideContent), 0o644))

		top := EvidenceChunk{
			Source:      guidePath,
			HeadingPath: "Guide > RG-ERR-001 - Wrap propagated errors with context > Good",
			ChunkIndex:  9,
			Score:       0.91,
			Content:     "### Good\nreturn fmt.Errorf(\"could not load config: %w\", err)",
		}

		metadata, err := classifyGuidance([]EvidenceChunk{top}, top)
		require.NoError(t, err)
		assert.Equal(t, GuidanceTypeRule, metadata.GuidanceType)
		assert.Equal(t, EnforcementBlock, metadata.Enforcement)
		assert.Equal(t, "error_handling", metadata.Taxonomy)
		assert.Equal(t, "Logic", metadata.SkillPrimary)
		assert.Equal(t, "Challenging", metadata.DifficultyBase)
	})
}

func TestExtractGoodExample(t *testing.T) {
	t.Parallel()

	t.Run("selects matching rule good example", func(t *testing.T) {
		t.Parallel()

		evidence := []EvidenceChunk{
			{
				HeadingPath: "Guide > RG-ERR-001 - Wrap propagated errors with context > Bad",
				Score:       0.99,
				Content:     "### Bad\nreturn err",
			},
			{
				HeadingPath: "Guide > RG-ERR-001 - Wrap propagated errors with context > Good",
				Score:       0.91,
				Content:     "### Good\n```go\nreturn fmt.Errorf(\"could not load config: %w\", err)\n```",
			},
		}

		example := extractGoodExample(evidence, "RG-ERR-001")
		require.NotEmpty(t, example)
		assert.Contains(t, example, "fmt.Errorf")
		assert.NotContains(t, example, "### Good")
	})

	t.Run("returns empty when no good example exists", func(t *testing.T) {
		t.Parallel()

		evidence := []EvidenceChunk{
			{
				HeadingPath: "Guide > RG-ERR-001 - Wrap propagated errors with context > Statement",
				Score:       0.91,
				Content:     "Wrap errors with context",
			},
		}

		example := extractGoodExample(evidence, "RG-ERR-001")
		assert.Empty(t, example)
	})

	t.Run("falls back to source file good section when chunk is heading only", func(t *testing.T) {
		t.Parallel()

		styleGuide := "# Disco Go Style Guide\n\n" +
			"## RG-ERR-001 - Wrap propagated errors with context\n\n" +
			"type: rule\n" +
			"enforcement: block\n" +
			"taxonomy: error_handling\n" +
			"skill_primary: Logic\n" +
			"difficulty_base: Challenging\n\n" +
			"### Good\n\n" +
			"```go\n" +
			"return fmt.Errorf(\"could not save config %q: %w\", cfg.Name, err)\n" +
			"```\n\n" +
			"### Bad\n\n" +
			"return err\n"

		tempDir := t.TempDir()
		sourcePath := filepath.Join(tempDir, "style.md")
		require.NoError(t, os.WriteFile(sourcePath, []byte(styleGuide), 0o600))

		evidence := []EvidenceChunk{
			{
				Source:      sourcePath,
				HeadingPath: "Disco Go Style Guide > RG-ERR-001 - Wrap propagated errors with context > Good",
				Score:       0.95,
				Content:     "# Disco Go Style Guide\n## RG-ERR-001 - Wrap propagated errors with context\n### Good",
			},
		}

		example := extractGoodExample(evidence, "RG-ERR-001")
		require.NotEmpty(t, example)
		assert.Contains(t, example, "fmt.Errorf")
		assert.Contains(t, example, "```go")
		assert.NotContains(t, example, "## RG-ERR-001")
	})
}
