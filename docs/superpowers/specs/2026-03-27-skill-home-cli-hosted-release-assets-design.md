# Skill Home Hosted CLI Release Assets Design

## 1. Goal

Move the public `skill-home` CLI download path from "install script on Skill Home + binaries on GitHub" to "install script and release assets on Skill Home, with GitHub Release retained as fallback."

The intended user-visible outcome is:

- `http://47.122.112.210:8080/install.sh` still works.
- The script downloads `checksums.txt` and platform archives from Skill Home first.
- `skill-home self-update` uses the same hosted source first.
- GitHub Release remains available as a fallback when hosted assets are missing.

This solves the current China-mainland reliability problem without discarding the existing GitHub release pipeline.

## 2. Current State

Today the download chain is split:

- Skill Home serves `/install.sh`.
- `install.sh` still resolves versions from GitHub Releases and downloads archives/checksums from GitHub.
- `self-update` also fetches release metadata and artifacts from GitHub.
- the GitHub Actions release workflow publishes release artifacts to GitHub only.
- the deployment script updates `server`, `skill-home`, `web`, and `install.sh`, but does not ship any versioned CLI release assets.

That means the public install entrypoint is on Skill Home, but the large downloads still depend on GitHub availability and latency.

## 3. Design Decision

Adopt a hosted-release layout on Skill Home and keep GitHub Release as a compatibility fallback.

### 3.1 Hosted layout

Skill Home will expose a static release tree:

- `/releases/latest.json`
- `/releases/<version>/checksums.txt`
- `/releases/<version>/skill-home-darwin-amd64.tar.gz`
- `/releases/<version>/skill-home-darwin-arm64.tar.gz`
- `/releases/<version>/skill-home-linux-amd64.tar.gz`
- `/releases/<version>/skill-home-linux-arm64.tar.gz`
- `/releases/<version>/skill-home-windows-amd64.zip`

The server does not need dynamic artifact generation. It only needs to serve a versioned static directory that deployments populate.

### 3.2 Version discovery

`install.sh` and `self-update` currently need a "latest version" endpoint. GitHub provides that via API, but hosted assets need their own version source.

Skill Home should therefore serve a small `latest.json` file:

```json
{"tag_name":"v0.2.4"}
```

This keeps version discovery stable and cheap, and avoids scraping HTML or depending on GitHub API rate limits.

### 3.3 Fallback model

Hosted source is primary. GitHub Release is secondary.

Behavior:

- If hosted `latest.json` exists, use it.
- If hosted version metadata or hosted artifacts fail, fall back to GitHub.
- If a user passes an explicit version, skip latest discovery and resolve hosted asset URLs directly first.

This ensures:

- fast path for domestic users
- no breakage if hosted assets lag or a deployment misses one version
- no need to change existing GitHub release behavior immediately

## 4. Scope

### 4.1 In scope

- Serve versioned CLI release assets from Skill Home.
- Add a hosted `latest.json`.
- Update `install.sh` to prefer hosted version metadata and hosted downloads.
- Update `self-update` to prefer hosted version metadata and hosted downloads.
- Extend deployment so hosted release assets are uploaded and rolled back together with binaries.
- Extend the release workflow so packaged CLI assets are easy to sync to Skill Home.

### 4.2 Out of scope

- No registry redesign.
- No CDN or object storage migration.
- No artifact signing beyond the existing checksum flow.
- No removal of GitHub Release publishing.
- No generic artifact hosting platform for unrelated files.

## 5. File and Route Layout

### 5.1 On-disk deployment layout

Under `/opt/skill-home`, add:

- `releases/latest.json`
- `releases/<version>/...`

This mirrors the public URL structure and keeps deployment simple.

### 5.2 Public URLs

Expose these through the existing web UI static routing:

- `GET /releases/latest.json`
- `GET /releases/<version>/<asset>`
- `HEAD` requests should work for those assets via the normal static file serving behavior.

The existing SPA fallback must not swallow missing release assets. A missing release file should return `404`, not `index.html`.

## 6. Installer Behavior

`install.sh` should support three source concepts:

- hosted metadata base, default `http://47.122.112.210:8080/releases`
- hosted download base, same root as metadata
- GitHub fallback, default existing GitHub Release URLs

Proposed environment variables:

- `SKILL_HOME_RELEASES_BASE_URL`
- `SKILL_HOME_RELEASE_REPO`

Behavior:

1. detect platform as today
2. resolve version:
   - explicit version wins
   - otherwise try hosted `latest.json`
   - if hosted latest fails, fall back to GitHub API latest
3. download archive/checksum:
   - try hosted `/releases/<version>/...`
   - if either download fails, retry against GitHub Release
4. verify checksum and install as today

## 7. Self-Update Behavior

`internal/selfupdate` should mirror the installer source order.

Add support for:

- hosted metadata base URL
- hosted download base URL
- GitHub fallback

This keeps `install.sh` and `self-update` consistent, which matters for operational debugging.

## 8. Release and Deployment Flow

### 8.1 Packaging

The existing packaging script already creates the right assets. It only needs one more output:

- `latest.json`

That file should contain the tag name used for the release build.

### 8.2 GitHub Actions

The GitHub workflow should continue publishing release assets to GitHub.

It should also keep the packaged asset directory intact so it can be reused for deployment or manual upload. No second packaging format is required.

### 8.3 Server deployment

Deployment must gain a release-assets step:

- upload a staged `releases/` directory or a single version subtree plus `latest.json`
- back up the existing `releases/`
- activate the new assets atomically enough for rollback

Rollback should restore:

- `server`
- `skill-home`
- `install.sh`
- `releases/`

## 9. Error Handling

Key failure modes and expected behavior:

- Hosted `latest.json` missing: fall back to GitHub latest.
- Hosted asset missing for a known version: fall back to GitHub download.
- Hosted asset present but checksum file missing: retry the pair from GitHub.
- Hosted URL returns HTML due to routing bug: treat as download failure, do not install.
- Server receives unknown `/releases/...` path: return `404`, not SPA HTML.

## 10. Testing Strategy

### 10.1 Server tests

- verify `/releases/latest.json` is served when present
- verify `/releases/<version>/<asset>` is served when present
- verify missing `/releases/...` paths return `404`
- verify SPA fallback still serves `index.html` for app routes

### 10.2 CLI unit tests

- `self-update` prefers hosted latest metadata
- `self-update` falls back to GitHub latest metadata
- `self-update` prefers hosted artifact URLs
- `self-update` falls back to GitHub artifacts when hosted download fails

### 10.3 Installer smoke tests

- run `install.sh` against a local mock hosted release tree
- verify install succeeds without GitHub URLs being needed

### 10.4 Deployment verification

- confirm deployed server returns:
  - `/install.sh`
  - `/releases/latest.json`
  - one concrete archive URL for the current version

## 11. Implementation Notes

- Keep URL construction centralized so `install.sh` and `self-update` do not drift.
- Prefer simple static file hosting over new API endpoints.
- Do not rename current release assets; reuse the archive filenames already published on GitHub.
- Treat hosted assets as a distribution mirror, not a new source of truth. GitHub Release remains the canonical public release history.
