<script>
  import { tick } from 'svelte';

  export let open = false;
  export let title = '';
  export let message = '';
  export let confirmLabel = 'Confirm';
  export let confirmClass = 'btn-primary';
  export let onConfirm = () => {};
  export let onCancel = () => {};

  let cancelBtn;
  let confirmBtn;
  let previouslyFocused;

  $: if (open) {
    previouslyFocused = document.activeElement;
    tick().then(() => cancelBtn?.focus());
  }
  $: if (!open && previouslyFocused) {
    const el = previouslyFocused;
    previouslyFocused = null;
    tick().then(() => el?.focus());
  }

  function trapFocus(e) {
    if (e.key !== 'Tab' || !open) return;
    const focusable = [cancelBtn, confirmBtn].filter(Boolean);
    if (focusable.length === 0) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  }
</script>

<svelte:window on:keydown={(e) => {
  if (open && e.key === 'Escape') onCancel();
  if (open) trapFocus(e);
}} />

{#if open}
  <div class="modal modal-open" role="dialog" aria-modal="true" aria-labelledby="modal-title">
    <div class="modal-box">
      <h3 id="modal-title" class="font-bold text-lg">{title}</h3>
      <p class="py-4">{message}</p>
      <div class="modal-action">
        <button bind:this={cancelBtn} class="btn btn-ghost" on:click={onCancel}>Cancel</button>
        <button bind:this={confirmBtn} class="btn {confirmClass}" on:click={onConfirm}>{confirmLabel}</button>
      </div>
    </div>
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <div class="modal-backdrop" on:click={onCancel}></div>
  </div>
{/if}
