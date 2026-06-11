<script>
  import { currentScreen, projects, activeProject, activeProjectId } from '../stores/appState.js';

  // Svelte 5: use $derived for computed values
  let projectCount = $derived($projects.length);
</script>

<nav class="navbar">
  <div class="nav-left">
    <span class="nav-brand" onclick={() => currentScreen.set('list')}>
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
        class:active={$currentScreen === 'list'}
        onclick={() => currentScreen.set('list')}
      >
        📋 Pipelines
      </button>
      <button
        class="nav-btn"
        class:active={$currentScreen === 'trigger'}
        onclick={() => currentScreen.set('trigger')}
      >
        ▶ Run
      </button>
    {/if}
    <button
      class="nav-btn"
      class:active={$currentScreen === 'manage'}
      onclick={() => currentScreen.set('manage')}
    >
      ⚙ Manage
    </button>
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
</style>