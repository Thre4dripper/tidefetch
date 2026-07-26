<script lang="ts">
  import { Image, MonitorUp } from '@lucide/svelte';
  import type { ImageMedia } from './media';
  import ProductMock from './ProductMock.svelte';

  let { item, class: className = '' }: { item: ImageMedia; class?: string } = $props();
</script>

<figure class={`media-frame ${item.tone} ${className}`}>
  {#if item.enabled}
    <img src={item.src} alt={item.alt} loading="lazy" decoding="async" />
  {:else if item.tone === 'dashboard'}
    <ProductMock compact />
  {:else if item.tone === 'terminal'}
    <div class="terminal-placeholder" aria-label="Terminal screenshot placeholder">
      <div class="terminal-bar"><span></span><span></span><span></span><b>tidefetch · downloads</b></div>
      <pre><em>⬡ tidefetch</em>  ▼ 27.1 MB/s  ▲ 1.2 MB/s    ▁▂▃▅▇█▆▅▃   <b>● aria2 1.37.0</b>

╭─ Downloads · 4 ─────────────────────────────╮ ╭─ Selected ─────────────╮
│ ┃ ⇣ debian-13.1.0-amd64-netinst.iso  ACTIVE │ │ debian-13.1.0          │
│   ████████████████████████████████░░  92.7% │ │                        │
│   18.4 MB/s · eta 3s · 16 connections       │ │  ▼ 18.4 MB/s          │
│                                             │ │  ▁▂▃▅▇▆██▇▅▆█        │
│   ⇣ archive-footage-04.tar.zst       ACTIVE │ │                        │
│   ██████████████░░░░░░░░░░░░░░░░░░  41.0% │ │  pieces  2,714 / 2,948│
│   8.7 MB/s · eta 36m                        │ │  disk    354 GB free  │
╰─────────────────────────────────────────────╯ ╰────────────────────────╯</pre>
    </div>
  {:else}
    <div class="capture-placeholder">
      <div class="capture-icon"><Image size={22} /></div>
      <span>MEDIA SLOT</span>
      <strong>{item.label}</strong>
      <p>{item.dimensions}</p>
      <code>{item.src.replace('/media/', '')}</code>
    </div>
  {/if}
  {#if !item.enabled}
    <figcaption><MonitorUp size={13} /> Replace-ready placeholder · {item.dimensions}</figcaption>
  {/if}
</figure>