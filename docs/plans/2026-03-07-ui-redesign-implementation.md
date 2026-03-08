# UI Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Incrementally reskin the Rook frontend with tab navigation, card views, light/dark theming, and purple accent — inspired by the Figma prototype.

**Architecture:** Modify existing Svelte + DaisyUI components in-place. Add tab navigation via reactive variable (no router). New components: SiteCard, AddSiteModal, ServiceCard, SettingsTab. Theme switching via DaisyUI's built-in multi-theme support + localStorage persistence.

**Tech Stack:** Svelte 3, DaisyUI 5.5, Tailwind CSS 4, Vitest, @testing-library/svelte

---

### Task 1: Theme Configuration — Purple Accent + Light/Dark

**Files:**
- Modify: `frontend/src/style.css`

**Step 1: Update style.css to support light and dark themes with purple accent**

Replace the contents of `frontend/src/style.css` with:

```css
@import "tailwindcss";
@plugin "daisyui" {
  themes: light --default, dark;
}

@theme {
  --color-primary: #7c3aed;
  --color-primary-content: #ffffff;
}

@font-face {
  font-family: "Nunito";
  font-style: normal;
  font-weight: 400;
  src: local(""),
    url("assets/fonts/nunito-v16-latin-regular.woff2") format("woff2");
}

body {
  font-family: "Nunito", -apple-system, BlinkMacSystemFont, "Segoe UI", "Roboto",
    "Oxygen", "Ubuntu", "Cantarell", "Fira Sans", "Droid Sans", "Helvetica Neue",
    sans-serif;
}

#app {
  height: 100vh;
}
```

**Step 2: Run existing tests to verify nothing breaks**

Run: `cd frontend && npm test`
Expected: All existing tests pass (theme change is CSS-only, no test impact)

**Step 3: Commit**

```bash
git add frontend/src/style.css
git commit -m "style: switch to purple accent with light/dark DaisyUI themes"
```

---

### Task 2: Theme Store — Persistence + Toggle

**Files:**
- Create: `frontend/src/lib/theme.js`
- Test: `frontend/src/lib/theme.test.js`

**Step 1: Write the failing test**

Create `frontend/src/lib/theme.test.js`:

```javascript
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import { theme, toggleTheme, initTheme } from './theme.js';

describe('theme store', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.setAttribute('data-theme', '');
    theme.set('light');
  });

  it('defaults to light', () => {
    expect(get(theme)).toBe('light');
  });

  it('toggleTheme switches light to dark', () => {
    toggleTheme();
    expect(get(theme)).toBe('dark');
  });

  it('toggleTheme switches dark to light', () => {
    theme.set('dark');
    toggleTheme();
    expect(get(theme)).toBe('light');
  });

  it('toggleTheme persists to localStorage', () => {
    toggleTheme();
    expect(localStorage.getItem('rook-theme')).toBe('dark');
  });

  it('toggleTheme sets data-theme attribute on html', () => {
    toggleTheme();
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
  });

  it('initTheme reads from localStorage', () => {
    localStorage.setItem('rook-theme', 'dark');
    initTheme();
    expect(get(theme)).toBe('dark');
    expect(document.documentElement.getAttribute('data-theme')).toBe('dark');
  });

  it('initTheme defaults to light when no stored value', () => {
    initTheme();
    expect(get(theme)).toBe('light');
    expect(document.documentElement.getAttribute('data-theme')).toBe('light');
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/lib/theme.test.js`
Expected: FAIL — module not found

**Step 3: Write implementation**

Create `frontend/src/lib/theme.js`:

```javascript
import { writable, get } from 'svelte/store';

export const theme = writable('light');

function applyTheme(value) {
  document.documentElement.setAttribute('data-theme', value);
  localStorage.setItem('rook-theme', value);
}

export function toggleTheme() {
  const next = get(theme) === 'light' ? 'dark' : 'light';
  theme.set(next);
  applyTheme(next);
}

export function initTheme() {
  const stored = localStorage.getItem('rook-theme') || 'light';
  theme.set(stored);
  applyTheme(stored);
}
```

**Step 4: Run test to verify it passes**

Run: `cd frontend && npx vitest run src/lib/theme.test.js`
Expected: All PASS

**Step 5: Run all tests**

Run: `cd frontend && npm test`
Expected: All pass

**Step 6: Commit**

```bash
git add frontend/src/lib/theme.js frontend/src/lib/theme.test.js
git commit -m "feat: add theme store with light/dark toggle and localStorage persistence"
```

