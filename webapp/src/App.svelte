<script>
  import { projects, activeProjectId, activeProject, notification } from './stores/appState.js';
  import { page, initRoute, navigateTo, workspaceFromUrl, repoSlugFromUrl } from './stores/router.js';
  import { listProjects } from './stores/api.js';
  import Navbar from './lib/Navbar.svelte';
  import PipelineList from './lib/PipelineList.svelte';
  import PipelineDetail from './lib/PipelineDetail.svelte';
  import PipelineLog from './lib/PipelineLog.svelte';
  import ManageProjects from './lib/ManageProjects.svelte';
  import PipelineTrigger from './lib/PipelineTrigger.svelte';
  import Notification from './lib/Notification.svelte';

  let loading = $state(true);
  let loadError = $state(null);
  let dataLoaded = false;

  /** Find project index by workspace and repoSlug. Returns -1 if not found. */
  function findProjectIndex(workspace, repoSlug) {
    return $projects.findIndex(p => p.workspace === workspace && p.repo_slug === repoSlug);
  }

  async function loadData() {
    loading = true;
    loadError = null;
    try {
      const data = await listProjects();
      projects.set(data.projects || []);
      const count = (data.projects || []).length;

      if (count > 0) {
        // Restore active project from workspace/repoSlug in URL, otherwise default to 0
        const ws = $workspaceFromUrl;
        const rs = $repoSlugFromUrl;
        let idx = -1;
        if (ws && rs) {
          idx = findProjectIndex(ws, rs);
        }
        if (idx < 0) idx = 0;
        activeProjectId.set(idx);
        initRoute(count > 0);
      } else {
        initRoute(false);
      }
    } catch (e) {
      console.error('Failed to load projects:', e);
      loadError = e.message || 'Failed to load projects';
    } finally {
      loading = false;
    }
  }

  // Keep the URL hash in sync with the current project and page.
  // Only syncs for pages that involve a project (not 'manage').
  $effect(() => {
    const pg = $page;
    if (!pg || pg === 'manage') return;
    const proj = $activeProject;
    if (!proj) return;
    const ws = encodeURIComponent(proj.workspace);
    const rs = encodeURIComponent(proj.repo_slug);

    let expected;
    if (pg === 'list') {
      expected = `#/bbc/${ws}/${rs}`;
    } else if (pg === 'trigger') {
      expected = `#/bbc/${ws}/${rs}/trigger`;
    } else {
      // For detail/logs pages, keep the rest of the hash intact
      return;
    }

    if (window.location.hash !== expected) {
      window.location.hash = expected;
    }
  });

  // Restore activeProjectId from URL workspace/repoSlug when navigating
  $effect(() => {
    const ws = $workspaceFromUrl;
    const rs = $repoSlugFromUrl;
    if (!ws || !rs) return;
    const idx = findProjectIndex(ws, rs);
    if (idx >= 0 && idx !== $activeProjectId && idx < $projects.length) {
      activeProjectId.set(idx);
    }
  });

  $effect(() => {
    if (dataLoaded) return;
    dataLoaded = true;
    loadData();
  });
</script>

{#if loading}
  <div class="loading-screen">
    <div class="spinner"></div>
    <p>Loading pipeline manager...</p>
  </div>
{:else if loadError}
  <div class="loading-screen error">
    <div class="error-icon">⚠️</div>
    <p>{loadError}</p>
    <button class="btn btn-primary" onclick={loadData}>Retry</button>
  </div>
{:else}
  <div class="app">
    <Notification />
    <Navbar />

    <main class="main-content">
      {#if $activeProject}
        {#if $page === 'list'}
          <PipelineList />
        {:else if $page === 'detail'}
          <PipelineDetail />
        {:else if $page === 'logs'}
          <PipelineLog />
        {:else if $page === 'trigger'}
          <PipelineTrigger />
        {:else if $page === 'manage'}
          <ManageProjects />
        {/if}
      {:else}
        {#if $page === 'manage'}
          <ManageProjects />
        {:else}
          <div class="empty-state">
            <div class="empty-icon">🚀</div>
            <h2>No Projects Configured</h2>
            <p>Go to Manage Projects to add your first Bitbucket repository.</p>
            <button class="btn btn-primary" onclick={() => navigateTo('manage')}>
              Manage Projects
            </button>
          </div>
        {/if}
      {/if}
    </main>
  </div>
{/if}

<style>
  :global(*) {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
  }

  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    background: #0f1419;
    color: #e7e9ea;
    min-height: 100vh;
  }

  .loading-screen {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    gap: 1rem;
    color: #71767b;
  }

  .loading-screen.error {
    color: #e7e9ea;
  }

  .error-icon {
    font-size: 3rem;
  }

  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid #2f3336;
    border-top: 3px solid #1d9bf0;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .app {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  .main-content {
    flex: 1;
    max-width: 1100px;
    width: 100%;
    margin: 0 auto;
    padding: 1.5rem;
  }

  .empty-state {
    text-align: center;
    padding: 4rem 2rem;
    background: #16181c;
    border-radius: 16px;
    border: 1px solid #2f3336;
  }

  .empty-icon {
    font-size: 4rem;
    margin-bottom: 1rem;
  }

  .empty-state h2 {
    font-size: 1.5rem;
    margin-bottom: 0.5rem;
  }

  .empty-state p {
    color: #71767b;
    margin-bottom: 1.5rem;
  }

  .btn {
    padding: 0.625rem 1.25rem;
    border: none;
    border-radius: 9999px;
    font-size: 0.9rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.2s;
  }

  .btn-primary {
    background: #1d9bf0;
    color: #fff;
  }

  .btn-primary:hover {
    background: #1a8cd8;
  }
</style>