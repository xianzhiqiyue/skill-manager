# Skill Home Public OSS Skill Package Delivery Design

## 1. Goal

Move public skill package delivery from the Skill Home application server to Alibaba Cloud OSS while keeping Skill Home as the management, search, and publishing system.

The intended user-visible outcome is:

- public skill metadata continues to come from Skill Home
- newly published public skills return an OSS-backed `download_url`
- the web UI downloads public skills from OSS instead of the application server
- the CLI also prefers the returned `download_url`
- the legacy `GET /api/v1/download/:namespace/:name/:version` route remains available during a compatibility period

This gives us object-storage-backed package delivery now without blocking on private skill authorization design.

## 2. Current State

Today Skill Home behaves as both metadata registry and binary delivery gateway:

- skill archives are uploaded through Skill Home and stored by the object storage abstraction
- the database stores skill metadata and a `storage_path` per version
- public API responses return `/api/v1/download/...` as the `download_url`
- the download route reads the object, optionally converts archive format, increments download count, and streams bytes back to the client
- the web UI currently derives download links from `/api/v1/download/...`
- the CLI also hard-codes `/api/v1/download/...` and does not trust server-provided `download_url`

That means:

- package bandwidth still hits the application server
- switching object storage alone does not move download traffic away from Skill Home
- clients are partially coupled to the legacy download route shape

## 3. Design Decision

Adopt OSS as the public package origin and make `download_url` the canonical package URL for public skills.

### 3.1 Core decision

For public skills:

- the archive object lives in OSS
- API responses return an OSS URL in `download_url`
- clients should prefer that returned URL

For compatibility:

- `GET /api/v1/download/:namespace/:name/:version` stays in place
- for public skills whose requested format matches the stored archive format, the route responds with `302` to the same OSS object
- only legacy edge cases that still require server-side archive conversion continue streaming through Skill Home

For private or team-only skills:

- no new direct-download behavior is introduced in this phase
- future work can replace public direct URLs with signed OSS URLs or restore gateway behavior behind the same metadata model

### 3.2 Why this design

This design intentionally makes Skill Home a metadata and control plane, not the primary package transfer plane.

Compared with simply swapping MinIO for OSS behind the current download route, it actually removes public package traffic from the application server.

Compared with removing the legacy download route entirely, it avoids breaking already-installed CLI versions and older web assumptions during rollout.

## 4. Scope

### 4.1 In scope

- store public skill archives in OSS
- add configuration for public object base URLs
- return OSS `download_url` values from create and publish APIs
- teach the web UI to use API-provided `download_url`
- teach the CLI to prefer API-provided `download_url` and fall back to the legacy route
- keep the legacy download route as a compatibility redirect for public skills
- support phased migration of existing public skill archives to OSS

### 4.2 Out of scope

- private skill signed download URLs
- team-only authorization redesign
- CDN acceleration policy
- changing the metadata database model beyond what is needed to compute direct download URLs
- removing the legacy download route in this phase

## 5. Constraints and Assumptions

- Public packages may be openly downloadable for this first phase.
- Private and team-only packages will be handled in a later phase.
- OSS will expose a public download domain or custom domain suitable for direct package access.
- Clients must tolerate either absolute URLs or legacy relative download paths during rollout.
- Existing `storage_path` values remain the source of truth for locating package objects.

## 6. Architecture

### 6.1 Storage model

We keep the current object key model:

- `skills/<namespace>/<name>/<uuid>.<ext>`

Skill Home continues generating and storing `storage_path` values exactly as it does today. The change is not in object naming, but in how public download URLs are derived.

The storage configuration gains a public URL concept, for example:

- `storage.public_base_url`
- optionally `storage.download_strategy`

The object storage abstraction should be able to derive a public URL from a stored object key when the deployment is configured for direct public delivery.

### 6.2 API model

`download_url` becomes the canonical server contract for public package delivery.

Create and publish responses should return an absolute OSS URL for public skills instead of a relative `/api/v1/download/...` path.

Skill detail and list payloads should expose the same semantics consistently so both web and CLI clients can rely on one field.

### 6.3 Compatibility route

The legacy download route remains operational:

- if the skill is public and the requested output format matches the stored object format, respond with `302 Found` to the OSS URL
- if a legacy conversion path is still needed, continue to stream the converted archive through Skill Home
- if the skill is private in the future, the route remains the natural place to reintroduce authorization and signed URL generation

This route becomes a compatibility bridge instead of the primary download plane.

