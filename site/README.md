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

The static output is written to `site/dist/` and can be hosted on GitHub Pages,
Cloudflare Pages, Netlify, an object store, or any static web server.

## Add screenshots and video

See [`public/media/README.md`](public/media/README.md) for the expected names,
sizes, privacy checklist, and ffmpeg commands. After adding an asset, set its
`enabled` value in [`src/lib/media.ts`](src/lib/media.ts) to `true`.

Until then, the site renders purpose-built product mockups and clearly labeled
capture slots; it never requests missing files.