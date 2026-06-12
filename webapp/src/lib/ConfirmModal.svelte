<script>
  let { open = false, title = 'Confirm', message = '', confirmText = 'OK', cancelText = 'Cancel', danger = false, onconfirm, oncancel } = $props();

  function handleKeydown(e) {
    if (e.key === 'Escape') {
      oncancel?.();
    }
    if (e.key === 'Enter' && open) {
      e.preventDefault();
      onconfirm?.();
    }
  }

  function onBackdropClick(e) {
    if (e.target === e.currentTarget) oncancel?.();
  }
</script>

<svelte:window onkeydown={open ? handleKeydown : undefined} />

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="modal-backdrop" onclick={onBackdropClick} role="dialog" aria-modal="true" aria-label={title}>
    <div class="modal-card" class:danger>
      <h3>{title}</h3>
      <p>{message}</p>
      <div class="modal-actions">
        <button class="btn btn-cancel" onclick={() => oncancel?.()}>{cancelText}</button>
        <button class="btn" class:btn-danger={danger} class:btn-primary={!danger} onclick={() => onconfirm?.()}>
          {confirmText}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.65);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    animation: fadeIn 0.15s ease;
  }
  .modal-card {
    background: #16181c;
    border: 1px solid #2f3336;
    border-radius: 14px;
    padding: 1.5rem;
    max-width: 420px;
    width: 90%;
    animation: scaleIn 0.2s ease;
    box-shadow: 0 12px 40px rgba(0,0,0,0.5);
  }
  .modal-card.danger { border-color: #da3633; }
  h3 { margin: 0 0 0.75rem; font-size: 1.1rem; color: #e7e9ea; }
  .danger h3 { color: #f85149; }
  p { margin: 0 0 1.25rem; color: #8b949e; font-size: 0.9rem; line-height: 1.5; }
  .modal-actions { display: flex; justify-content: flex-end; gap: 0.5rem; }
  .btn {
    padding: 0.5rem 1.25rem;
    border: none;
    border-radius: 8px;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s;
    font-family: inherit;
  }
  .btn-cancel { background: #2f3336; color: #e7e9ea; }
  .btn-cancel:hover { background: #3e4144; }
  .btn-primary { background: #1d9bf0; color: #fff; }
  .btn-primary:hover { background: #1a8cd8; }
  .btn-danger { background: #da3633; color: #fff; }
  .btn-danger:hover { background: #b91c1c; }

  @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
  @keyframes scaleIn { from { opacity: 0; transform: scale(0.95); } to { opacity: 1; transform: scale(1); } }
</style>