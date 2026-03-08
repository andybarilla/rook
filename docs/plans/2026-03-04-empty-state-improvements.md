# Empty State Improvements — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace plain-text empty states in Sites and Services with a shared visual component featuring icons, guidance text, and call-to-action buttons.

**Architecture:** New reusable `EmptyState.svelte` component in `frontend/src/lib/`, consumed by `SiteList.svelte` and `ServiceList.svelte`. The Sites CTA opens the AddSiteForm collapse via a bindable prop. The Services CTA opens a docs URL.

**Tech Stack:** Svelte 3, DaisyUI 5, Tailwind CSS 4, Vitest + @testing-library/svelte

---

### Task 1: Create EmptyState component with tests

**Files:**
- Create: `frontend/src/lib/EmptyState.svelte`
- Create: `frontend/src/lib/EmptyState.test.js`

**Step 1: Write the failing tests**

Create `frontend/src/lib/EmptyState.test.js`:

```js
import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import EmptyState from './EmptyState.svelte';

describe('EmptyState', () => {
  it('renders the message', () => {
    const { getByText } = render(EmptyState, {
      props: { message: 'No items yet' },
    });
    expect(getByText('No items yet')).toBeTruthy();
  });

  it('renders the subtitle when provided', () => {
    const { getByText } = render(EmptyState, {
      props: { message: 'No items yet', subtitle: 'Add one to get started.' },
    });
    expect(getByText('Add one to get started.')).toBeTruthy();
  });

  it('does not render subtitle element when not provided', () => {
    const { container } = render(EmptyState, {
      props: { message: 'No items yet' },
    });
    expect(container.querySelector('[data-testid="empty-subtitle"]')).toBeNull();
  });

  it('renders action button when actionLabel is provided', () => {
    const { getByText } = render(EmptyState, {
      props: { message: 'No items', actionLabel: 'Add Item' },
    });
    expect(getByText('Add Item')).toBeTruthy();
  });

  it('does not render action button when actionLabel is not provided', () => {
    const { container } = render(EmptyState, {
      props: { message: 'No items' },
    });
    expect(container.querySelector('button')).toBeNull();
  });

  it('renders the icon slot content', () => {
    const { container } = render(EmptyState, {
      props: { message: 'No items', icon: '🌐' },
    });
    expect(container.querySelector('[data-testid="empty-icon"]')).toBeTruthy();
  });
});
```

**Step 2: Run tests to verify they fail**

Run: `cd frontend && npx vitest run src/lib/EmptyState.test.js`
Expected: FAIL — module not found

**Step 3: Write the EmptyState component**

Create `frontend/src/lib/EmptyState.svelte`:

```svelte
<script>
  import { createEventDispatcher } from 'svelte';

  export let message = '';
  export let subtitle = '';
  export let actionLabel = '';
  export let icon = '';

  const dispatch = createEventDispatcher();
</script>

<div class="py-10 text-center">
  {#if icon}
    <div data-testid="empty-icon" class="text-base-content/30 mb-3 flex justify-center">
      {@html icon}
    </div>
  {/if}
  <p class="text-base-content/60 font-semibold">{message}</p>
  {#if subtitle}
    <p data-testid="empty-subtitle" class="text-base-content/40 text-sm mt-1">{subtitle}</p>
  {/if}
  {#if actionLabel}
    <button class="btn btn-ghost btn-sm mt-4" on:click={() => dispatch('action')}>
      {actionLabel}
    </button>
  {/if}
</div>
```

**Step 4: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/lib/EmptyState.test.js`
Expected: All 6 tests PASS

**Step 5: Commit**

```bash
git add frontend/src/lib/EmptyState.svelte frontend/src/lib/EmptyState.test.js
git commit -m "feat: add EmptyState component with tests"
```

---

### Task 2: Wire EmptyState into SiteList

**Files:**
- Modify: `frontend/src/SiteList.svelte` (line 57 — the empty `<p>` tag)
- Modify: `frontend/src/SiteList.test.js` (line 54-59 — the "shows empty message" test)

**Step 1: Update the existing empty-state test**

In `frontend/src/SiteList.test.js`, replace the test at line 54-59:

```js
  it('shows empty message when no sites', () => {
    const { getByText } = render(SiteList, {
      props: { sites: [], loaded: true, onRemove: vi.fn() },
    });
    expect(getByText(/No sites registered/)).toBeTruthy();
  });
```

With:

```js
  it('shows empty state with icon and action when no sites', () => {
    const { getByText, container } = render(SiteList, {
      props: { sites: [], loaded: true, onRemove: vi.fn() },
    });
    expect(getByText('No sites yet')).toBeTruthy();
    expect(getByText('Add your first site to start developing locally.')).toBeTruthy();
    expect(getByText('Add Site')).toBeTruthy();
    expect(container.querySelector('[data-testid="empty-icon"]')).toBeTruthy();
  });
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/SiteList.test.js`
Expected: FAIL — "No sites yet" not found (still shows old text)

**Step 3: Update SiteList.svelte**

In `frontend/src/SiteList.svelte`:

Add import at top of `<script>` block (after existing imports):

```js
  import EmptyState from './lib/EmptyState.svelte';
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();
```

Replace line 57:

```svelte
  <p class="text-base-content/50 py-8">No sites registered. Add one below.</p>
