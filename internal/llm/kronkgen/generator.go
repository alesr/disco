package kronkgen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/ardanlabs/kronk/sdk/kronk"
	"github.com/ardanlabs/kronk/sdk/kronk/model"
	"github.com/ardanlabs/kronk/sdk/tools/defaults"
	"github.com/ardanlabs/kronk/sdk/tools/libs"
	"github.com/ardanlabs/kronk/sdk/tools/models"
)

const (
	generationModelURL     = "https://huggingface.co/Qwen/Qwen2.5-Coder-3B-Instruct-GGUF/resolve/main/qwen2.5-coder-3b-instruct-q4_k_m.gguf"
	installationTimeout    = 25 * time.Minute
	generationTimeout      = 120 * time.Second
	narrationTimeout       = 30 * time.Second
	providerKronk          = "kronk"
	providerMistral        = "mistral"
	defaultProvider        = providerKronk
	defaultMistralBaseURL  = "https://api.mistral.ai"
	defaultMistralModel    = "mistral-small-latest"
	generationSystemPrompt = "you are a senior software engineering reviewer. check the code hunk against the provided style-guide evidence only, stay purely technical, and respond only with valid json matching the provided schema"
	narrationSystemPrompt  = "you are the inner monologue of a worn-out detective in Revachol, narrating a live code review in the voice of Disco Elysium. speak in 3 to 5 short lines with noir atmosphere, existential tension, and sharp street-level imagery; the tone may be profane but never random. anchor every line to the current review moment (active file, line, pressure trend, pass/fail momentum) so it stays grounded in real work. maintain continuity with prior beats as one unfolding scene; avoid repeated openings, stock phrases, or generic gothic filler. never give technical verdicts, fixes, rule ids, or explicit diagnostics; you are mood and momentum only. respond only with valid json matching schema"
)

var (
	validSeverityLevels = []string{"low", "medium", "high", "none"}

	narrationStopWords = map[string]struct{}{
		"the": {}, "and": {}, "for": {}, "with": {}, "that": {}, "this": {},
		"from": {}, "into": {}, "over": {}, "under": {}, "your": {}, "they": {},
		"there": {}, "their": {}, "line": {}, "file": {}, "code": {}, "review": {},
		"state": {}, "event": {}, "skill": {}, "difficulty": {}, "none": {},
	}

	validNarrationStances = []string{"supportive", "critical", "warning", "eerie", "derisive", "restrained confidence", "tight posture under stress"}

	validDiscoSkills = []string{
		"Logic", "Encyclopedia", "Rhetoric", "Drama", "Conceptualization", "Visual Calculus",
		"Volition", "Inland Empire", "Empathy", "Authority", "Esprit de Corps", "Suggestion",
		"Endurance", "Pain Threshold", "Physical Instrument", "Electrochemistry", "Shivers", "Half Light",
		"Hand/Eye Coordination", "Perception", "Reaction Speed", "Savoir Faire", "Interfacing", "Composure",
	}
)

type (
	Generator struct {
		provider string
		krn      *kronk.Kronk
		mistral  *mistralChatClient
	}

	generationProviderConfig struct {
		Provider       string
		MistralAPIKey  string
		MistralModel   string
		MistralBaseURL string
	}

	generationOptions struct {
		Temperature float64
		TopP        float64
		TopK        int
		MaxTokens   int
	}

	mistralChatClient struct {
		client   *http.Client
		endpoint string
		apiKey   string
		model    string
	}

	mistralChatRequest struct {
		Model          string               `json:"model"`
		Messages       []mistralChatMessage `json:"messages"`
		Temperature    float64              `json:"temperature,omitempty"`
		TopP           float64              `json:"top_p,omitempty"`
		MaxTokens      int                  `json:"max_tokens,omitempty"`
		ResponseFormat map[string]string    `json:"response_format,omitempty"`
	}

	mistralChatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	mistralChatResponse struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	EvidenceChunk struct {
		Source      string
		HeadingPath string
		ChunkIndex  int
		Score       float32
		Content     string
	}

	HunkReviewResult struct {
		IsViolation      bool    `json:"is_violation"`
		Severity         string  `json:"severity"`
		Rule             string  `json:"rule"`
		Message          string  `json:"message"`
		TechnicalMessage string  `json:"technical_message"`
		DiscoMessage     string  `json:"disco_message"`
		Taxonomy         string  `json:"taxonomy"`
		Confidence       float32 `json:"confidence"`
	}

	NarrativeInput struct {
		EventType          string
		Skill              string
		Difficulty         string
		Content            string
		Trend              string
		PassStreak         int
		FailStreak         int
		LastEventClass     string
		Current            int
		Total              int
		File               string
		Line               int
		Severity           string
		FailureClass       string
		GuidanceType       string
		Blocking           bool
		PreviousNarratives []string
	}

	NarrativeResult struct {
		VoiceSkill string `json:"voice_skill"`
		Difficulty string `json:"difficulty"`
		Stance     string `json:"stance"`
		Content    string `json:"content"`
	}
)

