package rag

import (
	"fmt"
	"os"
)

func readStyleGuide(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("could not read style guide %q: %w", path, err)
	}
	return string(content), nil
}