---

### Task 3: Tab Navigation in App.svelte

**Files:**
- Modify: `frontend/src/App.svelte`
- Modify: `frontend/src/App.test.js`

**Step 1: Write failing tests for tab navigation**

Add these tests to `frontend/src/App.test.js` (keep all existing tests, add new `describe` block):

```javascript
describe('tab navigation', () => {
  it('renders three tabs: Sites, Services, Settings', async () => {
    ListSites.mockResolvedValue([]);
    DatabaseServices.mockResolvedValue([]);
    const { getByRole } = render(App);
    await vi.waitFor(() => {
      expect(getByRole('tab', { name: 'Sites' })).toBeTruthy();
      expect(getByRole('tab', { name: 'Services' })).toBeTruthy();
      expect(getByRole('tab', { name: 'Settings' })).toBeTruthy();
    });
  });

  it('shows Sites tab as active by default', async () => {
    ListSites.mockResolvedValue([]);
    DatabaseServices.mockResolvedValue([]);
    const { getByRole } = render(App);
    await vi.waitFor(() => {
      expect(getByRole('tab', { name: 'Sites' }).classList.contains('tab-active')).toBe(true);
    });
  });

  it('switches to Services tab on click', async () => {
    ListSites.mockResolvedValue([]);
    DatabaseServices.mockResolvedValue([]);
    const { getByRole } = render(App);
    await vi.waitFor(() => {
      expect(getByRole('tab', { name: 'Services' })).toBeTruthy();
    });
    await fireEvent.click(getByRole('tab', { name: 'Services' }));
    expect(getByRole('tab', { name: 'Services' }).classList.contains('tab-active')).toBe(true);
    expect(getByRole('tab', { name: 'Sites' }).classList.contains('tab-active')).toBe(false);
  });

  it('switches to Settings tab on click', async () => {
    ListSites.mockResolvedValue([]);
    DatabaseServices.mockResolvedValue([]);
    const { getByRole } = render(App);
    await vi.waitFor(() => {
      expect(getByRole('tab', { name: 'Settings' })).toBeTruthy();
    });
    await fireEvent.click(getByRole('tab', { name: 'Settings' }));
    expect(getByRole('tab', { name: 'Settings' }).classList.contains('tab-active')).toBe(true);
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/App.test.js`
Expected: FAIL — tab roles not found

**Step 3: Update App.svelte with tab navigation layout**

Replace the `<main>` section in `frontend/src/App.svelte` with the tabbed layout. The script section needs a new `activeTab` variable and import for `initTheme`. The full updated component:

