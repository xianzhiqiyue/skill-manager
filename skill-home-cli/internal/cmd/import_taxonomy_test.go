package cmd

import (
	"strings"
	"testing"
)

func TestCodexImporterGeneratedSkillUsesClassificationDefaults(t *testing.T) {
	t.Parallel()

	importer := &CodexImporter{skillName: "demo-skill"}
	content := importer.generateSkillMD("system prompt")

	if !strings.Contains(content, "category: productivity") {
		t.Fatalf("expected default category, got:\n%s", content)
	}
	if !strings.Contains(content, "tags:\n  - workflow") {
		t.Fatalf("expected default workflow tag, got:\n%s", content)
	}
}

func TestCursorImporterGeneratedSkillUsesClassificationDefaults(t *testing.T) {
	t.Parallel()

	importer := &CursorImporter{skillName: "demo-skill"}
	content := importer.convertMdcToSkill(`---
title: Demo Skill
description: Demo description
---

body`)

	if !strings.Contains(content, "category: productivity") {
		t.Fatalf("expected default category, got:\n%s", content)
	}
	if !strings.Contains(content, "tags:\n  - workflow") {
		t.Fatalf("expected default workflow tag, got:\n%s", content)
	}
}
