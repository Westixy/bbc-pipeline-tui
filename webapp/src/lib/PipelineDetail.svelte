<script>
  import { get } from 'svelte/store';
  import { activeProjectId, activeProject, selectedPipeline, selectedSteps, selectedVariables, selectedLogVariables, detailState, showError, showSuccess, logStepName, logStepUUID, triggerPreTarget, triggerPreSelector, triggerPreVars, refreshTrigger } from '../stores/appState.js';
  import { page, navigateTo } from '../stores/router.js';
  import { getPipeline, listSteps, listVariables, getLogVariables, listRepositories, stopPipeline } from '../stores/api.js';
  import { formatDate, formatDuration, formatDurationCompact, statusLabel, statusClassForState } from './utils.js';

  let stopping = $state(false);
  let running = $state(false);

  async function loadDetail() {
    if (!$activeProject || !$selectedPipeline?.uuid) return;
    detailState.set('loading');
    try {
      const [pipeResp, varsResp, logVarsResp] = await Promise.all([
        getPipeline($activeProject.id, $selectedPipeline.uuid),
        listVariables($activeProject.id).catch(() => ({ variables: [] })),
        getLogVariables($activeProject.id, $selectedPipeline.uuid).catch(() => ({ variables: [] })),
      ]);

      // getPipeline returns { pipeline: {...}, steps: [...] }
      const pipelineData = pipeResp.pipeline || pipeResp;
      const stepsData = pipeResp.steps || [];
      const varsData = varsResp.variables || [];
      const logVarsData = logVarsResp.variables || [];

      selectedPipeline.set(pipelineData);
      selectedSteps.set(stepsData);
      selectedVariables.set(varsData);
      selectedLogVariables.set(logVarsData);
      detailState.set('ready');
    } catch (e) {
      showError(e.message);
      detailState.set('error');
    }
  }

  function handleRun() {
    if (!$activeProject || !$selectedPipeline) return;
    const target = $selectedPipeline.target?.ref_name || $selectedPipeline.target?.type;
    if (!target) {
      showError('No target branch found on this pipeline');
      return;
    }
    const variables = ($selectedLogVariables || []).map(v => ({ key: v.key, value: v.value }));
    const selector = $selectedPipeline.target?.selector || null;
    triggerPreTarget.set(target);
    triggerPreSelector.set(selector);
    triggerPreVars.set(variables);
    navigateTo('trigger');
  }

  async function handleStop() {
    if (!$activeProject || !$selectedPipeline?.uuid) return;
    if (!confirm('Are you sure you want to stop this pipeline?')) return;
    stopping = true;
    try {
      await stopPipeline($activeProject.id, $selectedPipeline.uuid);
      showSuccess('Pipeline stopped');
      await loadDetail();
    } catch (e) {
      showError(e.message);
    } finally {
      stopping = false;
    }
  }

  let detailLoadedFor = null; // plain variable — prevents effect re-entrance

  $effect(() => {
    if ($page !== 'detail' || !$activeProject || !$selectedPipeline?.uuid) return;
    const pipeId = `${$activeProject.id}-${$selectedPipeline.uuid}-${$refreshTrigger}`;
    if (detailLoadedFor === pipeId) return;
    detailLoadedFor = pipeId;
    loadDetail();
  });
</script>

