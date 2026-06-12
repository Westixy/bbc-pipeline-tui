<script>
  import { get } from 'svelte/store';
  import { activeProject, pipelines, pipelinesNext, listState, listError, listFilter, listSort, selectedPipeline, selectedSteps, selectedVariables, detailState, refreshTrigger } from '../stores/appState.js';
  import { page, navigateTo } from '../stores/router.js';
  import { listPipelines } from '../stores/api.js';
  import { formatDate, formatDuration, statusLabel } from './utils.js';

  let root = $state(null);
  let loading = $state(false);
  let loadingMore = $state(false);
  let searchTimeout = $state(null);
  let localFilter = $state('');
  let currentPage = $state(1);
  let hasMore = $state(false);
  let showScrollTop = $state(false);
  let sentinelVisible = $state(false);

  // Sentinel element ref for IntersectionObserver
  let sentinel = $state(null);

  /** Load the first page of pipelines, fully replacing the list. */
  async function loadPipelinesFirst() {
    if (!$activeProject) return;
    loading = true;
    listState.set('loading');
    listError.set('');
    currentPage = 1;
    hasMore = false;
    try {
      const data = await listPipelines($activeProject.id, {
        filter: get(listFilter),
        sort: get(listSort),
        page: 1,
      });
      pipelines.set(data.values || []);
      pipelinesNext.set(data.next || '');
      hasMore = !!(data.next);
      listState.set('ready');
    } catch (e) {
      listError.set(e.message);
      listState.set('error');
    } finally {
      loading = false;
      loadingMore = false;
    }
  }

  /** Load the next page and append results (infinite scroll). */
  async function loadPipelinesNext() {
    if (!$activeProject || !hasMore || loadingMore) return;
    loadingMore = true;
    const nextPage = currentPage + 1;
    try {
      const data = await listPipelines($activeProject.id, {
        filter: get(listFilter),
        sort: get(listSort),
        page: nextPage,
      });
      pipelines.update(existing => [...existing, ...(data.values || [])]);
      pipelinesNext.set(data.next || '');
      hasMore = !!(data.next);
      currentPage = nextPage;
    } catch (_) {
      // Silently fail on infinite-scroll loads — data stays intact
    } finally {
      loadingMore = false;
    }
  }

  function handleFilterInput() {
    if (searchTimeout) clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      listFilter.set(localFilter);
    }, 400);
  }

  function clearFilter() {
    localFilter = '';
    listFilter.set('');
  }

  function viewDetail(pipeline) {
    selectedPipeline.set(pipeline);
    selectedSteps.set([]);
    selectedVariables.set([]);
    detailState.set('loading');
    navigateTo('detail', pipeline.uuid);
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

  let lastLoaded = '';
  let selectedIndex = $state(null);
  let gridBody = $state(null);
  let scrollTopBtn = $state(null);

  // ── Infinite-scroll IntersectionObserver ──────────────────────────────
  $effect(() => {
    const el = sentinel;
    if (!el) return;
    const observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        sentinelVisible = entry.isIntersecting;
      }
      if (sentinelVisible && hasMore && !loadingMore) {
        loadPipelinesNext();
      }
    }, { rootMargin: '200px' });
    observer.observe(el);
    return () => observer.disconnect();
  });

  // ── Reactive data loading ────────────────────────────────────────────
  $effect(() => {
    const key = `${$activeProject?.id || ''}:${$listFilter}:${$listSort}:${$refreshTrigger}`;
    if (!key || key === ':::') return;
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
      const next = Math.min(index + 1, $pipelines.length - 1);
      selectedIndex = next;
      document.querySelector(`[data-row-index="${next}"]`)?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      const prev = Math.max(index - 1, 0);
      selectedIndex = prev;
      document.querySelector(`[data-row-index="${prev}"]`)?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    } else if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      const pipe = $pipelines[index];
      if (pipe) viewDetail(pipe);
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
        <span class="mono">{$pipelines.length}</span>
        {#if hasMore}<span class="count-suffix">+</span>{/if}
        <span class="count-label"> pipeline{$pipelines.length !== 1 ? 's' : ''}</span>
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
          placeholder="Filter by branch, status, #number…"
          bind:value={localFilter}
          oninput={handleFilterInput}
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
        <div class="data-grid-cell col-build" aria-hidden="true">BUILD</div>
        <div class="data-grid-cell col-branch" aria-hidden="true">TARGET</div>
        <div class="data-grid-cell col-status-hdr" aria-hidden="true">STATUS</div>
        <div class="data-grid-cell col-dur" aria-hidden="true">DURATION</div>
        <div class="data-grid-cell col-creator" aria-hidden="true">CREATOR</div>
        <div class="data-grid-cell col-date" aria-hidden="true">CREATED</div>
      </div>
      {#each Array(10) as _}
        <div class="data-grid-row skeleton-row">
          <div class="data-grid-cell col-build"><div class="skeleton sk-num"></div></div>
          <div class="data-grid-cell col-branch"><div class="skeleton sk-branch"></div></div>
          <div class="data-grid-cell col-status-hdr"><div class="skeleton sk-badge"></div></div>
          <div class="data-grid-cell col-dur"><div class="skeleton sk-dur"></div></div>
          <div class="data-grid-cell col-creator"><div class="skeleton sk-name"></div></div>
          <div class="data-grid-cell col-date"><div class="skeleton sk-date"></div></div>
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

  <!-- ── Empty state ────────────────────────────────────── -->
  {:else if $listState === 'ready' && $pipelines.length === 0}
    {#if $listFilter}
      <div class="empty-state">
        <div class="empty-icon">∅</div>
        <h3>No matching pipelines</h3>
        <p>No pipelines match filter <code>{$listFilter}</code></p>
        <button class="btn btn-secondary" onclick={clearFilter}>Clear filter</button>
      </div>
    {:else}
      <div class="empty-state">
        <div class="empty-icon">∅</div>
        <h3>No pipelines</h3>
        <p>No pipeline runs have been found for this repository.</p>
      </div>
    {/if}

  <!-- ── Data grid with infinite scroll ──────────────────── -->
  {:else}
    <div class="panel data-grid">
      <div class="data-grid-header">
        <div class="data-grid-cell col-build">BUILD</div>
        <div class="data-grid-cell col-branch">TARGET</div>
        <div class="data-grid-cell col-status-hdr">STATUS</div>
        <div class="data-grid-cell col-dur">DURATION</div>
        <div class="data-grid-cell col-creator">CREATOR</div>
        <div class="data-grid-cell col-date">CREATED</div>
      </div>
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div
        class="data-grid-body"
        bind:this={gridBody}
        onscroll={onGridScroll}
        role="grid"
        aria-label="Pipelines list"
      >
        {#each $pipelines as pipe, index}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div
            class="data-grid-row"
            class:row-selected={selectedIndex === index}
            data-row-index={index}
            tabindex="0"
            role="row"
            aria-label="Pipeline #{pipe.build_number} - {statusLabel(pipe.state)}"
            aria-selected={selectedIndex === index}
            onclick={() => viewDetail(pipe)}
            onkeydown={(e) => handleRowKeydown(e, index)}
          >
            <div class="data-grid-cell col-build">
              <span class="mono" class:text-tertiary={true}>#{pipe.build_number || '—'}</span>
            </div>
            <div class="data-grid-cell col-branch">
              <span class="branch-ref" title={pipe.target?.ref_name}>{pipe.target?.ref_name || '—'}</span>
              {#if pipe.target?.commit?.hash}
                <span class="commit-tag" title={pipe.target.commit.hash}>{shortHash(pipe.target.commit.hash)}</span>
              {/if}
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
          </div>
        {/each}

        <!-- Infinite-scroll sentinel -->
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
    width: 210px;
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

  /* Column widths */
  .col-build      { width: 64px;   flex-shrink: 0; }
  .col-branch     { width: 190px;  flex-shrink: 0; }
  .col-status-hdr { width: 96px;   flex-shrink: 0; }
  .col-dur        { width: 80px;   flex-shrink: 0; justify-content: flex-end; padding-right: var(--space-6); }
  .col-creator    { flex: 1; min-width: 80px; }
  .col-date       { width: 142px;  flex-shrink: 0; }

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

  .sk-num    { width: 32px;  height: 12px; }
  .sk-branch { width: 100px; height: 12px; }
  .sk-badge  { width: 56px;  height: 12px; }
  .sk-dur    { width: 44px;  height: 12px; margin-left: auto; }
  .sk-name   { width: 64px;  height: 12px; }
  .sk-date   { width: 88px;  height: 12px; }

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
</style>