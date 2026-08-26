<script>
  import { projects, activeProject, activeProjectId, showError, showSuccess, workspaceRepos, workspaceReposState, workspaceReposError, refreshTrigger } from '../stores/appState.js';
  import { navigateTo, navigateFromClick, isNewTabClick, openInNewTab } from '../stores/router.js';
  import { listProjects, addProject, removeProject, listRepositories } from '../stores/api.js';
  import ConfirmModal from './ConfirmModal.svelte';

  let newWorkspace = $state('');
  let newRepoSlug = $state('');
  let adding = $state(false);
  let removing = $state(null);
  let showSuggestions = $state(false);
  let highlightedIdx = $state(-1);
  let searchLoading = $state(false);
  let debounceTimer = null;
  let removeConfirm = $state(null);
  let formExpanded = $state(false);

  let suggestions = $derived.by(() => {
    if (!$workspaceRepos || $workspaceRepos.length === 0) return [];
    const term = newRepoSlug.trim().toLowerCase();
    if (!term) {
      return [...$workspaceRepos].sort((a, b) => {
        const na = (a.name || '').toLowerCase();
        const nb = (b.name || '').toLowerCase();
        if (na !== nb) return na < nb ? -1 : 1;
        return (a.full_name || '').toLowerCase() < (b.full_name || '').toLowerCase() ? -1 : 1;
      });
    }
    return $workspaceRepos
      .filter(r =>
        (r.name && r.name.toLowerCase().includes(term)) ||
        (r.full_name && r.full_name.toLowerCase().includes(term))
      )
      .sort((a, b) => {
        const na = (a.name || '').toLowerCase();
        const nb = (b.name || '').toLowerCase();
        if (na !== nb) return na < nb ? -1 : 1;
        return (a.full_name || '').toLowerCase() < (b.full_name || '').toLowerCase() ? -1 : 1;
      });
  });

  function handleWorkspaceInput() {
    showSuggestions = false;
    highlightedIdx = -1;
    workspaceReposState.set('idle');
    workspaceRepos.set([]);
    if (debounceTimer) clearTimeout(debounceTimer);
    const ws = newWorkspace.trim();
    if (ws.length < 2) return;
    debounceTimer = setTimeout(() => {
      loadRepos(ws);
    }, 500);
  }

  async function loadRepos(workspace) {
    if (!workspace || workspace.length < 2) return;
    workspaceReposState.set('loading');
    searchLoading = true;
    workspaceReposError.set('');
    try {
      const data = await listRepositories(workspace);
      workspaceRepos.set(data.repositories || []);
      workspaceReposState.set('ready');
      if ((data.repositories || []).length > 0) {
        showSuggestions = true;
      }
    } catch (e) {
      workspaceReposError.set(e.message);
      workspaceReposState.set('error');
    } finally {
      searchLoading = false;
    }
  }

  function handleSuggestionKeydown(e) {
    if (!showSuggestions || suggestions.length === 0) {
      if (e.key === 'ArrowDown' && $workspaceRepos.length > 0) {
        showSuggestions = true;
        highlightedIdx = 0;
      }
      return;
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      highlightedIdx = (highlightedIdx + 1) % suggestions.length;
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      highlightedIdx = (highlightedIdx - 1 + suggestions.length) % suggestions.length;
    } else if (e.key === 'Enter' && highlightedIdx >= 0) {
      e.preventDefault();
      selectRepoSlug(suggestions[highlightedIdx]);
    } else if (e.key === 'Escape') {
      showSuggestions = false;
      highlightedIdx = -1;
    }
  }

  function selectRepoSlug(repo) {
    newRepoSlug = repo.name;
    showSuggestions = false;
    highlightedIdx = -1;
  }

  function clearSlugInput() {
    newRepoSlug = '';
    showSuggestions = false;
    highlightedIdx = -1;
  }

  function hideSuggestions() {
    setTimeout(() => { showSuggestions = false; highlightedIdx = -1; }, 200);
  }

  async function handleAdd(e) {
    e.preventDefault();
    if (!newWorkspace.trim() || !newRepoSlug.trim()) {
      showError('Both workspace and repository slug are required');
      return;
    }
    adding = true;
    try {
      await addProject(newWorkspace.trim(), newRepoSlug.trim());
      showSuccess('Project added successfully');
      newWorkspace = '';
      newRepoSlug = '';
      workspaceRepos.set([]);
      workspaceReposState.set('idle');
      showSuggestions = false;
      formExpanded = false;
      const data = await listProjects();
      projects.set(data.projects || []);
    } catch (e) {
      showError(e.message);
    } finally {
      adding = false;
    }
  }

  function promptRemove(proj) {
    removeConfirm = { id: proj.id, name: proj.name };
  }

  async function confirmRemove() {
    if (!removeConfirm) return;
    const projId = removeConfirm.id;
    removeConfirm = null;
    removing = projId;
    try {
      await removeProject(projId);
      showSuccess('Project removed');
      const data = await listProjects();
      projects.set(data.projects || []);
      if ($projects.length === 0) {
        activeProjectId.set(0);
      } else if ($activeProjectId >= $projects.length) {
        activeProjectId.set($projects.length - 1);
      }
    } catch (e) {
      showError(e.message);
    } finally {
      removing = null;
    }
  }

  $effect(() => {
    void $refreshTrigger;
    (async () => {
      try {
        const data = await listProjects();
        projects.set(data.projects || []);
      } catch (e) {
        showError(e.message);
      }
    })();
  });

  function projectIcon(w) {
    const first = (w || '?').charAt(0).toUpperCase();
    const hue = [...(w || 'x')].reduce((a, c) => a + c.charCodeAt(0), 0) * 17 % 360;
    return { letter: first, hue };
  }
