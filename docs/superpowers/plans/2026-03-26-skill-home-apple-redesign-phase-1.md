# Skill Home Apple Redesign Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the first visible slice of the approved Apple-style redesign by rebuilding the global shell, home page, and skills workbench, while keeping existing routes functional and making catalog filters URL-backed.

**Architecture:** Keep the current single-entry React app, but introduce focused helpers for catalog query state so navigation and search can be shared cleanly between the top bar and the skills page. Rework layout and presentation in place for this phase instead of splitting every page immediately, so the UI changes ship fast without destabilizing detail, publish, and console flows.

**Tech Stack:** React 18, TypeScript, Vite, CSS, Vitest, jsdom, React Testing Library

---

### Task 1: Add front-end test harness and catalog-query helpers

**Files:**
- Modify: `skill-home-web/package.json`
- Modify: `skill-home-web/vite.config.ts`
- Create: `skill-home-web/src/lib/catalogState.ts`
- Create: `skill-home-web/src/lib/catalogState.test.ts`
- Create: `skill-home-web/src/test/setup.ts`

- [ ] **Step 1: Write the failing tests**

```ts
import { describe, expect, it } from 'vitest';

import { parseCatalogSearch, toCatalogSearch } from './catalogState';

describe('catalog search params', () => {
  it('reads known filters from the URL query string', () => {
    expect(
      parseCatalogSearch('?q=doc&namespace=testuser&sort=updated&view=cards'),
    ).toMatchObject({
      query: 'doc',
      namespace: 'testuser',
      sort: 'updated',
      view: 'cards',
    });
  });

  it('omits default values when serializing filters', () => {
    expect(
      toCatalogSearch({
        query: '',
        namespace: 'all',
        tag: 'all',
        license: 'all',
        sort: 'downloads',
        view: 'list',
      }),
    ).toBe('');
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- src/lib/catalogState.test.ts`
Expected: FAIL because Vitest and the helper module do not exist yet.

- [ ] **Step 3: Add minimal test tooling and helper implementation**

```ts
export function parseCatalogSearch(search: string) {
  return {
    query: '',
    namespace: 'all',
    tag: 'all',
    license: 'all',
    sort: 'downloads',
    view: 'list',
  };
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm run test -- src/lib/catalogState.test.ts`
Expected: PASS

### Task 2: Make catalog filters URL-backed in app state

**Files:**
- Modify: `skill-home-web/src/hooks/useRegistryApp.ts`
- Modify: `skill-home-web/src/App.tsx`
- Modify: `skill-home-web/src/lib/routes.ts`
- Test: `skill-home-web/src/lib/catalogState.test.ts`

- [ ] **Step 1: Write the failing test for query updates**

```ts
it('serializes non-default filters into the URL query string', () => {
  expect(
    toCatalogSearch({
      query: 'github',
      namespace: 'testuser',
      tag: 'all',
      license: 'MIT',
      sort: 'updated',
      view: 'list',
    }),
  ).toBe('?q=github&namespace=testuser&license=MIT&sort=updated');
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- src/lib/catalogState.test.ts`
Expected: FAIL because the serializer does not yet encode all supported filters.

- [ ] **Step 3: Implement minimal URL sync**

```ts
useEffect(() => {
  if (route.name !== 'skills') return;
  navigate(nextPath, { replace: true });
}, [catalogFilters, route.name, navigate]);
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm run test -- src/lib/catalogState.test.ts`
Expected: PASS

### Task 3: Rebuild the app shell, home page, and skills page

**Files:**
- Modify: `skill-home-web/src/App.tsx`
- Modify: `skill-home-web/src/styles.css`
- Modify: `skill-home-web/src/hooks/useRegistryApp.ts`

- [ ] **Step 1: Write the failing UI test for compact skills results**

```ts
it('keeps metadata in an inline summary instead of separate stat cards', () => {
  // render a result row and assert one meta line exists
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm run test -- src/lib/catalogState.test.ts`
Expected: FAIL or missing assertion target because the new presentation does not exist yet.

- [ ] **Step 3: Implement the minimal redesign**

```tsx
<header className="app-shell">
  <nav>{/* compact nav */}</nav>
  <form>{/* search */}</form>
</header>
```

- [ ] **Step 4: Run tests and build**

Run: `npm run test -- --run`
Expected: PASS

Run: `npm run build`
Expected: PASS

### Task 4: Verify the visible experience in a browser

**Files:**
- Modify: none

- [ ] **Step 1: Start preview**

Run: `npm run preview -- --host 127.0.0.1 --port 4173`
Expected: Vite preview starts successfully.

- [ ] **Step 2: Check desktop routes**

Run: Playwright open/snapshot for `/` and `/skills`
Expected: compact Apple-style shell, reduced border usage, home hero reduced to core content, skills list shows inline metadata with visible primary action.

- [ ] **Step 3: Check mobile route**

Run: Playwright resize to `390x844` and snapshot `/skills`
Expected: list rows remain compact and the main action is visible without long fact-card stacking.

- [ ] **Step 4: Record follow-up gaps**

Expected: leave detail page, publish page, console, and auth page for phase 2 unless a regression is introduced by phase 1 changes.
