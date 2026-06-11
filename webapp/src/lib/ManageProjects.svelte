<script>
  import { currentScreen, projects, activeProject, activeProjectId, showError, showSuccess, workspaceRepos, workspaceReposState, workspaceReposError } from '../stores/appState.js';
  import { listProjects, addProject, removeProject, listRepositories } from '../stores/api.js';

  let newWorkspace = $state('');
  let newRepoSlug = $state('');
  let adding = $state(false);
  let removing = $state(null); // id being removed

  async function loadRepos(workspace) {
    if (!workspace.trim()) return;
    workspaceReposState.set('loading');
    workspaceReposError.set('');
    try {
      const data = await listRepositories(workspace.trim());
      workspaceRepos.set(data.repositories || []);
      workspaceReposState.set('ready');
    } catch (e) {
      workspaceReposError.set(e.message);
      workspaceReposState.set('error');
    }
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
      // Reload projects
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
      // Reset active project if needed
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
</script>

<div class="manage-projects">
  <div class="manage-header">
    <button class="btn btn-secondary" onclick={() => currentScreen.set($projects.length > 0 ? 'list' : 'manage')}>
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
                onclick={() => { activeProjectId.set(i); currentScreen.set('list'); }}
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
          <div class="input-with-button">
            <input
              id="workspace"
              type="text"
              class="form-input"
              placeholder="e.g. secutix"
              bind:value={newWorkspace}
              required
            />
            <button
              type="button"
              class="btn btn-secondary"
              onclick={() => loadRepos(newWorkspace)}
            >
              🔍 Browse
            </button>
          </div>
        </div>
        <div class="form-group">
          <label class="form-label" for="repoSlug">Repository Slug</label>
          <input
            id="repoSlug"
            type="text"
            class="form-input"
            placeholder="e.g. infra-bbc_pipeline-tui"
            bind:value={newRepoSlug}
            required
          />
        </div>
        <button type="submit" class="btn btn-primary btn-add" disabled={adding}>
          {adding ? 'Adding...' : '+ Add'}
        </button>
      </div>
    </form>

    <!-- Repository browser results -->
    {#if workspaceReposState === 'loading'}
      <div class="repo-results loading-text">Loading repositories...</div>
    {:else if workspaceReposState === 'error'}
      <div class="repo-results error-text">❌ {workspaceReposError}</div>
    {:else if workspaceReposState === 'ready' && workspaceRepos.length > 0}
      <div class="repo-results">
        <span class="repo-count">Found {workspaceRepos.length} repositories</span>
        <div class="repo-grid">
          {#each workspaceRepos as repo}
            <button
              class="repo-chip"
              onclick={async () => {
                newRepoSlug = repo.slug;
                try {
                  adding = true;
                  await addProject(newWorkspace.trim(), repo.slug);
                  showSuccess('Project added successfully');
                  newRepoSlug = '';
                  workspaceReposState.set('idle');
                  workspaceRepos.set([]);
                  const data = await listProjects();
                  projects.set(data.projects || []);
                } catch (e) {
                  showError(e.message);
                } finally {
                  adding = false;
                }
              }}
            >
              + {repo.name}
              <span class="repo-slug">{repo.full_name}</span>
            </button>
          {/each}
        </div>
      </div>
    {:else if workspaceReposState === 'ready' && workspaceRepos.length === 0}
      <div class="repo-results">No repositories found for this workspace.</div>
    {/if}
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

  .btn-secondary:hover {
    background: #3e4144;
  }

  .btn-primary {
    background: #1d9bf0;
    color: #fff;
  }

  .btn-primary:hover {
    background: #1a8cd8;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
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

  .btn-danger-small {
    padding: 0.25rem 0.5rem;
    font-size: 0.8rem;
    background: #b91c1c;
    color: #fff;
    border: none;
    border-radius: 9999px;
    cursor: pointer;
  }

  .btn-danger-small:hover {
    background: #991b1b;
  }

  .btn-danger-small:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

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

  .project-card.active {
    border-color: #1d9bf0;
  }

  .project-info {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }

  .project-name {
    font-weight: 600;
  }

  .project-meta {
    display: flex;
    gap: 0.75rem;
    font-size: 0.8rem;
    color: #8b949e;
  }

  .meta-label {
    font-family: 'SF Mono', 'Fira Code', monospace;
  }

  .meta-id {
    color: #484f58;
  }

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
    align-items: flex-end;
    flex-wrap: wrap;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    flex: 1;
    min-width: 160px;
  }

  .form-label {
    font-size: 0.8rem;
    font-weight: 600;
    color: #8b949e;
  }

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

  .form-input:focus {
    border-color: #1d9bf0;
  }

  .form-input::placeholder {
    color: #484f58;
  }

  .input-with-button {
    display: flex;
    gap: 0.35rem;
  }

  .input-with-button .form-input {
    flex: 1;
  }

  .repo-results {
    margin-top: 0.75rem;
    padding: 1rem;
    background: #1a1d23;
    border: 1px solid #2f3336;
    border-radius: 8px;
  }

  .repo-count {
    display: block;
    margin-bottom: 0.75rem;
    font-size: 0.85rem;
    color: #8b949e;
  }

  .repo-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    max-height: 300px;
    overflow-y: auto;
  }

  .repo-chip {
    display: flex;
    flex-direction: column;
    background: #0d1117;
    border: 1px solid #2f3336;
    border-radius: 8px;
    padding: 0.5rem 0.75rem;
    cursor: pointer;
    transition: all 0.15s;
    text-align: left;
    color: #e7e9ea;
    font-size: 0.85rem;
  }

  .repo-chip:hover {
    border-color: #1d9bf0;
    background: #1d2e3e;
  }

  .repo-slug {
    font-size: 0.7rem;
    color: #8b949e;
    font-family: 'SF Mono', 'Fira Code', monospace;
  }

  .loading-text {
    color: #8b949e;
  }

  .error-text {
    color: #f4a2a2;
  }
</style>