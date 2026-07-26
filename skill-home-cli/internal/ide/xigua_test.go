package ide

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNewAdapterReturnsXiguaAdapter(t *testing.T) {
	t.Parallel()

	adapter, err := NewAdapter("xigua", "/tmp/skills")
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

	if got := adapter.GetType(); got != "xigua" {
		t.Fatalf("unexpected adapter type: %q", got)
	}
}

func TestXiguaAdapterInstallWritesPackageLayout(t *testing.T) {
	t.Parallel()

	targetDir := t.TempDir()
	adapter := NewXiguaAdapter(targetDir)

	if got := adapter.GetTargetPath("demo-skill"); got != filepath.Join(targetDir, "demo-skill") {
		t.Fatalf("unexpected target path: %q", got)
	}
	if !adapter.SupportsSymlink() {
		t.Fatalf("expected symlink support")
	}

	err := adapter.InstallSkill(SkillData{
		Name:     "demo-skill",
		Manifest: []byte("---\nname: demo-skill\nversion: 1.0.0\ndescription: demo"),
		Body:     "Body",
		References: map[string][]byte{
			"guide.md": []byte("guide"),
		},
		Scripts: map[string][]byte{
			"run.sh": []byte("#!/usr/bin/env bash\necho ok\n"),
		},
	})
	if err != nil {
		t.Fatalf("InstallSkill returned error: %v", err)
	}

	skillPath := filepath.Join(targetDir, "demo-skill")
	for _, path := range []string{
		filepath.Join(skillPath, "SKILL.md"),
		filepath.Join(skillPath, "skill.json"),
		filepath.Join(skillPath, "references", "guide.md"),
		filepath.Join(skillPath, "scripts", "run.sh"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}

	scriptInfo, err := os.Stat(filepath.Join(skillPath, "scripts", "run.sh"))
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if scriptInfo.Mode().Perm()&0o111 == 0 {
		t.Fatalf("expected script to be executable, mode=%v", scriptInfo.Mode())
	}

	manifestBytes, err := os.ReadFile(filepath.Join(skillPath, "skill.json"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("skill.json is not valid JSON: %v", err)
	}
	if manifest["name"] != "demo-skill" || manifest["slug"] != "demo-skill" || manifest["sourceType"] != "skill-home" {
		t.Fatalf("unexpected skill.json: %#v", manifest)
	}

	skills, err := adapter.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills returned error: %v", err)
	}
	if !reflect.DeepEqual(skills, []string{"demo-skill"}) {
		t.Fatalf("unexpected skills: %#v", skills)
	}
}

func TestXiguaAdapterUninstallRemovesSkillDirectory(t *testing.T) {
	t.Parallel()

	targetDir := t.TempDir()
	adapter := NewXiguaAdapter(targetDir)

	skillPath := filepath.Join(targetDir, "demo-skill")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("body"), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if err := adapter.UninstallSkill("demo-skill"); err != nil {
		t.Fatalf("UninstallSkill returned error: %v", err)
	}

	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Fatalf("expected skill directory to be removed, got err=%v", err)
	}
}
