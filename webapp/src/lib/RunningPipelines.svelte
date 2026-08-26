<script>
  import { get } from 'svelte/store';
  import {
    activeProjectId,
    projects,
    runningPipelines,
    runningPipelinesState,
    runningPipelinesErrors,
    refreshTrigger,
    selectedPipeline,
    selectedSteps,
    selectedVariables,
    selectedLogVariables,
    detailState,
    showError,
  } from '../stores/appState.js';
  import { page, navigateTo, isNewTabClick, openInNewTab } from '../stores/router.js';
  import { listRunningPipelines, getLogVariables } from '../stores/api.js';
  import { formatDate, statusLabel } from './utils.js';

  let autoRefresh = $state(false);
  // Auto-refresh interval selector: 'asap' | '1m' | '5m' | 'pause'.
  let refreshInterval = $state('1m');
  let tick = $state(Date.now());

  const REFRESH_MS = {
    asap: 5000,
    '1m': 60_000,
    '5m': 300_000,
  };

  // Repository filter: 'all' or a configured project id (as string).
  let repoFilter = $state('all');

  // Options for the repo dropdown, derived from ALL configured projects.
  let repoOptions = $derived(
    ($projects || []).map((p) => ({
      id: String(p.id),
      name: p.name || `${p.workspace}/${p.repo_slug}`,
    })),
  );

  // Pipelines shown after applying the repository filter.
  let displayedPipelines = $derived.by(() => {
    const list = $runningPipelines;
    if (repoFilter === 'all') return list;
    return list.filter((p) => String(p.project_id) === repoFilter);
  });

  let displayedRepoCount = $derived(new Set(displayedPipelines.map((p) => p.project_id)).size);

  function statusToBadgeClass(state) {
    const name = state?.name || '';
    if (name === 'COMPLETED' && state?.result?.name === 'SUCCESSFUL') return 'badge-success';
    if (name === 'COMPLETED' && state?.result?.name === 'FAILED') return 'badge-error';
    if (name === 'FAILED' || name === 'ERROR') return 'badge-error';
    if (name === 'IN_PROGRESS' || name === 'PENDING') return 'badge-info';
    if (name === 'IN_PROGRESS_STOPPING' || name === 'STOPPED') return 'badge-warning';
    return 'badge-neutral';
  }

  function shortHash(hash) {
    if (!hash) return '';
    return hash.substring(0, 7);
  }

  function triggerLabel(trigger) {
    const name = (trigger?.name || '').toLowerCase();
    if (name === 'push') return 'push';
    if (name === 'manual') return 'manual';
    if (name === 'schedule' || name === 'scheduled') return 'schedule';
    if (name === 'pull_request' || name === 'pullrequest') return 'PR';
    return trigger?.name || '—';
  }

  /** Live wall-clock elapsed time since the pipeline was created. */
  function elapsedLabel(pipe) {
    if (!pipe?.created_on) return '—';
    const created = new Date(pipe.created_on).getTime();
    if (isNaN(created)) return '—';
    const diff = Math.floor((tick - created) / 1000);
    if (diff < 0) return '0s';
    if (diff < 60) return `${diff}s`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ${diff % 60}s`;
    const h = Math.floor(diff / 3600);
    const m = Math.floor((diff % 3600) / 60);
    return `${h}h ${m}m`;
  }

  async function load() {
    if (get(page) !== 'running') return;
    runningPipelinesState.set('loading');
    try {
      const data = await listRunningPipelines();
      if (get(page) !== 'running') return;
      runningPipelines.set(data.running || []);
      runningPipelinesErrors.set(data.errors || []);
      runningPipelinesState.set('ready');
    } catch (e) {
      if (get(page) !== 'running') return;
      runningPipelinesState.set('error');
      showError(e.message);
    }
  }

  function openDetail(e, item) {
    // Ctrl/Cmd/Shift + click (or middle-click) opens the detail in a new tab
    // without switching the active project in the current tab.
    if (isNewTabClick(e)) {
      e.preventDefault();
      openInNewTab('detail', item.pipeline.uuid, undefined, {
        workspace: item.workspace,
        repo_slug: item.repo_slug,
      });
      return;
    }
    // Switch the active project FIRST so navigateTo('detail') builds the
    // correct URL from activeProject.
    activeProjectId.set(item.project_id);
    // Navigate before setting stores (matches PipelineList.viewDetail) so
    // PipelineDetail's cleanup $effect doesn't clear the pipeline we just set.
    navigateTo('detail', item.pipeline.uuid);
    selectedPipeline.set(item.pipeline);
    selectedSteps.set([]);
    selectedVariables.set([]);
    selectedLogVariables.set([]);
    detailState.set('loading');
  }

  function truncateVal(val) {
    if (!val) return '—';
    return val.length > 60 ? val.slice(0, 60) + '…' : val;
  }

  // ── Hover popover for log variables ──────────────────────────────────
  let hoveredPipeline = $state(null);
  let hoverVars = $state([]);
  let hoverVarsLoading = $state(false);
  let hoverVarsError = $state('');
  let hoverPos = $state({ x: 0, y: 0 });
  let hoverTimer = null;
  let hoverActive = $state(false);

  // Cache log variables per pipeline UUID (never expires within a session).
  const hoverVarsCache = new Map();

  async function fetchHoverVars(item) {
    if (!item?.pipeline?.uuid) return;
    const uuid = item.pipeline.uuid;
    const cached = hoverVarsCache.get(uuid);
    if (cached) {
      hoverVars = cached;
      hoverVarsLoading = false;
      hoverVarsError = '';
      return;
    }
    hoverVarsLoading = true;
    hoverVarsError = '';
    hoverVars = [];
    try {
      // Each running entry carries its own project_id, so fetch using the
      // project that actually owns this pipeline (not the active project).
      const data = await getLogVariables(item.project_id, uuid);
      const vars = data.variables || [];
      hoverVars = vars;
      hoverVarsCache.set(uuid, vars);
    } catch (e) {
      hoverVarsError = e.message;
    } finally {
      hoverVarsLoading = false;
    }
  }

  function onRowMouseEnter(e, item) {
    if (hoverTimer) clearTimeout(hoverTimer);
    hoverPos = { x: e.clientX, y: e.clientY };
    hoverTimer = setTimeout(() => {
      hoveredPipeline = item.pipeline;
      hoverActive = true;
      fetchHoverVars(item);
    }, 400);
  }

  function onRowMouseMove(e) {
    if (hoverActive) {
      hoverPos = { x: e.clientX, y: e.clientY };
    }
  }

  function onRowMouseLeave() {
    if (hoverTimer) clearTimeout(hoverTimer);
    hoverTimer = null;
    hoverActive = false;
    hoveredPipeline = null;
    hoverVars = [];
    hoverVarsError = '';
  }

  // While on the running page: tick every second (for live elapsed times).
  // The data refresh interval is driven by the refreshInterval selector.
  $effect(() => {
    if ($page !== 'running') {
      autoRefresh = false;
      return;
    }
    const tickId = setInterval(() => { tick = Date.now(); }, 1000);
    const ms = REFRESH_MS[refreshInterval];
    let refreshId = null;
    if (ms) {
      refreshId = setInterval(() => load(), ms);
      autoRefresh = true;
    } else {
      autoRefresh = false;
    }
    return () => {
      clearInterval(tickId);
      if (refreshId) clearInterval(refreshId);
      autoRefresh = false;
    };
  });

  // Initial load (and reload whenever the global refresh button is clicked).
  $effect(() => {
    void $refreshTrigger;
    if ($page === 'running') load();
  });
</script>

<div class="running-page">
  <!-- Toolbar ─────────────────────────────────────────────── -->
  <div class="toolbar">
    <div class="toolbar-left">
      <h2 class="toolbar-title">Running Pipelines</h2>
      {#if $runningPipelinesState === 'ready'}
        <span class="toolbar-count">
          <span class="mono">{displayedPipelines.length}</span>
          <span class="count-label"> running pipeline{displayedPipelines.length !== 1 ? 's' : ''}</span>
          {#if repoFilter !== 'all'}
            <span class="count-filtered"> of {$runningPipelines.length}</span>
          {/if}
          <span class="count-label">across</span>
          <span class="mono">{displayedRepoCount}</span>
          <span class="count-label">repo{displayedRepoCount !== 1 ? 's' : ''}</span>
        </span>
      {/if}
    </div>
    <div class="toolbar-right">
      <select class="form-select repo-select" bind:value={repoFilter} aria-label="Filter by repository">
        <option value="all">All repositories</option>
        {#each repoOptions as opt}
          <option value={opt.id}>{opt.name}</option>
        {/each}
      </select>
      {#if autoRefresh}
        <span class="live-indicator">
          <span class="live-dot"></span>
          auto-refresh
        </span>
      {:else}
        <span class="live-indicator paused">
          <span class="live-dot"></span>
          paused
        </span>
      {/if}
      <select
        class="form-select interval-select"
        bind:value={refreshInterval}
        aria-label="Auto-refresh interval"
        title="Auto-refresh interval"
      >
        <option value="pause">PAUSE</option>
        <option value="asap">ASAP</option>
        <option value="1m">1m</option>
        <option value="5m">5m</option>
      </select>
      <button class="btn btn-secondary btn-sm" onclick={load} disabled={$runningPipelinesState === 'loading'}>
        {#if $runningPipelinesState === 'loading'}
          <span class="spinner spinner-xs"></span>
        {/if}
        Refresh
      </button>
    </div>
  </div>

  <!-- Per-repo errors ─────────────────────────────────────── -->
  {#if $runningPipelinesErrors.length > 0}
    <div class="errors-banner">
      <span class="errors-banner-title">⚠ Some repositories could not be reached</span>
      <ul>
        {#each $runningPipelinesErrors as err}
          <li>{err}</li>
        {/each}
      </ul>
    </div>
  {/if}

  <!-- Loading skeleton (first load) ───────────────────────── -->
  {#if $runningPipelinesState === 'loading' && $runningPipelines.length === 0}
    <div class="panel data-grid">
      <div class="data-grid-header">
        <div class="data-grid-cell col-project">PROJECT</div>
        <div class="data-grid-cell col-build">#</div>
        <div class="data-grid-cell col-branch">TARGET</div>
        <div class="data-grid-cell col-trig">VIA</div>
        <div class="data-grid-cell col-status-hdr">STATUS</div>
        <div class="data-grid-cell col-dur">ELAPSED</div>
        <div class="data-grid-cell col-creator">CREATOR</div>
        <div class="data-grid-cell col-date">CREATED</div>
      </div>
      {#each Array(8) as _}
        <div class="data-grid-row skeleton-row">
          <div class="data-grid-cell col-project"><div class="skeleton sk-project"></div></div>
          <div class="data-grid-cell col-build"><div class="skeleton sk-num"></div></div>
          <div class="data-grid-cell col-branch"><div class="skeleton sk-branch"></div></div>
          <div class="data-grid-cell col-trig"><div class="skeleton sk-trig"></div></div>
          <div class="data-grid-cell col-status-hdr"><div class="skeleton sk-badge"></div></div>
          <div class="data-grid-cell col-dur"><div class="skeleton sk-dur"></div></div>
          <div class="data-grid-cell col-creator"><div class="skeleton sk-name"></div></div>
          <div class="data-grid-cell col-date"><div class="skeleton sk-date"></div></div>
        </div>
      {/each}
    </div>

  <!-- Error state ─────────────────────────────────────────── -->
  {:else if $runningPipelinesState === 'error' && $runningPipelines.length === 0}
    <div class="empty-state">
      <div class="empty-icon">!</div>
      <h3>Failed to load running pipelines</h3>
      <p>Could not fetch pipeline status from Bitbucket. Check your connection and try again.</p>
      <button class="btn btn-primary" onclick={load}>Retry</button>
    </div>

  <!-- Empty state ─────────────────────────────────────────── -->
  {:else if $runningPipelinesState === 'ready' && $runningPipelines.length === 0}
    <div class="empty-state">
      <div class="empty-icon">✓</div>
      <h3>No running pipelines</h3>
      <p>All configured repositories are idle right now.</p>
    </div>

  <!-- Filtered-empty state ───────────────────────────────── -->
  {:else if $runningPipelinesState === 'ready' && displayedPipelines.length === 0}
    <div class="empty-state">
      <div class="empty-icon">⌀</div>
      <h3>No matching pipelines</h3>
      <p>No running pipelines match the selected repository filter.</p>
      <button class="btn btn-secondary" onclick={() => (repoFilter = 'all')}>Show all repositories</button>
    </div>

  <!-- Data grid ───────────────────────────────────────────── -->
  {:else}
    <div class="panel data-grid">
      <div class="data-grid-header">
        <div class="data-grid-cell col-project">PROJECT</div>
        <div class="data-grid-cell col-build">#</div>
        <div class="data-grid-cell col-branch">TARGET</div>
        <div class="data-grid-cell col-trig">VIA</div>
        <div class="data-grid-cell col-status-hdr">STATUS</div>
        <div class="data-grid-cell col-dur">ELAPSED</div>
        <div class="data-grid-cell col-creator">CREATOR</div>
        <div class="data-grid-cell col-date">CREATED</div>
      </div>
      <div class="data-grid-body" role="grid" aria-label="Running pipelines">
        {#each displayedPipelines as item (item.pipeline.uuid)}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
          <div
            class="data-grid-row"
            role="row"
            tabindex="0"
            onclick={(e) => openDetail(e, item)}
            onkeydown={(e) => { if (e.key === 'Enter') openDetail(null, item); }}
            onmouseenter={(e) => onRowMouseEnter(e, item)}
            onmousemove={onRowMouseMove}
            onmouseleave={onRowMouseLeave}
          >
            <div class="data-grid-cell col-project">
              <span class="project-path truncate" title="{item.workspace}/{item.repo_slug}">
                <span class="pc-workspace">{item.workspace}</span><span class="pc-sep">/</span><span class="pc-slug">{item.repo_slug}</span>
              </span>
            </div>
            <div class="data-grid-cell col-build">
              <span class="mono build-num">#{item.pipeline.build_number || '—'}</span>
            </div>
            <div class="data-grid-cell col-branch">
              <span class="branch-ref" title={item.pipeline.target?.ref_name}>{item.pipeline.target?.ref_name || '—'}</span>
              {#if item.pipeline.target?.commit?.hash}
                <span class="commit-tag" title={item.pipeline.target.commit.hash}>{shortHash(item.pipeline.target.commit.hash)}</span>
              {/if}
            </div>
            <div class="data-grid-cell col-trig">
              <span class="trigger-tag">{triggerLabel(item.pipeline.trigger)}</span>
            </div>
            <div class="data-grid-cell col-status-hdr">
              <span
                class="badge {statusToBadgeClass(item.pipeline.state)}"
                class:badge-animated={item.pipeline.state?.name === 'IN_PROGRESS'}
              >{statusLabel(item.pipeline.state)}</span>
            </div>
            <div class="data-grid-cell col-dur mono text-tertiary">
              {elapsedLabel(item.pipeline)}
            </div>
            <div class="data-grid-cell col-creator truncate" title={item.pipeline.creator?.display_name || item.pipeline.creator?.username}>
              {item.pipeline.creator?.display_name || item.pipeline.creator?.username || '—'}
            </div>
            <div class="data-grid-cell col-date text-tertiary">
              {formatDate(item.pipeline.created_on)}
            </div>
          </div>
        {/each}
      </div>
    </div>
  {/if}

  <!-- ── Hover popover for log variables ─────────────────── -->
  {#if hoverActive && hoveredPipeline}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="vars-popover"
      style="left: {hoverPos.x + 12}px; top: {hoverPos.y - 8}px"
      onmouseenter={() => { if (hoverTimer) clearTimeout(hoverTimer); }}
      onmouseleave={onRowMouseLeave}
    >
      <div class="vars-popover-header">
        <span class="vars-popover-title">#{hoveredPipeline.build_number || '—'} variables</span>
        {#if hoverVarsLoading}
          <span class="spinner vars-spinner"></span>
        {:else if hoverVarsError}
          <span class="vars-popover-error">{hoverVarsError}</span>
        {/if}
      </div>
      {#if !hoverVarsLoading && !hoverVarsError && hoverVars.length > 0}
        <div class="vars-popover-grid">
          {#each hoverVars as v}
            <div class="vp-key">{v.key}</div>
            <div class="vp-val" class:vp-val-secret={v.secured} title={v.secured ? '' : v.value}>{v.secured ? '••••••••' : truncateVal(v.value)}</div>
          {/each}
        </div>
      {:else if !hoverVarsLoading && !hoverVarsError && hoverVars.length === 0}
        <div class="vars-popover-empty">No variables for this pipeline</div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .running-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
  }

  /* ── Toolbar ─────────────────────────────────────────────── */
  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-5);
    padding: var(--space-5) var(--space-6);
    flex-shrink: 0;
  }
  .toolbar-left {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    min-width: 0;
  }
  .toolbar-title {
    font-size: var(--font-size-md);
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .toolbar-count {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--font-size-sm);
    color: var(--text-tertiary);
    white-space: nowrap;
  }
  .count-label {
    color: var(--text-tertiary);
  }
  .count-filtered {
    color: var(--text-tertiary);
  }
  .repo-select {
    width: auto;
    min-width: 180px;
    max-width: 260px;
  }
  .interval-select {
    width: auto;
    min-width: 92px;
  }
  .toolbar-right {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    flex-shrink: 0;
  }
  .live-indicator {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--font-size-xs);
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    user-select: none;
  }
  .live-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--success);
    box-shadow: 0 0 6px var(--success);
    animation: pulse-dot 2s ease-in-out infinite;
  }
  .live-indicator.paused {
    color: var(--warning);
  }
  .live-indicator.paused .live-dot {
    background: var(--warning);
    box-shadow: 0 0 6px var(--warning);
    animation: none;
  }
  @keyframes pulse-dot {
    0%, 100% { opacity: 1; }
    50%      { opacity: 0.4; }
  }

  /* ── Errors banner ───────────────────────────────────────── */
  .errors-banner {
    margin: 0 var(--space-6) var(--space-4);
    padding: var(--space-3) var(--space-5);
    background: var(--warning-bg);
    border: 1px solid var(--warning);
    border-radius: var(--radius-md);
    font-size: var(--font-size-xs);
    color: var(--warning);
  }
  .errors-banner-title {
    font-weight: 600;
  }
  .errors-banner ul {
    margin: var(--space-2) 0 0;
    padding-left: var(--space-6);
  }
  .errors-banner li {
    font-family: var(--font-mono);
    color: var(--text-secondary);
  }

  /* ── Data grid ───────────────────────────────────────────── */
  .data-grid {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    margin: 0 var(--space-6) var(--space-6);
    border-radius: var(--radius-lg);
    border: 1px solid var(--border-default);
    background: var(--bg-panel);
  }
  .data-grid-header {
    display: flex;
    align-items: center;
    background: var(--bg-panel);
    border-bottom: 1px solid var(--border-default);
    flex-shrink: 0;
  }
  .data-grid-header .data-grid-cell {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-tertiary);
    padding: var(--space-3) var(--space-5);
    user-select: none;
  }
  .data-grid-body {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    overscroll-behavior: contain;
  }
  .data-grid-row {
    display: flex;
    align-items: center;
    min-height: 32px;
    transition: background var(--transition-fast);
    border-bottom: 1px solid var(--border-subtle);
    cursor: pointer;
  }
  .data-grid-row:hover {
    background: var(--bg-hover);
  }
  .data-grid-row:focus-visible {
    outline: none;
    box-shadow: inset 0 0 0 1px var(--border-focus);
  }
  .data-grid-cell {
    padding: var(--space-2) var(--space-5);
    font-size: var(--font-size-sm);
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 0;
    line-height: 1.3;
  }

  .col-project    { flex: 1 1 12rem;   min-width: 8rem;  max-width: 22rem; }
  .col-build      { flex: 0 0 3.2rem; }
  .col-branch     { flex: 1 1 11rem;   min-width: 7.5rem; max-width: 20rem; }
  .col-trig       { flex: 0 0 5rem;    justify-content: center; }
  .col-status-hdr { flex: 0 0 7rem; }
  .col-dur        { flex: 0 0 5.5rem;  justify-content: flex-end; }
  .col-creator    { flex: 1 1 7rem;    min-width: 5.5rem; }
  .col-date       { flex: 0 0 9rem;    white-space: nowrap; }

  .project-path {
    font-family: var(--font-mono);
    font-size: var(--font-size-xs);
    color: var(--text-secondary);
    display: inline-flex;
    align-items: center;
    gap: 2px;
  }
  .pc-workspace {
    color: var(--text-primary);
    font-weight: 600;
  }
  .pc-sep {
    color: var(--text-tertiary);
  }
  .pc-slug {
    color: var(--text-secondary);
  }

  .build-num {
    font-weight: 600;
    color: var(--text-secondary);
  }

  .branch-ref {
    font-family: var(--font-mono);
    font-size: var(--font-size-xs);
    color: var(--accent-text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .commit-tag {
    font-family: var(--font-mono);
    font-size: 10px;
    color: var(--text-tertiary);
    background: var(--bg-badge);
    padding: 1px 4px;
    border-radius: var(--radius-sm);
    flex-shrink: 0;
  }
  .trigger-tag {
    font-size: var(--font-size-xs);
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  /* Animated badge for in-progress pipelines */
  @keyframes pulse-border {
    0%, 100% { box-shadow: 0 0 0 0 var(--accent-muted); }
    50%      { box-shadow: 0 0 0 2px var(--accent-muted); }
  }
  .badge-animated {
    animation: pulse-border 2s ease-in-out infinite;
  }

  /* ── Skeleton sizing ─────────────────────────────────────── */
  .skeleton { display: block; }
  .sk-project { width: 110px; height: 12px; }
  .sk-num     { width: 26px;  height: 12px; }
  .sk-branch  { width: 90px;  height: 12px; }
  .sk-trig    { width: 40px;  height: 12px; }
  .sk-badge   { width: 50px;  height: 12px; }
  .sk-dur     { width: 36px;  height: 12px; margin-left: auto; }
  .sk-name    { width: 60px;  height: 12px; }
  .sk-date    { width: 80px;  height: 12px; }

  /* ── Hover popover for log variables ───────────────────────── */
  .vars-popover {
    position: fixed;
    z-index: 1000;
    min-width: 220px;
    max-width: 340px;
    max-height: 280px;
    overflow-y: auto;
    background: var(--bg-panel);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-dropdown);
    padding: var(--space-3);
    pointer-events: auto;
    animation: popIn 0.12s ease-out;
  }
  @keyframes popIn {
    from { opacity: 0; transform: translateY(3px); }
    to   { opacity: 1; transform: translateY(0); }
  }

  .vars-popover-header {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    margin-bottom: var(--space-2);
    padding-bottom: var(--space-2);
    border-bottom: 1px solid var(--border-subtle);
  }
  .vars-popover-title {
    font-size: var(--font-size-xs);
    font-weight: 600;
    color: var(--text-secondary);
    font-family: var(--font-mono);
  }
  .vars-spinner {
    width: 12px;
    height: 12px;
    border-width: 2px;
  }
  .vars-popover-error {
    font-size: 10.5px;
    color: var(--danger-text);
  }

  .vars-popover-grid {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 2px var(--space-3);
    align-items: baseline;
  }
  .vp-key {
    font-size: var(--font-size-xs);
    font-family: var(--font-mono);
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    padding: 1px 0;
    min-width: 0;
  }
  .vp-val {
    font-size: var(--font-size-xs);
    font-family: var(--font-mono);
    color: var(--text-secondary);
    word-break: break-all;
    padding: 1px 0;
    min-width: 0;
  }
  .vp-val-secret {
    color: var(--text-tertiary);
    font-style: italic;
    letter-spacing: 0.15em;
  }

  .vars-popover-empty {
    font-size: var(--font-size-xs);
    color: var(--text-tertiary);
    text-align: center;
    padding: var(--space-2) 0;
  }
</style>