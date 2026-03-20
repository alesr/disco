package review

const (
	// enum EventType
	EventTypeProgress  = "progress"
	EventTypeNarrative = "narrative"
	EventTypeResult    = "result"
	EventTypeError     = "error"

	// enum NarrativeEvent

	NarrativeEventNoRule         = "no_rule"
	NarrativeEventTimeout        = "timeout"
	NarrativeEventModelError     = "model_error"
	NarrativeEventSuccess        = "success"
	NarrativeEventFiltered       = "filtered"
	NarrativeEventSoftFailure    = "soft_failure"
	NarrativeEventHardFailure    = "hard_failure"
	NarrativeEventWarningFailure = "warning_failure"

	// enum FindingKind

	FindingKindViolation      = "violation"
	FindingKindRecommendation = "recommendation"

	// enum GuidanceType

	GuidanceTypeRule           = "rule"
	GuidanceTypeRecommendation = "recommendation"

	// enum Enforcement

	EnforcementBlock = "block"
	EnforcementWarn  = "warn"
	EnforcementInfo  = "info"

	// enum FailureClass

	FailureClassHard = "hard"
	FailureClassSoft = "soft"
)

var ValidDiscoDifficulties = []string{
	"Trivial",
	"Easy",
	"Medium",
	"Challenging",
	"Formidable",
	"Legendary",
	"Impossible",
}

type (
	DiffHunk struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Content string `json:"content"`
	}

	Citation struct {
		Source      string  `json:"source"`
		HeadingPath string  `json:"heading_path"`
		ChunkIndex  int     `json:"chunk_index"`
		Score       float32 `json:"score"`
	}

	Finding struct {
		Kind             string   `json:"kind"`
		GuidanceType     string   `json:"guidance_type"`
		Enforcement      string   `json:"enforcement"`
		SkillPrimary     string   `json:"skill_primary"`
		DifficultyBase   string   `json:"difficulty_base"`
		DifficultyMin    string   `json:"difficulty_min,omitempty"`
		DifficultyMax    string   `json:"difficulty_max,omitempty"`
		FailureClass     string   `json:"failure_class"`
		Blocking         bool     `json:"blocking"`
		File             string   `json:"file"`
		Line             int      `json:"line"`
		Severity         string   `json:"severity"`
		Rule             string   `json:"rule"`
		Message          string   `json:"message"`
		TechnicalMessage string   `json:"technical_message"`
		DiscoMessage     string   `json:"disco_message"`
		Taxonomy         string   `json:"taxonomy"`
		Confidence       float32  `json:"confidence"`
		EvidenceScore    float32  `json:"evidence_score"`
		Citation         Citation `json:"citation"`
		Hunk             string   `json:"hunk"`
		GoodExample      string   `json:"good_example,omitempty"`
	}

	SkillCheck struct {
		Skill            string   `json:"skill"`
		Category         string   `json:"category"`
		Difficulty       string   `json:"difficulty"`
		Success          bool     `json:"success"`
		Content          string   `json:"content"`
		TechnicalMessage string   `json:"technical_message,omitempty"`
		Rule             string   `json:"rule"`
		Severity         string   `json:"severity"`
		GuidanceType     string   `json:"guidance_type"`
		Enforcement      string   `json:"enforcement"`
		SkillPrimary     string   `json:"skill_primary"`
		DifficultyBase   string   `json:"difficulty_base"`
		DifficultyMin    string   `json:"difficulty_min,omitempty"`
		DifficultyMax    string   `json:"difficulty_max,omitempty"`
		FailureClass     string   `json:"failure_class"`
		Blocking         bool     `json:"blocking"`
		Citation         Citation `json:"citation"`
		File             string   `json:"file"`
		Line             int      `json:"line"`
		GoodExample      string   `json:"good_example,omitempty"`
	}

	ReviewSummary struct {
		FilesScanned               int `json:"files_scanned"`
		HunksScanned               int `json:"hunks_scanned"`
		HunksEvaluated             int `json:"hunks_evaluated"`
		Violations                 int `json:"violations"`
		Recommendations            int `json:"recommendations"`
		SoftFailures               int `json:"soft_failures"`
		BlockingFindings           int `json:"blocking_findings"`
		NoApplicableRule           int `json:"no_applicable_rule"`
		HunksFilteredLowConfidence int `json:"hunks_filtered_low_confidence"`
		HunksSkippedTimeout        int `json:"hunks_skipped_timeout"`
		HunksModelError            int `json:"hunks_model_error"`
	}

	ReviewResult struct {
		Summary  ReviewSummary `json:"summary"`
		Findings []Finding     `json:"findings"`
		Checks   []SkillCheck  `json:"checks"`
	}

	ReviewEvent struct {
		Type         string        `json:"type"`
		Current      int           `json:"current,omitempty"`
		Total        int           `json:"total,omitempty"`
		File         string        `json:"file,omitempty"`
		Line         int           `json:"line,omitempty"`
		Phase        string        `json:"phase,omitempty"`
		Message      string        `json:"message,omitempty"`
		Severity     string        `json:"severity,omitempty"`
		FailureClass string        `json:"failure_class,omitempty"`
		GuidanceType string        `json:"guidance_type,omitempty"`
		Blocking     bool          `json:"blocking,omitempty"`
		Stance       string        `json:"stance,omitempty"`
		Result       *ReviewResult `json:"result,omitempty"`
		Error        string        `json:"error,omitempty"`
		EventType    string        `json:"event_type,omitempty"`
		Skill        string        `json:"skill,omitempty"`
		Difficulty   string        `json:"difficulty,omitempty"`
		Content      string        `json:"content,omitempty"`
		NonBlocking  bool          `json:"non_blocking,omitempty"`
	}

	EvaluationInput struct {
		File     string
		Line     int
		Hunk     string
		Evidence []EvidenceChunk
	}

	EvidenceChunk struct {
		Source      string
		HeadingPath string
		ChunkIndex  int
		Score       float32
		Content     string
	}

	EvaluationResult struct {
		IsViolation      bool
		Severity         string
		Rule             string
		Message          string
		TechnicalMessage string
		DiscoMessage     string
		Taxonomy         string
		Confidence       float32
	}
)
