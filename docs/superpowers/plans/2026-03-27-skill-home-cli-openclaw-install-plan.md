# Skill Home CLI OpenClaw Install Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `openclaw` as a first-class install platform so `install`, `sync`, `uninstall`, and `doctor` support OpenClaw using the documented default paths `skills/` and `~/.openclaw/skills`.

**Architecture:** Keep the existing CLI structure: config and path resolution define platform defaults, `internal/ide` provides the platform adapter, and command files reuse the shared sync pipeline. Do not introduce a host abstraction; extend the current `--ide` selector and let sync mode continue to be decided by the target adapter's symlink capability.

**Tech Stack:** Go 1.21, Cobra, Viper, Vitest-free Go unit tests, existing CLI adapter/sync architecture

---

## File Structure Map

### Existing files to modify

- Modify: `skill-home-cli/internal/config/config.go`
  - Add `OpenClaw` config loading and defaults.
- Modify: `skill-home-cli/internal/config/config_test.go`
  - Verify `ide.openclaw.*` values load and expand correctly.
- Modify: `skill-home-cli/internal/config/paths.go`
  - Resolve OpenClaw project/global paths.
- Modify: `skill-home-cli/internal/config/paths_test.go`
  - Cover `skills/` and `~/.openclaw/skills`.
- Modify: `skill-home-cli/internal/ide/adapter.go`
  - Teach `NewAdapter` about `openclaw`.
- Modify: `skill-home-cli/internal/cmd/install.go`
- Modify: `skill-home-cli/internal/cmd/sync.go`
- Modify: `skill-home-cli/internal/cmd/uninstall.go`
- Modify: `skill-home-cli/internal/cmd/update.go`
- Modify: `skill-home-cli/internal/cmd/doctor.go`
  - Extend docs, target selection, and doctor reporting.
- Modify: `skill-home-cli/README.md`
  - Document OpenClaw platform support and path defaults.

### New files to create

- Create: `skill-home-cli/internal/ide/openclaw.go`
  - Directory-style adapter for OpenClaw installs.
- Create: `skill-home-cli/internal/ide/openclaw_test.go`
  - Verify install/list/uninstall behavior.
- Create: `skill-home-cli/internal/cmd/openclaw_test.go`
  - Verify command-layer target selection includes OpenClaw.

---

### Task 1: Add OpenClaw config defaults and path resolution

**Files:**
- Modify: `skill-home-cli/internal/config/config.go`
- Modify: `skill-home-cli/internal/config/config_test.go`
- Modify: `skill-home-cli/internal/config/paths.go`
- Modify: `skill-home-cli/internal/config/paths_test.go`

- [ ] **Step 1: Write the failing config and path tests**

```go
func TestInitLoadsOpenClawConfigFields(t *testing.T) {
	content := []byte(`ide:
  openclaw:
    enabled: true
    project_path: "skills"
    global_path: "~/.openclaw/skills"
`)

	if err := Init(configFile); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	if got := C.IDE.OpenClaw.ProjectPath; got != "skills" {
		t.Fatalf("unexpected openclaw project_path: %q", got)
	}
	if !strings.Contains(C.IDE.OpenClaw.GlobalPath, ".openclaw/skills") {
		t.Fatalf("unexpected openclaw global_path: %q", C.IDE.OpenClaw.GlobalPath)
	}
}
```

```go
func TestPathResolverSupportsOpenClawPaths(t *testing.T) {
	C = &Config{
		IDE: IDEConfig{
			OpenClaw: IDE{
				Enabled: true,
				ProjectPath: "skills",
				GlobalPath: "/tmp/.openclaw/skills",
			},
		},
	}

	resolver := &PathResolver{projectRoot: "/tmp/project"}
	projectPath, _ := resolver.GetIDEProjectPath("openclaw")
	globalPath, _ := resolver.GetIDEGlobalPath("openclaw")

	if projectPath != "/tmp/project/skills" { t.Fatal(projectPath) }
	if globalPath != "/tmp/.openclaw/skills" { t.Fatal(globalPath) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config -run 'TestInitLoadsOpenClawConfigFields|TestPathResolverSupportsOpenClawPaths'`
Expected: FAIL because `IDEConfig` and `PathResolver` do not know `openclaw`.

- [ ] **Step 3: Implement minimal config and path support**

```go
type IDEConfig struct {
	Claude    IDE `yaml:"claude" mapstructure:"claude"`
	Copilot   IDE `yaml:"copilot" mapstructure:"copilot"`
	Cursor    IDE `yaml:"cursor" mapstructure:"cursor"`
	Codex     IDE `yaml:"codex" mapstructure:"codex"`
	OpenClaw  IDE `yaml:"openclaw" mapstructure:"openclaw"`
}
```

```go
viper.SetDefault("ide.openclaw.enabled", false)
viper.SetDefault("ide.openclaw.project_path", "skills")
viper.SetDefault("ide.openclaw.global_path", "~/.openclaw/skills")
```

```go
case "openclaw":
	return filepath.Join(p.projectRoot, C.IDE.OpenClaw.ProjectPath), nil
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skill-home-cli/internal/config/config.go \
  skill-home-cli/internal/config/config_test.go \
  skill-home-cli/internal/config/paths.go \
  skill-home-cli/internal/config/paths_test.go
git commit -m "feat: add openclaw config and path defaults"
```

### Task 2: Add the OpenClaw adapter

