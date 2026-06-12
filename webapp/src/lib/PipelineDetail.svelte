<script>
  import { onDestroy } from 'svelte';
  import { get } from 'svelte/store';
  import { activeProject, selectedPipeline, selectedSteps, selectedVariables, selectedLogVariables, detailState, showError, showSuccess, logStepName, logStepUUID, triggerPreTarget, triggerPreSelector, triggerPreVars, refreshTrigger } from '../stores/appState.js';
  import { page, pipelineUUIDFromUrl, workspaceFromUrl, repoSlugFromUrl, navigateTo } from '../stores/router.js';
  import { getPipeline, listVariables, getLogVariables, stopPipeline } from '../stores/api.js';
  import { formatDate, formatDuration, formatDurationCompact, statusLabel, statusClassForState, resolveStepStatus } from './utils.js';
  import ConfirmModal from './ConfirmModal.svelte';

  let stopping = $state(false);
  let running = $state(false);
  let autoRefreshRunning = $state(false);
  let autoRefreshInterval = null;
  let autoRefreshPipelineUuid = null;
  let elapsedInterval = null;
  let lastRefreshed = $state(null);
  let showStopConfirm = $state(false);
  let copiedHash = $state(false);
  let elapsed = $state('');
  let varsTab = $state('log'); // 'log' | 'pipeline' — prefer parsed log variables
  let varsFilter = $state('');
  let showAllVars = $state(false);
  const VARS_PREVIEW_COUNT = 6;

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
    autoRefreshPipelineUuid = $selectedPipeline?.uuid;
    autoRefreshRunning = true;
    elapsedInterval = setInterval(updateElapsed, 1000);
    autoRefreshInterval = setInterval(() => {
      if ($page !== 'detail' || $selectedPipeline?.uuid !== autoRefreshPipelineUuid) {
        stopAutoRefresh();
        return;
      }
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
    autoRefreshPipelineUuid = null;
  }

  onDestroy(() => { stopAutoRefresh(); });

  let lastPipelineUuid = null;
  let prevPage = $state(null);

  // Stop auto-refresh and clean up when navigating away or switching pipelines
  $effect(() => {
    const currentUuid = $selectedPipeline?.uuid;
    const currentPage = $page;
    // Only clean up if we were previously on detail and left
    if (prevPage === 'detail' && currentPage !== 'detail') {
      stopAutoRefresh();
      detailLoadedFor = null;
      lastProjectId = null;
      lastPipelineUuid = null;
      bootstrapped = false;
      selectedPipeline.set(null);
      selectedSteps.set([]);
      selectedVariables.set([]);
      selectedLogVariables.set([]);
      detailState.set('idle');
    }
    // Track pipeline UUID changes while staying on detail
    if (currentPage === 'detail' && currentUuid && currentUuid !== lastPipelineUuid) {
      stopAutoRefresh();
      detailLoadedFor = null;
    }
    lastPipelineUuid = currentUuid;
    prevPage = currentPage;
    return stopAutoRefresh;
  });

  async function loadDetail(silent = false) {
    if (!$activeProject || !$selectedPipeline?.uuid) return;
    if ($page !== 'detail') return; // Don't refresh if user navigated away
    if (!silent) detailState.set('loading');
    const fetchUuid = $selectedPipeline.uuid;
    const fetchProjectId = $activeProject.id;
    try {
      const [pipeResp, varsResp, logVarsResp] = await Promise.all([
        getPipeline(fetchProjectId, fetchUuid),
        listVariables(fetchProjectId).catch(() => ({ variables: [] })),
        getLogVariables(fetchProjectId, fetchUuid).catch(() => ({ variables: [] })),
      ]);

      // Guard: don't update stores if user navigated away or switched pipelines during fetch
      if ($page !== 'detail') return;
      if ($selectedPipeline?.uuid !== fetchUuid || $activeProject?.id !== fetchProjectId) return;

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

  function stepNodeTitle(step, i) {
    const name = step.name || `Step ${i + 1}`;
    const status = statusLabel(step.state);
    if (step.duration_in_seconds != null) {
      return `${name} — ${status} — ${formatDurationCompact(step.duration_in_seconds || 0)}`;
    }
    return `${name} — ${status}`;
  }

  function stepDotClass(state, idx) {
    const status = effectiveStepStatus(state, idx);
    if (status === 'SUCCESSFUL') return 'dot-success';
    if (status === 'FAILED') return 'dot-error';
    if (status === 'SKIPPED') return 'dot-skipped';
    if (status === 'IN_PROGRESS') return 'dot-running';
    if (status === 'STOPPED') return 'dot-stopped';
    return 'dot-pending';
  }

  function statusBadgeClass(state) {
    const name = state?.name || '';
    if (name === 'COMPLETED' && state?.result?.name === 'SUCCESSFUL') return 'badge-success';
    if (name === 'COMPLETED' && state?.result?.name === 'FAILED') return 'badge-error';
    if (name === 'FAILED' || name === 'ERROR') return 'badge-error';
    if (name === 'IN_PROGRESS' || name === 'PENDING') return 'badge-info';
    if (name === 'IN_PROGRESS_STOPPING' || name === 'STOPPED') return 'badge-warning';
    return 'badge-neutral';
  }

  function stepStatusBadgeClass(state, idx) {
    const status = effectiveStepStatus(state, idx);
    if (status === 'SUCCESSFUL') return 'badge-success';
    if (status === 'FAILED') return 'badge-error';
    if (status === 'SKIPPED') return 'badge-skipped';
    if (status === 'IN_PROGRESS') return 'badge-info';
    if (status === 'STOPPED') return 'badge-warning';
    return 'badge-neutral';
  }

  let lastProjectId = null;
  let detailLoadedFor = null;
  let bootstrapped = $state(false);

  // Main effect: load detail when selectedPipeline is available
  $effect(() => {
    if ($page !== 'detail' || !$activeProject || !$selectedPipeline?.uuid) return;
    if ($activeProject.id !== lastProjectId) {
      lastProjectId = $activeProject.id;
      detailLoadedFor = null;
    }
    const pipeId = `${$activeProject.id}-${$selectedPipeline.uuid}-${$refreshTrigger}`;
    if (detailLoadedFor === pipeId) return;
    detailLoadedFor = pipeId;
    loadDetail();
  });

  // Bootstrap from URL when stores are empty (page reload scenario)
  $effect(() => {
    const urlUuid = $pipelineUUIDFromUrl;
    if ($page !== 'detail' || !$activeProject || !urlUuid || bootstrapped) return;
    // Wait until the activeProject matches the URL workspace/repoSlug
    const urlWs = $workspaceFromUrl;
    const urlRs = $repoSlugFromUrl;
    if ($activeProject.workspace !== urlWs || $activeProject.repo_slug !== urlRs) return;
    // Don't double-load if selectedPipeline is already being loaded by the main effect
    if ($selectedPipeline?.uuid === urlUuid) return;
    bootstrapped = true;
    detailLoadedFor = `${$activeProject.id}-${urlUuid}-${$refreshTrigger}`;
    loadDetailFromUrl(urlUuid);
  });

  async function loadDetailFromUrl(uuid) {
    if ($page !== 'detail') return;
    detailState.set('loading');
    try {
      const [pipeResp, varsResp, logVarsResp] = await Promise.all([
        getPipeline($activeProject.id, uuid),
        listVariables($activeProject.id).catch(() => ({ variables: [] })),
        getLogVariables($activeProject.id, uuid).catch(() => ({ variables: [] })),
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
      showError(e.message);
      detailState.set('error');
    }
  }

  $effect(() => {
    function onRefresh() {
      if ($page === 'detail') { detailLoadedFor = null; loadDetail(); }
    }
    window.addEventListener('app:refresh', onRefresh);
    return () => window.removeEventListener('app:refresh', onRefresh);
  });

  let completedSteps = $derived($selectedSteps.filter(s => resolveStepStatus(s.state) === 'SUCCESSFUL').length);
  let finishedSteps = $derived($selectedSteps.filter(s => {
    const st = resolveStepStatus(s.state);
    return st === 'SUCCESSFUL' || st === 'FAILED' || st === 'STOPPED';
  }).length);
  let totalSteps = $derived($selectedSteps.length);
  let pipelineRunning = $derived($selectedPipeline?.state?.name === 'IN_PROGRESS' || $selectedPipeline?.state?.name === 'PENDING' || $selectedPipeline?.state?.name === 'IN_PROGRESS_STOPPING');
  let pipelineFailed = $derived($selectedPipeline?.state?.name === 'FAILED' || $selectedPipeline?.state?.result?.name === 'FAILED');
  let pipelineStopped = $derived($selectedPipeline?.state?.name === 'STOPPED' || $selectedPipeline?.state?.result?.name === 'STOPPED');
  let hasFailedStep = $derived($selectedSteps.some(s => resolveStepStatus(s.state) === 'FAILED'));
  let hasStoppedStep = $derived($selectedSteps.some(s => resolveStepStatus(s.state) === 'STOPPED'));
  let firstFailedIdx = $derived($selectedSteps.findIndex(s => resolveStepStatus(s.state) === 'FAILED'));
  let firstStoppedIdx = $derived($selectedSteps.findIndex(s => resolveStepStatus(s.state) === 'STOPPED'));
  let breakIdx = $derived(firstFailedIdx >= 0 ? firstFailedIdx : (firstStoppedIdx >= 0 ? firstStoppedIdx : -1));

  function effectiveStepStatus(state, idx) {
    const status = resolveStepStatus(state);
    if (breakIdx >= 0 && idx > breakIdx) return 'SKIPPED';
    return status;
  }

  let summaryItems = $derived.by(() => {
    const steps = $selectedSteps;
    const succeeded = steps.filter((s, i) => effectiveStepStatus(s.state, i) === 'SUCCESSFUL').length;
    const failed = steps.filter((s, i) => effectiveStepStatus(s.state, i) === 'FAILED').length;
    const skipped = steps.filter((s, i) => effectiveStepStatus(s.state, i) === 'SKIPPED').length;
    const running = steps.filter((s, i) => effectiveStepStatus(s.state, i) === 'IN_PROGRESS').length;
    const pending = steps.filter((s, i) => effectiveStepStatus(s.state, i) === 'PENDING').length;
    const stopped = steps.filter((s, i) => effectiveStepStatus(s.state, i) === 'STOPPED').length;
    const items = [];
    if (succeeded > 0) items.push({ label: 'Done', count: succeeded, icon: 'check', cls: 'chip-success' });
    if (failed > 0) items.push({ label: 'Failed', count: failed, icon: 'x', cls: 'chip-error' });
    if (skipped > 0) items.push({ label: 'Skipped', count: skipped, icon: 'skip', cls: 'chip-skipped' });
    if (running > 0) items.push({ label: 'Running', count: running, icon: 'loading', cls: 'chip-running' });
    if (pending > 0) items.push({ label: 'Pending', count: pending, icon: 'pending', cls: 'chip-pending' });
    if (stopped > 0) items.push({ label: 'Stopped', count: stopped, icon: 'clock', cls: 'chip-stopped' });
    return items;
  });

  // Default to log variables if they exist, fall back to pipeline variables
  let effectiveVarsTab = $derived.by(() => {
    if (varsTab === 'log' && hasLogVars) return 'log';
    if (varsTab === 'pipeline' && hasPipelineVars) return 'pipeline';
    if (hasLogVars) return 'log';
    if (hasPipelineVars) return 'pipeline';
    return 'log'; // irrelevant when empty
  });
  let currentVars = $derived(effectiveVarsTab === 'pipeline' ? $selectedVariables : $selectedLogVariables);
  let hasPipelineVars = $derived($selectedVariables.length > 0);
  let hasLogVars = $derived($selectedLogVariables.length > 0);
  let hasAnyVars = $derived(hasPipelineVars || hasLogVars);

  let filteredVars = $derived.by(() => {
    const filter = varsFilter.trim().toLowerCase();
    const vars = currentVars;
    if (!filter) return vars;
    return vars.filter(v => v.key.toLowerCase().includes(filter) || (v.value || '').toLowerCase().includes(filter));
  });

  let displayedVars = $derived(showAllVars ? filteredVars : filteredVars.slice(0, VARS_PREVIEW_COUNT));
  let hasMoreVars = $derived(filteredVars.length > VARS_PREVIEW_COUNT && !showAllVars);
</script>

<div class="pipeline-detail">
  <!-- Sticky Header Bar ──────────────────────────────────── -->
  <div class="sticky-header">
    <div class="header-row">
      <div class="header-left">
        <button class="btn btn-ghost" onclick={() => navigateTo('list')}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="19" y1="12" x2="5" y2="12"></line><polyline points="12 19 5 12 12 5"></polyline>
          </svg>
          Pipelines
        </button>
        <span class="header-sep">/</span>
        <span class="header-build">#{$selectedPipeline?.build_number || '—'}</span>
        <span class="badge {statusBadgeClass($selectedPipeline?.state)} status-lg">
          {statusLabel($selectedPipeline?.state)}
        </span>
        {#if $selectedPipeline?.state?.result?.name && statusLabel($selectedPipeline?.state) !== $selectedPipeline.state.result.name.toLowerCase()}
          <span class="result-tag">{$selectedPipeline.state.result.name}</span>
        {/if}
      </div>
      <div class="header-spacer"></div>
      <div class="header-actions">
        {#if autoRefreshRunning}
          <span class="live-indicator">
            <span class="live-dot"></span>
            Live
          </span>
        {/if}
        {#if pipelineRunning}
          <button class="btn btn-danger btn-sm" onclick={handleStop} disabled={stopping}>
            {stopping ? 'Stopping…' : 'Stop'}
          </button>
        {/if}
        <button class="btn btn-primary btn-sm" onclick={handleRun} disabled={running}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>
          Re-run
        </button>
        <button class="btn btn-secondary btn-sm" onclick={() => { detailLoadedFor = null; loadDetail(); }} title="Refresh">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"></polyline><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path></svg>
        </button>
      </div>
    </div>

    {#if autoRefreshRunning && lastRefreshed}
      <div class="refresh-sub">
        <span>Auto-refreshing every 5s — last updated {lastRefreshed.toLocaleTimeString()}</span>
        {#if elapsed}
          <span class="elapsed-badge">Elapsed {elapsed}</span>
        {/if}
      </div>
    {/if}
  </div>

  {#if $detailState === 'loading'}
    <div class="loading-container">
      <div class="spinner"></div>
      <p class="text-secondary">Loading pipeline details…</p>
    </div>
  {:else if $detailState === 'error'}
    <div class="empty-state">
      <span class="empty-icon">⚠</span>
      <h3>Failed to load pipeline details</h3>
      <p>An error occurred while fetching pipeline data.</p>
      <button class="btn btn-primary" onclick={() => loadDetail()}>Retry</button>
    </div>
  {:else}
    <!-- Metadata Strip ─────────────────────────────────────── -->
    <div class="metadata-strip">
      <div class="meta-field">
        <span class="meta-label">Branch</span>
        <span class="meta-value meta-branch">
          {$selectedPipeline?.target?.ref_name || $selectedPipeline?.target?.type || '—'}
        </span>
      </div>
      {#if $selectedPipeline?.target?.selector}
        <div class="meta-field">
          <span class="meta-label">Pattern</span>
          <span class="meta-value meta-pattern">
            {typeof $selectedPipeline.target.selector === 'object'
              ? ($selectedPipeline.target.selector.pattern || '—')
              : $selectedPipeline.target.selector}
          </span>
        </div>
      {/if}
      <div class="meta-field">
        <span class="meta-label">Trigger</span>
        <span class="meta-value">{$selectedPipeline?.trigger?.name || '—'}</span>
      </div>
      <div class="meta-field">
        <span class="meta-label">Creator</span>
        <span class="meta-value">{$selectedPipeline?.creator?.display_name || $selectedPipeline?.creator?.username || '—'}</span>
      </div>
      <div class="meta-field">
        <span class="meta-label">Duration</span>
        <span class="meta-value">{formatDuration($selectedPipeline?.created_on, $selectedPipeline?.completed_on, $selectedPipeline?.build_seconds_used || 0)}</span>
      </div>
      <div class="meta-field">
        <span class="meta-label">Created</span>
        <span class="meta-value">{formatDate($selectedPipeline?.created_on)}</span>
      </div>
      {#if $selectedPipeline?.completed_on}
        <div class="meta-field">
          <span class="meta-label">Completed</span>
          <span class="meta-value">{formatDate($selectedPipeline?.completed_on)}</span>
        </div>
      {/if}
      {#if $selectedPipeline?.target?.commit?.hash}
        <div class="meta-field">
          <span class="meta-label">Commit</span>
          <span class="meta-value commit-row">
            <code>{$selectedPipeline.target.commit.hash.substring(0, 8)}</code>
            <button class="btn btn-ghost btn-icon" onclick={copyCommitHash} title="Copy full commit hash">
              {copiedHash ? '✓' : ''}
              {#if !copiedHash}
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
              {/if}
            </button>
          </span>
        </div>
      {/if}
    </div>

    <!-- Pipeline Progress Timeline ──────────────────────── -->
    {#if totalSteps > 0}
      <div class="pipeline-timeline-section">
        <div class="timeline-header">
          <h3 class="timeline-title">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline>
            </svg>
            Pipeline Progress
          </h3>
          <div class="timeline-summary">
            {#each summaryItems as item}
              <span class="summary-chip {item.cls}">
                {#if item.icon === 'check'}<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"></polyline></svg>{/if}
                {#if item.icon === 'x'}<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>{/if}
                {#if item.icon === 'loading'}<span class="summary-spinner"></span>{/if}
                {#if item.icon === 'pending'}<span class="summary-dot-pending"></span>{/if}
                {#if item.icon === 'clock'}<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg>{/if}
                {item.label}: <strong>{item.count}</strong>
              </span>
            {/each}
          </div>
          <span class="timeline-percentage" class:done={finishedSteps === totalSteps && !pipelineFailed && !pipelineStopped} class:failed={pipelineFailed || hasFailedStep} class:stopped={pipelineStopped || hasStoppedStep}>
            {#if finishedSteps === totalSteps && !pipelineFailed && !pipelineStopped}✓{/if}
            {#if pipelineFailed || hasFailedStep}✕{/if}
            {#if pipelineStopped || hasStoppedStep}■{/if}
            {finishedSteps}/{totalSteps}
          </span>
        </div>

        <div class="timeline-track">
          {#each $selectedSteps as step, i}
            {@const stepStatus = effectiveStepStatus(step.state, i)}
            <div class="timeline-step" style="flex: 1;">
              <button
                class="timeline-step-node"
                class:node-success={stepStatus === 'SUCCESSFUL'}
                class:node-failed={stepStatus === 'FAILED'}
                class:node-skipped={stepStatus === 'SKIPPED'}
                class:node-running={stepStatus === 'IN_PROGRESS'}
                class:node-stopped={stepStatus === 'STOPPED'}
                class:node-pending={stepStatus === 'PENDING'}
                onclick={() => {
                  if (step.state?.name !== 'NOT_STARTED') {
                    logStepName.set(step.name || `Step ${i + 1}`);
                    logStepUUID.set(step.uuid);
                    navigateTo('logs', $selectedPipeline.uuid, i);
                  }
                }}
                title={stepNodeTitle(step, i)}
              >
                {#if stepStatus === 'SUCCESSFUL'}
                  <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"></polyline></svg>
                {:else if stepStatus === 'FAILED'}
                  <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
                {:else if stepStatus === 'IN_PROGRESS'}
                  <span class="node-spinner"></span>
                {:else if stepStatus === 'STOPPED'}
                  <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4" width="4" height="16"></rect><rect x="14" y="4" width="4" height="16"></rect></svg>
                {:else if stepStatus === 'SKIPPED'}
                  <span class="node-skip-dash">—</span>
                {:else}
                  <span class="node-empty-dot"></span>
                {/if}
              </button>
              {#if i < totalSteps - 1}
                {@const nextStatus = effectiveStepStatus($selectedSteps[i + 1]?.state, i + 1)}
                <div class="timeline-connector"
                  class:conn-done={stepStatus === 'SUCCESSFUL' && nextStatus !== 'PENDING'}
                  class:conn-failed={stepStatus === 'FAILED'}
                  class:conn-stopped={stepStatus === 'STOPPED'}
                  class:conn-skipped={stepStatus === 'SKIPPED'}
                  class:conn-running={stepStatus === 'IN_PROGRESS'}
                ></div>
              {/if}
              <div class="timeline-step-label-wrap">
                <span class="timeline-step-num">#{i + 1}</span>
                <span class="timeline-step-name">{step.name || ''}</span>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Two-Column Layout: Steps + Variables ──────────────── -->
    <div class="detail-panels">
      <!-- Left Column: Steps ───────────────────────────────── -->
      <div class="panel steps-panel">
        <div class="panel-header">
          <h3>Steps</h3>
          <span class="badge badge-neutral">{$selectedSteps.length}</span>
        </div>
        {#if $selectedSteps.length === 0}
          <p class="empty-text">No steps found for this pipeline.</p>
        {:else}
          <div class="steps-table">
            <div class="steps-header">
              <div class="col-dot"></div>
              <div class="col-name">STEP</div>
              <div class="col-status">STATUS</div>
              <div class="col-duration">DURATION</div>
              <div class="col-actions">LOG</div>
            </div>
            <div class="steps-body">
              {#each $selectedSteps as step, i}
                {@const effStatus = effectiveStepStatus(step.state, i)}
                <div
                  class="step-row"
                  class:step-row-active={effStatus === 'IN_PROGRESS'}
                  class:step-row-failed={effStatus === 'FAILED'}
                  class:step-row-skipped={effStatus === 'SKIPPED'}
                >
                  <div class="col-dot">
                    <span class="step-dot {stepDotClass(step.state, i)}"></span>
                  </div>
                  <div class="col-name" title={step.name}>
                    <span class="step-name-text">{step.name || `Step ${i + 1}`}</span>
                  </div>
                  <div class="col-status">
                    <span class="badge {stepStatusBadgeClass(step.state, i)} step-badge">
                      {effStatus === 'SKIPPED' ? 'skipped' : statusLabel(step.state)}
                    </span>
                  </div>
                  <div class="col-duration">
                    {#if step.duration_in_seconds != null}
                      <span class="duration-primary" title="Wall duration">⏱ {formatDurationCompact(step.duration_in_seconds || 0)}</span>
                    {/if}
                    {#if step.run_duration_in_seconds != null}
                      <span class="duration-secondary" title="Run duration">▶ {formatDurationCompact(step.run_duration_in_seconds || 0)}</span>
                    {/if}
                    {#if step.build_duration_in_seconds != null}
                      <span class="duration-secondary" title="Build duration">⚙ {formatDurationCompact(step.build_duration_in_seconds || 0)}</span>
                    {/if}
                    {#if step.duration_in_seconds == null && step.run_duration_in_seconds == null && step.build_duration_in_seconds == null}
                      <span class="text-tertiary">—</span>
                    {/if}
                  </div>
                  <div class="col-actions">
                    {#if step.state?.name !== 'NOT_STARTED'}
                      <button
                        class="btn btn-secondary btn-xs"
                        onclick={() => {
                          logStepName.set(step.name || `Step ${i + 1}`);
                          logStepUUID.set(step.uuid);
                          navigateTo('logs', $selectedPipeline.uuid, i);
                        }}
                      >
                        Log
                      </button>
                    {:else}
                      <span class="text-tertiary text-xs">—</span>
                    {/if}
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>

      <!-- Right Column: Variables ──────────────────────────── -->
      {#if hasAnyVars}
        <div class="panel vars-panel">
          <div class="panel-header vars-panel-header">
            <h3>Variables</h3>
            <div class="vars-tabs-segmented">
              {#if hasLogVars}
                <button
                  class="vars-tab-segment"
                  class:active={effectiveVarsTab === 'log'}
                  onclick={() => (varsTab = 'log')}
                >
                  Log
                  <span class="tab-count">{$selectedLogVariables.length}</span>
                </button>
              {/if}
              {#if hasPipelineVars}
                <button
                  class="vars-tab-segment"
                  class:active={effectiveVarsTab === 'pipeline'}
                  onclick={() => (varsTab = 'pipeline')}
                >
                  Pipeline
                  <span class="tab-count">{$selectedVariables.length}</span>
                </button>
              {/if}
            </div>
          </div>
          <div class="vars-body">
            {#if effectiveVarsTab === 'log' && hasLogVars}
              <p class="vars-hint">Parsed from "Pipeline variables:" block in the first step's log.</p>
            {/if}

            {#if currentVars.length > VARS_PREVIEW_COUNT}
              <div class="vars-search">
                <svg class="search-icon-sm" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line>
                </svg>
                <input
                  type="text"
                  class="vars-search-input"
                  placeholder="Filter variables…"
                  bind:value={varsFilter}
                />
              </div>
            {/if}

            {#if filteredVars.length === 0}
              <p class="empty-text">No variables match <code>{varsFilter}</code></p>
            {:else}
              <div class="vars-list">
                {#each displayedVars as v}
                  <div class="var-row">
                    <div class="var-row-left">
                      <code class="var-key">{v.key}</code>
                      {#if v.secured}
                        <span class="secured-tag">SECURED</span>
                      {/if}
                    </div>
                    <span class="var-value" class:var-value-masked={v.secured}>
                      {v.secured ? '••••••••••••••••' : (v.value || '(empty)')}
                    </span>
                  </div>
                {/each}
              </div>

              {#if hasMoreVars}
                <button class="vars-show-more" onclick={() => (showAllVars = true)}>
                  Show all {filteredVars.length} variables…
                </button>
              {/if}
              {#if showAllVars && filteredVars.length > VARS_PREVIEW_COUNT}
                <button class="vars-show-more" onclick={() => (showAllVars = false)}>
                  Show fewer
                </button>
              {/if}
            {/if}
          </div>
        </div>
      {:else}
        <div class="panel vars-panel vars-panel-empty">
          <div class="panel-header">
            <h3>Variables</h3>
          </div>
          <div class="vars-body">
            <p class="empty-text">No variables available for this pipeline.</p>
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>

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
    gap: 0;
    height: 100%;
    overflow-y: auto;
  }

  /* ── Sticky Header ──────────────────────────────────────── */
  .sticky-header {
    position: sticky;
    top: 0;
    z-index: 10;
    background: var(--bg-root);
    padding: var(--space-4) var(--space-6);
    border-bottom: 1px solid var(--border-default);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    flex-shrink: 0;
  }

  .header-row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-height: 32px;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .header-sep {
    color: var(--text-tertiary);
    font-size: var(--font-size-sm);
  }

  .header-build {
    font-weight: 700;
    font-size: var(--font-size-lg);
    font-family: var(--font-mono);
    color: var(--text-primary);
  }

  .status-lg {
    font-size: var(--font-size-xs);
    padding: var(--space-1) var(--space-4);
    border-radius: var(--radius-md);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .result-tag {
    font-size: 10px;
    color: var(--text-tertiary);
    background: var(--bg-input);
    padding: 1px 6px;
    border-radius: var(--radius-sm);
    text-transform: uppercase;
    font-weight: 600;
    letter-spacing: 0.03em;
  }

  .header-spacer { flex: 1; }

  .header-actions {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .live-indicator {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--accent-text);
    background: var(--accent-muted);
    padding: 2px 8px;
    border-radius: var(--radius-sm);
  }

  .live-dot {
    width: 6px; height: 6px;
    border-radius: 50%;
    background: var(--accent-text);
    animation: pulse-dot 1.5s ease-in-out infinite;
  }

  @keyframes pulse-dot {
    0%, 100% { opacity: 1; transform: scale(1); }
    50% { opacity: 0.4; transform: scale(0.7); }
  }

  .refresh-sub {
    display: flex;
    align-items: center;
    gap: var(--space-5);
    font-size: 11px;
    color: var(--text-tertiary);
  }

  .elapsed-badge {
    background: var(--bg-input);
    color: var(--accent-text);
    padding: 1px 8px;
    border-radius: var(--radius-sm);
    font-weight: 600;
    font-family: var(--font-mono);
  }

  /* ── Loading / Error ────────────────────────────────────── */
  .loading-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-4);
    padding: var(--space-9);
    flex: 1;
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-4);
    padding: var(--space-9);
    flex: 1;
    color: var(--text-secondary);
  }

  .empty-icon { font-size: 32px; }

  /* ── Metadata Strip ─────────────────────────────────────── */
  .metadata-strip {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-6);
    padding: var(--space-4) var(--space-6);
    border-bottom: 1px solid var(--border-subtle);
    flex-shrink: 0;
  }

  .meta-field {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 90px;
  }

  .meta-label {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-tertiary);
  }

  .meta-value {
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    font-weight: 500;
  }

  .meta-branch {
    font-family: var(--font-mono);
    color: var(--accent-text);
  }

  .meta-pattern {
    font-family: var(--font-mono);
    color: #d2a8ff;
    font-size: var(--font-size-xs);
  }

  .commit-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .commit-row code {
    font-family: var(--font-mono);
    font-size: var(--font-size-xs);
    background: var(--bg-input);
    padding: 1px 6px;
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
  }

  /* ── Pipeline Progress Timeline ─────────────────────────── */
  .pipeline-timeline-section {
    border-bottom: 1px solid var(--border-subtle);
    padding: var(--space-4) var(--space-6);
    flex-shrink: 0;
  }

  .timeline-header {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    margin-bottom: var(--space-5);
    flex-wrap: wrap;
  }

  .timeline-title {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--font-size-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-secondary);
    margin: 0;
    flex-shrink: 0;
  }

  .timeline-summary {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-wrap: wrap;
    flex: 1;
  }

  .summary-chip {
    display: flex;
    align-items: center;
    gap: 3px;
    font-size: 11px;
    font-weight: 500;
    padding: 2px 8px;
    border-radius: var(--radius-sm);
    font-family: var(--font-ui);
    white-space: nowrap;
  }
  .summary-chip.chip-success { background: rgba(52,211,153,0.12); color: var(--success); }
  .summary-chip.chip-error   { background: rgba(244,71,71,0.12);  color: var(--error); }
  .summary-chip.chip-running { background: var(--accent-muted);   color: var(--accent-text); }
  .summary-chip.chip-pending { background: var(--bg-input);       color: var(--text-tertiary); }
  .summary-chip.chip-skipped { background: var(--bg-input);       color: var(--text-tertiary); opacity: 0.65; }
  .summary-chip.chip-stopped { background: var(--warning-bg);     color: var(--warning); }
  .summary-chip strong { font-weight: 700; font-family: var(--font-mono); }

  .summary-spinner {
    width: 10px; height: 10px;
    border: 2px solid var(--border-default);
    border-top-color: var(--accent-text);
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }

  .summary-dot-pending {
    width: 8px; height: 8px;
    border-radius: 50%;
    border: 1.5px solid var(--text-tertiary);
    display: inline-block;
  }

  .timeline-percentage {
    font-family: var(--font-mono);
    font-size: var(--font-size-sm);
    font-weight: 700;
    color: var(--text-tertiary);
    flex-shrink: 0;
    min-width: 40px;
    text-align: right;
  }
  .timeline-percentage.done {
    color: var(--success);
  }
  .timeline-percentage.failed {
    color: var(--error);
  }
  .timeline-percentage.stopped {
    color: var(--warning);
  }

  /* ── Timeline Track ─────────────────────────────────── */
  .timeline-track {
    display: flex;
    align-items: flex-start;
    gap: 0;
    position: relative;
  }

  .timeline-step {
    display: flex;
    flex-direction: column;
    align-items: center;
    position: relative;
    padding-top: var(--space-1);
  }

  /* ── Step Node (clickable button) ──────────────────── */
  .timeline-step-node {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    border: 2px solid var(--border-default);
    background: var(--bg-root);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    color: var(--text-primary);
    transition: all var(--transition-fast);
    position: relative;
    z-index: 1;
    flex-shrink: 0;
    padding: 0;
  }
  .timeline-step-node:hover {
    border-color: var(--text-secondary);
    box-shadow: 0 0 0 4px rgba(255,255,255,0.04);
    transform: scale(1.1);
  }

  .node-success {
    background: rgba(52,211,153,0.15);
    border-color: var(--success);
    color: var(--success);
  }
  .node-failed {
    background: rgba(244,71,71,0.15);
    border-color: var(--error);
    color: var(--error);
  }
  .node-running {
    background: var(--accent-muted);
    border-color: var(--accent-text);
    color: var(--accent-text);
    animation: pulse-node-running 2s ease-in-out infinite;
  }
  @keyframes pulse-node-running {
    0%, 100% { box-shadow: 0 0 0 0 rgba(79,193,255,0.4); }
    50% { box-shadow: 0 0 0 8px rgba(79,193,255,0); }
  }
  .node-stopped {
    background: var(--warning-bg);
    border-color: var(--warning);
    color: var(--warning);
  }
  .node-skipped {
    background: var(--bg-root);
    border-color: var(--border-default);
    color: var(--text-tertiary);
    opacity: 0.5;
  }
  .node-pending {
    background: var(--bg-root);
    border-color: var(--border-default);
    color: var(--text-tertiary);
  }

  .node-spinner {
    width: 14px; height: 14px;
    border: 2px solid var(--border-default);
    border-top-color: var(--accent-text);
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }

  .node-skip-dash {
    font-size: 14px;
    font-weight: 700;
    color: var(--text-tertiary);
    line-height: 1;
    opacity: 0.4;
  }
  .node-empty-dot {
    width: 8px; height: 8px;
    border-radius: 50%;
    background: var(--border-default);
  }

  /* ── Connector Lines between Nodes ──────────────────── */
  .timeline-connector {
    position: absolute;
    top: 18px;
    left: calc(50% + 18px);
    width: calc(100% - 36px);
    height: 2px;
    background: var(--bg-input);
    z-index: 0;
  }
  .timeline-connector.conn-done {
    background: var(--success);
  }
  .timeline-connector.conn-failed {
    background: var(--error);
    opacity: 0.6;
  }
  .timeline-connector.conn-stopped {
    background: var(--border-emphasis);
    opacity: 0.5;
  }
  .timeline-connector.conn-skipped {
    background: var(--bg-input);
    opacity: 0.3;
  }
  .timeline-connector.conn-running {
    background: linear-gradient(90deg, var(--accent-text) 40%, var(--bg-input) 60%);
    background-size: 200% 100%;
    animation: connector-shimmer 1.5s ease-in-out infinite;
  }
  @keyframes connector-shimmer {
    0% { background-position: 100% 0; }
    100% { background-position: 0 0; }
  }

  /* ── Step Labels ─────────────────────────────────────── */
  .timeline-step-label-wrap {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1px;
    margin-top: var(--space-2);
    max-width: 100%;
    text-align: center;
  }

  .timeline-step-num {
    font-size: 9px;
    font-weight: 700;
    font-family: var(--font-mono);
    color: var(--text-tertiary);
    text-transform: uppercase;
  }

  .timeline-step-name {
    font-size: 10px;
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 90px;
  }

  /* ── Two-Column Detail Panels ───────────────────────────── */
  .detail-panels {
    display: flex;
    gap: var(--space-5);
    padding: var(--space-4) var(--space-6);
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  /* ── Steps Table (left column) ──────────────────────────── */
  .steps-panel {
    flex: 1;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .steps-table {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
  }

  .steps-header {
    display: flex;
    align-items: center;
    padding: var(--space-2) var(--space-4);
    background: var(--bg-input);
    border-bottom: 1px solid var(--border-default);
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-tertiary);
    flex-shrink: 0;
    gap: var(--space-2);
  }

  .steps-body {
    flex: 1;
    overflow-y: auto;
  }

  .step-row {
    display: flex;
    align-items: center;
    padding: var(--space-1) var(--space-4);
    border-bottom: 1px solid var(--border-subtle);
    min-height: 36px;
    transition: background-color 0.15s;
    gap: var(--space-2);
  }
  .step-row:hover { background: var(--bg-hover); }
  .step-row-active { background: var(--bg-selection); border-bottom-color: var(--accent); }
  .step-row-failed { background: rgba(244,71,71,0.04); }
  .step-row-skipped { opacity: 0.55; }

  .col-dot {
    width: 16px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .step-dot {
    width: 8px; height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
    transition: all 0.3s;
  }
  .dot-success { background: var(--success); }
  .dot-error   { background: var(--error); }
  .dot-running { background: var(--accent-text); animation: pulse-dot 1.5s ease-in-out infinite; }
  .dot-stopped { background: var(--border-emphasis); }
  .dot-skipped { background: var(--text-tertiary); opacity: 0.4; }
  .dot-pending { background: var(--bg-root); border: 1.5px solid var(--border-default); }

  .col-name {
    flex: 1;
    min-width: 0;
  }

  .step-name-text {
    font-size: var(--font-size-sm);
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    display: block;
  }

  .col-status {
    width: 90px;
    flex-shrink: 0;
    display: flex;
    justify-content: center;
  }

  .step-badge {
    font-size: 10px;
    padding: 1px 8px;
    font-weight: 600;
    text-transform: uppercase;
    white-space: nowrap;
  }

  .col-duration {
    width: 140px;
    flex-shrink: 0;
    display: flex;
    gap: var(--space-4);
    font-family: var(--font-mono);
    font-size: 12px;
    color: var(--text-secondary);
    justify-content: flex-end;
    padding-right: var(--space-4);
  }

  .duration-primary { color: var(--text-primary); font-weight: 500; }
  .duration-secondary { color: var(--text-tertiary); font-size: 11px; }
  .text-xs { font-size: var(--font-size-xs); }
  .text-tertiary { color: var(--text-tertiary); }

  .col-actions {
    width: 60px;
    flex-shrink: 0;
    display: flex;
    justify-content: flex-end;
  }

  .empty-text {
    color: var(--text-tertiary);
    font-style: italic;
    font-size: var(--font-size-sm);
    padding: var(--space-5);
  }

  /* ── Variables Panel (right column) ─────────────────────── */
  .vars-panel {
    width: 320px;
    flex-shrink: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    border-left: 1px solid var(--border-default);
  }
  .vars-panel-empty {
    opacity: 0.5;
  }

  .vars-panel-header {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-5);
    background: var(--bg-panel-raised);
    border-bottom: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg) var(--radius-lg) 0 0;
    user-select: none;
    flex-shrink: 0;
  }
  .vars-panel-header h3 {
    font-size: var(--font-size-xs);
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    margin: 0;
  }

  .vars-tabs-segmented {
    display: flex;
    gap: 0;
    background: var(--bg-input);
    border-radius: var(--radius-md);
    padding: 2px;
    width: 100%;
  }

  .vars-tab-segment {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-sm);
    border: none;
    background: transparent;
    color: var(--text-tertiary);
    font-family: var(--font-ui);
    font-size: var(--font-size-xs);
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition-fast);
    white-space: nowrap;
  }
  .vars-tab-segment:hover { color: var(--text-primary); }
  .vars-tab-segment.active {
    background: var(--bg-panel);
    color: var(--text-primary);
    box-shadow: 0 1px 3px rgba(0,0,0,0.12);
    font-weight: 600;
  }
  .tab-count {
    font-size: 10px;
    font-weight: 600;
    background: var(--bg-badge);
    color: var(--text-tertiary);
    padding: 0 5px;
    border-radius: var(--radius-sm);
    font-family: var(--font-mono);
  }

  .vars-body {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-3);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .vars-hint {
    font-size: var(--font-size-xs);
    color: var(--text-tertiary);
    font-style: italic;
    margin: 0;
    padding: var(--space-1) var(--space-2);
  }

  .vars-search {
    position: relative;
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }
  .search-icon-sm {
    position: absolute;
    left: 7px;
    color: var(--text-tertiary);
    pointer-events: none;
  }
  .vars-search-input {
    width: 100%;
    padding: var(--space-1) var(--space-3) var(--space-1) 26px;
    font-family: var(--font-ui);
    font-size: var(--font-size-xs);
    color: var(--text-primary);
    background: var(--bg-input);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-sm);
    outline: none;
  }
  .vars-search-input:focus { border-color: var(--border-focus); }
  .vars-search-input::placeholder { color: var(--text-tertiary); }

  .vars-list {
    display: flex;
    flex-direction: column;
    gap: 1px;
    flex: 1;
    overflow-y: auto;
    min-height: 0;
  }

  .var-row {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    padding: var(--space-2) var(--space-3);
    border-radius: var(--radius-sm);
    background: var(--bg-panel);
    transition: background var(--transition-fast);
  }
  .var-row:hover { background: var(--bg-hover); }

  .var-row-left {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-wrap: wrap;
  }

  .var-key {
    font-family: var(--font-mono);
    font-size: var(--font-size-xs);
    font-weight: 600;
    color: var(--accent-text);
    background: var(--accent-muted);
    padding: 1px 6px;
    border-radius: var(--radius-sm);
    flex-shrink: 0;
  }

  .secured-tag {
    font-size: 9px;
    color: var(--warning);
    background: var(--warning-bg);
    padding: 1px 5px;
    border-radius: var(--radius-sm);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    flex-shrink: 0;
  }

  .var-value {
    font-family: var(--font-mono);
    font-size: var(--font-size-xs);
    color: var(--text-primary);
    word-break: break-all;
    line-height: 1.5;
    padding-left: var(--space-2);
  }
  .var-value-masked {
    color: var(--text-tertiary);
    letter-spacing: 0.15em;
  }

  .vars-show-more {
    display: block;
    width: 100%;
    padding: var(--space-2);
    border: 1px dashed var(--border-default);
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--accent-text);
    font-family: var(--font-ui);
    font-size: var(--font-size-xs);
    font-weight: 500;
    cursor: pointer;
    text-align: center;
    transition: all var(--transition-fast);
    flex-shrink: 0;
  }
  .vars-show-more:hover {
    background: var(--bg-hover);
    border-color: var(--accent-text);
    color: var(--accent-text);
  }
</style>