</script>

<div class="manage-page">
  <!-- Header Bar ──────────────────────────────────────── -->
  <div class="manage-header">
    <div class="manage-header-left">
      <button class="btn btn-ghost btn-sm" onclick={(e) => navigateFromClick(e, $projects.length > 0 ? 'list' : 'manage')}>
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="19" y1="12" x2="5" y2="12"></line><polyline points="12 19 5 12 12 5"></polyline>
        </svg>
        Back
      </button>
      <h2 class="manage-title">Manage Projects</h2>
      <span class="badge badge-neutral">{$projects.length}</span>
    </div>
    <div class="manage-header-right">
      <button
        class="btn btn-primary btn-sm"
        onclick={() => { formExpanded = !formExpanded; }}
        class:btn-secondary={formExpanded}
      >
        {#if formExpanded}
          Cancel
        {:else}
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          Add Project
        {/if}
      </button>
    </div>
  </div>

  <!-- Add Form (conditionally visible) ────────────────── -->
  {#if formExpanded}
    <div class="add-section">
      <div class="add-section-inner">
        <div class="add-section-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
            <line x1="12" y1="11" x2="12" y2="17"></line>
            <line x1="9" y1="14" x2="15" y2="14"></line>
          </svg>
        </div>
        <form onsubmit={handleAdd} class="add-form">
          <div class="form-row">
            <div class="form-field">
              <label class="form-label" for="workspace">Workspace</label>
              <div class="input-with-icon">
                <svg class="input-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
                  <circle cx="9" cy="7" r="4"></circle>
                </svg>
                <input
                  id="workspace"
                  type="text"
                  class="form-input"
                  placeholder="e.g. secutix"
                  bind:value={newWorkspace}
                  oninput={handleWorkspaceInput}
                  autocomplete="off"
                  required
                />
              </div>
            </div>

            <div class="form-field">
              <label class="form-label" for="repoSlug">Repository</label>
              <div class="slug-input-wrapper">
                <svg class="input-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
                </svg>
                <input
                  id="repoSlug"
                  type="text"
                  class="form-input"
                  placeholder="e.g. infra-bbc_pipeline-tui"
                  bind:value={newRepoSlug}
                  onkeydown={handleSuggestionKeydown}
                  onfocus={() => { if ($workspaceRepos.length > 0) showSuggestions = true; }}
                  onblur={hideSuggestions}
                  oninput={() => { if ($workspaceRepos.length > 0) showSuggestions = true; highlightedIdx = -1; }}
                  autocomplete="off"
                  required
                />
                {#if newRepoSlug}
                  <button type="button" class="clear-btn" onclick={clearSlugInput} tabindex="-1" aria-label="Clear repository input">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line>
                    </svg>
                  </button>
                {/if}
                {#if showSuggestions && suggestions.length > 0}
                  <div class="suggestions-dropdown">
                    {#each suggestions as repo, i}
                      <button
                        type="button"
                        class="suggestion-item"
                        class:highlighted={i === highlightedIdx}
                        onclick={() => selectRepoSlug(repo)}
                      >
                        <span class="suggestion-name">{repo.name || repo.full_name}</span>
                        <span class="suggestion-fullname">{repo.full_name || ''}</span>
                      </button>
                    {/each}
                  </div>
                {/if}
              </div>
              <div class="form-help-row">
                {#if searchLoading}
                  <span class="form-help form-help-loading">
                    <span class="mini-spinner"></span> Searching repositories…
                  </span>
                {:else if $workspaceReposError}
                  <span class="form-help form-help-error">{$workspaceReposError}</span>
                {:else if $workspaceReposState === 'ready' && $workspaceRepos.length === 0}
                  <span class="form-help form-help-warn">No repositories found for this workspace.</span>
                {:else if $workspaceReposState === 'ready' && !showSuggestions}
                  <span class="form-help">{$workspaceRepos.length} repositories found. Click the field to browse.</span>
                {:else}
                  <span class="form-help">&nbsp;</span>
                {/if}
              </div>
            </div>

            <button type="submit" class="btn btn-primary add-submit-inline" disabled={adding || !newWorkspace.trim() || !newRepoSlug.trim()}>
              {#if adding}
                <span class="mini-spinner"></span> Adding…
              {:else}
                Add
              {/if}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}

  <!-- Project List ────────────────────────────────────── -->
  <div class="manage-body">
    {#if $projects.length === 0}
      <div class="empty-state">
        <div class="empty-icon-container">
          <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" opacity="0.4">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
          </svg>
        </div>
        <h3 class="empty-title">No projects configured</h3>
        <p class="empty-desc">Add a Bitbucket repository to start browsing its pipelines.</p>
        {#if !formExpanded}
          <button class="btn btn-primary btn-sm" onclick={() => { formExpanded = true; }}>
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
            Add First Project
          </button>
        {/if}
      </div>
    {:else}
      <div class="project-grid">
        {#each $projects as proj, i}
          <div class="project-card" class:active={i === $activeProjectId}>
            <div class="project-card-main">
              <div
                class="project-card-avatar"
                style="background: hsl({projectIcon(proj.workspace).hue}, 40%, 20%); color: hsl({projectIcon(proj.workspace).hue}, 55%, 78%)"
              >
                {projectIcon(proj.workspace).letter}
              </div>
              <div class="project-card-body">
                <div class="project-card-name">{proj.name}</div>
                <div class="project-card-path">
                  <span class="pc-workspace">{proj.workspace}</span>
                  <span class="pc-sep">/</span>
                  <span class="pc-slug">{proj.repo_slug}</span>
                </div>
              </div>
              <div class="project-card-meta">
                {#if i === $activeProjectId}
                  <span class="active-badge">
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><polyline points="20 6 9 17 4 12"></polyline></svg>
                    Active
                  </span>
                {/if}
              </div>
            </div>
            <div class="project-card-actions">
              <button
                class="btn btn-secondary btn-sm"
                onclick={(e) => {
                  if (isNewTabClick(e)) {
                    e.preventDefault();
                    openInNewTab('list', undefined, undefined, proj);
                    return;
                  }
                  activeProjectId.set(i);
                  navigateTo('list');
                }}
                title="Open in pipeline list"
              >
                Open
              </button>
              <button
                class="btn btn-ghost-danger btn-sm"
                disabled={removing === proj.id}
                onclick={() => promptRemove(proj)}
                title="Remove project"
              >
                {#if removing === proj.id}
                  <span class="mini-spinner-inline"></span> Removing…
                {:else}
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
                {/if}
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<ConfirmModal
  open={removeConfirm !== null}
  title="Remove Project"
  message={`Are you sure you want to remove project "${removeConfirm?.name || '—'}"? This action cannot be undone.`}
  confirmText="Remove"
  danger={true}
  onconfirm={confirmRemove}
  oncancel={() => (removeConfirm = null)}
/>

<style>
  .manage-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
    background: var(--bg-root);
  }

  /* ── Header ──────────────────────────────────────────── */
  .manage-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-4) var(--space-6);
    border-bottom: 1px solid var(--border-subtle);
    flex-shrink: 0;
    background: var(--bg-root);
  }

  .manage-header-left {
    display: flex;
    align-items: center;
    gap: var(--space-4);
  }

  .manage-title {
    font-size: var(--font-size-lg);
    font-weight: 700;
    color: var(--text-primary);
    margin: 0;
    letter-spacing: -0.01em;
  }

  .manage-header-right {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  /* ── Add Section (inline form) ───────────────────────── */
  .add-section {
    border-bottom: 1px solid var(--border-subtle);
    background: var(--bg-panel);
    flex-shrink: 0;
  }

  .add-section-inner {
    display: flex;
    align-items: flex-start;
    gap: var(--space-4);
    padding: var(--space-4) var(--space-6);
  }

  .add-section-icon {
    width: 40px;
    height: 40px;
    border-radius: var(--radius-md);
    background: var(--accent-muted);
    color: var(--accent-text);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    margin-top: var(--space-1);
  }

  .add-form {
    flex: 1;
  }

  .form-row {
    display: flex;
    align-items: flex-start;
    gap: var(--space-4);
  }

  .form-field {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    flex: 1;
    min-width: 0;
  }

  .form-label {
    font-size: 10px;
    font-weight: 700;
    color: var(--text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .input-with-icon,
  .slug-input-wrapper {
    position: relative;
    display: flex;
    align-items: center;
  }

  .input-icon {
    position: absolute;
    left: var(--space-3);
    color: var(--text-tertiary);
    pointer-events: none;
    flex-shrink: 0;
    z-index: 1;
  }

  .input-with-icon .form-input,
  .slug-input-wrapper .form-input {
    padding-left: calc(var(--space-3) * 2 + 14px);
    padding-right: var(--space-3);
  }

  .form-input {
    width: 100%;
    background: var(--bg-input);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md);
    padding: var(--space-2) var(--space-3);
    color: var(--text-primary);
    font-family: var(--font-mono);
    font-size: var(--font-size-sm);
    outline: none;
    transition: border-color 0.15s, box-shadow 0.15s;
  }
  .form-input:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px var(--accent-muted);
  }
  .form-input::placeholder {
    color: var(--text-tertiary);
    font-family: var(--font-ui);
  }

  .slug-input-wrapper .form-input {
    padding-right: 2.5rem;
  }

  .clear-btn {
    position: absolute;
    right: 4px;
    top: 50%;
    transform: translateY(-50%);
    background: none;
    border: none;
    color: var(--text-tertiary);
    cursor: pointer;
    padding: 4px;
    line-height: 1;
    border-radius: 4px;
    display: flex;
    align-items: center;
    z-index: 2;
  }
  .clear-btn:hover { color: var(--text-primary); background: var(--bg-hover); }

  .form-help-row {
    min-height: 20px;
  }

  .form-help {
    font-size: 11px;
    color: var(--text-tertiary);
    line-height: 1.4;
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .form-help-loading { color: var(--accent-text); }
  .form-help-error { color: var(--error-text); }
  .form-help-warn { color: var(--warning); }

  .add-submit-inline {
    align-self: flex-end;
    margin-bottom: var(--space-1);
    white-space: nowrap;
    padding: var(--space-2) var(--space-5);
  }

  /* ── Body ────────────────────────────────────────────── */
  .manage-body {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-5) var(--space-6);
  }

  /* ── Empty State ─────────────────────────────────────── */
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    text-align: center;
    gap: var(--space-4);
    padding: var(--space-8);
    color: var(--text-tertiary);
    min-height: 300px;
  }

  .empty-icon-container {
    width: 64px;
    height: 64px;
    border-radius: 50%;
    background: var(--bg-input);
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: var(--space-2);
  }

  .empty-title {
    font-size: var(--font-size-md);
    font-weight: 600;
    color: var(--text-secondary);
    margin: 0;
  }

  .empty-desc {
    font-size: var(--font-size-sm);
    color: var(--text-tertiary);
    max-width: 300px;
    line-height: 1.5;
    margin: 0;
  }

  /* ── Project Grid ────────────────────────────────────── */
  .project-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
    gap: var(--space-4);
  }

  .project-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    padding: var(--space-4) var(--space-5);
    border-radius: var(--radius-lg);
    border: 1px solid var(--border-subtle);
    background: var(--bg-panel);
    transition: border-color 0.2s, background 0.2s, box-shadow 0.2s;
  }
  .project-card:hover {
    background: var(--bg-panel-raised);
    border-color: var(--border-default);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  }
  .project-card.active {
    border-color: var(--accent);
    background: var(--bg-panel-raised);
    box-shadow: 0 0 0 1px var(--accent), 0 2px 12px rgba(0, 122, 204, 0.1);
  }

  .project-card-main {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    flex: 1;
    min-width: 0;
  }

  .project-card-avatar {
    width: 42px;
    height: 42px;
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    font-size: 16px;
    flex-shrink: 0;
    user-select: none;
    letter-spacing: 0.02em;
  }

  .project-card-body {
    flex: 1;
    min-width: 0;
  }

  .project-card-name {
    font-weight: 600;
    font-size: var(--font-size-md);
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    margin-bottom: 2px;
  }

  .project-card-path {
    display: flex;
    align-items: center;
    gap: 2px;
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--text-tertiary);
  }

  .pc-workspace {
    color: var(--text-secondary);
    font-weight: 500;
  }

  .pc-sep {
    color: var(--border-emphasis);
    margin: 0 1px;
  }

  .pc-slug {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text-tertiary);
  }

  .project-card-meta {
    flex-shrink: 0;
  }

  .active-badge {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.03em;
    color: var(--success);
    background: rgba(52, 211, 153, 0.1);
    padding: 3px 10px;
    border-radius: var(--radius-sm);
  }

  .project-card-actions {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-shrink: 0;
  }

  .btn-ghost-danger {
    color: var(--text-tertiary);
  }
  .btn-ghost-danger:hover {
    color: var(--error-text);
    background: var(--error-muted);
  }

  /* ── Mini Spinners ───────────────────────────────────── */
  .mini-spinner {
    width: 12px;
    height: 12px;
    border: 2px solid currentColor;
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
    flex-shrink: 0;
  }

  .mini-spinner-inline {
    display: inline-block;
    width: 10px;
    height: 10px;
    border: 2px solid currentColor;
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
    flex-shrink: 0;
  }

  @keyframes spin { to { transform: rotate(360deg); } }

  /* ── Suggestions Dropdown ────────────────────────────── */
  .suggestions-dropdown {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    right: 0;
    background: var(--bg-panel);
    border: 1px solid var(--accent);
    border-radius: var(--radius-md);
    max-height: 240px;
    overflow-y: auto;
    z-index: 100;
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.5);
  }
  .suggestions-dropdown::-webkit-scrollbar { width: 6px; }
  .suggestions-dropdown::-webkit-scrollbar-track { background: var(--bg-panel); }
  .suggestions-dropdown::-webkit-scrollbar-thumb { background: var(--border-emphasis); border-radius: 3px; }

  .suggestion-item {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    width: 100%;
    padding: var(--space-3) var(--space-4);
    background: none;
    border: none;
    border-bottom: 1px solid var(--border-subtle);
    color: var(--text-primary);
    cursor: pointer;
    text-align: left;
    font-size: var(--font-size-sm);
    transition: background 0.1s;
    font-family: var(--font-ui);
  }
  .suggestion-item:last-child { border-bottom: none; }
  .suggestion-item:hover { background: var(--bg-hover); }
  .suggestion-item.highlighted { background: var(--bg-selection); }

  .suggestion-name { font-weight: 600; font-size: var(--font-size-sm); }
  .suggestion-fullname { font-size: 11px; color: var(--text-tertiary); font-family: var(--font-mono); }
</style>
