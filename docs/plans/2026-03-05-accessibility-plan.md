# Accessibility Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Meet WCAG AA color contrast and add keyboard shortcuts (Ctrl+N, Escape, Ctrl+Enter) with proper focus management.

**Architecture:** All changes are in the Svelte frontend. Contrast fixes are CSS class swaps. Keyboard shortcuts use a single `svelte:window` keydown handler in App.svelte. Focus management adds auto-focus on form open, focus trap in ConfirmModal, and focus return on close.

**Tech Stack:** Svelte 3, DaisyUI 5, Tailwind CSS 4, Vitest + Testing Library

---

### Task 1: Color Contrast Fixes

**Files:**
- Modify: `frontend/src/App.svelte`
- Modify: `frontend/src/SiteList.svelte`
- Modify: `frontend/src/ServiceList.svelte`
- Modify: `frontend/src/AddSiteForm.svelte`
- Modify: `frontend/src/lib/EmptyState.svelte`

**Step 1: Update opacity classes across all components**

Apply this mapping everywhere:

| Old | New |
|-----|-----|
| `text-base-content/30` | `text-base-content/50` |
| `text-base-content/40` | `text-base-content/60` |
| `text-base-content/50` | `text-base-content/70` |
| `text-base-content/60` | `text-base-content/70` |

Specific changes per file:

**`App.svelte`:**
- Line: `<p class="text-base-content/50 ...">` → `text-base-content/70`
- Line: `<h2 class="text-sm text-base-content/60 ...">` (2 occurrences) → `text-base-content/70`

**`AddSiteForm.svelte`:**
- Line: `<div class="collapse-title text-xs text-base-content/50 ...">` → `text-base-content/70`
- All `<span class="text-xs text-base-content/50 ...">` label spans (5 occurrences) → `text-base-content/70`
- Line: `<div class="divider text-xs text-base-content/30">` → `text-base-content/50`

**`SiteList.svelte`:**
- Line: `<td class="text-base-content/60 text-sm">` → `text-base-content/70`

**`ServiceList.svelte`:**
- Line: `<td class="text-base-content/60 text-sm">` → `text-base-content/70`

**`EmptyState.svelte`:**
- Line: `<div ... class="text-base-content/30 ...">` (icon) → `text-base-content/50`
- Line: `<p class="text-base-content/60 ...">` (message) → `text-base-content/70`
- Line: `<p ... class="text-base-content/40 ...">` (subtitle) → `text-base-content/60`

**Step 2: Run existing tests to confirm nothing breaks**

