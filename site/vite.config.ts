import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

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

export default defineConfig({
  // Absolute base: history routing needs real, server-resolvable URLs.
  // GitHub Pages serves this project at /tidefetch/.
  base: '/tidefetch/',
  plugins: [svelte(), installScripts()],
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