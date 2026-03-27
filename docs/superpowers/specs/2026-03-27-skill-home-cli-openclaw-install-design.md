# Skill Home CLI OpenClaw Install Support Design

## 1. Goal

Extend `skill-home` CLI so `openclaw` becomes a first-class installation platform alongside `claude`, `copilot`, `cursor`, and `codex`.

The primary outcome is:

- `skill-home install <skill-ref> --ide openclaw` works.
- `skill-home sync --ide openclaw` works.
- `skill-home uninstall <skill-ref> --ide openclaw` works.
- `skill-home doctor` reports OpenClaw paths and sync capability.
- No new "host" abstraction is introduced. `--ide` remains the only selector.

This is intentionally scoped to installation and local platform support. It does not redesign the whole CLI platform model.

## 2. Research Basis

The default OpenClaw skills locations are based on public OpenClaw documentation:

- [OpenClaw Skills](https://docs.openclaw.ai/skills)
- [OpenClaw ClawHub](https://docs.openclaw.ai/tools/clawhub)
- [OpenClaw FAQ](https://docs.openclaw.ai/start/faq)
- [OpenClaw Creating Skills](https://docs.openclaw.ai/tools/creating-skills)

Key facts from those docs:

- Workspace skills live in `<workspace>/skills`.
- Shared skills live in `~/.openclaw/skills`.
- `clawhub install` installs into `./skills` by default, which OpenClaw treats as `<workspace>/skills`.

That means Skill Home should not invent a new OpenClaw project path like `.openclaw/skills`. The correct project-level default is `skills/`.

## 3. Current State

Today the CLI only knows these targets:

- `claude`
- `copilot`
- `cursor`
- `codex`

Relevant current behavior:

- `install` is a composed flow: pull -> optional security scan -> parse -> sync.
- `--ide` already behaves like a target platform selector, even though the flag is named `ide`.
- sync mode `auto` is chosen from the target adapter's `SupportsSymlink()` capability.
- directory-based platforms like Claude and Codex prefer symlink and fall back to mirror if symlink creation fails.
- Cursor is special because it writes a single `.mdc` file and does not support symlink mode.

The gap is not conceptual complexity. The gap is that `openclaw` is missing from config, path resolution, adapter creation, command validation, and operator-facing docs.

## 4. Scope

### 4.1 In scope

- Add `openclaw` to local platform configuration.
- Add OpenClaw project/global path resolution.
- Add an OpenClaw adapter.
- Extend install/sync/uninstall/doctor to recognize `openclaw`.
- Extend tests and docs accordingly.

### 4.2 Out of scope

- No new `--host` flag.
- No new runtime detection based on which AI client invoked the command.
- No OpenClaw-specific registry publishing flow.
- No `create`, `export`, `preview`, or `import` support in this task.
- No plugin-aware OpenClaw metadata generation.

## 5. Design Decision

Treat `openclaw` as a first-class target platform, not as an alias of `codex`.

This is the right middle ground:

- lighter than a whole platform-system rewrite
- cleaner than mapping `openclaw` to `codex`
- consistent with the existing adapter-based architecture

## 6. Default Paths

Skill Home should use these OpenClaw defaults:

- project path: `skills`
- global path: `~/.openclaw/skills`

Rationale:

- This matches OpenClaw workspace loading behavior.
- It matches ClawHub's install behavior.
- It keeps Skill Home interoperable with existing OpenClaw workflows instead of creating a parallel folder convention.

These defaults must remain configurable through `config.yaml`.

## 7. Behavioral Model

### 7.1 `install`

`skill-home install <skill-ref> --ide openclaw` will:

1. pull the requested skill into the local cache
2. run the existing install-time security scan unless skipped
3. parse the cached skill
4. sync the parsed skill into the OpenClaw target path

This remains the same composed install pipeline that already exists for other platforms.

### 7.2 `sync`

`skill-home sync --ide openclaw` will treat OpenClaw as a directory-based skill platform:

- default `auto` mode prefers symlink
- if symlink is unsupported or fails, it falls back to mirror

### 7.3 `uninstall`

`skill-home uninstall <skill-ref> --ide openclaw` removes the installed OpenClaw target entry and then follows the existing cache-removal behavior when `--keep-cache` is not set.

### 7.4 `doctor`

`skill-home doctor` must report:

- whether OpenClaw is enabled
- project path
- global path
- whether symlink mode is supported for OpenClaw

## 8. File Format Strategy

OpenClaw support should reuse the existing directory-style skill package layout:

- `SKILL.md`
- optional `references/`
- optional `scripts/`

The initial OpenClaw adapter should therefore mirror the Claude/Codex adapter family rather than the Cursor adapter family.

This is a deliberate compatibility choice:

- low implementation risk
- preserves current skill package shape
- avoids premature OpenClaw-specific transformation logic

If OpenClaw later needs richer metadata shaping, that can be a follow-up task.

## 9. Code Changes

### 9.1 Config layer

Update [config.go](/home/zhuyue/code/skill-manager/skill-home-cli/internal/config/config.go):

- add `OpenClaw IDE` to `IDEConfig`
- load `ide.openclaw.enabled`
- load `ide.openclaw.project_path`
- load `ide.openclaw.global_path`
- set defaults:
  - `ide.openclaw.enabled = false`
  - `ide.openclaw.project_path = "skills"`
  - `ide.openclaw.global_path = "~/.openclaw/skills"`
- expand `~` for the OpenClaw global path

### 9.2 Path resolution

Update [paths.go](/home/zhuyue/code/skill-manager/skill-home-cli/internal/config/paths.go):

- support `openclaw` in `GetIDEProjectPath`
- support `openclaw` in `GetIDEGlobalPath`

### 9.3 Adapter layer

Add a new adapter file under `internal/ide/`:

- `OpenClawAdapter`

Behavior:

- target path is `<base>/<skill-name>`
- install writes `SKILL.md`, `references/`, and `scripts/`
- uninstall removes the skill directory
- list inspects directories containing `SKILL.md`
- `SupportsSymlink()` returns `true`

Update [adapter.go](/home/zhuyue/code/skill-manager/skill-home-cli/internal/ide/adapter.go) so `NewAdapter("openclaw", ...)` works.

### 9.4 Command layer

Update command help text and target enumeration in:

- [install.go](/home/zhuyue/code/skill-manager/skill-home-cli/internal/cmd/install.go)
- [sync.go](/home/zhuyue/code/skill-manager/skill-home-cli/internal/cmd/sync.go)
- [uninstall.go](/home/zhuyue/code/skill-manager/skill-home-cli/internal/cmd/uninstall.go)
- [update.go](/home/zhuyue/code/skill-manager/skill-home-cli/internal/cmd/update.go)
- [doctor.go](/home/zhuyue/code/skill-manager/skill-home-cli/internal/cmd/doctor.go)

`getTargetIDEs()` must include OpenClaw when enabled in config.

### 9.5 Documentation

Update CLI docs so `openclaw` appears everywhere `--ide` values are documented.

## 10. Testing Strategy

### 10.1 Config tests

- verify `ide.openclaw.*` fields load from config
- verify defaults are applied correctly
- verify `~/.openclaw/skills` expands correctly

### 10.2 Path tests

- verify project path resolves to `<project-root>/skills`
- verify global path resolves to the configured OpenClaw global path

### 10.3 Adapter tests

- install writes `SKILL.md`
- install writes `references/` and `scripts/`
- uninstall removes the installed directory
- list returns valid installed OpenClaw skills

### 10.4 Command tests

- `getTargetIDEs()` includes `openclaw` when enabled
- root-level or command-level tests verify `--ide openclaw` flows do not reject the value
- doctor output path handling does not regress

## 11. Risks

### 11.1 Project path collision

Using `skills/` at project root is much broader than `.claude/skills` or `.agents/skills`.

This is still the correct default because it matches OpenClaw, but it means:

- users may already have a `skills/` folder for unrelated purposes
- project-root detection becomes more important

This risk is acceptable because the path is configurable and OpenClaw compatibility matters more than local symmetry.

### 11.2 Incomplete OpenClaw-specific metadata

OpenClaw supports richer metadata than Skill Home currently models.

This design intentionally does not try to generate those extra fields. The first version should install a valid skill directory, not emulate the entire OpenClaw ecosystem.

## 12. Acceptance Criteria

This task is complete when:

- `skill-home install <skill-ref> --ide openclaw` installs into the OpenClaw project/global path
- `skill-home sync --ide openclaw` works
- `skill-home uninstall <skill-ref> --ide openclaw` works
- `skill-home doctor` reports OpenClaw
- config supports `ide.openclaw`
- automated tests cover the new path and adapter behavior
- docs mention `openclaw` everywhere users choose a platform
