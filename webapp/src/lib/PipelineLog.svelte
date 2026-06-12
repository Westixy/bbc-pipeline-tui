<script>
  import { onDestroy, tick } from 'svelte';
  import { activeProject, selectedPipeline, selectedSteps, logContent, logStepName, logStepUUID, showError, refreshTrigger } from '../stores/appState.js';
  import { navigateTo } from '../stores/router.js';
  import { getStepLog } from '../stores/api.js';

  let logState = $state('idle'); // 'idle' | 'loading' | 'ready' | 'error'
  let logError = $state('');
  let currentStepIndex = $state(-1);
  let autoRefreshInterval = null;
  let autoScroll = $state(true);
  let wrapLines = $state(true);
  let searchTerm = $state('');
  let searchMode = $state(false);
  let matchCount = $state(0);
  let currentMatchIdx = $state(-1);
  let logViewerEl = $state(null);

  // ── ANSI to HTML conversion ─────────────────────────────────────────────────

  /**
   * Converts ANSI escape codes in log text to HTML spans with inline styles.
   * Handles: colors (3/4-bit, 8-bit, 24-bit), bold, italic, underline, reset.
   */
  function ansiToHtml(text) {
    if (!text) return '';
    let out = '';
    let i = 0;
    let spans = []; // stack of open tag styles

    function currentStyle() {
      if (spans.length === 0) return '';
      return spans[spans.length - 1];
    }

    function pushSpan(styleAttrs) {
      spans.push(styleAttrs);
      return `<span style="${styleAttrs}">`;
    }

    function closeSpans(count) {
      let tags = '';
      for (let j = 0; j < count; j++) {
        spans.pop();
        tags += '</span>';
      }
      return tags;
    }

    function closeAllSpans() {
      const count = spans.length;
      spans = [];
      return '</span>'.repeat(count);
    }

    // Map standard 4-bit SGR colors to CSS
    const fgColors = {
      30: '#000000', 31: '#cd0000', 32: '#00cd00', 33: '#cdcd00',
      34: '#0000ee', 35: '#cd00cd', 36: '#00cdcd', 37: '#e5e5e5',
    };
    const bgColors = {
      40: '#000000', 41: '#cd0000', 42: '#00cd00', 43: '#cdcd00',
      44: '#0000ee', 45: '#cd00cd', 46: '#00cdcd', 47: '#e5e5e5',
    };
    const brightFg = {
      90: '#7f7f7f', 91: '#ff0000', 92: '#00ff00', 93: '#ffff00',
      94: '#5c5cff', 95: '#ff00ff', 96: '#00ffff', 97: '#ffffff',
    };
    const brightBg = {
      100: '#7f7f7f', 101: '#ff0000', 102: '#00ff00', 103: '#ffff00',
      104: '#5c5cff', 105: '#ff00ff', 106: '#00ffff', 107: '#ffffff',
    };

    const len = text.length;
    let styleParts = [];

    function buildStyleString() {
      const existing = currentStyle();
      const parts = [...styleParts];
      if (existing) {
        const map = {};
        existing.split(';').forEach(p => {
          const [k, v] = p.trim().split(':').map(s => s.trim());
          if (k) map[k] = v || k;
        });
        for (const p of parts) {
          const [k, v] = p.split(':');
          if (v) map[k] = v;
          else delete map[k];
        }
        return Object.entries(map).map(([k, v]) => `${k}:${v}`).join(';');
      }
      return parts.join(';');
    }

    function openStyledSpan() {
      const style = buildStyleString();
      if (style) {
        out += pushSpan(style);
      }
      styleParts = [];
    }

    function applyCode(code) {
      // Reset
      if (code === 0) {
        styleParts = [];
        if (spans.length > 0) {
          out += closeAllSpans();
        }
        return;
      }
      // Bold
      if (code === 1) { styleParts.push('font-weight:bold'); return; }
      // Dim/faint
      if (code === 2) { styleParts.push('opacity:0.6'); return; }
      // Italic
      if (code === 3) { styleParts.push('font-style:italic'); return; }
      // Underline
      if (code === 4) { styleParts.push('text-decoration:underline'); return; }
      // Blink
      if (code === 5) { styleParts.push('animation:blink 1s step-end infinite'); return; }
      // Inverse (swap fg/bg)
      if (code === 7) {
        const style = currentStyle();
        if (style) {
          const map = {};
          style.split(';').forEach(p => {
            const [k, v] = p.trim().split(':').map(s => s.trim());
            if (k) map[k] = v || k;
          });
          const fg = map['color'] || 'inherit';
          const bg = map['background-color'] || 'inherit';
          if (fg !== 'inherit') styleParts.push(`background-color:${fg}`);
          if (bg !== 'inherit') styleParts.push(`color:${bg}`);
        }
        return;
      }
      // Strikethrough
      if (code === 9) { styleParts.push('text-decoration:line-through'); return; }
      // Normal intensity (not bold, not dim)
      if (code === 22) { styleParts = styleParts.filter(p => !p.includes('font-weight') && !p.includes('opacity')); return; }
      // Not italic
      if (code === 23) { styleParts = styleParts.filter(p => !p.includes('font-style')); return; }
      // Not underline
      if (code === 24) { styleParts = styleParts.filter(p => !p.includes('text-decoration:underline')); return; }
      // Not blink
      if (code === 25) { styleParts = styleParts.filter(p => !p.includes('animation:blink')); return; }
      // Not inverse
      if (code === 27) { return; }
      // Not strikethrough
      if (code === 29) { styleParts = styleParts.filter(p => !p.includes('text-decoration:line-through')); return; }
      // Foreground colors
      if (fgColors[code]) { styleParts.push(`color:${fgColors[code]}`); return; }
      if (brightFg[code]) { styleParts.push(`color:${brightFg[code]}`); return; }
      // Background colors
      if (bgColors[code]) { styleParts.push(`background-color:${bgColors[code]}`); return; }
      if (brightBg[code]) { styleParts.push(`background-color:${brightBg[code]}`); return; }
      // 8-bit foreground (38;5;n) — handled by sequence below
      if (code === 38) { return; }
      // 8-bit background (48;5;n) — handled by sequence below
      if (code === 48) { return; }
      // Default foreground
      if (code === 39) { styleParts = styleParts.filter(p => !p.startsWith('color:')); return; }
      // Default background
      if (code === 49) { styleParts = styleParts.filter(p => !p.startsWith('background-color')); return; }
    }

    while (i < len) {
      // Escape HTML special characters before any other processing
      if (text[i] === '&') { out += '&amp;'; i++; continue; }
      if (text[i] === '<') { out += '&lt;'; i++; continue; }
      if (text[i] === '>') { out += '&gt;'; i++; continue; }

      // Check for ANSI escape sequence
      if (text[i] === '\x1b' && i + 1 < len && text[i + 1] === '[') {
        i += 2; // skip ESC [
        let seq = '';
        while (i < len && text[i] !== 'm') {
          seq += text[i];
          i++;
        }
        if (i < len) i++; // skip 'm'

        // Parse SGR parameters
        const params = seq.split(';').map(Number);

        // Open a new styled span if we have accumulated style changes
        if (styleParts.length > 0 && spans.length === 0) {
          openStyledSpan();
        }

        let p = 0;
        while (p < params.length) {
          const code = params[p];
          if (code === 38 && p + 2 < params.length && params[p + 1] === 5) {
            // 8-bit foreground: 38;5;n
            const colorIdx = params[p + 2];
            const c = xterm256Color(colorIdx);
            if (c) {
              styleParts.push(`color:${c}`);
            }
            p += 3;
          } else if (code === 48 && p + 2 < params.length && params[p + 1] === 5) {
            // 8-bit background: 48;5;n
            const colorIdx = params[p + 2];
            const c = xterm256Color(colorIdx);
            if (c) {
              styleParts.push(`background-color:${c}`);
            }
            p += 3;
          } else if (code === 38 && p + 4 < params.length && params[p + 1] === 2) {
            // 24-bit foreground: 38;2;r;g;b
            const r = params[p + 2], g = params[p + 3], b = params[p + 4];
            styleParts.push(`color:rgb(${r},${g},${b})`);
            p += 5;
          } else if (code === 48 && p + 4 < params.length && params[p + 1] === 2) {
            // 24-bit background: 48;2;r;g;b
            const r = params[p + 2], g = params[p + 3], b = params[p + 4];
            styleParts.push(`background-color:rgb(${r},${g},${b})`);
            p += 5;
          } else {
            applyCode(code);
            p++;
          }
        }

        // If we have style changes, open a new span (close old one first)
        if (styleParts.length > 0) {
          // Close previous span if any
          if (spans.length > 0) {
            out += closeSpans(1);
          }
          openStyledSpan();
        }
        continue;
      }

      out += text[i];
      i++;
    }

    // Close any remaining open spans
    out += closeAllSpans();
    return out;
  }

  /**
   * Converts xterm 256-color index to CSS hex color.
   * Covers the standard 6x6x6 cube + grayscale ramp.
   */
  function xterm256Color(idx) {
    if (idx < 16) {
      const colors = [
        '#000000','#cd0000','#00cd00','#cdcd00','#0000ee','#cd00cd','#00cdcd','#e5e5e5',
        '#7f7f7f','#ff0000','#00ff00','#ffff00','#5c5cff','#ff00ff','#00ffff','#ffffff'
      ];
      return colors[idx] || null;
    }
    if (idx >= 16 && idx <= 231) {
      // 6x6x6 color cube
      const n = idx - 16;
      const r = Math.floor(n / 36);
      const g = Math.floor((n % 36) / 6);
      const b = n % 6;
      const toHex = (v) => v === 0 ? '00' : (v * 40 + 55).toString(16);
      return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
    }
    if (idx >= 232 && idx <= 255) {
      // Grayscale ramp
      const v = (idx - 232) * 10 + 8;
      const h = v.toString(16).padStart(2, '0');
      return `#${h}${h}${h}`;
    }
    return null;
  }

  // ── Derived: colored HTML lines with line numbers ───────────────────────────

  let coloredLines = $derived.by(() => {
    if (logState !== 'ready' || !$logContent) return [];
    const raw = $logContent;
    // Split, keeping empty trailing lines
    const lines = raw.split('\n');
    return lines.map((line, idx) => ({
      num: idx + 1,
      html: ansiToHtml(line),
      raw: line,
    }));
  });

  let totalLines = $derived(coloredLines.length);

  // ── Search ─────────────────────────────────────────────────────────────────

  let searchMatches = $derived.by(() => {
    if (!searchTerm.trim() || !$logContent) return [];
    const term = searchTerm.toLowerCase();
    const matches = [];
    const lines = $logContent.split('\n');
    for (let i = 0; i < lines.length; i++) {
      if (lines[i].toLowerCase().includes(term)) {
        matches.push(i);
      }
    }
    return matches;
  });

  $effect(() => {
    matchCount = searchMatches.length;
    if (currentMatchIdx >= matchCount) currentMatchIdx = matchCount - 1;
    if (matchCount > 0 && currentMatchIdx < 0) currentMatchIdx = 0;
    if (matchCount === 0) currentMatchIdx = -1;
  });

  // ── Scroll to line ─────────────────────────────────────────────────────────

  function scrollToLine(lineNum) {
    const viewer = logViewerEl;
    if (!viewer) return;
    const lineHeight = 20.8; // 0.8rem * 1.6 line-height ≈ 20.8px at 16px base
    const target = (lineNum - 1) * lineHeight;
    viewer.scrollTo({ top: target, behavior: 'smooth' });
  }

  // ── Log loading ────────────────────────────────────────────────────────────

  async function loadLog(stepIndex) {
    if (!$activeProject || !$selectedPipeline?.uuid || !$selectedSteps[stepIndex]) return;

    const step = $selectedSteps[stepIndex];
    logState = 'loading';
    logError = '';

    try {
      const data = await getStepLog($activeProject.id, $selectedPipeline.uuid, step.uuid);
      logContent.set(data.log || 'No log content available');
      logStepName.set(step.name || `Step ${stepIndex + 1}`);
      logStepUUID.set(step.uuid);
      currentStepIndex = stepIndex;
      logState = 'ready';
      // Auto-scroll to bottom after content renders
      await tick();
      if (autoScroll && logViewerEl) {
        logViewerEl.scrollTop = logViewerEl.scrollHeight;
      }
    } catch (e) {
      logError = e.message;
      logState = 'error';
    }
  }

  function handleRefresh() {
    if ($logStepUUID) {
      const idx = $selectedSteps.findIndex(s => s.uuid === $logStepUUID);
      if (idx >= 0) loadLog(idx);
    }
  }

  function startAutoRefresh(stepIndex) {
    stopAutoRefresh();
    const step = $selectedSteps[stepIndex];
    if (!step) return;
    if (step.state?.name === 'IN_PROGRESS' || step.state?.name === 'PENDING') {
      autoRefreshInterval = setInterval(() => {
        if (logState !== 'loading') {
          loadLog(stepIndex);
        }
      }, 5000);
    }
  }

  function stopAutoRefresh() {
    if (autoRefreshInterval) {
      clearInterval(autoRefreshInterval);
      autoRefreshInterval = null;
    }
  }

  let logLoadedForStep = null;

  $effect(() => {
    if ($selectedSteps.length === 0 || !$selectedSteps[0]?.uuid) return;
    const stepId = `${$selectedSteps[0].uuid}-${$refreshTrigger}`;
    if (logLoadedForStep === stepId) return;
    logLoadedForStep = stepId;
    loadLog(0);
    startAutoRefresh(0);
  });

  onDestroy(() => {
    stopAutoRefresh();
  });

  // ── Search navigation ──────────────────────────────────────────────────────

  function nextMatch() {
    if (matchCount === 0) return;
    currentMatchIdx = (currentMatchIdx + 1) % matchCount;
    scrollToLine(searchMatches[currentMatchIdx] + 1);
  }

  function prevMatch() {
    if (matchCount === 0) return;
    currentMatchIdx = (currentMatchIdx - 1 + matchCount) % matchCount;
    scrollToLine(searchMatches[currentMatchIdx] + 1);
  }

  function clearSearch() {
    searchTerm = '';
    searchMode = false;
    currentMatchIdx = -1;
    matchCount = 0;
  }

  // ── Keyboard shortcuts ─────────────────────────────────────────────────────

  function handleKeydown(e) {
    if (searchMode) {
      if (e.key === 'Escape') { clearSearch(); e.preventDefault(); return; }
      if (e.key === 'Enter') {
        searchMode = false;
        if (matchCount > 0) {
          currentMatchIdx = 0;
          scrollToLine(searchMatches[0] + 1);
        }
        e.preventDefault();
        return;
      }
      return; // Let the input handle text
    }

    // Global shortcuts for log viewer
    if (e.key === '/' && !e.ctrlKey && !e.metaKey) {
      e.preventDefault();
      searchMode = true;
      searchTerm = '';
      setTimeout(() => {
        const input = document.querySelector('.search-input');
        if (input) input.focus();
      }, 50);
      return;
    }
    if (e.key === 'n' && !e.ctrlKey && !e.metaKey && matchCount > 0) {
      e.preventDefault();
      nextMatch();
      return;
    }
    if (e.key === 'N' && !e.ctrlKey && !e.metaKey && matchCount > 0) {
      e.preventDefault();
      prevMatch();
      return;
    }
    if (e.key === 'Escape') {
      if (matchCount > 0) {
        clearSearch();
        e.preventDefault();
      } else {
        navigateTo('detail', $selectedPipeline?.uuid);
      }
      return;
    }
  }

  // ── Copy to clipboard ──────────────────────────────────────────────────────

  let copyMsg = $state('');
  async function copyLog() {
    try {
      await navigator.clipboard.writeText($logContent || '');
      copyMsg = 'Copied!';
      setTimeout(() => { copyMsg = ''; }, 2000);
    } catch {
      copyMsg = 'Failed';
    }
  }

  // ── Track manual scroll to toggle auto-scroll ──────────────────────────────

  function onScroll(e) {
    const el = e.currentTarget;
    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 30;
    autoScroll = atBottom;
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="pipeline-log">
  <div class="log-header">
    <button class="btn btn-secondary" onclick={() => navigateTo('detail', $selectedPipeline?.uuid)}>
      ← Back to detail
    </button>
    <div class="log-title">
      <h2>Log: {$logStepName || 'Step Log'}</h2>
      <span class="log-pipe-info">Pipeline #{$selectedPipeline?.build_number || '—'}</span>
    </div>
    <div class="log-actions">
      <button class="btn btn-secondary" onclick={copyLog} title="Copy full log to clipboard">
        📋 {copyMsg || 'Copy'}
      </button>
      <button class="btn btn-secondary" onclick={() => (wrapLines = !wrapLines)} title="Toggle word wrap">
        {wrapLines ? '📏 Wrap' : '↔️ No wrap'}
      </button>
      <button class="btn btn-secondary" onclick={() => { autoScroll = true; if (logViewerEl) logViewerEl.scrollTop = logViewerEl.scrollHeight; }} title="Scroll to bottom">
        ⬇ Bottom
      </button>
      <button class="btn btn-secondary" onclick={handleRefresh}>
        🔄 Refresh
      </button>
    </div>
  </div>

  <!-- Step selector tabs -->
  {#if $selectedSteps.length > 0}
    <div class="step-tabs">
      {#each $selectedSteps as step, i}
        <button
          class="step-tab"
          class:active={$logStepUUID === step.uuid}
          onclick={() => {
            loadLog(i);
            startAutoRefresh(i);
            navigateTo('logs', $selectedPipeline?.uuid, step.uuid);
          }}
        >
          <span class="tab-icon">
            {#if step.state?.name === 'COMPLETED'}
              ✅
            {:else if step.state?.name === 'FAILED' || step.state?.name === 'ERROR'}
              ❌
            {:else if step.state?.name === 'IN_PROGRESS' || step.state?.name === 'PENDING'}
              🔄
            {:else}
              ⬜
            {/if}
          </span>
          {step.name || `Step ${i + 1}`}
        </button>
      {/each}
    </div>
  {/if}

  <!-- Search bar -->
  {#if searchMode}
    <div class="search-bar">
      <span class="search-icon">🔍</span>
      <input
        class="search-input"
        type="text"
        bind:value={searchTerm}
        placeholder="Search log… (Enter to confirm, Esc to cancel)"
        autofocus
      />
      <button class="btn btn-tiny" onclick={clearSearch}>✕</button>
    </div>
  {:else if searchTerm}
    <div class="search-bar search-bar--results">
      <span class="search-icon">🔍</span>
      <span class="search-term-text">"{searchTerm}"</span>
      {#if matchCount > 0}
        <span class="search-match-count">Match {currentMatchIdx + 1} of {matchCount}</span>
        <button class="btn btn-tiny" onclick={prevMatch} title="Previous match (Shift+N)">▲</button>
        <button class="btn btn-tiny" onclick={nextMatch} title="Next match (N)">▼</button>
      {/if}
      <button class="btn btn-tiny" onclick={clearSearch}>✕</button>
    </div>
  {/if}

  {#if logState === 'loading'}
    <div class="loading-state">
      <div class="spinner"></div>
      <p>Loading log...</p>
    </div>
  {:else if logState === 'error'}
    <div class="error-state">
      <p>❌ {logError}</p>
      <button class="btn btn-secondary" onclick={handleRefresh}>Retry</button>
    </div>
  {:else}
    <div
      class="log-viewer"
      class:wrap-lines={wrapLines}
      class:no-wrap={!wrapLines}
      bind:this={logViewerEl}
      onscroll={onScroll}
    >
      <div class="log-lines">
        {#each coloredLines as line (line.num)}
          <div class="log-line" class:search-match={searchMatches.includes(line.num - 1) && currentMatchIdx >= 0 && searchMatches[currentMatchIdx] === line.num - 1}>
            <span class="line-num">{line.num}</span>
            <span class="line-content">{@html line.html}</span>
          </div>
        {/each}
      </div>
    </div>

    <!-- Status bar -->
    <div class="status-bar">
      <span>{totalLines} lines</span>
      {#if autoScroll}
        <span class="status-badge">📌 Auto-scroll ON</span>
      {:else}
        <span class="status-badge status-badge--dim">Auto-scroll OFF</span>
      {/if}
      <span class="status-help">/ search · n next · N prev · Esc dismiss</span>
    </div>
  {/if}
</div>

<style>
  .pipeline-log {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .log-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .log-title {
    display: flex;
    align-items: baseline;
    gap: 0.75rem;
  }

  .log-title h2 {
    font-size: 1.3rem;
    font-weight: 600;
    margin: 0;
  }

  .log-pipe-info {
    color: #71767b;
    font-size: 0.85rem;
  }

  .log-actions {
    margin-left: auto;
    display: flex;
    gap: 0.35rem;
    flex-wrap: wrap;
  }

  .btn {
    padding: 0.5rem 0.85rem;
    border: none;
    border-radius: 9999px;
    font-size: 0.8rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s;
    white-space: nowrap;
  }

  .btn-secondary {
    background: #21262d;
    color: #c9d1d9;
    border: 1px solid #30363d;
  }

  .btn-secondary:hover {
    background: #30363d;
  }

  .btn-tiny {
    padding: 0.15rem 0.45rem;
    font-size: 0.7rem;
    background: #21262d;
    color: #8b949e;
    border: 1px solid #30363d;
    border-radius: 6px;
    cursor: pointer;
  }

  .btn-tiny:hover {
    background: #30363d;
    color: #c9d1d9;
  }

  .step-tabs {
    display: flex;
    gap: 0.25rem;
    overflow-x: auto;
    padding-bottom: 0.25rem;
  }

  .step-tab {
    background: #16181c;
    border: 1px solid #2f3336;
    color: #71767b;
    padding: 0.4rem 0.65rem;
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.8rem;
    white-space: nowrap;
    transition: all 0.15s;
    display: flex;
    align-items: center;
    gap: 0.3rem;
  }

  .step-tab:hover {
    background: #1a1d23;
    color: #e7e9ea;
  }

  .step-tab.active {
    background: #1d2e3e;
    border-color: #1d9bf0;
    color: #6cb6ff;
  }

  .tab-icon { font-size: 0.7rem; }

  /* ── Search bar ───────────────────────────────────────────── */

  .search-bar {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    background: #16181c;
    border: 1px solid #1d9bf0;
    border-radius: 10px;
    padding: 0.4rem 0.75rem;
  }

  .search-bar--results {
    border-color: #30363d;
  }

  .search-icon {
    font-size: 0.85rem;
  }

  .search-input {
    flex: 1;
    background: transparent;
    border: none;
    color: #e7e9ea;
    font-family: inherit;
    font-size: 0.85rem;
    outline: none;
  }

  .search-input::placeholder {
    color: #484f58;
  }

  .search-term-text {
    color: #6cb6ff;
    font-size: 0.85rem;
  }

  .search-match-count {
    color: #7ee787;
    font-size: 0.8rem;
    margin-left: 0.5rem;
  }

  /* ── Loading / Error ──────────────────────────────────────── */

  .loading-state, .error-state {
    text-align: center;
    padding: 3rem;
    background: #16181c;
    border-radius: 12px;
    border: 1px solid #2f3336;
    color: #71767b;
  }

  .spinner {
    width: 30px;
    height: 30px;
    border: 3px solid #2f3336;
    border-top: 3px solid #1d9bf0;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin: 0 auto 0.5rem;
  }

  @keyframes spin { to { transform: rotate(360deg); } }

  /* ── Log viewer (terminal-like) ───────────────────────────── */

  .log-viewer {
    background: #0d1117;
    border: 1px solid #30363d;
    border-radius: 10px;
    overflow-y: auto;
    overflow-x: hidden;
    max-height: 65vh;
    min-height: 300px;
  }

  .log-viewer.wrap-lines .line-content {
    white-space: pre-wrap;
    word-break: break-word;
  }

  .log-viewer.no-wrap {
    overflow-x: auto;
  }

  .log-viewer.no-wrap .line-content {
    white-space: pre;
  }

  .log-viewer.no-wrap .log-line {
    min-width: max-content;
  }

  .log-lines {
    padding: 0.5rem 0;
  }

  .log-line {
    display: flex;
    font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', 'Consolas', 'JetBrains Mono', 'DejaVu Sans Mono', monospace;
    font-size: 0.8rem;
    line-height: 1.6;
    min-height: 1.6em;
    color: #e6edf3;
  }

  .log-line:hover {
    background: #161b22;
  }

  .log-line.search-match {
    background: #1a2332;
    outline: 1px solid #1f6feb;
    outline-offset: -1px;
  }

  .line-num {
    flex-shrink: 0;
    width: 3.5rem;
    text-align: right;
    padding-right: 0.75rem;
    color: #484f58;
    user-select: none;
    border-right: 1px solid #21262d;
    margin-right: 0.75rem;
  }

  .line-content {
    flex: 1;
    min-width: 0;
    padding-right: 0.75rem;
  }

  /* ── Scrollbar ────────────────────────────────────────────── */

  .log-viewer::-webkit-scrollbar { width: 8px; height: 8px; }
  .log-viewer::-webkit-scrollbar-track { background: #0d1117; }
  .log-viewer::-webkit-scrollbar-thumb { background: #30363d; border-radius: 4px; }
  .log-viewer::-webkit-scrollbar-thumb:hover { background: #484f58; }

  /* ── Status bar ───────────────────────────────────────────── */

  .status-bar {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.35rem 0.75rem;
    background: #16181c;
    border: 1px solid #21262d;
    border-radius: 8px;
    font-size: 0.75rem;
    color: #8b949e;
    font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
  }

  .status-badge {
    color: #7ee787;
    font-weight: 600;
  }

  .status-badge--dim {
    color: #484f58;
  }

  .status-help {
    margin-left: auto;
    color: #484f58;
  }

  @keyframes blink {
    50% { opacity: 0; }
  }
</style>