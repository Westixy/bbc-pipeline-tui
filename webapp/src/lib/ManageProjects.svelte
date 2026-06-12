<script>
  import { projects, activeProject, activeProjectId, showError, showSuccess, workspaceRepos, workspaceReposState, workspaceReposError, refreshTrigger } from '../stores/appState.js';
  import { navigateTo } from '../stores/router.js';
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
  <div class="manage-header">
    <div class="manage-header-left">
      <button class="btn btn-ghost btn-sm" onclick={() => navigateTo($projects.length > 0 ? 'list' : 'manage')}>
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="19" y1="12" x2="5" y2="12"></line><polyline points="12 19 5 12 12 5"></polyline>
        </svg>
        Back
      </button>
      <h2 class="manage-title">Projects</h2>
      <span class="badge badge-neutral">{$projects.length}</span>
    </div>
  </div>

  <div class="manage-body">
    <div class="manage-left">
      {#if $projects.length === 0}
        <div class="empty-state">
          <div class="empty-icon-container">
            <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.2" opacity="0.4">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
            </svg>
          </div>
          <h3 class="empty-title">No projects yet</h3>
          <p class="empty-desc">Add your first Bitbucket project to start browsing pipeline data.</p>
        </div>
      {:else}
        <div class="project-list">
          {#each $projects as proj, i}
            <div class="project-row" class:active={i === $activeProjectId}>
              <div
                class="project-avatar"
                style="background: hsl({projectIcon(proj.workspace).hue}, 45%, 25%); color: hsl({projectIcon(proj.workspace).hue}, 60%, 80%)"
              >
                {projectIcon(proj.workspace).letter}
              </div>
              <div class="project-body">
                <div class="project-name">{proj.name}</div>
                <div class="project-path">
                  <span class="project-workspace">{proj.workspace}</span>
                  <span class="project-sep">/</span>
                  <span class="project-slug">{proj.repo_slug}</span>
                </div>
              </div>
              <div class="project-actions">
                <button
                  class="btn btn-secondary btn-sm"
                  onclick={() => { activeProjectId.set(i); navigateTo('list'); }}
                  title="Open in pipeline list"
                >
                  Select
                </button>
                <button
                  class="btn btn-ghost-danger btn-sm"
                  disabled={removing === proj.id}
                  onclick={() => promptRemove(proj)}
                  title="Remove project"
                >
                  {#if removing === proj.id}
                    Removing…
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

    <div class="manage-right">
      <div class="add-panel">
        <div class="add-panel-header">
          <h3 class="add-panel-title">Add Project</h3>
          <p class="add-panel-desc">Connect a Bitbucket repository to monitor its pipelines.</p>
        </div>
        <form onsubmit={handleAdd} class="add-form">
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
            <span class="form-help">Enter a Bitbucket workspace to search its repositories.</span>
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
            {/if}
          </div>

          <button type="submit" class="btn btn-primary add-submit-btn" disabled={adding || !newWorkspace.trim() || !newRepoSlug.trim()}>
            {#if adding}
              <span class="mini-spinner"></span> Adding…
            {:else}
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line>
              </svg>
              Add Project
            {/if}
          </button>
        </form>
      </div>
    </div>
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
  }

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

  .manage-body {
    display: flex;
    flex: 1;
    overflow: hidden;
  }

  .manage-left {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-5);
    border-right: 1px solid var(--border-subtle);
    background: var(--bg-root);
  }

  .manage-right {
    width: 400px;
    flex-shrink: 0;
    overflow-y: auto;
    padding: var(--space-5);
    background: var(--bg-panel);
  }

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
    max-width: 280px;
    line-height: 1.5;
    margin: 0;
  }

  .project-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .project-row {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    padding: var(--space-4);
    border-radius: var(--radius-md);
    border: 1px solid transparent;
    background: var(--bg-panel);
    transition: border-color 0.15s, background 0.15s, box-shadow 0.15s;
  }
  .project-row:hover {
    background: var(--bg-panel-raised);
    border-color: var(--border-subtle);
  }
  .project-row.active {
    border-color: var(--accent);
    background: var(--bg-panel-raised);
    box-shadow: 0 0 0 1px var(--accent);
  }

  .project-avatar {
    width: 36px;
    height: 36px;
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    font-size: 14px;
    flex-shrink: 0;
    user-select: none;
  }

  .project-body {
    flex: 1;
    min-width: 0;
  }

  .project-name {
    font-weight: 600;
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    margin-bottom: 2px;
  }

  .project-path {
    display: flex;
    align-items: center;
    gap: 2px;
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--text-tertiary);
  }

  .project-workspace {
    color: var(--text-secondary);
    font-weight: 500;
  }

  .project-sep {
    color: var(--border-emphasis);
    margin: 0 1px;
  }

  .project-slug {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .project-actions {
    display: flex;
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

  .add-panel {
    display: flex;
    flex-direction: column;
  }

  .add-panel-header {
    margin-bottom: var(--space-6);
  }

  .add-panel-title {
    font-size: var(--font-size-md);
    font-weight: 600;
    color: var(--text-primary);
    margin: 0 0 var(--space-1);
  }

  .add-panel-desc {
    font-size: var(--font-size-xs);
    color: var(--text-tertiary);
    margin: 0;
    line-height: 1.4;
  }

  .add-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
  }

  .form-field {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .form-label {
    font-size: var(--font-size-xs);
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
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
    padding: var(--space-3) var(--space-4);
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
  }
  .clear-btn:hover { color: var(--text-primary); background: var(--bg-hover); }

  .form-help {
    font-size: 11px;
    color: var(--text-tertiary);
    line-height: 1.4;
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .form-help-loading {
    color: var(--accent-text);
  }

  .form-help-error {
    color: var(--error-text);
  }

  .form-help-warn {
    color: var(--warning);
  }

  .mini-spinner {
    width: 12px;
    height: 12px;
    border: 2px solid currentColor;
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
    flex-shrink: 0;
  }

  @keyframes spin { to { transform: rotate(360deg); } }

  .add-submit-btn {
    width: 100%;
    justify-content: center;
    margin-top: var(--space-2);
  }

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
    box-shadow: 0 12px 32px rgba(0, 0, 0, 0.45);
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