<script>
  import { projects, activeProject, activeProjectId, refreshTrigger } from '../stores/appState.js';
  import { page, navigateTo } from '../stores/router.js';

  let projectCount = $derived($projects.length);
  let dropdownOpen = $state(false);

  function toggleDropdown() {
    dropdownOpen = !dropdownOpen;
  }

  function selectProject(index) {
    dropdownOpen = false;
    if (index === $activeProjectId) return;
    activeProjectId.set(index);
    // Navigate to list for the newly selected project
    navigateTo('list');
  }

  function closeDropdown() {
    dropdownOpen = false;
  }

  // Close dropdown when clicking outside
  $effect(() => {
    if (!dropdownOpen) return;
    function handleClick(e) {
      if (!e.target.closest('.project-dropdown')) {
        dropdownOpen = false;
      }
    }
    document.addEventListener('click', handleClick);
    return () => document.removeEventListener('click', handleClick);
  });
</script>

<nav class="navbar">
  <div class="nav-left">
    <button class="nav-brand" onclick={() => navigateTo('list')}>
      🚀 BBC Pipeline Manager
    </button>
    {#if projectCount > 0}
      <span class="nav-sep">/</span>
      <div class="project-dropdown">
        <button class="nav-project-btn" onclick={toggleDropdown}>
          {$activeProject?.name || 'No project'}
          <span class="dropdown-arrow" class:open={dropdownOpen}>▾</span>
        </button>
        {#if dropdownOpen}
          <div class="dropdown-menu">
            {#each $projects as proj, i}
              <button
                class="dropdown-item"
                class:active={i === $activeProjectId}
                onclick={() => selectProject(i)}
              >
                <span class="item-name">{proj.name}</span>
                <span class="item-meta">{proj.workspace}/{proj.repo_slug}</span>
              </button>
            {/each}
          </div>
        {/if}
      </div>
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
    background: none;
    border: none;
    font-family: inherit;
  }

  .nav-brand:hover {
    color: #1d9bf0;
  }

  .nav-sep {
    color: #71767b;
  }

  .project-dropdown {
    position: relative;
  }

  .nav-project-btn {
    background: none;
    border: 1px solid transparent;
    color: #1d9bf0;
    font-weight: 500;
    font-size: 0.95rem;
    cursor: pointer;
    padding: 0.25rem 0.5rem;
    border-radius: 6px;
    display: flex;
    align-items: center;
    gap: 0.35rem;
    transition: all 0.15s;
    font-family: inherit;
  }

  .nav-project-btn:hover {
    background: #2f3336;
    border-color: #2f3336;
  }

  .dropdown-arrow {
    font-size: 0.7rem;
    transition: transform 0.2s;
  }

  .dropdown-arrow.open {
    transform: rotate(180deg);
  }

  .dropdown-menu {
    position: absolute;
    top: 100%;
    left: 0;
    margin-top: 0.4rem;
    background: #16181c;
    border: 1px solid #2f3336;
    border-radius: 10px;
    min-width: 280px;
    max-height: 320px;
    overflow-y: auto;
    z-index: 200;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
    padding: 0.35rem;
  }

  .dropdown-item {
    display: flex;
    flex-direction: column;
    width: 100%;
    background: none;
    border: none;
    color: #e7e9ea;
    padding: 0.5rem 0.75rem;
    border-radius: 6px;
    cursor: pointer;
    text-align: left;
    transition: background 0.12s;
    font-family: inherit;
  }

  .dropdown-item:hover {
    background: #2f3336;
  }

  .dropdown-item.active {
    background: #1d2e3e;
    border: 1px solid #1d9bf0;
    padding: calc(0.5rem - 1px) calc(0.75rem - 1px);
  }

  .item-name {
    font-weight: 600;
    font-size: 0.9rem;
  }

  .item-meta {
    font-size: 0.75rem;
    color: #71767b;
    font-family: 'SF Mono', 'Fira Code', monospace;
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
