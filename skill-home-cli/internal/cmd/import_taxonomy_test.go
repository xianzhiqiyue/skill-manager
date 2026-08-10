package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexImporterGeneratedSkillUsesClassificationDefaults(t *testing.T) {
	t.Parallel()

	importer := &CodexImporter{skillName: "demo-skill"}
	content := importer.generateSkillMD("system prompt")

	if !strings.Contains(content, "category: 效率与协作") {
		t.Fatalf("expected default category, got:\n%s", content)
	}
	if !strings.Contains(content, "tags:\n  - workflow") {
		t.Fatalf("expected default workflow tag, got:\n%s", content)
	}
}

func TestXiguaImporterPreservesClassificationDefaults(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "demo-skill")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte(`---
name: demo-skill
version: 0.1.0
description: Demo description
category: 效率与协作
tags:
  - workflow
---

body`), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	importer := &XiguaImporter{skillName: "demo-skill"}
	skill, err := importer.ConvertToSkill(sourceDir)
	if err != nil {
		t.Fatalf("ConvertToSkill returned error: %v", err)
	}
	content := skill.Content

	if !strings.Contains(content, "category: 效率与协作") {
		t.Fatalf("expected default category, got:\n%s", content)
	}
	if !strings.Contains(content, "tags:\n  - workflow") {
		t.Fatalf("expected default workflow tag, got:\n%s", content)
	}
}
