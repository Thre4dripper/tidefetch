<script lang="ts">
  import { GitFork } from '@lucide/svelte';

  const repo = 'https://github.com/Thre4dripper/tidefetch';
</script>

<footer class="site-footer">
  <div class="footer-inner">
    <div class="footer-brand">
      <a class="wordmark" href="#/"><span>⬡</span>Tidefetch</a>
      <p>An independent open-source project powered by the aria2 engine.</p>
    </div>
    <div class="footer-col">
      <span>Product</span>
      <a href="#/#features">Features</a>
      <a href="#/#interfaces">Interfaces</a>
      <a href="#/#install">Install</a>
    </div>
    <div class="footer-col">
      <span>Docs</span>
      <a href="#/docs/getting-started">Getting Started</a>
      <a href="#/docs/installation">Installation</a>
      <a href="#/docs/homelab">Homelab Guide</a>
      <a href="#/docs/troubleshooting">Troubleshooting</a>
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