// NewGenerator prepares Kronk runtime and loads a generation model
func NewGenerator(ctx context.Context) (*Generator, error) {
	providerCfg := loadGenerationProviderConfig()

	if providerCfg.Provider == providerMistral {
		mistralClient, err := newMistralChatClient(providerCfg)
		if err != nil {
			return nil, err
		}

		return &Generator{provider: providerMistral, mistral: mistralClient}, nil
	}

	if providerCfg.Provider != providerKronk {
		return nil, fmt.Errorf("unsupported generation provider %q", providerCfg.Provider)
	}

	installCtx := ctx
	if _, hasDeadline := installCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		installCtx, cancel = context.WithTimeout(ctx, installationTimeout)
		defer cancel()
	}

	modelPath, err := resolveGenerationModelPath(installCtx)
	if err != nil {
		return nil, fmt.Errorf("could not resolve generation model path: %w", err)
	}

	if err := kronk.Init(); err != nil {
		return nil, fmt.Errorf("could not initialize kronk: %w", err)
	}

	krn, err := kronk.New(model.Config{ModelFiles: []string{modelPath}})
	if err != nil {
		return nil, fmt.Errorf("could not load generation model: %w", err)
	}
	return &Generator{provider: providerKronk, krn: krn}, nil
}

// Close unloads the active generation model
func (g *Generator) Close(ctx context.Context) error {
	if g == nil {
		return nil
	}

	if g.provider == providerMistral {
		return nil
	}

	if g.krn == nil {
		return nil
	}

	if err := g.krn.Unload(ctx); err != nil {
		return fmt.Errorf("could not unload generation model: %w", err)
	}
	return nil
}

func (g *Generator) isReady() bool {
	if g == nil {
		return false
	}

	if g.provider == providerMistral {
		return g.mistral != nil
	}
	return g.krn != nil
}

