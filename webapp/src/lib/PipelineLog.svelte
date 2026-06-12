<script>
  import { onDestroy } from 'svelte';
  import { activeProject, selectedPipeline, selectedSteps, logContent, logStepName, logStepUUID, showError, refreshTrigger } from '../stores/appState.js';
  import { navigateTo } from '../stores/router.js';
  import { getStepLog } from '../stores/api.js';

  let logState = $state('idle'); // 'idle' | 'loading' | 'ready' | 'error'
  let logError = $state('');
  let currentStepIndex = $state(-1);
  let autoRefreshInterval = null;

  async function loadLog(stepIndex) {
    if (!$activeProject || !$selectedPipeline?.uuid || !$selectedSteps[stepIndex]) return;

    const step = $selectedSteps[stepIndex];
    logState = 'loading';
    logError = '';

    try {
      const data = await getStepLog($activeProject.id, $selectedPipeline.uuid, step.uuid);
      logContent.set(data.log || 'No log content available');
      logStepName.set(step.name || `Step ${stepIndex + 1}`);
      logStepUUID.set(step.uuid);
      currentStepIndex = stepIndex;
      logState = 'ready';
    } catch (e) {
      logError = e.message;
      logState = 'error';
    }
  }

  function handleRefresh() {
    if ($logStepUUID) {
      const idx = $selectedSteps.findIndex(s => s.uuid === $logStepUUID);
      if (idx >= 0) loadLog(idx);
    }
  }

  function startAutoRefresh(stepIndex) {
    stopAutoRefresh();
    const step = $selectedSteps[stepIndex];
    if (!step) return;
    // Only auto-refresh for actively running steps
    if (step.state?.name === 'IN_PROGRESS' || step.state?.name === 'PENDING') {
      autoRefreshInterval = setInterval(() => {
        if (logState !== 'loading') {
          loadLog(stepIndex);
        }
      }, 5000);
    }
  }

  function stopAutoRefresh() {
    if (autoRefreshInterval) {
      clearInterval(autoRefreshInterval);
      autoRefreshInterval = null;
    }
  }

  let logLoadedForStep = null; // plain variable — prevents re-entrant loops

  // Auto-load the first step's log on mount, and react to refreshTrigger
  $effect(() => {
    if ($selectedSteps.length === 0 || !$selectedSteps[0]?.uuid) return;
    const stepId = `${$selectedSteps[0].uuid}-${$refreshTrigger}`;
    if (logLoadedForStep === stepId) return;
    logLoadedForStep = stepId;
    loadLog(0);
    startAutoRefresh(0);
  });

  onDestroy(() => {
    stopAutoRefresh();
  });
</script>

<div class="pipeline-log">
  <div class="log-header">
    <button class="btn btn-secondary" onclick={() => navigateTo('detail', $selectedPipeline?.uuid)}>
      ← Back to detail
    </button>
    <div class="log-title">
      <h2>Log: {$logStepName || 'Step Log'}</h2>
      <span class="log-pipe-info">Pipeline #{$selectedPipeline?.build_number || '—'}</span>
    </div>
    <div class="log-actions">
      <button class="btn btn-secondary" onclick={handleRefresh}>
        🔄 Refresh
      </button>
    </div>
  </div>

  <!-- Step selector tabs -->
  {#if $selectedSteps.length > 0}
    <div class="step-tabs">
      {#each $selectedSteps as step, i}
        <button
          class="step-tab"
          class:active={$logStepUUID === step.uuid}
          onclick={() => {
            loadLog(i);
            startAutoRefresh(i);
            navigateTo('logs', $selectedPipeline?.uuid, step.uuid);
          }}
        >
          <span class="tab-icon">
            {#if step.state?.name === 'COMPLETED'}
              ✅
            {:else if step.state?.name === 'FAILED' || step.state?.name === 'ERROR'}
              ❌
            {:else if step.state?.name === 'IN_PROGRESS' || step.state?.name === 'PENDING'}
              🔄
            {:else}
              ⬜
            {/if}
          </span>
          {step.name || `Step ${i + 1}`}
        </button>
      {/each}
    </div>
  {/if}

  {#if logState === 'loading'}
    <div class="loading-state">
      <div class="spinner"></div>
      <p>Loading log...</p>
    </div>
  {:else if logState === 'error'}
    <div class="error-state">
      <p>❌ {logError}</p>
      <button class="btn btn-secondary" onclick={handleRefresh}>Retry</button>
    </div>
  {:else}
    <div class="log-viewer">
      <pre class="log-content">{$logContent}</pre>
    </div>
  {/if}
</div>

<style>
  .pipeline-log {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .log-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .log-title {
    display: flex;
    align-items: baseline;
    gap: 0.75rem;
  }

  .log-title h2 {
    font-size: 1.3rem;
    font-weight: 600;
  }

  .log-pipe-info {
    color: #71767b;
    font-size: 0.85rem;
  }

  .log-actions {
    margin-left: auto;
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

  .step-tabs {
    display: flex;
    gap: 0.25rem;
    overflow-x: auto;
    padding-bottom: 0.25rem;
  }

  .step-tab {
    background: #16181c;
    border: 1px solid #2f3336;
    color: #71767b;
    padding: 0.5rem 0.75rem;
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.85rem;
    white-space: nowrap;
    transition: all 0.15s;
    display: flex;
    align-items: center;
    gap: 0.35rem;
  }

  .step-tab:hover {
    background: #1a1d23;
    color: #e7e9ea;
  }

  .step-tab.active {
    background: #1d2e3e;
    border-color: #1d9bf0;
    color: #6cb6ff;
  }

  .tab-icon {
    font-size: 0.75rem;
  }

  .loading-state, .error-state {
    text-align: center;
    padding: 3rem;
    background: #16181c;
    border-radius: 12px;
    border: 1px solid #2f3336;
    color: #71767b;
  }

  .spinner {
    width: 30px;
    height: 30px;
    border: 3px solid #2f3336;
    border-top: 3px solid #1d9bf0;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin: 0 auto 0.5rem;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .log-viewer {
    background: #0d1117;
    border: 1px solid #2f3336;
    border-radius: 12px;
    overflow: hidden;
    max-height: 70vh;
  }

  .log-content {
    padding: 1.25rem;
    font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
    font-size: 0.8rem;
    line-height: 1.6;
    color: #7ee787;
    white-space: pre-wrap;
    word-break: break-word;
    overflow-y: auto;
    max-height: 70vh;
    margin: 0;
  }

  .log-content::-webkit-scrollbar {
    width: 8px;
  }

  .log-content::-webkit-scrollbar-track {
    background: #0d1117;
  }

  .log-content::-webkit-scrollbar-thumb {
    background: #30363d;
    border-radius: 4px;
  }

  .log-content::-webkit-scrollbar-thumb:hover {
    background: #484f58;
  }
</style>