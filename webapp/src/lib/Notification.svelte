<script>
  import { notification } from '../stores/appState.js';
</script>

{#if $notification}
  <div class="notification" class:error={$notification.type === 'error'} class:success={$notification.type === 'success'}>
    <span class="icon">{$notification.type === 'error' ? '❌' : '✅'}</span>
    <span class="message">{$notification.message}</span>
  </div>
{/if}

<style>
  .notification {
    position: fixed;
    top: 1rem;
    right: 1rem;
    z-index: 1000;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1.25rem;
    border-radius: 12px;
    font-size: 0.9rem;
    font-weight: 500;
    animation: slideIn 0.3s ease-out;
    max-width: 420px;
  }

  .notification.error {
    background: #3e1a1a;
    border: 1px solid #670606;
    color: #f4a2a2;
  }

  .notification.success {
    background: #1a3e2a;
    border: 1px solid #06671c;
    color: #a2f4b3;
  }

  @keyframes slideIn {
    from { transform: translateX(100%); opacity: 0; }
    to { transform: translateX(0); opacity: 1; }
  }
</style>