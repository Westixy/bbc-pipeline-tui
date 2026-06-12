<script>
  import { get } from 'svelte/store';
  import { activeProject, selectedPipeline, selectedSteps, selectedVariables, selectedLogVariables, triggerPreTarget, triggerPreSelector, triggerPreVars, showError, showSuccess } from '../stores/appState.js';
  import { navigateTo } from '../stores/router.js';
  import { triggerPipeline } from '../stores/api.js';

  let targetBranch = $state('');
  let pipelineSelector = $state('');
  let variables = $state([]);
  let running = $state(false);
  let showPreFilled = $state(false);

  // Check whether we have pre-filled data from a previous pipeline detail
  let isRunAgain = $derived(
    Boolean($triggerPreTarget) || ($triggerPreVars && $triggerPreVars.length > 0)
  );

  function usePreFilled() {
    targetBranch = $triggerPreTarget || '';
    pipelineSelector = typeof $triggerPreSelector === 'object'
      ? ($triggerPreSelector?.pattern || '')
      : ($triggerPreSelector || '');
    variables = ($triggerPreVars || []).map(v => ({ key: v.key, value: v.value }));
    showPreFilled = true;
  }

  function addVariableRow() {
    variables = [...variables, { key: '', value: '' }];
  }

  function updateVariableKey(index, value) {
    variables = variables.map((v, i) => i === index ? { ...v, key: value } : v);
  }

  function updateVariableValue(index, value) {
    variables = variables.map((v, i) => i === index ? { ...v, value: value } : v);
  }

  function removeVariable(index) {
    variables = variables.filter((_, i) => i !== index);
  }

  async function handleRun(e) {
    e.preventDefault();
    if (!$activeProject) return;
    if (!targetBranch.trim()) {
      showError('Target branch is required');
      return;
    }
    running = true;
    try {
      const cleanedVars = variables
        .filter(v => v.key.trim())
        .map(v => ({ key: v.key.trim(), value: v.value }));
      const selector = pipelineSelector.trim()
        ? { type: 'custom', pattern: pipelineSelector.trim() }
        : null;
      const result = await triggerPipeline(
        $activeProject.id,
        targetBranch.trim(),
        cleanedVars,
        selector
      );
      showSuccess(`Pipeline #${result.build_number || 'new'} started`);
      navigateTo('list');
    } catch (e) {
      showError(e.message);
    } finally {
      running = false;
    }
  }

  $effect(() => {
    if (isRunAgain && !showPreFilled && !targetBranch) {
      usePreFilled();
    }
  });
</script>

<div class="trigger-pipeline">
  <div class="trigger-header">
    <button class="btn btn-secondary" onclick={() => navigateTo('list')}>
      ← Back to list
    </button>
    <h2>Trigger Pipeline</h2>
    {#if isRunAgain}
      <span class="run-again-badge">🔁 Run again — pre-filled from previous pipeline</span>
    {/if}
  </div>

  <form class="trigger-form" onsubmit={handleRun}>
    <div class="trigger-card">
      <div class="form-row">
        <div class="form-group">
          <label class="form-label" for="trigger-target">Target Branch *</label>
          <input
            id="trigger-target"
            type="text"
            class="form-input"
            placeholder="e.g. main, develop, feature/xyz"
            bind:value={targetBranch}
            required
          />
          <span class="form-hint">The git branch to run the pipeline against</span>
        </div>

        <div class="form-group">
          <label class="form-label" for="trigger-selector">Pipeline Selector (optional)</label>
          <input
            id="trigger-selector"
            type="text"
            class="form-input"
            placeholder="e.g. default, custom-pattern"
            bind:value={pipelineSelector}
          />
          <span class="form-hint">Filter which pipeline steps to execute</span>
        </div>
      </div>

      <div class="variables-section">
        <div class="variables-header">
          <h3>Variables</h3>
          <button type="button" class="btn btn-small" onclick={addVariableRow}>
            + Add Variable
          </button>
        </div>
        {#if variables.length === 0}
          <p class="empty-text">No custom variables. Click "+ Add Variable" to add one.</p>
        {:else}
          <div class="variables-list">
            {#each variables as v, i}
              <div class="variable-row">
                <input
                  type="text"
                  class="form-input var-key"
                  placeholder="Key"
                  aria-label="Variable key"
                  value={v.key}
                  oninput={(e) => updateVariableKey(i, e.target.value)}
                />
                <input
                  type="text"
                  class="form-input var-value"
                  placeholder="Value"
                  aria-label="Variable value"
                  value={v.value}
                  oninput={(e) => updateVariableValue(i, e.target.value)}
                />
                <button
                  type="button"
                  class="btn btn-icon btn-remove"
                  onclick={() => removeVariable(i)}
                  title="Remove variable"
                >
                  ✕
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <div class="form-actions">
        <button type="submit" class="btn btn-primary btn-run" disabled={running}>
          {running ? '⏳ Running...' : '▶ Run Pipeline'}
        </button>
      </div>
    </div>
  </form>
</div>

<style>
  .trigger-pipeline {
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

  .run-again-badge {
    background: #1d2e3e;
    color: #6cb6ff;
    padding: 0.25rem 0.75rem;
    border-radius: 9999px;
    font-size: 0.8rem;
    font-weight: 500;
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

  .btn-secondary:hover { background: #3e4144; }

  .btn-primary {
    background: #1d9bf0;
    color: #fff;
  }

  .btn-primary:hover { background: #1a8cd8; }
  .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

  .btn-small {
    padding: 0.25rem 0.75rem;
    font-size: 0.8rem;
    background: #2f3336;
    color: #e7e9ea;
    border: none;
    border-radius: 9999px;
    cursor: pointer;
  }

  .btn-small:hover { background: #3e4144; }

  .btn-icon {
    background: none;
    border: none;
    color: #8b949e;
    font-size: 1rem;
    cursor: pointer;
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
    line-height: 1;
  }

  .btn-icon:hover { background: #2f3336; color: #f85149; }

  .trigger-form {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .trigger-card {
    background: #16181c;
    border: 1px solid #2f3336;
    border-radius: 12px;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .form-row {
    display: flex;
    gap: 1.5rem;
    flex-wrap: wrap;
  }

  .form-group {
    flex: 1;
    min-width: 200px;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .form-label {
    font-size: 0.8rem;
    font-weight: 600;
    color: #8b949e;
  }

  .form-hint {
    font-size: 0.72rem;
    color: #484f58;
  }

  .form-input {
    background: #0d1117;
    border: 1px solid #2f3336;
    border-radius: 8px;
    padding: 0.5rem 0.75rem;
    color: #e7e9ea;
    font-size: 0.9rem;
    outline: none;
    width: 100%;
    box-sizing: border-box;
  }

  .form-input:focus { border-color: #1d9bf0; }
  .form-input::placeholder { color: #484f58; }

  h3 {
    font-size: 1rem;
    color: #e7e9ea;
    margin: 0;
  }

  .variables-section {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .variables-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .empty-text {
    color: #71767b;
    font-style: italic;
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

  .var-key { flex: 1; }
  .var-value { flex: 2; }

  .btn-remove { flex-shrink: 0; }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    border-top: 1px solid #2f3336;
    padding-top: 1rem;
  }

  .btn-run { padding: 0.6rem 2rem; }
</style>