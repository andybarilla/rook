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
