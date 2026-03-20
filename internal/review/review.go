package review

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultMinConfidence         = 0.65
	defaultMinEvidenceScore      = 0.60
	highSeverityMinConfidence    = 0.75
	highSeverityMinEvidenceScore = 0.70
	defaultHunkEvaluationTimeout = 90 * time.Second
)

type RetrieveFunc func(ctx context.Context, query string) ([]EvidenceChunk, error)

type EvaluateFunc func(ctx context.Context, input EvaluationInput) (EvaluationResult, error)

type ProgressFunc func(event ReviewEvent)

type Engine struct {
	retrieve RetrieveFunc
	evaluate EvaluateFunc
	progress ProgressFunc
}

func NewEngine(retrieve RetrieveFunc, evaluate EvaluateFunc, progress ProgressFunc) *Engine {
	return &Engine{
		retrieve: retrieve,
		evaluate: evaluate,
		progress: progress,
	}
}

func (e *Engine) ReviewDiff(ctx context.Context, diff string) (ReviewResult, error) {
	return reviewDiffCore(ctx, diff, e.retrieve, e.evaluate, e.progress)
}

type narrativeRequest struct {
	eventType    string
	skill        string
	difficulty   string
	content      string
	severity     string
	failureClass string
	guidanceType string
	blocking     bool
	current      int
	total        int
	file         string
	line         int
}

