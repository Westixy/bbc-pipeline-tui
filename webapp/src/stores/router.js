import { readable, get } from 'svelte/store';
import { activeProjectId } from './appState.js';

/**
 * Hash-based router for the BBC Pipeline Manager SPA.
 *
 * Routes:
 *   #/list/<n>   – Pipeline list for project at index n
 *   #/detail/<n> – Pipeline detail for project at index n
 *   #/logs/<n>   – Step log viewer for project at index n
 *   #/trigger/<n>– Trigger pipeline form for project at index n
 *   #/manage     – Manage projects (no project needed)
 */

const VALID_PAGES = new Set(['list', 'detail', 'logs', 'trigger', 'manage']);
const PROJECT_PAGES = new Set(['list', 'detail', 'logs', 'trigger']);

/**
 * Parse the current hash into { page, projectIndex }.
 * projectIndex is null when not present or not applicable (manage).
 */
function parseHash() {
  const hash = window.location.hash;
  if (!hash.startsWith('#/')) return { page: '', projectIndex: null };

  const parts = hash.slice(2).split('/');
  const page = parts[0];
  if (!VALID_PAGES.has(page)) return { page: '', projectIndex: null };

  let projectIndex = null;
  if (PROJECT_PAGES.has(page) && parts[1] !== undefined) {
    const n = parseInt(parts[1], 10);
    if (!isNaN(n) && n >= 0 && Number.isInteger(n)) {
      projectIndex = n;
    }
  }

  return { page, projectIndex };
}

/**
 * Navigate to a page, automatically including the current activeProjectId
 * for project-dependent routes.
 */
export function navigateTo(page) {
  if (!VALID_PAGES.has(page)) {
    console.error(`Invalid page: ${page}`);
    return;
  }
  if (PROJECT_PAGES.has(page)) {
    const id = get(activeProjectId);
    window.location.hash = `#/${page}/${id >= 0 ? id : 0}`;
  } else {
    window.location.hash = `#/${page}`;
  }
}

/**
 * Reactive store reflecting the current hash-based page.
 * Empty string means no hash is set yet (initial load).
 */
export const page = readable(parseHash().page, (set) => {
  const handler = () => set(parseHash().page);
  window.addEventListener('hashchange', handler);
  return () => window.removeEventListener('hashchange', handler);
});

/**
 * Reactive store reflecting the project index from the URL hash.
 * null means no project index in the URL.
 */
export const projectIndexFromUrl = readable(parseHash().projectIndex, (set) => {
  const handler = () => set(parseHash().projectIndex);
  window.addEventListener('hashchange', handler);
  return () => window.removeEventListener('hashchange', handler);
});

/**
 * Initialize the default route.
 * Call once after projects are loaded in App.svelte.
 * If no hash is present, navigate to 'manage' when no projects exist, otherwise 'list'.
 */
export function initRoute(hasProjects) {
  if (parseHash().page === '') {
    navigateTo(hasProjects ? 'list' : 'manage');
  }
}
