<script lang="ts">
  // Bar chart used by the metrics band and transfer detail (mock-style).
  let {
    points,
    height = 44,
    color = 'var(--accent)',
    max: maxOverride
  }: { points: number[]; height?: number; color?: string; max?: number } = $props();

  const BARS = 36;
  const bars = $derived.by(() => {
    const tail = points.slice(-BARS);
    const max = maxOverride ?? Math.max(...tail, 1);
    return tail.map((p) => Math.max(4, Math.round((p / max) * 100)));
  });
</script>

<div class="bars" style="height:{height}px" aria-hidden="true">
  {#each bars as h, i (i)}
    <i style="height:{h}%;background:{color}"></i>
  {/each}
</div>

<style>
  .bars {
    display: flex;
    align-items: flex-end;
    gap: 3px;
    width: 100%;
  }
  .bars i {
    flex: 1;
    min-width: 3px;
    max-width: 12px;
    border-radius: 2px 2px 1px 1px;
    opacity: 0.85;
    transition: height 0.35s ease;
  }
</style>
