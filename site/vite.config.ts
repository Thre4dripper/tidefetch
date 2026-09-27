import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { resolveSite } from './site.config.mjs';

const site = resolveSite();

/**
 * Publishes the repo's install scripts at the site root so the documented
 * one-liners resolve, keeping scripts/ as the single source of truth.
 */
function installScripts() {
  const names = ['install.sh', 'install.ps1'];
  return {
    name: 'tidefetch-install-scripts',
    generateBundle() {
      for (const name of names) {
        this.emitFile({
          type: 'asset',
          fileName: name,
          source: readFileSync(fileURLToPath(new URL(`../scripts/${name}`, import.meta.url)), 'utf8')
        });
      }
    }
  };
}

/**
 * index.html carries absolute URLs (canonical, social card, JSON-LD) that
 * must point at wherever this build is published. They are written as the
 * __SITE_URL__ token and filled in here; Vite's own %ENV% replacement only
 * sees VITE_-prefixed variables, and the same token is defined for scripts.
 */
function siteUrl() {
  return {
    name: 'tidefetch-site-url',
    transformIndexHtml: {
      order: 'pre' as const,
      handler(html: string) {
        return html.replaceAll('__SITE_URL__', site.url);
      }
    }
  };
}

export default defineConfig({
  // Absolute base: history routing needs real, server-resolvable URLs.
  // Derived from the publish URL in site.config.mjs ("/" in production).
  base: site.base,
  define: {
    __SITE_URL__: JSON.stringify(site.url)
  },
  plugins: [svelte(), installScripts(), siteUrl()],
  server: {
    fs: {
      allow: ['..']
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    target: 'es2022'
  }
});