```svelte
<script>
  import { onMount } from 'svelte';
  import { ListSites, AddSite, RemoveSite, DatabaseServices, StartDatabase, StopDatabase } from '../wailsjs/go/main/App.js';
  import { notifySuccess, notifyError, dismissLatest } from './lib/notifications.js';
  import { friendlyError } from './lib/errorMessages.js';
  import { initTheme } from './lib/theme.js';
  import SiteList from './SiteList.svelte';
  import AddSiteForm from './AddSiteForm.svelte';
  import ServiceList from './ServiceList.svelte';
  import ToastContainer from './lib/ToastContainer.svelte';

  let sites = [];
  let services = [];
  let sitesLoaded = false;
  let servicesLoaded = false;
  let addFormOpen = false;
  let addSiteForm;
  let activeTab = 'sites';

  function handleKeydown(e) {
    if (e.ctrlKey && e.key === 'n') {
      e.preventDefault();
      activeTab = 'sites';
      addFormOpen = true;
      setTimeout(() => addSiteForm?.focusPathInput(), 0);
      return;
    }
    if (e.ctrlKey && e.key === 'Enter') {
      if (addFormOpen) {
        e.preventDefault();
        addSiteForm?.handleSubmit();
      }
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

  async function refreshSites() {
    try {
      sites = await ListSites() || [];
    } catch (e) {
      notifyError(friendlyError(e.message || String(e)));
    } finally {
      sitesLoaded = true;
    }
  }

  async function handleAdd(path, domain, phpVersion, nodeVersion, tls) {
    await AddSite(path, domain, phpVersion, nodeVersion, tls);
    await refreshSites();
  }

  async function handleRemove(domain) {
    try {
      await RemoveSite(domain);
      await refreshSites();
      notifySuccess(`Site "${domain}" removed.`);
    } catch (e) {
      notifyError(friendlyError(e.message || String(e)));
    }
  }

  async function refreshServices() {
    try {
      services = await DatabaseServices() || [];
    } catch (e) {
      notifyError(friendlyError(e.message || String(e)));
    } finally {
      servicesLoaded = true;
    }
  }

  async function handleStartService(svc) {
    try {
      await StartDatabase(svc);
      await refreshServices();
      notifySuccess(`${svc} started.`);
    } catch (e) {
      notifyError(friendlyError(e.message || String(e)));
    }
  }

  async function handleStopService(svc) {
    try {
      await StopDatabase(svc);
      await refreshServices();
      notifySuccess(`${svc} stopped.`);
    } catch (e) {
      notifyError(friendlyError(e.message || String(e)));
    }
  }

  onMount(() => {
    initTheme();
    refreshSites();
    refreshServices();
  });
</script>

<svelte:window on:keydown={handleKeydown} />

<main class="h-full flex flex-col">
  <!-- Header with logo and tabs -->
  <header class="bg-base-100 border-b border-base-300 px-6">
    <div class="max-w-5xl mx-auto flex items-center gap-6">
      <div class="flex items-center gap-2 py-3">
        <div class="w-7 h-7 bg-primary rounded-lg flex items-center justify-center">
          <span class="text-primary-content text-sm font-bold">F</span>
        </div>
        <span class="font-bold text-base-content">Rook</span>
      </div>
      <div role="tablist" class="tabs tabs-bordered flex-1">
        <button role="tab" class="tab" class:tab-active={activeTab === 'sites'} on:click={() => activeTab = 'sites'}>Sites</button>
        <button role="tab" class="tab" class:tab-active={activeTab === 'services'} on:click={() => activeTab = 'services'}>Services</button>
        <button role="tab" class="tab" class:tab-active={activeTab === 'settings'} on:click={() => activeTab = 'settings'}>Settings</button>
      </div>
    </div>
  </header>

  <!-- Tab Content -->
  <div class="flex-1 overflow-auto">
    <div class="max-w-5xl mx-auto px-6 py-6">
      {#if activeTab === 'sites'}
        <SiteList {sites} loaded={sitesLoaded} onRemove={handleRemove} on:addsite={() => { addFormOpen = true; }} />
        <AddSiteForm bind:this={addSiteForm} onAdd={handleAdd} bind:collapseOpen={addFormOpen} />
      {:else if activeTab === 'services'}
        <ServiceList {services} loaded={servicesLoaded} onStart={handleStartService} onStop={handleStopService} />
      {:else if activeTab === 'settings'}
        <p class="text-base-content/70">Settings coming in next task.</p>
      {/if}
    </div>
  </div>
</main>

<ToastContainer />
```

Note: The old `<section class="card bg-base-200 p-6">` wrappers and `<header>` with `<h1>Rook</h1>` are removed. The Sites/Services headings were in the old card sections — those will be handled in later tasks when we add section headers.

**Step 4: Update existing tests that rely on old layout**

Some existing tests may query for elements that were in the old layout (e.g., the "Rook" heading as `<h1>`). Update any selectors that break. The key change: the sites and services sections are no longer both visible at once — tests that check for both will need the tab to be switched.

Review each failing test and fix selectors. Common fixes:
- Old `<h1>Rook</h1>` is now a `<span>` in the header — update any queries for it
- Services-related tests need `activeTab = 'services'` — but since services tests are in `ServiceList.test.js` (rendered independently), they should still work
- `App.test.js` tests that check for service behavior may need to click the Services tab first

**Step 5: Run all tests to verify**

Run: `cd frontend && npm test`
Expected: All pass

**Step 6: Commit**

```bash
git add frontend/src/App.svelte frontend/src/App.test.js
git commit -m "feat: add tab navigation (Sites, Services, Settings) with theme init"
```

---

### Task 4: Settings Tab Component

**Files:**
- Create: `frontend/src/SettingsTab.svelte`
- Test: `frontend/src/SettingsTab.test.js`
- Modify: `frontend/src/App.svelte` (import and use SettingsTab)

**Step 1: Write failing tests**

Create `frontend/src/SettingsTab.test.js`:

