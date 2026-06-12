<script>
  import { projects, activeProject, activeProjectId, refreshTrigger } from '../stores/appState.js';
  import { page, navigateTo } from '../stores/router.js';

  // Svelte 5: use $derived for computed values
  let projectCount = $derived($projects.length);
</script>

<nav class="navbar">
  <div class="nav-left">
    <span class="nav-brand" onclick={() => navigateTo('list')}>
      🚀 BBC Pipeline Manager
    </span>
    {#if projectCount > 0}
      <span class="nav-sep">/</span>
      <span class="nav-project">{$activeProject?.name || 'No project'}</span>
    {/if}
  </div>
  <div class="nav-right">
    {#if $activeProject}
      <button
        class="nav-btn"
        class:active={$page === 'list'}
        onclick={() => navigateTo('list')}
      >
        📋 Pipelines
      </button>
    {/if}
    <button
      class="nav-btn"
      class:active={$page === 'manage'}
      onclick={() => navigateTo('manage')}
    >
      ⚙ Manage
    </button>
    {#if $activeProject}
      <button
        class="nav-btn nav-btn-refresh"
        title="Refresh current view"
        onclick={() => refreshTrigger.update(n => n + 1)}
      >
        🔄 Refresh
      </button>
    {/if}
  </div>
</nav>

<style>
  .navbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem 1.5rem;
    background: #16181c;
    border-bottom: 1px solid #2f3336;
    position: sticky;
    top: 0;
    z-index: 100;
  }

  .nav-left {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .nav-brand {
    font-weight: 700;
    font-size: 1.1rem;
    cursor: pointer;
    color: #e7e9ea;
  }

  .nav-brand:hover {
    color: #1d9bf0;
  }

  .nav-sep {
    color: #71767b;
  }

  .nav-project {
    color: #1d9bf0;
    font-weight: 500;
    font-size: 0.95rem;
  }

  .nav-right {
    display: flex;
    gap: 0.25rem;
  }

  .nav-btn {
    background: none;
    border: none;
    color: #71767b;
    padding: 0.5rem 1rem;
    border-radius: 9999px;
    cursor: pointer;
    font-size: 0.9rem;
    font-weight: 500;
    transition: all 0.15s;
  }

  .nav-btn:hover {
    background: #2f3336;
    color: #e7e9ea;
  }

  .nav-btn.active {
    background: #2f3336;
    color: #1d9bf0;
  }

  .nav-btn-refresh {
    margin-left: 0.5rem;
    border-left: 1px solid #2f3336;
    padding-left: 1.25rem;
  }
</style>
