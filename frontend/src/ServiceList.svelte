<script>
  export let services = [];
  export let onStart = () => {};
  export let onStop = () => {};

  const displayName = {
    mysql: 'MySQL',
    postgres: 'PostgreSQL',
    redis: 'Redis',
  };
</script>

{#if services.length === 0}
  <p class="text-base-content/50 py-8">No database services configured.</p>
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
                <button class="btn btn-ghost btn-sm hover:btn-error" on:click={() => onStop(svc.type)}>
                  Stop
                </button>
              {:else}
                <button class="btn btn-ghost btn-sm hover:btn-success" on:click={() => onStart(svc.type)}>
                  Start
                </button>
              {/if}
            {/if}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
