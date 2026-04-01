package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePreservesOpenClawCredentialsMetadata(t *testing.T) {
	dir := t.TempDir()
	content := `---
name: github
version: 1.0.0
description: GitHub skill
metadata:
  openclaw:
    credentials:
      - id: openai_api_key
        env: OPENAI_API_KEY
        label: OpenAI API Key
        description: Used to access OpenAI
        secret: true
        required: true
        input: password
        help_url: https://platform.openai.com/api-keys
        group: llm_provider
---
body
`
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md failed: %v", err)
	}

	s, err := Parse(dir)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	metadata, ok := s.Manifest.Metadata["openclaw"].(map[string]interface{})
	if !ok {
		t.Fatalf("openclaw metadata missing: %#v", s.Manifest.Metadata)
	}
	credentials, ok := metadata["credentials"].([]interface{})
	if !ok || len(credentials) != 1 {
		t.Fatalf("credentials = %#v, want 1 item", metadata["credentials"])
	}
	credential, ok := credentials[0].(map[string]interface{})
	if !ok {
		t.Fatalf("credential = %#v", credentials[0])
	}
	if got := credential["env"]; got != "OPENAI_API_KEY" {
		t.Fatalf("credential env = %#v, want OPENAI_API_KEY", got)
	}
}

func TestParseDerivesRequiresFromOpenClawCredentials(t *testing.T) {
	dir := t.TempDir()
	content := `---
name: github
version: 1.0.0
description: GitHub skill
category: ops
metadata:
  openclaw:
    credentials:
      - id: openai_api_key
        env: OPENAI_API_KEY
      - id: anthropic_api_key
        env: ANTHROPIC_API_KEY
---
body
`
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md failed: %v", err)
	}

	s, err := Parse(dir)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if len(s.Manifest.Requires) != 2 {
		t.Fatalf("requires = %#v, want 2 items", s.Manifest.Requires)
	}
	if s.Manifest.Requires[0] != "OPENAI_API_KEY" || s.Manifest.Requires[1] != "ANTHROPIC_API_KEY" {
		t.Fatalf("requires = %#v, want derived envs", s.Manifest.Requires)
	}
	if s.Manifest.Category != "ops" {
		t.Fatalf("category = %q, want ops", s.Manifest.Category)
	}
}
