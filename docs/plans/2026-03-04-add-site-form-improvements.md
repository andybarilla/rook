# Add Site Form Improvements — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Improve AddSiteForm visual hierarchy with grouped sections and add Node Version column to the site table.

**Architecture:** Two small UI changes — restructure AddSiteForm markup into "Site" (required) and "Options" (optional) groups separated by a DaisyUI divider, and add a Node column to SiteList. Both are pure Svelte template changes with no backend work.

**Tech Stack:** Svelte, DaisyUI, Vitest + @testing-library/svelte

---

### Task 1: Update AddSiteForm Tests for Grouped Sections

**Files:**
- Modify: `frontend/src/AddSiteForm.test.js`

**Step 1: Update tests to reflect new grouped layout**

Replace the existing row-based structure tests with section-based tests. The form will have two sections: a required "Site" section and an optional "Options" section separated by a divider.

```javascript
import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import AddSiteForm from './AddSiteForm.svelte';

describe('AddSiteForm', () => {
  it('is collapsed by default', () => {
    const { container } = render(AddSiteForm);
    const collapse = container.querySelector('.collapse');
    expect(collapse).toBeTruthy();
    const checkbox = collapse.querySelector('input[type="checkbox"]');
    expect(checkbox.checked).toBe(false);
  });

  it('expands when the collapse title is clicked', async () => {
    const { container } = render(AddSiteForm);
    const checkbox = container.querySelector('.collapse input[type="checkbox"]');
    await fireEvent.click(checkbox);
    expect(checkbox.checked).toBe(true);
  });

  it('has a required section with Path and Domain fields', () => {
    const { container } = render(AddSiteForm);
    const requiredSection = container.querySelector('[data-section="required"]');
    expect(requiredSection).toBeTruthy();
    expect(requiredSection.textContent).toContain('Path');
    expect(requiredSection.textContent).toContain('Domain');
  });

  it('required fields use input-md for visual prominence', () => {
    const { container } = render(AddSiteForm);
    const requiredSection = container.querySelector('[data-section="required"]');
    const inputs = requiredSection.querySelectorAll('input[type="text"]');
    inputs.forEach((input) => {
      expect(input.classList.contains('input-md')).toBe(true);
    });
  });

  it('has a divider separating required and optional sections', () => {
    const { container } = render(AddSiteForm);
    const divider = container.querySelector('.divider');
    expect(divider).toBeTruthy();
    expect(divider.textContent).toContain('Options');
  });

  it('has an optional section with PHP, Node, TLS', () => {
    const { container } = render(AddSiteForm);
    const optionalSection = container.querySelector('[data-section="optional"]');
    expect(optionalSection).toBeTruthy();
    expect(optionalSection.textContent).toContain('PHP Version');
    expect(optionalSection.textContent).toContain('Node Version');
    expect(optionalSection.textContent).toContain('TLS');
  });

  it('optional fields use input-sm to de-emphasize', () => {
    const { container } = render(AddSiteForm);
    const optionalSection = container.querySelector('[data-section="optional"]');
    const inputs = optionalSection.querySelectorAll('input[type="text"]');
    inputs.forEach((input) => {
      expect(input.classList.contains('input-sm')).toBe(true);
    });
  });

  it('auto-collapses after successful submission', async () => {
    const onAdd = vi.fn().mockResolvedValue(undefined);
    const { container, getByPlaceholderText } = render(AddSiteForm, {
      props: { onAdd },
    });
    const checkbox = container.querySelector('.collapse input[type="checkbox"]');
    await fireEvent.click(checkbox);
    expect(checkbox.checked).toBe(true);
    const pathInput = getByPlaceholderText('/home/user/projects/myapp');
    const domainInput = getByPlaceholderText('myapp.test');
    await fireEvent.input(pathInput, { target: { value: '/tmp/app' } });
    await fireEvent.input(domainInput, { target: { value: 'app.test' } });
    const form = container.querySelector('form');
    await fireEvent.submit(form);
    await vi.waitFor(() => {
      expect(checkbox.checked).toBe(false);
    });
  });
});
```

**Step 2: Run tests to verify they fail**

Run: `cd frontend && npx vitest run src/AddSiteForm.test.js`
Expected: FAIL — tests reference `[data-section="required"]` and `.divider` which don't exist yet.

**Step 3: Commit failing tests**

```bash
git add frontend/src/AddSiteForm.test.js
git commit -m "test: update AddSiteForm tests for grouped sections layout"
```

---

### Task 2: Implement AddSiteForm Grouped Sections

**Files:**
- Modify: `frontend/src/AddSiteForm.svelte`

**Step 1: Update the template to use grouped sections**

Replace the form body (everything inside `<form>`) with:

