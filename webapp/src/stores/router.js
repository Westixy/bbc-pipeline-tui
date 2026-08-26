import { readable, get } from 'svelte/store';
import { activeProject, selectedPipeline } from './appState.js';

/**
 * Hash-based router for the BBC Pipeline Manager SPA.
 *
 * Routes:
 *   #/manage                                           – Manage projects
 *   #/running                                          – Running pipelines across all projects
 *   #/bbc/<workspace>/<repoSlug>                       – Pipeline list
 *   #/bbc/<workspace>/<repoSlug>/<buildNumber>          – Pipeline detail
 *   #/bbc/<workspace>/<repoSlug>/<buildNumber>/logs/<stepNum> – Step log viewer
 *   #/bbc/<workspace>/<repoSlug>/trigger               – Trigger pipeline form
 */

/**
 * Parse the current hash into { page, workspace, repoSlug, buildNumber, stepNum }.
 */
function parseHash() {
  const hash = window.location.hash;
  // #/manage
  if (hash === '#/manage') {
    return { page: 'manage', workspace: null, repoSlug: null, buildNumber: null, stepNum: null };
  }

  // #/running
  if (hash === '#/running') {
    return { page: 'running', workspace: null, repoSlug: null, buildNumber: null, stepNum: null };
  }

  // #/bbc/workspace/repoSlug/...
  const bbcMatch = hash.match(/^#\/bbc\/([^/]+)\/([^/]+)(?:\/(.*))?$/);
  if (!bbcMatch) return { page: '', workspace: null, repoSlug: null, buildNumber: null, stepNum: null };

  const workspace = decodeURIComponent(bbcMatch[1]);
  const repoSlug = decodeURIComponent(bbcMatch[2]);
  const rest = bbcMatch[3] || '';

  if (!rest) {
    return { page: 'list', workspace, repoSlug, buildNumber: null, stepNum: null };
  }

  // Check for trigger
  if (rest === 'trigger') {
    return { page: 'trigger', workspace, repoSlug, buildNumber: null, stepNum: null };
  }

  // Rest could be: buildNumber or buildNumber/logs/stepNum
  const parts = rest.split('/');
  const buildNumber = parts[0] || null;

  if (parts.length === 1) {
    return { page: 'detail', workspace, repoSlug, buildNumber, stepNum: null };
  }

  if (parts.length >= 3 && parts[1] === 'logs') {
    const stepNum = parseInt(parts[2], 10);
    if (!isNaN(stepNum) && stepNum >= 0) {
      return { page: 'logs', workspace, repoSlug, buildNumber, stepNum };
    }
  }

  return { page: '', workspace: null, repoSlug: null, buildNumber: null, stepNum: null };
}

/**
 * Resolve a page + params into a hash string (e.g. "#/bbc/ws/repo"), or null
 * when the target can't be determined (e.g. no active project).
 *
 *   resolveHash('manage')
 *   resolveHash('running')
 *   resolveHash('list')
 *   resolveHash('detail', buildNumber)
 *   resolveHash('logs', buildNumber, stepNum)
 *   resolveHash('trigger')
 *
 * Workspace and repoSlug are pulled from the activeProject store unless an
 * explicit `project` ({ workspace, repo_slug }) override is provided.
 */
export function resolveHash(page, buildNumber, stepNum, project) {
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
    const bnum = buildNumber !== undefined && buildNumber !== null ? buildNumber : (get(selectedPipeline)?.build_number ?? '');
    return bnum !== '' && bnum !== undefined ? `#/bbc/${ws}/${rs}/${bnum}` : `#/bbc/${ws}/${rs}`;
  }
  if (page === 'logs') {
    const bnum = buildNumber !== undefined && buildNumber !== null ? buildNumber : (get(selectedPipeline)?.build_number ?? '');
    const sn = stepNum !== undefined ? stepNum : 0;
    return bnum !== '' && bnum !== undefined ? `#/bbc/${ws}/${rs}/${bnum}/logs/${sn}` : `#/bbc/${ws}/${rs}`;
  }
  return null;
}

/**
 * Navigate to a page in the current tab.
 *   navigateTo('manage')
 *   navigateTo('running')
 *   navigateTo('list')
 *   navigateTo('detail', buildNumber)
 *   navigateTo('logs', buildNumber, stepNum)
 *   navigateTo('trigger')
 */
export function navigateTo(page, buildNumber, stepNum) {
  const hash = resolveHash(page, buildNumber, stepNum);
  if (hash) window.location.hash = hash;
}

/**
 * Open the target in a new browser tab.
 */
export function openInNewTab(page, buildNumber, stepNum, project) {
  const hash = resolveHash(page, buildNumber, stepNum, project);
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
export function navigateFromClick(event, page, buildNumber, stepNum, project) {
  if (isNewTabClick(event)) {
    event.preventDefault();
    openInNewTab(page, buildNumber, stepNum, project);
    return true;
  }
  navigateTo(page, buildNumber, stepNum);
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

export const buildNumberFromUrl = readable(parseHash().buildNumber, (set) => {
  const handler = () => set(parseHash().buildNumber);
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