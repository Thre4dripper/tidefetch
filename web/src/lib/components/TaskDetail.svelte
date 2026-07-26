<script lang="ts">
  import { onMount } from 'svelte';
  import { store } from '../store.svelte';
  import { api, type TaskDetail } from '../api';
  import { fmtBytes, fmtSpeed, fmtPct, basename } from '../format';
  import PieceMap from './PieceMap.svelte';
  import Sparkline from './Sparkline.svelte';

  let { gid, onclose }: { gid: string; onclose: () => void } = $props();

  let detail = $state<TaskDetail | null>(null);
  let tab = $state<'overview' | 'files' | 'peers' | 'options'>('overview');
  let options = $state<Record<string, string>>({});
  let optEdits = $state<Record<string, string>>({});
  let fileSel = $state<Set<number>>(new Set());
  let selDirty = $state(false);

  const EDITABLE = [
    'max-download-limit',
    'max-upload-limit',
    'max-connection-per-server',
    'split',
    'seed-ratio',
    'seed-time'
  ];

  async function load() {
    try {
      detail = await api.taskDetail(gid);
      if (!selDirty && detail) {
        fileSel = new Set(detail.files.filter((f) => f.selected).map((f) => f.index));
      }
    } catch {
      /* task disappeared */
    }
  }

  async function loadOptions() {
    try {
      options = await api.taskOptions(gid);
    } catch {
      options = {};
    }
  }

  onMount(() => {
    load();
    const iv = setInterval(load, 1500);
    return () => clearInterval(iv);
  });

  $effect(() => {
    gid; // re-run when gid changes
    detail = null;
    selDirty = false;
    load();
    if (tab === 'options') loadOptions();
  });

  function switchTab(t: typeof tab) {
    tab = t;
    if (t === 'options') loadOptions();
  }

  function toggleFile(index: number) {
    const next = new Set(fileSel);
    next.has(index) ? next.delete(index) : next.add(index);
    fileSel = next;
    selDirty = true;
  }

  function applySelection() {
    store.run('File selection updated', () => api.selectFiles(gid, [...fileSel]));
    selDirty = false;
  }

  function saveOptions() {
    const changed: Record<string, string> = {};
    for (const [k, v] of Object.entries(optEdits)) {
      if (v !== (options[k] ?? '')) changed[k] = v;
    }
    if (Object.keys(changed).length === 0) return;
    store.run('Options saved', () => api.setTaskOptions(gid, changed));
    optEdits = {};
    loadOptions();
  }

  function move(dir: 'top' | 'up' | 'down' | 'bottom') {
    const map = {
      top: [0, 'POS_SET'],
      up: [-1, 'POS_CUR'],
      down: [1, 'POS_CUR'],
      bottom: [0, 'POS_END']
    } as const;
    const [pos, how] = map[dir];
    store.run('Queue updated', () => api.position(gid, pos, how));
  }

  const t = $derived(detail?.task ?? store.selected);
</script>

