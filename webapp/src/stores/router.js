import { readable, get } from 'svelte/store';
import { activeProjectId, selectedPipeline, logStepUUID } from './appState.js';

/**
 * Hash-based router for the BBC Pipeline Manager SPA.
 *
 * Routes:
 *   #/list/<n>                         – Pipeline list for project at index n
 *   #/detail/<n>/<pipelineUUID>        – Pipeline detail for a given pipeline
 *   #/logs/<n>/<pipelineUUID>/<stepUUID> – Step log viewer
 *   #/trigger/<n>                      – Trigger pipeline form
 *   #/manage                           – Manage projects (no project needed)
 */

const VALID_PAGES = new Set(['list', 'detail', 'logs', 'trigger', 'manage']);
const PROJECT_PAGES = new Set(['list', 'detail', 'logs', 'trigger']);
const PIPELINE_PAGES = new Set(['detail', 'logs']);
const STEP_PAGES = new Set(['logs']);

/**
 * Parse the current hash into { page, projectIndex, pipelineUUID, stepUUID }.
 */
function parseHash() {
  const hash = window.location.hash;
  if (!hash.startsWith('#/')) return { page: '', projectIndex: null, pipelineUUID: null, stepUUID: null };

  const parts = hash.slice(2).split('/').map(s => decodeURIComponent(s));
  const page = parts[0];
  if (!VALID_PAGES.has(page)) return { page: '', projectIndex: null, pipelineUUID: null, stepUUID: null };

  let projectIndex = null;
  if (PROJECT_PAGES.has(page) && parts[1] !== undefined) {
    const n = parseInt(parts[1], 10);
    if (!isNaN(n) && n >= 0 && Number.isInteger(n)) {
      projectIndex = n;
    }
  }

  let pipelineUUID = null;
  if (PIPELINE_PAGES.has(page) && parts[2] !== undefined && parts[2] !== '') {
    pipelineUUID = parts[2];
  }

  let stepUUID = null;
  if (STEP_PAGES.has(page) && parts[3] !== undefined && parts[3] !== '') {
    stepUUID = parts[3];
  }

  return { page, projectIndex, pipelineUUID, stepUUID };
}

/**
 * Navigate to a page.
 *   navigateTo('list')
 *   navigateTo('detail', pipelineUUID)
 *   navigateTo('logs', pipelineUUID, stepUUID)
 *   navigateTo('trigger')
 *   navigateTo('manage')
 *
 * If pipelineUUID or stepUUID are omitted, they are pulled from current store state.
 */
export function navigateTo(page, pipelineUUID, stepUUID) {
  if (!VALID_PAGES.has(page)) {
    console.error(`Invalid page: ${page}`);
    return;
  }
  if (page === 'manage') {
    window.location.hash = '#/manage';
    return;
  }

  const id = get(activeProjectId);
  const projIdx = id >= 0 ? id : 0;
  const puuid = pipelineUUID !== undefined ? pipelineUUID : (get(selectedPipeline)?.uuid || '');
  const suuid = stepUUID !== undefined ? stepUUID : get(logStepUUID);

  if (page === 'list' || page === 'trigger') {
    window.location.hash = `#/${page}/${projIdx}`;
  } else if (page === 'detail') {
    if (puuid) {
      window.location.hash = `#/${page}/${projIdx}/${puuid}`;
    } else {
      window.location.hash = `#/${page}/${projIdx}`;
    }
  } else if (page === 'logs') {
    if (puuid && suuid) {
      window.location.hash = `#/${page}/${projIdx}/${puuid}/${suuid}`;
    } else if (puuid) {
      window.location.hash = `#/${page}/${projIdx}/${puuid}`;
    } else {
      window.location.hash = `#/${page}/${projIdx}`;
    }
  }
}

/**
 * Reactive store reflecting the current hash-based page.
 */
export const page = readable(parseHash().page, (set) => {
  const handler = () => set(parseHash().page);
  window.addEventListener('hashchange', handler);
  return () => window.removeEventListener('hashchange', handler);
});

/**
 * Reactive store reflecting the project index from the URL hash.
 */
export const projectIndexFromUrl = readable(parseHash().projectIndex, (set) => {
  const handler = () => set(parseHash().projectIndex);
  window.addEventListener('hashchange', handler);
  return () => window.removeEventListener('hashchange', handler);
});

/**
 * Reactive store reflecting the pipeline UUID from the URL hash.
 */
export const pipelineUUIDFromUrl = readable(parseHash().pipelineUUID, (set) => {
  const handler = () => set(parseHash().pipelineUUID);
  window.addEventListener('hashchange', handler);
  return () => window.removeEventListener('hashchange', handler);
});

/**
 * Reactive store reflecting the step UUID from the URL hash.
 */
export const stepUUIDFromUrl = readable(parseHash().stepUUID, (set) => {
  const handler = () => set(parseHash().stepUUID);
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