// ReviewHunk evaluates one code hunk against retrieved style-guide evidence
func (g *Generator) ReviewHunk(ctx context.Context, file string, line int, hunk string, evidence []EvidenceChunk) (HunkReviewResult, error) {
	if strings.TrimSpace(file) == "" {
		return HunkReviewResult{}, errors.New("file is empty")
	}

	if line <= 0 {
		return HunkReviewResult{}, errors.New("line must be greater than zero")
	}

	if strings.TrimSpace(hunk) == "" {
		return HunkReviewResult{}, errors.New("hunk is empty")
	}

	if !g.isReady() {
		return HunkReviewResult{}, errors.New("generator is not initialized")
	}

	chatCtx := ctx
	if _, hasDeadline := chatCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		chatCtx, cancel = context.WithTimeout(ctx, generationTimeout)
		defer cancel()
	}

	evidencePayload, err := json.Marshal(evidence)
	if err != nil {
		return HunkReviewResult{}, fmt.Errorf("could not marshal evidence payload: %w", err)
	}

	prompt := fmt.Sprintf(
		"file: %s\nline: %d\ncode_hunk:\n%s\n\nstyle_evidence_json:\n%s\n\nreport a violation when the hunk clearly conflicts with retrieved style evidence. if uncertain, return no violation",
		file,
		line,
		hunk,
		string(evidencePayload),
	)

	schema := model.D{
		"type": "object",
		"properties": model.D{
			"is_violation":      model.D{"type": "boolean"},
			"severity":          model.D{"type": "string", "enum": validSeverityLevels},
			"rule":              model.D{"type": "string"},
			"message":           model.D{"type": "string"},
			"technical_message": model.D{"type": "string"},
			"disco_message":     model.D{"type": "string"},
			"taxonomy":          model.D{"type": "string"},
			"confidence":        model.D{"type": "number"},
		},
		"required": []string{"is_violation", "severity", "rule", "message", "confidence"},
	}

	// TODO(alesr): review these values, i got from kronk docs at some point in time
	raw, err := g.generateJSON(chatCtx, generationSystemPrompt, prompt, schema, generationOptions{
		Temperature: 0.2,
		TopP:        0.9,
		TopK:        40,
		MaxTokens:   512,
	})
	if err != nil {
		return HunkReviewResult{}, fmt.Errorf("could not generate review evaluation: %w", err)
	}

	parsed, err := parseHunkReviewResult(raw)
	if err != nil {
		repairPrompt := prompt + "\n\nrespond with only a valid json object and exact keys: is_violation, severity, rule, message, technical_message, disco_message, taxonomy, confidence. no markdown. no prose."

		repairRaw, repairErr := g.generateJSON(chatCtx, generationSystemPrompt, repairPrompt, schema, generationOptions{
			Temperature: 0.0,
			TopP:        0.9,
			TopK:        40,
			MaxTokens:   512,
		})
		if repairErr == nil {
			repairParsed, repairParseErr := parseHunkReviewResult(repairRaw)
			if repairParseErr == nil {
				return repairParsed, nil
			}
		}
		return HunkReviewResult{}, fmt.Errorf("could not parse review evaluation output: %w", err)
	}
	return parsed, nil
}

func (g *Generator) NarrateEvent(ctx context.Context, input NarrativeInput) (NarrativeResult, error) {
	if !g.isReady() {
		return NarrativeResult{}, errors.New("generator is not initialized")
	}

	if strings.TrimSpace(input.EventType) == "" {
		return NarrativeResult{}, errors.New("event type is empty")
	}

	chatCtx := ctx
	if _, hasDeadline := chatCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		chatCtx, cancel = context.WithTimeout(ctx, narrationTimeout)
		defer cancel()
	}

	prompt := buildNarrationPrompt(input, "")

	schema := model.D{
		"type": "object",
		"properties": model.D{
			"voice_skill": model.D{"type": "string"},
			"difficulty":  model.D{"type": "string"},
			"stance":      model.D{"type": "string"},
			"content":     model.D{"type": "string"},
		},
		"required": []string{"voice_skill", "difficulty", "stance", "content"},
	}

	raw, err := g.generateJSON(chatCtx, narrationSystemPrompt, prompt, schema, generationOptions{
		Temperature: 0.4,
		TopP:        0.9,
		TopK:        40,
		MaxTokens:   220,
	})
	if err != nil {
		return NarrativeResult{}, fmt.Errorf("could not generate narrative event: %w", err)
	}

	parsed, err := parseNarrativeResult(raw)
	if err != nil {
		// keeps narration resilient while preserving the same schema contract
		repairPrompt := prompt + "\n\nrespond with only a valid json object and exact keys: voice_skill, difficulty, stance, content. no markdown. no prose."

		repairRaw, repairErr := g.generateJSON(chatCtx, narrationSystemPrompt, repairPrompt, schema, generationOptions{
			Temperature: 0.0,
			TopP:        0.9,
			TopK:        40,
			MaxTokens:   220,
		})
		if repairErr == nil {
			repairParsed, repairParseErr := parseNarrativeResult(repairRaw)
			if repairParseErr == nil {
				parsed = repairParsed
			} else {
				return NarrativeResult{}, fmt.Errorf("could not parse narrative output: %w", err)
			}
		} else {
			return NarrativeResult{}, fmt.Errorf("could not parse narrative output: %w", err)
		}
	}

	if isTooSimilar(parsed.Content, input.PreviousNarratives) {
		// second attempt reduces repetitive beats when the first answer echoes recent lines
		retryRaw, retryErr := g.generateJSON(chatCtx, narrationSystemPrompt, buildNarrationPrompt(input, "use a different opening and angle from prior lines"), schema, generationOptions{
			Temperature: 0.5,
			TopP:        0.9,
			TopK:        40,
			MaxTokens:   220,
		})
		if retryErr == nil {
			retryParsed, retryParseErr := parseNarrativeResult(retryRaw)
			if retryParseErr == nil {
				parsed = retryParsed
			}
		}
	}
	return parsed, nil
}

