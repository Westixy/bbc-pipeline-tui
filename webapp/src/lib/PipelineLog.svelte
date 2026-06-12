<script>
  import { onDestroy, tick } from 'svelte';
  import { activeProject, selectedPipeline, selectedSteps, logContent, logStepName, logStepUUID, showError, refreshTrigger } from '../stores/appState.js';
  import { navigateTo, stepNumFromUrl } from '../stores/router.js';
  import { getStepLog } from '../stores/api.js';

  let logState = $state('idle');
  let logError = $state('');
  let currentStepIndex = $state(-1);
  let autoRefreshInterval = $state(null);
  let autoScroll = $state(true);
  let wrapLines = $state(true);
  let searchTerm = $state('');
  let matchCount = $state(0);
  let currentMatchIdx = $state(-1);
  let logViewerEl = $state(null);
  let searchInputEl = $state(null);
  let stepDropdownOpen = $state(false);
  let elapsedInterval = $state(null);
  let elapsed = $state('');
  let lastRefreshed = $state(null);

  function updateElapsed() {
    const step = $selectedSteps[currentStepIndex];
    if (!step?.started_on) { elapsed = ''; return; }
    const started = new Date(step.started_on);
    const now = new Date();
    const diff = Math.floor((now - started) / 1000);
    if (diff < 60) elapsed = `${diff}s`;
    else if (diff < 3600) elapsed = `${Math.floor(diff / 60)}m ${diff % 60}s`;
    else {
      const h = Math.floor(diff / 3600);
      const m = Math.floor((diff % 3600) / 60);
      elapsed = `${h}h ${m}m`;
    }
  }

  function ansiToHtml(text) {
    if (!text) return '';
    let out = '';
    let i = 0;
    let spans = [];
    function currentStyle() { return spans.length === 0 ? '' : spans[spans.length - 1]; }
    function pushSpan(styleAttrs) { spans.push(styleAttrs); return `<span style="${styleAttrs}">`; }
    function closeSpans(count) { for (let j = 0; j < count; j++) spans.pop(); return '</span>'.repeat(count); }
    function closeAllSpans() { const c = spans.length; spans = []; return '</span>'.repeat(c); }
    const fg = {30:'#000',31:'#cd0000',32:'#00cd00',33:'#cdcd00',34:'#0000ee',35:'#cd00cd',36:'#00cdcd',37:'#e5e5e5'};
    const bg = {40:'#000',41:'#cd0000',42:'#00cd00',43:'#cdcd00',44:'#0000ee',45:'#cd00cd',46:'#00cdcd',47:'#e5e5e5'};
    const bfg = {90:'#7f7f7f',91:'#f00',92:'#0f0',93:'#ff0',94:'#5c5cff',95:'#f0f',96:'#0ff',97:'#fff'};
    const bbg = {100:'#7f7f7f',101:'#f00',102:'#0f0',103:'#ff0',104:'#5c5cff',105:'#f0f',106:'#0ff',107:'#fff'};
    let sp = [];
    function ss() { const e=currentStyle(); const p=[...sp]; if(e){const m={};e.split(';').forEach(x=>{const[k,v]=x.split(':').map(s=>s.trim());if(k)m[k]=v||k}); for(const x of p){const[k,v]=x.split(':');if(v)m[k]=v;else delete m[k];} return Object.entries(m).map(([k,v])=>`${k}:${v}`).join(';');} return p.join(';'); }
    function os() { const s=ss(); if(s) out+=pushSpan(s); sp=[]; }
    function ac(code) {
      if(code===0){sp=[];if(spans.length>0)out+=closeAllSpans();return;}
      if(code===1){sp.push('font-weight:bold');return;} if(code===2){sp.push('opacity:0.6');return;} if(code===3){sp.push('font-style:italic');return;}
      if(code===4){sp.push('text-decoration:underline');return;} if(code===5){sp.push('animation:blink 1s step-end infinite');return;}
      if(code===7){const s=currentStyle();if(s){const m={};s.split(';').forEach(x=>{const[k,v]=x.split(':').map(y=>y.trim());if(k)m[k]=v||k});const f=m['color']||'inherit',b=m['background-color']||'inherit';if(f!=='inherit')sp.push(`background-color:${f}`);else sp.push('background-color:inherit');if(b!=='inherit')sp.push(`color:${b}`);else sp.push('color:inherit');}return;}
      if(code===9){sp.push('text-decoration:line-through');return;} if(code===22){sp=sp.filter(p=>!p.includes('font-weight')&&!p.includes('opacity'));return;}
      if(code===23){sp=sp.filter(p=>!p.includes('font-style'));return;} if(code===24){sp=sp.filter(p=>!p.includes('text-decoration:underline'));return;}
      if(code===25){sp=sp.filter(p=>!p.includes('animation:blink'));return;} if(code===27||code===29)return; if(fg[code]){sp.push(`color:${fg[code]}`);return;}
      if(bfg[code]){sp.push(`color:${bfg[code]}`);return;} if(bg[code]){sp.push(`background-color:${bg[code]}`);return;} if(bbg[code]){sp.push(`background-color:${bbg[code]}`);return;}
      if(code===38||code===48)return; if(code===39){sp=sp.filter(p=>!p.startsWith('color:'));return;} if(code===49){sp=sp.filter(p=>!p.startsWith('background-color'));return;}
    }
    const len=text.length;
    while(i<len){if(text[i]==='&'){out+='&';i++;continue;}if(text[i]==='<'){out+='<';i++;continue;}if(text[i]==='>'){out+='>';i++;continue;}
      if(text[i]==='\x1b'&&i+1<len&&text[i+1]==='['){i+=2;let seq='';while(i<len&&text[i]!=='m'){seq+=text[i];i++;}if(i<len)i++;
        const params=seq.split(';').map(Number);if(sp.length>0&&spans.length===0)os();let p=0;
        while(p<params.length){const c=params[p];
          if(c===38&&p+2<params.length&&params[p+1]===5){const ci=params[p+2];const col=xterm256Color(ci);if(col)sp.push(`color:${col}`);p+=3;}
          else if(c===48&&p+2<params.length&&params[p+1]===5){const ci=params[p+2];const col=xterm256Color(ci);if(col)sp.push(`background-color:${col}`);p+=3;}
          else if(c===38&&p+4<params.length&&params[p+1]===2){sp.push(`color:rgb(${params[p+2]},${params[p+3]},${params[p+4]})`);p+=5;}
          else if(c===48&&p+4<params.length&&params[p+1]===2){sp.push(`background-color:rgb(${params[p+2]},${params[p+3]},${params[p+4]})`);p+=5;}
          else{ac(c);p++;}}
        if(sp.length>0){if(spans.length>0)out+=closeSpans(1);os();}continue;}out+=text[i];i++;}
    out+=closeAllSpans();return out;
  }

  function xterm256Color(idx) {
    if(idx<16){return ['#000','#cd0000','#00cd00','#cdcd00','#0000ee','#cd00cd','#00cdcd','#e5e5e5','#7f7f7f','#f00','#0f0','#ff0','#5c5cff','#f0f','#0ff','#fff'][idx]||null;}
    if(idx>=16&&idx<=231){const n=idx-16,r=Math.floor(n/36),g=Math.floor((n%36)/6),b=n%6;const th=v=>v===0?'00':(v*40+55).toString(16);return`#${th(r)}${th(g)}${th(b)}`;}
    if(idx>=232&&idx<=255){const v=(idx-232)*10+8,h=v.toString(16).padStart(2,'0');return`#${h}${h}${h}`;}return null;
  }

  let coloredLines = $derived.by(() => {
    if (logState !== 'ready' || !$logContent) return [];
    const lines = $logContent.split('\n');
    return lines.map((line, idx) => ({ num: idx + 1, html: ansiToHtml(line), raw: line }));
  });

  let searchMatches = $derived.by(() => {
    if (!searchTerm.trim() || !$logContent) return [];
    const term = searchTerm.toLowerCase();
    const matches = [];
    const lines = $logContent.split('\n');
    for (let i = 0; i < lines.length; i++) if (lines[i].toLowerCase().includes(term)) matches.push(i);
    return matches;
  });

  $effect(() => { matchCount = searchMatches.length; if (currentMatchIdx >= matchCount) currentMatchIdx = matchCount - 1; if (matchCount > 0 && currentMatchIdx < 0) currentMatchIdx = 0; if (matchCount === 0) currentMatchIdx = -1; });

  function scrollToLine(lineNum) {
    if (!logViewerEl) return;
    const lineHeight = 20.8;
    logViewerEl.scrollTo({ top: (lineNum - 1) * lineHeight, behavior: 'smooth' });
  }

  let latestRequestId = 0; let consecutiveErrorCount = 0; const MAX_CONSECUTIVE_ERRORS = 3; let logLoadedForStep = null;

  async function loadLog(stepIndex) {
    if (!$activeProject || !$selectedPipeline?.uuid || !$selectedSteps[stepIndex]) return;
    const step = $selectedSteps[stepIndex];
    const requestId = ++latestRequestId;
    logState = 'loading'; logError = '';
    try {
      const data = await getStepLog($activeProject.id, $selectedPipeline.uuid, step.uuid);
      if (requestId !== latestRequestId) return;
      logContent.set(data.log || 'No log content available');
      logStepName.set(step.name || `Step ${stepIndex + 1}`);
      logStepUUID.set(step.uuid);
      currentStepIndex = stepIndex;
      logState = 'ready';
      lastRefreshed = new Date();
      consecutiveErrorCount = 0;
      updateElapsed();
      await tick();
      if (autoScroll && logViewerEl) logViewerEl.scrollTop = logViewerEl.scrollHeight;
    } catch (e) {
      if (requestId !== latestRequestId) return;
      logError = e.message;
      logState = 'error';
      consecutiveErrorCount++;
    }
  }

  function handleRefresh() { if ($logStepUUID) { const idx = $selectedSteps.findIndex(s => s.uuid === $logStepUUID); if (idx >= 0) loadLog(idx); } }

  function startAutoRefresh(stepIndex) {
    stopAutoRefresh();
    const step = $selectedSteps[stepIndex];
    if (!step) return;
    if (step.state?.name === 'IN_PROGRESS' || step.state?.name === 'PENDING') {
      elapsedInterval = setInterval(updateElapsed, 1000);
      autoRefreshInterval = setInterval(() => {
        if (consecutiveErrorCount >= MAX_CONSECUTIVE_ERRORS) { stopAutoRefresh(); return; }
        if (logState !== 'loading') loadLog(stepIndex);
      }, 5000);
    }
  }

  function stopAutoRefresh() {
    if (autoRefreshInterval) { clearInterval(autoRefreshInterval); autoRefreshInterval = null; }
    if (elapsedInterval) { clearInterval(elapsedInterval); elapsedInterval = null; }
  }

  function navigateStep(i) {
    if (i < 0 || i >= $selectedSteps.length) return;
    loadLog(i);
    startAutoRefresh(i);
    navigateTo('logs', $selectedPipeline?.uuid, i);
    stepDropdownOpen = false;
  }

  $effect(() => {
    if ($selectedSteps.length === 0) return;
    // Prefer explicit logStepUUID (set by detail page Log button) over URL stepNum
    let idx = 0;
    if ($logStepUUID) {
      const found = $selectedSteps.findIndex(s => s.uuid === $logStepUUID);
      if (found >= 0) idx = found;
    } else if (typeof $stepNumFromUrl === 'number' && $stepNumFromUrl >= 0) {
      idx = $stepNumFromUrl;
    }
    const step = $selectedSteps[idx];
    if (!step?.uuid) return;
    const stepId = `${step.uuid}-${$refreshTrigger}`;
    if (logLoadedForStep === stepId) return;
    logLoadedForStep = stepId;
    loadLog(idx);
    startAutoRefresh(idx);
  });

  onDestroy(() => { stopAutoRefresh(); });

  function nextMatch() { if (matchCount === 0) return; currentMatchIdx = (currentMatchIdx + 1) % matchCount; scrollToLine(searchMatches[currentMatchIdx] + 1); }
  function prevMatch() { if (matchCount === 0) return; currentMatchIdx = (currentMatchIdx - 1 + matchCount) % matchCount; scrollToLine(searchMatches[currentMatchIdx] + 1); }
  function clearSearch() { searchTerm = ''; currentMatchIdx = -1; matchCount = 0; }

  function handleKeydown(e) {
    if (stepDropdownOpen && e.key === 'Escape') { stepDropdownOpen = false; e.preventDefault(); return; }
    if (e.key === '/' && !e.ctrlKey && !e.metaKey && !stepDropdownOpen) {
      e.preventDefault();
      if (searchInputEl) searchInputEl.focus();
      return;
    }
    if (e.key === 'n' && !e.ctrlKey && !e.metaKey && matchCount > 0 && document.activeElement !== searchInputEl) { e.preventDefault(); nextMatch(); return; }
    if (e.key === 'N' && !e.ctrlKey && !e.metaKey && matchCount > 0 && document.activeElement !== searchInputEl) { e.preventDefault(); prevMatch(); return; }
    if (e.key === 'Escape') {
      if (searchTerm) { clearSearch(); e.preventDefault(); }
      else if (stepDropdownOpen) { stepDropdownOpen = false; e.preventDefault(); }
      else { navigateTo('detail', $selectedPipeline?.uuid); }
      return;
    }
  }

  let copyMsg = $state('');
  async function copyLog() {
    try { await navigator.clipboard.writeText($logContent || ''); copyMsg = 'Copied'; setTimeout(() => { copyMsg = ''; }, 1500); }
    catch { copyMsg = 'Failed'; setTimeout(() => { copyMsg = ''; }, 1500); }
  }
  function onScroll(e) {
    const el = e.currentTarget;
    const wasAutoScroll = autoScroll;
    autoScroll = el.scrollHeight - el.scrollTop - el.clientHeight < 30;
    // Stop auto-refresh when user scrolls away from bottom; resume when they scroll back
    if (!autoScroll && wasAutoScroll) {
      stopAutoRefresh();
    } else if (autoScroll && !wasAutoScroll) {
      startAutoRefresh(currentStepIndex);
    }
  }

  function stepDotClass(state) {
    const name = state?.name;
    if (name === 'SUCCESSFUL') return 'dot-success';
    if (name === 'FAILED' || name === 'ERROR') return 'dot-error';
    if (name === 'IN_PROGRESS') return 'dot-running';
    if (name === 'STOPPED' || name === 'PAUSED' || name === 'EXPIRED') return 'dot-stopped';
    return 'dot-pending';
  }

  function stepStatusBadgeClass(state) {
    const name = state?.name || '';
    if (name === 'SUCCESSFUL') return 'badge-success';
    if (name === 'FAILED' || name === 'ERROR') return 'badge-error';
    if (name === 'IN_PROGRESS') return 'badge-info';
    if (name === 'STOPPED' || name === 'PAUSED' || name === 'EXPIRED') return 'badge-warning';
    return 'badge-neutral';
  }

  function statusLabel(state) {
    if (!state) return 'unknown';
    const name = (state.name || '').toLowerCase();
    if (name === 'completed') {
      const result = (state.result?.name || '').toLowerCase();
      if (result === 'failed' || result === 'error') return 'failed';
      if (result === 'stopped') return 'stopped';
      return 'succeeded';
    }
    if (name === 'in_progress') return 'running';
    if (name === 'pending') return 'pending';
    if (name === 'stopped') return 'stopped';
    if (name === 'not_started') return 'not started';
    return state.name;
  }

  let currentStep = $derived.by(() => {
    if (currentStepIndex < 0 || currentStepIndex >= $selectedSteps.length) return null;
    return $selectedSteps[currentStepIndex];
  });

  let isRunning = $derived(currentStep?.state?.name === 'IN_PROGRESS' || currentStep?.state?.name === 'PENDING');
  let hasPrev = $derived(currentStepIndex > 0);
  let hasNext = $derived(currentStepIndex >= 0 && currentStepIndex < $selectedSteps.length - 1);

  function closeDropdownOnOutside(e) {
    if (stepDropdownOpen && !e.target.closest('.step-dropdown-wrapper')) stepDropdownOpen = false;
  }
