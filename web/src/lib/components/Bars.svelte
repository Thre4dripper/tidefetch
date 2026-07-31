<script lang="ts">
  // Throughput bar chart. Renders a baseline axis at all times so an idle
  // series reads as "no activity" rather than a loading indicator.
  let {
    points,
    height = 44,
    color = 'var(--accent)',
    max: maxOverride,
    label = 'No transfer activity'
  }: {
    points: number[];
    height?: number;
    color?: string;
    max?: number;
    label?: string;
  } = $props();

  const BARS = 36;

  const tail = $derived(points.slice(-BARS));
  const rawPeak = $derived(Math.max(...tail, 0));
  const peak = $derived(maxOverride ?? rawPeak * 1.15);
  const idle = $derived(rawPeak <= 0);
  const bars = $derived(tail.map((p) => (peak > 0 ? Math.max(2, (p / peak) * 100) : 0)));
</script>

<div class="chart" style="height:{height}px" class:idle>
  {#if idle}
    <span class="idlelabel">{label}</span>
  {:else}
    <div class="bars">
      {#each bars as h, i (i)}
        <i style="height:{h}%;background:{color}"></i>
      {/each}
    </div>
  {/if}
</div>

<style>
  .chart {
    position: relative;
    display: flex;
    align-items: flex-end;
    width: 100%;
    min-width: 0;
    /* Persistent axis: makes this read as a chart, never as a progress bar. */
    border-bottom: 1px solid var(--border);
  }
  .bars {
    display: flex;
    align-items: flex-end;
    gap: 3px;
    width: 100%;
    height: 100%;
  }
  .bars i {
    flex: 1;
    min-width: 3px;
    max-width: 12px;
    border-radius: 2px 2px 0 0;
    opacity: 0.85;
    transition: height 0.35s ease;
  }
  .idlelabel {
    width: 100%;
    padding-bottom: 6px;
    color: var(--text-faint);
    font-size: 10.5px;
    text-align: center;
    letter-spacing: 0.02em;
  }
</style>
