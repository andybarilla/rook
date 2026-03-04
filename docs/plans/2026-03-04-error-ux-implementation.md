# Error UX Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add dismissable toast notifications, loading states, skeleton loading, and friendly error messages to the Flock frontend.

**Architecture:** Frontend-only changes using a shared Svelte writable store for notifications, a ToastContainer component rendering DaisyUI toasts, per-component loading booleans with disabled buttons/spinners, and a simple error message mapper. See `docs/plans/2026-03-04-error-ux-design.md` for full design.

**Tech Stack:** Svelte 3, DaisyUI 5.5, Tailwind CSS 4, Vitest (new dev dependency)

---

### Task 1: Set Up Frontend Testing (Vitest)

No test infrastructure exists in the frontend. Set up Vitest with Svelte support.

**Files:**
- Modify: `frontend/package.json`
- Create: `frontend/vitest.config.js`

**Step 1: Install vitest and testing dependencies**

Run:
```bash
cd frontend && npm install --save-dev vitest @testing-library/svelte @testing-library/jest-dom jsdom
```

**Step 2: Create vitest config**

Create `frontend/vitest.config.js`:
```js
import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte({ hot: !process.env.VITEST })],
  test: {
    environment: 'jsdom',
    globals: true,
  },
});
```

**Step 3: Add test script to package.json**

Add to `frontend/package.json` scripts:
```json
"test": "vitest run",
"test:watch": "vitest"
```

**Step 4: Verify vitest runs (no tests yet, should exit clean)**

Run: `cd frontend && npm test`
Expected: Exit 0, no tests found

**Step 5: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/vitest.config.js
git commit -m "chore: add vitest testing infrastructure to frontend"
```

---

### Task 2: Notification Store

**Files:**
- Create: `frontend/src/lib/notifications.js`
- Create: `frontend/src/lib/notifications.test.js`

**Step 1: Write the failing tests**

Create `frontend/src/lib/notifications.test.js`:
```js
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { notifications, notifySuccess, notifyError, notifyInfo, dismiss } from './notifications.js';
import { get } from 'svelte/store';

beforeEach(() => {
  // Clear all notifications before each test
  notifications.set([]);
});

describe('notifications store', () => {
  it('starts empty', () => {
    expect(get(notifications)).toEqual([]);
  });

  it('notifySuccess adds a success notification', () => {
    notifySuccess('Site added');
    const items = get(notifications);
    expect(items).toHaveLength(1);
    expect(items[0].type).toBe('success');
    expect(items[0].message).toBe('Site added');
    expect(items[0].timeout).toBe(3000);
  });

  it('notifyError adds an error notification', () => {
    notifyError('Something failed');
    const items = get(notifications);
    expect(items).toHaveLength(1);
    expect(items[0].type).toBe('error');
    expect(items[0].message).toBe('Something failed');
    expect(items[0].timeout).toBe(8000);
  });

  it('notifyInfo adds an info notification', () => {
    notifyInfo('Heads up');
    const items = get(notifications);
    expect(items).toHaveLength(1);
    expect(items[0].type).toBe('info');
    expect(items[0].timeout).toBe(3000);
  });

  it('dismiss removes a notification by id', () => {
    notifySuccess('one');
    notifyError('two');
    const items = get(notifications);
    dismiss(items[0].id);
    expect(get(notifications)).toHaveLength(1);
    expect(get(notifications)[0].message).toBe('two');
  });

  it('each notification gets a unique id', () => {
    notifySuccess('a');
    notifySuccess('b');
    const items = get(notifications);
    expect(items[0].id).not.toBe(items[1].id);
  });
});
```

**Step 2: Run tests to verify they fail**

Run: `cd frontend && npm test`
Expected: FAIL — cannot find module `./notifications.js`

**Step 3: Write minimal implementation**

Create `frontend/src/lib/notifications.js`:
```js
import { writable } from 'svelte/store';

export const notifications = writable([]);

let nextId = 1;

function addNotification(type, message, timeout) {
  const id = nextId++;
  notifications.update(n => [...n, { id, type, message, timeout }]);
  return id;
}

export function notifySuccess(message) {
  return addNotification('success', message, 3000);
}

export function notifyError(message) {
  return addNotification('error', message, 8000);
}

export function notifyInfo(message) {
  return addNotification('info', message, 3000);
}

