<script>
  import { get } from 'svelte/store';
  import { currentScreen, activeProject, pipelines, pipelinesNext, listState, listError, listFilter, listSort, selectedPipeline, selectedSteps, selectedVariables, detailState } from '../stores/appState.js';
  import { listPipelines } from '../stores/api.js';
  import { formatDate, formatDuration, statusLabel, statusClassForState } from './utils.js';

  let loading = $state(false);
  let loadingMore = $state(false);
  let searchTimeout = $state(null);
  let localFilter = $state('');
  let currentPage = $state(1);
  let hasMore = $state(false);

  async function loadPipelines(page = 1) {
    if (!$activeProject) return;
    const isFirstPage = page === 1;
    if (isFirstPage) {
      loading = true;
      listState.set('loading');
      listError.set('');
      currentPage = 1;
    } else {
      loadingMore = true;
    }
    try {
      const data = await listPipelines($activeProject.id, {
        filter: get(listFilter),
        sort: get(listSort),
        page: page,
      });
      if (isFirstPage) {
        pipelines.set(data.values || []);
      } else {
        pipelines.update(existing => [...existing, ...(data.values || [])]);
      }
      pipelinesNext.set(data.next || '');
      hasMore = !!(data.next);
      if ((data.values || []).length > 0) {
        const p = data.values[0];
        if (!p.target) p.target = {};
        if (!p.creator) p.creator = {};
        if (!p.state) p.state = {};
        if (!p.repository) p.repository = {};
      }
      listState.set('ready');
    } catch (e) {
      if (isFirstPage) {
        listError.set(e.message);
        listState.set('error');
      }
    } finally {
      loading = false;
      loadingMore = false;
    }
  }

  async function loadNextPage() {
    const nextPage = currentPage + 1;
    currentPage = nextPage;
    await loadPipelines(nextPage);
  }

  function handleFilterInput() {
    if (searchTimeout) clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      listFilter.set(localFilter);
    }, 400);
  }

  function viewDetail(pipeline) {
    selectedPipeline.set(pipeline);
    selectedSteps.set([]);
    selectedVariables.set([]);
    detailState.set('loading');
    currentScreen.set('detail');
  }

  let lastLoaded = ''; // plain variable, NOT reactive — prevents re-entrant loops

  // Load/reload pipelines when project, filter, or sort changes
  $effect(() => {
    const key = `${$activeProject?.id || ''}:${$listFilter}:${$listSort}`;
    if (!key || key === '::') return;
    if (lastLoaded === key) return;
    lastLoaded = key;
    loadPipelines(1);
  });
</script>

