<script>
  import { activeProject, triggerPreTarget, triggerPreSelector, triggerPreVars, triggerState, triggerError, showSuccess, showError } from '../stores/appState.js';
  import { navigateTo } from '../stores/router.js';
  import { triggerPipeline } from '../stores/api.js';

  let target = $state('');
  let selector = $state(null);
  let variables = $state([]);
  let initialized = false;

  $effect(() => {
    if (initialized) return;
    initialized = true;
    target = $triggerPreTarget || '';
    selector = $triggerPreSelector || null;
    const preVars = $triggerPreVars || [];
    variables = preVars.length > 0
      ? preVars.map(v => ({ key: v.key, value: v.value }))
      : [{ key: '', value: '' }];
  });

  function addVar() {
    variables = [...variables, { key: '', value: '' }];
  }

  function removeVar(index) {
    variables = variables.filter((_, i) => i !== index);
  }

  async function handleTrigger() {
    if (!$activeProject) return;
    if (!target.trim()) {
      triggerError.set('Target branch is required');
      return;
    }
    triggerState.set('submitting');
    triggerError.set('');
    try {
      const cleanVars = variables
        .filter(v => v.key.trim() !== '')
        .map(v => ({ key: v.key.trim(), value: v.value }));
      await triggerPipeline($activeProject.id, target.trim(), cleanVars, selector);
      showSuccess(`Pipeline triggered on "${target.trim()}"`);
      navigateTo('list');
    } catch (e) {
      triggerError.set(e.message || 'Trigger failed');
    } finally {
      triggerState.set('idle');
    }
  }
</script>

<div class="trigger-view">
  <div class="trigger-header">
    <button class="btn btn-secondary" onclick={() => navigateTo('list')}>
      ← Back to list
    </button>
    <h2>▶ Run Pipeline</h2>
  </div>

  <form class="trigger-form" onsubmit={(e) => { e.preventDefault(); handleTrigger(); }}>
    <div class="form-section">
      <label class="form-label" for="target-input">Target Branch</label>
      <input
        id="target-input"
        type="text"
        class="form-input"
        bind:value={target}
        placeholder="e.g. main"
      />
    </div>

    {#if selector}
      <div class="form-section">
        <label class="form-label">Pipeline Definition</label>
        <div class="selector-display">
          <span class="selector-badge">{selector.type}:{selector.pattern}</span>
        </div>
      </div>
    {/if}

    <div class="form-section">
      <div class="vars-header">
        <label class="form-label">Variables</label>
        <button type="button" class="btn btn-small" onclick={addVar}>+ Add</button>
      </div>
      {#if variables.length === 0}
        <p class="empty-text">No variables</p>
      {:else}
        <div class="vars-list">
          {#each variables as v, i}
            <div class="var-row">
              <input
                type="text"
                class="form-input var-key"
                placeholder="KEY"
                bind:value={v.key}
              />
              <input
                type="text"
                class="form-input var-value"
                placeholder="value"
                bind:value={v.value}
              />
              <button type="button" class="btn btn-icon btn-danger" onclick={() => removeVar(i)} title="Remove">✕</button>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    {#if $triggerError}
      <div class="trigger-error">❌ {$triggerError}</div>
    {/if}

    <button type="submit" class="btn btn-primary btn-run" disabled={$triggerState === 'submitting'}>
      {$triggerState === 'submitting' ? '⏳ Running...' : '▶ Run Pipeline'}
    </button>
  </form>
</div>

<style>
  .trigger-view {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .trigger-header {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .trigger-header h2 {
    font-size: 1.3rem;
    font-weight: 600;
  }

  .trigger-form {
    background: #16181c;
    border: 1px solid #2f3336;
    border-radius: 12px;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    max-width: 600px;
  }

  .form-section {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .form-label {
    font-size: 0.85rem;
    font-weight: 600;
    color: #e7e9ea;
  }

  .form-input {
    background: #0f1419;
    border: 1px solid #2f3336;
    border-radius: 8px;
    padding: 0.6rem 0.75rem;
    color: #e7e9ea;
    font-size: 0.9rem;
    font-family: 'SF Mono', 'Fira Code', monospace;
    outline: none;
    transition: border-color 0.15s;
  }

  .form-input:focus {
    border-color: #1d9bf0;
  }

  .form-input::placeholder {
    color: #71767b;
  }

  .vars-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .vars-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .var-row {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .var-key {
    flex: 1;
  }

  .var-value {
    flex: 2;
  }

  .btn-icon {
    background: none;
    border: none;
    color: #f85149;
    cursor: pointer;
    font-size: 1rem;
    padding: 0.25rem 0.5rem;
    border-radius: 6px;
    transition: background 0.15s;
  }

  .btn-icon:hover {
    background: #3e1a1a;
  }

  .trigger-error {
    background: #3e1a1a;
    color: #f85149;
    padding: 0.75rem 1rem;
    border-radius: 8px;
    font-size: 0.85rem;
    border: 1px solid #5c2424;
  }

  .btn-run {
    align-self: flex-start;
    padding: 0.625rem 2rem;
  }

  .btn-run:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .empty-text {
    color: #71767b;
    font-style: italic;
    font-size: 0.85rem;
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

  .btn-small {
    padding: 0.25rem 0.75rem;
    font-size: 0.8rem;
    background: #2f3336;
    color: #e7e9ea;
  }

  .btn-small:hover {
    background: #3e4144;
  }
</style>