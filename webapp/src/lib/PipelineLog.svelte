<script>
  import { onDestroy, tick } from 'svelte';
  import { activeProject, selectedPipeline, selectedSteps, logContent, logStepName, logStepUUID, showError, refreshTrigger } from '../stores/appState.js';
  import { navigateTo, stepNumFromUrl } from '../stores/router.js';
  import { getStepLog } from '../stores/api.js';

  let logState = $state('idle');
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
      consecutiveErrorCount = 0;
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
      autoRefreshInterval = setInterval(() => {
        if (consecutiveErrorCount >= MAX_CONSECUTIVE_ERRORS) { stopAutoRefresh(); return; }
        if (logState !== 'loading') loadLog(stepIndex);
      }, 5000);
    }
  }

  function stopAutoRefresh() { if (autoRefreshInterval) { clearInterval(autoRefreshInterval); autoRefreshInterval = null; } }

  $effect(() => {
    if ($selectedSteps.length === 0) return;
    const idx = typeof $stepNumFromUrl === 'number' && $stepNumFromUrl >= 0 ? $stepNumFromUrl : 0;
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
  function clearSearch() { searchTerm = ''; searchMode = false; currentMatchIdx = -1; matchCount = 0; }

  function handleKeydown(e) {
    if (searchMode) {
      if (e.key === 'Escape') { clearSearch(); e.preventDefault(); return; }
      if (e.key === 'Enter') { searchMode = false; if (matchCount > 0) { currentMatchIdx = 0; scrollToLine(searchMatches[0] + 1); } e.preventDefault(); return; }
      return;
    }
    if (e.key === '/' && !e.ctrlKey && !e.metaKey) { e.preventDefault(); searchMode = true; searchTerm = ''; setTimeout(() => { const inp = document.querySelector('.search-input'); if (inp) inp.focus(); }, 50); return; }
    if (e.key === 'n' && !e.ctrlKey && !e.metaKey && matchCount > 0) { e.preventDefault(); nextMatch(); return; }
    if (e.key === 'N' && !e.ctrlKey && !e.metaKey && matchCount > 0) { e.preventDefault(); prevMatch(); return; }
    if (e.key === 'Escape') { if (matchCount > 0) { clearSearch(); e.preventDefault(); } else { navigateTo('detail', $selectedPipeline?.uuid); } return; }
  }

  let copyMsg = $state('');
  async function copyLog() { try { await navigator.clipboard.writeText($logContent || ''); copyMsg = 'Copied!'; setTimeout(() => { copyMsg = ''; }, 2000); } catch { copyMsg = 'Failed'; } }
  function onScroll(e) { const el = e.currentTarget; autoScroll = el.scrollHeight - el.scrollTop - el.clientHeight < 30; }

  function stepDotClass(state) {
    const name = state?.name;
    if (name === 'SUCCESSFUL') return 'dot-success';
    if (name === 'FAILED' || name === 'ERROR') return 'dot-error';
    if (name === 'IN_PROGRESS') return 'dot-running';
    if (name === 'STOPPED' || name === 'PAUSED' || name === 'EXPIRED') return 'dot-stopped';
    return 'dot-pending';
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="pipeline-log">
  <!-- Log Toolbar -->
  <div class="log-toolbar">
    <button class="btn btn-ghost" onclick={() => navigateTo('detail', $selectedPipeline?.uuid)}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="19" y1="12" x2="5" y2="12"></line><polyline points="12 19 5 12 12 5"></polyline>
      </svg>
      Back
    </button>
    <div class="log-toolbar-title">
      <span class="log-title-text">{$logStepName || 'Step Log'}</span>
      {#if autoRefreshInterval}
        <span class="log-auto-tag">auto-refreshing</span>
      {/if}
    </div>
    <div class="log-toolbar-spacer"></div>
    <div class="log-actions">
      <button class="btn btn-secondary btn-sm" onclick={copyLog} title="Copy full log">{copyMsg || 'Copy'}</button>
      <button class="btn btn-secondary btn-sm" onclick={() => (wrapLines = !wrapLines)} title="Toggle word wrap">{wrapLines ? 'Wrap' : 'No wrap'}</button>
      <button class="btn btn-secondary btn-sm" onclick={() => { autoScroll = true; if (logViewerEl) logViewerEl.scrollTop = logViewerEl.scrollHeight; }} title="Scroll to bottom">Bottom</button>
      <button class="btn btn-secondary btn-sm" onclick={handleRefresh}>Refresh</button>
    </div>
  </div>

  <!-- Step Tabs -->
  {#if $selectedSteps.length > 0}
    <div class="step-tabs">
      {#each $selectedSteps as step, i}
        <button
          class="step-tab"
          class:active={$logStepUUID === step.uuid}
          onclick={() => {
            loadLog(i);
            startAutoRefresh(i);
            navigateTo('logs', $selectedPipeline?.uuid, i);
          }}
        >
          <span class="step-tab-dot {stepDotClass(step.state)}"></span>
          {step.name || `Step ${i + 1}`}
        </button>
      {/each}
    </div>
  {/if}

  <!-- Search Bar -->
  {#if searchMode}
    <div class="search-bar search-bar--active">
      <svg class="search-bar-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line>
      </svg>
      <input class="search-input" type="text" bind:value={searchTerm} placeholder="Search log… (Enter to confirm, Esc to cancel)" autofocus />
      <button class="btn btn-ghost btn-icon" onclick={clearSearch}>✕</button>
    </div>
  {:else if searchTerm}
    <div class="search-bar">
      <svg class="search-bar-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line>
      </svg>
      <span class="search-term-text">"{searchTerm}"</span>
      {#if matchCount > 0}
        <span class="search-info">Match {currentMatchIdx + 1} of {matchCount}</span>
        <button class="btn btn-ghost btn-icon" onclick={prevMatch} title="Previous (Shift+N)">▲</button>
        <button class="btn btn-ghost btn-icon" onclick={nextMatch} title="Next (N)">▼</button>
      {/if}
      <button class="btn btn-ghost btn-icon" onclick={clearSearch}>✕</button>
    </div>
  {/if}

  {#if logState === 'loading'}
    <div class="log-loading">
      <div class="spinner"></div>
      <p>Loading log…</p>
    </div>
  {:else if logState === 'error'}
    <div class="log-error">
      <span class="empty-icon">⚠</span>
      <p>{logError}</p>
      <button class="btn btn-primary" onclick={handleRefresh}>Retry</button>
    </div>
  {:else}
    <div class="log-viewer" class:wrap-lines={wrapLines} class:no-wrap={!wrapLines} bind:this={logViewerEl} onscroll={onScroll}>
      <div class="log-lines">
        {#each coloredLines as line (line.num)}
          <div class="log-line" class:search-match={searchMatches.includes(line.num - 1) && currentMatchIdx >= 0 && searchMatches[currentMatchIdx] === line.num - 1}>
            <span class="line-num">{line.num}</span>
            <span class="line-content">{@html line.html}</span>
          </div>
        {/each}
      </div>
    </div>

    <div class="log-statusbar">
      <span>{coloredLines.length} lines</span>
      <span class:status-on={autoScroll}>{autoScroll ? 'Auto-scroll ON' : 'Auto-scroll OFF'}</span>
      <span class="statusbar-help">/ search · n next · N prev · Esc back</span>
    </div>
  {/if}
</div>
<style>
  .pipeline-log {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
    padding: var(--space-6);
    height: 100%;
  }

  /* ── Toolbar ────────────────────────────────────────────── */
  .log-toolbar {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    flex-shrink: 0;
    user-select: none;
  }

  .log-toolbar-title {
    display: flex;
    align-items: baseline;
    gap: var(--space-4);
  }

  .log-title-text {
    font-size: var(--font-size-md);
    font-weight: 600;
    color: var(--text-primary);
  }

  .log-auto-tag {
    font-size: 10px;
    font-weight: 600;
    color: var(--accent-text);
    background: var(--accent-muted);
    padding: 1px 6px;
    border-radius: var(--radius-sm);
  }

  .log-toolbar-spacer { flex: 1; }

  .log-actions {
    display: flex;
    gap: var(--space-3);
  }

  /* ── Step Tabs ──────────────────────────────────────────── */
  .step-tabs {
    display: flex;
    gap: var(--space-2);
    overflow-x: auto;
    padding-bottom: var(--space-1);
    flex-shrink: 0;
  }

  .step-tab {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    background: var(--bg-panel);
    border: 1px solid var(--border-subtle);
    color: var(--text-tertiary);
    padding: var(--space-2) var(--space-4);
    border-radius: var(--radius-md);
    cursor: pointer;
    font-size: var(--font-size-xs);
    white-space: nowrap;
    transition: all var(--transition-fast);
    font-family: var(--font-ui);
  }
  .step-tab:hover { color: var(--text-primary); border-color: var(--border-default); }
  .step-tab.active { background: var(--accent-muted); border-color: var(--accent); color: var(--accent-text); }

  .step-tab-dot {
    width: 6px; height: 6px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .dot-success { background: var(--success); }
  .dot-error   { background: var(--error); }
  .dot-running { background: var(--accent-text); animation: pulse-dot 1.5s ease-in-out infinite; }
  .dot-stopped { background: var(--border-emphasis); }
  .dot-pending { background: var(--border-default); }
  @keyframes pulse-dot { 0%,100%{opacity:1;transform:scale(1)}50%{opacity:0.4;transform:scale(0.7)} }

  /* ── Search Bar ──────────────────────────────────────────── */
  .search-bar {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    background: var(--bg-panel);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md);
    padding: var(--space-3) var(--space-5);
    flex-shrink: 0;
  }
  .search-bar--active { border-color: var(--accent); }

  .search-bar-icon { color: var(--text-tertiary); flex-shrink: 0; }

  .search-input {
    flex: 1;
    background: none;
    border: none;
    color: var(--text-primary);
    font-family: var(--font-ui);
    font-size: var(--font-size-sm);
    outline: none;
  }
  .search-input::placeholder { color: var(--text-tertiary); }

  .search-term-text { color: var(--accent-text); font-size: var(--font-size-sm); }
  .search-info { color: var(--success); font-size: var(--font-size-xs); font-family: var(--font-mono); }

  /* ── Loading / Error ─────────────────────────────────────── */
  .log-loading, .log-error {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--space-4);
    padding: var(--space-9);
    color: var(--text-secondary);
  }

  /* ── Log Viewer (terminal) ────────────────────────────────── */
  .log-viewer {
    flex: 1;
    background: var(--terminal-bg);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-md);
    overflow-y: auto;
    overflow-x: hidden;
    min-height: 0;
    font-family: var(--font-mono);
  }
  .log-viewer.no-wrap { overflow-x: auto; }
  .log-viewer.no-wrap .log-line { min-width: max-content; }
  .wrap-lines .line-content { white-space: pre-wrap; word-break: break-word; }
  .no-wrap .line-content { white-space: pre; }

  .log-lines { padding: var(--space-3) 0; }

  .log-line {
    display: flex;
    font-size: 12px;
    line-height: 1.55;
    min-height: 1.55em;
    color: var(--text-primary);
  }
  .log-line:hover { background: var(--bg-hover); }
  .log-line.search-match { background: var(--accent-muted); outline: 1px solid var(--accent); outline-offset: -1px; }

  .line-num {
    flex-shrink: 0;
    width: 3.2rem;
    text-align: right;
    padding-right: var(--space-4);
    color: var(--text-tertiary);
    user-select: none;
    border-right: 1px solid var(--border-subtle);
    margin-right: var(--space-4);
  }

  .line-content { flex: 1; min-width: 0; padding-right: var(--space-5); }

  .log-viewer::-webkit-scrollbar { width: 8px; height: 8px; }
  .log-viewer::-webkit-scrollbar-track { background: var(--terminal-bg); }
  .log-viewer::-webkit-scrollbar-thumb { background: var(--border-emphasis); border-radius: 4px; }

  /* ── Status Bar ──────────────────────────────────────────── */
  .log-statusbar {
    display: flex;
    align-items: center;
    gap: var(--space-5);
    padding: var(--space-2) var(--space-5);
    background: var(--bg-panel);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-md);
    font-size: 11px;
    color: var(--text-tertiary);
    font-family: var(--font-mono);
    flex-shrink: 0;
    user-select: none;
  }

  .statusbar-help { margin-left: auto; }
  .status-on { color: var(--success); font-weight: 600; }

  @keyframes blink { 50% { opacity: 0; } }
</style>