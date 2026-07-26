<script lang="ts">
  import { onMount } from 'svelte';
  import { store } from '../store.svelte';
  import { api, type HistoryEntry } from '../api';
  import { fmtBytes, fmtDate } from '../format';

  let entries = $state<HistoryEntry[]>([]);
  let categories = $state<string[]>([]);
  let q = $state('');
  let cat = $state('');
  let loaded = $state(false);

  async function load() {
    try {
      const res = await api.history(q, cat);
      entries = res.entries;
      categories = res.categories;
    } catch (e) {
      store.toast('err', (e as Error).message);
    } finally {
      loaded = true;
    }
  }

  onMount(load);

  let timer: ReturnType<typeof setTimeout>;
  function onSearch() {
    clearTimeout(timer);
    timer = setTimeout(load, 250);
  }

  function redownload(e: HistoryEntry) {
    if (!e.url) return;
    store.run('Re-added', () =>
      api.add({ kind: 'uri', uris: [e.url!], options: { dir: e.dir || store.downloadDir } })
    );
  }

  function remove(e: HistoryEntry) {
    store.run('Deleted from history', () => api.deleteHistory(e.gid));
    entries = entries.filter((x) => x.gid !== e.gid);
  }

  function clearAll() {
    if (!confirm('Clear the entire download history?')) return;
    store.run('History cleared', () => api.clearHistory());
    entries = [];
  }
</script>

<div class="wrap">
  <div class="bar">
    <input placeholder="Search history…" bind:value={q} oninput={onSearch} />
    <select bind:value={cat} onchange={load}>
      <option value="">All categories</option>
      {#each categories as c (c)}
        <option value={c}>{c}</option>
      {/each}
    </select>
    <span class="gap"></span>
    <button class="btn sm danger" onclick={clearAll}>Clear history</button>
  </div>

  <div class="list">
    {#if loaded && entries.length === 0}
      <div class="empty dim">No history yet</div>
    {/if}
    {#each entries as e (e.gid + e.status)}
      <div class="entry card">
        <div class="main">
          <span class="ellipsis name" title={e.name}>{e.name}</span>
          <span class="badge {e.status}">{e.status}</span>
        </div>
        <div class="sub">
          <span class="chip">{e.category}</span>
          <span class="dim">{fmtBytes(e.size)}</span>
          <span class="faint">{fmtDate(e.finished)}</span>
          <span class="gap"></span>
          {#if e.url}
            <button class="btn sm" onclick={() => redownload(e)} title="Download this file again">↻ Again</button>
          {/if}
          <button class="btn icon danger" onclick={() => remove(e)} title="Delete entry" aria-label="Delete entry">✕</button>
        </div>
      </div>
    {/each}
  </div>
</div>

<style>
  .wrap {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 4px 12px 16px;
  }
  .bar {
    display: flex;
    gap: 10px;
    align-items: center;
    padding-bottom: 10px;
  }
  .bar input { width: 280px; }
  .gap { flex: 1; }
  .list {
    flex: 1;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .empty { text-align: center; padding: 60px 0; }
  .entry { padding: 12px 14px; }
  .main {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 7px;
  }
  .name { font-weight: 600; flex: 1; }
  .sub {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 12.5px;
  }
  .chip {
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 999px;
    background: rgba(59, 130, 246, 0.12);
    color: #93b8f8;
    border: 1px solid rgba(59, 130, 246, 0.25);
  }
</style>