<div class="pipeline-list">
  <div class="list-header">
    <div class="header-left">
      <h2>{$activeProject?.name || 'Unknown'} Pipelines</h2>
      <span class="pipeline-count">{$pipelines.length} pipelines</span>
    </div>
    <div class="header-right">
      <input
        type="text"
        class="filter-input"
        placeholder="Filter pipelines..."
        bind:value={localFilter}
        oninput={handleFilterInput}
      />
      <select class="sort-select" bind:value={$listSort}>
        <option value="-created_on">Newest first</option>
        <option value="+created_on">Oldest first</option>
      </select>
      <button class="btn btn-primary" onclick={() => currentScreen.set('trigger')}>
        ▶ Run Pipeline
      </button>
    </div>
  </div>

  {#if $listState === 'loading' && $pipelines.length === 0}
    <div class="loading-state">
      <div class="spinner"></div>
      <p>Loading pipelines...</p>
    </div>
  {:else if $listState === 'error'}
    <div class="error-state">
      <p>❌ {$listError}</p>
      <button class="btn btn-secondary" onclick={loadPipelines}>Retry</button>
    </div>
  {:else if $pipelines.length === 0}
    <div class="empty-pipelines">
      <p>No pipelines found</p>
      <button class="btn btn-primary" onclick={() => currentScreen.set('trigger')}>
        Trigger a pipeline
      </button>
    </div>
  {:else}
    <div class="table-container">
      <table>
        <thead>
          <tr>
            <th>#</th>
            <th>Target</th>
            <th>Status</th>
            <th>Duration</th>
            <th>Creator</th>
            <th>Created</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#each $pipelines as pipe}
            <tr class="pipe-row" ondblclick={() => viewDetail(pipe)}>
              <td class="cell-num">{pipe.build_number || '—'}</td>
              <td>
                <span class="target-badge">{pipe.target?.ref_name || pipe.target?.type || '—'}</span>
              </td>
              <td>
                <span class="status-badge {statusClassForState(pipe.state)}">
                  {statusLabel(pipe.state)}
                </span>
              </td>
              <td class="cell-duration">
                <div class="duration-tooltip-wrapper">
                  <span>{formatDuration(pipe.created_on, pipe.completed_on, pipe.build_seconds_used || 0)}</span>
                  <div class="tooltip-popup">
                    <div class="tooltip-title">Pipeline #{pipe.build_number || '—'}</div>
                    <div class="tooltip-row"><span class="tip-label">State:</span> <span>{pipe.state?.name || 'unknown'} {pipe.state?.result?.name ? `(${pipe.state.result.name})` : ''}</span></div>
                    <div class="tooltip-row"><span class="tip-label">Target:</span> <span>{pipe.target?.ref_name || pipe.target?.type || '—'}</span></div>
                    <div class="tooltip-row"><span class="tip-label">Trigger:</span> <span>{pipe.trigger?.name || '—'}</span></div>
                    <div class="tooltip-row"><span class="tip-label">Creator:</span> <span>{pipe.creator?.display_name || pipe.creator?.username || '—'}</span></div>
                    <div class="tooltip-row"><span class="tip-label">Commit:</span> <span class="commit-hash">{pipe.target?.commit?.hash?.substring(0, 8) || '—'}</span></div>
                    <div class="tooltip-row"><span class="tip-label">Created:</span> <span>{formatDate(pipe.created_on)}</span></div>
                    <div class="tooltip-row"><span class="tip-label">Completed:</span> <span>{formatDate(pipe.completed_on)}</span></div>
                  </div>
                </div>
              </td>
              <td>{pipe.creator?.display_name || pipe.creator?.username || '—'}</td>
              <td class="cell-date">{formatDate(pipe.created_on)}</td>
              <td>
                <button class="btn btn-small" onclick={() => viewDetail(pipe)}>View</button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

  {#if loading}
    <div class="loading-bar"></div>
  {/if}

  {#if hasMore}
    <div class="load-more-container">
      <button
        class="btn btn-secondary load-more-btn"
        onclick={loadNextPage}
        disabled={loadingMore}
      >
        {#if loadingMore}
          <span class="mini-spinner"></span> Loading...
        {:else}
          Load More Pipelines
        {/if}
      </button>
    </div>
  {/if}
</div>

<style>
  .pipeline-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .list-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 0.75rem;
  }

  .header-left {
    display: flex;
    align-items: baseline;
    gap: 0.75rem;
  }

  .header-left h2 {
    font-size: 1.3rem;
    font-weight: 600;
  }

  .pipeline-count {
    color: #71767b;
    font-size: 0.85rem;
  }

  .header-right {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .filter-input {
    background: #16181c;
    border: 1px solid #2f3336;
    border-radius: 9999px;
    padding: 0.5rem 1rem;
    color: #e7e9ea;
    font-size: 0.9rem;
    width: 200px;
    outline: none;
  }

  .filter-input:focus {
    border-color: #1d9bf0;
  }

  .filter-input::placeholder {
    color: #71767b;
  }

  .sort-select {
    background: #16181c;
    border: 1px solid #2f3336;
    border-radius: 9999px;
    padding: 0.5rem 0.75rem;
    color: #e7e9ea;
    font-size: 0.85rem;
    outline: none;
    cursor: pointer;
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

  .btn-primary {
    background: #1d9bf0;
    color: #fff;
  }

  .btn-primary:hover {
    background: #1a8cd8;
  }

  .btn-secondary {
    background: #2f3336;
    color: #e7e9ea;
  }

  .btn-secondary:hover {
    background: #3e4144;
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

  .loading-state, .error-state, .empty-pipelines {
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

  .table-container {
    overflow-x: auto;
    background: #16181c;
    border-radius: 12px;
    border: 1px solid #2f3336;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.9rem;
  }

  th {
    text-align: left;
    padding: 0.75rem 1rem;
    color: #71767b;
    font-weight: 600;
    font-size: 0.8rem;
    text-transform: uppercase;
    letter-spacing: 0.03em;
    border-bottom: 1px solid #2f3336;
    background: #1a1d23;
  }

  td {
    padding: 0.625rem 1rem;
    border-bottom: 1px solid #1f2228;
  }

  .pipe-row {
    transition: background 0.1s;
  }

  .pipe-row:hover {
    background: #1a1d23;
  }

  .cell-num {
    font-family: 'SF Mono', 'Fira Code', monospace;
    color: #71767b;
  }

  .target-badge {
    background: #1d2e3e;
    color: #6cb6ff;
    padding: 0.15rem 0.5rem;
    border-radius: 6px;
    font-size: 0.8rem;
    font-family: 'SF Mono', 'Fira Code', monospace;
  }

  .status-badge {
    padding: 0.15rem 0.5rem;
    border-radius: 6px;
    font-size: 0.8rem;
    font-weight: 600;
    text-transform: uppercase;
  }

  .status-success { background: #1a3e2a; color: #3fb950; }
  .status-error { background: #3e1a1a; color: #f85149; }
  .status-running { background: #3e3520; color: #d29922; }
  .status-stopped { background: #2f3336; color: #8b949e; }

  .result-text {
    color: #8b949e;
    font-size: 0.8rem;
    margin-left: 0.25rem;
  }

  .cell-duration {
    font-family: 'SF Mono', 'Fira Code', monospace;
    color: #71767b;
  }

  .cell-date {
    font-size: 0.82rem;
    color: #8b949e;
  }

  .loading-bar {
    height: 3px;
    background: #1d9bf0;
    animation: loadingBar 1.5s ease-in-out infinite;
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
  }

  @keyframes loadingBar {
    0% { width: 0%; }
    50% { width: 70%; }
    100% { width: 100%; }
  }

  /* Tooltip styles */
  .duration-tooltip-wrapper {
    position: relative;
    cursor: default;
  }

  .duration-tooltip-wrapper .tooltip-popup {
    display: none;
    position: absolute;
    bottom: 100%;
    left: 50%;
    transform: translateX(-50%);
    margin-bottom: 8px;
    background: #0d1117;
    border: 1px solid #30363d;
    border-radius: 8px;
    padding: 0.75rem 1rem;
    min-width: 260px;
    z-index: 100;
    box-shadow: 0 8px 24px rgba(0,0,0,0.5);
    white-space: nowrap;
  }

  .duration-tooltip-wrapper:hover .tooltip-popup {
    display: block;
  }

  .tooltip-title {
    font-weight: 600;
    font-size: 0.85rem;
    color: #e7e9ea;
    margin-bottom: 0.5rem;
    padding-bottom: 0.4rem;
    border-bottom: 1px solid #21262d;
  }

  .tooltip-row {
    display: flex;
    gap: 0.5rem;
    font-size: 0.78rem;
    margin-bottom: 0.2rem;
    color: #c9d1d9;
  }

  .tip-label {
    color: #8b949e;
    min-width: 70px;
  }

  .commit-hash {
    font-family: 'SF Mono', 'Fira Code', monospace;
    color: #7ee787;
  }

  .load-more-container {
    text-align: center;
    padding: 1rem 0;
  }

  .load-more-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.5rem 1.5rem;
  }

  .load-more-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .mini-spinner {
    width: 14px;
    height: 14px;
    border: 2px solid #3e4144;
    border-top: 2px solid #8b949e;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    display: inline-block;
  }
</style>