```javascript
import { describe, it, expect, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SettingsTab from './SettingsTab.svelte';
import { theme } from './lib/theme.js';
import { get } from 'svelte/store';

describe('SettingsTab', () => {
  beforeEach(() => {
    localStorage.clear();
    theme.set('light');
    document.documentElement.setAttribute('data-theme', 'light');
  });

  it('renders Appearance section', () => {
    const { getByText } = render(SettingsTab);
    expect(getByText('Appearance')).toBeTruthy();
  });

  it('renders theme toggle', () => {
    const { getByRole } = render(SettingsTab);
    expect(getByRole('checkbox', { name: /dark mode/i })).toBeTruthy();
  });

  it('toggles theme on checkbox change', async () => {
    const { getByRole } = render(SettingsTab);
    const toggle = getByRole('checkbox', { name: /dark mode/i });
    await fireEvent.click(toggle);
    expect(get(theme)).toBe('dark');
  });

  it('renders About section', () => {
    const { getByText } = render(SettingsTab);
    expect(getByText('About')).toBeTruthy();
  });

  it('shows app version', () => {
    const { getByText } = render(SettingsTab);
    expect(getByText(/version/i)).toBeTruthy();
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/SettingsTab.test.js`
Expected: FAIL — module not found

**Step 3: Write SettingsTab component**

Create `frontend/src/SettingsTab.svelte`:

```svelte
<script>
  import { theme, toggleTheme } from './lib/theme.js';
</script>

<div class="space-y-6">
  <div class="card bg-base-200 p-6">
    <h3 class="font-semibold text-base-content mb-4">Appearance</h3>
    <label class="flex items-center justify-between cursor-pointer">
      <div>
        <span class="font-medium text-base-content">Dark mode</span>
        <p class="text-sm text-base-content/60">Switch between light and dark themes</p>
      </div>
      <input
        type="checkbox"
        class="toggle toggle-primary"
        checked={$theme === 'dark'}
        on:change={toggleTheme}
        aria-label="Dark mode"
      />
    </label>
  </div>

  <div class="card bg-base-200 p-6">
    <h3 class="font-semibold text-base-content mb-4">About</h3>
    <div class="space-y-2 text-sm">
      <div class="flex justify-between">
        <span class="text-base-content/70">Version</span>
        <span class="font-medium text-base-content">1.0.0</span>
      </div>
    </div>
  </div>
</div>
```

**Step 4: Wire SettingsTab into App.svelte**

In `frontend/src/App.svelte`, add the import and replace the placeholder:

Add import: `import SettingsTab from './SettingsTab.svelte';`

Replace `<p class="text-base-content/70">Settings coming in next task.</p>` with `<SettingsTab />`.

**Step 5: Run tests**

Run: `cd frontend && npm test`
Expected: All pass

**Step 6: Commit**

```bash
git add frontend/src/SettingsTab.svelte frontend/src/SettingsTab.test.js frontend/src/App.svelte
git commit -m "feat: add Settings tab with dark mode toggle and About section"
```

---

### Task 5: SiteCard Component

**Files:**
- Create: `frontend/src/SiteCard.svelte`
- Test: `frontend/src/SiteCard.test.js`

**Step 1: Write failing tests**

Create `frontend/src/SiteCard.test.js`:

```javascript
import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SiteCard from './SiteCard.svelte';

const mockSite = {
  domain: 'myapp.test',
  path: '/home/user/projects/myapp',
  php_version: '8.3',
  node_version: '20',
  tls: true,
};

describe('SiteCard', () => {
  it('renders site domain', () => {
    const { getByText } = render(SiteCard, { props: { site: mockSite, onRemove: vi.fn() } });
    expect(getByText('myapp.test')).toBeTruthy();
  });

  it('renders site path truncated', () => {
    const { container } = render(SiteCard, { props: { site: mockSite, onRemove: vi.fn() } });
    expect(container.textContent).toContain('/home/user/projects/myapp');
  });

  it('renders PHP version badge', () => {
    const { getByText } = render(SiteCard, { props: { site: mockSite, onRemove: vi.fn() } });
    expect(getByText('PHP 8.3')).toBeTruthy();
  });

  it('renders Node version badge', () => {
    const { getByText } = render(SiteCard, { props: { site: mockSite, onRemove: vi.fn() } });
    expect(getByText('Node 20')).toBeTruthy();
  });

  it('renders TLS badge when enabled', () => {
    const { getByText } = render(SiteCard, { props: { site: mockSite, onRemove: vi.fn() } });
    expect(getByText('TLS')).toBeTruthy();
  });

  it('does not render TLS badge when disabled', () => {
    const site = { ...mockSite, tls: false };
    const { queryByText } = render(SiteCard, { props: { site, onRemove: vi.fn() } });
    expect(queryByText('TLS')).toBeNull();
  });

  it('calls onRemove with domain on remove button click', async () => {
    const onRemove = vi.fn();
    const { getByTitle } = render(SiteCard, { props: { site: mockSite, onRemove } });
    await fireEvent.click(getByTitle('Remove site'));
    expect(onRemove).toHaveBeenCalledWith('myapp.test');
  });

  it('handles missing php_version gracefully', () => {
    const site = { ...mockSite, php_version: '' };
    const { queryByText } = render(SiteCard, { props: { site, onRemove: vi.fn() } });
    expect(queryByText(/PHP/)).toBeNull();
  });

  it('handles missing node_version gracefully', () => {
    const site = { ...mockSite, node_version: '' };
    const { queryByText } = render(SiteCard, { props: { site, onRemove: vi.fn() } });
    expect(queryByText(/Node/)).toBeNull();
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/SiteCard.test.js`
Expected: FAIL — module not found

