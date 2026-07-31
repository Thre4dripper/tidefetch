<script lang="ts">
  import { ArrowRight, BookOpen, GitFork, Menu, X } from '@lucide/svelte';
  import Wordmark from './Wordmark.svelte';
  import { href } from './router';

  let { solid = false }: { solid?: boolean } = $props();

  const repo = 'https://github.com/Thre4dripper/tidefetch';
  let menuOpen = $state(false);
  let scrolled = $state(false);

  $effect(() => {
    const onScroll = () => (scrolled = window.scrollY > 8);
    onScroll();
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  });
</script>

<header class="site-header" class:scrolled class:solid>
  <div class="header-inner">
    <a class="wordmark" href={href('')} aria-label="Tidefetch home"><Wordmark size={26} text={15.5} id="tf-hdr" /></a>
    <nav class:open={menuOpen} aria-label="Main navigation">
      <a href={href('#features')} onclick={() => (menuOpen = false)}>Features</a>
      <a href={href('#interfaces')} onclick={() => (menuOpen = false)}>Interfaces</a>
      <a href={href('#deploy')} onclick={() => (menuOpen = false)}>Deploy</a>
      <a href={href('docs/getting-started')} onclick={() => (menuOpen = false)}><BookOpen size={14} /> Docs</a>
    </nav>
    <div class="header-actions">
      <a class="gh" href={repo} target="_blank" rel="noreferrer" aria-label="GitHub repository">
        <GitFork size={15} />
        <span>GitHub</span>
      </a>
      <a class="cta" href={href('#install')}>Install <ArrowRight size={13} /></a>
      <button
        class="menu-toggle"
        type="button"
        aria-label="Toggle menu"
        aria-expanded={menuOpen}
        onclick={() => (menuOpen = !menuOpen)}>
        {#if menuOpen}<X size={18} />{:else}<Menu size={18} />{/if}
      </button>
    </div>
  </div>
</header>
