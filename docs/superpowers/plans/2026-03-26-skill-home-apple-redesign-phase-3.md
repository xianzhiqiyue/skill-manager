# Skill Home Apple Redesign Phase 3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the Apple-style redesign to the publish flow, auth pages, and skill-management console so the remaining high-traffic routes match the new compact visual system and reduce repeated guidance.

**Architecture:** Keep the current React app structure and route model, but replace the remaining copy-heavy sections in place. Reuse the phase-1 and phase-2 surface patterns, collapse repeated sidebars into tighter action groups, and keep existing state and mutations untouched unless presentation changes require a small structural wrapper.

**Tech Stack:** React 18, TypeScript, Vite, CSS, Vitest, jsdom, React Testing Library, Playwright CLI

---

### Task 1: Add failing UI tests for publish, auth, and console-skills polish

**Files:**
- Modify: `skill-home-web/src/App.test.tsx`

- [ ] **Step 1: Write the failing tests**

```tsx
it('uses a compact publish checklist instead of the old process wall', () => {
  expect(screen.queryByText('推荐流程')).not.toBeInTheDocument();
  expect(screen.getByText('发布清单')).toBeInTheDocument();
});

it('uses a compact auth card instead of the old marketing split layout', () => {
  expect(screen.getByRole('heading', { name: '登录 Skill Home' })).toBeInTheDocument();
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npm run test -- src/App.test.tsx`
Expected: FAIL because publish/auth/console still render the older verbose sections.

### Task 2: Rebuild publish and auth routes around denser action-first layouts

**Files:**
- Modify: `skill-home-web/src/App.tsx`
- Modify: `skill-home-web/src/styles.css`

- [ ] **Step 1: Simplify the publish route**

```tsx
<div className="publish-layout">
  <section>{/* form */}</section>
  <aside className="publish-checklist">{/* 3-item checklist */}</aside>
</div>
```

- [ ] **Step 2: Compress the auth route**

```tsx
<section className="surface-panel auth-shell auth-shell--compact">
  <div className="auth-shell__card">{/* compact intro + form */}</div>
</section>
```

- [ ] **Step 3: Run focused tests**

Run: `npm run test -- src/App.test.tsx`
Expected: PASS

### Task 3: Refine the console skill editor into grouped management blocks

**Files:**
- Modify: `skill-home-web/src/App.tsx`
- Modify: `skill-home-web/src/styles.css`

- [ ] **Step 1: Replace the old top-heavy editor header**

```tsx
<form className="form-grid-stack console-form-stack">
  <section className="console-section-block">{/* metadata */}</section>
  <section className="console-section-block">{/* visibility */}</section>
  <section className="console-section-block">{/* versions */}</section>
</form>
```

- [ ] **Step 2: Isolate destructive actions**

```tsx
<section className="console-section-block console-section-block--danger">
  {/* delete skill */}
</section>
```

- [ ] **Step 3: Run full tests and build**

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

- [ ] **Step 2: Check publish and auth routes**

Run: Playwright screenshots for `/publish` and `/login`
Expected: publish route uses a compact checklist with reduced border noise; auth route is centered, concise, and no longer repeats the homepage product pitch.

- [ ] **Step 3: Record residual gaps**

Expected: authenticated `/console/skills` visual verification remains dependent on a live signed-in session; rely on component tests unless preview auth is available.
