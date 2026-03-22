package ide

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/skill-home/cli/internal/skill"
)

func TestConvertToCodexFormatIncludesScriptsAndReferences(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "references"), 0755); err != nil {
		t.Fatalf("MkdirAll references returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "scripts"), 0755); err != nil {
		t.Fatalf("MkdirAll scripts returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "references", "guide.md"), []byte("guide"), 0644); err != nil {
		t.Fatalf("WriteFile reference returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "scripts", "run.sh"), []byte("#!/usr/bin/env bash\necho ok\n"), 0644); err != nil {
		t.Fatalf("WriteFile script returned error: %v", err)
	}

	data := ConvertToCodexFormat(&skill.Skill{
		Path: tempDir,
		Manifest: skill.Manifest{
			Name:        "demo-skill",
			Version:     "1.0.0",
			Description: "Demo skill",
			Author:      "Tester",
		},
		Body: "Body",
	})

	if got := string(data.References["guide.md"]); got != "guide" {
		t.Fatalf("unexpected reference content: %q", got)
	}
	if got := string(data.Scripts["run.sh"]); got != "#!/usr/bin/env bash\necho ok\n" {
		t.Fatalf("unexpected script content: %q", got)
	}
}

func TestCodexAdapterInstallSkillWritesScripts(t *testing.T) {
	t.Parallel()

	targetDir := t.TempDir()
	adapter := NewCodexAdapter(targetDir)

	err := adapter.InstallSkill(SkillData{
		Name:       "demo-skill",
		Manifest:   []byte("---\nname: demo-skill\nversion: 1.0.0\ndescription: demo"),
		Body:       "Body",
		Scripts:    map[string][]byte{"run.sh": []byte("#!/usr/bin/env bash\necho ok\n")},
		References: map[string][]byte{"guide.md": []byte("guide")},
	})
	if err != nil {
		t.Fatalf("InstallSkill returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "demo-skill", "scripts", "run.sh")); err != nil {
		t.Fatalf("expected installed script file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "demo-skill", "references", "guide.md")); err != nil {
		t.Fatalf("expected installed reference file: %v", err)
	}
}
