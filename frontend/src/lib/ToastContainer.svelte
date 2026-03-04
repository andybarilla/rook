<script>
  import { notifications, dismiss } from './notifications.js';
  import { fade } from 'svelte/transition';
  import { onDestroy } from 'svelte';

  let timers = {};

  function scheduleAutoDismiss(item) {
    if (timers[item.id]) return;
    timers[item.id] = setTimeout(() => {
      dismiss(item.id);
      delete timers[item.id];
    }, item.timeout);
  }

  $: $notifications.forEach(item => scheduleAutoDismiss(item));

  onDestroy(() => {
    Object.values(timers).forEach(clearTimeout);
  });

  const alertClass = {
    success: 'alert-success',
    error: 'alert-error',
    info: 'alert-info',
    warning: 'alert-warning',
  };
</script>

<div class="toast toast-end toast-bottom z-50">
  {#each $notifications as item (item.id)}
    <div class="alert {alertClass[item.type] || 'alert-info'} text-sm shadow-lg" transition:fade={{ duration: 200 }}>
      <span>{item.message}</span>
      <button class="btn btn-ghost btn-xs" on:click={() => dismiss(item.id)}>✕</button>
    </div>
  {/each}
</div>
