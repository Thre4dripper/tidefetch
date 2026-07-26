<script lang="ts">
  import { CirclePause, CirclePlay, Eraser, Plus, Search } from '@lucide/svelte';
  import { store } from '../store.svelte';
  import { api } from '../api';

  const greeting = $derived.by(() => {
    const h = new Date().getHours();
    if (h < 5) return 'Up late';
    if (h < 12) return 'Good morning';
    if (h < 18) return 'Good afternoon';
    return 'Good evening';
  });

  const context = $derived(store.view === 'history' ? 'HISTORY' : 'DOWNLOADS');

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
  <div class="who">
    <span class="eyebrow">{context}</span>
    <h1>{greeting}</h1>
  </div>

  <label class="search">
    <Search size={15} strokeWidth={2} />
    <input placeholder="Search downloads" bind:value={store.search} />
    <kbd class="faint">/</kbd>
  </label>

  <div class="actions">
    <button class="btn icon lg" onclick={resumeAll} title="Resume all downloads" aria-label="Resume all">
      <CirclePlay size={17} />
    </button>
    <button class="btn icon lg" onclick={pauseAll} title="Pause all downloads" aria-label="Pause all">
      <CirclePause size={17} />
    </button>
    <button class="btn icon lg" onclick={purge} title="Clear finished and failed results" aria-label="Clear done">
      <Eraser size={16} />
    </button>
    <button class="btn primary" onclick={() => (store.addOpen = true)}>
      <Plus size={16} strokeWidth={2.4} /> Add download
    </button>
  </div>
</header>

<svelte:window
  onkeydown={(e) => {
    if (e.key === '/' && !(e.target instanceof HTMLInputElement) && !(e.target instanceof HTMLTextAreaElement)) {
      e.preventDefault();
      (document.querySelector('.search input') as HTMLInputElement | null)?.focus();
    }
  }}
/>

<style>
  header {
    display: grid;
    grid-template-columns: 1fr minmax(200px, 330px) auto;
    align-items: center;
    gap: 14px;
    padding: 18px 18px 4px;
  }
  .who {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
  }
  h1 {
    margin: 0;
    font-size: 21px;
    font-weight: 750;
    letter-spacing: -0.01em;
    white-space: nowrap;
  }
  .search {
    display: flex;
    align-items: center;
    gap: 9px;
    background: rgba(255, 255, 255, 0.035);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 0 12px;
    color: var(--text-faint);
    cursor: text;
  }
  .search input {
    flex: 1;
    min-width: 0;
    background: none;
    border: none;
    padding: 9px 0;
  }
  .search kbd {
    font-family: var(--mono);
    font-size: 10px;
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 1px 6px;
  }
  .search:focus-within { border-color: var(--accent); }
  .search input:focus-visible { outline: none; }
  .actions {
    display: flex;
    gap: 8px;
    align-items: center;
  }
  .actions :global(.btn.icon.lg) {
    width: 34px;
    height: 34px;
    color: var(--text-dim);
  }
  .actions :global(.btn.icon.lg:hover) { color: var(--text); }

  @media (max-width: 860px) {
    header {
      grid-template-columns: 1fr auto;
    }
    .search { display: none; }
  }
</style>
