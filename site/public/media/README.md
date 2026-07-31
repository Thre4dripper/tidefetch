# Site media

Assets used by the product site. Each slot renders a labelled placeholder until
the file exists **and** its `enabled` flag is set to `true` in
[`src/lib/media.ts`](../../src/lib/media.ts).

| File | Slot | Status |
| --- | --- | --- |
| `terminal-overview.png` | Hero + Interfaces (TUI) | **Needed** |
| `web-dashboard.png` | Interfaces (web UI) | Present |
| `hero-demo.webm` / `.mp4` | Hero video (optional, replaces the hero image) | Optional |
| `task-detail.png` | Transfer detail | Optional |
| `mobile-dashboard.png` | Mobile view | Optional |

## Social card: og-card.svg and og-card.png

These are **not duplicates**. `og-card.svg` is the editable source; edit only
that one. `og-card.png` is a raster export of it, and it has to exist because
Facebook, LinkedIn, Slack and X all reject SVG for `og:image` — an SVG-only
card renders as no card at all.

After editing the SVG, re-export the PNG at exactly 1200×630. Any of these work:

```sh
# rsvg-convert (brew install librsvg)
rsvg-convert -w 1200 -h 630 og-card.svg -o og-card.png

# ImageMagick
magick -background none -density 144 og-card.svg -resize 1200x630 og-card.png
```

Or open the SVG in a browser sized to 1200×630 and screenshot it — that route
keeps the webfonts, which command-line rasterizers substitute unless the font
files are installed locally.

The dimensions are declared in `index.html` (`og:image:width` / `height`), so
keep them in sync if you ever change the aspect ratio.

## Capturing the terminal UI

Rendering terminal output to HTML drifts on font metrics — block and braille
glyphs end up misaligned. Take a real screenshot instead.

Set up a representative state first: a few active downloads at different
speeds, one complete, one paused, with the side panel visible (`t`).

**Option A — screenshot a real terminal (most faithful)**

1. Size the window to about `172×40` cells.
2. Use a font with good block and braille coverage: JetBrains Mono, Iosevka,
   Fira Code or SF Mono.
3. Run `tidefetch`, then capture the window:
   - macOS: `Cmd+Shift+4`, then `Space`, then click the window
   - Linux: `gnome-screenshot -w` or `spectacle -a`
   - Windows: `Win+Shift+S`

**Option B — [freeze](https://github.com/charmbracelet/freeze)**

```sh
brew install charmbracelet/tap/freeze
freeze --execute "tidefetch" --window --border.radius 8 \
  --output terminal-overview.png
```

**Option C — [asciinema](https://asciinema.org) + agg** for an animated hero

```sh
asciinema rec demo.cast --command tidefetch
agg demo.cast demo.gif --font-family "JetBrains Mono" --theme dracula
```

Then convert to video for the hero slot:

```sh
ffmpeg -i demo.gif -an -c:v libvpx-vp9 -crf 34 -b:v 0 hero-demo.webm
ffmpeg -i demo.gif -an -c:v libx264 -crf 23 -movflags +faststart hero-demo.mp4
```

## Capturing the web UI

Run `tidefetch serve`, open the dashboard at 1600×1000, select a task so the
detail panel is visible, then screenshot the page.

## Guidelines

- Export at 2× where possible; the site downscales.
- Keep PNG under ~400 KB (`pngquant --quality 65-85` or `sips`).
- Scrub anything private: RPC secrets, private tracker URLs, usernames, real
  file paths and internal hostnames.
- The hero video autoplays muted — no audio track needed.
- The TUI theme in the screenshot should be **Surge** (the default) unless the
  shot is specifically demonstrating theming.

After adding a file, set its `enabled` flag in `src/lib/media.ts` and rebuild
with `make site`.