**Step 3: Write SiteCard component**

Create `frontend/src/SiteCard.svelte`:

```svelte
<script>
  export let site;
  export let onRemove = () => {};

  const langBadgeClass = {
    php: 'badge-primary',
    node: 'badge-success',
  };

  $: phpBadge = site.php_version ? `PHP ${site.php_version}` : '';
  $: nodeBadge = site.node_version ? `Node ${site.node_version}` : '';
</script>

<div class="card bg-base-200 p-5 hover:shadow-md transition-shadow">
  <div class="flex items-start justify-between mb-3">
    <div>
      <h3 class="font-semibold text-base-content">{site.domain}</h3>
      <p class="text-sm text-base-content/60 truncate max-w-[220px]" title={site.path}>{site.path}</p>
    </div>
    <button
      class="btn btn-ghost btn-sm btn-square hover:btn-error"
      title="Remove site"
      on:click={() => onRemove(site.domain)}
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>
    </button>
  </div>

  <div class="flex flex-wrap gap-1.5">
    {#if phpBadge}
      <span class="badge badge-sm badge-primary">{phpBadge}</span>
    {/if}
    {#if nodeBadge}
      <span class="badge badge-sm badge-success">{nodeBadge}</span>
    {/if}
    {#if site.tls}
      <span class="badge badge-sm badge-info">TLS</span>
    {/if}
  </div>
</div>
```

**Step 4: Run tests**

Run: `cd frontend && npx vitest run src/SiteCard.test.js`
Expected: All pass

**Step 5: Commit**

```bash
git add frontend/src/SiteCard.svelte frontend/src/SiteCard.test.js
git commit -m "feat: add SiteCard component with language badges and remove action"
```

---

### Task 6: Update SiteList with View Toggle + Cards + Search

**Files:**
- Modify: `frontend/src/SiteList.svelte`
- Modify: `frontend/src/SiteList.test.js`

**Step 1: Write failing tests for new features**

Add to `frontend/src/SiteList.test.js`:

```javascript
describe('view toggle', () => {
  it('renders view toggle buttons', () => {
    const { getByTitle } = render(SiteList, { props: { sites: mockSites, onRemove: vi.fn() } });
    expect(getByTitle('Table view')).toBeTruthy();
    expect(getByTitle('Card view')).toBeTruthy();
  });

  it('shows table view by default', () => {
    const { container } = render(SiteList, { props: { sites: mockSites, onRemove: vi.fn() } });
    expect(container.querySelector('table')).toBeTruthy();
  });

  it('switches to card view on click', async () => {
    const { getByTitle, container } = render(SiteList, { props: { sites: mockSites, onRemove: vi.fn() } });
    await fireEvent.click(getByTitle('Card view'));
    expect(container.querySelector('table')).toBeNull();
    expect(container.querySelector('[data-testid="site-cards"]')).toBeTruthy();
  });
});

describe('search', () => {
  it('renders search input', () => {
    const { getByPlaceholderText } = render(SiteList, { props: { sites: mockSites, onRemove: vi.fn() } });
    expect(getByPlaceholderText(/search/i)).toBeTruthy();
  });

  it('filters sites by domain', async () => {
    const sites = [
      { domain: 'foo.test', path: '/foo', php_version: '', node_version: '', tls: false },
      { domain: 'bar.test', path: '/bar', php_version: '', node_version: '', tls: false },
    ];
    const { getByPlaceholderText, queryByText } = render(SiteList, { props: { sites, onRemove: vi.fn() } });
    await fireEvent.input(getByPlaceholderText(/search/i), { target: { value: 'foo' } });
    expect(queryByText('foo.test')).toBeTruthy();
    expect(queryByText('bar.test')).toBeNull();
  });
});
```

