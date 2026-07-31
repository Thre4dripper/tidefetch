<script lang="ts">
  import { CircleAlert, Pause, Play, RotateCcw, X } from '@lucide/svelte';
  import { store } from '../store.svelte';
  import { api, type Task } from '../api';
  import { fmtBytes, fmtSpeed, fmtEta } from '../format';
  import { fileKind } from '../fileicon';
  import Sparkline from './Sparkline.svelte';

  let { task }: { task: Task } = $props();

  const pct = $derived(task.progress * 100);
  const running = $derived(task.status === 'active' && !task.seeding);
  const kind = $derived(fileKind(task.name, task.torrent));
  const KindIcon = $derived(kind.icon);

  const meta = $derived.by(() => {
    if (task.status === 'error' && task.errorMsg) return task.errorMsg;
    if (!task.total) return fmtBytes(task.done);
    if (task.status === 'complete') return `${fmtBytes(task.total)} · complete`;
    return `${fmtBytes(task.done)} of ${fmtBytes(task.total)}`;
  });

  const speedLabel = $derived.by(() => {
    if (running) return fmtSpeed(task.downSpeed);
    if (task.seeding) return `▲ ${fmtSpeed(task.upSpeed)}`;
    return '—';
  });

  const subLabel = $derived.by(() => {
    if (running) {
      const eta = fmtEta(task);
      return eta ? `ETA ${eta}` : `${task.conns} conn`;
    }
    if (task.seeding) return `${task.seeders} peers`;
    return '';
  });

  // Status is its own column so the STATUS header always has a value under it.
  const status = $derived.by(() => {
    if (task.seeding) return { label: 'Seeding', tone: 'seed' };
    switch (task.status) {
      case 'active': return { label: 'Downloading', tone: 'active' };
      case 'complete': return { label: 'Complete', tone: 'done' };
      case 'paused': return { label: 'Paused', tone: 'paused' };
      case 'waiting': return { label: 'Queued', tone: 'queued' };
      case 'error': return { label: 'Failed', tone: 'error' };
      case 'removed': return { label: 'Removed', tone: 'paused' };
      default: return { label: task.status, tone: 'paused' };
    }
  });

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
</script>

<div
  class="row"
  class:selected={store.selectedGid === task.gid}
  class:compact={store.prefs.compact}
  onclick={() => (store.selectedGid = store.selectedGid === task.gid ? null : task.gid)}
  role="button"
  tabindex="0"
  onkeydown={(e) => e.key === 'Enter' && (store.selectedGid = task.gid)}
