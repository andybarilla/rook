<script>
  export let onAdd = () => {};

  let path = '';
  let domain = '';
  let phpVersion = '';
  let nodeVersion = '';
  let tls = false;
  let error = '';

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
    error = '';
    if (!path || !domain) {
      error = 'Path and domain are required.';
      return;
    }
    try {
      await onAdd(path, domain, phpVersion, nodeVersion, tls);
      path = '';
      domain = '';
      phpVersion = '';
      nodeVersion = '';
      tls = false;
    } catch (e) {
      error = e.message || String(e);
    }
  }
</script>

<form class="mt-6 p-4 border border-base-300 rounded-lg" on:submit|preventDefault={handleSubmit}>
  <div class="flex gap-4 items-end mb-3">
    <label class="flex flex-col flex-1 text-left">
      <span class="text-xs text-base-content/50 uppercase tracking-wide mb-1">Path</span>
      <input type="text" class="input input-bordered input-sm" bind:value={path} on:input={handlePathInput} placeholder="/home/user/projects/myapp" />
    </label>
    <label class="flex flex-col flex-1 text-left">
      <span class="text-xs text-base-content/50 uppercase tracking-wide mb-1">Domain</span>
      <input type="text" class="input input-bordered input-sm" bind:value={domain} placeholder="myapp.test" />
    </label>
  </div>
  <div class="flex gap-4 items-end mb-3">
    <label class="flex flex-col flex-1 text-left">
      <span class="text-xs text-base-content/50 uppercase tracking-wide mb-1">PHP Version</span>
      <input type="text" class="input input-bordered input-sm" bind:value={phpVersion} placeholder="8.3 (optional)" />
    </label>
    <label class="flex flex-col flex-1 text-left">
      <span class="text-xs text-base-content/50 uppercase tracking-wide mb-1">Node Version</span>
      <input type="text" class="input input-bordered input-sm" bind:value={nodeVersion} placeholder="system (optional)" />
    </label>
    <label class="flex flex-row items-center gap-2 flex-none whitespace-nowrap">
      <input type="checkbox" class="checkbox checkbox-sm" bind:checked={tls} />
      <span class="text-xs text-base-content/50 uppercase tracking-wide">TLS</span>
    </label>
    <button type="submit" class="btn btn-success btn-sm">Add Site</button>
  </div>
  {#if error}
    <div class="alert alert-error text-sm mt-2">{error}</div>
  {/if}
</form>
