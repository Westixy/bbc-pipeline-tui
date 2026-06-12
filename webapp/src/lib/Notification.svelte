<script>
  import { notification, showError, showSuccess } from '../stores/appState.js';
  import { fly } from 'svelte/transition';

  let visible = $state(false);
  let dismissTimer = null;

  $effect(() => {
    if ($notification) {
      visible = true;
      // Auto-dismiss after 5s for success, 8s for errors
      const duration = $notification.type === 'error' ? 8000 : 5000;
      if (dismissTimer) clearTimeout(dismissTimer);
      dismissTimer = setTimeout(() => {
        visible = false;
        // Clear after transition completes
        setTimeout(() => notification.set(null), 300);
      }, duration);
    }
    return () => { if (dismissTimer) clearTimeout(dismissTimer); };
  });
</script>

{#if $notification && visible}
  <div
    class="notification"
    class:error={$notification.type === 'error'}
    class:success={$notification.type === 'success'}
    transition:fly={{ x: 360, duration: 300, opacity: 0 }}
  >
    <span class="icon">{$notification.type === 'error' ? '✕' : '✓'}</span>
    <span class="message">{$notification.message}</span>
    <button
      class="dismiss-btn"
      onclick={() => { visible = false; setTimeout(() => notification.set(null), 300); }}
      title="Dismiss"
    >
      ✕
    </button>
  </div>
{/if}

<style>
  .notification {
    position: fixed;
    top: 1rem;
    right: 1rem;
    z-index: 1000;
    display: flex;
    align-items: flex-start;
    gap: 0.6rem;
    padding: 0.85rem 1rem 0.85rem 1.25rem;
    border-radius: 12px;
    font-size: 0.88rem;
    font-weight: 500;
    max-width: 440px;
    box-shadow: 0 8px 24px rgba(0,0,0,0.4);
    line-height: 1.45;
  }

  .notification.error {
    background: #2d1214;
    border: 1px solid #670606;
    color: #f4a2a2;
  }

  .notification.success {
    background: #122d1a;
    border: 1px solid #06671c;
    color: #a2f4b3;
  }

  .icon {
    flex-shrink: 0;
    font-size: 0.75rem;
    margin-top: 0.15rem;
    width: 18px;
    height: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
  }

  .error .icon {
    background: rgba(248, 81, 73, 0.15);
  }

  .success .icon {
    background: rgba(63, 185, 80, 0.15);
  }

  .message {
    flex: 1;
    word-break: break-word;
  }

  .dismiss-btn {
    flex-shrink: 0;
    background: none;
    border: none;
    color: inherit;
    opacity: 0.5;
    cursor: pointer;
    font-size: 0.8rem;
    padding: 0.1rem 0.3rem;
    border-radius: 4px;
    transition: opacity 0.15s;
    font-family: inherit;
    line-height: 1;
    margin-top: 0.05rem;
  }

  .dismiss-btn:hover {
    opacity: 1;
    background: rgba(255,255,255,0.08);
  }
</style>