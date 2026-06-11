<script>
  import { currentScreen, activeProjectId, activeProject, showError, showSuccess, triggerState, triggerError, triggerSuccess } from '../stores/appState.js';
  import { triggerPipeline } from '../stores/api.js';

  let target = $state('');
  let variables = $state([]);
  let submitting = $state(false);

  function addVariable() {
    variables = [...variables, { key: '', value: '' }];
  }

  function removeVariable(i) {
    variables = variables.filter((_, idx) => idx !== i);
  }

  async function handleSubmit(e) {
    e.preventDefault();
    if (!target.trim()) {
      showError('Target (branch/tag) is required');
      return;
    }
    if (!$activeProject) return;

    submitting = true;
    triggerState.set('submitting');
    triggerError.set('');
    triggerSuccess.set('');

    try {
      const filteredVars = variables
        .filter(v => v.key.trim())
        .map(v => ({ key: v.key.trim(), value: v.value }));

      await triggerPipeline($activeProject.id, target.trim(), filteredVars);
      showSuccess(`Pipeline triggered on "${target.trim()}"`);
      target = '';
      variables = [];
      triggerState.set('success');
      // Navigate to list after a short delay
      setTimeout(() => currentScreen.set('list'), 1500);
    } catch (e) {
      triggerError.set(e.message);
      triggerState.set('error');
    } finally {
      submitting = false;
    }
  }
</script>

<div class="pipeline-trigger">
  <div class="trigger-header">
    <button class="btn btn-secondary" onclick={() => currentScreen.set('list')}>
      ← Back to list
    </button>
    <h2>Trigger Pipeline — {$activeProject?.name || 'Unknown'}</h2>
  </div>

  <form class="trigger-form" onsubmit={handleSubmit}>
    <div class="form-group">
      <label class="form-label" for="target">Target (branch or tag)</label>
      <input
        id="target"
        type="text"
        class="form-input"
        placeholder="e.g. main, develop, v1.0.0"
        bind:value={target}
        required
      />
    </div>

    <div class="form-group">
      <div class="form-label-row">
        <label class="form-label">Variables</label>
        <button type="button" class="btn btn-small" onclick={addVariable}>
          + Add Variable
        </button>
      </div>
      {#if variables.length > 0}
        <div class="variables-list">
          {#each variables as variable, i}
            <div class="variable-row">
              <input
                type="text"
                class="form-input var-key"
                placeholder="KEY"
                bind:value={variable.key}
              />
              <input
                type="text"
                class="form-input var-value"
                placeholder="Value"
                bind:value={variable.value}
              />
              <button type="button" class="btn btn-danger-small" onclick={() => removeVariable(i)}>
                ✕
              </button>
            </div>
          {/each}
        </div>
      {:else}
        <p class="form-hint">No custom variables. Add variables if your pipeline requires them.</p>
      {/if}
    </div>

    {#if $triggerError}
      <div class="form-error">❌ {$triggerError}</div>
    {/if}

    <button type="submit" class="btn btn-primary btn-submit" disabled={submitting}>
      {submitting ? '⏳ Triggering...' : '▶ Trigger Pipeline'}
    </button>
  </form>
</div>

<style>
  .pipeline-trigger {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .trigger-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .trigger-header h2 {
    font-size: 1.3rem;
    font-weight: 600;
  }

  .btn {
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 9999px;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s;
    white-space: nowrap;
  }

  .btn-secondary {
    background: #2f3336;
    color: #e7e9ea;
  }

  .btn-secondary:hover {
    background: #3e4144;
  }

  .btn-primary {
    background: #1d9bf0;
    color: #fff;
  }

  .btn-primary:hover {
    background: #1a8cd8;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-small {
    padding: 0.25rem 0.75rem;
    font-size: 0.8rem;
    background: #2f3336;
    color: #e7e9ea;
    border: none;
    border-radius: 9999px;
    cursor: pointer;
  }

  .btn-small:hover {
    background: #3e4144;
  }

  .btn-danger-small {
    padding: 0.25rem 0.5rem;
    font-size: 0.8rem;
    background: #b91c1c;
    color: #fff;
    border: none;
    border-radius: 9999px;
    cursor: pointer;
  }

  .btn-danger-small:hover {
    background: #991b1b;
  }

  .btn-submit {
    align-self: flex-start;
    padding: 0.75rem 2rem;
    font-size: 1rem;
  }

  .trigger-form {
    background: #16181c;
    border: 1px solid #2f3336;
    border-radius: 12px;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    max-width: 640px;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .form-label {
    font-size: 0.85rem;
    font-weight: 600;
    color: #8b949e;
  }

  .form-label-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .form-input {
    background: #0d1117;
    border: 1px solid #2f3336;
    border-radius: 8px;
    padding: 0.625rem 0.875rem;
    color: #e7e9ea;
    font-size: 0.9rem;
    outline: none;
    width: 100%;
    box-sizing: border-box;
  }

  .form-input:focus {
    border-color: #1d9bf0;
  }

  .form-input::placeholder {
    color: #484f58;
  }

  .form-hint {
    color: #484f58;
    font-size: 0.85rem;
    font-style: italic;
  }

  .form-error {
    background: #3e1a1a;
    border: 1px solid #670606;
    color: #f4a2a2;
    padding: 0.75rem 1rem;
    border-radius: 8px;
    font-size: 0.85rem;
  }

  .variables-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .variable-row {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .var-key {
    max-width: 160px;
    font-family: 'SF Mono', 'Fira Code', monospace;
  }

  .var-value {
    flex: 1;
  }
</style>