<script>
  import { onDestroy } from 'svelte';
  import { get } from 'svelte/store';
  import { activeProject, selectedPipeline, selectedSteps, selectedVariables, selectedLogVariables, detailState, showError, showSuccess, logStepName, logStepUUID, triggerPreTarget, triggerPreSelector, triggerPreVars, refreshTrigger } from '../stores/appState.js';
  import { page, navigateTo } from '../stores/router.js';
  import { getPipeline, listVariables, getLogVariables, stopPipeline } from '../stores/api.js';
  import { formatDate, formatDuration, formatDurationCompact, statusLabel, statusClassForState } from './utils.js';
  import ConfirmModal from './ConfirmModal.svelte';

  let stopping = $state(false);
  let running = $state(false);
  let autoRefreshRunning = $state(false);
  let autoRefreshInterval = null;
  let lastRefreshed = $state(null);
  let showStopConfirm = $state(false);
  let copiedHash = $state(false);
  let showPipelineVars = $state(false);
  let showLogVars = $state(false);
  let elapsed = $state('');

  let elapsedInterval = null;

  function updateElapsed() {
    const pipe = $selectedPipeline;
    if (!pipe?.created_on) { elapsed = ''; return; }
    const created = new Date(pipe.created_on);
    const now = new Date();
    const diff = Math.floor((now - created) / 1000);
    if (diff < 60) elapsed = `${diff}s`;
    else if (diff < 3600) elapsed = `${Math.floor(diff / 60)}m ${diff % 60}s`;
    else {
      const h = Math.floor(diff / 3600);
      const m = Math.floor((diff % 3600) / 60);
      elapsed = `${h}h ${m}m`;
    }
  }

  function startAutoRefresh() {
    if (autoRefreshInterval) return;
    autoRefreshRunning = true;
    elapsedInterval = setInterval(updateElapsed, 1000);
    autoRefreshInterval = setInterval(() => {
      const pipe = $selectedPipeline;
      if (pipe?.state?.name === 'IN_PROGRESS' || pipe?.state?.name === 'PENDING' || pipe?.state?.name === 'IN_PROGRESS_STOPPING') {
        loadDetail(true);
      } else {
        stopAutoRefresh();
      }
    }, 5000);
  }

  function stopAutoRefresh() {
    if (autoRefreshInterval) { clearInterval(autoRefreshInterval); autoRefreshInterval = null; }
    if (elapsedInterval) { clearInterval(elapsedInterval); elapsedInterval = null; }
    autoRefreshRunning = false;
  }

  onDestroy(() => {
    stopAutoRefresh();
  });

  async function loadDetail(silent = false) {
    if (!$activeProject || !$selectedPipeline?.uuid) return;
    if (!silent) detailState.set('loading');
    try {
      const [pipeResp, varsResp, logVarsResp] = await Promise.all([
        getPipeline($activeProject.id, $selectedPipeline.uuid),
        listVariables($activeProject.id).catch(() => ({ variables: [] })),
        getLogVariables($activeProject.id, $selectedPipeline.uuid).catch(() => ({ variables: [] })),
      ]);

      const pipelineData = pipeResp.pipeline || pipeResp;
      const stepsData = pipeResp.steps || [];
      const varsData = varsResp.variables || [];
      const logVarsData = logVarsResp.variables || [];

      selectedPipeline.set(pipelineData);
      selectedSteps.set(stepsData);
      selectedVariables.set(varsData);
      selectedLogVariables.set(logVarsData);
      detailState.set('ready');
      lastRefreshed = new Date();

      updateElapsed();

      if (pipelineData?.state?.name === 'IN_PROGRESS' || pipelineData?.state?.name === 'PENDING' || pipelineData?.state?.name === 'IN_PROGRESS_STOPPING') {
        startAutoRefresh();
      } else {
        stopAutoRefresh();
      }
    } catch (e) {
      if (!silent) {
        showError(e.message);
        detailState.set('error');
      }
    }
  }

  function handleRun() {
    if (!$activeProject || !$selectedPipeline) return;
    const target = $selectedPipeline.target?.ref_name || $selectedPipeline.target?.type;
    if (!target) { showError('No target branch found on this pipeline'); return; }
    const variables = ($selectedLogVariables || []).map(v => ({ key: v.key, value: v.value }));
    const selector = $selectedPipeline.target?.selector || null;
    triggerPreTarget.set(target);
    triggerPreSelector.set(selector);
    triggerPreVars.set(variables);
    navigateTo('trigger');
  }

  async function handleStop() {
    if (!$activeProject || !$selectedPipeline?.uuid) return;
    showStopConfirm = true;
  }

  async function confirmStop() {
    showStopConfirm = false;
    stopping = true;
    try {
      await stopPipeline($activeProject.id, $selectedPipeline.uuid);
      showSuccess('Pipeline stopped');
      await loadDetail();
    } catch (e) { showError(e.message); }
    finally { stopping = false; }
  }

  async function copyCommitHash() {
    const hash = $selectedPipeline?.target?.commit?.hash;
    if (!hash) return;
    try { await navigator.clipboard.writeText(hash); }
    catch {
      const ta = document.createElement('textarea'); ta.value = hash;
      ta.style.position = 'fixed'; ta.style.opacity = '0'; document.body.appendChild(ta);
      ta.select(); document.execCommand('copy'); document.body.removeChild(ta);
    }
    copiedHash = true;
    setTimeout(() => (copiedHash = false), 2000);
  }

  function stepIcon(state) {
    const name = state?.name;
    if (name === 'SUCCESSFUL') return '✅';
    if (name === 'FAILED') return '❌';
    if (name === 'IN_PROGRESS') return '🔄';
    if (name === 'STOPPED') return '⏹';
    if (name === 'PAUSED') return '⏸';
    if (name === 'NOT_STARTED') return '⬜';
    if (name === 'EXPIRED') return '⏰';
    return '⬜';
  }

  function stepDotClass(state) {
    const name = state?.name;
    if (name === 'SUCCESSFUL') return 'dot-success';
    if (name === 'FAILED' || name === 'ERROR') return 'dot-error';
    if (name === 'IN_PROGRESS') return 'dot-running';
    if (name === 'STOPPED' || name === 'PAUSED' || name === 'EXPIRED') return 'dot-stopped';
    return 'dot-pending';
  }

  let detailLoadedFor = null;

  $effect(() => {
    if ($page !== 'detail' || !$activeProject || !$selectedPipeline?.uuid) return;
    const pipeId = `${$activeProject.id}-${$selectedPipeline.uuid}-${$refreshTrigger}`;
    if (detailLoadedFor === pipeId) return;
    detailLoadedFor = pipeId;
    loadDetail();
  });

  $effect(() => {
    function onRefresh() {
      if ($page === 'detail') { detailLoadedFor = null; loadDetail(); }
    }
    window.addEventListener('app:refresh', onRefresh);
    return () => window.removeEventListener('app:refresh', onRefresh);
  });