>
  <div class="chip {kind.tint}" title={kind.label}>
    {#if task.status === 'error'}
      <CircleAlert size={17} />
    {:else}
      <KindIcon size={17} />
    {/if}
  </div>

  <div class="copy">
    <span class="tname ellipsis" title={task.name}>{task.name}</span>
    <span class="meta ellipsis" class:err={task.status === 'error'}>{meta}</span>
    <div class="bar">
      <div
        class="fill"
        class:done={task.status === 'complete'}
        class:err={task.status === 'error'}
        class:idle={task.status === 'paused' || task.status === 'waiting'}
        style="width:{pct}%"
      ></div>
    </div>
  </div>

  <span class="lifechart">
    {#if !store.prefs.compact && task.speeds && task.speeds.length > 1}
      <Sparkline
        points={task.speeds}
        height={26}
        color={task.status === 'complete' ? 'rgba(184,255,61,.75)' : 'rgba(94,216,231,.8)'}
      />
    {/if}
  </span>

  <div class="pace">
    <b class="mono" class:active={running}>{speedLabel}</b>
    {#if subLabel}<span class="faint">{subLabel}</span>{/if}
  </div>

  <div class="statuscell">
    <span class="pill {status.tone}">{status.label}</span>
  </div>

  <div class="rowbtns">
    {#if task.status === 'active' || task.status === 'waiting'}
      <button class="btn icon" onclick={toggle} title="Pause" aria-label="Pause"><Pause size={13} /></button>
    {:else if task.status === 'paused'}
      <button class="btn icon" onclick={toggle} title="Resume" aria-label="Resume"><Play size={13} /></button>
    {:else if task.status === 'error'}
      <button class="btn icon" onclick={toggle} title="Retry" aria-label="Retry"><RotateCcw size={13} /></button>
    {/if}
    <button class="btn icon danger" onclick={remove} title="Remove" aria-label="Remove"><X size={13} /></button>
  </div>
</div>

<style>
  .row {
    display: grid;
    grid-template-columns: var(--row-grid);
    align-items: center;
    gap: 14px;
    height: 72px;
    padding: 0 14px 0 12px;
    border: 1px solid transparent;
    border-bottom-color: var(--border);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: border-color 0.12s, background 0.12s;
    animation: fadeUp 0.15s ease;
  }
  .row:hover { background: rgba(255, 255, 255, 0.035); border-color: var(--border); }
  .row.selected { border-color: rgba(94, 216, 231, 0.45); background: rgba(94, 216, 231, 0.05); }

  .chip {
    display: grid;
    place-items: center;
    width: 38px;
    height: 38px;
    border-radius: 9px;
    background: rgba(255, 255, 255, 0.05);
    color: var(--text-dim);
  }
  .chip.cyan { color: var(--accent); background: rgba(94, 216, 231, 0.1); }
  .chip.lime { color: var(--signal); background: rgba(184, 255, 61, 0.09); }
  .chip.violet { color: var(--up); background: rgba(140, 130, 255, 0.11); }
  .chip.amber { color: var(--warn); background: rgba(240, 196, 69, 0.1); }

  .copy {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }
  .tname { font-weight: 600; font-size: 13.5px; }
  .meta { font-size: 11.5px; color: var(--text-dim); }
  .meta.err { color: var(--err); }
  .bar {
    height: 3px;
    background: rgba(255, 255, 255, 0.07);
    border-radius: 3px;
    overflow: hidden;
    margin-top: 2px;
  }
  .fill {
    height: 100%;
    background: var(--accent);
    border-radius: 3px;
    transition: width 0.4s ease;
  }
  .fill.done { background: var(--signal); }
  .fill.err { background: var(--err); }
  .fill.idle { background: var(--text-faint); }

  .lifechart {
    width: 100%;
    min-width: 0;
    opacity: 0.9;
  }

  .pace {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 3px;
    text-align: right;
    white-space: nowrap;
  }
  .pace b { font-size: 12.5px; font-weight: 650; }
  .pace b.active { color: var(--accent); }
  .pace span { font-size: 10.5px; }

  .statuscell { display: flex; justify-content: flex-start; min-width: 0; }
  .pill {
    padding: 3.5px 9px;
    border: 1px solid transparent;
    border-radius: 999px;
    font-size: 10.5px;
    font-weight: 700;
    letter-spacing: 0.02em;
    white-space: nowrap;
  }
  .pill.active { color: var(--accent); background: rgba(94, 216, 231, 0.1); border-color: rgba(94, 216, 231, 0.32); }
  .pill.seed { color: var(--accent); background: rgba(94, 216, 231, 0.08); border-color: rgba(94, 216, 231, 0.26); }
  .pill.done { color: var(--signal); background: rgba(184, 255, 61, 0.09); border-color: rgba(184, 255, 61, 0.3); }
  .pill.paused { color: var(--text-dim); background: rgba(255, 255, 255, 0.045); border-color: var(--border-strong); }
  .pill.queued { color: var(--up); background: rgba(140, 130, 255, 0.1); border-color: rgba(140, 130, 255, 0.3); }
  .pill.error { color: var(--err); background: rgba(255, 121, 95, 0.1); border-color: rgba(255, 121, 95, 0.32); }

  .rowbtns {
    display: flex;
    gap: 6px;
    justify-content: flex-end;
    opacity: 0;
    transition: opacity 0.12s;
  }
  .row:hover .rowbtns, .row.selected .rowbtns { opacity: 1; }

  .row.compact { height: 56px; gap: 12px; }
  .row.compact .chip { width: 32px; height: 32px; border-radius: 8px; }
  .row.compact .meta { display: none; }
  .row.compact .tname { font-size: 13px; }

  @media (max-width: 1180px) {
    .lifechart { display: none; }
  }
  @media (max-width: 860px) {
    .statuscell { display: none; }
  }
</style>