Where `mockSites` is defined at the top of the test file (use existing test data or add):
```javascript
const mockSites = [
  { domain: 'app.test', path: '/home/user/app', php_version: '8.3', node_version: '', tls: true },
  { domain: 'api.test', path: '/home/user/api', php_version: '', node_version: '20', tls: false },
];
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/SiteList.test.js`
Expected: FAIL — view toggle buttons not found

**Step 3: Update SiteList.svelte**

Update `frontend/src/SiteList.svelte` to include:
- A header with "Sites" title, search input, view toggle (table/card icons), and "Add Site" button
- Search filtering (by domain and path)
- Conditional rendering: table view (existing table, refreshed) or card view (grid of SiteCard components)
- View preference persisted to localStorage
- Import SiteCard

The component should:
- Add `let searchQuery = ''` and `let viewMode` initialized from `localStorage.getItem('rook-view') || 'table'`
- Filter sites: `$: filtered = sites.filter(s => s.domain.toLowerCase().includes(searchQuery.toLowerCase()) || s.path.toLowerCase().includes(searchQuery.toLowerCase()))`
- Toggle between table and card grid with `viewMode`
- Save viewMode to localStorage on change
- Move "Add Site" button into the header (dispatches `addsite` event)
- In card mode, render `<div data-testid="site-cards" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">` with `<SiteCard>` for each site

**Step 4: Run tests**

Run: `cd frontend && npm test`
Expected: All pass (existing + new)

**Step 5: Commit**

```bash
git add frontend/src/SiteList.svelte frontend/src/SiteList.test.js
git commit -m "feat: add search, view toggle (table/cards), and section header to SiteList"
```

---

### Task 7: Add Site Modal (Replace Collapsible Form)

**Files:**
- Modify: `frontend/src/AddSiteForm.svelte` (convert to modal)
- Modify: `frontend/src/AddSiteForm.test.js`
- Modify: `frontend/src/App.svelte` (update usage)

**Step 1: Write failing tests for modal behavior**

Update tests in `frontend/src/AddSiteForm.test.js` to expect a modal instead of a collapsible. Key changes:
- The form should render inside a `modal modal-open` when `open` is true
- The form should not be visible when `open` is false
- A cancel button should close the modal
- Form submission should still work the same way

Add/update tests:

```javascript
describe('AddSiteForm modal', () => {
  it('does not render form when open is false', () => {
    const { container } = render(AddSiteForm, { props: { onAdd: vi.fn(), open: false } });
    expect(container.querySelector('.modal-open')).toBeNull();
  });

  it('renders form inside modal when open is true', () => {
    const { container } = render(AddSiteForm, { props: { onAdd: vi.fn(), open: true } });
    expect(container.querySelector('.modal-open')).toBeTruthy();
  });

  it('renders cancel button that dispatches close', async () => {
    const { getByText, component } = render(AddSiteForm, { props: { onAdd: vi.fn(), open: true } });
    const handler = vi.fn();
    component.$on('close', handler);
    await fireEvent.click(getByText('Cancel'));
    expect(handler).toHaveBeenCalled();
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/AddSiteForm.test.js`
Expected: FAIL — old collapsible structure

**Step 3: Convert AddSiteForm to modal**

Update `frontend/src/AddSiteForm.svelte`:
- Replace `export let collapseOpen` with `export let open = false`
- Replace the `collapse` wrapper with DaisyUI `modal` structure (similar to ConfirmModal)
- Add a Cancel button that dispatches a `close` event
- Add focus management (focus first input when modal opens)
- Keep all existing form fields and submission logic
- The title should be "Add Site"

**Step 4: Update App.svelte to use modal pattern**

In `frontend/src/App.svelte`:
- Change `<AddSiteForm bind:collapseOpen={addFormOpen}>` to `<AddSiteForm open={addFormOpen} on:close={() => addFormOpen = false}>`
- Update `handleSubmit` success path: form dispatches `close` event instead of setting `collapseOpen = false`

**Step 5: Run tests**

Run: `cd frontend && npm test`
Expected: All pass

**Step 6: Commit**

```bash
git add frontend/src/AddSiteForm.svelte frontend/src/AddSiteForm.test.js frontend/src/App.svelte
git commit -m "feat: convert AddSiteForm from collapsible to modal dialog"
```

---

### Task 8: ServiceCard Component + Update ServiceList

