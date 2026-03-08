<script>
  import { onMount } from 'svelte';
  import { theme, toggleTheme } from './lib/theme.js';
  import { MiseStatus } from '../wailsjs/go/main/App.js';

  let miseInfo = { available: false, version: '' };

  onMount(async () => {
    try {
      miseInfo = await MiseStatus();
    } catch {
      // ignore — not critical
    }
  });
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
    <h3 class="font-semibold text-base-content mb-4">Runtime Manager</h3>
    {#if miseInfo.available}
      <div class="flex items-center gap-2">
        <span class="badge badge-success badge-sm">Detected</span>
        <span class="text-sm text-base-content">{miseInfo.version}</span>
      </div>
    {:else}
      <div class="flex items-center gap-2">
        <span class="badge badge-ghost badge-sm">Not found</span>
        <span class="text-sm text-base-content/60">
          Install mise for automatic runtime version management
        </span>
      </div>
      <a href="https://mise.jdx.dev" target="_blank" rel="noopener noreferrer" class="link link-primary text-sm mt-2 inline-block">
        mise.jdx.dev
      </a>
    {/if}
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
