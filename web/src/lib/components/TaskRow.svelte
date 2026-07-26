<script lang="ts">
  import { store } from '../store.svelte';
  import { api, type Task } from '../api';
  import { fmtBytes, fmtSpeed, fmtEta, fmtPct } from '../format';
  import Sparkline from './Sparkline.svelte';

  let { task }: { task: Task } = $props();

  const pct = $derived(task.progress * 100);
  const running = $derived(task.status === 'active' && !task.seeding);

  function toggle(e: Event) {
    e.stopPropagation();
    if (task.status === 'active' || task.status === 'waiting') {
      store.run('Paused', () => api.taskAction(task.gid, 'pause'));
    } else if (task.status === 'paused') {
      store.run('Resumed', () => api.taskAction(task.gid, 'resume'));
    } else if (task.status === 'error') {
      store.run('Retrying', () => api.taskAction(task.gid, 'retry'));
    }
  }

  function remove(e: Event) {
    e.stopPropagation();
    const gone = task.status === 'complete' || task.status === 'error' || task.status === 'removed';
    if (!gone && store.prefs.confirmRemove && !confirm(`Remove "${task.name}"?`)) return;
    store.run('Removed', () => api.taskAction(task.gid, 'remove'));
    if (store.selectedGid === task.gid) store.selectedGid = null;
  }

  const statusLabel = $derived(
    task.seeding ? 'seeding' : task.status
  );
</script>

<div
  class="row card"
  class:selected={store.selectedGid === task.gid}
  class:compact={store.prefs.compact}
  onclick={() => (store.selectedGid = store.selectedGid === task.gid ? null : task.gid)}
  role="button"
  tabindex="0"
  onkeydown={(e) => e.key === 'Enter' && (store.selectedGid = task.gid)}
>
  <div class="top">
    <span class="tname ellipsis" title={task.name}>
      {#if task.torrent}<span class="tor" title="BitTorrent">⧉</span>{/if}
      {task.name}
    </span>
    <span class="badge {task.status}">{statusLabel}</span>
  </div>

  <div class="bar">
    <div
      class="fill"
      class:done={task.status === 'complete'}
      class:err={task.status === 'error'}
      style="width:{pct}%"
    ></div>
  </div>

  <div class="meta">
    <span class="mono">{fmtPct(task.progress)}</span>
    <span class="dim">{fmtBytes(task.done)} / {task.total ? fmtBytes(task.total) : '?'}</span>
    {#if running}
      <span class="speed mono">▼ {fmtSpeed(task.downSpeed)}</span>
      {#if task.upSpeed > 0}<span class="up mono">▲ {fmtSpeed(task.upSpeed)}</span>{/if}
      {#if fmtEta(task)}<span class="dim">ETA {fmtEta(task)}</span>{/if}
      {#if task.torrent}<span class="faint">Seeds {task.seeders}</span>{/if}
    {:else if task.seeding}
      <span class="up mono">▲ {fmtSpeed(task.upSpeed)}</span>
    {:else if task.status === 'error' && task.errorMsg}
      <span class="errtext ellipsis" title={task.errorMsg}>{task.errorMsg}</span>
    {/if}
    <span class="gap"></span>
    {#if task.speeds && task.speeds.length > 1}
      <span class="lifechart" title="Download speed over this task's lifetime">
        <Sparkline points={task.speeds} height={20} color={task.status === 'complete' ? '#4ade80' : '#34d8c3'} />
      </span>
    {/if}
    <span class="rowbtns">
      {#if task.status === 'active' || task.status === 'waiting'}
        <button class="btn icon" onclick={toggle} title="Pause" aria-label="Pause">⏸</button>
      {:else if task.status === 'paused'}
        <button class="btn icon" onclick={toggle} title="Resume" aria-label="Resume">▶</button>
      {:else if task.status === 'error'}
        <button class="btn icon" onclick={toggle} title="Retry" aria-label="Retry">↻</button>
      {/if}
      <button class="btn icon danger" onclick={remove} title="Remove" aria-label="Remove">✕</button>
    </span>
  </div>
</div>

<style>
  .row {
    padding: 12px 14px 10px;
    cursor: pointer;
    transition: border-color 0.12s, background 0.12s;
    animation: fadeUp 0.15s ease;
  }
  .row:hover { border-color: var(--border-strong); background: rgba(255, 255, 255, 0.045); }
  .row.selected { border-color: rgba(52, 216, 195, 0.5); background: rgba(52, 216, 195, 0.05); }
  .top {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 8px;
  }
  .tname { font-weight: 600; flex: 1; }
  .tor { color: var(--accent-2); margin-right: 4px; }
  .bar {
    height: 5px;
    background: rgba(255, 255, 255, 0.06);
    border-radius: 3px;
    overflow: hidden;
    margin-bottom: 7px;
  }
  .fill {
    height: 100%;
    background: var(--grad);
    border-radius: 3px;
    transition: width 0.4s ease;
  }
  .fill.done { background: var(--ok); }
  .fill.err { background: var(--err); }
  .meta {
    display: flex;
    align-items: center;
    gap: 14px;
    font-size: 12.5px;
    min-height: 24px;
  }
  .speed { color: var(--accent); }
  .up { color: var(--up); }
  .errtext { color: var(--err); max-width: 40%; }
  .gap { flex: 1; }
  .lifechart {
    width: 96px;
    flex-shrink: 0;
    opacity: 0.9;
  }
  .rowbtns { display: flex; gap: 6px; opacity: 0; transition: opacity 0.12s; }
  .row:hover .rowbtns, .row.selected .rowbtns { opacity: 1; }

  .row.compact { padding: 8px 12px 7px; }
  .row.compact .top { margin-bottom: 5px; }
  .row.compact .bar { margin-bottom: 5px; height: 4px; }
  .row.compact .meta { min-height: 20px; font-size: 12px; }
</style>
