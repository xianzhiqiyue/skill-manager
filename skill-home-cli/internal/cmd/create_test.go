package cmd

import (
	"strings"
	"testing"
)

func TestRenderTemplateIncludesCategoryAndTags(t *testing.T) {
	t.Parallel()

	content, err := renderTemplate(`---
name: {{.Name}}
description: {{.Description}}
category: {{.Category}}
{{if .Tags}}tags:{{range .Tags}}
  - {{.}}{{end}}{{end}}
license: {{.License}}
{{.IDEConfig}}
---`, &SkillAnswers{
		Name:        "deploy-buddy",
		Description: "部署助手",
		Category:    "ops",
		Tags:        []string{"deployment", "ci-cd"},
		License:     "MIT",
		Platforms:   []string{"codex"},
	})
	if err != nil {
		t.Fatalf("renderTemplate returned error: %v", err)
	}

	if !strings.Contains(content, "category: ops") {
		t.Fatalf("expected category line, got:\n%s", content)
	}
	if !strings.Contains(content, "- deployment") || !strings.Contains(content, "- ci-cd") {
		t.Fatalf("expected tags block, got:\n%s", content)
	}
}