export function dismiss(id) {
  notifications.update(n => n.filter(item => item.id !== id));
}
```

**Step 4: Run tests to verify they pass**

Run: `cd frontend && npm test`
Expected: All 6 tests PASS

**Step 5: Commit**

```bash
git add frontend/src/lib/notifications.js frontend/src/lib/notifications.test.js
git commit -m "feat: add notification store with success/error/info helpers"
```

---

### Task 3: Friendly Error Messages

**Files:**
- Create: `frontend/src/lib/errorMessages.js`
- Create: `frontend/src/lib/errorMessages.test.js`

**Step 1: Write the failing tests**

Create `frontend/src/lib/errorMessages.test.js`:
```js
import { describe, it, expect } from 'vitest';
import { friendlyError } from './errorMessages.js';

describe('friendlyError', () => {
  it('maps "is not a directory" errors', () => {
    expect(friendlyError('path "/tmp/foo" is not a directory'))
      .toBe('The selected path is not a valid directory.');
  });

  it('maps "already registered" errors', () => {
    expect(friendlyError('domain "myapp.test" is already registered'))
      .toBe('A site with domain "myapp.test" already exists.');
  });

  it('maps "not found" domain errors', () => {
    expect(friendlyError('domain "myapp.test" not found'))
      .toBe('Could not find site "myapp.test".');
  });

  it('passes through unrecognized errors unchanged', () => {
    expect(friendlyError('something unexpected happened'))
      .toBe('something unexpected happened');
  });

  it('handles empty string', () => {
    expect(friendlyError('')).toBe('');
  });
});
```

**Step 2: Run tests to verify they fail**

Run: `cd frontend && npm test`
Expected: FAIL — cannot find module `./errorMessages.js`

**Step 3: Write minimal implementation**

Create `frontend/src/lib/errorMessages.js`:
```js
const patterns = [
  {
    match: /is not a directory/,
    message: () => 'The selected path is not a valid directory.',
  },
  {
    match: /domain "([^"]+)" is already registered/,
    message: (m) => `A site with domain "${m[1]}" already exists.`,
  },
  {
    match: /domain "([^"]+)" not found/,
    message: (m) => `Could not find site "${m[1]}".`,
  },
];

export function friendlyError(raw) {
  if (!raw) return raw;
  for (const { match, message } of patterns) {
    const m = raw.match(match);
    if (m) return message(m);
  }
  return raw;
}
```

**Step 4: Run tests to verify they pass**

Run: `cd frontend && npm test`
Expected: All 5 tests PASS

**Step 5: Commit**

```bash
git add frontend/src/lib/errorMessages.js frontend/src/lib/errorMessages.test.js
git commit -m "feat: add friendly error message mapper"
```

---

### Task 4: ToastContainer Component

**Files:**
- Create: `frontend/src/lib/ToastContainer.svelte`

**Step 1: Create the ToastContainer component**

Create `frontend/src/lib/ToastContainer.svelte`:
```svelte
<script>
  import { notifications, dismiss } from './notifications.js';
  import { fade } from 'svelte/transition';
  import { onDestroy } from 'svelte';

  let timers = {};

  function scheduleAutoDismiss(item) {
    if (timers[item.id]) return;
    timers[item.id] = setTimeout(() => {
      dismiss(item.id);
      delete timers[item.id];
    }, item.timeout);
  }

  $: $notifications.forEach(item => scheduleAutoDismiss(item));

  onDestroy(() => {
    Object.values(timers).forEach(clearTimeout);
  });

  const alertClass = {
    success: 'alert-success',
    error: 'alert-error',
    info: 'alert-info',
    warning: 'alert-warning',
  };
</script>