**Files:**
- Create: `skill-home-cli/internal/ide/openclaw.go`
- Create: `skill-home-cli/internal/ide/openclaw_test.go`
- Modify: `skill-home-cli/internal/ide/adapter.go`

- [ ] **Step 1: Write the failing adapter tests**

```go
func TestOpenClawAdapterInstallsAndListsSkill(t *testing.T) {
	adapter := NewOpenClawAdapter(t.TempDir())
	data := SkillData{
		Name: "github",
		Manifest: []byte("---\nname: github\nversion: 1.0.0\ndescription: demo"),
		Body: "body",
		References: map[string][]byte{"guide.md": []byte("ref")},
		Scripts: map[string][]byte{"setup.sh": []byte("echo ok")},
	}

	if err := adapter.InstallSkill(data); err != nil {
		t.Fatalf("InstallSkill returned error: %v", err)
	}

	got, err := adapter.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills returned error: %v", err)
	}
	if len(got) != 1 || got[0] != "github" {
		t.Fatalf("unexpected skills: %#v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ide -run 'TestOpenClawAdapterInstallsAndListsSkill'`
Expected: FAIL because `NewOpenClawAdapter` and the factory case do not exist.

- [ ] **Step 3: Implement the directory-style OpenClaw adapter**

```go
func (a *OpenClawAdapter) GetTargetPath(skillName string) string {
	return filepath.Join(a.targetPath, skillName)
}

func (a *OpenClawAdapter) SupportsSymlink() bool {
	return true
}
```

```go
case "openclaw":
	return NewOpenClawAdapter(targetPath), nil
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ide`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skill-home-cli/internal/ide/adapter.go \
  skill-home-cli/internal/ide/openclaw.go \
  skill-home-cli/internal/ide/openclaw_test.go
git commit -m "feat: add openclaw adapter"
```

### Task 3: Extend command-layer target selection and doctor reporting

**Files:**
- Modify: `skill-home-cli/internal/cmd/install.go`
- Modify: `skill-home-cli/internal/cmd/sync.go`
- Modify: `skill-home-cli/internal/cmd/uninstall.go`
- Modify: `skill-home-cli/internal/cmd/update.go`
- Modify: `skill-home-cli/internal/cmd/doctor.go`
- Create: `skill-home-cli/internal/cmd/openclaw_test.go`

- [ ] **Step 1: Write the failing command tests**

```go
func TestGetTargetIDEsIncludesOpenClawWhenEnabled(t *testing.T) {
	config.C = &config.Config{
		IDE: config.IDEConfig{
			OpenClaw: config.IDE{Enabled: true},
		},
	}

	got := getTargetIDEs(&syncOptions{})
	if !slices.Contains(got, "openclaw") {
		t.Fatalf("expected openclaw in %#v", got)
	}
}
```

```go
func TestSyncCommandHelpMentionsOpenClaw(t *testing.T) {
	cmd := newSyncCmd()
	flag := cmd.Flags().Lookup("ide")
	if !strings.Contains(flag.Usage, "openclaw") {
		t.Fatalf("usage = %q", flag.Usage)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cmd -run 'TestGetTargetIDEsIncludesOpenClawWhenEnabled|TestSyncCommandHelpMentionsOpenClaw'`
Expected: FAIL because `getTargetIDEs()` and flag usage strings do not include `openclaw`.

- [ ] **Step 3: Implement minimal command changes**

```go
cmd.Flags().StringVar(&opts.ide, "ide", "", "指定 IDE (claude/copilot/cursor/codex/openclaw)")
```

```go
if config.C.IDE.OpenClaw.Enabled {
	ides = append(ides, "openclaw")
}
```

```go
checkIDEPath("openclaw", config.C.IDE.OpenClaw.Enabled, config.C.IDE.OpenClaw.GlobalPath, resolver)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cmd`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add skill-home-cli/internal/cmd/install.go \
  skill-home-cli/internal/cmd/sync.go \
  skill-home-cli/internal/cmd/uninstall.go \
  skill-home-cli/internal/cmd/update.go \
  skill-home-cli/internal/cmd/doctor.go \
  skill-home-cli/internal/cmd/openclaw_test.go
git commit -m "feat: wire openclaw into install and sync commands"
```

### Task 4: Update docs and run full verification

**Files:**
- Modify: `skill-home-cli/README.md`

- [ ] **Step 1: Write the failing doc expectation as a focused grep check**

```bash
rg -n "openclaw" skill-home-cli/README.md
```

Expected: no relevant install-platform documentation yet.

- [ ] **Step 2: Update README**

```md
skill-home install @user/my-skill --ide openclaw
```

```yaml
ide:
  openclaw:
    enabled: false
    project_path: "skills"
    global_path: "~/.openclaw/skills"
```

- [ ] **Step 3: Run formatting and full test suite**

Run: `gofmt -w skill-home-cli/internal/config/*.go skill-home-cli/internal/ide/*.go skill-home-cli/internal/cmd/*.go`
Expected: files formatted with no output.

Run: `go test ./...`
Expected: PASS

- [ ] **Step 4: Run one focused behavioral check**

Run: `go run ./cmd/skill-home doctor`
Expected: output includes an `openclaw` line showing project/global path resolution.

Run: `go run ./cmd/skill-home sync --help`
Expected: `--ide` usage includes `openclaw`.

- [ ] **Step 5: Commit**

```bash
git add skill-home-cli/README.md
git commit -m "docs: describe openclaw install support"
```
