<script lang="ts">
  import { Activity, CircleCheck, Clock3, Download, History, Power, Settings2 } from '@lucide/svelte';
  import { store, type Filter } from '../store.svelte';
  import { fmtBytes, fmtSpeed } from '../format';
  import Sparkline from './Sparkline.svelte';

  const filters = [
    { id: 'all' as Filter, label: 'Downloads', icon: Download },
    { id: 'active' as Filter, label: 'Active', icon: Activity },
    { id: 'waiting' as Filter, label: 'Queued', icon: Clock3 },
    { id: 'stopped' as Filter, label: 'Finished', icon: CircleCheck }
  ];

  function pick(f: Filter) {
    store.view = 'downloads';
    store.filter = f;
  }

  const disk = $derived.by(() => {
    const { diskFree, diskTotal } = store.stat;
    if (!diskTotal) return null;
    const used = diskTotal - diskFree;
    return { pct: Math.min(100, Math.round((used / diskTotal) * 100)), free: diskFree };
  });
</script>

<aside class="card">
  <div class="brand">
    <span class="mark">⬡</span>
    <span class="name">Tidefetch</span>
  </div>

  <nav>
    {#each filters as f (f.id)}
      <button
        class="item"
        class:on={store.view === 'downloads' && store.filter === f.id}
        onclick={() => pick(f.id)}
      >
        <f.icon size={15} strokeWidth={2} />
        {f.label}
        <span class="count">{store.counts[f.id]}</span>
      </button>
    {/each}
    <button
      class="item"
      class:on={store.view === 'history'}
      onclick={() => (store.view = 'history')}
    >
      <History size={15} strokeWidth={2} />
      History
    </button>
  </nav>

  <div class="spacer"></div>

  <div class="net card">
    <div class="row">
      <span class="dot down"></span>
      <span class="dim">Down</span>
      <b class="mono">{fmtSpeed(store.stat.downSpeed)}</b>
    </div>
    <Sparkline points={store.downHistory} color="#5ed8e7" />
    <div class="row up-row">
      <span class="dot up"></span>
      <span class="dim">Up</span>
      <b class="mono">{fmtSpeed(store.stat.upSpeed)}</b>
    </div>
    <Sparkline points={store.upHistory} color="#8c82ff" />
  </div>

  {#if disk}
    <div class="storage">
      <div class="srow">
        <span class="eyebrow">STORAGE</span>
        <b class="mono">{disk.pct}%</b>
      </div>
      <div class="sbar"><i style="width:{disk.pct}%"></i></div>
      <span class="faint">{fmtBytes(disk.free)} free</span>
    </div>
  {/if}

  <div class="foot">
    <span class="status" class:down={!store.connected}>
      ● aria2 {store.aria2}
    </span>
    <span class="footbtns">
      <button class="btn icon" onclick={() => (store.settingsOpen = true)} title="Settings" aria-label="Settings">
        <Settings2 size={14} />
      </button>
      {#if store.authEnabled}
        <button class="btn icon" onclick={() => store.logout()} title="Sign out" aria-label="Sign out">
          <Power size={14} />
        </button>
      {/if}
    </span>
  </div>
</aside>

<style>
  aside {
    width: 226px;
    margin: 12px 0 12px 12px;
    padding: 16px 12px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    flex-shrink: 0;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 9px;
    padding: 4px 10px 18px;
  }
  .mark {
    display: grid;
    place-items: center;
    width: 26px;
    height: 26px;
    font-size: 15px;
    color: #10150a;
    background: var(--signal);
    border-radius: 7px;
  }
  .name {
    font-size: 15.5px;
    font-weight: 750;
    letter-spacing: 0.2px;
  }
  nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 9px 12px;
    border-radius: var(--radius-sm);
    color: var(--text-dim);
    font-weight: 550;
    text-align: left;
    transition: background 0.12s, color 0.12s;
  }
  .item:hover { background: rgba(255, 255, 255, 0.05); color: var(--text); }
  .item.on {
    background: rgba(255, 255, 255, 0.06);
    color: var(--text);
  }
  .count {
    margin-left: auto;
    font-size: 11px;
    font-family: var(--mono);
    color: var(--text-faint);
  }
  .item.on .count { color: var(--accent); }
  .spacer { flex: 1; }
  .net {
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    background: rgba(0, 0, 0, 0.22);
  }
  .row {
    display: flex;
    align-items: center;
    gap: 7px;
    font-size: 12.5px;
  }
  .row b { margin-left: auto; font-size: 12.5px; }
  .up-row { margin-top: 6px; }
  .dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
  }
  .dot.down { background: var(--accent); }
  .dot.up { background: var(--up); }
  .storage {
    display: flex;
    flex-direction: column;
    gap: 7px;
    padding: 12px 10px 4px;
    font-size: 11px;
  }
  .srow {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .srow b { font-size: 11.5px; color: var(--text-dim); }
  .sbar {
    height: 4px;
    border-radius: 3px;
    background: rgba(255, 255, 255, 0.07);
    overflow: hidden;
  }
  .sbar i {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: var(--accent);
    transition: width 0.5s ease;
  }
  .foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 6px 2px;
    gap: 6px;
  }
  .footbtns { display: flex; gap: 6px; align-items: center; }
  .status {
    font-size: 11.5px;
    color: var(--ok);
  }
  .status.down { color: var(--err); }
</style>
