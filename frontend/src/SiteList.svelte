<script>
  import ConfirmModal from './lib/ConfirmModal.svelte';
  import EmptyState from './lib/EmptyState.svelte';
  import { createEventDispatcher } from 'svelte';

  export let sites = [];
  export let loaded = true;
  export let onRemove = () => {};

  const dispatch = createEventDispatcher();

  let removingDomain = null;
  let pendingRemoveDomain = null;

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
{:else}
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
      {#each sites as site}
        <tr>
          <td class="font-semibold">{site.domain}</td>
          <td class="text-base-content/60 text-sm">{site.path}</td>
          <td>{site.php_version || '—'}</td>
          <td>{site.node_version || '—'}</td>
          <td>{site.tls ? '✓' : '—'}</td>
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
