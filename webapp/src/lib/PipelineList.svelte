<script>
  import { get } from 'svelte/store';
  import { onDestroy } from 'svelte';
  import { activeProject, pipelines, pipelinesNext, listState, listError, listSort, selectedPipeline, selectedSteps, selectedVariables, selectedLogVariables, detailState, refreshTrigger, logStepName, logStepUUID } from '../stores/appState.js';
  import { page, navigateTo, isNewTabClick, openInNewTab } from '../stores/router.js';
  import { listPipelines, getLogVariables, getPipeline } from '../stores/api.js';
  import { formatDate, formatDuration, statusLabel } from './utils.js';

  // ── Fetch token: guards against stale async completions overwriting
  //     data after a project switch (race condition).
  let fetchToken = 0;

  let root = $state(null);
  let loading = $state(false);
  let loadingMore = $state(false);
  let localFilter = $state('');
  let currentPage = $state(1);
  let hasMore = $state(false);
  let showScrollTop = $state(false);
  let sentinelVisible = $state(false);

  // Sentinel element ref for IntersectionObserver
  let sentinel = $state(null);

  // ── Computed: locally filtered pipelines ───────────────────────────
  let displayedPipelines = $derived.by(() => {
    const list = $pipelines;
    const filter = localFilter.trim().toLowerCase();
    if (!filter) return list;
    return list.filter(p => {
      // Search across multiple fields
      if (String(p.build_number || '').includes(filter)) return true;
      if ((p.target?.ref_name || '').toLowerCase().includes(filter)) return true;
      if ((p.target?.commit?.hash || '').toLowerCase().includes(filter)) return true;
      if (statusLabel(p.state).toLowerCase().includes(filter)) return true;
      if ((p.creator?.display_name || p.creator?.username || '').toLowerCase().includes(filter)) return true;
      if ((p.trigger?.name || '').toLowerCase().includes(filter)) return true;
      if ((p.target?.selector?.pattern || '').toLowerCase().includes(filter)) return true;
      if (formatDate(p.created_on).toLowerCase().includes(filter)) return true;
      return false;
    });
  });

  let isFiltering = $derived(localFilter.trim().length > 0);

  /** Load the first page of pipelines, fully replacing the list. */
  async function loadPipelinesFirst() {
    if (!$activeProject) return;
    const token = ++fetchToken;
    loading = true;
    listState.set('loading');
    listError.set('');
    currentPage = 1;
    hasMore = false;
    try {
      // No filter sent to server — filtering happens client-side
      const data = await listPipelines($activeProject.id, {
        sort: get(listSort),
        page: 1,
      });
      if (fetchToken !== token) return; // Stale fetch — project changed
      pipelines.set(data.values || []);
      pipelinesNext.set(data.next || '');
      hasMore = !!(data.next);
      listState.set('ready');
    } catch (e) {
      if (fetchToken !== token) return;
      listError.set(e.message);
      listState.set('error');
    } finally {
      if (fetchToken === token) {
        loading = false;
        loadingMore = false;
      }
    }
  }

  /** Load the next page and append results (infinite scroll). */
  async function loadPipelinesNext() {
    if (!$activeProject || !hasMore || loadingMore) return;
    const token = ++fetchToken;
    loadingMore = true;
    const nextPage = currentPage + 1;
    try {
      const data = await listPipelines($activeProject.id, {
        sort: get(listSort),
        page: nextPage,
      });
      if (fetchToken !== token) return; // Stale fetch — project changed
      pipelines.update(existing => [...existing, ...(data.values || [])]);
      pipelinesNext.set(data.next || '');
      hasMore = !!(data.next);
      currentPage = nextPage;
    } catch (_) {
      // Silently fail on infinite-scroll loads — data stays intact
    } finally {
      if (fetchToken === token) loadingMore = false;
    }
  }

  function clearFilter() {
    localFilter = '';
  }

  function viewDetail(e, pipeline) {
    // Ctrl/Cmd/Shift + click (or middle-click) opens in a new tab without
    // mutating the current tab's state.
    if (isNewTabClick(e)) {
      e.preventDefault();
      openInNewTab('detail', pipeline.uuid);
      return;
    }
    // Navigate FIRST so $page changes before we set stores.
    // Otherwise PipelineDetail's cleanup $effect can see $page==='list'
    // and clear the pipeline we just set.
    navigateTo('detail', pipeline.uuid);
    selectedPipeline.set(pipeline);
    selectedSteps.set([]);
    selectedVariables.set([]);
    selectedLogVariables.set([]);
    detailState.set('loading');
  }

  function statusToBadgeClass(state) {
    const name = state?.name || '';
    if (name === 'COMPLETED' && state?.result?.name === 'SUCCESSFUL') return 'badge-success';
    if (name === 'COMPLETED' && state?.result?.name === 'FAILED') return 'badge-error';
    if (name === 'FAILED' || name === 'ERROR') return 'badge-error';
    if (name === 'IN_PROGRESS' || name === 'PENDING') return 'badge-info';
    if (name === 'IN_PROGRESS_STOPPING' || name === 'STOPPED') return 'badge-warning';
    if (name === 'SUSPENDED' || name === 'DISABLED') return 'badge-warning';
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

  function truncateVal(val) {
    if (!val) return '—';
    return val.length > 60 ? val.slice(0, 60) + '…' : val;
  }

  // ── Direct log access for running pipelines ─────────────────────────
  let logLoadingFor = $state(null); // pipeline uuid currently loading

  async function goToLog(e, pipeline) {
    if (!$activeProject || !pipeline?.uuid) return;
    // New-tab click: the running step's index isn't known synchronously, so
    // open the pipeline detail in a new tab as a best-effort fallback.
    if (isNewTabClick(e)) {
      e.preventDefault();
      openInNewTab('detail', pipeline.uuid);
      return;
    }
    const pUuid = pipeline.uuid;
    logLoadingFor = pUuid;
    try {
      const data = await getPipeline($activeProject.id, pUuid);
      const steps = data.steps || [];
      // Find the first step that is IN_PROGRESS or get the last non-pending step
      const runningStep = steps.find(s => s.state?.name === 'IN_PROGRESS');
      const targetStep = runningStep || steps.find(s => s.state?.name !== 'NOT_STARTED') || steps[0];
      if (!targetStep) return;
      const stepIdx = steps.indexOf(targetStep);
      selectedPipeline.set(data.pipeline || data);
      selectedSteps.set(steps);
      logStepName.set(targetStep.name || `Step ${stepIdx + 1}`);
      logStepUUID.set(targetStep.uuid);
      navigateTo('logs', pUuid, stepIdx);
    } catch (e) {
      // Silently fail — user can still click through to detail
    } finally {
      logLoadingFor = null;
    }
  }

  let lastLoaded = '';
  let selectedIndex = $state(null);
  let gridBody = $state(null);
  let autoRefreshInterval = null;

  function startAutoRefresh() {
    stopAutoRefresh();
    const projectId = $activeProject?.id;
    if (!projectId) return;
    autoRefreshInterval = setInterval(async () => {
      // Only refresh if still on the list page and project has not changed
      if ($page !== 'list' || !$activeProject || $activeProject.id !== projectId) {
        stopAutoRefresh();
        return;
      }
      const token = ++fetchToken;
      try {
        // Fetch just the first page to check for state changes and new pipelines
        const data = await listPipelines($activeProject.id, {
          sort: get(listSort),
          page: 1,
        });
        if (fetchToken !== token) return; // Stale fetch — project changed
        const fresh = data.values || [];
        // Merge: update existing pipelines that match by uuid, prepend new ones
        const existing = $pipelines;
        const existingUuids = new Set(existing.map(p => p.uuid));
        const merged = [];
        // Updated/reordered from fresh page 1
        for (const fp of fresh) {
          if (existingUuids.has(fp.uuid)) {
            merged.push(fp); // update in-place
            existingUuids.delete(fp.uuid);
          } else {
            merged.push(fp); // new pipeline
          }
        }
        // Keep any remaining older pipelines that weren't in fresh page 1
        for (const ep of existing) {
          if (existingUuids.has(ep.uuid)) merged.push(ep);
        }
        pipelines.set(merged);
        pipelinesNext.set(data.next || '');
      } catch (_) {
        // Silently ignore errors during auto-refresh
      }
    }, 15000); // every 15 seconds
  }

  function stopAutoRefresh() {
    if (autoRefreshInterval) {
      clearInterval(autoRefreshInterval);
      autoRefreshInterval = null;
    }
  }

  // Start/stop auto-refresh based on page and project only.
  // The interval runs unconditionally; the callback quickly returns if no
  // running pipelines exist. This avoids a reactive dependency loop where
  // auto-refresh modifies $pipelines, the effect sees the change, and
  // re-runs — causing effect_update_depth_exceeded.
  $effect(() => {
    stopAutoRefresh();
    if ($page === 'list' && $activeProject) {
      startAutoRefresh();
    }
  });

  onDestroy(() => stopAutoRefresh());

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

  async function fetchHoverVars(pipe) {
    if (!$activeProject || !pipe?.uuid) return;
    const cached = hoverVarsCache.get(pipe.uuid);
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
      const data = await getLogVariables($activeProject.id, pipe.uuid);
      const vars = data.variables || [];
      hoverVars = vars;
      hoverVarsCache.set(pipe.uuid, vars);
    } catch (e) {
      hoverVarsError = e.message;
    } finally {
      hoverVarsLoading = false;
    }
  }

  function onRowMouseEnter(e, pipe) {
    if (hoverTimer) clearTimeout(hoverTimer);
    hoverPos = { x: e.clientX, y: e.clientY };
    hoverTimer = setTimeout(() => {
      hoveredPipeline = pipe;
      hoverActive = true;
      fetchHoverVars(pipe);
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

  // ── Infinite-scroll IntersectionObserver ──────────────────────────────
  $effect(() => {
    const el = sentinel;
    if (!el) return;
    const observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        sentinelVisible = entry.isIntersecting;
      }
      // Only trigger load if not filtering (filtering is client-side only)
      if (sentinelVisible && hasMore && !loadingMore && !isFiltering) {
        loadPipelinesNext();
      }
    }, { rootMargin: '200px' });
    observer.observe(el);
    return () => observer.disconnect();
  });

  // ── Reactive data loading ────────────────────────────────────────────
  $effect(() => {
    const key = `${$activeProject?.id || ''}:${$listSort}:${$refreshTrigger}`;
    if (!key || key === '::' || key === ':') return;
    if ($page !== 'list') return;
    if (lastLoaded === key) return;
    lastLoaded = key;
    selectedIndex = null;
    loadPipelinesFirst();
  });

  // ── Refresh event ────────────────────────────────────────────────────
  $effect(() => {
    function onRefresh() {
      if ($page === 'list') { lastLoaded = ''; loadPipelinesFirst(); }
    }
    window.addEventListener('app:refresh', onRefresh);
    return () => window.removeEventListener('app:refresh', onRefresh);
  });

  // ── Track scroll position for "back to top" button ───────────────────
  function onGridScroll(e) {
    showScrollTop = e.target.scrollTop > 400;
  }

  function scrollToTop() {
    gridBody?.scrollTo({ top: 0, behavior: 'smooth' });
  }

  // ── Keyboard navigation ─────────────────────────────────────────────
  function handleRowKeydown(e, index) {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      const next = Math.min(index + 1, displayedPipelines.length - 1);
      selectedIndex = next;
      document.querySelector(`[data-row-index="${next}"]`)?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      const prev = Math.max(index - 1, 0);
      selectedIndex = prev;
      document.querySelector(`[data-row-index="${prev}"]`)?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    } else if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      const pipe = displayedPipelines[index];
      if (pipe) viewDetail(null, pipe);
    }
  }
