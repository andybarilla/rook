<script>
  export let open = false;
  export let title = '';
  export let message = '';
  export let confirmLabel = 'Confirm';
  export let confirmClass = 'btn-primary';
  export let onConfirm = () => {};
  export let onCancel = () => {};
</script>

<svelte:window on:keydown={(e) => open && e.key === 'Escape' && onCancel()} />

{#if open}
  <div class="modal modal-open">
    <div class="modal-box">
      <h3 class="font-bold text-lg">{title}</h3>
      <p class="py-4">{message}</p>
      <div class="modal-action">
        <button class="btn btn-ghost" on:click={onCancel}>Cancel</button>
        <button class="btn {confirmClass}" on:click={onConfirm}>{confirmLabel}</button>
      </div>
    </div>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <div class="modal-backdrop" on:click={onCancel}></div>
  </div>
{/if}
