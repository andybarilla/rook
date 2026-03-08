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
