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
  <div class="toolbar trigger-toolbar">
    <button class="btn btn-ghost" onclick={() => navigateTo('list')}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="19" y1="12" x2="5" y2="12"></line>
        <polyline points="12 19 5 12 12 5"></polyline>
      </svg>
      Back
    </button>
    <h2 class="toolbar-title">Trigger Pipeline</h2>
    <div class="toolbar-spacer"></div>
    {#if isRunAgain}
      <span class="run-again-badge">Pre-filled from previous pipeline</span>
    {/if}
  </div>

  <form class="trigger-form" onsubmit={handleRun}>
    <div class="panel trigger-panel">
      <div class="panel-body" style="gap: var(--space-6)">
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
            <label class="form-label" for="trigger-selector">Pipeline Selector</label>
            <input
              id="trigger-selector"
              type="text"
              class="form-input"
              placeholder="e.g. default, custom-pattern (optional)"
              bind:value={pipelineSelector}
            />
            <span class="form-hint">Filter which pipeline steps to execute</span>
          </div>
        </div>

        <div class="variables-section">
          <div class="variables-header">
            <h3 class="text-sm text-primary fw-600">Variables</h3>
            <button type="button" class="btn btn-secondary btn-sm" onclick={addVariableRow}>
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
                    class="btn btn-ghost btn-remove"
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
          <button type="submit" class="btn btn-primary" disabled={running}>
            {#if running}
              <span class="spinner"></span> Running…
            {:else}
              ▶ Run Pipeline
            {/if}
          </button>
        </div>
      </div>
    </div>
  </form>
</div>

<style>
  .trigger-pipeline {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
    padding: var(--space-6);
    height: 100%;
    overflow-y: auto;
  }

  .trigger-toolbar {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    flex-shrink: 0;
  }

  .toolbar-title {
    font-size: var(--font-size-md);
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
  }

  .toolbar-spacer { flex: 1; }

  .run-again-badge {
    background: var(--accent-muted);
    color: var(--accent-text);
    padding: var(--space-2) var(--space-4);
    border-radius: var(--radius-full);
    font-size: var(--font-size-xs);
    font-weight: 500;
  }

  .trigger-panel {
    max-width: 720px;
  }

  .form-row {
    display: flex;
    gap: var(--space-5);
    flex-wrap: wrap;
  }

  .form-group {
    flex: 1;
    min-width: 200px;
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .variables-section {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
  }

  .variables-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .empty-text {
    color: var(--text-tertiary);
    font-style: italic;
    font-size: var(--font-size-sm);
    padding: var(--space-4) 0;
  }

  .variables-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .variable-row {
    display: flex;
    gap: var(--space-3);
    align-items: center;
  }

  .var-key { flex: 1; }
  .var-value { flex: 2; }

  .btn-remove {
    flex-shrink: 0;
    font-size: 16px;
    color: var(--text-tertiary);
    padding: var(--space-2);
  }
  .btn-remove:hover { color: var(--error); background: var(--bg-hover); }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    border-top: 1px solid var(--border-subtle);
    padding-top: var(--space-5);
  }
</style>