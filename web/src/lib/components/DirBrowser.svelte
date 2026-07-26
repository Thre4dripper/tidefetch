<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type BrowseResult } from '../api';
  import { fmtBytes } from '../format';
  import Modal from './Modal.svelte';

  let { start, onpick, onclose }: {
    start: string;
    onpick: (path: string) => void;
    onclose: () => void;
  } = $props();

  let result = $state<BrowseResult | null>(null);
  let error = $state('');
  let creating = $state(false);
  let newName = $state('');

  async function go(path: string) {
    error = '';
    creating = false;
    try {
      result = await api.browse(path);
    } catch (e) {
      error = (e as Error).message;
    }
  }

  async function createFolder(e: Event) {
    e.preventDefault();
    if (!result || !newName.trim()) return;
    try {
      const res = await api.mkdir(result.path, newName.trim());
      newName = '';
      creating = false;
      await go(res.path);
    } catch (err) {
      error = (err as Error).message;
    }
  }

  onMount(() => go(start));
</script>

<Modal title="Choose folder" {onclose} width={500}>
  {#if result}
    <div class="chips">
      <button class="chip" onclick={() => go(result!.home)} title={result.home}>⌂ Home</button>
      <button class="chip" onclick={() => go('/')} title="Filesystem root">/ Root</button>
      <button class="chip" onclick={() => go(result!.downloadDir)} title={result.downloadDir}>⬇ Downloads</button>
    </div>

    <div class="path mono">{result.path}</div>

    {#if error}
      <div class="errbar">{error}</div>
    {/if}

    <div class="list">
      {#if result.path !== result.parent}
        <button class="entry dim" onclick={() => go(result!.parent)}>↰ ..</button>
      {/if}
      {#each result.dirs as d (d.path)}
        <button class="entry" onclick={() => go(d.path)}>
          📁 {d.name}
        </button>
      {/each}
      {#if result.dirs.length === 0}
        <div class="dim none">No subfolders</div>
      {/if}
    </div>

    {#if creating}
      <form class="newrow" onsubmit={createFolder}>
        <input placeholder="Folder name" bind:value={newName} />
        <button class="btn sm primary" disabled={!newName.trim()}>Create</button>
        <button type="button" class="btn sm" onclick={() => (creating = false)}>Cancel</button>
      </form>
    {/if}

    <div class="foot">
      <span class="left">
        <button class="btn sm" onclick={() => (creating = !creating)}>＋ New folder</button>
        <span class="dim free">{result.free ? `${fmtBytes(result.free)} free` : ''}</span>
      </span>
      <div class="btns">
        <button class="btn" onclick={onclose}>Cancel</button>
        <button class="btn primary" onclick={() => onpick(result!.path)}>Use this folder</button>
      </div>
    </div>
  {:else if error}
    <div class="errbar">{error}</div>
    <div class="foot">
      <button class="btn" onclick={onclose}>Close</button>
    </div>
  {:else}
    <div class="dim">Loading…</div>
  {/if}
</Modal>

<style>
  .chips {
    display: flex;
    gap: 8px;
    margin-bottom: 10px;
  }
  .chip {
    padding: 5px 12px;
    border-radius: 999px;
    border: 1px solid var(--border);
    color: var(--text-dim);
    font-size: 12.5px;
    transition: color 0.12s, border-color 0.12s, background 0.12s;
    white-space: nowrap;
  }
  .chip:hover {
    color: var(--accent);
    border-color: rgba(52, 216, 195, 0.4);
    background: rgba(52, 216, 195, 0.06);
  }
  .path {
    font-size: 12px;
    color: var(--text-dim);
    background: rgba(0, 0, 0, 0.25);
    border-radius: 6px;
    padding: 7px 10px;
    margin-bottom: 10px;
    word-break: break-all;
  }
  .errbar {
    color: var(--err);
    font-size: 12.5px;
    background: rgba(248, 113, 113, 0.08);
    border: 1px solid rgba(248, 113, 113, 0.25);
    border-radius: 6px;
    padding: 8px 10px;
    margin-bottom: 10px;
  }
  .list {
    max-height: 280px;
    min-height: 160px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .entry {
    text-align: left;
    padding: 8px 10px;
    border-radius: 6px;
    font-size: 13.5px;
  }
  .entry:hover { background: rgba(255, 255, 255, 0.05); }
  .none { padding: 14px 10px; }
  .newrow {
    display: flex;
    gap: 8px;
    margin-top: 10px;
    animation: fadeUp 0.15s ease;
  }
  .newrow input { flex: 1; }
  .foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: 14px;
    gap: 10px;
  }
  .left { display: flex; align-items: center; gap: 10px; }
  .free { font-size: 12px; }
  .btns { display: flex; gap: 8px; }
</style>
