package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInitCreatesCategoryAndTagsSkeleton(t *testing.T) {
	t.Parallel()

	outputDir := t.TempDir()
	if err := runInit("github-helper", &initOptions{outputDir: outputDir}); err != nil {
		t.Fatalf("runInit returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "github-helper", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	text := string(content)
	if !strings.Contains(text, "category: \"\"") {
		t.Fatalf("expected category skeleton, got:\n%s", text)
	}
	if !strings.Contains(text, "tags: []") {
		t.Fatalf("expected tags skeleton, got:\n%s", text)
	}
}
