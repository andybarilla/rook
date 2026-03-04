<script>
  export let sites = [];
  export let onRemove = () => {};

  function handleRemove(domain) {
    onRemove(domain);
  }
</script>

{#if sites.length === 0}
  <p class="empty">No sites registered. Add one below.</p>
{:else}
  <table class="site-table">
    <thead>
      <tr>
        <th>Domain</th>
        <th>Path</th>
        <th>PHP</th>
        <th>TLS</th>
        <th></th>
      </tr>
    </thead>
    <tbody>
      {#each sites as site}
        <tr>
          <td class="domain">{site.domain}</td>
          <td class="path">{site.path}</td>
          <td>{site.php_version || '—'}</td>
          <td>{site.tls ? '✓' : '—'}</td>
          <td>
            <button class="btn-remove" on:click={() => handleRemove(site.domain)}>
              Remove
            </button>
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

  .site-table {
    width: 100%;
    border-collapse: collapse;
    text-align: left;
  }

  .site-table th {
    color: #888;
    font-weight: 600;
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding: 0.5rem 0.75rem;
    border-bottom: 1px solid #333;
  }

  .site-table td {
    padding: 0.6rem 0.75rem;
    border-bottom: 1px solid #222;
  }

  .domain {
    font-weight: 600;
  }

  .path {
    color: #aaa;
    font-size: 0.85rem;
  }

  .btn-remove {
    background: transparent;
    border: 1px solid #555;
    color: #ccc;
    padding: 0.2rem 0.5rem;
    border-radius: 3px;
    cursor: pointer;
    font-size: 0.8rem;
  }

  .btn-remove:hover {
    border-color: #e74c3c;
    color: #e74c3c;
  }
</style>
