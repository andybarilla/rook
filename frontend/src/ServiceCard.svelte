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
