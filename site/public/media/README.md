# Tidefetch site media

The site renders polished product mockups until each asset is ready. Add the
files below, then change the matching `enabled` value in
`site/src/lib/media.ts` to `true`.

| File | Recommended export | Content |
| --- | --- | --- |
| `hero-demo.webm` | 1920×1080, VP9/AV1, muted, ≤8 MB | 20–35 second dashboard-to-detail product loop |
| `hero-demo.mp4` | 1920×1080, H.264, muted, ≤12 MB | Safari/fallback version of the same loop |
| `hero-poster.webp` | 2400×1350, quality 82 | Strong frame from the demo video |
| `web-dashboard.webp` | 2400×1500, quality 82 | Queue with active, paused, and completed tasks |
| `terminal-overview.webp` | 2400×1500, quality 82 | Wide TUI with sidebar, disk, chart, and queue |
| `task-detail.webp` | 1800×1350, quality 82 | Files/peers/piece-map detail view |
| `mobile-dashboard.webp` | 1170×2532, quality 82 | Mobile queue and task controls |
| `og-card.webp` | 1200×630, quality 86 | Social sharing card; update `site/index.html` from `.svg` to `.webp` |

Capture screenshots at 2× scale with realistic filenames and mixed task
states. Avoid exposing RPC secrets, private tracker URLs, usernames, or local
filesystem paths. Keep the hero video silent because it autoplays muted.

Useful video conversion commands:

```sh
ffmpeg -i capture.mov -an -vf "scale=1920:-2" -c:v libvpx-vp9 -crf 34 -b:v 0 hero-demo.webm
ffmpeg -i capture.mov -an -vf "scale=1920:-2" -c:v libx264 -crf 23 -movflags +faststart hero-demo.mp4
```