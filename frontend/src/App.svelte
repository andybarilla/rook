<script>
  import { onMount } from 'svelte';
  import { ListSites, AddSite, RemoveSite, DatabaseServices, StartDatabase, StopDatabase } from '../wailsjs/go/main/App.js';
  import SiteList from './SiteList.svelte';
  import AddSiteForm from './AddSiteForm.svelte';
  import ServiceList from './ServiceList.svelte';

  let sites = [];
  let services = [];
  let error = '';

  async function refreshSites() {
    try {
      sites = await ListSites() || [];
      error = '';
    } catch (e) {
      error = 'Failed to load sites: ' + (e.message || String(e));
    }
  }

  async function handleAdd(path, domain, phpVersion, nodeVersion, tls) {
    await AddSite(path, domain, phpVersion, nodeVersion, tls);
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

  async function refreshServices() {
    try {
      services = await DatabaseServices() || [];
    } catch (e) {
      error = 'Failed to load services: ' + (e.message || String(e));
    }
  }

  async function handleStartService(svc) {
    try {
      await StartDatabase(svc);
      await refreshServices();
    } catch (e) {
      error = 'Failed to start service: ' + (e.message || String(e));
    }
  }

  async function handleStopService(svc) {
    try {
      await StopDatabase(svc);
      await refreshServices();
    } catch (e) {
      error = 'Failed to stop service: ' + (e.message || String(e));
    }
  }

  onMount(() => {
    refreshSites();
    refreshServices();
  });
</script>

<main class="max-w-3xl mx-auto px-6 py-8 text-left">
  <header class="mb-8">
    <h1 class="text-2xl font-bold m-0">Flock</h1>
    <p class="text-base-content/50 mt-1 text-sm">Local Development Environment</p>
  </header>

  {#if error}
    <div class="alert alert-error mb-4 text-sm">{error}</div>
  {/if}

  <section class="card bg-base-200 p-6">
    <h2 class="text-sm text-base-content/60 uppercase tracking-wide mb-4 font-semibold">Sites</h2>
    <SiteList {sites} onRemove={handleRemove} />
    <AddSiteForm onAdd={handleAdd} />
  </section>

  <section class="card bg-base-200 p-6 mt-6">
    <h2 class="text-sm text-base-content/60 uppercase tracking-wide mb-4 font-semibold">Services</h2>
    <ServiceList {services} onStart={handleStartService} onStop={handleStopService} />
  </section>
</main>
