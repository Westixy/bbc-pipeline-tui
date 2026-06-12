<script>
  import { projects, activeProjectId, activeProject, notification } from './stores/appState.js';
  import { page, initRoute, navigateTo, workspaceFromUrl, repoSlugFromUrl } from './stores/router.js';
  import { listProjects } from './stores/api.js';
  import { registerShortcut, handleShortcut } from './lib/keyboard.js';
  import Navbar from './lib/Navbar.svelte';
  import PipelineList from './lib/PipelineList.svelte';
  import PipelineDetail from './lib/PipelineDetail.svelte';
  import PipelineLog from './lib/PipelineLog.svelte';
  import ManageProjects from './lib/ManageProjects.svelte';
  import PipelineTrigger from './lib/PipelineTrigger.svelte';
  import Notification from './lib/Notification.svelte';
  import { fade, fly } from 'svelte/transition';

  let loading = $state(true);
  let loadError = $state(null);
  let dataLoaded = false;
  let prevPage = $state(null);

  const pageTitles = {
    list: 'Pipelines',
    detail: 'Pipeline Detail',
    logs: 'Pipeline Logs',
    trigger: 'Trigger Pipeline',
    manage: 'Manage Projects',
  };

  /** Update document title based on current page and project */
  function updateTitle() {
    const pg = $page;
    const proj = $activeProject;
    const base = pageTitles[pg] || 'BBC Pipeline Manager';
    const suffix = proj ? ` · ${proj.name}` : '';
    document.title = `${base}${suffix} · BBC Pipeline Manager`;
  }

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
      updateTitle();
    } catch (e) {
      console.error('Failed to load projects:', e);
      loadError = e.message || 'Failed to load projects';
    } finally {
      loading = false;
    }
  }

  // Keep the URL hash in sync with the current project and page.
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

  // Update document title whenever page or project changes
  $effect(() => {
    if ($page && !loading) updateTitle();
    // Track previous page for transition direction
    prevPage = $page;
  });

  // Load data once on mount
  $effect(() => {
    if (dataLoaded) return;
    dataLoaded = true;
    loadData();
  });

  // Register global keyboard shortcuts
  $effect(() => {
    if (loading) return;

    registerShortcut('Escape', () => {
      const pg = $page;
      if (pg === 'detail' || pg === 'logs' || pg === 'trigger') {
        navigateTo('list');
      }
    }, 'Back to pipeline list');

    registerShortcut('r', () => {
      // Refresh — dispatch a custom event that views can listen to
      window.dispatchEvent(new CustomEvent('app:refresh'));
    }, 'Refresh current view');

    registerShortcut('n', () => {
      if ($activeProject) navigateTo('trigger');
    }, 'New pipeline trigger');

    registerShortcut('g', () => {
      if ($activeProject) navigateTo('list');
    }, 'Go to pipelines');

    registerShortcut('m', () => {
      navigateTo('manage');
    }, 'Manage projects');

    return () => {};
  });

  function onGlobalKeydown(e) {
    handleShortcut(e);
  }

  // Determine transition direction based on page navigation
  function pageDirection() {
    // If going from list -> detail/logs/trigger, slide left
    // If going from detail/logs/trigger -> list, slide right
    const forward = ['detail', 'logs', 'trigger'];
    const backward = ['list'];
    if (forward.includes($page) && backward.includes(prevPage || '')) return 'left';
    if (backward.includes($page) && forward.includes(prevPage || '')) return 'right';
    return 'none';
  }
</script>

<svelte:window onkeydown={onGlobalKeydown} />

{#if loading}
  <div class="loading-screen">
    <div class="spinner"></div>
    <p>Loading pipeline manager...</p>
  </div>
{:else if loadError}
  <div class="loading-screen error">
    <div class="error-icon">⚠️</div>
    <h2>Failed to Load</h2>
    <p>{loadError}</p>
    <button class="btn btn-primary" onclick={loadData}>
      🔄 Retry
    </button>
  </div>
{:else}
  <div class="app">
    <Notification />
    <Navbar />

    <main class="main-content">
      {#if $activeProject}
        {#if $page === 'list'}
          <div in:fly={{ x: pageDirection() === 'right' ? -80 : 80, duration: 200 }} out:fade={{ duration: 120 }}>
            <PipelineList />
          </div>
        {:else if $page === 'detail'}
          <div in:fade={{ duration: 180 }} out:fade={{ duration: 100 }}>
            <PipelineDetail />
          </div>
        {:else if $page === 'logs'}
          <div in:fade={{ duration: 180 }} out:fade={{ duration: 100 }}>
            <PipelineLog />
          </div>
        {:else if $page === 'trigger'}
          <div in:fade={{ duration: 180 }} out:fade={{ duration: 100 }}>
            <PipelineTrigger />
          </div>
        {:else if $page === 'manage'}
          <div in:fly={{ x: 80, duration: 200 }} out:fade={{ duration: 120 }}>
            <ManageProjects />
          </div>
        {/if}
      {:else}
        {#if $page === 'manage'}
          <div in:fade={{ duration: 200 }}>
            <ManageProjects />
          </div>
        {:else}
          <div class="empty-state" in:fade={{ duration: 300 }}>
            <div class="empty-icon">🚀</div>
            <h2>No Projects Configured</h2>
            <p>Go to Manage Projects to add your first Bitbucket repository.</p>
            <button class="btn btn-primary" onclick={() => navigateTo('manage')}>
              ⚙ Manage Projects
            </button>
          </div>
        {/if}
      {/if}
    </main>

    <!-- Keyboard shortcuts hint bar -->
    <footer class="shortcuts-bar">
      <span class="shortcut-hint">Shortcuts:</span>
      <kbd>g</kbd> List
      <kbd>n</kbd> Trigger
      <kbd>m</kbd> Manage
      <kbd>r</kbd> Refresh
      <kbd>Esc</kbd> Back
    </footer>
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

  :global(::selection) {
    background: rgba(29, 155, 240, 0.3);
    color: #e7e9ea;
  }

  :global(:focus-visible) {
    outline: 2px solid #1d9bf0;
    outline-offset: 2px;
    border-radius: 4px;
  }

  .loading-screen {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    gap: 1rem;
    color: #71767b;
    animation: fadeIn 0.3s ease;
  }

  .loading-screen.error {
    color: #e7e9ea;
  }

  .loading-screen.error h2 {
    margin-bottom: 0.25rem;
    font-size: 1.3rem;
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

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
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
    font-family: inherit;
  }

  .btn-primary {
    background: #1d9bf0;
    color: #fff;
  }

  .btn-primary:hover {
    background: #1a8cd8;
  }

  .shortcuts-bar {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    justify-content: center;
    padding: 0.5rem 1rem;
    background: #0d1117;
    border-top: 1px solid #2f3336;
    font-size: 0.72rem;
    color: #484f58;
  }

  .shortcut-hint {
    margin-right: 0.5rem;
    font-weight: 600;
    color: #71767b;
  }

  kbd {
    display: inline-block;
    padding: 0.1rem 0.45rem;
    margin: 0 0.35rem;
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 0.68rem;
    color: #8b949e;
    background: #16181c;
    border: 1px solid #2f3336;
    border-radius: 4px;
    line-height: 1.5;
  }
</style>