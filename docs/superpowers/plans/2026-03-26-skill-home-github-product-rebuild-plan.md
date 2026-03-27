# Skill Home GitHub-Style Product Rebuild Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild Skill Home into a GitHub-style product UI where skills behave like first-class objects with search workspaces, object tabs, and settings architecture instead of page-local card layouts.

**Architecture:** Keep the current React + Vite app, but stop extending the single-file page composition in [App.tsx](/home/zhuyue/code/skill-manager/skill-home-web/src/App.tsx). First introduce route-aware layouts and reusable GitHub-style primitives, then migrate search, object pages, settings, publish, and home in separate phases while preserving legacy URLs through redirects. Keep the existing API surface for now; this is a front-end information-architecture rebuild, not a back-end feature rewrite.

**Tech Stack:** React 18, TypeScript, Vite, CSS, Vitest, React Testing Library, Playwright CLI

---

## File Structure Map

### Existing files to keep and refactor

- Modify: `skill-home-web/src/App.tsx`
  - Reduce to top-level composition only.
- Modify: `skill-home-web/src/styles.css`
  - Replace Apple-style tokens and page-local card rules with GitHub-style tokens and structural primitives.
- Modify: `skill-home-web/src/api.ts`
  - Add typed helpers required by the new routes and object views without changing server contracts.
- Modify: `skill-home-web/src/hooks/useRegistryApp.ts`
  - Keep state source of truth, but split route-specific selectors and actions into smaller helpers.
- Modify: `skill-home-web/src/hooks/useRoute.ts`
  - Support deeper object routes and redirects.
- Modify: `skill-home-web/src/lib/routes.ts`
  - Expand route model to object pages and settings pages.
- Modify: `skill-home-web/src/App.test.tsx`
  - Shift tests from card-copy assertions to route structure, tabs, settings layout, and list scanning behavior.

### New files to create

- Create: `skill-home-web/src/components/layout/GlobalHeader.tsx`
- Create: `skill-home-web/src/components/layout/PageHeader.tsx`
- Create: `skill-home-web/src/components/layout/SidebarLayout.tsx`
- Create: `skill-home-web/src/components/object/SkillHeader.tsx`
- Create: `skill-home-web/src/components/object/SkillTabs.tsx`
- Create: `skill-home-web/src/components/search/FilterRail.tsx`
- Create: `skill-home-web/src/components/search/SkillResultList.tsx`
- Create: `skill-home-web/src/components/settings/SettingsLayout.tsx`
- Create: `skill-home-web/src/components/settings/SettingsNav.tsx`
- Create: `skill-home-web/src/components/settings/DangerZone.tsx`
- Create: `skill-home-web/src/pages/HomePage.tsx`
- Create: `skill-home-web/src/pages/SkillsSearchPage.tsx`
- Create: `skill-home-web/src/pages/skill/SkillOverviewPage.tsx`
- Create: `skill-home-web/src/pages/skill/SkillVersionsPage.tsx`
- Create: `skill-home-web/src/pages/skill/SkillInstallPage.tsx`
- Create: `skill-home-web/src/pages/skill/SkillActivityPage.tsx`
- Create: `skill-home-web/src/pages/settings/ProfileSettingsPage.tsx`
- Create: `skill-home-web/src/pages/settings/APIKeysSettingsPage.tsx`
- Create: `skill-home-web/src/pages/settings/SkillGeneralSettingsPage.tsx`
- Create: `skill-home-web/src/pages/settings/SkillVersionsSettingsPage.tsx`
- Create: `skill-home-web/src/pages/settings/SkillAccessSettingsPage.tsx`
- Create: `skill-home-web/src/pages/settings/SkillDangerSettingsPage.tsx`
- Create: `skill-home-web/src/pages/PublishNewPage.tsx`
- Create: `skill-home-web/src/pages/AuthPage.tsx`
- Create: `skill-home-web/src/pages/InstallDocsPage.tsx`
- Create: `skill-home-web/src/lib/settings.ts`
- Create: `skill-home-web/src/lib/skillTabs.ts`
- Create: `skill-home-web/src/lib/routes.test.ts`

### Test files to add or expand

- Modify: `skill-home-web/src/App.test.tsx`
- Modify: `skill-home-web/src/hooks/useRoute.test.tsx`
- Modify: `skill-home-web/src/hooks/useRegistryApp.test.tsx`
- Create: `skill-home-web/src/components/search/SkillResultList.test.tsx`
- Create: `skill-home-web/src/components/settings/SettingsLayout.test.tsx`
- Create: `skill-home-web/src/pages/skill/SkillOverviewPage.test.tsx`

---

### Task 1: Expand the route model and flatten `App.tsx` into page composition only

