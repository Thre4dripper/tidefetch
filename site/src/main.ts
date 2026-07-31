import { mount } from 'svelte';
import './app.css';
import App from './App.svelte';

const target = document.getElementById('app')!;

// Prerendered markup is for crawlers and the first paint; drop it before
// mounting so the client render does not append a second copy.
target.innerHTML = '';

// Svelte re-adds these per route via <svelte:head>. Removing the prerendered
// originals first avoids shipping two canonical/description tags at once.
for (const tag of document.head.querySelectorAll('[data-prerendered]')) {
  tag.remove();
}

const app = mount(App, { target });

export default app;