Run: `cd frontend && npm test`
Expected: All existing tests pass (contrast changes are CSS-only, won't break assertions).

**Step 3: Commit**

```bash
git add frontend/src/App.svelte frontend/src/SiteList.svelte frontend/src/ServiceList.svelte frontend/src/AddSiteForm.svelte frontend/src/lib/EmptyState.svelte
git commit -m "feat(a11y): bump text opacity classes to meet WCAG AA contrast"
```

---

### Task 2: Keyboard Shortcut — Ctrl+N to Open Add Site Form

**Files:**
- Modify: `frontend/src/App.svelte`
- Modify: `frontend/src/AddSiteForm.svelte`
- Test: `frontend/src/AddSiteForm.test.js`

**Step 1: Write the failing test for focusPathInput method**

Add to `frontend/src/AddSiteForm.test.js`:

```javascript
it('exposes focusPathInput method that focuses the path input', async () => {
  const { container, component } = render(AddSiteForm, {
    props: { collapseOpen: true },
  });
  component.focusPathInput();
  const pathInput = container.querySelector('input[placeholder="/home/user/projects/myapp"]');
  expect(document.activeElement).toBe(pathInput);
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/AddSiteForm.test.js`
Expected: FAIL — `component.focusPathInput is not a function`

**Step 3: Implement focusPathInput in AddSiteForm.svelte**

Add a `bind:this` ref on the Path input and export a focus method:

In the `<script>` block, add:
```javascript
let pathInput;

export function focusPathInput() {
  if (pathInput) pathInput.focus();
}
```

On the Path `<input>`, add `bind:this={pathInput}`:
```svelte
<input type="text" class="input input-bordered input-md" bind:value={path} bind:this={pathInput} on:input={handlePathInput} placeholder="/home/user/projects/myapp" disabled={submitting} />
```

**Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/AddSiteForm.test.js`
Expected: PASS

**Step 5: Write the failing test for Ctrl+N in App.svelte**

Create `frontend/src/App.test.js`. Since App.svelte imports Wails bindings, mock them:

```javascript
import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';

// Mock Wails bindings
vi.mock('../wailsjs/go/main/App.js', () => ({
  ListSites: vi.fn().mockResolvedValue([]),
  AddSite: vi.fn().mockResolvedValue(undefined),
  RemoveSite: vi.fn().mockResolvedValue(undefined),
  DatabaseServices: vi.fn().mockResolvedValue([]),
  StartDatabase: vi.fn().mockResolvedValue(undefined),
  StopDatabase: vi.fn().mockResolvedValue(undefined),
}));

import App from './App.svelte';

describe('App keyboard shortcuts', () => {
  it('Ctrl+N opens add site form and focuses path input', async () => {
    const { container } = render(App);
    // Wait for onMount to complete
    await vi.waitFor(() => {
      expect(container.querySelector('.collapse')).toBeTruthy();
    });
    await fireEvent.keyDown(window, { key: 'n', ctrlKey: true });
    await vi.waitFor(() => {
      const checkbox = container.querySelector('.collapse input[type="checkbox"]');
      expect(checkbox.checked).toBe(true);
    });
  });
});
```

**Step 6: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/App.test.js`
Expected: FAIL — no keydown handler exists yet

**Step 7: Add keydown handler to App.svelte**

In App.svelte `<script>`, add:
```javascript
let addSiteForm;

function handleKeydown(e) {
  if (e.ctrlKey && e.key === 'n') {
    e.preventDefault();
    addFormOpen = true;
    // Wait for DOM update, then focus
    setTimeout(() => addSiteForm?.focusPathInput(), 0);
  }
}
```

In the template, add the window listener and bind the form component:
```svelte
<svelte:window on:keydown={handleKeydown} />
```

On AddSiteForm, add `bind:this`:
```svelte
<AddSiteForm bind:this={addSiteForm} onAdd={handleAdd} bind:collapseOpen={addFormOpen} />
```

**Step 8: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/App.test.js`
Expected: PASS

**Step 9: Commit**

```bash
git add frontend/src/App.svelte frontend/src/App.test.js frontend/src/AddSiteForm.svelte frontend/src/AddSiteForm.test.js
git commit -m "feat(a11y): Ctrl+N opens Add Site form and focuses path input"
```

---

### Task 3: Keyboard Shortcut — Escape to Close Form and Dismiss Toast

**Files:**
- Modify: `frontend/src/App.svelte`
- Modify: `frontend/src/lib/notifications.js`
- Test: `frontend/src/App.test.js`
- Test: `frontend/src/lib/notifications.test.js`

**Step 1: Write the failing test for Escape closing the add form**

Add to `frontend/src/App.test.js`:

```javascript
it('Escape closes add site form when open', async () => {
  const { container } = render(App);
  await vi.waitFor(() => {
    expect(container.querySelector('.collapse')).toBeTruthy();
  });
  // Open the form first
  await fireEvent.keyDown(window, { key: 'n', ctrlKey: true });
  await vi.waitFor(() => {
    const checkbox = container.querySelector('.collapse input[type="checkbox"]');
    expect(checkbox.checked).toBe(true);
  });
  // Press Escape
  await fireEvent.keyDown(window, { key: 'Escape' });
  await vi.waitFor(() => {
    const checkbox = container.querySelector('.collapse input[type="checkbox"]');
    expect(checkbox.checked).toBe(false);
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/App.test.js`
Expected: FAIL — Escape doesn't close the form yet

**Step 3: Add dismissLatest to notifications.js**

First, check `frontend/src/lib/notifications.js`:

```javascript
// Add to notifications.js:
export function dismissLatest() {
  let dismissed = false;
  notifications.update(items => {
    if (items.length === 0) return items;
    dismissed = true;
    return items.slice(0, -1);
  });
  return dismissed;
}
```

**Step 4: Write failing test for dismissLatest**

Add to `frontend/src/lib/notifications.test.js`:

```javascript
import { notifications, notifySuccess, dismiss, dismissLatest } from './notifications.js';
import { get } from 'svelte/store';

it('dismissLatest removes the most recent notification', () => {
  notifySuccess('First');
  notifySuccess('Second');
  const result = dismissLatest();
  expect(result).toBe(true);
  const items = get(notifications);
  expect(items).toHaveLength(1);
  expect(items[0].message).toBe('First');
});

it('dismissLatest returns false when no notifications', () => {
  // Clear all first
  notifications.set([]);
  const result = dismissLatest();
  expect(result).toBe(false);
});
```

**Step 5: Run notification tests to verify they fail, then implement**

Run: `cd frontend && npx vitest run src/lib/notifications.test.js`
Expected: FAIL — `dismissLatest` not exported

Then add the `dismissLatest` function to `notifications.js` and re-run.
Expected: PASS

**Step 6: Add Escape handling to App.svelte handleKeydown**

```javascript
function handleKeydown(e) {
  if (e.ctrlKey && e.key === 'n') {
    e.preventDefault();
    addFormOpen = true;
    setTimeout(() => addSiteForm?.focusPathInput(), 0);
    return;
  }
  if (e.key === 'Escape') {
    if (addFormOpen) {
      addFormOpen = false;
      return;
    }
    dismissLatest();
  }
}
```

Add import at top of App.svelte:
```javascript
import { dismissLatest } from './lib/notifications.js';
```

**Step 7: Run all tests**

Run: `cd frontend && npm test`
Expected: All PASS

**Step 8: Commit**

```bash
git add frontend/src/App.svelte frontend/src/App.test.js frontend/src/lib/notifications.js frontend/src/lib/notifications.test.js
git commit -m "feat(a11y): Escape closes Add Site form and dismisses toasts"
```

---

### Task 4: Keyboard Shortcut — Ctrl+Enter to Submit Form

**Files:**
- Modify: `frontend/src/App.svelte`
- Test: `frontend/src/App.test.js`

**Step 1: Write the failing test**

Add to `frontend/src/App.test.js`:

```javascript
it('Ctrl+Enter submits the add site form when open', async () => {
  const { AddSite } = await import('../wailsjs/go/main/App.js');
  const { container } = render(App);
  await vi.waitFor(() => {
    expect(container.querySelector('.collapse')).toBeTruthy();
  });
  // Open form and fill in fields
  await fireEvent.keyDown(window, { key: 'n', ctrlKey: true });
  await vi.waitFor(() => {
    const checkbox = container.querySelector('.collapse input[type="checkbox"]');
    expect(checkbox.checked).toBe(true);
  });
  const pathInput = container.querySelector('input[placeholder="/home/user/projects/myapp"]');
  const domainInput = container.querySelector('input[placeholder="myapp.test"]');
  await fireEvent.input(pathInput, { target: { value: '/tmp/myapp' } });
  await fireEvent.input(domainInput, { target: { value: 'myapp.test' } });
  // Ctrl+Enter
  await fireEvent.keyDown(window, { key: 'Enter', ctrlKey: true });
  await vi.waitFor(() => {
    expect(AddSite).toHaveBeenCalledWith('/tmp/myapp', 'myapp.test', '', '', false);
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/App.test.js`
Expected: FAIL — Ctrl+Enter not handled

**Step 3: Add Ctrl+Enter handling and expose submit method**

In `AddSiteForm.svelte`, export the submit method:
```javascript
export { handleSubmit };
```

In `App.svelte`, add to handleKeydown:
```javascript
if (e.ctrlKey && e.key === 'Enter') {
  if (addFormOpen) {
    e.preventDefault();
    addSiteForm?.handleSubmit();
  }
  return;
}
```

**Step 4: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/App.test.js`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/App.svelte frontend/src/App.test.js frontend/src/AddSiteForm.svelte
git commit -m "feat(a11y): Ctrl+Enter submits Add Site form"
```

---

### Task 5: Focus Trap in ConfirmModal

**Files:**
- Modify: `frontend/src/lib/ConfirmModal.svelte`
- Test: `frontend/src/lib/ConfirmModal.test.js`

**Step 1: Write the failing test for focus trap**

Add to `frontend/src/lib/ConfirmModal.test.js`:

```javascript
it('focuses cancel button when modal opens', async () => {
  const { getByText } = render(ConfirmModal, {
    props: { open: true, title: 'T', message: 'M', onConfirm: vi.fn(), onCancel: vi.fn() },
  });
  await vi.waitFor(() => {
    expect(document.activeElement).toBe(getByText('Cancel'));
  });
});

it('traps focus within modal on Tab', async () => {
  const { getByText } = render(ConfirmModal, {
    props: { open: true, title: 'T', message: 'M', onConfirm: vi.fn(), onCancel: vi.fn() },
  });
  const cancelBtn = getByText('Cancel');
  const confirmBtn = getByText('Confirm');
  // Focus should start on Cancel
  await vi.waitFor(() => expect(document.activeElement).toBe(cancelBtn));
  // Tab to Confirm
  await fireEvent.keyDown(cancelBtn, { key: 'Tab' });
  // Shift+Tab from Cancel should wrap to Confirm
  cancelBtn.focus();
  await fireEvent.keyDown(cancelBtn, { key: 'Tab', shiftKey: true });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/lib/ConfirmModal.test.js`
Expected: FAIL — modal doesn't auto-focus

**Step 3: Implement focus trap in ConfirmModal.svelte**

Replace the ConfirmModal script and template:

```svelte
<script>
  import { onMount, afterUpdate, tick } from 'svelte';

  export let open = false;
  export let title = '';
  export let message = '';
  export let confirmLabel = 'Confirm';
  export let confirmClass = 'btn-primary';
  export let onConfirm = () => {};
  export let onCancel = () => {};

  let cancelBtn;
  let confirmBtn;
  let previouslyFocused;

  $: if (open) {
    previouslyFocused = document.activeElement;
    tick().then(() => cancelBtn?.focus());
  }
  $: if (!open && previouslyFocused) {
    previouslyFocused?.focus();
    previouslyFocused = null;
  }

  function trapFocus(e) {
    if (e.key !== 'Tab' || !open) return;
    const focusable = [cancelBtn, confirmBtn].filter(Boolean);
    if (focusable.length === 0) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  }
</script>

<svelte:window on:keydown={(e) => {
  if (open && e.key === 'Escape') onCancel();
  if (open) trapFocus(e);
}} />

{#if open}
  <div class="modal modal-open" role="dialog" aria-modal="true" aria-labelledby="modal-title">
    <div class="modal-box">
      <h3 id="modal-title" class="font-bold text-lg">{title}</h3>
      <p class="py-4">{message}</p>
      <div class="modal-action">
        <button bind:this={cancelBtn} class="btn btn-ghost" on:click={onCancel}>Cancel</button>
        <button bind:this={confirmBtn} class="btn {confirmClass}" on:click={onConfirm}>{confirmLabel}</button>
      </div>
    </div>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <div class="modal-backdrop" on:click={onCancel}></div>
  </div>
{/if}
```

**Step 4: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/lib/ConfirmModal.test.js`
Expected: PASS

**Step 5: Write failing test for focus return to Remove button**

Add to `frontend/src/SiteList.test.js`:

```javascript
it('returns focus to Remove button after modal cancel', async () => {
  const { getAllByText, getByText } = render(SiteList, {
    props: { sites: fakeSites, loaded: true, onRemove: vi.fn() },
  });
  const removeBtn = getAllByText('Remove')[0];
  await fireEvent.click(removeBtn);
  // Modal is open, cancel it
  await fireEvent.click(getByText('Cancel'));
  await vi.waitFor(() => {
    expect(document.activeElement).toBe(removeBtn);
  });
});
```

**Step 6: Run test — it should pass already** (ConfirmModal now restores focus via `previouslyFocused`)

Run: `cd frontend && npx vitest run src/SiteList.test.js`
Expected: PASS

**Step 7: Run all tests**

Run: `cd frontend && npm test`
Expected: All PASS

**Step 8: Commit**

```bash
git add frontend/src/lib/ConfirmModal.svelte frontend/src/lib/ConfirmModal.test.js frontend/src/SiteList.test.js
git commit -m "feat(a11y): focus trap and focus return in ConfirmModal"
```

---

### Task 6: Update Roadmap

**Files:**
- Modify: `docs/ROADMAP.md`

**Step 1: Mark accessibility item complete**

Change:
```markdown
- [ ] Accessibility (color contrast, keyboard shortcuts)
```
To:
```markdown
- [x] Accessibility (color contrast, keyboard shortcuts) — See: docs/plans/2026-03-05-accessibility-design.md
```

**Step 2: Commit**

```bash
git add docs/ROADMAP.md
git commit -m "docs: mark accessibility task complete in roadmap"
```
