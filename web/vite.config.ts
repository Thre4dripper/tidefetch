import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    target: 'es2022'
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8210',
        ws: true
      }
    }
  }
});