</script>

<div class="pipeline-detail">
  <!-- ─── Navigation Bar ──────────────────────────────────────────────────── -->
  <nav class="top-nav">
    <button class="btn btn-ghost" onclick={() => navigateTo('list')}>
      <span class="nav-icon">←</span> Back to list
    </button>
    <div class="nav-actions">
      {#if $selectedPipeline?.state?.name === 'IN_PROGRESS' || $selectedPipeline?.state?.name === 'PENDING' || $selectedPipeline?.state?.name === 'IN_PROGRESS_STOPPING'}
        <button class="btn btn-danger" onclick={handleStop} disabled={stopping}>
          {stopping ? '⏳ Stopping...' : '⏹ Stop Pipeline'}
        </button>
      {/if}
      <button class="btn btn-primary" onclick={handleRun} disabled={running}>
        {running ? '⏳ Running...' : '▶ Re-run Pipeline'}
      </button>
    </div>
  </nav>

  {#if $detailState === 'loading'}
    <div class="loading-state">
      <div class="spinner"></div>
      <p>Loading pipeline details…</p>
    </div>
  {:else if $detailState === 'error'}
    <div class="error-state">
      <p>❌ Failed to load pipeline details</p>
      <button class="btn btn-secondary" onclick={loadDetail}>Retry</button>
    </div>
  {:else}
    <!-- ─── Hero Header Card ──────────────────────────────────────────────── -->
    <div class="hero-card">
      <div class="hero-top">
        <div class="hero-status">
          <span class="status-pill status-pill-lg {statusClassForState($selectedPipeline?.state)}">
            {statusLabel($selectedPipeline?.state)}
          </span>
          {#if $selectedPipeline?.state?.result?.name}
            <span class="result-tag">{$selectedPipeline.state.result.name}</span>
          {/if}
        </div>
        <div class="hero-build">
          <span class="build-label">Pipeline</span>
          <span class="build-number">#{ $selectedPipeline?.build_number || '—' }</span>
        </div>
      </div>

      <div class="hero-meta">
        <div class="meta-item">
          <span class="meta-icon">⎇</span>
          <span class="meta-label">Branch</span>
          <span class="meta-value meta-branch">
            {$selectedPipeline?.target?.ref_name || $selectedPipeline?.target?.type || '—'}
          </span>
        </div>
        {#if $selectedPipeline?.target?.selector}
          <div class="meta-item">
            <span class="meta-icon">◈</span>
            <span class="meta-label">Pattern</span>
            <span class="meta-value meta-pattern">
              {typeof $selectedPipeline.target.selector === 'object'
                ? ($selectedPipeline.target.selector.pattern || '—')
                : $selectedPipeline.target.selector}
            </span>
          </div>
        {/if}
        <div class="meta-item">
          <span class="meta-icon">▶</span>
          <span class="meta-label">Trigger</span>
          <span class="meta-value">{$selectedPipeline?.trigger?.name || '—'}</span>
        </div>
        <div class="meta-item">
          <span class="meta-icon">👤</span>
          <span class="meta-label">Creator</span>
          <span class="meta-value">{$selectedPipeline?.creator?.display_name || $selectedPipeline?.creator?.username || '—'}</span>
        </div>
      </div>

      <div class="hero-stats">
        <div class="stat">
          <span class="stat-value">{formatDuration($selectedPipeline?.created_on, $selectedPipeline?.completed_on, $selectedPipeline?.build_seconds_used || 0)}</span>
          <span class="stat-label">Duration</span>
        </div>
        <div class="stat">
          <span class="stat-value">{formatDate($selectedPipeline?.created_on)}</span>
          <span class="stat-label">Created</span>
        </div>
        <div class="stat">
          <span class="stat-value">{formatDate($selectedPipeline?.completed_on)}</span>
          <span class="stat-label">Completed</span>
        </div>
        {#if $selectedPipeline?.target?.commit?.hash}
          <div class="stat">
            <span class="stat-value stat-commit">
              <code>{$selectedPipeline.target.commit.hash.substring(0, 8)}</code>
              <button class="copy-btn" onclick={copyCommitHash} title="Copy full commit hash">
                {copiedHash ? '✓ Copied!' : '📋'}
              </button>
            </span>
            <span class="stat-label">Commit</span>
          </div>
        {/if}
      </div>

      {#if autoRefreshRunning}
        <div class="auto-refresh-bar">
          <span class="refresh-dot"></span>
          Auto-refreshing — last updated {lastRefreshed?.toLocaleTimeString() || 'now'}
          {#if elapsed}
            <span class="elapsed-badge">Running {elapsed}</span>
          {/if}
        </div>
      {/if}

      <!-- Progress indicator -->
      {#if $selectedSteps.length > 0}
        {@const completed = $selectedSteps.filter(s => s.state?.name === 'SUCCESSFUL' || s.state?.name === 'FAILED' || s.state?.name === 'STOPPED').length}
        {@const total = $selectedSteps.length}
        <div class="progress-bar">
          <div class="progress-segments">
            {#each $selectedSteps as step}
              <div
                class="progress-segment {stepDotClass(step.state)}"
                style="width: {100 / total}%"
                title="{step.name || 'Step'}: {statusLabel(step.state)}"
              ></div>
            {/each}
          </div>
          <span class="progress-text">{completed}/{total} steps completed</span>
        </div>
      {/if}
    </div>

    <!-- ─── Step Timeline ─────────────────────────────────────────────────── -->
    <div class="steps-section">
      <div class="section-header">
        <h3>Steps</h3>
        <span class="section-badge">{$selectedSteps.length}</span>
      </div>
      {#if $selectedSteps.length === 0}
        <p class="empty-text">No steps found for this pipeline.</p>
      {:else}
        <div class="timeline">
          {#each $selectedSteps as step, i}
            <div class="timeline-row">
              <!-- Timeline connector -->
              <div class="timeline-track">
                <div class="timeline-dot {stepDotClass(step.state)}" title={statusLabel(step.state)}></div>
                {#if i < $selectedSteps.length - 1}
                  <div class="timeline-line" class:line-done={step.state?.name === 'SUCCESSFUL'}></div>
                {/if}
              </div>

              <!-- Step card -->
              <div class="step-card" class:step-active={step.state?.name === 'IN_PROGRESS'} class:step-failed={step.state?.name === 'FAILED'}>
                <div class="step-card-header">
                  <span class="step-icon">{stepIcon(step.state)}</span>
                  <div class="step-title">
                    <span class="step-name">{step.name || 'Unnamed step'}</span>
                    <span class="status-badge step-status-pill {statusClassForState(step.state)}">
                      {statusLabel(step.state)}
                    </span>
                  </div>
                </div>

                <div class="step-meta-bar">
                  {#if step.duration_in_seconds != null}
                    <span class="step-meta-item" title="Wall-clock duration">
                      <span class="meta-dot meta-dot-wall"></span> {formatDurationCompact(step.duration_in_seconds || 0)}
                    </span>
                  {/if}
                  {#if step.run_duration_in_seconds != null}
                    <span class="step-meta-item" title="Run duration">
                      <span class="meta-dot meta-dot-run"></span> {formatDurationCompact(step.run_duration_in_seconds || 0)}
                    </span>
                  {/if}
                  {#if step.build_duration_in_seconds != null}
                    <span class="step-meta-item" title="Build duration">
                      <span class="meta-dot meta-dot-build"></span> {formatDurationCompact(step.build_duration_in_seconds || 0)}
                    </span>
                  {/if}
                </div>

                {#if step.state?.name !== 'NOT_STARTED'}
                  <button
                    class="btn btn-outline btn-sm"
                    onclick={() => {
                      logStepName.set(step.name || 'Unnamed step');
                      logStepUUID.set(step.uuid);
                      navigateTo('logs', $selectedPipeline.uuid, i);
                    }}
                  >
                    📜 View Log
                  </button>
                {:else}
                  <span class="step-not-started">Not started yet</span>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- ─── Pipeline Variables (Collapsible) ───────────────────────────────── -->
    {#if $selectedVariables.length > 0}
      <div class="collapsible-section">
        <button class="collapsible-header" onclick={() => (showPipelineVars = !showPipelineVars)}>
          <span class="collapsible-arrow" class:expanded={showPipelineVars}>▸</span>
          <h3>Pipeline Variables</h3>
          <span class="section-badge">{$selectedVariables.length}</span>
          <span class="collapsible-hint">{showPipelineVars ? '(click to collapse)' : '(click to expand)'}</span>
        </button>
        {#if showPipelineVars}
          <div class="collapsible-body">
            <div class="vars-grid">
              {#each $selectedVariables as v}
                <div class="var-item">
                  <code class="var-key">{v.key}</code>
                  <span class="var-value">{v.secured ? '••••••••' : (v.value || '(empty)')}</span>
                  {#if v.secured}
                    <span class="secured-tag">🔒 secured</span>
                  {/if}
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {/if}

    <!-- ─── Log Variables (Collapsible) ────────────────────────────────────── -->
    {#if $selectedLogVariables.length > 0}
      <div class="collapsible-section">
        <button class="collapsible-header" onclick={() => (showLogVars = !showLogVars)}>
          <span class="collapsible-arrow" class:expanded={showLogVars}>▸</span>
          <h3>Log Variables</h3>
          <span class="section-badge">{$selectedLogVariables.length}</span>
          <span class="collapsible-hint">{showLogVars ? '(click to collapse)' : '(click to expand)'}</span>
        </button>
        <p class="log-vars-note">Parsed from "Pipeline variables:" block in the first step's log</p>
        {#if showLogVars}
          <div class="collapsible-body">
            <div class="vars-grid">
              {#each $selectedLogVariables as v}
                <div class="var-item">
                  <code class="var-key">{v.key}</code>
                  <span class="var-value">{v.value || '(empty)'}</span>
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {/if}
  {/if}
</div>

<!-- Stop Pipeline Confirmation Modal -->
<ConfirmModal
  open={showStopConfirm}
  title="Stop Pipeline"
  message="Are you sure you want to stop pipeline #{$selectedPipeline?.build_number || '—'}? This action cannot be undone."
  confirmText="Stop Pipeline"
  danger={true}
  onconfirm={confirmStop}
  oncancel={() => (showStopConfirm = false)}
/>

<style>
  .pipeline-detail {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  /* ── Top Navigation ──────────────────────────────────────────────────── */
  .top-nav {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  .nav-actions {
    display: flex;
    gap: 0.5rem;
  }

  .btn {
    padding: 0.45rem 1rem;
    border: none;
    border-radius: 8px;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s;
    white-space: nowrap;
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
  }

  .btn-ghost {
    background: transparent;
    color: #8b949e;
    padding: 0.4rem 0.6rem;
  }
  .btn-ghost:hover { color: #e7e9ea; background: #21262d; }

  .btn-primary {
    background: #1d9bf0;
    color: #fff;
  }
  .btn-primary:hover { background: #1a8cd8; }
  .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

  .btn-danger {
    background: #da3633;
    color: #fff;
  }
  .btn-danger:hover { background: #c53030; }
  .btn-danger:disabled { opacity: 0.5; cursor: not-allowed; }

  .btn-secondary {
    background: #2f3336;
    color: #e7e9ea;
  }
  .btn-secondary:hover { background: #3e4144; }

  .btn-outline {
    background: transparent;
    border: 1px solid #30363d;
    color: #c9d1d9;
    padding: 0.3rem 0.75rem;
    font-size: 0.78rem;
  }
  .btn-outline:hover { background: #21262d; border-color: #484f58; }

  .btn-sm {
    padding: 0.25rem 0.65rem;
    font-size: 0.78rem;
  }

  .nav-icon { font-size: 1rem; }

  /* ── Loading / Error ──────────────────────────────────────────────────── */
  .loading-state, .error-state {
    text-align: center;
    padding: 4rem 2rem;
    background: #16181c;
    border-radius: 12px;
    border: 1px solid #2f3336;
    color: #71767b;
  }

  .spinner {
    width: 36px; height: 36px;
    border: 3px solid #2f3336;
    border-top: 3px solid #1d9bf0;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin: 0 auto 0.75rem;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  /* ── Hero Card ────────────────────────────────────────────────────────── */
  .hero-card {
    background: linear-gradient(135deg, #16181c 0%, #1a1d23 100%);
    border: 1px solid #2f3336;
    border-radius: 14px;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .hero-top {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .hero-status {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .status-pill {
    padding: 0.2rem 0.7rem;
    border-radius: 8px;
    font-size: 0.78rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .status-pill-lg {
    padding: 0.35rem 1rem;
    font-size: 0.9rem;
    border-radius: 10px;
  }

  .status-pill.status-success { background: #1a3e2a; color: #3fb950; }
  .status-pill.status-error   { background: #3e1a1a; color: #f85149; }
  .status-pill.status-running { background: #1d2e3e; color: #6cb6ff; }
  .status-pill.status-stopped { background: #2f3336; color: #8b949e; }

  .result-tag {
    background: #21262d;
    color: #8b949e;
    padding: 0.15rem 0.6rem;
    border-radius: 6px;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
  }

  .hero-build {
    margin-left: auto;
    display: flex;
    flex-direction: column;
    align-items: flex-end;
  }

  .build-label {
    font-size: 0.7rem;
    color: #484f58;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    font-weight: 600;
  }

  .build-number {
    font-size: 1.5rem;
    font-weight: 700;
    color: #e7e9ea;
    font-family: 'SF Mono', 'Fira Code', monospace;
    letter-spacing: -0.02em;
  }

  /* ── Hero Meta ────────────────────────────────────────────────────────── */
  .hero-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 1.5rem;
  }

  .meta-item {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    min-width: 120px;
  }

  .meta-icon {
    font-size: 0.7rem;
    color: #484f58;
  }

  .meta-label {
    font-size: 0.68rem;
    color: #484f58;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    font-weight: 600;
  }

  .meta-value {
    font-size: 0.9rem;
    color: #c9d1d9;
    font-weight: 500;
  }

  .meta-branch {
    font-family: 'SF Mono', 'Fira Code', monospace;
    color: #6cb6ff;
  }

  .meta-pattern {
    font-family: 'SF Mono', 'Fira Code', monospace;
    color: #d2a8ff;
    font-size: 0.82rem;
  }

  /* ── Hero Stats ───────────────────────────────────────────────────────── */
  .hero-stats {
    display: flex;
    flex-wrap: wrap;
    gap: 1.5rem;
    padding-top: 0.75rem;
    border-top: 1px solid #21262d;
  }

  .stat {
    display: flex;
    flex-direction: column;
    gap: 0.12rem;
    min-width: 80px;
  }

  .stat-value {
    font-size: 0.92rem;
    font-weight: 600;
    color: #c9d1d9;
  }

  .stat-value code {
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.82rem;
    background: #21262d;
    padding: 0.1rem 0.4rem;
    border-radius: 4px;
    color: #8b949e;
  }

  .stat-commit {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }

  .copy-btn {
    background: none;
    border: none;
    cursor: pointer;
    font-size: 0.85rem;
    padding: 0.1rem 0.25rem;
    border-radius: 4px;
    color: #8b949e;
    transition: all 0.15s;
  }
  .copy-btn:hover { color: #e7e9ea; background: #30363d; }

  .stat-label {
    font-size: 0.68rem;
    color: #484f58;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    font-weight: 600;
  }

  /* ── Auto-refresh bar ──────────────────────────────────────────────────── */
  .auto-refresh-bar {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.8rem;
    background: #161f2c;
    border-radius: 8px;
    font-size: 0.75rem;
    color: #6cb6ff;
  }

  .refresh-dot {
    width: 8px; height: 8px;
    border-radius: 50%;
    background: #1d9bf0;
    animation: pulse-dot 1.5s ease-in-out infinite;
  }
  @keyframes pulse-dot {
    0%, 100% { opacity: 1; transform: scale(1); }
    50% { opacity: 0.4; transform: scale(0.7); }
  }

  .elapsed-badge {
    margin-left: auto;
    background: #1d2e3e;
    color: #6cb6ff;
    padding: 0.15rem 0.6rem;
    border-radius: 12px;
    font-size: 0.75rem;
    font-weight: 600;
    font-family: 'SF Mono', 'Fira Code', monospace;
  }

  /* ── Progress Bar ──────────────────────────────────────────────────────── */
  .progress-bar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .progress-segments {
    display: flex;
    flex: 1;
    gap: 3px;
    height: 6px;
    border-radius: 3px;
    overflow: hidden;
  }

  .progress-segment {
    height: 100%;
    border-radius: 1px;
    transition: background 0.3s;
  }

  .progress-segment.dot-success { background: #3fb950; }
  .progress-segment.dot-error   { background: #f85149; }
  .progress-segment.dot-running { background: #6cb6ff; animation: shimmer-progress 1.5s ease-in-out infinite; background-size: 200% 100%; background-image: linear-gradient(90deg, #6cb6ff 25%, #9dcfff 50%, #6cb6ff 75%); }
  .progress-segment.dot-stopped { background: #484f58; }
  .progress-segment.dot-pending { background: #21262d; }

  @keyframes shimmer-progress {
    0% { background-position: -200% 0; }
    100% { background-position: 200% 0; }
  }

  .progress-text {
    font-size: 0.72rem;
    color: #484f58;
    white-space: nowrap;
    font-weight: 600;
  }

  /* ── Steps Section ────────────────────────────────────────────────────── */
  .steps-section {
    background: #16181c;
    border: 1px solid #2f3336;
    border-radius: 14px;
    padding: 1.25rem 1.5rem;
  }

  .section-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 1.25rem;
  }

  .section-header h3 {
    font-size: 1rem;
    font-weight: 600;
    color: #e7e9ea;
    margin: 0;
  }

  .section-badge {
    background: #21262d;
    color: #8b949e;
    padding: 0.1rem 0.5rem;
    border-radius: 10px;
    font-size: 0.75rem;
    font-weight: 600;
  }

  .empty-text {
    color: #484f58;
    font-style: italic;
    font-size: 0.88rem;
  }

  /* ── Timeline ─────────────────────────────────────────────────────────── */
  .timeline {
    display: flex;
    flex-direction: column;
  }

  .timeline-row {
    display: flex;
    gap: 1rem;
    min-height: 60px;
  }

  .timeline-track {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 24px;
    flex-shrink: 0;
  }

  .timeline-dot {
    width: 14px;
    height: 14px;
    border-radius: 50%;
    border: 2px solid #30363d;
    background: #16181c;
    flex-shrink: 0;
    margin-top: 6px;
    transition: all 0.3s;
  }

  .dot-success { background: #3fb950; border-color: #3fb950; }
  .dot-error   { background: #f85149; border-color: #f85149; }
  .dot-running { background: #6cb6ff; border-color: #6cb6ff; animation: pulse-dot 1.5s ease-in-out infinite; }
  .dot-stopped { background: #484f58; border-color: #484f58; }
  .dot-pending { background: #16181c; border-color: #30363d; }

  .timeline-line {
    width: 2px;
    flex: 1;
    background: #21262d;
    margin-top: 4px;
  }

  .line-done {
    background: #3fb950;
  }

  /* ── Step Card ────────────────────────────────────────────────────────── */
  .step-card {
    flex: 1;
    background: #1a1d23;
    border: 1px solid #21262d;
    border-radius: 10px;
    padding: 0.75rem 1rem;
    margin-bottom: 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    transition: border-color 0.2s, box-shadow 0.2s;
  }

  .step-card:hover {
    border-color: #30363d;
  }

  .step-active {
    border-color: #1d9bf0;
    box-shadow: 0 0 0 1px rgba(29,155,240,0.15);
    background: #13233a;
  }

  .step-failed {
    border-color: #da3633;
    box-shadow: 0 0 0 1px rgba(218,54,51,0.12);
  }

  .step-card-header {
    display: flex;
    align-items: center;
    gap: 0.65rem;
  }

  .step-icon {
    font-size: 1rem;
    flex-shrink: 0;
    width: 20px;
    text-align: center;
  }

  .step-title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex: 1;
    flex-wrap: wrap;
  }

  .step-name {
    font-weight: 500;
    font-size: 0.9rem;
    color: #e7e9ea;
  }

  .step-status-pill {
    font-size: 0.68rem;
    padding: 0.12rem 0.45rem;
    border-radius: 5px;
    font-weight: 600;
    text-transform: uppercase;
    margin-left: auto;
  }

  .status-badge {
    display: inline-block;
  }
  .status-success { background: #1a3e2a; color: #3fb950; }
  .status-error   { background: #3e1a1a; color: #f85149; }
  .status-running { background: #1d2e3e; color: #6cb6ff; animation: shimmer-text 2s ease-in-out infinite; background-size: 200% 100%; background-image: linear-gradient(90deg, #1d2e3e 50%, #17283a 100%); }
  .status-stopped { background: #2f3336; color: #8b949e; }

  @keyframes shimmer-text {
    0% { color: #6cb6ff; }
    50% { color: #9dcfff; }
    100% { color: #6cb6ff; }
  }

  .step-meta-bar {
    display: flex;
    gap: 1rem;
    font-size: 0.78rem;
    color: #8b949e;
  }

  .step-meta-item {
    display: flex;
    align-items: center;
    gap: 0.3rem;
  }

  .meta-dot {
    width: 6px; height: 6px;
    border-radius: 50%;
    display: inline-block;
  }

  .meta-dot-wall  { background: #8b949e; }
  .meta-dot-run   { background: #d2a8ff; }
  .meta-dot-build { background: #7ee787; }

  .step-not-started {
    font-size: 0.75rem;
    color: #484f58;
    font-style: italic;
  }

  /* ── Collapsible Sections ────────────────────────────────────────────── */
  .collapsible-section {
    background: #16181c;
    border: 1px solid #2f3336;
    border-radius: 14px;
    padding: 1rem 1.5rem;
  }

  .collapsible-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    background: none;
    border: none;
    color: #e7e9ea;
    cursor: pointer;
    padding: 0.25rem 0;
    font-size: 0.95rem;
  }

  .collapsible-header h3 {
    margin: 0;
    font-size: 0.95rem;
    font-weight: 600;
  }

  .collapsible-arrow {
    font-size: 0.8rem;
    color: #484f58;
    transition: transform 0.2s;
    display: inline-block;
  }

  .collapsible-arrow.expanded {
    transform: rotate(90deg);
  }

  .collapsible-hint {
    font-size: 0.65rem;
    color: #484f58;
    margin-left: auto;
  }

  .collapsible-body {
    margin-top: 0.75rem;
    padding-top: 0.75rem;
    border-top: 1px solid #21262d;
  }

  .log-vars-note {
    color: #484f58;
    font-size: 0.75rem;
    font-style: italic;
    margin: 0.25rem 0 -0.25rem 0;
    padding-left: 1.35rem;
  }

  .vars-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
    gap: 0.65rem;
  }

  .var-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.6rem;
    background: #1a1d23;
    border-radius: 6px;
    border: 1px solid #21262d;
    flex-wrap: wrap;
  }

  .var-key {
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.78rem;
    color: #d2a8ff;
    background: #1d1d2e;
    padding: 0.1rem 0.35rem;
    border-radius: 3px;
  }

  .var-value {
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.78rem;
    color: #7ee787;
    word-break: break-all;
    flex: 1;
  }

  .secured-tag {
    font-size: 0.68rem;
    color: #484f58;
    margin-left: auto;
  }
</style>