package review

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var hunkHeaderPattern = regexp.MustCompile(`^@@\s+-\d+(?:,\d+)?\s+\+(\d+)(?:,\d+)?\s+@@`)

func ParseUnifiedDiff(diff string) ([]DiffHunk, error) {
	if strings.TrimSpace(diff) == "" {
		return nil, errors.New("diff is empty")
	}

	lines := strings.Split(diff, "\n")
	hunks := make([]DiffHunk, 0, 8)

	var (
		currentFile  string
		currentStart int
		currentLines []string
		inHunk       bool
	)

	flushHunk := func() {
		if !inHunk || currentFile == "" || len(currentLines) == 0 {
			currentLines = nil
			return
		}

		hunks = append(hunks, DiffHunk{
			File:    currentFile,
			Line:    currentStart,
			Content: strings.Join(currentLines, "\n"),
		})

		currentLines = nil
	}

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++ "):
			flushHunk()
			inHunk = false
			filePath := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
			currentFile = normalizeDiffPath(filePath)

		case strings.HasPrefix(line, "@@ "):
			flushHunk()
			start, err := parseHunkStart(line)
			if err != nil {
				return nil, err
			}
			currentStart = start
			inHunk = true

		case inHunk:
			if strings.HasPrefix(line, "diff --git ") || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
				flushHunk()
				inHunk = false
				continue
			}
			currentLines = append(currentLines, line)
		}
	}

	flushHunk()
	return hunks, nil
}

func parseHunkStart(header string) (int, error) {
	matches := hunkHeaderPattern.FindStringSubmatch(header)
	if len(matches) != 2 {
		return 0, fmt.Errorf("could not parse hunk header %q", header)
	}

	start, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("could not parse hunk start from %q: %w", header, err)
	}

	if start < 0 {
		return 0, fmt.Errorf("could not parse non-negative hunk start from %q", header)
	}
	return start, nil
}

func normalizeDiffPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if after, ok := strings.CutPrefix(trimmed, "b/"); ok {
		return after
	}
	return trimmed
}
