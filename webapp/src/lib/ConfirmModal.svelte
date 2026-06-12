<script>
  let { open = false, title = 'Confirm', message = '', confirmText = 'Confirm', danger = false, onconfirm = () => {}, oncancel = () => {} } = $props();

  function onKeydown(e) {
    if (e.key === 'Escape') { oncancel(); e.preventDefault(); }
    if (e.key === 'Enter' && open) { onconfirm(); e.preventDefault(); }
  }
</script>

<svelte:window onkeydown={onKeydown} />

{#if open}
  <div class="modal-backdrop" onclick={oncancel} role="dialog" aria-modal="true">
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <div class="modal-header">
        <h3 class="modal-title">{title}</h3>
      </div>
      <div class="modal-body">
        <p>{message}</p>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" onclick={oncancel}>Cancel</button>
        <button
          class="btn"
          class:btn-danger={danger}
          class:btn-primary={!danger}
          onclick={onconfirm}
        >
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
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
    animation: fadeIn 150ms ease-out;
  }

  .modal {
    background: var(--bg-panel);
    border: 1px solid var(--border-default);
    border-radius: var(--radius-lg);
    width: 100%;
    max-width: 420px;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
    animation: scaleIn 200ms ease-out;
  }

  .modal-header {
    padding: var(--space-6) var(--space-6) var(--space-3);
    border-bottom: 1px solid var(--border-subtle);
  }

  .modal-title {
    margin: 0;
    font-size: var(--font-size-md);
    font-weight: 600;
    color: var(--text-primary);
  }

  .modal-body {
    padding: var(--space-5) var(--space-6);
    color: var(--text-secondary);
    font-size: var(--font-size-sm);
    line-height: 1.5;
  }

  .modal-body p {
    margin: 0;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-4);
    padding: var(--space-4) var(--space-6) var(--space-6);
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  @keyframes scaleIn {
    from { opacity: 0; transform: scale(0.95); }
    to { opacity: 1; transform: scale(1); }
  }
</style>