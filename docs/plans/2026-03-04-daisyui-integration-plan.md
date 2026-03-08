# DaisyUI + Tailwind CSS Integration Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace hand-written CSS with DaisyUI + Tailwind CSS for consistent, themeable component styling.

**Architecture:** Install Tailwind CSS via `@tailwindcss/vite` Vite plugin and DaisyUI as a Tailwind plugin. Migrate each Svelte component's hand-written `<style>` blocks to DaisyUI class names in markup. Use DaisyUI's built-in `dark` theme with a custom primary color.

**Tech Stack:** Tailwind CSS (latest), `@tailwindcss/vite`, DaisyUI 5 (CSS-only, framework-agnostic)

**Design doc:** `docs/plans/2026-03-04-daisyui-integration-design.md`

---

### Task 1: Install Dependencies and Configure Tooling

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/vite.config.js`
- Modify: `frontend/src/style.css`

**Step 1: Install packages**

Run from `frontend/` directory:

```bash
cd frontend && npm install -D tailwindcss@latest @tailwindcss/vite@latest daisyui@latest
```

**Step 2: Add Tailwind Vite plugin alongside Svelte plugin**

Update `frontend/vite.config.js`:

```js
import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [svelte(), tailwindcss()]
})
```

**Step 3: Replace style.css with Tailwind + DaisyUI directives**

Replace contents of `frontend/src/style.css` with:

```css
@import "tailwindcss";
@plugin "daisyui" {
  themes: dark --default;
}

@plugin "daisyui/theme" {
  name: "dark";
  default: true;
  --color-primary: oklch(0.72 0.15 160);
  --color-success: oklch(0.72 0.15 160);
}
```

Note: `oklch(0.72 0.15 160)` is the oklch equivalent of `#66cc99` (the existing green accent). The `success` color is also set to match, since DaisyUI's `btn-success` is used for the Add Site button.

Keep the Nunito font-face declaration and body styles:

```css
@import "tailwindcss";
@plugin "daisyui" {
  themes: dark --default;
}

@plugin "daisyui/theme" {
  name: "dark";
  default: true;
  --color-primary: oklch(0.72 0.15 160);
  --color-success: oklch(0.72 0.15 160);
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
```

**Step 4: Verify dev server starts**

Run: `cd frontend && npm run dev`
Expected: Vite dev server starts without errors. Page loads with DaisyUI's dark theme applied (background color will change slightly from the hand-written value).