```

With:

```svelte
  <EmptyState
    icon='<svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>'
    message="No sites yet"
    subtitle="Add your first site to start developing locally."
    actionLabel="Add Site"
    on:action={() => dispatch('addsite')}
  />
```

**Step 4: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/SiteList.test.js`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add frontend/src/SiteList.svelte frontend/src/SiteList.test.js
git commit -m "feat: use EmptyState component in SiteList"
```

---

### Task 3: Make AddSiteForm externally controllable

**Files:**
- Modify: `frontend/src/AddSiteForm.svelte` (line 13 — `collapseOpen` variable)
- Modify: `frontend/src/AddSiteForm.test.js` (add new test)

**Step 1: Write a failing test for external open control**

Add to the end of `frontend/src/AddSiteForm.test.js`:

```js
  it('can be opened externally via collapseOpen prop', () => {
    const { container } = render(AddSiteForm, {
      props: { collapseOpen: true },
    });
    const checkbox = container.querySelector('.collapse input[type="checkbox"]');
    expect(checkbox.checked).toBe(true);
  });
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/AddSiteForm.test.js`
Expected: FAIL — collapseOpen prop is not recognized (it's currently a local `let`, not an `export let`)

**Step 3: Make collapseOpen a prop**

In `frontend/src/AddSiteForm.svelte`, change line 13 from:

```js
  let collapseOpen = false;
```

To:

```js
  export let collapseOpen = false;
```

**Step 4: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/AddSiteForm.test.js`
Expected: All tests PASS (existing tests still pass because default is `false`)

**Step 5: Commit**

```bash
git add frontend/src/AddSiteForm.svelte frontend/src/AddSiteForm.test.js
git commit -m "feat: expose collapseOpen as bindable prop on AddSiteForm"
```

---

### Task 4: Wire SiteList CTA to open AddSiteForm in App.svelte

**Files:**
- Modify: `frontend/src/App.svelte` (lines 83-87 — the Sites section)

**Step 1: Add state variable and event handler**

No automated test for this — it's pure wiring between components. App.svelte is not unit-tested (it's the root component tested via manual `wails dev`).

In `frontend/src/App.svelte`, add a new variable after `let servicesLoaded = false;` (line 14):

```js
  let addFormOpen = false;
```

**Step 2: Update the Sites section template**

Replace lines 85-86:

```svelte
    <SiteList {sites} loaded={sitesLoaded} onRemove={handleRemove} />
    <AddSiteForm onAdd={handleAdd} />
```

With:

```svelte
    <SiteList {sites} loaded={sitesLoaded} onRemove={handleRemove} on:addsite={() => { addFormOpen = true; }} />
    <AddSiteForm onAdd={handleAdd} bind:collapseOpen={addFormOpen} />
```

**Step 3: Run all frontend tests to verify nothing is broken**

Run: `cd frontend && npx vitest run`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add frontend/src/App.svelte
git commit -m "feat: wire SiteList empty-state CTA to open AddSiteForm"
```

---

### Task 5: Wire EmptyState into ServiceList

**Files:**
- Modify: `frontend/src/ServiceList.svelte` (lines 55-56 — the empty `<p>` tag)

**Step 1: Update ServiceList.svelte**

No existing test for the empty message in ServiceList (there's no `ServiceList.test.js`). Add the component inline.

In `frontend/src/ServiceList.svelte`, add import at top of `<script>` block:

```js
  import EmptyState from './lib/EmptyState.svelte';
```

Replace lines 55-56:

```svelte
  <p class="text-base-content/50 py-8">No database services configured.</p>
```

With:

```svelte
  <EmptyState
    icon='<svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5"/><path d="M3 12c0 1.66 4.03 3 9 3s9-1.34 9-3"/></svg>'
    message="No services available"
    subtitle="Install database plugins to manage MySQL, PostgreSQL, and Redis."
    actionLabel="Setup Guide"
    on:action={() => window.open('https://github.com/andybarilla/rook#services', '_blank')}
  />
```

Note: The docs URL (`https://github.com/andybarilla/rook#services`) is a placeholder. Update it when actual documentation exists.

**Step 2: Run all frontend tests to verify nothing is broken**

Run: `cd frontend && npx vitest run`
Expected: All tests PASS

**Step 3: Commit**

```bash
git add frontend/src/ServiceList.svelte
git commit -m "feat: use EmptyState component in ServiceList"
```

---

### Task 6: Final verification and cleanup

**Step 1: Run all frontend tests**

Run: `cd frontend && npx vitest run`
Expected: All tests PASS

**Step 2: Run all Go tests**

Run: `cd /home/andy/dev/andybarilla/rook && go test ./internal/... -v`
Expected: All 54+ tests PASS (no Go changes, just confirming nothing is broken)

**Step 3: Final commit with any remaining changes**

If any files were missed, stage and commit them.

**Step 4: Update roadmap**

In `docs/ROADMAP.md`, change:

```markdown
- [ ] Empty state improvements (actionable guidance for sites and services)
```

To:

```markdown
- [x] Empty state improvements (actionable guidance for sites and services) — See: docs/plans/2026-03-04-empty-state-improvements-design.md
```

Commit:

```bash
git add docs/ROADMAP.md
git commit -m "docs: mark empty state improvements as complete"
```