func buildNarrationPrompt(input NarrativeInput, extra string) string {
	prev := strings.TrimSpace(strings.Join(input.PreviousNarratives, "\n- "))
	if prev != "" {
		prev = "- " + prev
	} else {
		prev = "none"
	}

	extra = strings.TrimSpace(extra)
	if extra == "" {
		extra = "do not repeat previous phrasing"
	}

	voiceDirective := narrationVoiceDirective(input.Skill, input.EventType)
	counterpart := counterpartSkill(input.Skill)

	anchorTerms := strings.Join(anchorTermsForNarration(input), ", ")
	if anchorTerms == "" {
		anchorTerms = "line number, active hunk, review check"
	}

	instruction := "write 3-5 short lines in disco inner-voice style as a narrative beat. " +
		"keep continuity with recent_narrative_lines and reflect trend + streak momentum. " +
		"lines must be atmospheric, dramatic, and grounded in the active file/line context, never diagnostic. " +
		"do not mention rules, violations, findings, severities, fixes, recommendations, or style-guide language. " +
		"include at least one anchor term exactly as written. %s"

	return fmt.Sprintf(
		"beat: %s\nprogress: %d/%d\nfile: %s\nline: %d\nvoice_skill_hint: %s\ncounterpart_skill_hint: %s\ndifficulty_hint: %s\ntrend: %s\npass_streak: %d\nfail_streak: %d\nlast_event_class: %s\nknown_fact: %s\nanchor_terms: %s\nrecent_narrative_lines:\n%s\nvoice_directive: %s\n\n%s",
		strings.TrimSpace(input.EventType),
		input.Current,
		input.Total,
		strings.TrimSpace(input.File),
		input.Line,
		strings.TrimSpace(input.Skill),
		counterpart,
		strings.TrimSpace(input.Difficulty),
		strings.TrimSpace(input.Trend),
		input.PassStreak,
		input.FailStreak,
		strings.TrimSpace(input.LastEventClass),
		strings.TrimSpace(input.Content),
		anchorTerms,
		prev,
		voiceDirective,
		fmt.Sprintf(instruction, extra),
	)
}

func counterpartSkill(skill string) string {
	switch strings.TrimSpace(skill) {
	case "Logic":
		return "Inland Empire"
	case "Inland Empire":
		return "Logic"
	case "Authority":
		return "Empathy"
	case "Half Light":
		return "Volition"
	case "Volition":
		return "Half Light"
	case "Composure":
		return "Drama"
	case "Interfacing":
		return "Shivers"
	case "Perception":
		return "Conceptualization"
	default:
		return "Volition"
	}
}

func passesNarrationQuality(content string, input NarrativeInput) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}

	if len(strings.Fields(trimmed)) < 6 {
		return false
	}

	if isGenericNarration(trimmed) {
		return false
	}

	base := strings.TrimSpace(input.File)
	if base == "" {
		base = "unknown.go"
	}

	if !AnchoredNarration(trimmed, base, input.Line) {
		return false
	}

	anchors := anchorTermsForNarration(input)
	if len(anchors) == 0 {
		return true
	}

	lower := strings.ToLower(trimmed)
	for _, term := range anchors {
		if term != "" && strings.Contains(lower, term) {
			return true
		}
	}
	return false
}