## 7. Client Behavior

### 7.1 Web UI

The web UI should stop deriving public download links by reconstructing `/api/v1/download/...`.

Instead:

- if API data provides `download_url`, use it directly
- only fall back to the legacy route if older API payloads omit the field

This keeps browser downloads aligned with the backend contract and avoids repeating URL policy in frontend code.

### 7.2 CLI

The CLI currently hard-codes `/api/v1/download/...` during installation and pull flows.

It should be updated to:

1. fetch skill metadata or version metadata that includes `download_url`
2. use that URL directly when present
3. fall back to `/api/v1/download/...` only for older servers that do not yet provide usable download URLs

This keeps new CLI versions aligned with the direct-download model while preserving compatibility with mixed environments.

### 7.3 Existing clients

Already-installed CLI builds and any older browser logic continue to work through the compatibility route because `/api/v1/download/...` remains valid.

## 8. Rollout Plan

### 8.1 Phase 1: Server support

- add public OSS URL configuration
- teach the storage layer or a helper to build public URLs from `storage_path`
- update create/publish responses to emit OSS `download_url`
- change the legacy download route to redirect for public direct-download cases

At this point:

- new publishes can already point to OSS
- old clients still work

### 8.2 Phase 2: Client adoption

- update web download helpers to trust `download_url`
- update CLI download logic to trust `download_url`
- release a new CLI version

At this point:

- most public package traffic should move off the app server

### 8.3 Phase 3: Historical backfill

- migrate historical public package objects to OSS if they are not already there
- verify that stored `storage_path` values map to reachable objects
- smoke-test representative public skills from web and CLI

### 8.4 Phase 4: Private skill follow-up

- introduce signed OSS URLs or gateway-controlled delivery for team-only skills
- decide whether `download_url` becomes time-limited for private objects or whether the compatibility route remains the only private download path

## 9. Error Handling

Expected behaviors:

- if OSS public URL configuration is missing, public skills should continue to use the legacy route instead of returning broken absolute URLs
- if a public object is missing from OSS, the legacy route should surface a clear download failure rather than redirecting to a dead object
- if a requested output format requires conversion, do not redirect blindly; use the existing conversion pipeline
- if a client receives an absolute `download_url`, it should not prepend the registry base URL

## 10. Testing Strategy

### 10.1 Server tests

- verify public create and publish responses return absolute OSS URLs
- verify legacy public download requests redirect to OSS when no conversion is needed
- verify conversion requests still return converted archive payloads
- verify behavior falls back safely when no public base URL is configured

### 10.2 Web tests

- verify skill overview downloads use API-provided `download_url`
- verify publish-success download links use returned `download_url`
- verify fallback logic still works against older payloads

### 10.3 CLI tests

- verify download logic uses absolute `download_url` when present
- verify fallback to `/api/v1/download/...` when metadata lacks `download_url`
- verify absolute OSS URLs are not incorrectly rewritten relative to the registry base

### 10.4 Operational verification

- publish a public test skill
- confirm API payloads return an OSS URL
- confirm browser download goes directly to OSS
- confirm a current CLI can install the skill using the new metadata path
- confirm an older CLI still succeeds through `/api/v1/download/...`

## 11. Code Areas Affected

Likely primary files:

- `skill-home-server/internal/config/config.go`
- `skill-home-server/internal/storage/object.go`
- `skill-home-server/internal/api/handlers/skill.go`
- `skill-home-server/internal/api/handlers/version.go`
- `skill-home-server/internal/api/handlers/handler_integration_test.go`
- `skill-home-web/src/api.ts`
- `skill-home-web/src/pages/PublishNewPage.tsx`
- `skill-home-web/src/pages/skill/SkillOverviewPage.tsx`
- `skill-home-web/src/hooks/useRegistryApp.test.tsx`
- `skill-home-cli/internal/registry/client.go`
- `skill-home-cli/internal/registry/types.go`
- `skill-home-cli/internal/registry/client_test.go`
- API and README documentation that describe download semantics

## 12. Trade-offs

Advantages:

- public package bandwidth moves to OSS
- web and CLI align on a single backend contract
- rollout is backward-compatible because the legacy route remains
- private skill support still has a clean path forward

Costs:

- direct URLs increase coupling to object-hosting configuration
- client code must learn to handle absolute download URLs
- later private skill work will need either signed URLs or a mixed-mode gateway

This trade is acceptable because the current objective is specifically to move public package delivery off the application server now, while preserving a safe migration path for private delivery later.
