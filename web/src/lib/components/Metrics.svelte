<script lang="ts">
  import { ArrowDownToLine, ArrowUpFromLine, Gauge } from '@lucide/svelte';
  import { store } from '../store.svelte';
  import { fmtBytes } from '../format';
  import Bars from './Bars.svelte';

  function split(n: number): { value: string; unit: string } {
    const [value, unit] = fmtBytes(n).split(' ');
    return { value, unit };
  }

  const down = $derived(split(store.stat.downSpeed));
  const up = $derived(split(store.stat.upSpeed));
  const session = $derived(split(store.stat.sessionDown));
</script>

<section class="metrics card" aria-label="Transfer metrics">
  <div class="metric">
    <span class="eyebrow"><ArrowDownToLine size={12} /> DOWN</span>
    <strong>{down.value} <small>{down.unit}/s</small></strong>
  </div>
  <div class="metric">
    <span class="eyebrow"><ArrowUpFromLine size={12} /> UP</span>
    <strong>{up.value} <small>{up.unit}/s</small></strong>
  </div>
  <div class="metric">
    <span class="eyebrow"><Gauge size={12} /> SESSION</span>
    <strong>{session.value} <small>{session.unit}</small></strong>
  </div>
  <div class="chart">
    <span class="charthead">THROUGHPUT · LAST 36 SAMPLES</span>
    <Bars points={store.downHistory} height={42} color="var(--accent)" />
  </div>
</section>

<style>
  .metrics {
    display: grid;
    grid-template-columns: 118px 108px 118px 1fr;
    align-items: stretch;
    gap: 16px;
    margin: 14px 18px 4px;
    padding: 15px 18px;
    background: rgba(255, 255, 255, 0.022);
  }
  .metric {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    gap: 10px;
    border-right: 1px solid var(--border);
    padding-right: 16px;
  }
  .metric .eyebrow {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .metric strong {
    font-size: 21px;
    font-weight: 700;
    letter-spacing: -0.01em;
    font-variant-numeric: tabular-nums;
  }
  .metric small {
    font-size: 10.5px;
    font-weight: 600;
    color: var(--text-dim);
  }
  .chart {
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
    gap: 6px;
    min-width: 0;
    padding: 2px 0 1px;
  }
  .charthead {
    color: var(--text-faint);
    font-family: var(--mono);
    font-size: 8.5px;
    letter-spacing: 0.1em;
  }

  @media (max-width: 1180px) {
    .metrics { grid-template-columns: 112px 102px 112px 1fr; gap: 12px; }
  }
  @media (max-width: 700px) {
    .metrics { grid-template-columns: 1fr 1fr 1fr; }
    .chart { display: none; }
    .metric:last-of-type { border-right: none; padding-right: 0; }
  }
</style>