<div class="pipeline-detail">
  <div class="detail-header">
    <button class="btn btn-secondary" onclick={() => navigateTo('list')}>
      ← Back to list
    </button>
    <h2>Pipeline #{ $selectedPipeline?.build_number || '—' }</h2>
    <button class="btn btn-primary" onclick={handleRun} disabled={running}>
      {running ? '⏳ Running...' : '▶ Run Pipeline'}
    </button>
    {#if $selectedPipeline?.state?.name === 'IN_PROGRESS' || $selectedPipeline?.state?.name === 'PENDING' || $selectedPipeline?.state?.name === 'IN_PROGRESS_STOPPING'}
      <button class="btn btn-danger" onclick={handleStop} disabled={stopping}>
        {stopping ? '⏳ Stopping...' : '⏹ Stop'}
      </button>
    {/if}
  </div>

  {#if $detailState === 'loading'}
    <div class="loading-state">
      <div class="spinner"></div>
      <p>Loading pipeline details...</p>
    </div>
  {:else if $detailState === 'error'}
    <div class="error-state">
      <p>❌ Failed to load details</p>
      <button class="btn btn-secondary" onclick={loadDetail}>Retry</button>
    </div>
  {:else}
    <!-- Pipeline Info Card -->
    <div class="info-card">
      <div class="info-grid">
        <div class="info-item">
          <span class="info-label">Status</span>
          <span class="status-badge {statusClassForState($selectedPipeline?.state)}">
            {statusLabel($selectedPipeline?.state)}
          </span>
        </div>
        <div class="info-item">
          <span class="info-label">Result</span>
          <span>{$selectedPipeline?.state?.result?.name || '—'}</span>
        </div>
        <div class="info-item">
          <span class="info-label">Target</span>
          <span class="target-badge">{$selectedPipeline?.target?.ref_name || $selectedPipeline?.target?.type || '—'}</span>
        </div>
        <div class="info-item">
          <span class="info-label">Trigger</span>
          <span>{$selectedPipeline?.trigger?.name || '—'}</span>
        </div>
        <div class="info-item">
          <span class="info-label">Creator</span>
          <span>{$selectedPipeline?.creator?.display_name || $selectedPipeline?.creator?.username || '—'}</span>
        </div>
        <div class="info-item">
          <span class="info-label">Duration</span>
          <span>{formatDuration($selectedPipeline?.created_on, $selectedPipeline?.completed_on, $selectedPipeline?.build_seconds_used || 0)}</span>
        </div>
        <div class="info-item">
          <span class="info-label">Created</span>
          <span class="info-date">{formatDate($selectedPipeline?.created_on)}</span>
        </div>
        <div class="info-item">
          <span class="info-label">Completed</span>
          <span class="info-date">{formatDate($selectedPipeline?.completed_on)}</span>
        </div>
        <div class="info-item">
          <span class="info-label">Commit</span>
          <span class="commit-hash">{$selectedPipeline?.target?.commit?.hash ? $selectedPipeline.target.commit.hash.substring(0, 8) : '—'}</span>
        </div>
      </div>
    </div>

    <!-- Steps -->
    <div class="steps-section">
      <h3>Steps ({$selectedSteps.length})</h3>
      {#if $selectedSteps.length === 0}
        <p class="empty-text">No steps found</p>
      {:else}
        <div class="steps-list">
          {#each $selectedSteps as step}
            <div class="step-item">
              <div class="step-header">
                <span class="step-name">{step.name || 'Unnamed step'}</span>
                <span class="status-badge step-status {statusClassForState(step.state)}">
                  {statusLabel(step.state)}
                </span>
              </div>
              <div class="step-meta">
                <span>Duration: {formatDurationCompact(step.duration_in_seconds || 0)}</span>
                <span>Run: {formatDurationCompact(step.run_duration_in_seconds || 0)}</span>
                <span>Build: {formatDurationCompact(step.build_duration_in_seconds || 0)}</span>
              </div>
              {#if step.state?.name !== 'NOT_STARTED'}
                <button
                  class="btn btn-small"
                  onclick={() => {
                    // Save pipeline UUID and step info for the log view
                    logStepName.set(step.name || 'Unnamed step');
                    logStepUUID.set(step.uuid);
                    navigateTo('logs', $selectedPipeline.uuid, step.uuid);
                  }}
                >
                  📜 View Log
                </button>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Variables -->
    {#if $selectedVariables.length > 0}
      <div class="variables-section">
        <h3>Variables ({$selectedVariables.length})</h3>
        <div class="info-grid">
          {#each $selectedVariables as v}
            <div class="info-item">
              <span class="info-label">{v.key}</span>
              <span>{v.secured ? '••••••••' : (v.value || '(empty)')}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Log-parsed Variables (from first step log) -->
    {#if $selectedLogVariables.length > 0}
      <div class="variables-section">
        <h3>Log Variables ({$selectedLogVariables.length})</h3>
        <p class="log-vars-note">Parsed from "Pipeline variables:" block in the first step's log</p>
        <div class="info-grid">
          {#each $selectedLogVariables as v}
            <div class="info-item">
              <span class="info-label">{v.key}</span>
              <span>{v.value || '(empty)'}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  .pipeline-detail {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .detail-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .detail-header h2 {
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

  .btn-danger {
    background: #b91c1c;
    color: #fff;
  }

  .btn-danger:hover {
    background: #991b1b;
  }

  .btn-danger:disabled {
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

  .info-card, .steps-section, .variables-section {
    background: #16181c;
    border: 1px solid #2f3336;
    border-radius: 12px;
    padding: 1.25rem;
  }

  h3 {
    font-size: 1rem;
    margin-bottom: 1rem;
    color: #e7e9ea;
  }

  .info-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 1rem;
  }

  .info-item {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .info-label {
    font-size: 0.75rem;
    color: #71767b;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    font-weight: 600;
  }

  .info-date {
    font-size: 0.85rem;
    color: #8b949e;
  }

  .status-badge {
    padding: 0.15rem 0.5rem;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 600;
    text-transform: uppercase;
    display: inline-block;
    width: fit-content;
  }

  .status-success { background: #1a3e2a; color: #3fb950; }
  .status-error { background: #3e1a1a; color: #f85149; }
  .status-running { background: #1d2e3e; color: #6cb6ff; }
  .status-stopped { background: #2f3336; color: #8b949e; }

  .target-badge {
    background: #1d2e3e;
    color: #6cb6ff;
    padding: 0.15rem 0.5rem;
    border-radius: 6px;
    font-size: 0.8rem;
    font-family: 'SF Mono', 'Fira Code', monospace;
  }

  .commit-hash {
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.85rem;
    color: #8b949e;
  }

  .empty-text {
    color: #71767b;
    font-style: italic;
  }

  .log-vars-note {
    color: #8b949e;
    font-size: 0.8rem;
    font-style: italic;
    margin-top: -0.75rem;
    margin-bottom: 0.75rem;
  }

  .steps-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .step-item {
    background: #1a1d23;
    border: 1px solid #2f3336;
    border-radius: 8px;
    padding: 0.75rem 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .step-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .step-name {
    font-weight: 500;
  }

  .step-status {
    font-size: 0.7rem;
  }

  .step-meta {
    display: flex;
    gap: 1rem;
    font-size: 0.8rem;
    color: #8b949e;
  }
</style>