package rag

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func resolveStyleGuidePath(styleGuideDir string) (string, error) {
	if styleGuideDir == "" {
		return "", errors.New("style guide directory is empty")
	}

	markdownFiles := make([]string, 0, 1)
	err := filepath.WalkDir(styleGuideDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(strings.ToLower(path), ".md") {
			markdownFiles = append(markdownFiles, path)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("could not scan style guide directory %q: %w", styleGuideDir, err)
	}

	sort.Strings(markdownFiles)

	switch len(markdownFiles) {
	case 0:
		return "", fmt.Errorf("could not find markdown style guides in %q", styleGuideDir)
	case 1:
		return markdownFiles[0], nil
	default:
		return "", fmt.Errorf(
			"could not resolve a single style guide in %q: directory must contain exactly one markdown file (first candidates: %s, %s)",
			styleGuideDir,
			markdownFiles[0],
			markdownFiles[1],
		)
	}
}