**Files:**
- Modify: `skill-home-web/src/lib/routes.ts`
- Modify: `skill-home-web/src/hooks/useRoute.ts`
- Modify: `skill-home-web/src/App.tsx`
- Create: `skill-home-web/src/lib/routes.test.ts`
- Modify: `skill-home-web/src/hooks/useRoute.test.tsx`

- [ ] **Step 1: Write the failing route tests**

```ts
import { describe, expect, it } from 'vitest';

import { parseRoute } from './routes';

describe('GitHub-style routes', () => {
  it('parses skill object tabs', () => {
    expect(parseRoute('/skills/testuser/github/install')).toEqual({
      name: 'skill-tab',
      namespace: 'testuser',
      skillName: 'github',
      tab: 'install',
    });
  });

  it('parses settings routes', () => {
    expect(parseRoute('/settings/skills/testuser/github/danger')).toEqual({
      name: 'skill-settings',
      namespace: 'testuser',
      skillName: 'github',
      section: 'danger',
    });
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm run test -- src/lib/routes.test.ts src/hooks/useRoute.test.tsx`
Expected: FAIL because `parseRoute` only knows the flat route model.

- [ ] **Step 3: Implement the new route model minimally**

```ts
export type AppRoute =
  | { name: 'home' }
  | { name: 'skills' }
  | { name: 'skill-tab'; namespace: string; skillName: string; tab: 'overview' | 'versions' | 'install' | 'activity' }
  | { name: 'settings'; section: 'profile' | 'api-keys' }
  | { name: 'skill-settings'; namespace: string; skillName: string; section: 'general' | 'versions' | 'access' | 'danger' }
  | { name: 'publish-new' }
  | { name: 'install' }
  | { name: 'auth'; mode: 'login' | 'register' };
```

- [ ] **Step 4: Collapse `App.tsx` into page routing only**

