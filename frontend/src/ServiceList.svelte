<script>
  import EmptyState from './lib/EmptyState.svelte';

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
  <EmptyState
    icon='<svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5"/><path d="M3 12c0 1.66 4.03 3 9 3s9-1.34 9-3"/></svg>'
    message="No services available"
    subtitle="Install database plugins to manage MySQL, PostgreSQL, and Redis."
    actionLabel="Setup Guide"
    on:action={() => window.open('https://github.com/andybarilla/flock#services', '_blank')}
  />
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
