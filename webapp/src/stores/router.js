import { readable, writable } from 'svelte/store';

/**
 * Hash-based router for the BBC Pipeline Manager SPA.
 *
 * Routes:
 *   #/list    – Pipeline list (requires active project)
 *   #/detail  – Pipeline detail (requires selected pipeline)
 *   #/logs    – Step log viewer
 *   #/trigger – Trigger pipeline form
 *   #/manage  – Manage projects (no project needed)
 */

const VALID_PAGES = new Set(['list', 'detail', 'logs', 'trigger', 'manage']);

function getPageFromHash() {
  const hash = window.location.hash;
  if (hash.startsWith('#/')) {
    const p = hash.slice(2);
    if (VALID_PAGES.has(p)) return p;
  }
  return '';
}

export function navigateTo(page) {
  if (!VALID_PAGES.has(page)) {
    console.error(`Invalid page: ${page}`);
    return;
  }
  window.location.hash = `#/${page}`;
}

/**
 * Reactive store that reflects the current hash-based page.
 * Empty string means no hash is set yet (initial load).
 */
export const page = readable(getPageFromHash(), (set) => {
  const handler = () => set(getPageFromHash());
  window.addEventListener('hashchange', handler);
  return () => window.removeEventListener('hashchange', handler);
});

/**
 * Initialize the default route.
 * Call once after projects are loaded in App.svelte.
 * If no hash is present, navigate to 'manage' when no projects exist, otherwise 'list'.
 */
export function initRoute(hasProjects) {
  if (getPageFromHash() === '') {
    navigateTo(hasProjects ? 'list' : 'manage');
  }
}