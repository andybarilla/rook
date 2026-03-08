<script>
  import ConfirmModal from './lib/ConfirmModal.svelte';
  import EmptyState from './lib/EmptyState.svelte';
  import SiteCard from './SiteCard.svelte';
  import { createEventDispatcher } from 'svelte';

  export let sites = [];
  export let loaded = true;
  export let onRemove = () => {};
  export let runtimeStatuses = [];
  export let miseAvailable = false;
  export let onInstall = () => {};

  const dispatch = createEventDispatcher();

  let removingDomain = null;
  let pendingRemoveDomain = null;
  let searchQuery = '';
  let viewMode = (typeof localStorage !== 'undefined' && localStorage.getItem('flock-view')) || 'table';

  function setViewMode(mode) {
    viewMode = mode;
    if (typeof localStorage !== 'undefined') localStorage.setItem('flock-view', mode);
  }

  $: filtered = sites.filter(s =>
    s.domain.toLowerCase().includes(searchQuery.toLowerCase()) ||
    s.path.toLowerCase().includes(searchQuery.toLowerCase())
  );

  function requestRemove(domain) {
    pendingRemoveDomain = domain;
  }

  function cancelRemove() {
    pendingRemoveDomain = null;
  }

  async function confirmRemove() {
    const domain = pendingRemoveDomain;
    pendingRemoveDomain = null;
    removingDomain = domain;
    try {
      await onRemove(domain);
    } finally {
      removingDomain = null;
    }
  }
</script>

{#if loaded && sites.length > 0}
<div class="flex items-center justify-between mb-4">
  <h2 class="text-lg font-semibold text-base-content">Sites</h2>
  <div class="flex items-center gap-2">
    <button class="btn btn-ghost btn-sm btn-square" class:btn-active={viewMode === 'table'} title="Table view" on:click={() => setViewMode('table')}>
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>
    </button>
    <button class="btn btn-ghost btn-sm btn-square" class:btn-active={viewMode === 'cards'} title="Card view" on:click={() => setViewMode('cards')}>
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
    </button>
    <button class="btn btn-primary btn-sm" on:click={() => dispatch('addsite')}>+ Add Site</button>
  </div>
</div>
<div class="mb-4">
  <input type="text" class="input input-bordered input-sm w-full" placeholder="Search sites..." bind:value={searchQuery} />
</div>
{/if}

{#if !loaded}
  <table class="table table-zebra">
    <thead>
      <tr>
        <th>Domain</th>
        <th>Path</th>
        <th>PHP</th>
        <th>Node</th>
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
          <td><div class="skeleton h-4 w-10"></div></td>
          <td><div class="skeleton h-4 w-6"></div></td>
          <td><div class="skeleton h-4 w-16"></div></td>
        </tr>
      {/each}
    </tbody>
  </table>
{:else if sites.length === 0}
  <EmptyState
    icon='<svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>'
    message="No sites yet"
    subtitle="Add your first site to start developing locally."
    actionLabel="Add Site"
    on:action={() => dispatch('addsite')}
  />
{:else if filtered.length === 0}
  <p class="text-center text-base-content/60 py-8">No matching sites</p>
{:else if viewMode === 'table'}
  <table class="table table-zebra">
    <thead>
      <tr>
        <th>Domain</th>
        <th>Path</th>
        <th>PHP</th>
        <th>Node</th>
        <th>TLS</th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      {#each filtered as site}
        <tr>
          <td class="font-semibold">{site.domain}</td>
          <td class="text-base-content/70 text-sm">{site.path}</td>
          <td>
            {#if site.php_version}
              <span class="badge badge-sm badge-primary">PHP {site.php_version}</span>
              {#if runtimeStatuses.some(s => s.domain === site.domain && s.tool === 'php' && !s.installed)}
                <span class="badge badge-sm badge-warning ml-1" title="Not installed">!</span>
              {/if}
            {:else}
              <span class="text-base-content/30">—</span>
            {/if}
          </td>
          <td>
            {#if site.node_version}
              <span class="badge badge-sm badge-success">Node {site.node_version}</span>
              {#if runtimeStatuses.some(s => s.domain === site.domain && s.tool === 'node' && !s.installed)}
                <span class="badge badge-sm badge-warning ml-1" title="Not installed">!</span>
              {/if}
            {:else}
              <span class="text-base-content/30">—</span>
            {/if}
          </td>
          <td>
            {#if site.tls}
              <span class="badge badge-sm badge-info">TLS</span>
            {:else}
              <span class="text-base-content/30">—</span>
            {/if}
          </td>
          <td>
            <button
              class="btn btn-ghost btn-sm hover:btn-error"
              disabled={removingDomain === site.domain}
              on:click={() => requestRemove(site.domain)}
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
{:else}
  <div data-testid="site-cards" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
    {#each filtered as site}
      <SiteCard {site} onRemove={requestRemove} {runtimeStatuses} {miseAvailable} {onInstall} />
    {/each}
  </div>
{/if}

<ConfirmModal
  open={pendingRemoveDomain !== null}
  title="Remove Site"
  message={'Are you sure you want to remove "' + (pendingRemoveDomain || '') + '"?'}
  confirmLabel="Yes, Remove"
  confirmClass="btn-error"
  onConfirm={confirmRemove}
  onCancel={cancelRemove}
/>
