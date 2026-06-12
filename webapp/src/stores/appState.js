import { writable, derived } from 'svelte/store';

// Current project index
export const activeProjectId = writable(0);

// List of projects (from /api/projects)
export const projects = writable([]);

// Pipeline list state
export const pipelines = writable([]);
export const pipelinesNext = writable('');
export const listState = writable('loading'); // 'loading' | 'ready' | 'error'
export const listError = writable('');
export const listFilter = writable('');
export const listSort = writable('-created_on');

// Pipeline detail
export const selectedPipeline = writable(null);
export const selectedSteps = writable([]);
export const selectedVariables = writable([]);
export const selectedLogVariables = writable([]);
export const detailState = writable('idle');

// Log state
export const logContent = writable('');
export const logStepName = writable('');
export const logPipeUUID = writable('');
export const logStepUUID = writable('');

// Trigger form
export const triggerState = writable('idle');
export const triggerError = writable('');
export const triggerSuccess = writable('');
export const triggerPreTarget = writable('');
export const triggerPreSelector = writable(null);
export const triggerPreVars = writable([]);

// Manage projects
export const workspaceRepos = writable([]);
export const workspaceReposState = writable('idle'); // 'idle' | 'loading' | 'ready' | 'error'
export const workspaceReposError = writable('');

// Active project (derived)
export const activeProject = derived([projects, activeProjectId], ([$projects, $activeProjectId]) => {
  return $projects[$activeProjectId] || null;
});

// Global refresh trigger (increment to refresh current page)
export const refreshTrigger = writable(0);

// Error notification
export const notification = writable(null); // { type: 'error'|'success', message: '' }

export function showError(message) {
  notification.set({ type: 'error', message });
  setTimeout(() => notification.set(null), 5000);
}

export function showSuccess(message) {
  notification.set({ type: 'success', message });
  setTimeout(() => notification.set(null), 3000);
}