<div class="toast toast-end toast-bottom z-50">
  {#each $notifications as item (item.id)}
    <div class="alert {alertClass[item.type] || 'alert-info'} text-sm shadow-lg" transition:fade={{ duration: 200 }}>
      <span>{item.message}</span>
      <button class="btn btn-ghost btn-xs" on:click={() => dismiss(item.id)}>✕</button>
    </div>
  {/each}
</div>
```

**Step 2: Verify the component compiles**

Run: `cd frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds (component is not yet mounted, but should compile)

Note: We'll do a manual smoke test after wiring it into App.svelte in Task 5.

**Step 3: Commit**

```bash
git add frontend/src/lib/ToastContainer.svelte
git commit -m "feat: add ToastContainer component for dismissable notifications"
```

---

### Task 5: Wire Up App.svelte — Replace Error Banner with Toasts + Loading

**Files:**
- Modify: `frontend/src/App.svelte`

**Step 1: Update App.svelte**

Replace the entire content of `frontend/src/App.svelte` with:
```svelte
<script>
  import { onMount } from 'svelte';
  import { ListSites, AddSite, RemoveSite, DatabaseServices, StartDatabase, StopDatabase } from '../wailsjs/go/main/App.js';
  import { notifySuccess, notifyError } from './lib/notifications.js';
  import { friendlyError } from './lib/errorMessages.js';
  import SiteList from './SiteList.svelte';
  import AddSiteForm from './AddSiteForm.svelte';
  import ServiceList from './ServiceList.svelte';
  import ToastContainer from './lib/ToastContainer.svelte';

  let sites = [];
  let services = [];
  let sitesLoaded = false;
  let servicesLoaded = false;

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
    refreshSites();
    refreshServices();
  });
</script>

<main class="max-w-3xl mx-auto px-6 py-8 text-left">
  <header class="mb-8">
    <h1 class="text-2xl font-bold m-0">Flock</h1>
    <p class="text-base-content/50 mt-1 text-sm">Local Development Environment</p>
  </header>

  <section class="card bg-base-200 p-6">
    <h2 class="text-sm text-base-content/60 uppercase tracking-wide mb-4 font-semibold">Sites</h2>
    <SiteList {sites} loaded={sitesLoaded} onRemove={handleRemove} />
    <AddSiteForm onAdd={handleAdd} />
  </section>

  <section class="card bg-base-200 p-6 mt-6">
    <h2 class="text-sm text-base-content/60 uppercase tracking-wide mb-4 font-semibold">Services</h2>
    <ServiceList {services} loaded={servicesLoaded} onStart={handleStartService} onStop={handleStopService} />
  </section>
</main>

<ToastContainer />
```

Key changes:
- Removed `error` variable and inline error banner
- Added `sitesLoaded`/`servicesLoaded` flags for skeleton loading
- All catch blocks now use `notifyError(friendlyError(...))` instead of setting error string
- Success operations push `notifySuccess()` toasts
- `handleAdd` still throws to AddSiteForm (it awaits `onAdd` in a try/catch)
- Passes `loaded` prop to SiteList and ServiceList
- Mounts ToastContainer at root

**Step 2: Verify the app builds**

Run: `cd frontend && npx vite build 2>&1 | tail -5`
Expected: Build may warn about unused `loaded` props (not yet accepted by child components) but should succeed

**Step 3: Commit**

```bash
git add frontend/src/App.svelte
git commit -m "feat: replace error banner with toast notifications in App.svelte"
```

---

### Task 6: Update AddSiteForm — Loading State + Toast Notifications

**Files:**
- Modify: `frontend/src/AddSiteForm.svelte`

**Step 1: Update AddSiteForm.svelte**

Replace the entire content of `frontend/src/AddSiteForm.svelte` with:
```svelte
<script>
  import { notifySuccess, notifyError } from './lib/notifications.js';
  import { friendlyError } from './lib/errorMessages.js';

  export let onAdd = () => {};

  let path = '';
  let domain = '';
  let phpVersion = '';
  let nodeVersion = '';
  let tls = false;
  let submitting = false;

  function inferDomain(p) {
    if (!p) return '';
    const parts = p.replace(/[\\/]+$/, '').split(/[\\/]/);
    return (parts[parts.length - 1] || '') + '.test';
  }

  function handlePathInput() {
    if (!domain || domain === inferDomain(path.slice(0, path.length - 1))) {
      domain = inferDomain(path);
    }
  }

  async function handleSubmit() {
    if (!path || !domain) {
      notifyError('Path and domain are required.');
      return;
    }
    submitting = true;
    try {
      await onAdd(path, domain, phpVersion, nodeVersion, tls);
      notifySuccess(`Site "${domain}" added.`);
      path = '';
      domain = '';
      phpVersion = '';
      nodeVersion = '';
      tls = false;
    } catch (e) {
      notifyError(friendlyError(e.message || String(e)));
    } finally {
      submitting = false;
    }
  }
</script>

<form class="mt-6 p-4 border border-base-300 rounded-lg" on:submit|preventDefault={handleSubmit}>
  <div class="flex gap-4 items-end mb-3">
    <label class="flex flex-col flex-1 text-left">
      <span class="text-xs text-base-content/50 uppercase tracking-wide mb-1">Path</span>
      <input type="text" class="input input-bordered input-sm" bind:value={path} on:input={handlePathInput} placeholder="/home/user/projects/myapp" disabled={submitting} />
    </label>
    <label class="flex flex-col flex-1 text-left">
      <span class="text-xs text-base-content/50 uppercase tracking-wide mb-1">Domain</span>
      <input type="text" class="input input-bordered input-sm" bind:value={domain} placeholder="myapp.test" disabled={submitting} />
    </label>
  </div>
  <div class="flex gap-4 items-end mb-3">
    <label class="flex flex-col flex-1 text-left">
      <span class="text-xs text-base-content/50 uppercase tracking-wide mb-1">PHP Version</span>
      <input type="text" class="input input-bordered input-sm" bind:value={phpVersion} placeholder="8.3 (optional)" disabled={submitting} />
    </label>
    <label class="flex flex-col flex-1 text-left">
      <span class="text-xs text-base-content/50 uppercase tracking-wide mb-1">Node Version</span>
      <input type="text" class="input input-bordered input-sm" bind:value={nodeVersion} placeholder="system (optional)" disabled={submitting} />
    </label>
    <label class="flex flex-row items-center gap-2 flex-none whitespace-nowrap">
      <input type="checkbox" class="checkbox checkbox-sm" bind:checked={tls} disabled={submitting} />
      <span class="text-xs text-base-content/50 uppercase tracking-wide">TLS</span>
    </label>
    <button type="submit" class="btn btn-success btn-sm" disabled={submitting}>
      {#if submitting}
        <span class="loading loading-spinner loading-xs"></span>
        Adding…
      {:else}
        Add Site
      {/if}
    </button>
  </div>
</form>
```

Key changes:
- Removed local `error` variable and inline error banner
- Added `submitting` boolean — disables all inputs and shows spinner on button
- Validation error uses `notifyError` toast
- Success uses `notifySuccess` toast
- Backend errors use `friendlyError` mapper

**Step 2: Verify the app builds**

Run: `cd frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add frontend/src/AddSiteForm.svelte
git commit -m "feat: add loading state and toast notifications to AddSiteForm"
```

---

### Task 7: Update SiteList — Skeleton Loading + Remove Button Loading

**Files:**
- Modify: `frontend/src/SiteList.svelte`

**Step 1: Update SiteList.svelte**

Replace the entire content of `frontend/src/SiteList.svelte` with:
```svelte
<script>
  export let sites = [];
  export let loaded = true;
  export let onRemove = () => {};

  let removingDomain = null;

  async function handleRemove(domain) {
    removingDomain = domain;
    try {
      await onRemove(domain);
    } finally {
      removingDomain = null;
    }
  }
</script>

{#if !loaded}
  <table class="table table-zebra">
    <thead>
      <tr>
        <th>Domain</th>
        <th>Path</th>
        <th>PHP</th>
        <th>TLS</th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      {#each Array(3) as _}
        <tr>
          <td><div class="skeleton h-4 w-28"></div></td>
          <td><div class="skeleton h-4 w-40"></div></td>
          <td><div class="skeleton h-4 w-10"></div></td>
          <td><div class="skeleton h-4 w-6"></div></td>
          <td><div class="skeleton h-4 w-16"></div></td>
        </tr>
      {/each}
    </tbody>
  </table>
{:else if sites.length === 0}
  <p class="text-base-content/50 py-8">No sites registered. Add one below.</p>
{:else}
  <table class="table table-zebra">
    <thead>
      <tr>
        <th>Domain</th>
        <th>Path</th>
        <th>PHP</th>
        <th>TLS</th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      {#each sites as site}
        <tr>
          <td class="font-semibold">{site.domain}</td>
          <td class="text-base-content/60 text-sm">{site.path}</td>
          <td>{site.php_version || '—'}</td>
          <td>{site.tls ? '✓' : '—'}</td>
          <td>
            <button
              class="btn btn-ghost btn-sm hover:btn-error"
              disabled={removingDomain === site.domain}
              on:click={() => handleRemove(site.domain)}
            >
              {#if removingDomain === site.domain}
                <span class="loading loading-spinner loading-xs"></span>
              {:else}
                Remove
              {/if}
            </button>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
```

Key changes:
- Added `loaded` prop — shows skeleton rows when `false`
- Added `removingDomain` tracking — disables and shows spinner on the specific Remove button being clicked
- `handleRemove` is now async, wraps `onRemove` with loading state

**Step 2: Verify the app builds**

Run: `cd frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add frontend/src/SiteList.svelte
git commit -m "feat: add skeleton loading and remove button loading to SiteList"
```

---

### Task 8: Update ServiceList — Skeleton Loading + Start/Stop Button Loading

**Files:**
- Modify: `frontend/src/ServiceList.svelte`

**Step 1: Update ServiceList.svelte**

Replace the entire content of `frontend/src/ServiceList.svelte` with:
```svelte
<script>
  export let services = [];
  export let loaded = true;
  export let onStart = () => {};
  export let onStop = () => {};

  let loadingService = null;

  const displayName = {
    mysql: 'MySQL',
    postgres: 'PostgreSQL',
    redis: 'Redis',
  };

  async function handleStart(svc) {
    loadingService = svc;
    try {
      await onStart(svc);
    } finally {
      loadingService = null;
    }
  }

  async function handleStop(svc) {
    loadingService = svc;
    try {
      await onStop(svc);
    } finally {
      loadingService = null;
    }
  }
</script>

{#if !loaded}
  <table class="table table-zebra">
    <thead>
      <tr>
        <th>Service</th>
        <th>Port</th>
        <th>Status</th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      {#each Array(3) as _}
        <tr>
          <td><div class="skeleton h-4 w-24"></div></td>
          <td><div class="skeleton h-4 w-12"></div></td>
          <td><div class="skeleton h-4 w-16"></div></td>
          <td><div class="skeleton h-4 w-14"></div></td>
        </tr>
      {/each}
    </tbody>
  </table>
{:else if services.length === 0}
  <p class="text-base-content/50 py-8">No database services configured.</p>
{:else}
  <table class="table table-zebra">
    <thead>
      <tr>
        <th>Service</th>
        <th>Port</th>
        <th>Status</th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      {#each services as svc}
        <tr class:opacity-50={!svc.enabled}>
          <td class="font-semibold">{displayName[svc.type] || svc.type}</td>
          <td class="text-base-content/60 text-sm">{svc.port}</td>
          <td>
            {#if !svc.enabled}
              <span class="badge badge-warning badge-sm">Not installed</span>
            {:else if svc.running}
              <span class="badge badge-success badge-sm">Running</span>
            {:else}
              <span class="badge badge-ghost badge-sm">Stopped</span>
            {/if}
          </td>
          <td>
            {#if svc.enabled}
              {#if svc.running}
                <button
                  class="btn btn-ghost btn-sm hover:btn-error"
                  disabled={loadingService === svc.type}
                  on:click={() => handleStop(svc.type)}
                >
                  {#if loadingService === svc.type}
                    <span class="loading loading-spinner loading-xs"></span>
                  {:else}
                    Stop
                  {/if}
                </button>
              {:else}
                <button
                  class="btn btn-ghost btn-sm hover:btn-success"
                  disabled={loadingService === svc.type}
                  on:click={() => handleStart(svc.type)}
                >
                  {#if loadingService === svc.type}
                    <span class="loading loading-spinner loading-xs"></span>
                  {:else}
                    Start
                  {/if}
                </button>
              {/if}
            {/if}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
```

Key changes:
- Added `loaded` prop — shows skeleton rows when `false`
- Added `loadingService` — tracks which service has an in-progress operation
- Start/Stop buttons show spinner and are disabled during their operation
- Handlers are now async, wrapping `onStart`/`onStop`

**Step 2: Verify the app builds**

Run: `cd frontend && npx vite build 2>&1 | tail -5`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add frontend/src/ServiceList.svelte
git commit -m "feat: add skeleton loading and button loading to ServiceList"
```

---

### Task 9: Final Verification

**Step 1: Run all tests**

Run: `cd frontend && npm test`
Expected: All tests pass (11 total: 6 notification + 5 errorMessages)

**Step 2: Run full build**

Run: `cd frontend && npx vite build`
Expected: Build succeeds with no errors

**Step 3: Run Go tests to verify no backend regressions**

Run: `go test ./...`
Expected: All Go tests pass

**Step 4: Commit any remaining changes**

If any fixes were needed, commit them.

**Step 5: Update roadmap**

Mark "Error UX" as complete in `docs/ROADMAP.md`:
```markdown
- [x] Error UX (dismissable banners, friendly messages, loading states) — See: docs/plans/2026-03-04-error-ux-design.md
```
