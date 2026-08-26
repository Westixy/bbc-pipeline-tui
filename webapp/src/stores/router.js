import { readable, get } from 'svelte/store';
import { activeProject, selectedPipeline } from './appState.js';

/**
 * Hash-based router for the BBC Pipeline Manager SPA.
 *
 * Routes:
 *   #/manage                                           – Manage projects
 *   #/running                                          – Running pipelines across all projects
 *   #/bbc/<workspace>/<repoSlug>                       – Pipeline list
 *   #/bbc/<workspace>/<repoSlug>/<pipelineUUID>         – Pipeline detail
 *   #/bbc/<workspace>/<repoSlug>/<pipelineUUID>/logs/<stepNum> – Step log viewer
 *   #/bbc/<workspace>/<repoSlug>/trigger               – Trigger pipeline form
 */

/**
 * Parse the current hash into { page, workspace, repoSlug, pipelineUUID, stepNum }.
 */
function parseHash() {
  const hash = window.location.hash;
  // #/manage
  if (hash === '#/manage') {
    return { page: 'manage', workspace: null, repoSlug: null, pipelineUUID: null, stepNum: null };
  }

  // #/running
  if (hash === '#/running') {
    return { page: 'running', workspace: null, repoSlug: null, pipelineUUID: null, stepNum: null };
  }

  // #/bbc/workspace/repoSlug/...
  const bbcMatch = hash.match(/^#\/bbc\/([^/]+)\/([^/]+)(?:\/(.*))?$/);
  if (!bbcMatch) return { page: '', workspace: null, repoSlug: null, pipelineUUID: null, stepNum: null };

  const workspace = decodeURIComponent(bbcMatch[1]);
  const repoSlug = decodeURIComponent(bbcMatch[2]);
  const rest = bbcMatch[3] || '';

  if (!rest) {
    return { page: 'list', workspace, repoSlug, pipelineUUID: null, stepNum: null };
  }

  // Check for trigger
  if (rest === 'trigger') {
    return { page: 'trigger', workspace, repoSlug, pipelineUUID: null, stepNum: null };
  }

  // Rest could be: pipelineUUID or pipelineUUID/logs/stepNum
  const parts = rest.split('/');
  const pipelineUUID = parts[0] || null;

  if (parts.length === 1) {
    return { page: 'detail', workspace, repoSlug, pipelineUUID, stepNum: null };
  }

  if (parts.length >= 3 && parts[1] === 'logs') {
    const stepNum = parseInt(parts[2], 10);
    if (!isNaN(stepNum) && stepNum >= 0) {
      return { page: 'logs', workspace, repoSlug, pipelineUUID, stepNum };
    }
  }

  return { page: '', workspace: null, repoSlug: null, pipelineUUID: null, stepNum: null };
}

/**
 * Resolve a page + params into a hash string (e.g. "#/bbc/ws/repo"), or null
 * when the target can't be determined (e.g. no active project).
 *
 *   resolveHash('manage')
 *   resolveHash('running')
 *   resolveHash('list')
 *   resolveHash('detail', pipelineUUID)
 *   resolveHash('logs', pipelineUUID, stepNum)
 *   resolveHash('trigger')
 *
 * Workspace and repoSlug are pulled from the activeProject store unless an
 * explicit `project` ({ workspace, repo_slug }) override is provided.
 */
export function resolveHash(page, pipelineUUID, stepNum, project) {
  if (page === 'manage') return '#/manage';
  if (page === 'running') return '#/running';

  const proj = project || get(activeProject);
  if (!proj || !proj.workspace || !proj.repo_slug) {
    console.error('No active project to build URL');
    return null;
  }
  const ws = encodeURIComponent(proj.workspace);
  const rs = encodeURIComponent(proj.repo_slug);

  if (page === 'list') return `#/bbc/${ws}/${rs}`;
  if (page === 'trigger') return `#/bbc/${ws}/${rs}/trigger`;
  if (page === 'detail') {
    const puuid = pipelineUUID !== undefined ? pipelineUUID : (get(selectedPipeline)?.uuid || '');
    return puuid ? `#/bbc/${ws}/${rs}/${puuid}` : `#/bbc/${ws}/${rs}`;
  }
  if (page === 'logs') {
    const puuid = pipelineUUID !== undefined ? pipelineUUID : (get(selectedPipeline)?.uuid || '');
    const sn = stepNum !== undefined ? stepNum : 0;
    return puuid ? `#/bbc/${ws}/${rs}/${puuid}/logs/${sn}` : `#/bbc/${ws}/${rs}`;
  }
  return null;
}

/**
 * Navigate to a page in the current tab.
 *   navigateTo('manage')
 *   navigateTo('running')
 *   navigateTo('list')
 *   navigateTo('detail', pipelineUUID)
 *   navigateTo('logs', pipelineUUID, stepNum)
 *   navigateTo('trigger')
 */
export function navigateTo(page, pipelineUUID, stepNum) {
  const hash = resolveHash(page, pipelineUUID, stepNum);
  if (hash) window.location.hash = hash;
}

/**
 * Open the target in a new browser tab.
 */
export function openInNewTab(page, pipelineUUID, stepNum, project) {
  const hash = resolveHash(page, pipelineUUID, stepNum, project);
  if (!hash) return;
  const base = window.location.href.split('#')[0];
  window.open(base + hash, '_blank', 'noopener');
}

/**
 * Whether a click should open the target in a new tab rather than navigating
 * in place: Ctrl/Cmd/Shift + click, or a middle-click.
 */
export function isNewTabClick(event) {
  if (!event) return false;
  return event.ctrlKey || event.metaKey || event.shiftKey || event.button === 1;
}

/**
 * Click handler for navigation "links". Honors Ctrl/Cmd/Shift + click and
 * middle-click by opening the target in a new tab instead of navigating in
 * place. Use this for simple handlers where no pre-navigation setup (e.g.
 * switching the active project or prefilling stores) is required.
 */
export function navigateFromClick(event, page, pipelineUUID, stepNum, project) {
  if (isNewTabClick(event)) {
    event.preventDefault();
    openInNewTab(page, pipelineUUID, stepNum, project);
    return true;
  }
  navigateTo(page, pipelineUUID, stepNum);
  return false;
}

// ── Reactive stores derived from URL hash ────────────────────────────────────

export const page = readable(parseHash().page, (set) => {
  const handler = () => set(parseHash().page);
  window.addEventListener('hashchange', handler);
  return () => window.removeEventListener('hashchange', handler);
});

export const workspaceFromUrl = readable(parseHash().workspace, (set) => {
  const handler = () => set(parseHash().workspace);
  window.addEventListener('hashchange', handler);
  return () => window.removeEventListener('hashchange', handler);
});

export const repoSlugFromUrl = readable(parseHash().repoSlug, (set) => {
  const handler = () => set(parseHash().repoSlug);
  window.addEventListener('hashchange', handler);
  return () => window.removeEventListener('hashchange', handler);
});

export const pipelineUUIDFromUrl = readable(parseHash().pipelineUUID, (set) => {
  const handler = () => set(parseHash().pipelineUUID);
  window.addEventListener('hashchange', handler);
  return () => window.removeEventListener('hashchange', handler);
});

export const stepNumFromUrl = readable(parseHash().stepNum, (set) => {
  const handler = () => set(parseHash().stepNum);
  window.addEventListener('hashchange', handler);
  return () => window.removeEventListener('hashchange', handler);
});

/**
 * Initialize the default route.
 * Call once after projects are loaded in App.svelte.
 */
export function initRoute(hasProjects) {
  if (parseHash().page === '') {
    navigateTo(hasProjects ? 'list' : 'manage');
  }
}