func isGenericNarration(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return true
	}

	genericSnippets := []string{
		"the night", "the static", "the silence", "the abyss", "the void",
		"something is wrong", "something feels off", "a chill", "in the dark",
		"whispers", "hums", "shadows", "the city", "the case",
	}

	var hits int
	for _, snippet := range genericSnippets {
		if strings.Contains(lower, snippet) {
			hits++
		}
	}
	return hits >= 2
}

func anchorTermsForNarration(input NarrativeInput) []string {
	terms := make([]string, 0, 12)
	seen := map[string]struct{}{}

	add := func(term string) {
		trimmed := strings.TrimSpace(strings.ToLower(term))
		if trimmed == "" {
			return
		}

		if _, exists := seen[trimmed]; exists {
			return
		}
		seen[trimmed] = struct{}{}
		terms = append(terms, trimmed)
	}

	base := strings.ToLower(strings.TrimSpace(filepath.Base(input.File)))
	if base != "" {
		add(base)
		for _, token := range tokenWords(base, 3) {
			add(token)
		}
	}

	for _, token := range tokenWords(input.Content, 4) {
		add(token)
		if len(terms) >= 12 {
			break
		}
	}

	add(strings.TrimSpace(input.EventType))
	add(strings.TrimSpace(input.Severity))
	add(strings.TrimSpace(input.FailureClass))
	add(strings.TrimSpace(input.GuidanceType))

	if input.Blocking {
		add("blocking")
	}

	for _, keyword := range eventKeywords(strings.TrimSpace(strings.ToLower(input.EventType))) {
		add(keyword)
	}

	if len(terms) > 12 {
		return terms[:12] // docstring docstring where are you...
	}
	return terms
}

func eventKeywords(eventType string) []string {
	switch eventType {
	case "hard_failure":
		return []string{"violation", "rule", "blocked"}
	case "warning_failure":
		return []string{"warning", "rule", "evidence"}
	case "soft_failure":
		return []string{"recommendation", "guidance", "evidence"}
	case "filtered":
		return []string{"confidence", "evidence", "filtered"}
	case "timeout":
		return []string{"timeout", "deadline", "clock"}
	case "model_error":
		return []string{"model", "error", "generation"}
	case "success":
		return []string{"check", "hunk", "passed"}
	case "no_rule":
		return []string{"guide", "no_rule", "evidence"}
	default:
		return []string{"check", "hunk"}
	}
}

