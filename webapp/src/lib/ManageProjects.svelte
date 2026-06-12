<script>
  import { projects, activeProject, activeProjectId, showError, showSuccess, workspaceRepos, workspaceReposState, workspaceReposError, refreshTrigger } from '../stores/appState.js';
  import { navigateTo } from '../stores/router.js';
  import { listProjects, addProject, removeProject, listRepositories } from '../stores/api.js';

  let newWorkspace = $state('');
  let newRepoSlug = $state('');
  let adding = $state(false);
  let removing = $state(null);
  let showSuggestions = $state(false);
  let highlightedIdx = $state(-1);
  let searchLoading = $state(false);
  let debounceTimer = null;

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

  async function handleRemove(projId) {
    if (!confirm('Are you sure you want to remove this project?')) return;
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
  <div class="manage-header">
    <button class="btn btn-secondary" onclick={() => navigateTo($projects.length > 0 ? 'list' : 'manage')}>
      ← Back
    </button>
    <h2>Manage Projects</h2>
  </div>

  <!-- Current Projects -->
  <div class="section">
    <h3>Current Projects ({$projects.length})</h3>
    {#if $projects.length === 0}
      <p class="empty-text">No projects configured. Add one below.</p>
    {:else}
      <div class="projects-list">
        {#each $projects as proj, i}
          <div class="project-card" class:active={i === $activeProjectId}>
            <div class="project-info">
              <div class="project-name">{proj.name}</div>
              <div class="project-meta">
                <span class="meta-label">{proj.workspace}/{proj.repo_slug}</span>
                <span class="meta-id">ID: {proj.id}</span>
              </div>
            </div>
            <div class="project-actions">
              <button
                class="btn btn-small"
                onclick={() => { activeProjectId.set(i); navigateTo('list'); }}
              >
                Select
              </button>
              <button
                class="btn btn-danger-small"
                disabled={removing === proj.id}
                onclick={() => handleRemove(proj.id)}
              >
                {removing === proj.id ? 'Removing...' : 'Remove'}
              </button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Add Project -->
  <div class="section">
    <h3>Add New Project</h3>
    <form class="add-form" onsubmit={handleAdd}>
      <div class="add-row">
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
              required
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
            <span class="form-hint loading-hint">🔍 Searching repositories...</span>
          {:else if $workspaceReposError}
            <span class="form-hint error-hint">❌ {$workspaceReposError}</span>
          {:else if $workspaceReposState === 'ready' && $workspaceRepos.length === 0}
            <span class="form-hint no-results-hint">No repositories found for this workspace</span>
          {:else if $workspaceReposState === 'ready' && !showSuggestions}
            <span class="form-hint">{$workspaceRepos.length} repository(s) — click field to browse</span>
          {/if}
        </div>
        <button type="submit" class="btn btn-primary btn-add" disabled={adding}>
          {adding ? 'Adding...' : '+ Add'}
        </button>
      </div>
    </form>
  </div>
</div>

<style>
  .manage-projects {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .manage-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .manage-header h2 {
    font-size: 1.3rem;
    font-weight: 600;
  }

  .section {
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

  .empty-text {
    color: #71767b;
    font-style: italic;
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

  .btn-secondary:hover { background: #3e4144; }

  .btn-primary {
    background: #1d9bf0;
    color: #fff;
  }

  .btn-primary:hover { background: #1a8cd8; }
  .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

  .btn-small {
    padding: 0.25rem 0.75rem;
    font-size: 0.8rem;
    background: #2f3336;
    color: #e7e9ea;
    border: none;
    border-radius: 9999px;
    cursor: pointer;
  }
  .btn-small:hover { background: #3e4144; }

  .btn-danger-small {
    padding: 0.25rem 0.5rem;
    font-size: 0.8rem;
    background: #b91c1c;
    color: #fff;
    border: none;
    border-radius: 9999px;
    cursor: pointer;
  }
  .btn-danger-small:hover { background: #991b1b; }
  .btn-danger-small:disabled { opacity: 0.5; cursor: not-allowed; }

  .btn-add {
    height: fit-content;
    align-self: flex-end;
  }

  .projects-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .project-card {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem 1rem;
    background: #1a1d23;
    border: 1px solid #2f3336;
    border-radius: 8px;
    transition: border-color 0.15s;
  }

  .project-card.active { border-color: #1d9bf0; }

  .project-info {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }

  .project-name { font-weight: 600; }

  .project-meta {
    display: flex;
    gap: 0.75rem;
    font-size: 0.8rem;
    color: #8b949e;
  }

  .meta-label { font-family: 'SF Mono', 'Fira Code', monospace; }
  .meta-id { color: #484f58; }

  .project-actions {
    display: flex;
    gap: 0.5rem;
  }

  .add-form {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .add-row {
    display: flex;
    gap: 1rem;
    align-items: flex-start;
    flex-wrap: wrap;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    flex: 1;
    min-width: 200px;
  }

  .form-label {
    font-size: 0.8rem;
    font-weight: 600;
    color: #8b949e;
  }

  .form-hint {
    font-size: 0.72rem;
    color: #484f58;
    min-height: 1em;
  }

  .loading-hint { color: #8b949e; }
  .error-hint { color: #f4a2a2; }
  .no-results-hint { color: #c6903b; }

  .form-input {
    background: #0d1117;
    border: 1px solid #2f3336;
    border-radius: 8px;
    padding: 0.5rem 0.75rem;
    color: #e7e9ea;
    font-size: 0.9rem;
    outline: none;
    width: 100%;
    box-sizing: border-box;
  }

  .form-input:focus { border-color: #1d9bf0; }
  .form-input::placeholder { color: #484f58; }

  .slug-group { flex: 2; }

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
    color: #8b949e;
    font-size: 1.1rem;
    cursor: pointer;
    padding: 2px 6px;
    line-height: 1;
    border-radius: 4px;
  }

  .clear-input-btn:hover {
    color: #e7e9ea;
    background: #2f3336;
  }

  .suggestions-dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    margin-top: 4px;
    background: #1a1d23;
    border: 1px solid #1d9bf0;
    border-radius: 8px;
    max-height: 240px;
    overflow-y: auto;
    z-index: 100;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  }

  .suggestion-item {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    width: 100%;
    padding: 0.5rem 0.75rem;
    background: none;
    border: none;
    border-bottom: 1px solid #21262d;
    color: #e7e9ea;
    cursor: pointer;
    text-align: left;
    font-size: 0.85rem;
    transition: background 0.1s;
  }

  .suggestion-item:last-child { border-bottom: none; }
  .suggestion-item:hover { background: #1d2e3e; }
  .suggestion-item.highlighted { background: #1d3e5e; }

  .suggestion-name {
    font-weight: 600;
    font-size: 0.85rem;
  }

  .suggestion-fullname {
    font-size: 0.72rem;
    color: #8b949e;
    font-family: 'SF Mono', 'Fira Code', monospace;
  }

  .suggestions-dropdown::-webkit-scrollbar { width: 6px; }
  .suggestions-dropdown::-webkit-scrollbar-track { background: #1a1d23; }
  .suggestions-dropdown::-webkit-scrollbar-thumb { background: #30363d; border-radius: 3px; }
</style>