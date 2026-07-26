<script lang="ts">
  import { ArrowLeft, ArrowRight, BookOpen, ChevronRight, GitFork, Menu, X } from '@lucide/svelte';
  import { adjacentDocs, docSections, findDoc } from './docs-content';
  import { renderDoc } from './markdown';

  let { slug, anchor = '' }: { slug: string; anchor?: string } = $props();

  const repo = 'https://github.com/Thre4dripper/tidefetch';
  let navOpen = $state(false);
  let contentEl: HTMLElement | undefined = $state();
  let activeHeading = $state('');

  const doc = $derived(findDoc(slug) ?? findDoc('getting-started')!);
  const rendered = $derived(renderDoc(doc.source));
  const neighbors = $derived(adjacentDocs(doc.slug));

  // After each render: wire copy buttons, jump to anchor, observe headings.
  $effect(() => {
    void rendered;
    navOpen = false;
    const root = contentEl;
    if (!root) return;

    for (const button of root.querySelectorAll<HTMLButtonElement>('.copy-code')) {
      button.onclick = async () => {
        const code = button.closest('.codeblock')?.querySelector('code')?.textContent ?? '';
        await navigator.clipboard.writeText(code);
        button.textContent = 'Copied';
        setTimeout(() => (button.textContent = 'Copy'), 1400);
      };
    }

    if (anchor) {
      document.getElementById(anchor)?.scrollIntoView({ block: 'start' });
    } else {
      window.scrollTo(0, 0);
    }

    const headings = [...root.querySelectorAll<HTMLElement>('h2[id], h3[id]')];
    activeHeading = anchor || headings[0]?.id || '';
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) activeHeading = (entry.target as HTMLElement).id;
        }
      },
      { rootMargin: '-72px 0px -70% 0px' }
    );
    for (const heading of headings) observer.observe(heading);
    return () => observer.disconnect();
  });
</script>

<div class="docs">
  <button class="docs-nav-toggle" type="button" onclick={() => (navOpen = !navOpen)} aria-expanded={navOpen}>
    {#if navOpen}<X size={15} />{:else}<Menu size={15} />{/if}
    {doc.title}
  </button>

  <aside class="docs-nav" class:open={navOpen} aria-label="Documentation">
    <a class="docs-home" href="#/docs/getting-started"><BookOpen size={15} /> Documentation</a>
    {#each docSections as section (section.label)}
      <div class="nav-section">
        <span class="nav-label">{section.label}</span>
        {#each section.pages as p (p.slug)}
          <a class="nav-item" class:active={p.slug === doc.slug} href={`#/docs/${p.slug}`}>{p.title}</a>
        {/each}
      </div>
    {/each}
    <a class="nav-github" href={repo} target="_blank" rel="noreferrer"><GitFork size={14} /> Edit on GitHub</a>
  </aside>

  <article class="docs-body">
    <nav class="crumbs" aria-label="Breadcrumb">
      <a href="#/">Tidefetch</a>
      <ChevronRight size={12} />
      <a href="#/docs/getting-started">Docs</a>
      <ChevronRight size={12} />
      <span>{doc.title}</span>
    </nav>
    <p class="doc-description">{doc.description}</p>
    <div class="doc-content" bind:this={contentEl}>
      <!-- eslint-disable-next-line svelte/no-at-html-tags — trusted repository markdown -->
      {@html rendered.html}
    </div>

    <footer class="doc-pager">
      {#if neighbors.prev}
        <a class="pager prev" href={`#/docs/${neighbors.prev.slug}`}>
          <span><ArrowLeft size={13} /> Previous</span>
          <strong>{neighbors.prev.title}</strong>
        </a>
      {:else}<span></span>{/if}
      {#if neighbors.next}
        <a class="pager next" href={`#/docs/${neighbors.next.slug}`}>
          <span>Next <ArrowRight size={13} /></span>
          <strong>{neighbors.next.title}</strong>
        </a>
      {/if}
    </footer>
  </article>

  <aside class="docs-toc" aria-label="On this page">
    {#if rendered.toc.length > 1}
      <span class="toc-label">On this page</span>
      {#each rendered.toc as entry (entry.id)}
        <a
          class="toc-item"
          class:sub={entry.level === 3}
          class:active={entry.id === activeHeading}
          href={`#/docs/${doc.slug}#${entry.id}`}>{entry.text}</a>
      {/each}
    {/if}
  </aside>
</div>