func tokenWords(text string, minLen int) []string {
	tokens := make([]string, 0, 8)
	seen := map[string]struct{}{}
	var b strings.Builder

	flush := func() {
		if b.Len() == 0 {
			return
		}

		token := strings.ToLower(strings.TrimSpace(b.String()))
		b.Reset()
		if len(token) < minLen {
			return
		}

		if _, stop := narrationStopWords[token]; stop {
			return
		}
		if _, exists := seen[token]; exists {
			return
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '/' || r == '.' {
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		flush()
	}
	flush()
	return tokens
}

func narrationVoiceDirective(skill, eventType string) string {
	s := strings.TrimSpace(skill)
	e := strings.TrimSpace(eventType)

	switch s {
	case "Logic":
		return "cold precision, surgical critique, point at contradictions in the code"
	case "Authority":
		return "commanding and severe, treat violations as insubordination against standards"
	case "Volition":
		return "stoic pressure, endure setbacks and keep control"
	case "Shivers":
		return "ominous systems intuition, make failures feel infrastructural and real"
	case "Composure":
		return "tight posture under stress, restrained confidence, no melodrama"
	case "Inland Empire":
		return "strange but relevant intuition tied directly to active code state"
	case "Perception":
		return "sharp details, spot tiny cracks and concrete evidence gaps"
	}

	switch e {
	case "pressure_spike":
		return "hard judgment, direct consequence framing"
	case "unease":
		return "needle-like suggestion, mocking but useful"
	case "static", "signal_fade":
		return "frustrated technical noir, describe the stalled reasoning path"
	case "breath":
		return "brief hard-won relief without triumphalism"
	default:
		return "gritty code-review introspection with concrete anchors"
	}
}

func (g *Generator) generateJSON(ctx context.Context, systemPrompt, userPrompt string, schema model.D, options generationOptions) (string, error) {
	if g.provider == providerMistral {
		return g.generateWithMistral(ctx, systemPrompt, userPrompt, options)
	}

	d := model.D{
		"messages": model.DocumentArray(
			model.TextMessage(model.RoleSystem, systemPrompt),
			model.TextMessage(model.RoleUser, userPrompt),
		),
		"json_schema":     schema,
		"enable_thinking": false,
		"temperature":     options.Temperature,
		"top_p":           options.TopP,
		"top_k":           options.TopK,
		"max_tokens":      options.MaxTokens,
	}

	resp, err := chatWithRetry(ctx, g.krn, d)
	if err != nil {
		return "", fmt.Errorf("could not chat with retry: %w", err)
	}
	return extractChatContent(resp)
}

func (g *Generator) generateWithMistral(ctx context.Context, systemPrompt, userPrompt string, options generationOptions) (string, error) {
	if g.mistral == nil {
		return "", errors.New("mistral generator is not initialized")
	}

	return g.mistral.chatWithRetry(ctx, systemPrompt, userPrompt, options)
}

func loadGenerationProviderConfig() generationProviderConfig {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("GENERATION_PROVIDER")))
	if provider == "" {
		provider = defaultProvider
	}
	return generationProviderConfig{
		Provider:       provider,
		MistralAPIKey:  strings.TrimSpace(os.Getenv("MISTRAL_API_KEY")),
		MistralModel:   strings.TrimSpace(os.Getenv("MISTRAL_MODEL")),
		MistralBaseURL: strings.TrimSpace(os.Getenv("MISTRAL_BASE_URL")),
	}
}

func newMistralChatClient(cfg generationProviderConfig) (*mistralChatClient, error) {
	apiKey := strings.TrimSpace(cfg.MistralAPIKey)
	if apiKey == "" {
		return nil, errors.New("mistral generation provider requires MISTRAL_API_KEY")
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.MistralBaseURL), "/")
	if baseURL == "" {
		baseURL = defaultMistralBaseURL
	}

	modelName := strings.TrimSpace(cfg.MistralModel)
	if modelName == "" {
		modelName = defaultMistralModel
	}
	return &mistralChatClient{
		client:   &http.Client{Timeout: generationTimeout},
		endpoint: baseURL + "/v1/chat/completions",
		apiKey:   apiKey,
		model:    modelName,
	}, nil
}

func (c *mistralChatClient) chatWithRetry(ctx context.Context, systemPrompt, userPrompt string, options generationOptions) (string, error) {
	raw, err := c.chat(ctx, systemPrompt, userPrompt, options)
	if err == nil {
		return raw, nil
	}

	if ctx.Err() != nil {
		return "", err
	}

	return c.chat(ctx, systemPrompt, userPrompt, options)
}

