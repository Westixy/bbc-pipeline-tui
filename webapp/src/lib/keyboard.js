/**
 * Global keyboard shortcuts manager.
 * Registers shortcuts that work anywhere in the app.
 */

const shortcuts = new Map(); // key -> { handler, description }

/**
 * Register a global keyboard shortcut.
 * @param {string} key - e.g. 'Escape', 'r', '/', 'n'
 * @param {() => void} handler
 * @param {string} description - for future help dialog
 */
export function registerShortcut(key, handler, description = '') {
  shortcuts.set(key, { handler, description });
}

/**
 * Unregister a shortcut.
 */
export function unregisterShortcut(key) {
  shortcuts.delete(key);
}

/**
 * Returns the handler for a key if registered, ignores shortcuts when
 * an input/textarea/select is focused (except Escape, which always works).
 */
export function handleShortcut(e) {
  const tag = document.activeElement?.tagName?.toLowerCase();
  const isInput = tag === 'input' || tag === 'textarea' || tag === 'select' || document.activeElement?.isContentEditable;

  // Escape always works
  if (e.key === 'Escape') {
    const s = shortcuts.get('Escape');
    if (s) { e.preventDefault(); s.handler(); return true; }
  }

  // Don't fire other shortcuts when typing in inputs
  if (isInput) return false;

  const s = shortcuts.get(e.key);
  if (s) {
    e.preventDefault();
    s.handler();
    return true;
  }
  return false;
}