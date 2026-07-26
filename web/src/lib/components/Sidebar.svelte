<script lang="ts">
  import { store, type Filter } from '../store.svelte';
  import { fmtSpeed } from '../format';
  import Sparkline from './Sparkline.svelte';

  const filters: { id: Filter; label: string; icon: string }[] = [
    { id: 'all', label: 'All', icon: '◎' },
    { id: 'active', label: 'Active', icon: '⇣' },
    { id: 'waiting', label: 'Queued', icon: '𝍪' },
    { id: 'stopped', label: 'Finished', icon: '✓' }
  ];

  function pick(f: Filter) {
    store.view = 'downloads';
    store.filter = f;
  }
</script>

<aside class="card">
  <div class="brand">
    <span class="mark">⬡</span>
    <span class="name">tidefetch</span>
  </div>

  <nav>
    {#each filters as f (f.id)}
      <button
        class="item"
        class:on={store.view === 'downloads' && store.filter === f.id}
        onclick={() => pick(f.id)}
      >
        <span class="icon">{f.icon}</span>
        {f.label}
        <span class="count">{store.counts[f.id]}</span>
      </button>
    {/each}
    <button
      class="item"
      class:on={store.view === 'history'}
      onclick={() => (store.view = 'history')}
    >
      <span class="icon">⟲</span>
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
    <Sparkline points={store.downHistory} color="#34d8c3" />
    <div class="row up-row">
      <span class="dot up"></span>
      <span class="dim">Up</span>
      <b class="mono">{fmtSpeed(store.stat.upSpeed)}</b>
    </div>
    <Sparkline points={store.upHistory} color="#c084fc" />
  </div>

  <div class="foot">
    <span class="status" class:down={!store.connected}>
      ● aria2 {store.aria2}
    </span>
    <span class="footbtns">
      <button class="btn sm" onclick={() => (store.settingsOpen = true)}>⚙ Settings</button>
      {#if store.authEnabled}
        <button class="btn icon" onclick={() => store.logout()} title="Sign out" aria-label="Sign out">⏻</button>
      {/if}
    </span>
  </div>
</aside>

<style>
  aside {
    width: 230px;
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
    padding: 4px 10px 16px;
  }
  .mark {
    font-size: 22px;
    background: var(--grad);
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
  }
  .name {
    font-size: 17px;
    font-weight: 700;
    letter-spacing: 0.4px;
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
    font-weight: 500;
    text-align: left;
    transition: background 0.12s, color 0.12s;
  }
  .item:hover { background: rgba(255, 255, 255, 0.05); color: var(--text); }
  .item.on {
    background: rgba(52, 216, 195, 0.10);
    color: var(--accent);
  }
  .icon { width: 18px; text-align: center; }
  .count {
    margin-left: auto;
    font-size: 11.5px;
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
    background: rgba(0, 0, 0, 0.25);
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