<aside class="card drawer">
  {#if !t}
    <div class="loading dim">loading…</div>
  {:else}
    <div class="head">
      <span class="ellipsis title" title={t.name}>{t.name}</span>
      <button class="btn icon" onclick={onclose} aria-label="Close details">✕</button>
    </div>

    <div class="tabs">
      {#each ['overview', 'files', 'peers', 'options'] as tb (tb)}
        {#if tb !== 'peers' || t.torrent}
          <button class="tab" class:on={tab === tb} onclick={() => switchTab(tb as typeof tab)}>
            {tb}
          </button>
        {/if}
      {/each}
    </div>

    <div class="body">
      {#if tab === 'overview'}
        <div class="stat-grid">
          <div><span class="faint">Status</span><span class="badge {t.status}">{t.status}</span></div>
          <div><span class="faint">Progress</span><b>{fmtPct(t.progress)}</b></div>
          <div><span class="faint">Size</span><b>{t.total ? fmtBytes(t.total) : 'unknown'}</b></div>
          <div><span class="faint">Downloaded</span><b>{fmtBytes(t.done)}</b></div>
          <div><span class="faint">Down</span><b class="mono">{fmtSpeed(t.downSpeed)}</b></div>
          <div><span class="faint">Up</span><b class="mono">{fmtSpeed(t.upSpeed)}</b></div>
          <div><span class="faint">Connections</span><b>{t.conns}</b></div>
          {#if t.torrent}<div><span class="faint">Seeders</span><b>{t.seeders}</b></div>{/if}
        </div>

        {#if detail?.speedHistory && detail.speedHistory.length > 1}
          <h4>Speed over lifetime {#if t.status !== 'active'}<span class="faint">(final)</span>{/if}</h4>
          <Sparkline points={detail.speedHistory} height={52} color={t.status === 'complete' ? '#4ade80' : '#34d8c3'} />
        {/if}

        {#if detail?.pieces?.length}
          <h4>Pieces <span class="faint">({detail.bt.numPieces})</span></h4>
          <PieceMap pieces={detail.pieces} />
        {/if}

        <h4>Location</h4>
        <div class="mono path">{t.dir}</div>
        {#if t.uri}
          <h4>Source</h4>
          <div class="mono path ellipsis" title={t.uri}>{t.uri}</div>
        {/if}
        {#if detail?.bt.infoHash}
          <h4>Info hash</h4>
          <div class="mono path">{detail.bt.infoHash}</div>
        {/if}
        {#if detail?.servers?.length}
          <h4>Mirrors</h4>
          {#each [...new Set(detail.servers)] as sv (sv)}
            <div class="mono path ellipsis" title={sv}>{sv}</div>
          {/each}
        {/if}
        {#if t.errorMsg}
          <h4>Error</h4>
          <div class="errbox">{t.errorCode}: {t.errorMsg}</div>
        {/if}

        {#if t.status === 'waiting' || t.status === 'paused'}
          <h4>Queue</h4>
          <div class="qbtns">
            <button class="btn sm" onclick={() => move('top')}>⤒ top</button>
            <button class="btn sm" onclick={() => move('up')}>↑ up</button>
            <button class="btn sm" onclick={() => move('down')}>↓ down</button>
            <button class="btn sm" onclick={() => move('bottom')}>⤓ bottom</button>
          </div>
        {/if}

        <div class="danger-zone">
          <button
            class="btn sm danger"
            onclick={() => {
              if (confirm('Remove this download AND delete its files from disk?')) {
                store.run('Removed with files', () => api.taskAction(gid, 'remove', true));
                onclose();
              }
            }}
          >🗑 Remove + delete files</button>
        </div>
      {:else if tab === 'files'}
        {#if detail}
          <div class="filelist">
            {#each detail.files as f (f.index)}
              <label class="file">
                <input
                  type="checkbox"
                  checked={fileSel.has(f.index)}
                  disabled={t.status === 'complete'}
                  onchange={() => toggleFile(f.index)}
                />
                <span class="ellipsis" title={f.path}>{basename(f.path) || f.path}</span>
                <span class="fmeta mono">{f.length ? Math.round((f.done / f.length) * 100) : 0}%</span>
                <span class="fmeta dim">{fmtBytes(f.length)}</span>
              </label>
            {/each}
          </div>
          {#if selDirty}
            <button class="btn primary sm apply" onclick={applySelection}>Apply selection</button>
          {/if}
        {/if}
      {:else if tab === 'peers'}
        {#if detail?.peers?.length}
          <div class="peerlist">
            {#each detail.peers as p (p.ip + ':' + p.port)}
              <div class="peer">
                <span class="mono ellipsis">{p.ip}:{p.port}</span>
                <span class="mono speed">▼{fmtSpeed(p.downSpeed)}</span>
                <span class="mono up">▲{fmtSpeed(p.upSpeed)}</span>
                <span class="dim">{Math.round(p.progress * 100)}%{p.seeder ? ' ⚑' : ''}</span>
              </div>
            {/each}
          </div>
        {:else}
          <p class="dim">No peers connected.</p>
        {/if}
      {:else if tab === 'options'}
        <div class="opts">
          {#each EDITABLE as key (key)}
            <label>
              <span class="mono okey">{key}</span>
              <input
                value={optEdits[key] ?? options[key] ?? ''}
                oninput={(e) => (optEdits = { ...optEdits, [key]: (e.target as HTMLInputElement).value })}
              />
            </label>
          {/each}
          <button class="btn primary sm" onclick={saveOptions}>Save options</button>
          <details>
            <summary class="dim">All options ({Object.keys(options).length})</summary>
            <div class="allopts mono">
              {#each Object.entries(options) as [k, v] (k)}
                <div><span class="faint">{k}</span> {v}</div>
              {/each}
            </div>
          </details>
        </div>
      {/if}
    </div>
  {/if}
</aside>

<style>
  .drawer {
    width: 360px;
    margin: 12px 12px 12px 0;
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    overflow: hidden;
    animation: fadeUp 0.18s ease;
  }
  .loading { padding: 30px; text-align: center; }
  .head {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 14px 14px 10px;
  }
  .title { font-weight: 650; flex: 1; }
  .tabs {
    display: flex;
    gap: 2px;
    padding: 0 12px;
    border-bottom: 1px solid var(--border);
  }
  .tab {
    padding: 8px 12px;
    color: var(--text-dim);
    font-weight: 550;
    text-transform: capitalize;
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
  }
  .tab.on { color: var(--accent); border-bottom-color: var(--accent); }
  .body {
    flex: 1;
    overflow-y: auto;
    padding: 14px;
  }
  .stat-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px 14px;
  }
  .stat-grid > div {
    display: flex;
    flex-direction: column;
    gap: 3px;
    font-size: 13px;
  }
  .stat-grid .faint { font-size: 11px; text-transform: uppercase; letter-spacing: 0.5px; }
  h4 {
    margin: 18px 0 8px;
    font-size: 11.5px;
    text-transform: uppercase;
    letter-spacing: 0.7px;
    color: var(--text-dim);
  }
  .path {
    font-size: 12px;
    color: var(--text-dim);
    word-break: break-all;
    background: rgba(0, 0, 0, 0.25);
    border-radius: 6px;
    padding: 7px 9px;
  }
  .errbox {
    color: var(--err);
    font-size: 12.5px;
    background: rgba(248, 113, 113, 0.08);
    border: 1px solid rgba(248, 113, 113, 0.25);
    border-radius: 6px;
    padding: 8px 10px;
  }
  .qbtns { display: flex; gap: 6px; flex-wrap: wrap; }
  .danger-zone { margin-top: 22px; }
  .filelist { display: flex; flex-direction: column; gap: 2px; }
  .file {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 6px;
    border-radius: 6px;
    font-size: 12.5px;
    cursor: pointer;
  }
  .file:hover { background: rgba(255, 255, 255, 0.04); }
  .file span:first-of-type { flex: 1; }
  .fmeta { font-size: 11.5px; flex-shrink: 0; }
  .apply { margin-top: 10px; }
  .peerlist { display: flex; flex-direction: column; gap: 4px; font-size: 12px; }
  .peer {
    display: flex;
    gap: 10px;
    align-items: center;
    padding: 5px 6px;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.025);
  }
  .peer span:first-child { flex: 1; }
  .speed { color: var(--accent); }
  .up { color: var(--up); }
  .opts { display: flex; flex-direction: column; gap: 10px; }
  .opts label { display: flex; flex-direction: column; gap: 4px; }
  .okey { font-size: 11.5px; color: var(--text-dim); }
  .allopts {
    font-size: 11px;
    display: flex;
    flex-direction: column;
    gap: 3px;
    margin-top: 8px;
    max-height: 300px;
    overflow-y: auto;
  }
  details summary { cursor: pointer; font-size: 12.5px; }
</style>
