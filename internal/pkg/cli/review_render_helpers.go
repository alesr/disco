package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alesr/disco/internal/review"
)

var skillANSIColors = map[string]string{
	"INTELLECT": rgbANSI(77, 166, 179),
	"PSYCHE":    rgbANSI(98, 74, 180),
	"PHYSIQUE":  rgbANSI(174, 60, 92),
	"MOTORICS":  rgbANSI(195, 162, 43),
}

// UI syntax highlighting of the good example
var goKeywordPattern = regexp.MustCompile(`\b(break|case|chan|const|continue|default|defer|else|fallthrough|for|func|go|if|import|interface|map|package|range|return|select|struct|switch|type|var)\b`)

func printCompactCheck(output io.Writer, check review.SkillCheck) {
	result := "advisory"
	if check.FailureClass == review.FailureClassHard {
		result = "violation"
	}

	technical := strings.TrimSpace(check.TechnicalMessage)
	if technical == "" {
		technical = strings.TrimSpace(check.Content)
	}

	source := filepath.Base(strings.TrimSpace(check.Citation.Source))
	if source == "" {
		source = "style-guide"
	}

	fmt.Fprintf(output, "- %s %s %s:%d\n", renderSkillLabel(check), renderCheckMeta(check.Difficulty, result, check.Severity), check.File, check.Line)
	for _, line := range wrapText(technical, interactiveRenderWide-2) {
		fmt.Fprintf(output, "  %s\n", line)
	}
	fmt.Fprintln(output)
	fmt.Fprintf(output, "  %s%s | %s#%d%s\n", ansiDim, check.Rule, source, check.Citation.ChunkIndex, ansiReset)

	goodExample := strings.TrimSpace(check.GoodExample)
	if goodExample != "" {
		fmt.Fprintln(output)
		fmt.Fprintf(output, "  %sgood example:%s\n", ansiDim, ansiReset)
		printGoodExample(output, goodExample)
	}
}

func printGoodExample(output io.Writer, example string) {
	lines := strings.Split(strings.TrimSpace(example), "\n")
	if len(lines) == 0 {
		return
	}

	isGoFence := strings.HasPrefix(strings.TrimSpace(lines[0]), "```go")
	// syntax color is limited to fenced go blocks so plain examples stay predictable
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if isGoFence {
				fmt.Fprintf(output, "    %s%s%s\n", ansiDim, line, ansiReset)
			} else {
				fmt.Fprintf(output, "    %s\n", line)
			}
			continue
		}

		rendered := line
		if isGoFence {
			rendered = highlightGoLine(line)
		}
		fmt.Fprintf(output, "    %s\n", rendered)
	}
}

func printContinuePrompt(output io.Writer) {
	bg := "\x1b[48;2;143;37;16m"
	fg := "\x1b[38;2;255;255;255m"
	fmt.Fprintf(output, "%s%s  CONTINUE >  %s\n", bg, fg+ansiBold, ansiReset)
}

func wrapText(text string, width int) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}

	if width < 20 {
		// min width avoids pathological wrapping when terminal detection is unavailable
		width = 20
	}

	paragraphs := strings.Split(trimmed, "\n")
	wrapped := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		line := strings.TrimSpace(paragraph)
		if line == "" {
			wrapped = append(wrapped, "")
			continue
		}

		if len([]rune(line)) <= width {
			wrapped = append(wrapped, line)
			continue
		}

		words := strings.Fields(line)
		if len(words) == 0 {
			continue
		}

		current := words[0]
		for _, word := range words[1:] {
			candidate := current + " " + word
			if len([]rune(candidate)) <= width {
				current = candidate
				continue
			}

			wrapped = append(wrapped, current)
			current = word
		}
		wrapped = append(wrapped, current)
	}

	if len(wrapped) > 0 {
		trimmedLines := make([]string, 0, len(wrapped))
		for _, line := range wrapped {
			if strings.TrimSpace(line) == "" {
				continue
			}
			trimmedLines = append(trimmedLines, line)
		}
		wrapped = trimmedLines
	}
	return wrapped
}

func highlightGoLine(line string) string {
	keywordColor := rgbANSI(113, 194, 255) + ansiBold
	codeColor := rgbANSI(203, 214, 255)

	highlighted := goKeywordPattern.ReplaceAllStringFunc(line, func(token string) string {
		return keywordColor + token + ansiReset + codeColor
	})
	return codeColor + highlighted + ansiReset
}

func renderSkillLabel(check review.SkillCheck) string {
	skill := strings.ToUpper(strings.TrimSpace(check.Skill))
	if skill == "" {
		skill = "LOGIC"
	}

	color, ok := skillANSIColors[strings.TrimSpace(check.Category)]
	if !ok {
		return skill
	}
	return color + ansiBold + skill + ansiReset
}

func renderCheckMeta(difficulty, result, severity string) string {
	difficulty = strings.TrimSpace(difficulty)
	result = strings.TrimSpace(result)
	severity = strings.TrimSpace(severity)
	metaColor := checkMetaANSIColor()
	sep := ansiDim + " | " + ansiReset

	return ansiDim + "[" + ansiReset +
		metaColor + ansiBold + difficulty + ansiReset +
		sep +
		metaColor + ansiBold + result + ansiReset +
		sep +
		metaColor + ansiBold + severity + ansiReset +
		ansiDim + "]" + ansiReset
}

func checkMetaANSIColor() string { return rgbANSI(176, 186, 201) }

func rgbANSI(r, g, b int) string { return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b) }
