<script>
  import { createEventDispatcher } from 'svelte';

  export let message = '';
  export let subtitle = '';
  export let actionLabel = '';
  export let icon = '';

  const dispatch = createEventDispatcher();
</script>

<div class="py-10 text-center">
  {#if icon}
    <div data-testid="empty-icon" class="text-base-content/30 mb-3 flex justify-center">
      {@html icon}
    </div>
  {/if}
  <p class="text-base-content/60 font-semibold">{message}</p>
  {#if subtitle}
    <p data-testid="empty-subtitle" class="text-base-content/40 text-sm mt-1">{subtitle}</p>
  {/if}
  {#if actionLabel}
    <button class="btn btn-ghost btn-sm mt-4" on:click={() => dispatch('action')}>
      {actionLabel}
    </button>
  {/if}
</div>
