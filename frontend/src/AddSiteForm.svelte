<script>
  import { createEventDispatcher } from 'svelte';
  import { notifySuccess, notifyError } from './lib/notifications.js';
  import { friendlyError } from './lib/errorMessages.js';

  const dispatch = createEventDispatcher();

  export let onAdd = () => {};

  let path = '';
  let domain = '';
  let phpVersion = '';
  let nodeVersion = '';
  let tls = false;
  let submitting = false;
  export let open = false;
  let pathInput;

  $: if (open && pathInput) {
    pathInput.focus();
  }

  export function focusPathInput() {
    if (pathInput) pathInput.focus();
  }

  function inferDomain(p) {
    if (!p) return '';
    const parts = p.replace(/[\\/]+$/, '').split(/[\\/]/);
    return (parts[parts.length - 1] || '') + '.test';
  }

  function handlePathInput() {
    if (!domain || domain === inferDomain(path.slice(0, path.length - 1))) {
      domain = inferDomain(path);
    }
  }

  export async function handleSubmit() {
    if (!path || !domain) {
      notifyError('Path and domain are required.');
      return;
    }
    submitting = true;
    try {
      await onAdd(path, domain, phpVersion, nodeVersion, tls);
      notifySuccess(`Site "${domain}" added.`);
      path = '';
      domain = '';
      phpVersion = '';
      nodeVersion = '';
      tls = false;
      dispatch('close');
    } catch (e) {
      notifyError(friendlyError(e.message || String(e)));
    } finally {
      submitting = false;
    }
  }
</script>

{#if open}
<div class="modal modal-open" role="dialog" aria-modal="true" aria-labelledby="add-site-title">
  <div class="modal-box">
    <h3 id="add-site-title" class="font-bold text-lg mb-4">Add Site</h3>
    <form on:submit|preventDefault={handleSubmit}>
      <div data-section="required" class="mb-2">
        <div class="form-row flex gap-4 items-end mb-3">
          <label class="flex flex-col flex-[2] text-left">
            <span class="text-xs text-base-content/70 uppercase tracking-wide mb-1">Path</span>
            <input type="text" class="input input-bordered input-md" bind:value={path} bind:this={pathInput} on:input={handlePathInput} placeholder="/home/user/projects/myapp" disabled={submitting} />
          </label>
          <label class="flex flex-col flex-1 text-left">
            <span class="text-xs text-base-content/70 uppercase tracking-wide mb-1">Domain</span>
            <input type="text" class="input input-bordered input-md" bind:value={domain} placeholder="myapp.test" disabled={submitting} />
          </label>
        </div>
      </div>
      <div class="divider text-xs text-base-content/50">Options</div>
      <div data-section="optional">
        <div class="form-row flex gap-4 items-end mb-3">
          <label class="flex flex-col flex-1 text-left">
            <span class="text-xs text-base-content/70 uppercase tracking-wide mb-1">PHP Version</span>
            <input type="text" class="input input-bordered input-sm" bind:value={phpVersion} placeholder="8.3 (optional)" disabled={submitting} />
          </label>
          <label class="flex flex-col flex-1 text-left">
            <span class="text-xs text-base-content/70 uppercase tracking-wide mb-1">Node Version</span>
            <input type="text" class="input input-bordered input-sm" bind:value={nodeVersion} placeholder="system (optional)" disabled={submitting} />
          </label>
        </div>
        <div class="form-row flex gap-4 items-end">
          <label class="flex flex-row items-center gap-2 flex-none whitespace-nowrap">
            <input type="checkbox" class="checkbox checkbox-sm" bind:checked={tls} disabled={submitting} />
            <span class="text-xs text-base-content/70 uppercase tracking-wide">TLS</span>
          </label>
        </div>
      </div>
      <div class="modal-action">
        <button type="button" class="btn btn-ghost" on:click={() => dispatch('close')}>Cancel</button>
        <button type="submit" class="btn btn-primary" disabled={submitting}>
          {#if submitting}
            <span class="loading loading-spinner loading-xs"></span>
            Adding…
          {:else}
            Add Site
          {/if}
        </button>
      </div>
    </form>
  </div>
  <div class="modal-backdrop" on:click={() => dispatch('close')} on:keydown={() => {}}></div>
</div>
{/if}
