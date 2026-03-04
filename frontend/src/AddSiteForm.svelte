<script>
  export let onAdd = () => {};

  let path = '';
  let domain = '';
  let phpVersion = '';
  let tls = false;
  let error = '';

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

  async function handleSubmit() {
    error = '';
    if (!path || !domain) {
      error = 'Path and domain are required.';
      return;
    }
    try {
      await onAdd(path, domain, phpVersion, tls);
      path = '';
      domain = '';
      phpVersion = '';
      tls = false;
    } catch (e) {
      error = e.message || String(e);
    }
  }
</script>

<form class="add-form" on:submit|preventDefault={handleSubmit}>
  <div class="form-row">
    <label>
      <span>Path</span>
      <input type="text" bind:value={path} on:input={handlePathInput} placeholder="/home/user/projects/myapp" />
    </label>
    <label>
      <span>Domain</span>
      <input type="text" bind:value={domain} placeholder="myapp.test" />
    </label>
  </div>
  <div class="form-row">
    <label>
      <span>PHP Version</span>
      <input type="text" bind:value={phpVersion} placeholder="8.3 (optional)" />
    </label>
    <label class="checkbox-label">
      <input type="checkbox" bind:checked={tls} />
      <span>TLS</span>
    </label>
    <button type="submit" class="btn-add">Add Site</button>
  </div>
  {#if error}
    <p class="error">{error}</p>
  {/if}
</form>

<style>
  .add-form {
    margin-top: 1.5rem;
    padding: 1rem;
    border: 1px solid #333;
    border-radius: 6px;
  }

  .form-row {
    display: flex;
    gap: 1rem;
    align-items: flex-end;
    margin-bottom: 0.75rem;
  }

  label {
    display: flex;
    flex-direction: column;
    flex: 1;
    text-align: left;
  }

  label span {
    font-size: 0.75rem;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.25rem;
  }

  input[type="text"] {
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid #444;
    border-radius: 3px;
    color: white;
    padding: 0.4rem 0.6rem;
    font-size: 0.9rem;
    outline: none;
  }

  input[type="text"]:focus {
    border-color: #6c9;
  }

  .checkbox-label {
    flex-direction: row;
    align-items: center;
    gap: 0.4rem;
    flex: 0;
    white-space: nowrap;
  }

  .checkbox-label input {
    margin: 0;
  }

  .btn-add {
    background: #6c9;
    border: none;
    color: #1b2636;
    padding: 0.4rem 1rem;
    border-radius: 3px;
    cursor: pointer;
    font-weight: 600;
    font-size: 0.9rem;
    white-space: nowrap;
  }

  .btn-add:hover {
    background: #7da;
  }

  .error {
    color: #e74c3c;
    font-size: 0.85rem;
    margin: 0.5rem 0 0;
    text-align: left;
  }
</style>
