package config

import "testing"

func TestPathResolverSupportsCopilotPaths(t *testing.T) {
	t.Parallel()

	C = &Config{
		IDE: IDEConfig{
			Copilot: IDE{
				Enabled:     true,
				ProjectPath: ".github/skills",
				GlobalPath:  "/tmp/.copilot/skills",
			},
		},
	}

	resolver := &PathResolver{projectRoot: "/tmp/project"}

	projectPath, err := resolver.GetIDEProjectPath("copilot")
	if err != nil {
		t.Fatalf("GetIDEProjectPath returned error: %v", err)
	}
	if projectPath != "/tmp/project/.github/skills" {
		t.Fatalf("unexpected project path: %s", projectPath)
	}

	globalPath, err := resolver.GetIDEGlobalPath("copilot")
	if err != nil {
		t.Fatalf("GetIDEGlobalPath returned error: %v", err)
	}
	if globalPath != "/tmp/.copilot/skills" {
		t.Fatalf("unexpected global path: %s", globalPath)
	}
}
