package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/skill-home/cli/internal/ide"
	"github.com/skill-home/cli/internal/skill"
)

func TestCodexMirrorSyncPreservesAssetsAndOpenAIConfig(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	assetPath := filepath.Join(sourceDir, "assets", "images", "diagram.png")
	if err := os.MkdirAll(filepath.Dir(assetPath), 0755); err != nil {
		t.Fatalf("MkdirAll asset directory returned error: %v", err)
	}
	if err := os.WriteFile(assetPath, []byte("diagram"), 0644); err != nil {
		t.Fatalf("WriteFile asset returned error: %v", err)
	}

	openAIConfigPath := filepath.Join(sourceDir, "agents", "openai.yaml")
	if err := os.MkdirAll(filepath.Dir(openAIConfigPath), 0755); err != nil {
		t.Fatalf("MkdirAll agents directory returned error: %v", err)
	}
	if err := os.WriteFile(openAIConfigPath, []byte("interface:\n  display_name: Demo\n"), 0644); err != nil {
		t.Fatalf("WriteFile openai config returned error: %v", err)
	}

	targetDir := t.TempDir()
	engine := NewEngine(ModeMirror)
	adapter := ide.NewCodexAdapter(targetDir)
	err := engine.Sync(&skill.Skill{
		Path: sourceDir,
		Manifest: skill.Manifest{
			Name:        "demo-skill",
			Version:     "1.0.0",
			Description: "Demo skill",
		},
		Body: "Body",
	}, adapter)
	if err != nil {
		t.Fatalf("Sync returned error: %v", err)
	}

	installedSkillDir := filepath.Join(targetDir, "demo-skill")
	if got, err := os.ReadFile(filepath.Join(installedSkillDir, "assets", "images", "diagram.png")); err != nil {
		t.Fatalf("expected mirrored asset file: %v", err)
	} else if string(got) != "diagram" {
		t.Fatalf("unexpected mirrored asset content: %q", got)
	}
	if got, err := os.ReadFile(filepath.Join(installedSkillDir, "agents", "openai.yaml")); err != nil {
		t.Fatalf("expected mirrored openai config: %v", err)
	} else if string(got) != "interface:\n  display_name: Demo\n" {
		t.Fatalf("unexpected mirrored openai config content: %q", got)
	}
}