**Files:**
- Create: `frontend/src/ServiceCard.svelte`
- Test: `frontend/src/ServiceCard.test.js`
- Modify: `frontend/src/ServiceList.svelte`
- Modify: `frontend/src/ServiceList.test.js`

**Step 1: Write failing tests for ServiceCard**

Create `frontend/src/ServiceCard.test.js`:

```javascript
import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import ServiceCard from './ServiceCard.svelte';

const serviceInfo = {
  mysql: { emoji: '\uD83D\uDCBE', name: 'MySQL', description: 'Relational database management system' },
  postgres: { emoji: '\uD83D\uDC18', name: 'PostgreSQL', description: 'Advanced open source database' },
  redis: { emoji: '\uD83D\uDD34', name: 'Redis', description: 'In-memory data structure store' },
};

describe('ServiceCard', () => {
  const mockService = { type: 'mysql', enabled: true, running: true, autostart: true, port: 3306 };

  it('renders service name', () => {
    const { getByText } = render(ServiceCard, { props: { service: mockService, onStart: vi.fn(), onStop: vi.fn() } });
    expect(getByText('MySQL')).toBeTruthy();
  });

  it('renders running badge when running', () => {
    const { getByText } = render(ServiceCard, { props: { service: mockService, onStart: vi.fn(), onStop: vi.fn() } });
    expect(getByText('Running')).toBeTruthy();
  });

  it('renders stopped badge when not running', () => {
    const svc = { ...mockService, running: false };
    const { getByText } = render(ServiceCard, { props: { service: svc, onStart: vi.fn(), onStop: vi.fn() } });
    expect(getByText('Stopped')).toBeTruthy();
  });

  it('renders port number', () => {
    const { getByText } = render(ServiceCard, { props: { service: mockService, onStart: vi.fn(), onStop: vi.fn() } });
    expect(getByText('3306')).toBeTruthy();
  });

  it('calls onStop when stop button clicked for running service', async () => {
    const onStop = vi.fn();
    const { getByTitle } = render(ServiceCard, { props: { service: mockService, onStart: vi.fn(), onStop } });
    await fireEvent.click(getByTitle('Stop service'));
    expect(onStop).toHaveBeenCalledWith('mysql');
  });

  it('calls onStart when start button clicked for stopped service', async () => {
    const svc = { ...mockService, running: false };
    const onStart = vi.fn();
    const { getByTitle } = render(ServiceCard, { props: { service: svc, onStart, onStop: vi.fn() } });
    await fireEvent.click(getByTitle('Start service'));
    expect(onStart).toHaveBeenCalledWith('mysql');
  });

  it('shows dimmed state for disabled service', () => {
    const svc = { ...mockService, enabled: false };
    const { container } = render(ServiceCard, { props: { service: svc, onStart: vi.fn(), onStop: vi.fn() } });
    expect(container.querySelector('.opacity-50')).toBeTruthy();
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/ServiceCard.test.js`
Expected: FAIL — module not found

**Step 3: Write ServiceCard component**

Create `frontend/src/ServiceCard.svelte`:

```svelte
<script>
  export let service;
  export let onStart = () => {};
  export let onStop = () => {};

  let loading = false;

  const serviceInfo = {
    mysql: { emoji: '\uD83D\uDCBE', name: 'MySQL', description: 'Relational database management system' },
    postgres: { emoji: '\uD83D\uDC18', name: 'PostgreSQL', description: 'Advanced open source database' },
    redis: { emoji: '\uD83D\uDD34', name: 'Redis', description: 'In-memory data structure store' },
  };

  $: info = serviceInfo[service.type] || { emoji: '\uD83D\uDCE6', name: service.type, description: '' };

  async function handleToggle() {
    loading = true;
    try {
      if (service.running) {
        await onStop(service.type);
      } else {
        await onStart(service.type);
      }
    } finally {
      loading = false;
    }
  }
</script>

<div class="card bg-base-200 p-5 hover:shadow-md transition-shadow" class:opacity-50={!service.enabled}>
  <div class="flex items-start gap-4">
    <span class="text-3xl">{info.emoji}</span>
    <div class="flex-1 min-w-0">
      <div class="flex items-center justify-between mb-1">
        <div class="flex items-center gap-2">
          <h3 class="font-semibold text-base-content">{info.name}</h3>
          {#if !service.enabled}
            <span class="badge badge-sm badge-warning">Not installed</span>
          {:else if service.running}
            <span class="badge badge-sm badge-success">Running</span>
          {:else}
            <span class="badge badge-sm badge-ghost">Stopped</span>
          {/if}
        </div>
        {#if service.enabled}
          <button
            class="btn btn-ghost btn-sm btn-square"
            class:hover:btn-error={service.running}
            class:hover:btn-success={!service.running}
            title={service.running ? 'Stop service' : 'Start service'}
            disabled={loading}
            on:click={handleToggle}
          >
            {#if loading}
              <span class="loading loading-spinner loading-xs"></span>
            {:else if service.running}
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16" rx="1"/><rect x="14" y="4" width="4" height="16" rx="1"/></svg>
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>
            {/if}
          </button>
        {/if}
      </div>
      <p class="text-sm text-base-content/60">{info.description}</p>
      <div class="flex items-center gap-4 mt-3 text-sm">
        <div>
          <span class="text-base-content/50">Port:</span>
          <span class="font-medium text-base-content ml-1">{service.port}</span>
        </div>
        {#if service.autostart}
          <span class="badge badge-sm badge-outline">Auto-start</span>
        {/if}
      </div>
    </div>
  </div>
</div>
```

