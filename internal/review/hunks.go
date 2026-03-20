package review

import (
	"fmt"
	"strings"
)

func countReviewableHunks(hunks []DiffHunk) int {
	total := 0
	for _, hunk := range hunks {
		if isReviewableFile(hunk.File) {
			total++
		}
	}

	return total
}

func buildHunkQuery(hunk DiffHunk) string {
	return fmt.Sprintf("file: %s\nline: %d\ncode:\n%s", hunk.File, hunk.Line, hunk.Content)
}

func isReviewableFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(path)), ".go")
}
