<script lang="ts">
  import { store } from '../store.svelte';
  import { api } from '../api';

  function pauseAll() {
    store.run('Paused all', () => api.bulkAction('pauseAll'));
  }
  function resumeAll() {
    store.run('Resumed all', () => api.bulkAction('resumeAll'));
  }
  function purge() {
    store.run('Cleared finished', () => api.bulkAction('purge'));
  }
</script>

<header>
  <div class="search">
    <span class="faint">⌕</span>
    <input placeholder="Search downloads…" bind:value={store.search} />
  </div>
  <div class="actions">
    <button class="btn sm" onclick={resumeAll} title="Resume all downloads">▶ Resume all</button>
    <button class="btn sm" onclick={pauseAll} title="Pause all downloads">⏸ Pause all</button>
    <button class="btn sm" onclick={purge} title="Clear all finished and errored results">✕ Clear done</button>
    <button class="btn primary" onclick={() => (store.addOpen = true)}>＋ Add download</button>
  </div>
</header>

<style>
  header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 12px 8px;
  }
  .search {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 0 12px;
    max-width: 420px;
  }
  .search input {
    flex: 1;
    background: none;
    border: none;
    padding: 9px 0;
  }
  .search:focus-within {
    border-color: var(--accent);
  }
  .search input:focus-visible {
    outline: none;
  }
  .actions {
    display: flex;
    gap: 8px;
    margin-left: auto;
  }
</style>
