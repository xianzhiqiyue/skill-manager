# Skill Home Hosted CLI Release Assets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Host `skill-home` CLI release assets on Skill Home itself, make `install.sh` and `self-update` use hosted assets first, and keep GitHub Release as fallback.

**Architecture:** Reuse the existing static web serving path. Deployment will publish a versioned `releases/` tree alongside `install.sh`; the installer and self-update logic will resolve hosted metadata and artifact URLs first, then fall back to the current GitHub sources. No new service or storage backend is introduced.

**Tech Stack:** Go 1.21, Gin, Bash, GitHub Actions, existing package-release script, Go unit tests

---

## File Structure Map

### Existing files to modify

- Modify: `skill-home-server/internal/webui/webui.go`
  - Serve `/releases/*` as static files and return `404` for missing release assets.
- Modify: `skill-home-server/internal/webui/webui_test.go`
  - Cover hosted release asset serving and missing-asset behavior.
- Modify: `skill-home-cli/install.sh`
  - Prefer hosted `latest.json` and hosted release assets, then fall back to GitHub.
- Modify: `skill-home-cli/internal/selfupdate/updater.go`
  - Mirror hosted-first, GitHub-fallback resolution.
- Modify: `skill-home-cli/internal/selfupdate/updater_test.go`
  - Add hosted metadata/download and fallback coverage.
- Modify: `skill-home-cli/scripts/package-release.sh`
  - Emit `latest.json`.
- Modify: `.github/workflows/release-cli.yml`
  - Keep packaged release directory suitable for Skill Home mirroring.
- Modify: `deploy-update.sh`
  - Publish and roll back the `releases/` directory.
- Modify: `skill-home-cli/README.md`
  - Document hosted-first install/update behavior.
- Modify: `README.md`
  - Document Skill Home-hosted CLI assets.

### New files to create

- Create: `skill-home-cli/install_test.sh` is **not needed**
  - Keep installer smoke verification in shell commands instead of introducing a separate shell test harness.

---

### Task 1: Add failing server tests for hosted release assets

**Files:**
- Modify: `skill-home-server/internal/webui/webui_test.go`
- Modify: `skill-home-server/internal/webui/webui.go`

- [ ] **Step 1: Write the failing tests**
  - Add a test that places `releases/latest.json` under the served root and expects `GET /releases/latest.json` to return the JSON payload.
  - Add a test that places `releases/v1.2.3/skill-home-linux-amd64.tar.gz` under the served root and expects the asset bytes back from `GET /releases/v1.2.3/skill-home-linux-amd64.tar.gz`.
  - Add a test that requests a missing `/releases/v9.9.9/checksums.txt` path and expects `404` instead of `index.html`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/webui -run 'TestRegisterServesHostedReleaseAssets|TestRegisterMissingReleaseAssetReturns404'`
Expected: FAIL because `/releases/...` currently falls into the generic SPA handling path.

- [ ] **Step 3: Implement minimal server behavior**
  - Treat `/releases/` like `/api/` and `/health` for missing-path handling.
  - If the requested release asset exists under `distDir`, serve it directly.
  - If it does not exist, return JSON `404` instead of SPA fallback.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/webui`
Expected: PASS

### Task 2: Add failing self-update tests for hosted-first resolution

**Files:**
- Modify: `skill-home-cli/internal/selfupdate/updater_test.go`
- Modify: `skill-home-cli/internal/selfupdate/updater.go`

- [ ] **Step 1: Write the failing tests**
  - Add a test where hosted `/releases/latest.json` and hosted asset URLs exist, and verify the updater installs from hosted URLs without hitting GitHub paths.
  - Add a test where hosted metadata or hosted asset URLs return `404`, and verify the updater falls back to GitHub API/download URLs.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/selfupdate -run 'TestUpdaterUsesHostedReleaseAssetsFirst|TestUpdaterFallsBackToGitHubWhenHostedAssetsFail'`
Expected: FAIL because the updater only knows GitHub API and GitHub download URLs.

- [ ] **Step 3: Implement minimal hosted-first resolution**
  - Add hosted metadata/download base configuration with sensible defaults.
  - Resolve latest version from hosted `latest.json` first.
  - Download archive and checksums from hosted URLs first.
  - Retry metadata/download against GitHub on hosted failure.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/selfupdate`
Expected: PASS

### Task 3: Add failing installer test coverage through shell smoke commands

**Files:**
- Modify: `skill-home-cli/install.sh`
- Modify: `skill-home-cli/scripts/package-release.sh`

- [ ] **Step 1: Prepare a failing local smoke scenario**
  - Package release assets locally.
  - Create a temporary hosted release tree with `releases/latest.json` and one version directory.
  - Run `install.sh` with `SKILL_HOME_RELEASES_BASE_URL` pointed at the local server and with GitHub access intentionally irrelevant.
  - Confirm the current script cannot resolve hosted metadata or hosted assets yet.

- [ ] **Step 2: Implement minimal installer changes**
  - Emit `latest.json` from `package-release.sh`.
  - Update `install.sh` to prefer hosted `latest.json`, then hosted asset URLs, then GitHub.

- [ ] **Step 3: Re-run smoke verification**

Run:
`bash skill-home-cli/scripts/package-release.sh v9.9.9 /tmp/skill-home-dist`
`python3 -m http.server --directory /tmp/hosted-release-root 18080`
`SKILL_HOME_RELEASES_BASE_URL=http://127.0.0.1:18080/releases HOME=/tmp/install-home bash skill-home-cli/install.sh v9.9.9`

Expected: PASS, with the archive fetched from the local hosted mirror.

### Task 4: Extend release/deploy/docs and verify end-to-end

**Files:**
- Modify: `.github/workflows/release-cli.yml`
- Modify: `deploy-update.sh`
- Modify: `skill-home-cli/README.md`
- Modify: `README.md`

- [ ] **Step 1: Update workflow and deployment scripts**
  - Ensure the release output directory includes `latest.json`.
  - Make deployment back up, replace, and roll back `/opt/skill-home/releases`.

- [ ] **Step 2: Update docs**
  - Explain that public installs prefer Skill Home-hosted release assets and only fall back to GitHub.

- [ ] **Step 3: Run verification**

Run:
- `go test ./...`
- `bash -n skill-home-cli/install.sh`
- `go test ./internal/selfupdate`
- `npm run build` only if web assets are touched beyond server static routing expectations

Expected: PASS

- [ ] **Step 4: Optional deployment verification after publish**

Run on target environment:
- `curl -fsSL http://47.122.112.210:8080/releases/latest.json`
- `curl -I http://47.122.112.210:8080/releases/<current-version>/checksums.txt`
- `curl -I http://47.122.112.210:8080/releases/<current-version>/skill-home-linux-amd64.tar.gz`

Expected: all endpoints respond successfully.
