<script lang="ts">
  import Landing from './lib/Landing.svelte';
  import SiteHeader from './lib/SiteHeader.svelte';
  import SiteFooter from './lib/SiteFooter.svelte';
  import { findDoc } from './lib/docs-content';
  import { interceptLinks, parse, type Route } from './lib/router';

  const SITE = 'https://thre4dripper.github.io/tidefetch/';

  let route = $state<Route>(parse());

  // The docs bundle carries the markdown renderer and syntax highlighter, so
  // it is loaded on demand — the landing page never pays for it.
  let DocsView = $state<typeof import('./lib/Docs.svelte').default | null>(null);
  let docsError = $state(false);

  const doc = $derived(route.view === 'docs' ? findDoc(route.slug) : undefined);

  const title = $derived(
    route.view === 'docs'
      ? `${doc?.title ?? 'Documentation'} · Tidefetch`
      : 'Tidefetch — Terminal UI and Self-Hosted Web UI for aria2'
  );

  const description = $derived(
    route.view === 'docs'
      ? (doc?.description ??
          'Documentation for Tidefetch, a terminal UI and self-hosted web UI for the aria2 download manager.')
      : 'Tidefetch is a keyboard-first terminal UI (TUI) for the aria2 download manager, plus an optional self-hosted web UI for headless servers and homelabs.'
  );

  const canonical = $derived(
    route.view === 'docs' ? `${SITE}docs/${route.slug}` : SITE
  );

  $effect(() => {
    const onPop = () => (route = parse());
    window.addEventListener('popstate', onPop);
    const teardown = interceptLinks();
    return () => {
      window.removeEventListener('popstate', onPop);
      teardown();
    };
  });

  $effect(() => {
    if (route.view !== 'docs' || DocsView) return;
    import('./lib/Docs.svelte')
      .then((m) => (DocsView = m.default))
      .catch(() => (docsError = true));
  });

  $effect(() => {
    if (route.view !== 'landing') return;
    const anchor = route.anchor;
    requestAnimationFrame(() => {
      if (anchor) document.getElementById(anchor)?.scrollIntoView({ behavior: 'smooth' });
      else window.scrollTo(0, 0);
    });
  });
</script>

<svelte:head>
  <title>{title}</title>
  <meta name="description" content={description} />
  <link rel="canonical" href={canonical} />
  <meta property="og:title" content={title} />
  <meta property="og:description" content={description} />
  <meta property="og:url" content={canonical} />
  <meta name="twitter:title" content={title} />
  <meta name="twitter:description" content={description} />
</svelte:head>

<SiteHeader solid={route.view === 'docs'} />

{#if route.view === 'docs'}
  {#if DocsView}
    <DocsView slug={route.slug} anchor={route.anchor} />
  {:else if docsError}
    <div class="docs-fallback">
      <p>Documentation failed to load.</p>
      <a href="https://github.com/Thre4dripper/tidefetch/tree/main/docs">Read it on GitHub</a>
    </div>
  {:else}
    <div class="docs-fallback"><span class="spinner" aria-label="Loading documentation"></span></div>
  {/if}
{:else}
  <Landing />
{/if}

<SiteFooter />

<style>
  .docs-fallback {
    display: grid;
    place-items: center;
    gap: 12px;
    min-height: 60vh;
    padding-top: 80px;
    color: var(--text-dim);
  }
  .spinner {
    width: 26px;
    height: 26px;
    border: 2px solid var(--border-strong);
    border-top-color: var(--cyan);
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
