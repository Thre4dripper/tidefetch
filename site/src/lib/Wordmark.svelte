<script lang="ts">
  import TideMark from './TideMark.svelte';

  let {
    size = 26,
    text = 15.5,
    animated = true,
    id = 'tf'
  }: { size?: number; text?: number; animated?: boolean; id?: string } = $props();
</script>

<span class="lockup" style="--wm-size:{text}px">
  <TideMark {size} {animated} {id} />
  <span class="wm"><span class="tide">tide</span><span class="fetch">Fetch</span></span>
</span>

<style>
  .lockup {
    display: inline-flex;
    align-items: center;
    gap: 9px;
    font-size: var(--wm-size);
    line-height: 1;
    white-space: nowrap;
  }
  .wm { display: inline-flex; align-items: baseline; }
  /* "tide" — monospace roots, gradient swell */
  .tide {
    font-family: var(--mono);
    font-weight: 700;
    letter-spacing: -0.02em;
    background: linear-gradient(100deg, #5ed8e7 5%, #b8ff3d 95%);
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
  }
  /* "Fetch" — light sans counterweight */
  .fetch {
    font-family: var(--font);
    font-weight: 300;
    letter-spacing: 0.01em;
    color: var(--text);
  }
</style>