</script>

<div class="pipeline-list" bind:this={root}>
  <!-- ── Toolbar ────────────────────────────────────────── -->
  <div class="toolbar">
    <div class="toolbar-left">
      <h2 class="toolbar-title" title="{$activeProject?.workspace || ''}/{$activeProject?.repo_slug || ''}">
        {$activeProject?.name || 'Unknown'}
      </h2>
      <span class="toolbar-count">
        <span class="mono">{isFiltering ? displayedPipelines.length : $pipelines.length}</span>
        {#if hasMore && !isFiltering}<span class="count-suffix">+</span>{/if}
        <span class="count-label"> pipeline{$pipelines.length !== 1 ? 's' : ''}</span>
        {#if isFiltering}
          <span class="count-filtered"> of {$pipelines.length}</span>
        {/if}
      </span>
    </div>
    <div class="toolbar-right">
      <div class="search-group">
        <svg class="search-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="8"></circle>
          <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        </svg>
        <input
          type="text"
          class="form-input search-input"
          placeholder="Filter locally — branch, status, #, trigger…"
          bind:value={localFilter}
        />
        {#if localFilter}
          <button class="search-clear" onclick={clearFilter} title="Clear filter (Esc)">×</button>
        {/if}
      </div>
      <select class="form-select toolbar-select" bind:value={$listSort}>
        <option value="-created_on">Newest first</option>
        <option value="+created_on">Oldest first</option>
      </select>
    </div>
  </div>

  <!-- ── Loading skeleton (no data yet) ─────────────────── -->
  {#if $listState === 'loading' && $pipelines.length === 0}
    <div class="panel skeleton-panel">
      <div class="data-grid-header">
        <div class="data-grid-cell col-build" aria-hidden="true">#</div>
        <div class="data-grid-cell col-branch" aria-hidden="true">TARGET</div>
        <div class="data-grid-cell col-sel" aria-hidden="true">PIPELINE</div>
        <div class="data-grid-cell col-trig" aria-hidden="true">VIA</div>
        <div class="data-grid-cell col-status-hdr" aria-hidden="true">STATUS</div>
        <div class="data-grid-cell col-dur" aria-hidden="true">DUR</div>
        <div class="data-grid-cell col-creator" aria-hidden="true">CREATOR</div>
        <div class="data-grid-cell col-date" aria-hidden="true">CREATED</div>
        <div class="data-grid-cell col-log-hdr" aria-hidden="true"></div>
      </div>
      {#each Array(10) as _}
        <div class="data-grid-row skeleton-row">
          <div class="data-grid-cell col-build"><div class="skeleton sk-num"></div></div>
          <div class="data-grid-cell col-branch"><div class="skeleton sk-branch"></div></div>
          <div class="data-grid-cell col-sel"><div class="skeleton sk-sel"></div></div>
          <div class="data-grid-cell col-trig"><div class="skeleton sk-trig"></div></div>
          <div class="data-grid-cell col-status-hdr"><div class="skeleton sk-badge"></div></div>
          <div class="data-grid-cell col-dur"><div class="skeleton sk-dur"></div></div>
          <div class="data-grid-cell col-creator"><div class="skeleton sk-name"></div></div>
          <div class="data-grid-cell col-date"><div class="skeleton sk-date"></div></div>
          <div class="data-grid-cell col-log"></div>
        </div>
      {/each}
    </div>

  <!-- ── Error state ────────────────────────────────────── -->
  {:else if $listState === 'error'}
    <div class="empty-state">
      <div class="empty-icon">!</div>
      <h3>Failed to load pipelines</h3>
      <p>{$listError || 'An unknown error occurred'}</p>
      <button class="btn btn-primary" onclick={() => { lastLoaded = ''; loadPipelinesFirst(); }}>
        Retry
      </button>
    </div>

  <!-- ── Empty state (no pipelines from server) ─────────── -->
  {:else if $listState === 'ready' && $pipelines.length === 0}
    <div class="empty-state">
      <div class="empty-icon">∅</div>
      <h3>No pipelines</h3>
      <p>No pipeline runs have been found for this repository.</p>
    </div>

  <!-- ── Empty filter state ─────────────────────────────── -->
  {:else if $listState === 'ready' && isFiltering && displayedPipelines.length === 0}
    <div class="empty-state">
      <div class="empty-icon">∅</div>
      <h3>No matching pipelines</h3>
      <p>No pipelines in the list match <code>{localFilter}</code></p>
      <button class="btn btn-secondary" onclick={clearFilter}>Clear filter</button>
    </div>

  <!-- ── Data grid with infinite scroll ──────────────────── -->
  {:else}
    <div class="panel data-grid">
      <div class="data-grid-header">
        <div class="data-grid-cell col-build">#</div>
        <div class="data-grid-cell col-branch">TARGET</div>
        <div class="data-grid-cell col-sel">PIPELINE</div>
        <div class="data-grid-cell col-trig">VIA</div>
        <div class="data-grid-cell col-status-hdr">STATUS</div>
        <div class="data-grid-cell col-dur">DUR</div>
        <div class="data-grid-cell col-creator">CREATOR</div>
        <div class="data-grid-cell col-date">CREATED</div>
        <div class="data-grid-cell col-log-hdr"></div>
      </div>
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="data-grid-body"
        bind:this={gridBody}
        onscroll={onGridScroll}
        role="grid"
        aria-label="Pipelines list"
      >
        {#each displayedPipelines as pipe, index}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div
            class="data-grid-row"
            class:row-selected={selectedIndex === index}
            data-row-index={index}
            tabindex="0"
            role="row"
            aria-label="Pipeline #{pipe.build_number} - {statusLabel(pipe.state)}"
            aria-selected={selectedIndex === index}
            onclick={(e) => viewDetail(e, pipe)}
            onkeydown={(e) => handleRowKeydown(e, index)}
            onmouseenter={(e) => onRowMouseEnter(e, pipe)}
            onmousemove={onRowMouseMove}
            onmouseleave={onRowMouseLeave}
          >
            <div class="data-grid-cell col-build">
              <span class="mono build-num">#{pipe.build_number || '—'}</span>
            </div>
            <div class="data-grid-cell col-branch">
              <span class="branch-ref" title={pipe.target?.ref_name}>{pipe.target?.ref_name || '—'}</span>
              {#if pipe.target?.commit?.hash}
                <span class="commit-tag" title={pipe.target.commit.hash}>{shortHash(pipe.target.commit.hash)}</span>
              {/if}
            </div>
            <div class="data-grid-cell col-sel">
              {#if pipe.target?.selector?.pattern}
                <span class="selector-tag" title={pipe.target.selector.pattern}>{pipe.target.selector.pattern}</span>
              {:else}
                <span class="text-tertiary">default</span>
              {/if}
            </div>
            <div class="data-grid-cell col-trig">
              <span class="trigger-tag" class:trigger-push={pipe.trigger?.name?.toLowerCase() === 'push'}
                class:trigger-manual={pipe.trigger?.name?.toLowerCase() === 'manual'}
                class:trigger-schedule={pipe.trigger?.name?.toLowerCase() === 'schedule' || pipe.trigger?.name?.toLowerCase() === 'scheduled'}
              >{triggerLabel(pipe.trigger)}</span>
            </div>
            <div class="data-grid-cell col-status-hdr">
              <span class="badge {statusToBadgeClass(pipe.state)}" class:badge-animated={pipe.state?.name === 'IN_PROGRESS'}>{statusLabel(pipe.state)}</span>
            </div>
            <div class="data-grid-cell col-dur mono text-tertiary">
              {formatDuration(pipe.created_on, pipe.completed_on, pipe.build_seconds_used || 0)}
            </div>
            <div class="data-grid-cell col-creator truncate" title={pipe.creator?.display_name || pipe.creator?.username}>
              {pipe.creator?.display_name || pipe.creator?.username || '—'}
            </div>
            <div class="data-grid-cell col-date text-tertiary">
              {formatDate(pipe.created_on)}
            </div>
            <div class="data-grid-cell col-log">
              {#if pipe.state?.name === 'IN_PROGRESS' || pipe.state?.name === 'PENDING'}
                <button
                  class="btn btn-secondary btn-xs"
                  onclick={(e) => { e.stopPropagation(); goToLog(e, pipe); }}
                  disabled={logLoadingFor === pipe.uuid}
                  title="Open running step log"
                >
                  {#if logLoadingFor === pipe.uuid}
                    <span class="spinner spinner-xs"></span>
                  {:else}
                    Log
                  {/if}
                </button>
              {/if}
            </div>
          </div>
        {/each}

        <!-- Infinite-scroll sentinel (hidden when filtering) -->
        {#if !isFiltering}
          <div class="scroll-sentinel" bind:this={sentinel}>
            {#if loadingMore}
              <div class="load-more-hint">
                <span class="spinner"></span>
                <span class="text-tertiary">Loading more…</span>
              </div>
            {:else if hasMore}
              <div class="load-more-hint text-tertiary">Scroll for more</div>
            {:else if $pipelines.length >= 50}
              <div class="load-more-hint"><span class="dot"></span> All pipelines loaded</div>
            {/if}
          </div>
        {:else}
          <div class="scroll-sentinel filter-hint">
            <span class="text-tertiary">Filter active — showing {displayedPipelines.length} of {$pipelines.length} pipelines</span>
          </div>
        {/if}
      </div>
    </div>

    <!-- Floating scroll-to-top button -->
    {#if showScrollTop}
      <button class="scroll-top-btn btn btn-secondary btn-sm" onclick={scrollToTop} title="Scroll to top">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="18 15 12 9 6 15"></polyline>
        </svg>
      </button>
    {/if}
  {/if}

  <!-- ── Global loading bar ──────────────────────────────── -->
  {#if loading && $pipelines.length > 0}
    <div class="loading-indicator"></div>
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
  .pipeline-list {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: 0;
    height: 100%;
    padding: var(--space-5) var(--space-6);
    overflow: hidden;
  }

  /* ── Toolbar ──────────────────────────────────────────────── */
  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-5);
    flex-shrink: 0;
    user-select: none;
    padding-bottom: var(--space-4);
  }

  .toolbar-left {
    display: flex;
    align-items: baseline;
    gap: var(--space-4);
    min-width: 0;
  }

  .toolbar-title {
    font-size: var(--font-size-md);
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 280px;
  }

  .toolbar-count {
    display: flex;
    align-items: baseline;
    gap: 2px;
    font-size: var(--font-size-xs);
    color: var(--text-tertiary);
    white-space: nowrap;
  }
  .toolbar-count .mono {
    font-size: var(--font-size-xs);
    color: var(--text-secondary);
  }
  .count-suffix {
    color: var(--accent-text);
    font-weight: 600;
  }
  .count-label {
    color: var(--text-tertiary);
  }
  .count-filtered {
    color: var(--accent-text);
    font-weight: 500;
  }

  .toolbar-right {
    display: flex;
    gap: var(--space-3);
    align-items: center;
    flex-shrink: 0;
  }

  .search-group {
    position: relative;
    display: flex;
    align-items: center;
  }
  .search-icon {
    position: absolute;
    left: 8px;
    color: var(--text-tertiary);
    pointer-events: none;
    z-index: 1;
  }
  .search-input {
    padding-left: 28px !important;
    width: 230px;
  }
  .search-clear {
    position: absolute;
    right: 4px;
    background: none;
    border: none;
    color: var(--text-tertiary);
    font-size: 15px;
    cursor: pointer;
    padding: 2px 5px;
    line-height: 1;
    border-radius: var(--radius-sm);
    font-family: var(--font-ui);
  }
  .search-clear:hover { color: var(--text-primary); background: var(--bg-hover); }

  .toolbar-select {
    width: auto;
    min-width: 130px;
  }

  /* ── Data Grid ──────────────────────────────────────────────── */
  .data-grid {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
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
    cursor: default;
  }
  .data-grid-row:last-of-type:not(.skeleton-row) {
    border-bottom: 1px solid var(--border-subtle);
  }

  .data-grid-row:hover {
    background: var(--bg-hover);
  }
  .data-grid-row:focus-visible {
    outline: none;
    box-shadow: inset 0 0 0 1px var(--border-focus);
  }
  .row-selected {
    background: var(--bg-selection) !important;
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

  /* Column widths — use rem so header (10px) and data (12px) cells align */
  .col-build      { flex: 0 0 3.2rem; }
  .col-branch     { flex: 1 1 11rem;   min-width: 7.5rem;  max-width: 20rem; }
  .col-sel        { flex: 1 1 8.5rem;  min-width: 5.5rem;  max-width: 13rem; }
  .col-trig       { flex: 0 0 5rem;    justify-content: center; }
  .col-status-hdr { flex: 0 0 6.5rem; }
  .col-dur        { flex: 0 0 5.5rem;  justify-content: flex-end; }
  .col-creator    { flex: 1 1 7rem;    min-width: 5.5rem; }
  .col-date       { flex: 0 0 9rem;    white-space: nowrap; }
  .col-log-hdr    { flex: 0 0 3.2rem; }
  .col-log        { flex: 0 0 3.2rem;  justify-content: center; padding: var(--space-2) var(--space-1); }

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
    background: var(--bg-input);
    padding: 1px 5px;
    border-radius: var(--radius-sm);
    flex-shrink: 0;
    letter-spacing: 0.02em;
  }

  .selector-tag {
    font-family: var(--font-mono);
    font-size: 10.5px;
    color: var(--text-secondary);
    background: var(--bg-input);
    padding: 1px 6px;
    border-radius: var(--radius-sm);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 92px;
  }

  .trigger-tag {
    font-size: 10.5px;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.03em;
    padding: 1px 5px;
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    background: var(--bg-input);
  }
  .trigger-push {
    color: var(--accent-text);
    background: var(--accent-muted);
  }
  .trigger-manual {
    color: var(--warning);
    background: var(--warning-bg);
  }
  .trigger-schedule {
    color: var(--success);
    background: var(--success-bg);
  }

  /* Animated badge for running pipelines */
  @keyframes pulse-border {
    0%, 100% { box-shadow: 0 0 0 0 var(--accent-muted); }
    50%      { box-shadow: 0 0 0 2px var(--accent-muted); }
  }
  .badge-animated {
    animation: pulse-border 2s ease-in-out infinite;
  }

  /* ── Scroll sentinel ────────────────────────────────────────── */
  .scroll-sentinel {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4);
    min-height: 38px;
  }

  .load-more-hint {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    font-size: var(--font-size-xs);
    user-select: none;
  }
  .dot {
    width: 4px;
    height: 4px;
    border-radius: 50%;
    background: var(--text-tertiary);
  }

  .filter-hint {
    background: var(--bg-hover);
    border-top: 1px dashed var(--border-default);
  }

  /* ── Scroll-to-top ──────────────────────────────────────────── */
  .scroll-top-btn {
    position: absolute;
    bottom: var(--space-5);
    right: var(--space-8);
    z-index: 5;
    width: 30px;
    height: 30px;
    padding: 0;
    border-radius: 50%;
    box-shadow: var(--shadow-dropdown);
    opacity: 0.85;
  }
  .scroll-top-btn:hover { opacity: 1; }

  /* ── Skeleton rows ──────────────────────────────────────────── */
  .skeleton-panel {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }
  .skeleton-row {
    min-height: 32px;
    border-bottom: 1px solid var(--border-subtle);
  }
  .skeleton-row:hover { background: transparent !important; cursor: default; }

  .sk-num    { width: 28px;  height: 12px; }
  .sk-branch { width: 90px;  height: 12px; }
  .sk-sel    { width: 56px;  height: 12px; }
  .sk-trig   { width: 40px;  height: 12px; }
  .sk-badge  { width: 50px;  height: 12px; }
  .sk-dur    { width: 36px;  height: 12px; margin-left: auto; }
  .sk-name   { width: 60px;  height: 12px; }
  .sk-date   { width: 80px;  height: 12px; }

  /* ── Loading indicator bar ──────────────────────────────────── */
  .loading-indicator {
    height: 2px;
    background: var(--accent);
    animation: loadingBar 1.8s ease-in-out infinite;
    position: fixed;
    top: var(--titlebar-height);
    left: 0;
    right: 0;
    z-index: 10;
  }
  @keyframes loadingBar {
    0% { transform: translateX(-100%); }
    40% { transform: translateX(0%); }
    60% { transform: translateX(0%); }
    100% { transform: translateX(100%); }
  }

  /* ── Empty / error states ──────────────────────────────────── */
  .empty-icon {
    font-size: 28px;
    opacity: 0.35;
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 1px dashed var(--border-default);
    border-radius: var(--radius-xl);
    margin-bottom: var(--space-2);
    font-family: var(--font-mono);
    font-weight: 600;
    color: var(--text-tertiary);
  }

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