</script>

<svelte:window onkeydown={handleKeydown} onclick={closeDropdownOnOutside} />

<div class="log-page">
  <!-- Sticky Header ──────────────────────────────────────── -->
  <div class="log-sticky-header">
    <div class="log-header-row">
      <div class="log-header-left">
        <button class="btn btn-ghost btn-sm" onclick={() => navigateTo('detail', $selectedPipeline?.uuid)}>
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="19" y1="12" x2="5" y2="12"></line><polyline points="12 19 5 12 12 5"></polyline>
          </svg>
          #{$selectedPipeline?.build_number || '—'}
        </button>

        <!-- Step Dropdown Selector -->
        <div class="step-dropdown-wrapper">
          <button class="step-dropdown-trigger" onclick={() => (stepDropdownOpen = !stepDropdownOpen)}>
            <span class="step-dot {stepDotClass(currentStep?.state)}"></span>
            <span class="step-dropdown-label">{$logStepName || 'Step'}</span>
            <svg class="step-dropdown-chevron" class:open={stepDropdownOpen} width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="6 9 12 15 18 9"></polyline>
            </svg>
          </button>
          {#if stepDropdownOpen}
            <div class="step-dropdown-menu">
              {#each $selectedSteps as step, i}
                <button
                  class="step-dropdown-item"
                  class:active={i === currentStepIndex}
                  onclick={() => navigateStep(i)}
                >
                  <span class="step-dot {stepDotClass(step.state)}"></span>
                  <span class="step-dropdown-item-name">{step.name || `Step ${i + 1}`}</span>
                  <span class="badge {stepStatusBadgeClass(step.state)} step-dropdown-badge">{statusLabel(step.state)}</span>
                </button>
              {/each}
            </div>
          {/if}
        </div>

        {#if currentStep}
          <span class="badge {stepStatusBadgeClass(currentStep.state)} step-header-badge">
            {statusLabel(currentStep.state)}
          </span>
        {/if}
      </div>

      <div class="log-header-spacer"></div>

      <div class="log-header-actions">
        {#if autoRefreshInterval}
          <span class="live-indicator">
            <span class="live-dot"></span>
            Live
          </span>
        {/if}
        {#if isRunning && elapsed}
          <span class="elapsed-counter">{elapsed}</span>
        {/if}

        <button class="btn btn-ghost btn-icon" class:btn-active={wrapLines} onclick={() => (wrapLines = !wrapLines)} title="Toggle word wrap">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M4 6h16"></path><path d="M4 12h10"></path><path d="M4 18h16"></path>
          </svg>
        </button>

        <button class="btn btn-ghost btn-icon" onclick={handleRefresh} title="Refresh">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="23 4 23 10 17 10"></polyline><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
          </svg>
        </button>

        <button class="btn btn-ghost btn-icon" onclick={copyLog} title={copyMsg || 'Copy full log'}>
          {#if copyMsg}
            ✓
          {:else}
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
          {/if}
        </button>

        <div class="step-nav-group">
          <button class="btn btn-ghost btn-icon" disabled={!hasPrev} onclick={() => navigateStep(currentStepIndex - 1)} title="Previous step">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>
          <span class="step-nav-label">{currentStepIndex + 1}/{$selectedSteps.length}</span>
          <button class="btn btn-ghost btn-icon" disabled={!hasNext} onclick={() => navigateStep(currentStepIndex + 1)} title="Next step">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="9 18 15 12 9 6"></polyline>
            </svg>
          </button>
        </div>

        <button class="btn btn-ghost btn-icon" onclick={() => { autoScroll = true; if (logViewerEl) logViewerEl.scrollTop = logViewerEl.scrollHeight; }} title="Scroll to bottom">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="6 9 12 15 18 9"></polyline>
          </svg>
        </button>
      </div>
    </div>
  </div>

  <!-- Persistent Search Bar ──────────────────────────────── -->
  <div class="search-bar" class:search-bar--has-term={!!searchTerm}>
    <svg class="search-icon" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line>
    </svg>
    <input
      class="search-input"
      type="text"
      bind:value={searchTerm}
      bind:this={searchInputEl}
      placeholder="Search log…  ( / to focus, Enter to confirm, Esc to clear )"
    />
    {#if searchTerm}
      <span class="search-match-info">
        {#if matchCount > 0}
          {currentMatchIdx + 1}/{matchCount}
        {:else}
          no matches
        {/if}
      </span>
      <div class="search-nav">
        <button class="btn btn-ghost btn-icon btn-xs" onclick={prevMatch} title="Previous match (Shift+N)">▲</button>
        <button class="btn btn-ghost btn-icon btn-xs" onclick={nextMatch} title="Next match (N)">▼</button>
      </div>
      <button class="btn btn-ghost btn-icon" onclick={clearSearch} title="Clear search">✕</button>
    {/if}
  </div>

  <!-- Content Area ───────────────────────────────────────── -->
  {#if logState === 'loading'}
    <div class="log-status-center">
      <div class="spinner"></div>
      <p>Loading log{currentStep ? ` for "${currentStep.name || `Step ${currentStepIndex + 1}`}"` : '…'}</p>
    </div>
  {:else if logState === 'error'}
    <div class="log-status-center error">
      <span class="empty-icon">⚠</span>
      <p>{logError}</p>
      <button class="btn btn-primary btn-sm" onclick={handleRefresh}>Retry</button>
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
          <div class="log-line" class:search-hit={searchMatches.includes(line.num - 1) && currentMatchIdx >= 0 && searchMatches[currentMatchIdx] === line.num - 1}>
            <span class="line-num">{line.num}</span>
            <span class="line-content">{@html line.html}</span>
          </div>
        {/each}
      </div>

      <!-- Integrated bottom status bar -->
      <div class="log-bottom-bar">
        <span>{coloredLines.length} lines</span>
        <span class="log-bottom-auto {autoScroll ? 'on' : 'off'}">
          {autoScroll ? '● Auto-scroll' : '○ Scrolled'}
        </span>
        {#if lastRefreshed}
          <span class="log-bottom-refresh">Updated {lastRefreshed.toLocaleTimeString()}</span>
        {/if}
      </div>
    </div>
  {/if}
</div>

<style>
  .log-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--bg-root);
  }

  /* ── Sticky Header ──────────────────────────────────────── */
  .log-sticky-header {
    flex-shrink: 0;
    background: var(--bg-root);
    border-bottom: 1px solid var(--border-default);
    z-index: 10;
  }

  .log-header-row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-5);
    min-height: 44px;
  }

  .log-header-left {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 0;
  }

  .log-header-spacer { flex: 1; }

  .log-header-actions {
    display: flex;
    align-items: center;
    gap: var(--space-1);
  }

  .step-header-badge {
    font-size: 10px;
    padding: 1px 8px;
    font-weight: 600;
    text-transform: uppercase;
  }

  /* Live indicator */
  .live-indicator {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--accent-text);
    background: var(--accent-muted);
    padding: 1px 6px;
    border-radius: var(--radius-sm);
  }
  .live-dot {
    width: 5px; height: 5px;
    border-radius: 50%;
    background: var(--accent-text);
    animation: pulse-dot 1.5s ease-in-out infinite;
  }
  @keyframes pulse-dot { 0%,100%{opacity:1;transform:scale(1)}50%{opacity:0.4;transform:scale(0.7)} }

  .elapsed-counter {
    font-family: var(--font-mono);
    font-size: 11px;
    font-weight: 600;
    color: var(--accent-text);
    background: var(--bg-input);
    padding: 1px 6px;
    border-radius: var(--radius-sm);
  }

  .btn-active {
    color: var(--accent-text);
    background: var(--accent-muted);
  }

  /* Step navigation group */
  .step-nav-group {
    display: flex;
    align-items: center;
    gap: 0;
    background: var(--bg-input);
    border-radius: var(--radius-sm);
    padding: 0 var(--space-1);
    margin: 0 var(--space-2);
    border: 1px solid var(--border-subtle);
  }
  .step-nav-label {
    font-family: var(--font-mono);
    font-size: 10px;
    color: var(--text-tertiary);
    padding: 0 var(--space-1);
    min-width: 32px;
    text-align: center;
    font-weight: 600;
  }

  /* ── Step Dropdown ──────────────────────────────────────── */
  .step-dropdown-wrapper {
    position: relative;
    min-width: 0;
  }

  .step-dropdown-trigger {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    background: var(--bg-input);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md);
    padding: var(--space-2) var(--space-4);
    cursor: pointer;
    font-family: var(--font-ui);
    font-size: var(--font-size-sm);
    font-weight: 600;
    color: var(--text-primary);
    max-width: 260px;
    transition: all var(--transition-fast);
  }
  .step-dropdown-trigger:hover { border-color: var(--accent); background: var(--bg-panel); }

  .step-dropdown-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .step-dropdown-chevron {
    flex-shrink: 0;
    color: var(--text-tertiary);
    transition: transform var(--transition-fast);
    margin-left: var(--space-1);
  }
  .step-dropdown-chevron.open { transform: rotate(180deg); }

  .step-dot {
    width: 7px; height: 7px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .dot-success { background: var(--success); }
  .dot-error   { background: var(--error); }
  .dot-running { background: var(--accent-text); animation: pulse-dot 1.5s ease-in-out infinite; }
  .dot-stopped { background: var(--border-emphasis); }
  .dot-pending { background: var(--bg-root); border: 1.5px solid var(--border-default); }

  .step-dropdown-menu {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    min-width: 280px;
    max-height: 320px;
    overflow-y: auto;
    background: var(--bg-panel);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md);
    box-shadow: 0 8px 24px rgba(0,0,0,0.35);
    z-index: 100;
    padding: var(--space-1);
  }

  .step-dropdown-item {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    width: 100%;
    padding: var(--space-2) var(--space-4);
    border: none;
    background: transparent;
    color: var(--text-primary);
    font-family: var(--font-ui);
    font-size: var(--font-size-sm);
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: background-color 0.1s;
  }
  .step-dropdown-item:hover { background: var(--bg-hover); }
  .step-dropdown-item.active { background: var(--accent-muted); }

  .step-dropdown-item-name {
    flex: 1;
    text-align: left;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .step-dropdown-badge {
    font-size: 9px;
    padding: 1px 6px;
    font-weight: 600;
    text-transform: uppercase;
    white-space: nowrap;
    flex-shrink: 0;
  }

  /* ── Search Bar ─────────────────────────────────────────── */
  .search-bar {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-1) var(--space-5);
    height: 32px;
    background: var(--bg-input);
    border-bottom: 1px solid var(--border-subtle);
    flex-shrink: 0;
    transition: border-color var(--transition-fast);
  }
  .search-bar--has-term { border-bottom-color: var(--accent); }

  .search-icon {
    color: var(--text-tertiary);
    flex-shrink: 0;
  }

  .search-input {
    flex: 1;
    background: none;
    border: none;
    color: var(--text-primary);
    font-family: var(--font-ui);
    font-size: 12px;
    outline: none;
  }
  .search-input::placeholder { color: var(--text-tertiary); }

  .search-match-info {
    font-family: var(--font-mono);
    font-size: 10px;
    color: var(--success);
    font-weight: 600;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .search-nav {
    display: flex;
    gap: 0;
  }

  /* ── Loading / Error Center ─────────────────────────────── */
  .log-status-center {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-4);
    flex: 1;
    color: var(--text-secondary);
    padding: var(--space-9);
  }
  .log-status-center.error { color: var(--error); }
  .empty-icon { font-size: 32px; }

  /* ── Log Viewer ─────────────────────────────────────────── */
  .log-viewer {
    flex: 1;
    background: var(--terminal-bg);
    overflow-y: auto;
    overflow-x: hidden;
    min-height: 0;
    font-family: var(--font-mono);
    position: relative;
  }
  .log-viewer.no-wrap { overflow-x: auto; }
  .log-viewer.no-wrap .log-line { min-width: max-content; }
  .wrap-lines .line-content { white-space: pre-wrap; word-break: break-word; }
  .no-wrap .line-content { white-space: pre; }

  .log-lines {
    padding: var(--space-1) 0;
    min-height: calc(100% - 26px);
  }

  .log-line {
    display: flex;
    font-size: 12px;
    line-height: 1.55;
    min-height: 1.55em;
    color: var(--text-primary);
  }
  .log-line:hover { background: rgba(255,255,255,0.02); }
  .log-line.search-hit { background: var(--accent-muted); outline: 1px solid var(--accent); outline-offset: -1px; }

  .line-num {
    flex-shrink: 0;
    width: 3.2rem;
    text-align: right;
    padding-right: var(--space-4);
    color: var(--text-tertiary);
    user-select: none;
    border-right: 1px solid var(--border-subtle);
    margin-right: var(--space-4);
    opacity: 0.5;
  }
  .log-line:hover .line-num { opacity: 0.8; }
  .log-line.search-hit .line-num { opacity: 1; color: var(--accent-text); }

  .line-content { flex: 1; min-width: 0; padding-right: var(--space-4); }

  .log-viewer::-webkit-scrollbar { width: 8px; height: 8px; }
  .log-viewer::-webkit-scrollbar-track { background: var(--terminal-bg); }
  .log-viewer::-webkit-scrollbar-thumb { background: var(--border-emphasis); border-radius: 4px; }

  /* ── Integrated Bottom Bar ──────────────────────────────── */
  .log-bottom-bar {
    position: sticky;
    bottom: 0;
    display: flex;
    align-items: center;
    gap: var(--space-5);
    padding: 0 var(--space-5);
    height: 26px;
    background: rgba(30,30,30,0.92);
    backdrop-filter: blur(4px);
    border-top: 1px solid var(--border-subtle);
    font-size: 10px;
    font-family: var(--font-mono);
    color: var(--text-tertiary);
    user-select: none;
  }

  .log-bottom-auto { font-weight: 600; }
  .log-bottom-auto.on { color: var(--success); }
  .log-bottom-auto.off { color: var(--text-tertiary); }

  .log-bottom-refresh { margin-left: auto; opacity: 0.6; }

  @keyframes blink { 50% { opacity: 0; } }
</style>