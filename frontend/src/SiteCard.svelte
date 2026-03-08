<script>
  export let site;
  export let onRemove = () => {};
  export let runtimeStatuses = [];
  export let miseAvailable = false;
  export let onInstall = () => {};

  $: phpBadge = site.php_version ? `PHP ${site.php_version}` : '';
  $: nodeBadge = site.node_version ? `Node ${site.node_version}` : '';
  $: missingRuntimes = runtimeStatuses.filter(
    s => s.domain === site.domain && !s.installed
  );
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

  {#each missingRuntimes as missing}
    <div class="flex items-center gap-2 mt-2">
      <span class="badge badge-warning badge-sm">{missing.tool === 'php' ? 'PHP' : 'Node'} {missing.version} not found</span>
      {#if miseAvailable}
        <button class="btn btn-xs btn-outline btn-warning" on:click={() => onInstall(missing.tool, missing.version)}>
          Install
        </button>
      {/if}
    </div>
  {/each}
</div>
