// History-based routing. Every route resolves to a real URL such as
// /tidefetch/docs/installation, which is what search engines index — URL
// fragments (#/docs/...) are never sent to the server and are ignored by
// crawlers, so the whole documentation set would collapse into one page.

export const BASE = import.meta.env.BASE_URL;

export type Route =
  | { view: 'landing'; anchor: string }
  | { view: 'docs'; slug: string; anchor: string };

/** Build an absolute, base-prefixed href for an internal path. */
export function href(path: string): string {
  return `${BASE}${path.replace(/^\/+/, '')}`;
}

export function parse(loc: Location | URL = window.location): Route {
  let path = loc.pathname;
  if (path.startsWith(BASE)) path = path.slice(BASE.length);
  path = path.replace(/^\/+|\/+$/g, '');

  const anchor = loc.hash.replace(/^#/, '');

  if (path === 'docs' || path.startsWith('docs/')) {
    const slug = path.slice('docs'.length).replace(/^\/+/, '');
    return { view: 'docs', slug: slug || 'getting-started', anchor };
  }
  return { view: 'landing', anchor };
}

/** Push a new URL and notify listeners without a full page load. */
export function navigate(url: string): void {
  history.pushState({}, '', url);
  window.dispatchEvent(new PopStateEvent('popstate'));
}

/**
 * Intercept same-origin link clicks so internal navigation stays client-side.
 * Returns a teardown function.
 */
export function interceptLinks(): () => void {
  const onClick = (event: MouseEvent) => {
    if (
      event.defaultPrevented ||
      event.button !== 0 ||
      event.metaKey ||
      event.ctrlKey ||
      event.shiftKey ||
      event.altKey
    ) {
      return;
    }

    const anchor = (event.target as HTMLElement | null)?.closest('a');
    if (!anchor) return;

    const target = anchor.getAttribute('target');
    if (target && target !== '_self') return;
    if (anchor.hasAttribute('download')) return;

    const url = new URL(anchor.href, window.location.href);
    if (url.origin !== window.location.origin) return;
    if (!url.pathname.startsWith(BASE)) return;

    // Let the browser handle real files such as /install.sh.
    if (/\.[a-z0-9]+$/i.test(url.pathname)) return;

    event.preventDefault();
    if (url.href === window.location.href) return;
    navigate(url.href);
  };

  document.addEventListener('click', onClick);
  return () => document.removeEventListener('click', onClick);
}