```svelte
<form on:submit|preventDefault={handleSubmit}>
  <div data-section="required" class="mb-2">
    <div class="form-row flex gap-4 items-end mb-3">
      <label class="flex flex-col flex-[2] text-left">
        <span class="text-xs text-base-content/50 uppercase tracking-wide mb-1">Path</span>
        <input type="text" class="input input-bordered input-md" bind:value={path} on:input={handlePathInput} placeholder="/home/user/projects/myapp" disabled={submitting} />
      </label>
      <label class="flex flex-col flex-1 text-left">
        <span class="text-xs text-base-content/50 uppercase tracking-wide mb-1">Domain</span>
        <input type="text" class="input input-bordered input-md" bind:value={domain} placeholder="myapp.test" disabled={submitting} />
      </label>
    </div>
  </div>
  <div class="divider text-xs text-base-content/30">Options</div>
  <div data-section="optional">
    <div class="form-row flex gap-4 items-end mb-3">
      <label class="flex flex-col flex-1 text-left">
        <span class="text-xs text-base-content/50 uppercase tracking-wide mb-1">PHP Version</span>
        <input type="text" class="input input-bordered input-sm" bind:value={phpVersion} placeholder="8.3 (optional)" disabled={submitting} />
      </label>
      <label class="flex flex-col flex-1 text-left">
        <span class="text-xs text-base-content/50 uppercase tracking-wide mb-1">Node Version</span>
        <input type="text" class="input input-bordered input-sm" bind:value={nodeVersion} placeholder="system (optional)" disabled={submitting} />
      </label>
    </div>
    <div class="form-row flex gap-4 items-end">
      <label class="flex flex-row items-center gap-2 flex-none whitespace-nowrap">
        <input type="checkbox" class="checkbox checkbox-sm" bind:checked={tls} disabled={submitting} />
        <span class="text-xs text-base-content/50 uppercase tracking-wide">TLS</span>
      </label>
      <div class="flex-1"></div>
      <button type="submit" class="btn btn-success btn-sm" disabled={submitting}>
        {#if submitting}
          <span class="loading loading-spinner loading-xs"></span>
          Adding…
        {:else}
          Add Site
        {/if}
      </button>
    </div>
  </div>
</form>
```

**Step 2: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/AddSiteForm.test.js`
Expected: PASS — all 8 tests green.

**Step 3: Commit**

```bash
git add frontend/src/AddSiteForm.svelte
git commit -m "feat: group AddSiteForm fields into required/optional sections"
```

---

### Task 3: Update SiteList Tests for Node Version Column

**Files:**
- Modify: `frontend/src/SiteList.test.js`

**Step 1: Update test data and add Node column tests**

Update `fakeSites` to include `node_version` and add tests for the new column:

At the top of the file, update `fakeSites`:
```javascript
const fakeSites = [
  { domain: 'app.test', path: '/home/user/app', php_version: '8.3', node_version: '20', tls: true },
  { domain: 'blog.test', path: '/home/user/blog', php_version: '', node_version: '', tls: false },
];
```

Add these two new tests inside the `describe` block:

```javascript
  it('shows Node column header', () => {
    const { container } = render(SiteList, {
      props: { sites: fakeSites, loaded: true, onRemove: vi.fn() },
    });
    const headers = container.querySelectorAll('th');
    const headerTexts = Array.from(headers).map((h) => h.textContent);
    expect(headerTexts).toContain('Node');
  });

  it('displays node_version or dash for each site', () => {
    const { container } = render(SiteList, {
      props: { sites: fakeSites, loaded: true, onRemove: vi.fn() },
    });
    const rows = container.querySelectorAll('tbody tr');
    // First site has node_version '20'
    expect(rows[0].textContent).toContain('20');
    // Second site has empty node_version, should show dash
    expect(rows[1].cells[3].textContent).toBe('—');
  });
```

**Step 2: Run tests to verify they fail**

Run: `cd frontend && npx vitest run src/SiteList.test.js`
Expected: FAIL — no "Node" header or column exists yet.

**Step 3: Commit failing tests**

```bash
git add frontend/src/SiteList.test.js
git commit -m "test: add SiteList tests for Node Version column"
```

---

### Task 4: Implement Node Version Column in SiteList

**Files:**
- Modify: `frontend/src/SiteList.svelte`

**Step 1: Add Node column to both table headers and skeleton**

In both `<thead>` sections (skeleton and data table), add `<th>Node</th>` after `<th>PHP</th>`.

In the skeleton `<tbody>`, add a skeleton cell after the PHP skeleton cell:
```svelte
<td><div class="skeleton h-4 w-10"></div></td>
```

In the data `<tbody>`, add this cell after the PHP cell:
```svelte
<td>{site.node_version || '—'}</td>
```

**Step 2: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/SiteList.test.js`
Expected: PASS — all 7 tests green.

**Step 3: Run all frontend tests**

Run: `cd frontend && npx vitest run`
Expected: All tests pass.

**Step 4: Commit**

```bash
git add frontend/src/SiteList.svelte
git commit -m "feat: add Node Version column to site table"
```

---

### Task 5: Update Roadmap

**Files:**
- Modify: `docs/ROADMAP.md`

**Step 1: Mark both items as complete**

In `docs/ROADMAP.md`, update:
- `- [ ] Add Site form improvements` → `- [x] Add Site form improvements (collapsible form, better layout, confirmation on remove) — See: docs/plans/2026-03-04-add-site-form-improvements-design.md`
- `- [ ] Site table: show Node Version column` → `- [x] Site table: show Node Version column — See: docs/plans/2026-03-04-add-site-form-improvements-design.md`

**Step 2: Commit**

```bash
git add docs/ROADMAP.md
git commit -m "docs: mark Add Site form improvements and Node Version column as complete"
```
