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
  let removeConfirm = $state(null); // { id, name }

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
</script>

<div class="manage-projects">
  <div class="toolbar manage-toolbar">
    <button class="btn btn-ghost" onclick={() => navigateTo($projects.length > 0 ? 'list' : 'manage')}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="19" y1="12" x2="5" y2="12"></line>
        <polyline points="12 19 5 12 12 5"></polyline>
      </svg>
      Back
    </button>
    <h2 class="toolbar-title">Manage Projects</h2>
  </div>

  <!-- Current Projects -->
  <div class="panel">
    <div class="panel-header">
      <h3>Current Projects</h3>
      <span class="badge badge-neutral">{$projects.length}</span>
    </div>
    {#if $projects.length === 0}
      <p class="empty-text">No projects configured. Add one below.</p>
    {:else}
      <div class="panel-body">
        <div class="projects-grid">
          {#each $projects as proj, i}
            <div class="project-card" class:selected={i === $activeProjectId}>
              <div class="project-info">
                <div class="project-name">{proj.name}</div>
                <div class="project-meta">
                  <code class="project-path">{proj.workspace}/{proj.repo_slug}</code>
                  <span class="project-id">ID: {proj.id}</span>
                </div>
              </div>
              <div class="project-actions">
                <button
                  class="btn btn-secondary btn-sm"
                  onclick={() => { activeProjectId.set(i); navigateTo('list'); }}
                >
                  Select
                </button>
                <button
                  class="btn btn-danger btn-sm"
                  disabled={removing === proj.id}
                  onclick={() => promptRemove(proj)}
                >
                  {removing === proj.id ? 'Removing…' : 'Remove'}
                </button>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>

  <!-- Add Project -->
  <div class="panel">
    <div class="panel-header">
      <h3>Add Project</h3>
    </div>
    <div class="panel-body">
      <form onsubmit={handleAdd}>
        <div class="add-form">
          <div class="form-row">
            <div class="form-group">
              <label class="form-label" for="workspace">Workspace</label>
              <input
                id="workspace"
                type="text"
                class="form-input"
                placeholder="e.g. secutix"
                bind:value={newWorkspace}
                oninput={handleWorkspaceInput}
                required
              />
              <span class="form-hint">Start typing to search repositories</span>
            </div>
            <div class="form-group slug-group">
              <label class="form-label" for="repoSlug">Repository Slug</label>
              <div class="slug-input-wrapper">
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
                />
                {#if newRepoSlug}
                  <button type="button" class="clear-input-btn" onclick={clearSlugInput} tabindex="-1">&times;</button>
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
                <span class="form-hint hint-loading">Searching repositories…</span>
              {:else if $workspaceReposError}
                <span class="form-hint hint-error">{$workspaceReposError}</span>
              {:else if $workspaceReposState === 'ready' && $workspaceRepos.length === 0}
                <span class="form-hint hint-warning">No repositories found for this workspace</span>
              {:else if $workspaceReposState === 'ready' && !showSuggestions}
                <span class="form-hint">{$workspaceRepos.length} repository(s) — click field to browse</span>
              {/if}
            </div>
            <button type="submit" class="btn btn-primary add-btn" disabled={adding}>
              {adding ? 'Adding…' : '+ Add'}
            </button>
          </div>
        </div>
      </form>
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
  .manage-projects {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
    padding: var(--space-6);
    height: 100%;
    overflow-y: auto;
  }

  .manage-toolbar {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    flex-shrink: 0;
  }

  .toolbar-title {
    font-size: var(--font-size-md);
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
  }

  .empty-text {
    color: var(--text-tertiary);
    font-style: italic;
    font-size: var(--font-size-sm);
    padding: var(--space-6);
    text-align: center;
  }

  /* ── Project Cards ──────────────────────────────────────── */
  .projects-grid {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .project-card {
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: var(--bg-panel-raised);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    padding: var(--space-4) var(--space-5);
    transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
  }
  .project-card:hover { border-color: var(--border-default); }
  .project-card.selected {
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent);
  }

  .project-info {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    min-width: 0;
  }

  .project-name {
    font-weight: 600;
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .project-meta {
    display: flex;
    gap: var(--space-4);
    align-items: center;
    flex-wrap: wrap;
  }

  .project-path {
    font-family: var(--font-mono);
    font-size: var(--font-size-xs);
    background: var(--bg-input);
    padding: 1px 6px;
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .project-id {
    font-size: var(--font-size-xs);
    color: var(--text-tertiary);
  }

  .project-actions {
    display: flex;
    gap: var(--space-3);
    flex-shrink: 0;
  }

  /* ── Add Form ────────────────────────────────────────────── */
  .add-form {
    display: flex;
    flex-direction: column;
  }

  .form-row {
    display: flex;
    gap: var(--space-4);
    align-items: flex-end;
    flex-wrap: wrap;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    flex: 1;
    min-width: 200px;
  }

  .slug-group { flex: 2; }

  .add-btn {
    height: fit-content;
    align-self: flex-end;
  }

  /* Slug wrapper */
  .slug-input-wrapper {
    position: relative;
    display: flex;
    align-items: center;
  }
  .slug-input-wrapper .form-input { padding-right: 2rem; }

  .clear-input-btn {
    position: absolute;
    right: 6px;
    top: 50%;
    transform: translateY(-50%);
    background: none;
    border: none;
    color: var(--text-tertiary);
    font-size: 1.1rem;
    cursor: pointer;
    padding: 2px 6px;
    line-height: 1;
    border-radius: 4px;
    font-family: var(--font-ui);
  }
  .clear-input-btn:hover { color: var(--text-primary); background: var(--bg-hover); }

  /* Suggestions */
  .suggestions-dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    margin-top: 4px;
    background: var(--bg-panel);
    border: 1px solid var(--accent);
    border-radius: var(--radius-md);
    max-height: 240px;
    overflow-y: auto;
    z-index: 100;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
  }
  .suggestions-dropdown::-webkit-scrollbar { width: 6px; }
  .suggestions-dropdown::-webkit-scrollbar-track { background: var(--bg-panel); }
  .suggestions-dropdown::-webkit-scrollbar-thumb { background: var(--border-emphasis); border-radius: 3px; }

  .suggestion-item {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    width: 100%;
    padding: var(--space-3) var(--space-5);
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
  .suggestion-fullname { font-size: var(--font-size-xs); color: var(--text-tertiary); font-family: var(--font-mono); }

  .hint-loading { color: var(--text-tertiary); }
  .hint-error { color: var(--error-text); }
  .hint-warning { color: var(--warning); }
</style>