```tsx
<main className="app-main">
  {route.name === 'skills' ? <SkillsSearchPage model={model} /> : null}
  {route.name === 'skill-tab' ? <SkillOverviewPage model={model} route={route} /> : null}
</main>
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `npm run test -- src/lib/routes.test.ts src/hooks/useRoute.test.tsx`
Expected: PASS

### Task 2: Introduce GitHub-style shell primitives and token system

**Files:**
- Create: `skill-home-web/src/components/layout/GlobalHeader.tsx`
- Create: `skill-home-web/src/components/layout/PageHeader.tsx`
- Create: `skill-home-web/src/components/layout/SidebarLayout.tsx`
- Modify: `skill-home-web/src/App.tsx`
- Modify: `skill-home-web/src/styles.css`
- Modify: `skill-home-web/src/App.test.tsx`

- [ ] **Step 1: Write the failing shell test**

```tsx
it('renders a dark global header and no longer uses the old soft hero shell', () => {
  render(<App />);
  expect(screen.getByRole('banner')).toHaveClass('gh-header');
  expect(screen.queryByText('把 skill 变成能被搜索、安装和持续运营的产品入口')).not.toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- src/App.test.tsx`
Expected: FAIL because the current header and hero still use the Apple-style shell.

- [ ] **Step 3: Add shell primitives and token classes**

```tsx
export function GlobalHeader() {
  return (
    <header className="gh-header">
      <div className="gh-header__brand">Skill Home</div>
      <nav className="gh-header__nav">{/* links */}</nav>
      <form className="gh-header__search">{/* search */}</form>
    </header>
  );
}
```

```css
:root {
  --gh-canvas: #f6f8fa;
  --gh-surface: #ffffff;
  --gh-border: #d0d7de;
  --gh-text: #1f2328;
  --gh-muted: #57606a;
  --gh-link: #0969da;
  --gh-success: #1f883d;
  --gh-danger: #cf222e;
  --gh-header: #24292f;
}
```

- [ ] **Step 4: Run the focused UI test**

Run: `npm run test -- src/App.test.tsx`
Expected: PASS

### Task 3: Rebuild `/skills` as a GitHub-style search workspace

**Files:**
- Create: `skill-home-web/src/components/search/FilterRail.tsx`
- Create: `skill-home-web/src/components/search/SkillResultList.tsx`
- Create: `skill-home-web/src/components/search/SkillResultList.test.tsx`
- Create: `skill-home-web/src/pages/SkillsSearchPage.tsx`
- Modify: `skill-home-web/src/hooks/useRegistryApp.ts`
- Modify: `skill-home-web/src/styles.css`
- Modify: `skill-home-web/src/App.test.tsx`

- [ ] **Step 1: Write the failing result-list test**

```tsx
it('shows a left filter rail and compact result rows', () => {
  render(<SkillsSearchPage model={model} />);
  expect(screen.getByText('Filter by')).toBeInTheDocument();
  expect(screen.getByText('结果')).toBeInTheDocument();
  expect(screen.queryByText('查看详情')).not.toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- src/components/search/SkillResultList.test.tsx src/App.test.tsx`
Expected: FAIL because the current skills page is not a GitHub-style search workbench.

- [ ] **Step 3: Implement compact result rows and filter rail**

```tsx
<SidebarLayout
  sidebar={<FilterRail filters={model.catalogFilters} onChange={model.updateCatalogFilter} />}
  content={<SkillResultList skills={model.skills} total={model.skillsTotal} />}
  aside={<SearchContextCard />}
/>
```

- [ ] **Step 4: Keep URL-backed filtering intact**

Run: `npm run test -- src/lib/catalogState.test.ts src/components/search/SkillResultList.test.tsx`
Expected: PASS

### Task 4: Rebuild skill detail into object header + tabs + sidebar

**Files:**
- Create: `skill-home-web/src/components/object/SkillHeader.tsx`
- Create: `skill-home-web/src/components/object/SkillTabs.tsx`
- Create: `skill-home-web/src/pages/skill/SkillOverviewPage.tsx`
- Create: `skill-home-web/src/pages/skill/SkillOverviewPage.test.tsx`
- Create: `skill-home-web/src/pages/skill/SkillVersionsPage.tsx`
- Create: `skill-home-web/src/pages/skill/SkillInstallPage.tsx`
- Create: `skill-home-web/src/pages/skill/SkillActivityPage.tsx`
- Create: `skill-home-web/src/lib/skillTabs.ts`
- Modify: `skill-home-web/src/hooks/useRegistryApp.ts`
- Modify: `skill-home-web/src/styles.css`
- Modify: `skill-home-web/src/App.test.tsx`

- [ ] **Step 1: Write the failing object-page test**

```tsx
it('renders a skill object header with tabs and sidebar metadata', () => {
  render(<SkillOverviewPage model={model} route={route} />);
  expect(screen.getByText('testuser / github')).toBeInTheDocument();
  expect(screen.getByRole('tab', { name: 'Overview' })).toBeInTheDocument();
  expect(screen.getByRole('tab', { name: 'Install' })).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- src/pages/skill/SkillOverviewPage.test.tsx`
Expected: FAIL because the current detail view is still a custom card composition.

- [ ] **Step 3: Implement object header and tabs**

```tsx
<SkillHeader skill={model.detailSkill} />
<SkillTabs activeTab={route.tab} namespace={route.namespace} skillName={route.skillName} />
<SidebarLayout content={<OverviewPanel skill={model.detailSkill} />} aside={<MetadataSidebar skill={model.detailSkill} />} />
```

- [ ] **Step 4: Implement install and versions pages against the same object shell**

Run: `npm run test -- src/pages/skill/SkillOverviewPage.test.tsx src/App.test.tsx`
Expected: PASS

### Task 5: Introduce `/settings/*` and migrate API Keys plus skill management into settings architecture

**Files:**
- Create: `skill-home-web/src/components/settings/SettingsLayout.tsx`
- Create: `skill-home-web/src/components/settings/SettingsNav.tsx`
- Create: `skill-home-web/src/components/settings/DangerZone.tsx`
- Create: `skill-home-web/src/components/settings/SettingsLayout.test.tsx`
- Create: `skill-home-web/src/pages/settings/ProfileSettingsPage.tsx`
- Create: `skill-home-web/src/pages/settings/APIKeysSettingsPage.tsx`
- Create: `skill-home-web/src/pages/settings/SkillGeneralSettingsPage.tsx`
- Create: `skill-home-web/src/pages/settings/SkillVersionsSettingsPage.tsx`
- Create: `skill-home-web/src/pages/settings/SkillAccessSettingsPage.tsx`
- Create: `skill-home-web/src/pages/settings/SkillDangerSettingsPage.tsx`
- Create: `skill-home-web/src/lib/settings.ts`
- Modify: `skill-home-web/src/hooks/useRegistryApp.ts`
- Modify: `skill-home-web/src/styles.css`
- Modify: `skill-home-web/src/App.test.tsx`

- [ ] **Step 1: Write the failing settings-layout test**

```tsx
it('renders a settings nav and isolates danger actions in a dedicated section', () => {
  render(<SettingsLayout nav={<SettingsNav />} content={<SkillDangerSettingsPage model={model} />} />);
  expect(screen.getByText('General')).toBeInTheDocument();
  expect(screen.getByText('Danger Zone')).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- src/components/settings/SettingsLayout.test.tsx src/App.test.tsx`
Expected: FAIL because the current console still mixes skills and API keys inside one dashboard shell.

- [ ] **Step 3: Implement user settings and skill settings pages**

```tsx
<SettingsLayout
  nav={<SettingsNav items={items} activeKey={section} />}
  content={<APIKeysSettingsPage model={model} />}
/>
```

- [ ] **Step 4: Move current API key UI into `/settings/api-keys`**

Run: `npm run test -- src/components/settings/SettingsLayout.test.tsx src/App.test.tsx`
Expected: PASS

### Task 6: Rebuild publish, auth, and install docs around GitHub-style creation/settings flows

**Files:**
- Create: `skill-home-web/src/pages/PublishNewPage.tsx`
- Create: `skill-home-web/src/pages/AuthPage.tsx`
- Create: `skill-home-web/src/pages/InstallDocsPage.tsx`
- Modify: `skill-home-web/src/styles.css`
- Modify: `skill-home-web/src/App.tsx`
- Modify: `skill-home-web/src/App.test.tsx`

- [ ] **Step 1: Write the failing publish/auth tests**

```tsx
it('renders publish as a focused creation form under /publish/new', () => {
  render(<PublishNewPage model={model} />);
  expect(screen.getByRole('heading', { name: 'Create a new release' })).toBeInTheDocument();
});

it('renders auth inside the GitHub-style shell instead of the old compact marketing card', () => {
  render(<AuthPage model={model} mode="login" navigate={vi.fn()} />);
  expect(screen.getByText('Sign in to Skill Home')).toBeInTheDocument();
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm run test -- src/App.test.tsx`
Expected: FAIL because the current publish/auth copy and layout still follow the prior redesign.

- [ ] **Step 3: Implement the new pages**

```tsx
<SidebarLayout
  content={<PublishForm model={model} />}
  aside={<ChecklistPanel items={publishChecklist} />}
/>
```

- [ ] **Step 4: Run focused tests**

Run: `npm run test -- src/App.test.tsx`
Expected: PASS

### Task 7: Rebuild the home page so it matches the new product shell without becoming the main work surface

**Files:**
- Create: `skill-home-web/src/pages/HomePage.tsx`
- Modify: `skill-home-web/src/styles.css`
- Modify: `skill-home-web/src/App.test.tsx`

- [ ] **Step 1: Write the failing home-page test**

```tsx
it('keeps the home page concise and routes users into search instead of repeating long explainer sections', () => {
  render(<HomePage model={model} navigate={vi.fn()} />);
  expect(screen.getByRole('heading', { name: 'Find and ship skills faster' })).toBeInTheDocument();
  expect(screen.queryByText('精选技能')).not.toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- src/App.test.tsx`
Expected: FAIL because the existing home page still uses the earlier product-entry structure.

- [ ] **Step 3: Implement the concise GitHub-inspired home page**

```tsx
<PageHeader
  title="Find and ship skills faster"
  description="Search the registry, inspect a skill, then install or publish from one consistent workspace."
/>
```

- [ ] **Step 4: Run focused tests**

Run: `npm run test -- src/App.test.tsx`
Expected: PASS

### Task 8: Add redirects, cleanup, and full verification

**Files:**
- Modify: `skill-home-web/src/lib/routes.ts`
- Modify: `skill-home-web/src/App.tsx`
- Modify: `skill-home-web/src/styles.css`
- Modify: `skill-home-web/src/App.test.tsx`

- [ ] **Step 1: Add redirect coverage**

```tsx
it('redirects legacy console api keys route into settings api keys', () => {
  // render with /console/api-keys and assert navigation target
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- src/App.test.tsx src/lib/routes.test.ts`
Expected: FAIL because legacy aliases are not redirected yet.

- [ ] **Step 3: Implement redirects and remove obsolete styles**

```ts
if (segments[0] === 'console' && segments[1] === 'api-keys') {
  return { name: 'settings', section: 'api-keys' };
}
```

- [ ] **Step 4: Run the full suite**

Run: `npm run test -- --run`
Expected: PASS

Run: `npm run build`
Expected: PASS

- [ ] **Step 5: Browser verification**

Run: `npm run preview -- --host 127.0.0.1 --port 4173`
Expected: preview starts

Run: Playwright screenshots for:
- `/skills`
- `/skills/testuser/github`
- `/skills/testuser/github/install`
- `/settings/api-keys`
- `/settings/skills/testuser/github/general`
- `/publish/new`

Expected:
- search page uses left filter rail and compact result list
- detail page uses object header + tabs + metadata sidebar
- settings pages use left nav + focused right pane
- API key management no longer reads like a dashboard fragment
- publish page reads as a creation workflow, not a console page

---

Plan complete and saved to `docs/superpowers/plans/2026-03-26-skill-home-github-product-rebuild-plan.md`.

Two execution options:

1. Subagent-Driven (recommended) - dispatch a fresh subagent per task, review between tasks, fast iteration
2. Inline Execution - execute tasks in this session using executing-plans, batch execution with checkpoints
