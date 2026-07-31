// Emits a real, crawlable HTML file for every route.
//
// The site is a client-rendered SPA, so without this step search engines see an
// empty <div id="app"> and index nothing. Each generated page carries its own
// title, description, canonical URL and the fully rendered documentation body,
// then hands over to the SPA bundle on load.

import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { marked } from 'marked';
import { rewriteDocHref, slugifyHeading } from '../src/lib/doc-links.js';

const here = dirname(fileURLToPath(import.meta.url));
const siteDir = join(here, '..');
const repoDir = join(siteDir, '..');
const distDir = join(siteDir, 'dist');
const docsDir = join(repoDir, 'docs');

const SITE = 'https://thre4dripper.github.io/tidefetch/';
const BASE = '/tidefetch/';

const manifest = JSON.parse(await readFile(join(siteDir, 'src/docs-manifest.json'), 'utf8'));
const pages = manifest.flatMap((section) => section.pages);

const escapeHtml = (value) =>
  value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');

const slugify = (text) => slugifyHeading(text);

/** Shared with the client renderer so static and SPA links cannot diverge. */
function rewriteHref(href) {
  return rewriteDocHref(href, BASE);
}

const renderer = new marked.Renderer();
renderer.heading = function ({ tokens, depth }) {
  const text = this.parser.parseInline(tokens);
  const id = slugify(text);
  return `<h${depth} id="${id}">${text}</h${depth}>\n`;
};
renderer.link = function ({ href, title, tokens }) {
  const text = this.parser.parseInline(tokens);
  const titleAttr = title ? ` title="${escapeHtml(title)}"` : '';
  return `<a href="${escapeHtml(rewriteHref(href))}"${titleAttr}>${text}</a>`;
};

function head({ title, description, canonical }) {
  return `<title>${escapeHtml(title)}</title>
    <meta data-prerendered name="description" content="${escapeHtml(description)}" />
    <link data-prerendered rel="canonical" href="${escapeHtml(canonical)}" />
    <meta data-prerendered property="og:title" content="${escapeHtml(title)}" />
    <meta data-prerendered property="og:description" content="${escapeHtml(description)}" />
    <meta data-prerendered property="og:url" content="${escapeHtml(canonical)}" />
    <meta data-prerendered name="twitter:title" content="${escapeHtml(title)}" />
    <meta data-prerendered name="twitter:description" content="${escapeHtml(description)}" />`;
}

/**
 * Swap the tags that differ per route. The template's own copies are removed
 * first — stripping after insertion would delete the new tags instead, since
 * they appear earlier in the document.
 */
function applyHead(html, meta) {
  return html
    .replace(/\s*<meta[^>]*name="description"[^>]*>/g, '')
    .replace(/\s*<link[^>]*rel="canonical"[^>]*>/g, '')
    .replace(/\s*<meta[^>]*property="og:title"[^>]*>/g, '')
    .replace(/\s*<meta[^>]*property="og:description"[^>]*>/g, '')
    .replace(/\s*<meta[^>]*property="og:url"[^>]*>/g, '')
    .replace(/\s*<meta[^>]*name="twitter:title"[^>]*>/g, '')
    .replace(/\s*<meta[^>]*name="twitter:description"[^>]*>/g, '')
    .replace(/<title>[\s\S]*?<\/title>/, () => head(meta));
}

function inject(html, body) {
  return html.replace('<div id="app"></div>', `<div id="app">${body}</div>`);
}

function docNav(activeSlug) {
  const sections = manifest
    .map((section) => {
      const links = section.pages
        .map(
          (p) =>
            `<li><a href="${BASE}docs/${p.slug}"${
              p.slug === activeSlug ? ' aria-current="page"' : ''
            }>${escapeHtml(p.title)}</a></li>`
        )
        .join('');
      return `<li>${escapeHtml(section.label)}<ul>${links}</ul></li>`;
    })
    .join('');
  return `<nav aria-label="Documentation"><ul>${sections}</ul></nav>`;
}

const template = await readFile(join(distDir, 'index.html'), 'utf8');
const written = [];

// ── Documentation routes ─────────────────────────────────────────────────────
for (const page of pages) {
  const markdown = await readFile(join(docsDir, page.file), 'utf8');
  const content = marked.parse(markdown, { renderer, async: false });

  const body = `<main>${docNav(page.slug)}<article>${content}</article></main>`;
  const canonical = `${SITE}docs/${page.slug}`;

  const html = inject(
    applyHead(template, {
      title: `${page.title} · Tidefetch`,
      description: page.description,
      canonical
    }),
    body
  );

  const outDir = join(distDir, 'docs', page.slug);
  await mkdir(outDir, { recursive: true });
  await writeFile(join(outDir, 'index.html'), html);
  written.push(`docs/${page.slug}`);
}

// ── Landing route ────────────────────────────────────────────────────────────
// The template is already the landing page; give crawlers real copy inside #app
// instead of an empty mount point.
const landingBody = `<main>
  <h1>Tidefetch — the download manager that lives in your terminal</h1>
  <p>Tidefetch is a keyboard-first terminal UI (TUI) for the aria2 download engine, built for people who live in a shell. The same static binary also serves a self-hosted web UI, so headless servers and homelabs get a browser dashboard on demand.</p>
  <h2>A TUI for your laptop. A web UI for your server.</h2>
  <p>Run it as a terminal download manager on your own machine, or run <code>tidefetch serve</code> on a NAS, VPS or Raspberry Pi and manage the same queue from any browser on your network.</p>
  <h2>Install</h2>
  <pre><code>curl -fsSL https://thre4dripper.github.io/tidefetch/install.sh | sh</code></pre>
  <p>Also available via Homebrew, Docker, Helm and <code>go install</code>.</p>
  <h2>Features</h2>
  <ul>
    <li>Every aria2 protocol: HTTP(S), FTP, SFTP, BitTorrent, Metalink and magnet links</li>
    <li>Live telemetry: speed graphs, piece maps, per-task history and disk gauges</li>
    <li>One static Go binary — no runtime, no Electron</li>
    <li>Self-hosted web UI with bcrypt auth, strict CSP and WebSocket push</li>
    <li>Docker, Docker Swarm, Kubernetes, Unraid and bare-metal systemd deployments</li>
  </ul>
  <h2>Documentation</h2>
  ${docNav('')}
</main>`;

await writeFile(join(distDir, 'index.html'), inject(template, landingBody));
written.push('');

// ── SPA fallback for unknown deep links ──────────────────────────────────────
await writeFile(join(distDir, '404.html'), inject(template, landingBody));

// ── Sitemap ──────────────────────────────────────────────────────────────────
const urls = written
  .map(
    (route) =>
      `  <url>\n    <loc>${SITE}${route}</loc>\n    <priority>${route ? '0.8' : '1.0'}</priority>\n  </url>`
  )
  .join('\n');

await writeFile(
  join(distDir, 'sitemap.xml'),
  `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${urls}\n</urlset>\n`
);

console.log(`prerendered ${written.length} routes + sitemap.xml`);
