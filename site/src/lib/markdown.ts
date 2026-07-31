import { marked, type Tokens } from 'marked';
import { BASE } from './router';
import { rewriteDocHref, slugifyHeading } from './doc-links';
import { createHighlighterCoreSync, type HighlighterCore } from 'shiki/core';
import { createJavaScriptRegexEngine } from 'shiki/engine/javascript';
import bash from 'shiki/langs/bash.mjs';
import yaml from 'shiki/langs/yaml.mjs';
import json from 'shiki/langs/json.mjs';
import go from 'shiki/langs/go.mjs';
import ini from 'shiki/langs/ini.mjs';
import nginx from 'shiki/langs/nginx.mjs';
import docker from 'shiki/langs/docker.mjs';
import xml from 'shiki/langs/xml.mjs';
import powershell from 'shiki/langs/powershell.mjs';
import theme from 'shiki/themes/github-dark-default.mjs';

export type TocEntry = { id: string; text: string; level: number };
export type RenderedDoc = { html: string; toc: TocEntry[] };

const ALERT_KINDS: Record<string, { label: string; kind: string }> = {
  '[!NOTE]': { label: 'Note', kind: 'note' },
  '[!TIP]': { label: 'Tip', kind: 'tip' },
  '[!IMPORTANT]': { label: 'Important', kind: 'important' },
  '[!WARNING]': { label: 'Warning', kind: 'warning' },
  '[!CAUTION]': { label: 'Caution', kind: 'caution' }
};

// Shiki bundled synchronously so rendering stays a pure function.
const highlighter: HighlighterCore = createHighlighterCoreSync({
  themes: [theme],
  langs: [bash, yaml, json, go, ini, nginx, docker, xml, powershell],
  engine: createJavaScriptRegexEngine()
});

const LANG_ALIAS: Record<string, string> = {
  sh: 'bash',
  shell: 'bash',
  zsh: 'bash',
  console: 'bash',
  yml: 'yaml',
  dockerfile: 'docker',
  caddyfile: 'nginx',
  caddy: 'nginx',
  ps1: 'powershell',
  text: 'bash',
  '': 'bash'
};

const SUPPORTED = new Set(highlighter.getLoadedLanguages());

function slugify(text: string): string {
  return slugifyHeading(text);
}

function escapeHtml(value: string): string {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');
}

/** Rewrite repository-relative markdown links to embedded docs routes. */
function rewriteHref(href: string): string {
  return rewriteDocHref(href, BASE);
}

/** Render a markdown document into themed HTML plus a table of contents. */
export function renderDoc(source: string): RenderedDoc {
  const toc: TocEntry[] = [];
  const seen = new Map<string, number>();
  const renderer = new marked.Renderer();

  renderer.heading = ({ tokens, depth }: Tokens.Heading) => {
    const text = tokens
      .map((t) => ('text' in t ? (t as { text: string }).text : ''))
      .join('')
      .replace(/`/g, '');
    let id = slugify(text);
    const count = seen.get(id) ?? 0;
    seen.set(id, count + 1);
    if (count > 0) id = `${id}-${count}`;
    if (depth === 2 || depth === 3) toc.push({ id, text, level: depth });
    const inner = marked.Parser.parseInline(tokens, { renderer });
    return `<h${depth} id="${id}">${inner}</h${depth}>\n`;
  };

  renderer.link = ({ href, title, tokens }: Tokens.Link) => {
    const target = rewriteHref(href);
    const inner = marked.Parser.parseInline(tokens, { renderer });
    const external = /^https?:/.test(target);
    const attrs = [
      `href="${target}"`,
      title ? `title="${escapeHtml(title)}"` : '',
      external ? 'target="_blank" rel="noreferrer"' : ''
    ]
      .filter(Boolean)
      .join(' ');
    return `<a ${attrs}>${inner}</a>`;
  };

  renderer.code = ({ text, lang }: Tokens.Code) => {
    const declared = (lang ?? '').trim().toLowerCase();
    const resolved = LANG_ALIAS[declared] ?? declared;
    const language = SUPPORTED.has(resolved) ? resolved : 'bash';
    const label = declared || 'sh';

    let body: string;
    try {
      body = highlighter.codeToHtml(text, {
        lang: language,
        theme: 'github-dark-default'
      });
    } catch {
      body = `<pre class="shiki"><code>${escapeHtml(text)}</code></pre>`;
    }

    return (
      `<div class="codeblock"><div class="codeblock-bar"><span>${escapeHtml(label)}</span>` +
      `<button type="button" class="copy-code" aria-label="Copy code">Copy</button></div>` +
      body +
      `</div>`
    );
  };

  renderer.blockquote = ({ tokens }: Tokens.Blockquote) => {
    const body = marked.Parser.parse(tokens, { renderer });
    for (const [marker, meta] of Object.entries(ALERT_KINDS)) {
      const escaped = marker.replace('[', '\\[').replace(']', '\\]');
      const pattern = new RegExp(`^<p>${escaped}\\s*(<br\\s*/?>)?\\s*`);
      if (pattern.test(body)) {
        const inner = body.replace(pattern, '<p>').replace(/<p>\s*<\/p>/, '');
        return `<div class="alert ${meta.kind}"><span class="alert-label">${meta.label}</span>${inner}</div>\n`;
      }
    }
    return `<blockquote>${body}</blockquote>\n`;
  };

  const html = marked.parse(source, { renderer, async: false }) as string;
  return { html, toc };
}
