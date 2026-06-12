<script>
  import { projects, activeProject, activeProjectId, refreshTrigger } from '../stores/appState.js';
  import { page, navigateTo } from '../stores/router.js';

  let projectCount = $derived($projects.length);
  let switching = $state(false);

  function selectProject(index) {
    switching = true;
    if (index !== $activeProjectId) {
      activeProjectId.set(index);
    }
    navigateTo('list');
  }
</script>

<aside class="sidebar" role="navigation" aria-label="Primary navigation">
  <!-- Activity Bar ──────────────────────────────────────── -->
  <div class="activity-bar">
    <!-- Pipelines -->
    <button
      class="activity-item"
      class:active={$page === 'list' || $page === 'detail' || $page === 'logs'}
      onclick={() => navigateTo('list')}
      title="Pipelines"
      disabled={!$activeProject}
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <line x1="8" y1="6" x2="21" y2="6"></line>
        <line x1="8" y1="12" x2="21" y2="12"></line>
        <line x1="8" y1="18" x2="21" y2="18"></line>
        <line x1="3" y1="6" x2="3.01" y2="6"></line>
        <line x1="3" y1="12" x2="3.01" y2="12"></line>
        <line x1="3" y1="18" x2="3.01" y2="18"></line>
      </svg>
    </button>

    <!-- Trigger -->
    <button
      class="activity-item"
      class:active={$page === 'trigger'}
      onclick={() => navigateTo('trigger')}
      title="Trigger Pipeline"
      disabled={!$activeProject}
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <polygon points="5 3 19 12 5 21 5 3"></polygon>
      </svg>
    </button>

    <!-- Projects -->
    <button
      class="activity-item"
      class:active={$page === 'manage'}
      onclick={() => navigateTo('manage')}
      title="Manage Projects"
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
      </svg>
    </button>

    <!-- Spacer pushes refresh to bottom -->
    <div class="activity-spacer"></div>

    {#if $activeProject}
      <button
        class="activity-item"
        onclick={() => refreshTrigger.update(n => n + 1)}
        title="Refresh"
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="23 4 23 10 17 10"></polyline>
          <polyline points="1 20 1 14 7 14"></polyline>
          <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path>
        </svg>
      </button>
    {/if}
  </div>

  <!-- Sidebar Panel ──────────────────────────────────────── -->
  <div class="sidebar-panel">
    <div class="sidebar-brand">
      <svg class="brand-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2"/>
      </svg>
      <span class="brand-text">Pipelines</span>
      <span class="brand-badge">{$projects.length}</span>
    </div>

    <div class="divider"></div>

    <!-- Project List ───────────────────────────────────── -->
    {#if projectCount > 0}
      <div class="project-section">
        <div class="section-label">Projects</div>
        <div class="project-list">
          {#each $projects as proj, i}
            <button
              class="project-item"
              class:active={i === $activeProjectId}
              onclick={() => selectProject(i)}
              title="{proj.workspace}/{proj.repo_slug}"
            >
              <div class="project-item-content">
                <span class="project-item-name">{proj.name}</span>
                <span class="project-item-meta">{proj.workspace}/{proj.repo_slug}</span>
              </div>
              {#if i === $activeProjectId}
                <span class="project-check">✓</span>
              {/if}
            </button>
          {/each}
        </div>
      </div>
    {:else}
      <div class="sidebar-empty">
        <p>No projects configured</p>
      </div>
    {/if}

    <div class="sidebar-footer">
      <button class="sidebar-footer-btn" onclick={() => navigateTo('manage')}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="3"></circle>
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
        </svg>
        Settings
      </button>
    </div>
  </div>
</aside>

<style>
  .sidebar {
    display: flex;
    height: 100%;
    flex-shrink: 0;
    user-select: none;
  }

  /* ── Activity Bar (left icon rail) ──────────────────── */
  .activity-bar {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 42px;
    min-width: 42px;
    background: #333333;
    border-right: 1px solid var(--border-default);
    padding: var(--space-2) 0;
    gap: var(--space-1);
  }

  .activity-spacer {
    flex: 1;
  }

  .activity-item {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    background: none;
    border: none;
    border-radius: var(--radius-md);
    color: var(--text-tertiary);
    cursor: pointer;
    transition: color var(--transition-fast), background var(--transition-fast);
    position: relative;
  }

  .activity-item:hover {
    color: var(--text-primary);
    background: rgba(255,255,255,0.05);
  }

  .activity-item.active {
    color: var(--accent-text);
  }

  .activity-item.active::before {
    content: '';
    position: absolute;
    left: -4px;
    top: 50%;
    transform: translateY(-50%);
    width: 2px;
    height: 20px;
    background: var(--accent);
    border-radius: 1px;
  }

  .activity-item:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }
  .activity-item:disabled:hover {
    background: none;
    color: var(--text-tertiary);
  }

  /* ── Sidebar Panel ───────────────────────────────────── */
  .sidebar-panel {
    display: flex;
    flex-direction: column;
    width: var(--sidebar-width);
    background: var(--bg-panel);
    border-right: 1px solid var(--border-default);
    overflow: hidden;
  }

  .sidebar-brand {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-5) var(--space-5) var(--space-3);
    color: var(--text-primary);
  }

  .brand-icon {
    color: var(--accent);
    flex-shrink: 0;
  }

  .brand-text {
    font-size: var(--font-size-xs);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-secondary);
    flex: 1;
  }

  .brand-badge {
    font-size: var(--font-size-xs);
    font-family: var(--font-mono);
    color: var(--text-tertiary);
    background: var(--bg-input);
    padding: 1px 6px;
    border-radius: var(--radius-sm);
  }

  /* ── Divider ─────────────────────────────────────────── */
  .divider {
    height: 1px;
    background: var(--border-subtle);
    margin: 0 var(--space-5);
    flex-shrink: 0;
  }

  /* ── Project Section ─────────────────────────────────── */
  .project-section {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: var(--space-3) 0;
  }

  .section-label {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.07em;
    color: var(--text-tertiary);
    padding: var(--space-2) var(--space-5);
    user-select: none;
  }

  .project-list {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    padding: 0 var(--space-3);
  }

  .project-item {
    display: flex;
    align-items: center;
    padding: var(--space-2) var(--space-3);
    border: none;
    border-radius: var(--radius-md);
    background: none;
    color: var(--text-secondary);
    font-size: var(--font-size-sm);
    cursor: pointer;
    transition: all var(--transition-fast);
    text-align: left;
    width: 100%;
    font-family: var(--font-ui);
    gap: var(--space-2);
  }

  .project-item:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .project-item.active {
    background: rgba(0, 122, 204, 0.15);
    color: var(--accent-text);
  }

  .project-item-content {
    display: flex;
    flex-direction: column;
    min-width: 0;
    flex: 1;
  }

  .project-item-name {
    font-weight: 500;
    font-size: var(--font-size-sm);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    color: inherit;
  }

  .project-item-meta {
    font-size: 10px;
    color: var(--text-tertiary);
    font-family: var(--font-mono);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .project-item.active .project-item-meta {
    color: rgba(79, 193, 255, 0.7);
  }

  .project-check {
    font-size: 12px;
    color: var(--accent-text);
    flex-shrink: 0;
  }

  .project-list::-webkit-scrollbar {
    width: 4px;
  }

  /* ── Sidebar Empty ──────────────────────────────────── */
  .sidebar-empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-6);
    color: var(--text-tertiary);
    font-size: var(--font-size-xs);
    text-align: center;
  }

  /* ── Sidebar Footer ──────────────────────────────────── */
  .sidebar-footer {
    border-top: 1px solid var(--border-subtle);
    padding: var(--space-3);
  }

  .sidebar-footer-btn {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    width: 100%;
    padding: var(--space-2) var(--space-3);
    border: none;
    border-radius: var(--radius-md);
    background: none;
    color: var(--text-tertiary);
    font-size: var(--font-size-xs);
    cursor: pointer;
    font-family: var(--font-ui);
    transition: all var(--transition-fast);
  }

  .sidebar-footer-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }
</style>