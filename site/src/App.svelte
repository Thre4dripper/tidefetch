<script lang="ts">
  import Landing from './lib/Landing.svelte';
  import Docs from './lib/Docs.svelte';
  import SiteHeader from './lib/SiteHeader.svelte';
  import SiteFooter from './lib/SiteFooter.svelte';

  type Route = { view: 'landing'; anchor: string } | { view: 'docs'; slug: string; anchor: string };

  function parse(): Route {
    const raw = window.location.hash.replace(/^#\/?/, '');
    if (raw.startsWith('docs')) {
      const rest = raw.slice(4).replace(/^\//, '');
      const [slug, anchor = ''] = rest.split('#');
      return { view: 'docs', slug: slug || 'getting-started', anchor };
    }
    return { view: 'landing', anchor: raw.startsWith('#') ? raw.slice(1) : '' };
  }

  let route = $state<Route>(parse());

  $effect(() => {
    const onHash = () => (route = parse());
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  });

  // Landing anchors like #/#install are invisible to native scrolling.
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
  <title>
    {route.view === 'docs' ? 'Docs · Tidefetch' : 'Tidefetch — aria2, beautifully controlled'}
  </title>
</svelte:head>

<SiteHeader solid={route.view === 'docs'} />

{#if route.view === 'docs'}
  <Docs slug={route.slug} anchor={route.anchor} />
{:else}
  <Landing />
{/if}

<SiteFooter />
