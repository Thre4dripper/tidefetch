// Where the site is published. Asset paths, the client router, canonical and
// social-card tags, sitemap.xml and robots.txt all derive from this, so it is
// resolved once here for both vite.config.ts and scripts/prerender.mjs.
//
// Production is the Vercel deployment at tidefetch.ijlalahmad.dev. Another
// host can override the address with SITE_URL, and SITE_BASE when the path
// prefix must differ from the URL's own path (for example a project page
// served under /tidefetch/).

const PRODUCTION_URL = 'https://tidefetch.ijlalahmad.dev/';

/**
 * @param {NodeJS.ProcessEnv} [env]
 * @returns {{ url: string, base: string }} the absolute site URL with a
 *   trailing slash, and the path prefix Vite builds against
 */
export function resolveSite(env = process.env) {
  let url = env.SITE_URL || PRODUCTION_URL;
  if (!url.includes('://')) url = `https://${url}`;
  url = withTrailingSlash(url);

  const base = withTrailingSlash(withLeadingSlash(env.SITE_BASE || new URL(url).pathname));
  return { url, base };
}

/** @param {string} s */
function withTrailingSlash(s) {
  return s.endsWith('/') ? s : `${s}/`;
}

/** @param {string} s */
function withLeadingSlash(s) {
  return s.startsWith('/') ? s : `/${s}`;
}
