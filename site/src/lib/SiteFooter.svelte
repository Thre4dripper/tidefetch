<script lang="ts">
  import { GitFork } from '@lucide/svelte';
  import Wordmark from './Wordmark.svelte';
  import { href } from './router';

  const repo = 'https://github.com/Thre4dripper/tidefetch';
</script>

<footer class="site-footer">
  <div class="footer-inner">
    <div class="footer-brand">
      <a class="wordmark" href={href('')}><Wordmark size={26} text={15.5} id="tf-ftr" /></a>
      <p>A terminal-first download manager powered by the aria2 engine.</p>
    </div>
    <div class="footer-col">
      <span>Product</span>
      <a href={href('#features')}>Features</a>
      <a href={href('#interfaces')}>Interfaces</a>
      <a href={href('#install')}>Install</a>
    </div>
    <div class="footer-col">
      <span>Docs</span>
      <a href={href('docs/getting-started')}>Getting Started</a>
      <a href={href('docs/installation')}>Installation</a>
      <a href={href('docs/homelab')}>Homelab Guide</a>
      <a href={href('docs/troubleshooting')}>Troubleshooting</a>
    </div>
    <div class="footer-col">
      <span>Project</span>
      <a href={repo} target="_blank" rel="noreferrer">GitHub</a>
      <a href={`${repo}/releases`} target="_blank" rel="noreferrer">Releases</a>
      <a href={`${repo}/issues`} target="_blank" rel="noreferrer">Issues</a>
      <a href={`${repo}/blob/main/LICENSE`} target="_blank" rel="noreferrer">MIT License</a>
    </div>
  </div>
  <div class="footer-base">
    <span>MIT licensed. aria2 is a separate GPL-2.0-or-later project spoken to over JSON-RPC.</span>
    <a href={repo} target="_blank" rel="noreferrer" aria-label="Tidefetch on GitHub"><GitFork size={16} /></a>
  </div>
</footer>
