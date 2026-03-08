<script>
  import EmptyState from './lib/EmptyState.svelte';
  import ServiceCard from './ServiceCard.svelte';

  export let services = [];
  export let loaded = true;
  export let onStart = () => {};
  export let onStop = () => {};

  $: runningCount = services.filter(s => s.running).length;
</script>

{#if !loaded}
  <div class="space-y-4">
    <div class="flex items-center justify-between mb-2">
      <div>
        <div class="skeleton h-6 w-32"></div>
        <div class="skeleton h-4 w-48 mt-1"></div>
      </div>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      {#each Array(3) as _}
        <div class="card bg-base-200 p-5">
          <div class="flex items-start gap-4">
            <div class="skeleton h-8 w-8 rounded"></div>
            <div class="flex-1">
              <div class="skeleton h-5 w-24 mb-2"></div>
              <div class="skeleton h-4 w-48 mb-3"></div>
              <div class="skeleton h-4 w-20"></div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  </div>
{:else if services.length === 0}
  <EmptyState
    icon='<svg xmlns="http://www.w3.org/2000/svg" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14c0 1.66 4.03 3 9 3s9-1.34 9-3V5"/><path d="M3 12c0 1.66 4.03 3 9 3s9-1.34 9-3"/></svg>'
    message="No services available"
    subtitle="Install database plugins to manage MySQL, PostgreSQL, and Redis."
    actionLabel="Setup Guide"
    on:action={() => window.open('https://github.com/andybarilla/rook#services', '_blank')}
  />
{:else}
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h2 class="text-lg font-semibold text-base-content">Services</h2>
        <p class="text-sm text-base-content/60">{runningCount} of {services.length} running</p>
      </div>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      {#each services as svc}
        <ServiceCard service={svc} {onStart} {onStop} />
      {/each}
    </div>
  </div>
{/if}
