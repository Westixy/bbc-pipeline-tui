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
 * Navigate to a page.
 *   navigateTo('manage')
 *   navigateTo('running')
 *   navigateTo('list')
 *   navigateTo('detail', pipelineUUID)
 *   navigateTo('logs', pipelineUUID, stepNum)
 *   navigateTo('trigger')
 *
 * Workspace and repoSlug are pulled from the activeProject store.
 */
export function navigateTo(page, pipelineUUID, stepNum) {
  if (page === 'manage') {
    window.location.hash = '#/manage';
    return;
  }

  if (page === 'running') {
    window.location.hash = '#/running';
    return;
  }

  const proj = get(activeProject);
  if (!proj || !proj.workspace || !proj.repo_slug) {
    console.error('No active project to build URL');
    return;
  }
  const ws = encodeURIComponent(proj.workspace);
  const rs = encodeURIComponent(proj.repo_slug);

  if (page === 'list') {
    window.location.hash = `#/bbc/${ws}/${rs}`;
  } else if (page === 'trigger') {
    window.location.hash = `#/bbc/${ws}/${rs}/trigger`;
  } else if (page === 'detail') {
    const puuid = pipelineUUID !== undefined ? pipelineUUID : (get(selectedPipeline)?.uuid || '');
    if (puuid) {
      window.location.hash = `#/bbc/${ws}/${rs}/${puuid}`;
    } else {
      window.location.hash = `#/bbc/${ws}/${rs}`;
    }
  } else if (page === 'logs') {
    const puuid = pipelineUUID !== undefined ? pipelineUUID : (get(selectedPipeline)?.uuid || '');
    const sn = stepNum !== undefined ? stepNum : 0;
    if (puuid) {
      window.location.hash = `#/bbc/${ws}/${rs}/${puuid}/logs/${sn}`;
    } else {
      window.location.hash = `#/bbc/${ws}/${rs}`;
    }
  }
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