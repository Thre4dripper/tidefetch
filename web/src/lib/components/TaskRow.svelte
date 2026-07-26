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
    if (task.status === 'complete') return 'Complete';
    if (task.status === 'paused') return 'Paused';
    if (task.status === 'waiting') return 'Queued';
    if (task.status === 'error') return 'Failed';
    return task.status;
  });

  const subLabel = $derived.by(() => {
    if (running) {
      const eta = fmtEta(task);
      return eta ? `ETA ${eta}` : `${task.conns} connections`;
    }
    if (task.seeding) return `seeding · ${task.seeders} peers`;
    if (task.status === 'complete') return 'done';
    return kind.label.toLowerCase();
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

  {#if !store.prefs.compact && task.speeds && task.speeds.length > 1}
    <span class="lifechart" title="Download speed over this task's lifetime">
      <Sparkline points={task.speeds} height={26} color={task.status === 'complete' ? 'rgba(184,255,61,.75)' : 'rgba(94,216,231,.8)'} />
    </span>
  {/if}

  <div class="pace">
    <b class="mono" class:active={running} class:donetext={task.status === 'complete'} class:errtext={task.status === 'error'}>{speedLabel}</b>
    <span class="faint">{subLabel}</span>
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
    grid-template-columns: 40px minmax(0, 1fr) auto minmax(96px, auto) 66px;
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
    width: 88px;
    flex-shrink: 0;
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
  .pace b.donetext { color: var(--signal); }
  .pace b.errtext { color: var(--err); }
  .pace span { font-size: 10.5px; text-transform: lowercase; }

  .rowbtns {
    display: flex;
    gap: 6px;
    justify-content: flex-end;
    opacity: 0;
    transition: opacity 0.12s;
  }
  .row:hover .rowbtns, .row.selected .rowbtns { opacity: 1; }

  .row.compact { height: 56px; grid-template-columns: 34px minmax(0, 1fr) minmax(96px, auto) 66px; gap: 12px; }
  .row.compact .chip { width: 32px; height: 32px; border-radius: 8px; }
  .row.compact .meta { display: none; }
  .row.compact .tname { font-size: 13px; }
</style>
