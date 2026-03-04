<script>
  import { onMount } from 'svelte';
  import { ListSites, AddSite, RemoveSite, DatabaseServices, StartDatabase, StopDatabase } from '../wailsjs/go/main/App.js';
  import { notifySuccess, notifyError } from './lib/notifications.js';
  import { friendlyError } from './lib/errorMessages.js';
  import SiteList from './SiteList.svelte';
  import AddSiteForm from './AddSiteForm.svelte';
  import ServiceList from './ServiceList.svelte';
  import ToastContainer from './lib/ToastContainer.svelte';

  let sites = [];
  let services = [];
  let sitesLoaded = false;
  let servicesLoaded = false;

  async function refreshSites() {
    try {
      sites = await ListSites() || [];
    } catch (e) {
      notifyError(friendlyError(e.message || String(e)));
    } finally {
      sitesLoaded = true;
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
      notifySuccess(`Site "${domain}" removed.`);
    } catch (e) {
      notifyError(friendlyError(e.message || String(e)));
    }
  }

  async function refreshServices() {
    try {
      services = await DatabaseServices() || [];
    } catch (e) {
      notifyError(friendlyError(e.message || String(e)));
    } finally {
      servicesLoaded = true;
    }
  }

  async function handleStartService(svc) {
    try {
      await StartDatabase(svc);
      await refreshServices();
      notifySuccess(`${svc} started.`);
    } catch (e) {
      notifyError(friendlyError(e.message || String(e)));
    }
  }

  async function handleStopService(svc) {
    try {
      await StopDatabase(svc);
      await refreshServices();
      notifySuccess(`${svc} stopped.`);
    } catch (e) {
      notifyError(friendlyError(e.message || String(e)));
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

  <section class="card bg-base-200 p-6">
    <h2 class="text-sm text-base-content/60 uppercase tracking-wide mb-4 font-semibold">Sites</h2>
    <SiteList {sites} loaded={sitesLoaded} onRemove={handleRemove} />
    <AddSiteForm onAdd={handleAdd} />
  </section>

  <section class="card bg-base-200 p-6 mt-6">
    <h2 class="text-sm text-base-content/60 uppercase tracking-wide mb-4 font-semibold">Services</h2>
    <ServiceList {services} loaded={servicesLoaded} onStart={handleStartService} onStop={handleStopService} />
  </section>
</main>

<ToastContainer />
