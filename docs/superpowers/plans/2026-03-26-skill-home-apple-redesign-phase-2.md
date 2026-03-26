# Skill Home Apple Redesign Phase 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the Apple-style redesign to the skill detail page, install flow, and console/API Key management so the remaining core routes match the new shell and information-density rules.

**Architecture:** Keep the current single-file page composition for now, but rebuild the high-value route sections in place: detail summary, install guide, console split view, and API Key management blocks. Reuse the phase-1 visual tokens and keep state logic in existing hooks, only adding compact presentation-specific helpers when the data grouping needs to change.

**Tech Stack:** React 18, TypeScript, Vite, CSS, Vitest, jsdom, React Testing Library, Playwright CLI

---

### Task 1: Add failing UI tests for the remaining legacy sections

**Files:**
- Modify: `skill-home-web/src/App.test.tsx`

- [ ] **Step 1: Write the failing tests**

```tsx
it('removes the old install-step wall from the detail page', () => {
  expect(screen.queryByText('步骤 1：环境检查')).not.toBeInTheDocument();
});

it('replaces the old console copy-heavy cards with compact management sections', () => {
  expect(screen.queryByText('使用规则')).not.toBeInTheDocument();
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm run test -- src/App.test.tsx`
Expected: FAIL because the old detail/install/console content is still rendered.

### Task 2: Rebuild the skill detail summary and install guide

**Files:**
- Modify: `skill-home-web/src/App.tsx`
- Modify: `skill-home-web/src/styles.css`

- [ ] **Step 1: Implement the compact detail summary**

```tsx
<section className="detail-summary-shell">
  <div>{/* title, description, tags, inline facts */}</div>
  <aside>{/* sticky install actions */}</aside>
</section>
```

- [ ] **Step 2: Collapse the install guide**

```tsx
<section className="install-guide install-guide--compact">
  {/* ide tabs + one primary command + expandable supporting commands */}
</section>
```

- [ ] **Step 3: Run tests and build**

Run: `npm run test -- src/App.test.tsx`
Expected: PASS

Run: `npm run build`
Expected: PASS

### Task 3: Rebuild console/API Key management around compact list-first layout

**Files:**
- Modify: `skill-home-web/src/App.tsx`
- Modify: `skill-home-web/src/styles.css`

- [ ] **Step 1: Convert console shell to tighter split layout**

```tsx
<div className="console-layout console-layout--refined">
  <aside>{/* account + section nav + compact list */}</aside>
  <main>{/* focused editor or API key table */}</main>
</div>
```

- [ ] **Step 2: Simplify API Key management**

```tsx
<section className="api-key-panel api-key-panel--refined">
  {/* list first, create form second, simplified reveal */}
</section>
```

- [ ] **Step 3: Run tests and build**

Run: `npm run test -- --run`
Expected: PASS

Run: `npm run build`
Expected: PASS

### Task 4: Browser verification

**Files:**
- Modify: none

- [ ] **Step 1: Start preview**

Run: `npm run preview -- --host 127.0.0.1 --port 4173`
Expected: Vite preview starts.

- [ ] **Step 2: Check desktop routes**

Run: Playwright screenshots for `/skills/testuser/skill-home-manager` and `/console/api-keys`
Expected: detail hero no longer has large dead space; install section is shorter and more focused; console/API Key layout is compact and visually aligned with phase 1.

- [ ] **Step 3: Check mobile route**

Run: Playwright resize to `390x844` and screenshot `/skills/testuser/skill-home-manager`
Expected: detail actions stay visible early and install guide does not explode into a long step wall.