func reviewDiffCore(ctx context.Context, diff string, retrieve RetrieveFunc, evaluate EvaluateFunc, progress ProgressFunc) (ReviewResult, error) {
	if retrieve == nil {
		return ReviewResult{}, fmt.Errorf("could not run review: retrieve function is nil")
	}

	if evaluate == nil {
		return ReviewResult{}, fmt.Errorf("could not run review: evaluate function is nil")
	}

	hunks, err := ParseUnifiedDiff(diff)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("could not parse unified diff: %w", err)
	}

	result := ReviewResult{
		Summary: ReviewSummary{
			HunksScanned: 0,
		},
		Findings: make([]Finding, 0, 0),
		Checks:   make([]SkillCheck, 0, 0),
	}

	filesSeen := make(map[string]struct{}, len(hunks))
	findingKeys := make(map[string]struct{}, 0)
	totalReviewable := countReviewableHunks(hunks)
	currentReviewable := 0
	checksSinceNarrative := 0

	emitNarrative := func(req narrativeRequest, force bool) {
		if !force && checksSinceNarrative < 2 {
			return
		}

		req.current = currentReviewable
		req.total = totalReviewable
		emitNarrativeEvent(progress, req)
		checksSinceNarrative = 0
	}

	for _, hunk := range hunks {
		if !isReviewableFile(hunk.File) {
			continue
		}

		currentReviewable++
		checksSinceNarrative++
		if progress != nil {
			progress(ReviewEvent{
				Type:    EventTypeProgress,
				Current: currentReviewable,
				Total:   totalReviewable,
				File:    hunk.File,
				Line:    hunk.Line,
				Phase:   "evaluate_hunk",
			})
		}

		result.Summary.HunksScanned++

		if hunk.Line <= 0 {
			continue
		}

		if _, exists := filesSeen[hunk.File]; !exists {
			filesSeen[hunk.File] = struct{}{}
		}

		query := buildHunkQuery(hunk)
		evidence, err := retrieve(ctx, query)
		if err != nil {
			return ReviewResult{}, fmt.Errorf("could not retrieve style context for %s:%d: %w", hunk.File, hunk.Line, err)
		}

		if len(evidence) == 0 {
			result.Summary.NoApplicableRule++
			emitNarrative(narrativeRequest{
				eventType:  NarrativeEventNoRule,
				skill:      "Inland Empire",
				difficulty: "Easy",
				severity:   "none",
				file:       hunk.File,
				line:       hunk.Line,
			}, false)
			continue
		}

		result.Summary.HunksEvaluated++

		evalCtx, evalCancel := context.WithTimeout(ctx, defaultHunkEvaluationTimeout)
		evaluation, err := evaluate(evalCtx, EvaluationInput{
			File:     hunk.File,
			Line:     hunk.Line,
			Hunk:     hunk.Content,
			Evidence: evidence,
		})
		evalCancel()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				result.Summary.HunksSkippedTimeout++
				emitNarrative(narrativeRequest{
					eventType:  NarrativeEventTimeout,
					skill:      "Reaction Speed",
					difficulty: "Medium",
					severity:   "none",
					file:       hunk.File,
					line:       hunk.Line,
				}, true)
				continue
			}

			result.Summary.HunksModelError++
			emitNarrative(narrativeRequest{
				eventType:  NarrativeEventModelError,
				skill:      "Shivers",
				difficulty: "Medium",
				severity:   "none",
				file:       hunk.File,
				line:       hunk.Line,
			}, false)
			continue
		}

		if !evaluation.IsViolation {
			emitNarrative(narrativeRequest{
				eventType:  NarrativeEventSuccess,
				skill:      "Composure",
				difficulty: "Easy",
				severity:   "none",
				file:       hunk.File,
				line:       hunk.Line,
			}, false)
			continue
		}

		topEvidence := selectTopEvidence(evidence)
		evidenceScore := topEvidence.Score
		if !passesConfidenceGate(evaluation.Confidence, evidenceScore, evaluation.Severity) {
			result.Summary.HunksFilteredLowConfidence++
			emitNarrative(narrativeRequest{
				eventType:  NarrativeEventFiltered,
				skill:      "Perception",
				difficulty: "Easy",
				severity:   evaluation.Severity,
				file:       hunk.File,
				line:       hunk.Line,
			}, false)
			continue
		}

		metadata, err := classifyGuidance(evidence, topEvidence)
		if err != nil {
			return ReviewResult{}, fmt.Errorf("could not classify style metadata for %s:%d: %w", hunk.File, hunk.Line, err)
		}

		technicalMessage := normalizeMessage(evaluation.TechnicalMessage, evaluation.Message)
		discoMessage := strings.TrimSpace(evaluation.DiscoMessage)

		taxonomy := strings.TrimSpace(evaluation.Taxonomy)
		if taxonomy == "" {
			taxonomy = metadata.Taxonomy
		}

		if taxonomy == "" {
			taxonomy = "general"
		}

		kind := FindingKindViolation
		failureClass := FailureClassHard
		blocking := metadata.Enforcement == EnforcementBlock
		if metadata.GuidanceType == GuidanceTypeRecommendation {
			kind = FindingKindRecommendation
			failureClass = FailureClassSoft
			blocking = false
		}

		key := strings.Join([]string{hunk.File, strings.TrimSpace(evaluation.Rule), technicalMessage}, "|")
		if _, exists := findingKeys[key]; exists {
			// dedupe on rule+location+message to avoid repeated noise from similar retrieval slices
			continue
		}

		findingKeys[key] = struct{}{}

		finding := Finding{
			Kind:             kind,
			GuidanceType:     metadata.GuidanceType,
			Enforcement:      metadata.Enforcement,
			SkillPrimary:     metadata.SkillPrimary,
			DifficultyBase:   metadata.DifficultyBase,
			DifficultyMin:    metadata.DifficultyMin,
			DifficultyMax:    metadata.DifficultyMax,
			FailureClass:     failureClass,
			Blocking:         blocking,
			File:             hunk.File,
			Line:             hunk.Line,
			Severity:         normalizeSeverity(evaluation.Severity),
			Rule:             strings.TrimSpace(evaluation.Rule),
			Message:          technicalMessage,
			TechnicalMessage: technicalMessage,
			DiscoMessage:     discoMessage,
			Taxonomy:         taxonomy,
			Confidence:       normalizeConfidence(evaluation.Confidence),
			EvidenceScore:    evidenceScore,
			Citation: Citation{
				Source:      topEvidence.Source,
				HeadingPath: topEvidence.HeadingPath,
				ChunkIndex:  topEvidence.ChunkIndex,
				Score:       topEvidence.Score,
			},
			Hunk:        hunk.Content,
			GoodExample: extractGoodExample(evidence, strings.TrimSpace(evaluation.Rule)),
		}

		result.Findings = append(result.Findings, finding)
		result.Checks = append(result.Checks, findingToSkillCheck(finding))

		if finding.FailureClass == FailureClassSoft {
			// recommendations stay visible but non-blocking to preserve merge flow
			result.Summary.Recommendations++
			result.Summary.SoftFailures++
			emitNarrative(narrativeRequest{
				eventType:    NarrativeEventSoftFailure,
				skill:        metadata.SkillPrimary,
				difficulty:   metadata.DifficultyBase,
				severity:     finding.Severity,
				failureClass: finding.FailureClass,
				guidanceType: finding.GuidanceType,
				blocking:     finding.Blocking,
				file:         hunk.File,
				line:         hunk.Line,
			}, true)
			continue
		}

		result.Summary.Violations++
		if finding.Blocking {
			result.Summary.BlockingFindings++
			emitNarrative(narrativeRequest{
				eventType:    NarrativeEventHardFailure,
				skill:        metadata.SkillPrimary,
				difficulty:   metadata.DifficultyBase,
				severity:     finding.Severity,
				failureClass: finding.FailureClass,
				guidanceType: finding.GuidanceType,
				blocking:     finding.Blocking,
				file:         hunk.File,
				line:         hunk.Line,
			}, true)
		} else {
			emitNarrative(narrativeRequest{
				eventType:    NarrativeEventWarningFailure,
				skill:        metadata.SkillPrimary,
				difficulty:   metadata.DifficultyBase,
				severity:     finding.Severity,
				failureClass: finding.FailureClass,
				guidanceType: finding.GuidanceType,
				blocking:     finding.Blocking,
				file:         hunk.File,
				line:         hunk.Line,
			}, true)
		}
	}

	result.Summary.FilesScanned = len(filesSeen)
	return result, nil
}

func emitNarrativeEvent(progress ProgressFunc, req narrativeRequest) {
	if progress == nil {
		return
	}

	progress(ReviewEvent{
		Type:         EventTypeNarrative,
		Current:      req.current,
		Total:        req.total,
		File:         strings.TrimSpace(req.file),
		Line:         req.line,
		EventType:    req.eventType,
		Skill:        strings.TrimSpace(req.skill),
		Difficulty:   strings.TrimSpace(req.difficulty),
		Content:      strings.TrimSpace(req.content),
		Severity:     strings.TrimSpace(req.severity),
		FailureClass: strings.TrimSpace(req.failureClass),
		GuidanceType: strings.TrimSpace(req.guidanceType),
		Blocking:     req.blocking,
		NonBlocking:  true,
	})
}
