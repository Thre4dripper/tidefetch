// The documentation shown on the site is the repository's docs/ folder,
// imported at build time so there is exactly one source of truth.
import manifest from '../docs-manifest.json';

const sources = import.meta.glob('../../../docs/**/*.md', {
  query: '?raw',
  import: 'default',
  eager: true
}) as Record<string, string>;

export type DocPage = {
  slug: string;
  title: string;
  description: string;
  source: string;
};

export type DocSection = { label: string; pages: DocPage[] };

function sourceFor(file: string): string {
  const key = `../../../docs/${file}`;
  return sources[key] ?? `# Not found\n\nMissing documentation file: \`docs/${file}\`.`;
}

// The manifest is shared with scripts/prerender.mjs so the static pages and the
// client app can never disagree about slugs, titles or descriptions.
export const docSections: DocSection[] = manifest.map((section) => ({
  label: section.label,
  pages: section.pages.map((p) => ({
    slug: p.slug,
    title: p.title,
    description: p.description,
    source: sourceFor(p.file)
  }))
}));

export const docPages: DocPage[] = docSections.flatMap((section) => section.pages);

export function findDoc(slug: string): DocPage | undefined {
  return docPages.find((p) => p.slug === slug);
}

export function adjacentDocs(slug: string): { prev?: DocPage; next?: DocPage } {
  const index = docPages.findIndex((p) => p.slug === slug);
  if (index === -1) return {};
  return { prev: docPages[index - 1], next: docPages[index + 1] };
}
