<script lang="ts">
  import { store } from '../store.svelte';
  import TaskRow from './TaskRow.svelte';
  import TideMark from './TideMark.svelte';

  const OVERSCAN = 6;

  let viewport: HTMLDivElement | undefined = $state();
  let scrollTop = $state(0);
  let viewH = $state(600);

  const ROW = $derived(store.prefs.compact ? 62 : 78); // px per row incl. gap
  const list = $derived(store.filtered);
  const start = $derived(Math.max(0, Math.floor(scrollTop / ROW) - OVERSCAN));
  const end = $derived(Math.min(list.length, Math.ceil((scrollTop + viewH) / ROW) + OVERSCAN));
  const visible = $derived(list.slice(start, end));

  const heading = $derived.by(() => {
    switch (store.filter) {
      case 'active': return 'ACTIVE DOWNLOADS';
      case 'waiting': return 'QUEUED DOWNLOADS';
      case 'stopped': return 'FINISHED DOWNLOADS';
      default: return 'ALL DOWNLOADS';
    }
  });

  function onScroll() {
    if (viewport) scrollTop = viewport.scrollTop;
  }

  $effect(() => {
    if (!viewport) return;
    const ro = new ResizeObserver(() => {
      viewH = viewport!.clientHeight;
    });
    ro.observe(viewport);
    return () => ro.disconnect();
  });
</script>

<div class="listhead">
  <span></span>
  <span class="eyebrow">{heading} · {list.length}</span>
  <span></span>
  <span class="eyebrow pace">SPEED</span>
  <span class="eyebrow">STATUS</span>
  <span></span>
</div>

<div class="viewport" bind:this={viewport} onscroll={onScroll}>
  {#if list.length === 0}
    <div class="empty">
      {#if store.tasks.length > 0}
        <div class="mark">⌕</div>
        <p>No downloads match this view</p>
        <button class="btn sm" onclick={() => { store.search = ''; store.filter = 'all'; }}>Clear filters</button>
      {:else}
        <div class="mark"><TideMark size={40} /></div>
        <p>Nothing here yet</p>
        <button class="btn primary" onclick={() => (store.addOpen = true)}>＋ Add your first download</button>
      {/if}
    </div>
  {:else}
    <div class="canvas" style="height:{list.length * ROW}px">
      <div class="window" style="transform: translateY({start * ROW}px)">
        {#each visible as task (task.gid)}
          <TaskRow {task} />
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .listhead {
    display: grid;
    grid-template-columns: var(--row-grid);
    gap: 14px;
    padding: 16px 32px 8px 30px;
  }
  .listhead .pace { text-align: right; }
  .viewport {
    flex: 1;
    overflow-y: auto;
    padding: 0 18px 18px;
  }
  .canvas { position: relative; }
  .window {
    display: flex;
    flex-direction: column;
    gap: 6px;
    will-change: transform;
  }
  .empty {
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    color: var(--text-dim);
  }
  .empty .mark {
    font-size: 46px;
    opacity: 0.5;
    background: var(--grad);
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
  }
  .empty p { margin: 0 0 6px; }
</style>
