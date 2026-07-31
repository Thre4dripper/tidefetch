export type ImageMedia = {
  enabled: boolean;
  src: string;
  alt: string;
  label: string;
  dimensions: string;
  tone: 'terminal' | 'dashboard' | 'detail' | 'mobile';
};

// The web capture is a real browser screenshot. The terminal capture is left
// as a placeholder: rendering a TUI to HTML drifts on font metrics, so it
// should be a genuine screenshot. See public/media/README.md, then flip
// `enabled` to true.
export const media = {
  hero: {
    enabled: false,
    mp4: './media/hero-demo.mp4',
    webm: './media/hero-demo.webm',
    poster: './media/hero-poster.webp'
  },
  terminal: {
    enabled: false,
    src: './media/terminal-overview.png',
    alt: 'Tidefetch terminal UI showing active downloads, speed graphs, piece map and disk usage',
    label: 'Terminal interface',
    dimensions: '2400 × 1400 PNG · 172×40 cells',
    tone: 'terminal'
  } satisfies ImageMedia,
  web: {
    enabled: true,
    src: './media/web-dashboard.png',
    alt: 'Tidefetch web dashboard showing active downloads with speed, status badges, throughput chart and the transfer detail panel',
    label: 'Web dashboard',
    dimensions: '1600 × 1000 PNG',
    tone: 'dashboard'
  } satisfies ImageMedia,
  detail: {
    enabled: false,
    src: './media/task-detail.png',
    alt: 'Tidefetch transfer detail view with progress ring, speed history, piece map and per-task options',
    label: 'Transfer detail',
    dimensions: '1800 × 1350 PNG',
    tone: 'detail'
  } satisfies ImageMedia,
  mobile: {
    enabled: false,
    src: './media/mobile-dashboard.png',
    alt: 'Tidefetch web interface on a phone',
    label: 'Mobile dashboard',
    dimensions: '1170 × 2532 PNG',
    tone: 'mobile'
  } satisfies ImageMedia
};
