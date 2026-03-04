<script>
  import { onMount } from 'svelte';
  import { ListSites, AddSite, RemoveSite } from '../wailsjs/go/main/App.js';
  import SiteList from './SiteList.svelte';
  import AddSiteForm from './AddSiteForm.svelte';

  let sites = [];
  let error = '';

  async function refreshSites() {
    try {
      sites = await ListSites() || [];
      error = '';
    } catch (e) {
      error = 'Failed to load sites: ' + (e.message || String(e));
    }
  }

  async function handleAdd(path, domain, phpVersion, tls) {
    await AddSite(path, domain, phpVersion, tls);
    await refreshSites();
  }

  async function handleRemove(domain) {
    try {
      await RemoveSite(domain);
      await refreshSites();
    } catch (e) {
      error = 'Failed to remove site: ' + (e.message || String(e));
    }
  }

  onMount(refreshSites);
</script>

<main>
  <header>
    <h1>Flock</h1>
    <p class="subtitle">Local Development Environment</p>
  </header>

  {#if error}
    <p class="global-error">{error}</p>
  {/if}

  <section class="content">
    <h2>Sites</h2>
    <SiteList {sites} onRemove={handleRemove} />
    <AddSiteForm onAdd={handleAdd} />
  </section>
</main>

<style>
  main {
    max-width: 800px;
    margin: 0 auto;
    padding: 2rem 1.5rem;
    text-align: left;
  }

  header {
    margin-bottom: 2rem;
  }

  h1 {
    margin: 0;
    font-size: 1.5rem;
  }

  .subtitle {
    color: #888;
    margin: 0.25rem 0 0;
    font-size: 0.85rem;
  }

  h2 {
    font-size: 1rem;
    color: #aaa;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin: 0 0 1rem;
  }

  .global-error {
    background: rgba(231, 76, 60, 0.15);
    border: 1px solid #e74c3c;
    border-radius: 4px;
    padding: 0.5rem 0.75rem;
    color: #e74c3c;
    font-size: 0.85rem;
    margin-bottom: 1rem;
  }

  .content {
    background: rgba(255, 255, 255, 0.03);
    border-radius: 8px;
    padding: 1.5rem;
  }
</style>