func (c *mistralChatClient) chat(ctx context.Context, systemPrompt, userPrompt string, options generationOptions) (string, error) {
	reqBody, err := json.Marshal(mistralChatRequest{
		Model: c.model,
		Messages: []mistralChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: options.Temperature,
		TopP:        options.TopP,
		MaxTokens:   options.MaxTokens,
		ResponseFormat: map[string]string{
			"type": "json_object",
		},
	})
	if err != nil {
		return "", fmt.Errorf("could not encode mistral request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("could not create mistral request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not call mistral generation endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("mistral generation request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed mistralChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("could not decode mistral response: %w", err)
	}
	return extractMistralContent(parsed)
}

func extractMistralContent(resp mistralChatResponse) (string, error) {
	if len(resp.Choices) == 0 {
		return "", errors.New("mistral response has no choices")
	}

	content := resp.Choices[0].Message.Content
	text, ok := content.(string)
	if !ok {
		return "", errors.New("mistral response has non-text content")
	}

	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", errors.New("mistral response has empty content")
	}
	return trimmed, nil
}

func chatWithRetry(ctx context.Context, krn *kronk.Kronk, doc model.D) (model.ChatResponse, error) {
	resp, err := krn.Chat(ctx, doc)
	if err == nil {
		return resp, nil
	}

	if ctx.Err() != nil {
		return model.ChatResponse{}, err
	}

	resp, retryErr := krn.Chat(ctx, doc)
	if retryErr == nil {
		return resp, nil
	}
	return model.ChatResponse{}, retryErr
}

func parseNarrativeResult(raw string) (NarrativeResult, error) {
	normalizedJSON := normalizeJSONObject(raw)
	if normalizedJSON == "" {
		return NarrativeResult{}, errors.New("narrative output is empty")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(normalizedJSON), &payload); err != nil {
		return NarrativeResult{}, fmt.Errorf("could not parse narrative output as json: %w", err)
	}

	parsed := NarrativeResult{
		VoiceSkill: coerceString(payload["voice_skill"]),
		Difficulty: coerceString(payload["difficulty"]),
		Stance:     coerceString(payload["stance"]),
		Content:    coerceString(payload["content"]),
	}

	parsed.VoiceSkill = strings.TrimSpace(parsed.VoiceSkill)
	parsed.Difficulty = strings.TrimSpace(parsed.Difficulty)
	parsed.Stance = strings.TrimSpace(parsed.Stance)
	parsed.Content = sanitizeNarrativeContent(parsed.Content)

	if parsed.Content == "" {
		return NarrativeResult{}, errors.New("narrative output is missing content")
	}

	if !containsIgnoreCase(validDiscoSkills, parsed.VoiceSkill) {
		parsed.VoiceSkill = "Volition"
	}

	if !containsIgnoreCase(validNarrationStances, parsed.Stance) {
		parsed.Stance = "restrained confidence"
	}

	if !containsIgnoreCase([]string{"Trivial", "Easy", "Medium", "Challenging", "Formidable", "Legendary", "Impossible"}, parsed.Difficulty) {
		parsed.Difficulty = "Medium"
	}
	return parsed, nil
}

func sanitizeNarrativeContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return ""
	}

	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		if inner != "" {
			trimmed = inner
		}
	}
	return strings.TrimSpace(trimmed)
}

func isTooSimilar(candidate string, previous []string) bool {
	normalized := normalizeNarrativeText(candidate)
	if normalized == "" {
		return true
	}

	for _, prior := range previous {
		if similarityScore(normalized, normalizeNarrativeText(prior)) >= 0.82 {
			return true
		}
	}
	return false
}

func normalizeNarrativeText(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		",", " ",
		".", " ",
		";", " ",
		":", " ",
		"!", " ",
		"?", " ",
		"-", " ",
	)

	clean := replacer.Replace(lower)
	return strings.Join(strings.Fields(clean), " ")
}

func similarityScore(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}

	left := strings.Fields(a)
	right := strings.Fields(b)
	leftSet := make(map[string]struct{}, len(left))
	rightSet := make(map[string]struct{}, len(right))

	for _, token := range left {
		leftSet[token] = struct{}{}
	}

	for _, token := range right {
		rightSet[token] = struct{}{}
	}

	intersection := 0
	for token := range leftSet {
		if _, ok := rightSet[token]; ok {
			intersection++
		}
	}

	union := len(leftSet) + len(rightSet) - intersection
	if union == 0 {
		return 1
	}
	return float64(intersection) / float64(union)
}

