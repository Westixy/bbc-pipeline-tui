/**
 * Shared utility functions used across components.
 */

export function formatDurationCompact(seconds) {
  if (seconds <= 0) return '—';
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}m ${s}s`;
  }
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  return `${h}h ${m}m`;
}

export function formatDate(dateStr) {
  if (!dateStr) return '—';
  const d = new Date(dateStr);
  return d.toLocaleString();
}

/**
 * Returns the Bitbucket Cloud URL for a pipeline run's results page.
 */
export function bitbucketPipelineUrl(workspace, repoSlug, buildNumber) {
  if (!workspace || !repoSlug || buildNumber === undefined || buildNumber === null) return null;
  return `https://bitbucket.org/${encodeURIComponent(workspace)}/${encodeURIComponent(repoSlug)}/pipelines/results/${encodeURIComponent(buildNumber)}`;
}

/**
 * Opens the Bitbucket Cloud URL for a pipeline run in a new browser tab.
 * Returns false if the URL can't be constructed.
 */
export function openBitbucketPipeline(workspace, repoSlug, buildNumber) {
  const url = bitbucketPipelineUrl(workspace, repoSlug, buildNumber);
  if (!url) return false;
  window.open(url, '_blank', 'noopener');
  return true;
}

export function formatDuration(createdOn, completedOn, buildSecondsUsed) {
  // Primary: compute from timestamps (wall-clock time), matching TUI behavior
  if (completedOn) {
    const created = new Date(createdOn);
    const completed = new Date(completedOn);
    if (!isNaN(created) && !isNaN(completed)) {
      const diffSec = Math.round((completed - created) / 1000);
      if (diffSec > 0) return formatDurationCompact(diffSec);
    }
  }
  // Fallback to build_seconds_used
  if (buildSecondsUsed > 0) return formatDurationCompact(buildSecondsUsed);
  return '—';
}

/**
 * Returns a user-friendly display label for a pipeline/step state.
 * Uses state.result.name to differentiate COMPLETED succeeded vs failed.
 */
export function statusLabel(state) {
  if (!state) return 'unknown';
  const name = (state.name || '').toLowerCase();
  if (name === 'completed') {
    const result = (state.result?.name || '').toLowerCase();
    if (result === 'failed' || result === 'error') return 'failed';
    if (result === 'stopped') return 'stopped';
    return 'succeeded';
  }
  if (name === 'in_progress') return 'running';
  if (name === 'pending') return 'pending';
  if (name === 'stopped') return 'stopped';
  if (name === 'not_started') return 'not started';
  return state.name;
}

/**
 * Returns a CSS class name for the given pipeline/step state.
 * Considers both state.name and state.result for COMPLETED pipelines.
 */
/**
 * Resolves a pipeline step state to a canonical status string.
 * The Bitbucket API returns step states as COMPLETED with state.result.name
 * indicating SUCCESSFUL/FAILED, not as direct SUCCESSFUL/FAILED names.
 * Mirrors the TUI's resolveStepStatus logic in ui/helpers.go.
 */
export function resolveStepStatus(state) {
  if (!state) return 'UNKNOWN';
  const name = (state.name || '').toUpperCase();
  if (name === 'IN_PROGRESS') return 'IN_PROGRESS';
  if (name === 'PENDING') return 'PENDING';
  if (name === 'COMPLETED') {
    const result = (state.result?.name || '').toUpperCase();
    if (result === 'SUCCESSFUL') return 'SUCCESSFUL';
    if (result === 'FAILED' || result === 'ERROR') return 'FAILED';
    return 'STOPPED';
  }
  return name;
}

export function statusClassForState(state) {
  if (!state) return 'status-pending';
  const name = (state.name || '').toLowerCase();
  if (name === 'completed') {
    const result = (state.result?.name || '').toLowerCase();
    if (result === 'failed' || result === 'error') return 'status-error';
    if (result === 'stopped') return 'status-stopped';
    return 'status-success';
  }
  if (name === 'failed' || name === 'error') return 'status-error';
  if (name === 'in_progress') return 'status-running';
  if (name === 'pending') return 'status-pending';
  if (name === 'not_started') return 'status-pending';
  if (name === 'stopped') return 'status-stopped';
  return 'status-pending';
}
