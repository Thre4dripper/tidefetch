# Tidefetch product site

This is the standalone showcase site. It is intentionally separate from
`web/`, which is the operational dashboard embedded in the Go binary.

## Develop

```sh
make site-dev
# or: npm --prefix site run dev
```

## Validate and build

```sh
make site
```

The static output is written to `site/dist/`. It is plain HTML, CSS and JS,
so any static host can serve it.

## Where the site is published

Vercel deploys the site from the `tidefetch` project, which points at this
directory (root directory `site`, Vite preset). Every push to `main` redeploys
<https://tidefetch.ijlalahmad.dev/> and pull requests get preview URLs. The
install scripts in `scripts/` are copied to the site root, so the documented
one-liners resolve there too.

The build bakes in its publish address: asset paths, the client router, the
canonical and social-card tags, `sitemap.xml` and `robots.txt` all derive from
it. `site.config.mjs` resolves that address once for both Vite and the
prerender step. It defaults to the production domain and can be overridden
for another host:

| Variable | Purpose |
| --- | --- |
| `SITE_URL` | Absolute publish URL, e.g. `SITE_URL=https://tidefetch.example.com/` |
| `SITE_BASE` | Path prefix when it must differ from the URL's path, e.g. `/tidefetch/` for a project page |

Deep links such as `/docs/installation` are real files after prerendering, so
no rewrite rules are needed. Unknown paths fall back to `404.html`, which
Vercel serves automatically.

## Add screenshots and video

See [`public/media/README.md`](public/media/README.md) for the expected names,
sizes, privacy checklist, and ffmpeg commands. After adding an asset, set its
`enabled` value in [`src/lib/media.ts`](src/lib/media.ts) to `true`.

Until then, the site renders purpose-built product mockups and clearly labeled
capture slots; it never requests missing files.