**Step 5: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/vite.config.js frontend/src/style.css
git commit -m "feat: install Tailwind CSS + DaisyUI and configure dark theme"
```

---

### Task 2: Migrate App.svelte

**Files:**
- Modify: `frontend/src/App.svelte`

**Step 1: Replace markup with DaisyUI classes**

Replace the `<main>` template in `App.svelte`:

```svelte
<main class="max-w-3xl mx-auto p-6 text-left">
  <header class="mb-8">
    <h1 class="text-2xl font-bold">Rook</h1>
    <p class="text-sm text-base-content/60 mt-1">Local Development Environment</p>
  </header>

  {#if error}
    <div class="alert alert-error mb-4 text-sm">
      <span>{error}</span>
      <button class="btn btn-ghost btn-xs" on:click={() => error = ''}>✕</button>
    </div>
  {/if}

  <section class="card bg-base-200 p-6 mb-6">
    <h2 class="text-sm font-semibold uppercase tracking-wide text-base-content/60 mb-4">Sites</h2>
    <SiteList {sites} onRemove={handleRemove} />
    <AddSiteForm onAdd={handleAdd} />
  </section>

  <section class="card bg-base-200 p-6">
    <h2 class="text-sm font-semibold uppercase tracking-wide text-base-content/60 mb-4">Services</h2>
    <ServiceList {services} onStart={handleStartService} onStop={handleStopService} />
  </section>
</main>
```

**Step 2: Remove the entire `<style>` block from App.svelte**

All styling is now handled by DaisyUI/Tailwind classes in the markup. Delete the full `<style>...</style>` section.

**Step 3: Verify in browser**

Run: Open `http://localhost:5173` and verify:
- Header displays correctly
- Error banner (if visible) has DaisyUI alert styling with dismiss button
- Sections have card styling with dark background
- Inline style on Services section is gone (now using `mb-6` on Sites section)

**Step 4: Commit**

```bash
git add frontend/src/App.svelte
git commit -m "feat: migrate App.svelte to DaisyUI classes"
```

---

### Task 3: Migrate SiteList.svelte

**Files:**
- Modify: `frontend/src/SiteList.svelte`

**Step 1: Replace markup with DaisyUI classes**

```svelte
<script>
  export let sites = [];
  export let onRemove = () => {};

  function handleRemove(domain) {
    onRemove(domain);
  }
</script>

{#if sites.length === 0}
  <p class="text-base-content/50 py-8">No sites registered. Add one below.</p>
{:else}
  <div class="overflow-x-auto">
    <table class="table">
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
              <button class="btn btn-ghost btn-xs" on:click={() => handleRemove(site.domain)}>
                Remove
              </button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}
```

**Step 2: Remove the entire `<style>` block**

Delete the full `<style>...</style>` section.

**Step 3: Verify in browser**

Open the app and check:
- Empty state text renders with muted color
- Table (when sites exist) uses DaisyUI table styling
- Remove button is ghost-styled

**Step 4: Commit**

```bash
git add frontend/src/SiteList.svelte
git commit -m "feat: migrate SiteList.svelte to DaisyUI classes"
```

---

### Task 4: Migrate AddSiteForm.svelte

**Files:**
- Modify: `frontend/src/AddSiteForm.svelte`

**Step 1: Replace markup with DaisyUI classes**

Keep the `<script>` block unchanged. Replace the template:

```svelte
<form class="card bg-base-300 p-4 mt-6" on:submit|preventDefault={handleSubmit}>
  <div class="flex gap-4 items-end mb-3">
    <label class="flex flex-col flex-1 text-left">
      <span class="text-xs uppercase tracking-wide text-base-content/60 mb-1">Path</span>
      <input type="text" class="input input-bordered input-sm" bind:value={path} on:input={handlePathInput} placeholder="/home/user/projects/myapp" />
    </label>
    <label class="flex flex-col flex-1 text-left">
      <span class="text-xs uppercase tracking-wide text-base-content/60 mb-1">Domain</span>
      <input type="text" class="input input-bordered input-sm" bind:value={domain} placeholder="myapp.test" />
    </label>
  </div>
  <div class="flex gap-4 items-end mb-3">
    <label class="flex flex-col flex-1 text-left">
      <span class="text-xs uppercase tracking-wide text-base-content/60 mb-1">PHP Version</span>
      <input type="text" class="input input-bordered input-sm" bind:value={phpVersion} placeholder="8.3 (optional)" />
    </label>
    <label class="flex flex-col flex-1 text-left">
      <span class="text-xs uppercase tracking-wide text-base-content/60 mb-1">Node Version</span>
      <input type="text" class="input input-bordered input-sm" bind:value={nodeVersion} placeholder="system (optional)" />
    </label>
    <label class="flex items-center gap-2 whitespace-nowrap">
      <input type="checkbox" class="checkbox checkbox-sm" bind:checked={tls} />
      <span class="text-sm">TLS</span>
    </label>
    <button type="submit" class="btn btn-success btn-sm">Add Site</button>
  </div>
  {#if error}
    <p class="text-error text-sm mt-2 text-left">{error}</p>
  {/if}
</form>
```

**Step 2: Remove the entire `<style>` block**

Delete the full `<style>...</style>` section.

**Step 3: Verify in browser**

- Form has card styling with slightly different background shade
- Inputs use DaisyUI bordered input styling
- Checkbox uses DaisyUI checkbox
- Add Site button is green (success color)
- Focus states on inputs work (DaisyUI handles this)
- Error text shows in error color

**Step 4: Commit**

```bash
git add frontend/src/AddSiteForm.svelte
git commit -m "feat: migrate AddSiteForm.svelte to DaisyUI classes"
```

---

### Task 5: Migrate ServiceList.svelte

**Files:**
- Modify: `frontend/src/ServiceList.svelte`

**Step 1: Replace markup with DaisyUI classes**

Keep the `<script>` block unchanged. Replace the template:

```svelte
{#if services.length === 0}
  <p class="text-base-content/50 py-8">No database services configured.</p>
{:else}
  <div class="overflow-x-auto">
    <table class="table">
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
                  <button class="btn btn-outline btn-error btn-xs" on:click={() => onStop(svc.type)}>
                    Stop
                  </button>
                {:else}
                  <button class="btn btn-outline btn-success btn-xs" on:click={() => onStart(svc.type)}>
                    Start
                  </button>
                {/if}
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}
```

**Step 2: Remove the entire `<style>` block**

Delete the full `<style>...</style>` section.

**Step 3: Verify in browser**

- Empty state shows muted text
- Table uses DaisyUI styling
- Badges show correct colors (green=running, ghost=stopped, warning=not installed)
- Start button is green outline, Stop button is red outline
- Disabled rows have reduced opacity

**Step 4: Commit**

```bash
git add frontend/src/ServiceList.svelte
git commit -m "feat: migrate ServiceList.svelte to DaisyUI classes"
```

---

### Task 6: Visual Review and Cleanup

**Files:**
- Modify: `frontend/src/style.css` (if needed)
- Modify: any component (if needed)

**Step 1: Full visual review**

Open `http://localhost:5173` and verify the complete UI:
- Dark theme applied consistently
- All typography readable (check contrast)
- Buttons have proper hover/active states
- Inputs have focus ring
- Table rows align properly
- Cards have consistent spacing
- No leftover hand-written CSS conflicts

**Step 2: Check for unused CSS**

Verify no `<style>` blocks remain in any `.svelte` file (they should all be removed). If any scoped styles are still needed for layout that DaisyUI doesn't cover, keep only those.

**Step 3: Verify wails dev still works**

Run: `wails dev` (if not already running)
Expected: App loads correctly in the Wails window with the same styling as the browser.

**Step 4: Create task file**

Create `docs/tasks/012-daisyui-integration.md` following the task template format. Mark all steps as complete.

**Step 5: Final commit**

```bash
git add -A
git commit -m "feat: complete DaisyUI + Tailwind CSS integration"
```
