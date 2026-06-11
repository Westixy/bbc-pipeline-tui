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
export function statusClassForState(state) {
  if (!state) return 'status-error';
  const name = (state.name || '').toLowerCase();
  if (name === 'completed') {
    const result = (state.result?.name || '').toLowerCase();
    if (result === 'failed' || result === 'error') return 'status-error';
    return 'status-success';
  }
  if (name === 'failed' || name === 'error') return 'status-error';
  if (name === 'in_progress' || name === 'pending') return 'status-running';
  if (name === 'stopped') return 'status-stopped';
  return 'status-error';
}
