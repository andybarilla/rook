<script>
  import { onMount } from 'svelte';
  import { ListSites, AddSite, RemoveSite, DatabaseServices, StartDatabase, StopDatabase, CheckRuntimes, InstallRuntime, MiseStatus } from '../wailsjs/go/main/App.js';
  import { notifySuccess, notifyError, dismissLatest } from './lib/notifications.js';
  import { friendlyError } from './lib/errorMessages.js';
  import { initTheme } from './lib/theme.js';
  import SiteList from './SiteList.svelte';
  import AddSiteForm from './AddSiteForm.svelte';
  import ServiceList from './ServiceList.svelte';
  import SettingsTab from './SettingsTab.svelte';
  import ToastContainer from './lib/ToastContainer.svelte';

  let sites = [];
  let services = [];
  let sitesLoaded = false;
  let servicesLoaded = false;
  let addFormOpen = false;
  let addSiteForm;
  let activeTab = 'sites';
  let runtimeStatuses = [];
  let miseAvailable = false;

  function handleKeydown(e) {
    if (e.ctrlKey && e.key === 'n') {
      e.preventDefault();
      activeTab = 'sites';
      addFormOpen = true;
      setTimeout(() => addSiteForm?.focusPathInput(), 0);
      return;
    }
    if (e.ctrlKey && e.key === 'Enter') {
      if (addFormOpen) {
        e.preventDefault();
        addSiteForm?.handleSubmit();
      }
      return;
    }
    if (e.key === 'Escape') {
      if (addFormOpen) {
        addFormOpen = false;
        return;
      }
      dismissLatest();
    }
  }

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
    await refreshRuntimes();
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

  async function refreshRuntimes() {
    try {
      runtimeStatuses = await CheckRuntimes() || [];
      const status = await MiseStatus();
      miseAvailable = status.available;
    } catch {
      // non-critical
    }
  }

  async function handleInstall(tool, version) {
    try {
      await InstallRuntime(tool, version);
      notifySuccess(`${tool}@${version} installed.`);
      await refreshRuntimes();
    } catch (e) {
      notifyError(friendlyError(e.message || String(e)));
    }
  }

  onMount(() => {
    initTheme();
    refreshSites();
    refreshServices();
    refreshRuntimes();
  });
</script>

<svelte:window on:keydown={handleKeydown} />

<main class="h-full flex flex-col">
  <header class="bg-base-100 border-b border-base-300 px-6">
    <div class="max-w-5xl mx-auto flex items-center gap-6">
      <div class="flex items-center gap-2 py-3">
        <div class="w-7 h-7 bg-primary rounded-lg flex items-center justify-center">
          <span class="text-primary-content text-sm font-bold">R</span>
        </div>
        <span class="font-bold text-base-content">Rook</span>
      </div>
      <div role="tablist" class="tabs tabs-bordered flex-1">
        <button role="tab" class="tab" class:tab-active={activeTab === 'sites'} on:click={() => activeTab = 'sites'}>Sites</button>
        <button role="tab" class="tab" class:tab-active={activeTab === 'services'} on:click={() => activeTab = 'services'}>Services</button>
        <button role="tab" class="tab" class:tab-active={activeTab === 'settings'} on:click={() => activeTab = 'settings'}>Settings</button>
      </div>
    </div>
  </header>

  <div class="flex-1 overflow-auto">
    <div class="max-w-5xl mx-auto px-6 py-6">
      {#if activeTab === 'sites'}
        <SiteList {sites} loaded={sitesLoaded} onRemove={handleRemove} {runtimeStatuses} {miseAvailable} onInstall={handleInstall} on:addsite={() => { addFormOpen = true; }} />
        <AddSiteForm bind:this={addSiteForm} onAdd={handleAdd} open={addFormOpen} on:close={() => { addFormOpen = false; }} />
      {:else if activeTab === 'services'}
        <ServiceList {services} loaded={servicesLoaded} onStart={handleStartService} onStop={handleStopService} />
      {:else if activeTab === 'settings'}
        <SettingsTab />
      {/if}
    </div>
  </div>
</main>

<ToastContainer />
