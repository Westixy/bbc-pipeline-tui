<script>
  import { onDestroy, tick } from 'svelte';
  import { activeProject, selectedPipeline, selectedSteps, selectedVariables, selectedLogVariables, logContent, logStepName, logStepUUID, showError, refreshTrigger } from '../stores/appState.js';
  import { navigateTo, stepNumFromUrl, pipelineUUIDFromUrl, workspaceFromUrl, repoSlugFromUrl } from '../stores/router.js';
  import { getStepLog, getPipeline, listVariables, getLogVariables } from '../stores/api.js';
  import { resolveStepStatus, formatDate, formatDuration } from './utils.js';

  let logState = $state('idle');
  let logError = $state('');
  let currentStepIndex = $state(-1);
  let isAutoRefreshing = $state(false);
  let autoRefreshInterval = null;
  let autoScroll = $state(true);
  let wrapLines = $state(true);
  let searchTerm = $state('');
  let matchCount = $state(0);
  let currentMatchIdx = $state(-1);
  let logViewerEl = $state(null);
  let searchInputEl = $state(null);
  let stepDropdownOpen = $state(false);
  let elapsedInterval = null;
  let elapsed = $state('');
  let lastRefreshedTime = $state(null);
  let lastRefreshed = null;
  let copiedHash = $state(false);
  let varsExpanded = $state(false);
  let varsTab = $state('log');
  let varsFilter = $state('');
  let showAllVars = $state(false);
  const VARS_PREVIEW_COUNT = 6;

  let hasPipelineVars = $derived($selectedVariables.length > 0);
  let hasLogVars = $derived($selectedLogVariables.length > 0);
  let hasAnyVars = $derived(hasPipelineVars || hasLogVars);

  let effectiveVarsTab = $derived.by(() => {
    if (varsTab === 'log' && hasLogVars) return 'log';
    if (varsTab === 'pipeline' && hasPipelineVars) return 'pipeline';
    if (hasLogVars) return 'log';
    if (hasPipelineVars) return 'pipeline';
    return 'log';
  });
  let currentVars = $derived(effectiveVarsTab === 'pipeline' ? $selectedVariables : $selectedLogVariables);

  let filteredVars = $derived.by(() => {
    const filter = varsFilter.trim().toLowerCase();
    const vars = currentVars;
    if (!filter) return vars;
    return vars.filter(v => v.key.toLowerCase().includes(filter) || (v.value || '').toLowerCase().includes(filter));
  });

  let displayedVars = $derived(showAllVars ? filteredVars : filteredVars.slice(0, VARS_PREVIEW_COUNT));
  let hasMoreVars = $derived(filteredVars.length > VARS_PREVIEW_COUNT && !showAllVars);

  // Cache variables per pipeline UUID to avoid re-fetching on every auto-refresh cycle.
  const varsCache = new Map();

  async function fetchVariables(projectId, pipelineUuid) {
    try {
      const cached = varsCache.get(pipelineUuid);
      if (cached) {
        selectedVariables.set(cached.pipelineVars);
        selectedLogVariables.set(cached.logVars);
        return;
      }
      const [varsResp, logVarsResp] = await Promise.all([
        listVariables(projectId).catch(() => ({ variables: [] })),
        getLogVariables(projectId, pipelineUuid).catch(() => ({ variables: [] })),
      ]);
      const pipelineVars = varsResp.variables || [];
      const logVars = logVarsResp.variables || [];
      selectedVariables.set(pipelineVars);
      selectedLogVariables.set(logVars);
      // Cache only if we got at least one non-empty result
      if (pipelineVars.length > 0 || logVars.length > 0) {
        varsCache.set(pipelineUuid, { pipelineVars, logVars });
      }
    } catch (_) {}
  }

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
  let bootstrapped = $state(false);

  // Bootstrap from URL when stores are empty (page reload scenario)
  $effect(() => {
    const urlUuid = $pipelineUUIDFromUrl;
    if (!$activeProject || !urlUuid || bootstrapped) return;
    if ($selectedSteps.length > 0) return; // already have steps, main effect will handle
    // Wait until the activeProject matches the URL workspace/repoSlug
    const urlWs = $workspaceFromUrl;
    const urlRs = $repoSlugFromUrl;
    if ($activeProject.workspace !== urlWs || $activeProject.repo_slug !== urlRs) return;
    bootstrapped = true;
    bootstrapFromUrl(urlUuid);
  });

  async function bootstrapFromUrl(uuid) {
    logState = 'loading';
    try {
      const data = await getPipeline($activeProject.id, uuid);
      const pipelineData = data.pipeline || data;
      const stepsData = data.steps || [];
      selectedPipeline.set(pipelineData);
      selectedSteps.set(stepsData);
      fetchVariables($activeProject.id, uuid);
      await tick();
    } catch (e) {
      logState = 'error';
      logError = e.message;
      showError(e.message);
    }
  }

  async function loadLog(stepIndex, opts = {}) {
    if (!$activeProject || !$selectedPipeline?.uuid || !$selectedSteps[stepIndex]) return;
    const step = $selectedSteps[stepIndex];
    const requestId = ++latestRequestId;
    // Only show loading state on initial load, not on background refreshes
    if (!opts.silent) { logState = 'loading'; logError = ''; }
    try {
      const data = await getStepLog($activeProject.id, $selectedPipeline.uuid, step.uuid);
      if (requestId !== latestRequestId) return;
      logContent.set(data.log || 'No log content available');
      logStepName.set(step.name || `Step ${stepIndex + 1}`);
      logStepUUID.set(step.uuid);
      currentStepIndex = stepIndex;
      logState = 'ready';
      lastRefreshed = new Date();
      lastRefreshedTime = lastRefreshed;
      consecutiveErrorCount = 0;
      updateElapsed();
      await tick();
      if (autoScroll && logViewerEl) logViewerEl.scrollTop = logViewerEl.scrollHeight;
    } catch (e) {
      if (requestId !== latestRequestId) return;
      // On silent refresh, don't show error state unless we've exceeded the threshold
      if (opts.silent && logState === 'ready') {
        consecutiveErrorCount++;
      } else {
        logError = e.message;
        logState = 'error';
        consecutiveErrorCount++;
      }
    }
  }

  function handleRefresh() { if ($logStepUUID) { const idx = $selectedSteps.findIndex(s => s.uuid === $logStepUUID); if (idx >= 0) loadLog(idx); } }

  // ── Auto-refresh: refresh step log + pipeline/step status to auto-advance ──
  let refreshingPipelineState = $state(false);

  async function refreshPipelineAndSteps(stepIndex) {
    if (!$activeProject || !$selectedPipeline?.uuid) return;
    if (refreshingPipelineState) return;
    refreshingPipelineState = true;
    try {
      const data = await getPipeline($activeProject.id, $selectedPipeline.uuid);
      const freshSteps = data.steps || data.values || [];
      if (freshSteps.length > 0) {
        selectedSteps.set(freshSteps);
        const pipelineData = data.pipeline || data;
        if (pipelineData) selectedPipeline.set(pipelineData);
      }
      fetchVariables($activeProject.id, $selectedPipeline.uuid);
    } catch (_) {} finally {
      refreshingPipelineState = false;
    }
  }

  function findNextRunningStep(fromIndex) {
    const steps = $selectedSteps;
    // Look from the next step onward
    for (let i = fromIndex + 1; i < steps.length; i++) {
      const name = steps[i].state?.name;
      if (name === 'IN_PROGRESS' || name === 'PENDING' || name === 'NOT_STARTED') {
        return i;
      }
    }
    return -1;
  }

  function startAutoRefresh(stepIndex) {
    stopAutoRefresh();
    const step = $selectedSteps[stepIndex];
    if (!step) return;
    const sname = step.state?.name;
    if (sname === 'IN_PROGRESS' || sname === 'PENDING') {
      elapsedInterval = setInterval(updateElapsed, 1000);
      isAutoRefreshing = true;
      autoRefreshInterval = setInterval(async () => {
        if (consecutiveErrorCount >= MAX_CONSECUTIVE_ERRORS) { stopAutoRefresh(); return; }

        // 1. Refresh pipeline/step status to detect completions
        await refreshPipelineAndSteps(stepIndex);

        // 2. Check if the current step has completed
        const steps = $selectedSteps;
        const current = steps[stepIndex];
        const currentName = current?.state?.name;
        const isDone = currentName !== 'IN_PROGRESS' && currentName !== 'PENDING';

        if (isDone) {
          // Current step finished — find the next running step
          const nextIdx = findNextRunningStep(stepIndex);
          if (nextIdx >= 0) {
            // Navigate to next running step
            await loadLog(nextIdx);
            startAutoRefresh(nextIdx);
            navigateTo('logs', $selectedPipeline?.uuid, nextIdx);
          } else {
            // No more running steps — stop refreshing
            stopAutoRefresh();
          }
          return;
        }

        // 3. Refresh the log content silently
        if (logState !== 'loading') loadLog(stepIndex, { silent: true });
      }, 5000);
    }
  }

  function stopAutoRefresh() {
    if (autoRefreshInterval) { clearInterval(autoRefreshInterval); autoRefreshInterval = null; }
    isAutoRefreshing = false;
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
    const status = resolveStepStatus(state);
    if (status === 'SUCCESSFUL') return 'dot-success';
    if (status === 'FAILED') return 'dot-error';
    if (status === 'IN_PROGRESS') return 'dot-running';
    if (status === 'STOPPED') return 'dot-stopped';
    return 'dot-pending';
  }

  function stepStatusBadgeClass(state) {
    const status = resolveStepStatus(state);
    if (status === 'SUCCESSFUL') return 'badge-success';
    if (status === 'FAILED') return 'badge-error';
    if (status === 'IN_PROGRESS') return 'badge-info';
    if (status === 'STOPPED') return 'badge-warning';
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

  let isRunning = $derived(resolveStepStatus(currentStep?.state) === 'IN_PROGRESS' || resolveStepStatus(currentStep?.state) === 'PENDING');
  let hasPrev = $derived(currentStepIndex > 0);
  let hasNext = $derived(currentStepIndex >= 0 && currentStepIndex < $selectedSteps.length - 1);

  function closeDropdownOnOutside(e) {
    if (stepDropdownOpen && !e.target.closest('.step-dropdown-wrapper')) stepDropdownOpen = false;
  }

  async function copyCommitHash() {
    const hash = $selectedPipeline?.target?.commit?.hash;
    if (!hash) return;
    try { await navigator.clipboard.writeText(hash); }
    catch {
      const ta = document.createElement('textarea'); ta.value = hash;
      ta.style.position = 'fixed'; ta.style.opacity = '0'; document.body.appendChild(ta);
      ta.select(); document.execCommand('copy'); document.body.removeChild(ta);
    }
    copiedHash = true;
    setTimeout(() => (copiedHash = false), 2000);
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
        {#if isAutoRefreshing}
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

  <!-- Pipeline Info Strip ──────────────────────────────── -->
  {#if $selectedPipeline}
    <div class="pipeline-info-strip">
      <div class="info-field">
        <span class="info-label">Branch</span>
        <span class="info-value info-branch">{$selectedPipeline.target?.ref_name || $selectedPipeline.target?.type || '—'}</span>
      </div>
      {#if $selectedPipeline.target?.selector}
        <div class="info-field">
          <span class="info-label">Pattern</span>
          <span class="info-value info-pattern">
            {typeof $selectedPipeline.target.selector === 'object'
              ? ($selectedPipeline.target.selector.pattern || '—')
              : $selectedPipeline.target.selector}
          </span>
        </div>
      {/if}
      <div class="info-field">
        <span class="info-label">Trigger</span>
        <span class="info-value">{$selectedPipeline.trigger?.name || '—'}</span>
      </div>
      <div class="info-field">
        <span class="info-label">Creator</span>
        <span class="info-value">{$selectedPipeline.creator?.display_name || $selectedPipeline.creator?.username || '—'}</span>
      </div>
      <div class="info-field">
        <span class="info-label">Duration</span>
        <span class="info-value">{formatDuration($selectedPipeline.created_on, $selectedPipeline.completed_on, $selectedPipeline.build_seconds_used || 0)}</span>
      </div>
      <div class="info-field">
        <span class="info-label">Created</span>
        <span class="info-value">{formatDate($selectedPipeline.created_on)}</span>
      </div>
      {#if $selectedPipeline.completed_on}
        <div class="info-field">
          <span class="info-label">Completed</span>
          <span class="info-value">{formatDate($selectedPipeline.completed_on)}</span>
        </div>
      {/if}
      {#if $selectedPipeline.target?.commit?.hash}
        <div class="info-field">
          <span class="info-label">Commit</span>
          <span class="info-value commit-row">
            <code>{$selectedPipeline.target.commit.hash.substring(0, 8)}</code>
            <button class="btn btn-ghost btn-icon" onclick={copyCommitHash} title="Copy full commit hash">
              {#if copiedHash}✓{:else}
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
              {/if}
            </button>
          </span>
        </div>
      {/if}
    </div>
  {/if}

  <!-- Step Progress Bar ──────────────────────────────────── -->
  <div class="step-progress-bar">
    {#each $selectedSteps as step, i}
      <button
        class="step-progress-item"
        class:active={i === currentStepIndex}
        class:done={resolveStepStatus(step.state) === 'SUCCESSFUL'}
        class:failed={resolveStepStatus(step.state) === 'FAILED'}
        onclick={() => navigateStep(i)}
        title="{step.name || `Step ${i + 1}`} — {statusLabel(step.state)}"
      >
        <span class="step-progress-dot"></span>
        <span class="step-progress-label">{step.name || `Step ${i + 1}`}</span>
      </button>
      {#if i < $selectedSteps.length - 1}
        <span class="step-progress-connector" class:done={resolveStepStatus(step.state) === 'SUCCESSFUL'}></span>
      {/if}
    {/each}
  </div>

  <!-- Variables Section ──────────────────────────────────── -->
  {#if hasAnyVars}
    <div class="log-vars-section" class:vars-expanded={varsExpanded}>
      <div class="log-vars-header">
        <button class="vars-toggle" onclick={() => (varsExpanded = !varsExpanded)} title={varsExpanded ? 'Collapse variables' : 'Expand variables'}>
          <svg class="vars-toggle-icon" class:open={varsExpanded} width="8" height="8" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="9 18 15 12 9 6"></polyline>
          </svg>
        </button>
        <h3 class="log-vars-title">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="2" y="2" width="20" height="20" rx="2" ry="2"></rect><line x1="6" y1="6" x2="6.01" y2="6"></line><line x1="12" y1="6" x2="12.01" y2="6"></line><line x1="18" y1="6" x2="18.01" y2="6"></line><line x1="6" y1="12" x2="6.01" y2="12"></line><line x1="12" y1="12" x2="12.01" y2="12"></line><line x1="18" y1="12" x2="18.01" y2="12"></line>
          </svg>
          Variables
        </h3>

        {#if !varsExpanded}
          <div class="vars-inline-summary">
            {#each currentVars as v}
              <span class="vars-inline-pair">
                <code class="var-key">{v.key}</code>
                <span class="vars-inline-eq">=</span>
                <span class="vars-inline-val" class:var-value-masked={v.secured}>
                  {v.secured ? '••••••••' : (v.value || '(empty)')}
                </span>
              </span>
            {/each}
          </div>
        {:else}
          <div class="vars-tabs-segmented">
            {#if hasLogVars}
              <button
                class="vars-tab-segment"
                class:active={effectiveVarsTab === 'log'}
                onclick={() => (varsTab = 'log')}
              >
                Log
                <span class="tab-count">{$selectedLogVariables.length}</span>
              </button>
            {/if}
            {#if hasPipelineVars}
              <button
                class="vars-tab-segment"
                class:active={effectiveVarsTab === 'pipeline'}
                onclick={() => (varsTab = 'pipeline')}
              >
                Pipeline
                <span class="tab-count">{$selectedVariables.length}</span>
              </button>
            {/if}
          </div>
        {/if}
      </div>

      {#if varsExpanded}
        <div class="log-vars-body">
          {#if effectiveVarsTab === 'log' && hasLogVars}
            <p class="vars-hint">Parsed from "Pipeline variables:" block in the first step's log.</p>
          {/if}

          {#if currentVars.length > VARS_PREVIEW_COUNT}
            <div class="vars-search">
              <svg class="search-icon-sm" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="11" cy="11" r="8"></circle><line x1="21" y1="21" x2="16.65" y2="16.65"></line>
              </svg>
              <input
                type="text"
                class="vars-search-input"
                placeholder="Filter variables…"
                bind:value={varsFilter}
              />
            </div>
          {/if}

          {#if filteredVars.length === 0}
            <p class="empty-text">No variables match <code>{varsFilter}</code></p>
          {:else}
            <div class="vars-list">
              {#each displayedVars as v}
                <div class="var-row">
                  <div class="var-row-left">
                    <code class="var-key">{v.key}</code>
                    {#if v.secured}
                      <span class="secured-tag">SECURED</span>
                    {/if}
                  </div>
                  <span class="var-value" class:var-value-masked={v.secured}>
                    {v.secured ? '••••••••••••••••' : (v.value || '(empty)')}
                  </span>
                </div>
              {/each}
            </div>

            {#if hasMoreVars}
              <button class="vars-show-more" onclick={() => (showAllVars = true)}>
                Show all {filteredVars.length} variables…
              </button>
            {/if}
            {#if showAllVars && filteredVars.length > VARS_PREVIEW_COUNT}
              <button class="vars-show-more" onclick={() => (showAllVars = false)}>
                Show fewer
              </button>
            {/if}
          {/if}
        </div>
      {/if}
    </div>
  {/if}

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
        {#if lastRefreshedTime}
          <span class="log-bottom-refresh">Updated {lastRefreshedTime.toLocaleTimeString()}</span>
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

  /* ── Pipeline Info Strip ───────────────────────────────── */
  .pipeline-info-strip {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-4);
    padding: var(--space-2) var(--space-5);
    background: var(--bg-panel);
    border-bottom: 1px solid var(--border-subtle);
    flex-shrink: 0;
    min-height: 32px;
  }

  .info-field {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    white-space: nowrap;
  }

  .info-label {
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-tertiary);
  }

  .info-value {
    font-size: var(--font-size-xs);
    color: var(--text-secondary);
    font-weight: 500;
  }

  .info-branch {
    font-family: var(--font-mono);
    color: var(--accent-text);
  }

  .info-pattern {
    font-family: var(--font-mono);
    color: #d2a8ff;
  }

  .commit-row {
    display: flex;
    align-items: center;
    gap: var(--space-1);
  }

  .commit-row code {
    font-family: var(--font-mono);
    font-size: 10px;
    background: var(--bg-input);
    padding: 1px 5px;
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
  }

  /* ── Step Progress Bar ──────────────────────────────────── */
  .step-progress-bar {
    display: flex;
    align-items: center;
    gap: 0;
    padding: 0 var(--space-5);
    height: 36px;
    background: var(--bg-panel);
    border-bottom: 1px solid var(--border-subtle);
    overflow-x: auto;
    flex-shrink: 0;
    -ms-overflow-style: none;
    scrollbar-width: none;
  }
  .step-progress-bar::-webkit-scrollbar { display: none; }

  .step-progress-item {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: 0 var(--space-2);
    height: 28px;
    border: none;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--text-tertiary);
    cursor: pointer;
    font-family: var(--font-ui);
    font-size: 10px;
    font-weight: 500;
    white-space: nowrap;
    flex-shrink: 0;
    transition: all var(--transition-fast);
  }
  .step-progress-item:hover {
    color: var(--text-primary);
    background: var(--bg-hover);
  }
  .step-progress-item.active {
    color: var(--text-primary);
    background: var(--accent-muted);
    font-weight: 700;
  }
  .step-progress-item.done {
    color: var(--success);
  }
  .step-progress-item.failed {
    color: var(--error);
  }

  .step-progress-dot {
    width: 8px; height: 8px;
    border-radius: 50%;
    border: 1.5px solid var(--border-emphasis);
    background: transparent;
    flex-shrink: 0;
    transition: all var(--transition-fast);
  }
  .step-progress-item.done .step-progress-dot {
    background: var(--success);
    border-color: var(--success);
  }
  .step-progress-item.failed .step-progress-dot {
    background: var(--error);
    border-color: var(--error);
  }
  .step-progress-item.active .step-progress-dot {
    background: var(--accent-text);
    border-color: var(--accent-text);
    animation: pulse-dot 1.5s ease-in-out infinite;
  }

  .step-progress-label {
    max-width: 100px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .step-progress-item.active .step-progress-label {
    max-width: 140px;
  }

  .step-progress-connector {
    width: 16px;
    height: 1px;
    background: var(--border-emphasis);
    flex-shrink: 0;
    transition: background var(--transition-fast);
  }
  .step-progress-connector.done {
    background: var(--success);
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

  /* ── Variables Section ─────────────────────────────────── */
  .log-vars-section {
    flex-shrink: 0;
    background: var(--bg-panel);
    border-bottom: 1px solid var(--border-subtle);
  }

  .log-vars-header {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    padding: var(--space-2) var(--space-5);
    flex-wrap: wrap;
    min-height: 32px;
  }

  /* Collapse/expand toggle */
  .vars-toggle {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    border: none;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--text-tertiary);
    cursor: pointer;
    flex-shrink: 0;
    transition: all var(--transition-fast);
    padding: 0;
  }
  .vars-toggle:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .vars-toggle-icon {
    transition: transform var(--transition-fast);
    flex-shrink: 0;
  }
  .vars-toggle-icon.open {
    transform: rotate(90deg);
  }

  /* Inline summary when collapsed */
  .vars-inline-summary {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
    flex: 1;
    min-width: 0;
    overflow: hidden;
    padding: 2px 0;
  }

  .vars-inline-pair {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    max-width: 260px;
    flex-shrink: 0;
  }

  .vars-inline-eq {
    font-family: var(--font-mono);
    font-size: 10px;
    font-weight: 700;
    color: var(--text-tertiary);
    flex-shrink: 0;
    padding: 0 1px;
  }

  .vars-inline-val {
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 180px;
  }
  .vars-inline-val.var-value-masked {
    letter-spacing: 0.08em;
    color: var(--text-tertiary);
  }

  .log-vars-section.vars-expanded .vars-inline-summary {
    display: none;
  }

  .log-vars-title {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: 10px;
    font-weight: 700;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    margin: 0;
    flex-shrink: 0;
  }

  .vars-tabs-segmented {
    display: flex;
    gap: 0;
    background: var(--bg-input);
    border-radius: var(--radius-md);
    padding: 2px;
  }

  .vars-tab-segment {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    padding: var(--space-1) var(--space-3);
    border-radius: var(--radius-sm);
    border: none;
    background: transparent;
    color: var(--text-tertiary);
    font-family: var(--font-ui);
    font-size: var(--font-size-xs);
    font-weight: 500;
    cursor: pointer;
    transition: all var(--transition-fast);
    white-space: nowrap;
  }
  .vars-tab-segment:hover { color: var(--text-primary); }
  .vars-tab-segment.active {
    background: var(--bg-panel);
    color: var(--text-primary);
    box-shadow: 0 1px 3px rgba(0,0,0,0.12);
    font-weight: 600;
  }

  .tab-count {
    font-size: 10px;
    font-weight: 600;
    background: var(--bg-badge);
    color: var(--text-tertiary);
    padding: 0 5px;
    border-radius: var(--radius-sm);
    font-family: var(--font-mono);
  }

  .log-vars-body {
    padding: 0 var(--space-5) var(--space-3);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .vars-hint {
    font-size: var(--font-size-xs);
    color: var(--text-tertiary);
    font-style: italic;
    margin: 0;
    padding: var(--space-1) 0;
  }

  .vars-search {
    position: relative;
    display: flex;
    align-items: center;
  }

  .search-icon-sm {
    position: absolute;
    left: 7px;
    color: var(--text-tertiary);
    pointer-events: none;
  }

  .vars-search-input {
    width: 100%;
    padding: var(--space-1) var(--space-3) var(--space-1) 26px;
    font-family: var(--font-ui);
    font-size: var(--font-size-xs);
    color: var(--text-primary);
    background: var(--bg-input);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-sm);
    outline: none;
  }
  .vars-search-input:focus { border-color: var(--border-focus); }
  .vars-search-input::placeholder { color: var(--text-tertiary); }

  .vars-list {
    display: flex;
    flex-direction: column;
    gap: 1px;
    max-height: 220px;
    overflow-y: auto;
  }

  .var-row {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    padding: var(--space-2) var(--space-3);
    border-radius: var(--radius-sm);
    background: var(--bg-panel);
    transition: background var(--transition-fast);
  }
  .var-row:hover { background: var(--bg-hover); }

  .var-row-left {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-wrap: wrap;
  }

  .var-key {
    font-family: var(--font-mono);
    font-size: var(--font-size-xs);
    font-weight: 600;
    color: var(--accent-text);
    background: var(--accent-muted);
    padding: 1px 6px;
    border-radius: var(--radius-sm);
    flex-shrink: 0;
  }

  .secured-tag {
    font-size: 9px;
    color: var(--warning);
    background: var(--warning-bg);
    padding: 1px 5px;
    border-radius: var(--radius-sm);
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    flex-shrink: 0;
  }

  .var-value {
    font-family: var(--font-mono);
    font-size: var(--font-size-xs);
    color: var(--text-primary);
    word-break: break-all;
    line-height: 1.5;
    padding-left: var(--space-2);
  }

  .var-value-masked {
    color: var(--text-tertiary);
    letter-spacing: 0.15em;
  }

  .vars-show-more {
    display: block;
    width: 100%;
    padding: var(--space-2);
    border: 1px dashed var(--border-default);
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--accent-text);
    font-family: var(--font-ui);
    font-size: var(--font-size-xs);
    font-weight: 500;
    cursor: pointer;
    text-align: center;
    transition: all var(--transition-fast);
  }
  .vars-show-more:hover {
    background: var(--bg-hover);
    border-color: var(--accent-text);
    color: var(--accent-text);
  }
</style>
