package config

import (
	"path/filepath"
	"testing"
)

func TestPathResolverSupportsCopilotPaths(t *testing.T) {
	projectRoot := t.TempDir()
	globalRoot := t.TempDir()

	C = &Config{
		IDE: IDEConfig{
			Copilot: IDE{
				Enabled:     true,
				ProjectPath: ".github/skills",
				GlobalPath:  filepath.Join(globalRoot, ".copilot", "skills"),
			},
		},
	}

	resolver := &PathResolver{projectRoot: projectRoot}

	projectPath, err := resolver.GetIDEProjectPath("copilot")
	if err != nil {
		t.Fatalf("GetIDEProjectPath returned error: %v", err)
	}
	if projectPath != filepath.Join(projectRoot, ".github", "skills") {
		t.Fatalf("unexpected project path: %s", projectPath)
	}

	globalPath, err := resolver.GetIDEGlobalPath("copilot")
	if err != nil {
		t.Fatalf("GetIDEGlobalPath returned error: %v", err)
	}
	if globalPath != filepath.Join(globalRoot, ".copilot", "skills") {
		t.Fatalf("unexpected global path: %s", globalPath)
	}
}

func TestPathResolverSupportsOpenClawPaths(t *testing.T) {
	projectRoot := t.TempDir()
	globalRoot := t.TempDir()

	C = &Config{
		IDE: IDEConfig{
			OpenClaw: IDE{
				Enabled:     true,
				ProjectPath: "skills",
				GlobalPath:  filepath.Join(globalRoot, ".openclaw", "skills"),
			},
		},
	}

	resolver := &PathResolver{projectRoot: projectRoot}

	projectPath, err := resolver.GetIDEProjectPath("openclaw")
	if err != nil {
		t.Fatalf("GetIDEProjectPath returned error: %v", err)
	}
	if projectPath != filepath.Join(projectRoot, "skills") {
		t.Fatalf("unexpected project path: %s", projectPath)
	}

	globalPath, err := resolver.GetIDEGlobalPath("openclaw")
	if err != nil {
		t.Fatalf("GetIDEGlobalPath returned error: %v", err)
	}
	if globalPath != filepath.Join(globalRoot, ".openclaw", "skills") {
		t.Fatalf("unexpected global path: %s", globalPath)
	}
}
