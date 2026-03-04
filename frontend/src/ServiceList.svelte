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
  <p class="empty">No database services configured.</p>
{:else}
  <table class="service-table">
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
        <tr class:disabled={!svc.enabled}>
          <td class="name">{displayName[svc.type] || svc.type}</td>
          <td class="port">{svc.port}</td>
          <td>
            {#if !svc.enabled}
              <span class="status status-unavailable">Not installed</span>
            {:else if svc.running}
              <span class="status status-running">Running</span>
            {:else}
              <span class="status status-stopped">Stopped</span>
            {/if}
          </td>
          <td>
            {#if svc.enabled}
              {#if svc.running}
                <button class="btn-action btn-stop" on:click={() => onStop(svc.type)}>
                  Stop
                </button>
              {:else}
                <button class="btn-action btn-start" on:click={() => onStart(svc.type)}>
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

<style>
  .empty {
    color: #888;
    padding: 2rem;
  }

  .service-table {
    width: 100%;
    border-collapse: collapse;
    text-align: left;
  }

  .service-table th {
    color: #888;
    font-weight: 600;
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding: 0.5rem 0.75rem;
    border-bottom: 1px solid #333;
  }

  .service-table td {
    padding: 0.6rem 0.75rem;
    border-bottom: 1px solid #222;
  }

  .name {
    font-weight: 600;
  }

  .port {
    color: #aaa;
    font-size: 0.85rem;
  }

  .disabled td {
    opacity: 0.5;
  }

  .status {
    font-size: 0.8rem;
    padding: 0.15rem 0.4rem;
    border-radius: 3px;
  }

  .status-running {
    color: #2ecc71;
  }

  .status-stopped {
    color: #888;
  }

  .status-unavailable {
    color: #e67e22;
  }

  .btn-action {
    background: transparent;
    border: 1px solid #555;
    color: #ccc;
    padding: 0.2rem 0.5rem;
    border-radius: 3px;
    cursor: pointer;
    font-size: 0.8rem;
  }

  .btn-start:hover {
    border-color: #2ecc71;
    color: #2ecc71;
  }

  .btn-stop:hover {
    border-color: #e74c3c;
    color: #e74c3c;
  }
</style>
