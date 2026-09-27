// Shared by the SPA renderer (markdown.ts) and the static prerenderer
// (scripts/prerender.mjs). Kept as plain JS so Node can import it directly —
// if these two ever disagree, the prerendered links and the client-side links
// point at different places.

const REPO_BLOB = 'https://github.com/Thre4dripper/tidefetch/blob/main/';

/**
 * Rewrite a repository-relative markdown link to a site route.
 *
 * @param {string} href raw href from the markdown source
 * @param {string} base site base path, "/" in production
 * @returns {string}
 */
export function rewriteDocHref(href, base) {
  if (/^(https?:|mailto:|#)/.test(href)) return href;

  // A fragment such as configuration.md#web-server-flags stays attached to
  // the rewritten route.
  const [path, fragment] = href.split('#', 2);
  const suffix = fragment ? `#${fragment}` : '';

  const clean = path.replace(/^(\.\.?\/)+/, '');
  if (clean.endsWith('.md')) {
    const slug = clean
      .replace(/^docs\//, '')
      .replace(/\.md$/, '')
      .replace(/(^|\/)index$/, '$1getting-started');
    return `${base}docs/${slug || 'getting-started'}${suffix}`;
  }

  return `${REPO_BLOB}${clean}${suffix}`;
}

/** Slugify heading text into an anchor id. Must match on both renderers. */
export function slugifyHeading(text) {
  return text
    .toLowerCase()
    .replace(/<[^>]+>/g, '')
    .replace(/[^a-z0-9\s-]/g, '')
    .trim()
    .replace(/\s+/g, '-');
}
