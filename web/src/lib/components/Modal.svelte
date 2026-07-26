<script lang="ts">
  import type { Snippet } from 'svelte';

  let { title, onclose, children, width = 560 }: {
    title: string;
    onclose: () => void;
    children: Snippet;
    width?: number;
  } = $props();

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape') onclose();
  }
</script>

<svelte:window onkeydown={onKey} />

<div class="backdrop" onclick={(e) => e.target === e.currentTarget && onclose()} role="presentation">
  <div class="modal card" style="max-width:{width}px" role="dialog" aria-label={title}>
    <div class="mhead">
      <h3>{title}</h3>
      <button class="btn icon" onclick={onclose} aria-label="Close">✕</button>
    </div>
    <div class="mbody">
      {@render children()}
    </div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(4, 8, 14, 0.66);
    backdrop-filter: blur(4px);
    display: grid;
    place-items: center;
    z-index: 50;
    animation: fadeIn 0.15s ease;
  }
  .modal {
    width: calc(100% - 40px);
    max-height: calc(100% - 60px);
    display: flex;
    flex-direction: column;
    background: var(--panel-solid);
    box-shadow: 0 24px 80px rgba(0, 0, 0, 0.55);
    animation: fadeUp 0.18s ease;
  }
  .mhead {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 18px 12px;
    border-bottom: 1px solid var(--border);
  }
  h3 { margin: 0; font-size: 16px; }
  .mbody {
    padding: 16px 18px;
    overflow-y: auto;
  }
</style>