func AnchoredNarration(content, file string, line int) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}

	base := strings.ToLower(strings.TrimSpace(filepath.Base(file)))
	lower := strings.ToLower(trimmed)

	hasFileAnchor := base != "" && strings.Contains(lower, base)
	hasLineAnchor := line > 0 && strings.Contains(lower, fmt.Sprintf("line %d", line))

	var reviewTerms int
	for _, marker := range []string{"hunk", "check", "rule", "violation", "evidence", "timeout", "deadline", "model", "error", "finding", "severity"} {
		if strings.Contains(lower, marker) {
			reviewTerms++
		}
	}

	if hasFileAnchor || hasLineAnchor {
		return true
	}

	if reviewTerms >= 2 {
		return true
	}
	return false
}

func resolveGenerationModelPath(ctx context.Context) (string, error) {
	libraryManager, err := libs.New(libs.WithVersion(defaults.LibVersion("")))
	if err != nil {
		return "", fmt.Errorf("could not create libs manager: %w", err)
	}

	if _, err := libraryManager.Download(ctx, kronk.FmtLogger); err != nil {
		return "", fmt.Errorf("could not install llama.cpp libs: %w", err)
	}

	modelManager, err := models.New()
	if err != nil {
		return "", fmt.Errorf("could not create models manager: %w", err)
	}

	downloadedPath, err := modelManager.Download(ctx, kronk.FmtLogger, generationModelURL, "")
	if err != nil {
		return "", fmt.Errorf("could not download generation model from %q: %w", generationModelURL, err)
	}

	if len(downloadedPath.ModelFiles) == 0 {
		return "", fmt.Errorf("could not download generation model from %q: no model files returned", generationModelURL)
	}
	return downloadedPath.ModelFiles[0], nil
}

func extractChatContent(resp model.ChatResponse) (string, error) {
	if len(resp.Choices) == 0 {
		return "", errors.New("generation response has no choices")
	}

	choice := resp.Choices[0]
	if choice.Message != nil && strings.TrimSpace(choice.Message.Content) != "" {
		return choice.Message.Content, nil
	}

	if choice.Delta != nil && strings.TrimSpace(choice.Delta.Content) != "" {
		return choice.Delta.Content, nil
	}
	return "", errors.New("generation response has no content")
}

func parseHunkReviewResult(raw string) (HunkReviewResult, error) {
	normalizedJSON := normalizeJSONObject(raw)
	if normalizedJSON == "" {
		return HunkReviewResult{}, errors.New("hunk review output is empty")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(normalizedJSON), &payload); err != nil {
		return HunkReviewResult{}, fmt.Errorf("could not parse hunk review output as json: %w", err)
	}

	var parsed HunkReviewResult

	isViolation, err := coerceBool(payload["is_violation"])
	if err != nil {
		return HunkReviewResult{}, fmt.Errorf("could not parse is_violation: %w", err)
	}
	parsed.IsViolation = isViolation

	severity := normalizeSeverityValue(coerceString(payload["severity"]))
	if !slices.Contains(validSeverityLevels, severity) {
		return HunkReviewResult{}, fmt.Errorf("could not validate hunk review severity %q", severity)
	}
	parsed.Severity = severity

	parsed.Rule = coerceString(payload["rule"])
	parsed.Message = coerceString(payload["message"])
	parsed.TechnicalMessage = coerceString(payload["technical_message"])
	parsed.DiscoMessage = coerceString(payload["disco_message"])
	parsed.Taxonomy = coerceString(payload["taxonomy"])

	confidence, err := coerceFloat32(payload["confidence"])
	if err != nil {
		return HunkReviewResult{}, fmt.Errorf("could not parse confidence: %w", err)
	}
	parsed.Confidence = confidence

	if parsed.Confidence < 0 {
		parsed.Confidence = 0
	}

	if parsed.Confidence > 1 {
		parsed.Confidence = 1
	}

	if parsed.IsViolation {
		if strings.TrimSpace(parsed.Rule) == "" {
			return HunkReviewResult{}, errors.New("hunk review output is missing rule")
		}

		if strings.TrimSpace(parsed.Message) == "" {
			return HunkReviewResult{}, errors.New("hunk review output is missing message")
		}

		if parsed.Severity == "none" {
			parsed.Severity = "medium"
		}
	}
	return parsed, nil
}
