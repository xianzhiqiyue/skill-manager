package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestInitLoadsSnakeCaseConfigFields(t *testing.T) {
	t.Parallel()

	viper.Reset()
	t.Cleanup(viper.Reset)

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")

	content := []byte(`version: "1.0"
local:
  skills_dir: "/tmp/skill-home-cache"
  default_namespace: "@tester"
ide:
  codex:
    enabled: true
    project_path: ".codex/skills"
    global_path: "/tmp/codex-skills"
sync:
  mode: "mirror"
  conflict_strategy: "project_wins"
  auto_sync_on_push: true
security:
  scan_on_install: true
  allow_remote_scripts: false
`)
	if err := os.WriteFile(configFile, content, 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if err := Init(configFile); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	if got := C.Local.SkillsDir; got != "/tmp/skill-home-cache" {
		t.Fatalf("unexpected skills_dir: %q", got)
	}
	if got := C.Local.DefaultNamespace; got != "@tester" {
		t.Fatalf("unexpected default_namespace: %q", got)
	}
	if got := C.IDE.Codex.ProjectPath; got != ".codex/skills" {
		t.Fatalf("unexpected codex project_path: %q", got)
	}
	if got := C.IDE.Codex.GlobalPath; got != "/tmp/codex-skills" {
		t.Fatalf("unexpected codex global_path: %q", got)
	}
	if got := C.Sync.ConflictStrategy; got != "project_wins" {
		t.Fatalf("unexpected conflict_strategy: %q", got)
	}
	if got := C.Sync.AutoSyncOnPush; !got {
		t.Fatalf("unexpected auto_sync_on_push: %t", got)
	}
	if got := C.Security.ScanOnInstall; !got {
		t.Fatalf("unexpected scan_on_install: %t", got)
	}
}

func TestInitAllowsRegistryEnvOverrides(t *testing.T) {
	t.Parallel()

	viper.Reset()
	t.Cleanup(func() {
		_ = os.Unsetenv("SKILL_HOME_REGISTRY_ENDPOINT")
		_ = os.Unsetenv("SKILL_HOME_API_KEY")
		viper.Reset()
	})

	if err := os.Setenv("SKILL_HOME_REGISTRY_ENDPOINT", "https://registry.example.test"); err != nil {
		t.Fatalf("Setenv(endpoint) returned error: %v", err)
	}
	if err := os.Setenv("SKILL_HOME_API_KEY", "token-123"); err != nil {
		t.Fatalf("Setenv(api_key) returned error: %v", err)
	}

	if err := Init(""); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	if got := C.Registry.Endpoint; got != "https://registry.example.test" {
		t.Fatalf("unexpected registry endpoint: %q", got)
	}
	if got := C.Registry.APIKey; got != "token-123" {
		t.Fatalf("unexpected registry api key: %q", got)
	}
}