**Step 4: Run ServiceCard tests**

Run: `cd frontend && npx vitest run src/ServiceCard.test.js`
Expected: All pass

**Step 5: Update ServiceList.svelte to use cards**

Update `frontend/src/ServiceList.svelte`:
- Add a header with "Services" title, subtitle, and running count
- Replace the table with a grid of ServiceCard components
- Keep the loading skeleton (change from table rows to card skeletons)
- Keep the empty state
- Import ServiceCard

**Step 6: Update ServiceList tests**

Update `frontend/src/ServiceList.test.js` to match new card-based layout. Key changes:
- Remove queries for `<table>`, `<th>`, `<td>` elements
- Update to query for card-based content (service names, badges)
- Start/Stop button queries change from text buttons to icon buttons with titles

**Step 7: Run all tests**

Run: `cd frontend && npm test`
Expected: All pass

**Step 8: Commit**

```bash
git add frontend/src/ServiceCard.svelte frontend/src/ServiceCard.test.js frontend/src/ServiceList.svelte frontend/src/ServiceList.test.js
git commit -m "feat: add ServiceCard component and convert ServiceList from table to cards"
```

---

### Task 9: Visual Polish — Language Badges in Table, Status Dots

**Files:**
- Modify: `frontend/src/SiteList.svelte` (table styling updates)

**Step 1: Update table view styling**

In the table view of `frontend/src/SiteList.svelte`, update the PHP and Node columns to use colored badges (same badge classes as SiteCard) instead of plain text. Update TLS column to show a green dot (●) or gray dash.

**Step 2: Run all tests**

Run: `cd frontend && npm test`
Expected: All pass

**Step 3: Commit**

```bash
git add frontend/src/SiteList.svelte
git commit -m "style: add colored language badges and TLS status dots to site table"
```

---

### Task 10: Final Integration Test + Cleanup

**Files:**
- Modify: `frontend/src/App.test.js` (add integration tests)
- Review all files for consistency

**Step 1: Add integration tests**

Add tests to `frontend/src/App.test.js` that verify:
- Tab switching works end-to-end (click Services tab, see service cards)
- Ctrl+N shortcut opens Add Site modal (not collapsible)
- Settings tab renders theme toggle
- Theme toggle changes `data-theme` attribute

**Step 2: Run all tests**

Run: `cd frontend && npm test`
Expected: All pass

**Step 3: Review for consistency**

Check all components for:
- Consistent use of `bg-base-200` for cards (not `bg-base-100` or hardcoded colors)
- Consistent badge styling across SiteCard, SiteList table, ServiceCard
- No leftover references to old green `#66cc99` color
- No leftover collapsible form references

**Step 4: Commit**

```bash
git add -A
git commit -m "test: add integration tests and clean up UI redesign"
```

---

## Task Dependency Order

```
Task 1 (Theme CSS) → Task 2 (Theme Store) → Task 3 (Tab Navigation) → Task 4 (Settings Tab)
                                                                      → Task 5 (SiteCard)
                                                                      → Task 6 (SiteList Update) [depends on Task 5]
                                                                      → Task 7 (Add Site Modal)
                                                                      → Task 8 (ServiceCard + ServiceList)
                                                                      → Task 9 (Visual Polish) [depends on Task 6]
                                                                      → Task 10 (Integration) [depends on all above]
```

Tasks 4-8 can be worked in any order after Task 3 (they're independent). Task 9 depends on Task 6. Task 10 is the final sweep.
