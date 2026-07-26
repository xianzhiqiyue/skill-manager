package export

import (
	"strings"
	"testing"

	"github.com/skill-home/cli/internal/skill"
	"gopkg.in/yaml.v3"
)

func TestExportToClaudePreservesMetadata(t *testing.T) {
	engine := NewEngine("")
	s := &skill.Skill{
		Manifest: skill.Manifest{
			Name:        "github",
			Version:     "1.0.0",
			Description: "GitHub skill",
			Metadata: map[string]interface{}{
				"openclaw": map[string]interface{}{
					"credentials": []interface{}{
						map[string]interface{}{
							"id":    "openai_api_key",
							"env":   "OPENAI_API_KEY",
							"label": "OpenAI API Key",
						},
					},
				},
			},
		},
		Body: "body",
	}

	result, err := engine.Export(s, "claude")
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	frontmatter := extractFrontmatter(t, string(result.Files["SKILL.md"]))
	metadata, ok := frontmatter["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %#v", frontmatter)
	}
	openclaw, ok := metadata["openclaw"].(map[string]interface{})
	if !ok {
		t.Fatalf("openclaw metadata missing: %#v", metadata)
	}
	credentials, ok := openclaw["credentials"].([]interface{})
	if !ok || len(credentials) != 1 {
		t.Fatalf("credentials = %#v, want 1 item", openclaw["credentials"])
	}
	requires, ok := frontmatter["requires"].([]interface{})
	if !ok || len(requires) != 1 || requires[0] != "OPENAI_API_KEY" {
		t.Fatalf("requires = %#v, want derived OPENAI_API_KEY", frontmatter["requires"])
	}
}

func TestExportToXiguaPreservesMetadataAndWritesPackageManifest(t *testing.T) {
	engine := NewEngine("")
	s := &skill.Skill{
		Manifest: skill.Manifest{
			Name:        "github",
			Version:     "1.0.0",
			Description: "GitHub skill",
			Metadata: map[string]interface{}{
				"openclaw": map[string]interface{}{
					"credentials": []interface{}{
						map[string]interface{}{
							"id":    "openai_api_key",
							"env":   "OPENAI_API_KEY",
							"label": "OpenAI API Key",
						},
					},
				},
			},
		},
		Body: "body",
	}

	result, err := engine.Export(s, "xigua")
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	frontmatter := extractFrontmatter(t, string(result.Files["SKILL.md"]))
	metadata, ok := frontmatter["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("metadata missing: %#v", frontmatter)
	}
	openclaw, ok := metadata["openclaw"].(map[string]interface{})
	if !ok {
		t.Fatalf("openclaw metadata missing: %#v", metadata)
	}
	credentials, ok := openclaw["credentials"].([]interface{})
	if !ok || len(credentials) != 1 {
		t.Fatalf("credentials = %#v, want 1 item", openclaw["credentials"])
	}
	requires, ok := frontmatter["requires"].([]interface{})
	if !ok || len(requires) != 1 || requires[0] != "OPENAI_API_KEY" {
		t.Fatalf("requires = %#v, want derived OPENAI_API_KEY", frontmatter["requires"])
	}
	if _, ok := result.Files["skill.json"]; !ok {
		t.Fatalf("skill.json missing from xigua export: %#v", result.Files)
	}
}

func extractFrontmatter(t *testing.T, content string) map[string]interface{} {
	t.Helper()

	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---\n") {
		t.Fatalf("content missing frontmatter: %q", content)
	}
	rest := strings.TrimPrefix(trimmed, "---\n")
	end := strings.Index(rest, "\n---")
	if end == -1 {
		t.Fatalf("content missing frontmatter terminator: %q", content)
	}

	frontmatter := strings.TrimSpace(rest[:end])
	var manifest map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &manifest); err != nil {
		t.Fatalf("yaml unmarshal failed: %v", err)
	}
	return manifest
}
