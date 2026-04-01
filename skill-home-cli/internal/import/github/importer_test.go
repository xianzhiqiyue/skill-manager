package github

import (
	"strings"
	"testing"
)

func TestConvertReadmeToSkillUsesClassificationDefaults(t *testing.T) {
	t.Parallel()

	importer := &GitHubImporter{repo: "demo-skill"}
	content := importer.convertReadmeToSkill("# Demo")

	if !strings.Contains(content, "category: productivity") {
		t.Fatalf("expected default category, got:\n%s", content)
	}
	if !strings.Contains(content, "tags:\n  - workflow") {
		t.Fatalf("expected default workflow tag, got:\n%s", content)
	}
}
