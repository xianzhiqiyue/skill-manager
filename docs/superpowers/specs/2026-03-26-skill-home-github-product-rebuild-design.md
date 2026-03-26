# Skill Home GitHub-Style Product Rebuild Design

## 1. Goal

Replace the current "marketing shell + page-by-page card layouts" approach with a GitHub-style product interface system. This is a structural redesign, not a surface reskin.

The target outcome is:

- `Skill` becomes the primary product object, similar to how GitHub treats a `Repository`.
- Search, detail, installation, version history, and management all become part of one consistent object model.
- Settings and destructive actions move into a true settings architecture instead of being mixed into generic dashboard panels.
- The visual system shifts from soft Apple-like surfaces to GitHub-like neutral product UI: dark global chrome, light working surfaces, tighter radii, stronger separators, higher scanning efficiency.

## 2. Research Basis

The redesign direction is based on these public GitHub pages:

- [GitHub home](https://github.com/)
- [GitHub repository page](https://github.com/openai/openai-python)
- [GitHub repository pull requests list](https://github.com/openai/openai-python/pulls)
- [GitHub repository search](https://github.com/search?q=react&type=repositories)
- [GitHub Docs: repository settings](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/setting-repository-visibility)
- [GitHub Docs: managing Actions settings](https://docs.github.com/en/enterprise-cloud@latest/repositories/managing-your-repositorys-settings-and-features/enabling-features-for-your-repository/managing-github-actions-settings-for-a-repository)

Important note:

- The repository and search layouts are directly observed from public pages.
- The exact authenticated `Settings` layout is inferred from GitHub Docs structure and long-standing GitHub product patterns, because the full live settings UI is not publicly accessible without a signed-in session.

## 3. What GitHub Actually Does

### 3.1 GitHub is not one layout

GitHub uses different layout systems for different jobs:

- Home page: narrative, marketing, conversion-focused.
- Search page: filter rail + result workspace.
- Repository page: object header + tab navigation + primary content + metadata sidebar.
- Settings page: left settings navigation + focused editor pane + danger zone at the bottom of the relevant page.

The mistake would be to copy only the GitHub home page look. Skill Home needs mostly the repository/search/settings systems.

### 3.2 Structural rules visible across GitHub product pages

- One dark global top bar anchors the whole product.
- Every important object has an identity header.
- Secondary navigation is done with tabs, not with repeated page-level cards.
- Lists are optimized for scanning, not for decoration.
- Metadata is compressed into sidebars, labels, counters, and short inline rows.
- Settings are isolated from browsing and reading workflows.
- Destructive actions are grouped into a dedicated danger section instead of being mixed into normal form flow.

### 3.3 GitHub color and spacing behavior

GitHub product UI is not visually loud:

- neutral light canvas
- white panels
- visible but restrained borders
- tight radii
- muted secondary text
- blue links and active indicators
- green reserved for success and creation flows

This matters because Skill Home currently spends too much space on decorative grouping and too little on hierarchy.

## 4. Product Model For Skill Home

### 4.1 Core model

Treat a `Skill` the way GitHub treats a `Repository`.

Each skill has:

- identity: `namespace/name`
- summary: description, tags, license
- activity: updated time, versions, downloads
- documentation: README or install guidance
- operational state: public/private, deprecated, scan status
- management area: general settings, version controls, API access, danger actions

### 4.2 Consequence of this model

This means:

- skill browsing and skill management should no longer feel like separate products
- the detail page should stop being a loose “marketing detail” view
- API Keys should stop living as an isolated dashboard card system
- settings should become a first-class navigation mode

## 5. Information Architecture

### 5.1 Route system

Current routes are too flat for a GitHub-style product. The new route model should be:

- `/`
  - home entry and product framing only
- `/skills`
  - search workspace
- `/skills/:namespace/:name`
  - skill overview
- `/skills/:namespace/:name/versions`
  - version list and version-specific state
- `/skills/:namespace/:name/install`
  - installation and usage
- `/skills/:namespace/:name/activity`
  - change history and recent version actions
- `/settings/profile`
  - account profile and user-level settings
- `/settings/api-keys`
  - account API key management
- `/settings/skills/:namespace/:name/general`
  - skill general settings
- `/settings/skills/:namespace/:name/versions`
  - skill version management
- `/settings/skills/:namespace/:name/access`
  - visibility and publishing controls
- `/settings/skills/:namespace/:name/danger`
  - destructive actions
- `/publish/new`
  - create new skill release
- `/login`
- `/register`

### 5.2 Backward compatibility

The existing routes should remain as aliases during migration:

- `/console/skills` redirects to the appropriate settings or owned-skill index
- `/console/api-keys` redirects to `/settings/api-keys`
- `/publish` redirects to `/publish/new`

## 6. Page Model Mapping

### 6.1 Home page

GitHub pattern borrowed:

- dark top bar
- strong hero
- concise framing
- clear primary CTA

What Skill Home should do:

- keep the home page as product entry, not as the main working surface
- present one primary action: search skills
- support a secondary action: publish a new skill
- show only enough proof to establish trust: featured skills, recent updates, install flow summary

What not to do:

- do not turn every core workflow into a giant marketing section
- do not copy GitHub’s long storytelling homepage blocks into the product

### 6.2 Skills search page

GitHub pattern borrowed:

- left filter rail
- result count and sorting strip
- compact result list
- optional right-side context card

Skill Home adaptation:

- left rail:
  - namespace
  - tags
  - license
  - visibility
  - scan/deprecation state
  - sort
- main result column:
  - each skill row shows name, namespace, description, tags, license, last updated, latest version, downloads
  - primary action is opening the skill
- optional right column:
  - query tips
  - selected filter summary
  - highlighted featured skill when no query is active

This should replace the current large-panel catalog feel with a true search workspace.

### 6.3 Skill overview page

GitHub pattern borrowed:

- object identity row
- object tabs
- main content column
- right metadata sidebar

Skill Home adaptation:

- identity header:
  - `namespace / skill-name`
  - visibility badge
  - latest version
  - follow-up action buttons like install, copy reference, publish update if owner
- tabs:
  - `Overview`
  - `Versions`
  - `Install`
  - `Activity`
- main column:
  - description
  - README / install narrative
  - version highlights
- right sidebar:
  - tags
  - license
  - updated time
  - downloads
  - scan status
  - latest release summary

The key change is that installation is no longer the whole identity of the page. It becomes one tab inside a broader object surface.

### 6.4 Install page/tab

GitHub pattern borrowed:

- focused content section with copy-friendly controls
- supporting metadata nearby, not duplicated

Skill Home adaptation:

- top block:
  - primary install command
  - IDE-specific variants in tabs or segmented controls
- below:
  - prerequisites
  - verification command
  - update/uninstall commands
- right sidebar or inset note:
  - latest version
  - package size
  - scan state

The install area must stop repeating the same explanations in multiple cards.

### 6.5 Versions page/tab

GitHub pattern borrowed:

- list or table of object history
- dense scanning
- lightweight actions

Skill Home adaptation:

- version rows show:
  - version number
  - created time
  - size
  - scan state
  - changelog note if available
  - download / install / delete actions depending on permissions

This replaces the current generic list sections with a true release history surface.

### 6.6 Settings pages

GitHub pattern borrowed:

- left settings navigation
- focused right pane
- one task per page
- danger zone isolated at the bottom of the correct page

Skill Home adaptation:

- user settings:
  - profile
  - API keys
- skill settings:
  - general
  - access
  - versions
  - danger

Each settings page should have:

- page title
- short context sentence
- one or more grouped form sections
- save button scoped to the current page

### 6.7 API key settings page

GitHub pattern borrowed:

- settings-like form and list layout
- scanning-first rows
- actions at row edge

Skill Home adaptation:

- header:
  - page title
  - short explanation
  - refresh action
- creation form at top or top-right
- existing key list below as a table-like list
- metadata compressed into stable columns:
  - name
  - prefix
  - time info
  - status
  - action

This should live under `/settings/api-keys`, not as a detached console mode.

### 6.8 Publish page

GitHub pattern borrowed:

- object creation / release creation form
- focused editor with supporting right rail

Skill Home adaptation:

- large working form in main column
- right rail checklist:
  - validate package
  - confirm version
  - check namespace target
  - post-publish verification

Do not let publish become a dashboard page. It is a creation workflow.

## 7. Visual System

### 7.1 Color tokens

Adopt GitHub-like neutrals:

- canvas: `#f6f8fa`
- surface: `#ffffff`
- muted surface: `#f6f8fa`
- border: `#d0d7de`
- subtle border: `#d8dee4`
- text: `#1f2328`
- secondary text: `#57606a`
- link / active: `#0969da`
- success / create: `#1f883d`
- danger: `#cf222e`
- dark top bar: `#24292f`

Brand note:

- keep Skill Home brand green only where GitHub would use a create/success color
- do not keep the current app-wide green tint as the main visual identity

### 7.2 Typography

Use a GitHub-like system stack:

- `-apple-system`
- `BlinkMacSystemFont`
- `"Segoe UI"`
- `"Noto Sans"`
- `"Helvetica Neue"`
- `Arial`
- `sans-serif`

Typography rules:

- tighter titles
- smaller supporting text
- fewer oversized hero headings outside home
- tabular figures for counts and times

### 7.3 Radius and shadows

Use tighter radii than the Apple redesign:

- small: `6`
- medium: `8`
- large: `12`

Shadows should be minimal. GitHub relies more on border structure than on floating cards.

### 7.4 Surface behavior

- major sections may still use cards, but cards must behave like containers, not decorative tiles
- list rows should be bordered or separated, not boxed within additional inner cards
- sidebars should feel like structured metadata rails, not independent landing-page panels

## 8. Component System Changes

The current component approach is too page-specific. The redesign should introduce reusable GitHub-style primitives:

- `GlobalHeader`
- `PageHeader`
- `ObjectHeader`
- `SubnavTabs`
- `FilterRail`
- `ResultList`
- `MetadataSidebar`
- `SettingsLayout`
- `SettingsNav`
- `FormSection`
- `DangerZone`
- `InlineStat`
- `StateBadge`
- `KeyValueMetaList`

These primitives should replace the current repeated page-local combinations of:

- `surface-panel`
- large card wrappers
- eyebrow + title + paragraph triplets
- metric card grids used as generic structure

## 9. Current Codebase Implications

The current app is still organized as one large page-composition file in [App.tsx](/home/zhuyue/code/skill-manager/skill-home-web/src/App.tsx), with routes defined in [routes.ts](/home/zhuyue/code/skill-manager/skill-home-web/src/lib/routes.ts).

That structure is no longer a good fit for this redesign. The rebuild should include:

- route model expansion
- component extraction by page model, not by visual fragments only
- separation between browsing views and settings views
- migration from `console` as a generic umbrella to `settings` plus object pages

## 10. Migration Strategy

### Phase 1

- introduce GitHub-style tokens
- rebuild global header
- create new layout primitives

### Phase 2

- rebuild `/skills` as search workspace
- convert result rows to compact GitHub-like scanning layout

### Phase 3

- rebuild skill overview route with object header, tabs, and sidebar
- move install into a tabbed object page

### Phase 4

- build `/settings/*`
- migrate API Keys and skill management into settings architecture

### Phase 5

- rebuild publish flow as a focused creation form
- rebuild home page to match the new product shell without turning it into a pure marketing site

### Phase 6

- add route redirects from legacy `/console/*` and `/publish`
- remove obsolete page shells and duplicated card styles

## 11. Non-Goals

- Do not clone GitHub branding literally.
- Do not copy GitHub homepage storytelling structure into every page.
- Do not keep the current route model and pretend a token change is sufficient.
- Do not optimize for “pretty screenshot” at the expense of dense working surfaces.

## 12. Acceptance Criteria

The redesign is successful when:

- users can recognize each skill as a single product object with overview, versions, install, activity, and settings surfaces
- search feels like a real workbench, not a stack of cards
- API Keys and skill management feel like settings pages, not dashboard fragments
- destructive actions are isolated in danger zones
- the interface visually reads as GitHub-like product UI rather than Apple-style soft panels
- page-to-page structure feels consistent enough that users can predict where information and actions live

