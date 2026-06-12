<script>
  import './app.css';
  import { projects } from './stores/appState.js';
  import { page } from './stores/router.js';
  import Navbar from './lib/Navbar.svelte';
  import PipelineList from './lib/PipelineList.svelte';
  import PipelineDetail from './lib/PipelineDetail.svelte';
  import PipelineLog from './lib/PipelineLog.svelte';
  import PipelineTrigger from './lib/PipelineTrigger.svelte';
  import ManageProjects from './lib/ManageProjects.svelte';
  import Notification from './lib/Notification.svelte';

  let currentYear = new Date().getFullYear();

  $effect(() => {
    document.title = ($projects.length > 0 ? `BBC Pipeline Manager — ${$projects[0]?.name || 'Loading…'}` : 'BBC Pipeline Manager');
  });
</script>

<div class="app-shell">
  <!-- Titlebar ────────────────────────────────────────────── -->
  <div class="titlebar" role="banner">
    <div class="titlebar-left">
      <svg class="titlebar-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2"/>
      </svg>
      <span class="titlebar-title">BBC Pipeline Manager</span>
    </div>
    <div class="titlebar-center">
      {#if $page === 'list'}
        <span class="titlebar-breadcrumb">Pipelines</span>
      {:else if $page === 'detail'}
        <span class="titlebar-breadcrumb">Pipeline Detail</span>
      {:else if $page === 'logs'}
        <span class="titlebar-breadcrumb">Pipeline Logs</span>
      {:else if $page === 'trigger'}
        <span class="titlebar-breadcrumb">Trigger Pipeline</span>
      {:else if $page === 'manage'}
        <span class="titlebar-breadcrumb">Manage Projects</span>
      {/if}
    </div>
    <div class="titlebar-right">
      <span class="titlebar-version">v1.0</span>
    </div>
  </div>

  <!-- Main Layout ─────────────────────────────────────────── -->
  <div class="main-layout">
    <Navbar />
    <main class="content-area">
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
    </main>
  </div>

  <!-- Status Bar ──────────────────────────────────────────── -->
  <div class="statusbar" role="status">
    <div class="statusbar-left">
      {#if $projects.length > 0}
        <span class="statusbar-item">
          <span class="status-dot status-dot-active"></span>
          {$projects.length} project{$projects.length !== 1 ? 's' : ''}
        </span>
      {/if}
    </div>
    <div class="statusbar-right">
      <span class="statusbar-item">© {currentYear} BBC</span>
    </div>
  </div>

  <!-- Overlays ────────────────────────────────────────────── -->
  <Notification />
</div>

<style>
  /* ── App Shell ─────────────────────────────────────────── */
  .app-shell {
    display: flex;
    flex-direction: column;
    height: 100vh;
    overflow: hidden;
    background: var(--bg-root);
  }

  /* ── Titlebar ──────────────────────────────────────────── */
  .titlebar {
    display: flex;
    align-items: center;
    height: var(--titlebar-height);
    background: #323233;
    border-bottom: 1px solid var(--border-default);
    padding: 0 var(--space-4);
    user-select: none;
    flex-shrink: 0;
    gap: var(--space-5);
    -webkit-app-region: drag;
  }

  .titlebar-left {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: var(--sidebar-width);
    -webkit-app-region: no-drag;
  }

  .titlebar-icon {
    color: var(--accent);
    flex-shrink: 0;
  }

  .titlebar-title {
    font-size: var(--font-size-xs);
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .titlebar-center {
    display: flex;
    align-items: center;
    flex: 1;
    -webkit-app-region: no-drag;
  }

  .titlebar-breadcrumb {
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    font-weight: 500;
  }

  .titlebar-right {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    -webkit-app-region: no-drag;
  }

  .titlebar-version {
    font-size: var(--font-size-xs);
    color: var(--text-tertiary);
    font-family: var(--font-mono);
  }

  /* ── Main Layout ───────────────────────────────────────── */
  .main-layout {
    display: flex;
    flex: 1;
    overflow: hidden;
    position: relative;
  }

  .content-area {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    background: var(--bg-root);
    position: relative;
  }

  /* ── Status Bar ────────────────────────────────────────── */
  .statusbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: var(--statusbar-height);
    background: var(--accent);
    padding: 0 var(--space-4);
    flex-shrink: 0;
    font-size: var(--font-size-xs);
    color: #fff;
    user-select: none;
  }

  .statusbar-left,
  .statusbar-right {
    display: flex;
    align-items: center;
    gap: var(--space-5);
  }

  .statusbar-item {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--font-size-xs);
    font-weight: 500;
  }

  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: rgba(255,255,255,0.4);
  }

  .status-dot-active {
    background: #fff;
    box-shadow: 0 0 4px rgba(255,255,255,0.5);
  }
</style>