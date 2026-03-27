package ide

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNewAdapterReturnsOpenClawAdapter(t *testing.T) {
	t.Parallel()

	adapter, err := NewAdapter("openclaw", "/tmp/skills")
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

	if got := adapter.GetType(); got != "openclaw" {
		t.Fatalf("unexpected adapter type: %q", got)
	}
}

func TestOpenClawAdapterInstallWritesDirectoryLayout(t *testing.T) {
	t.Parallel()

	targetDir := t.TempDir()
	adapter := NewOpenClawAdapter(targetDir)

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
	checkFile := func(path string) {
		t.Helper()
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}

	checkFile(filepath.Join(skillPath, "SKILL.md"))
	checkFile(filepath.Join(skillPath, "references", "guide.md"))
	scriptPath := filepath.Join(skillPath, "scripts", "run.sh")
	checkFile(scriptPath)
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("expected %s to exist: %v", scriptPath, err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("expected %s to be executable, mode=%v", scriptPath, info.Mode())
	}

	skills, err := adapter.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills returned error: %v", err)
	}
	if !reflect.DeepEqual(skills, []string{"demo-skill"}) {
		t.Fatalf("unexpected skills: %#v", skills)
	}
}

func TestOpenClawAdapterInstallCreatesEmptyDirectories(t *testing.T) {
	t.Parallel()

	targetDir := t.TempDir()
	adapter := NewOpenClawAdapter(targetDir)

	err := adapter.InstallSkill(SkillData{
		Name:     "empty-skill",
		Manifest: []byte("---\nname: empty-skill\nversion: 1.0.0\ndescription: empty"),
		Body:     "Body",
	})
	if err != nil {
		t.Fatalf("InstallSkill returned error: %v", err)
	}

	for _, path := range []string{
		filepath.Join(targetDir, "empty-skill", "references"),
		filepath.Join(targetDir, "empty-skill", "scripts"),
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", path)
		}
	}
}

func TestOpenClawAdapterUninstallRemovesSkillDirectory(t *testing.T) {
	t.Parallel()

	targetDir := t.TempDir()
	adapter := NewOpenClawAdapter(targetDir)

	skillPath := filepath.Join(targetDir, "demo-skill")
	if err := os.MkdirAll(filepath.Join(skillPath, "references"), 0755); err != nil {
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
