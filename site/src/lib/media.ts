export type ImageMedia = {
  enabled: boolean;
  src: string;
  alt: string;
  label: string;
  dimensions: string;
  tone: 'terminal' | 'dashboard' | 'detail' | 'mobile';
};

// Both captures are real screenshots. Terminal output must never be recreated
// in HTML — block and braille glyphs drift on font metrics. See
// public/media/README.md for how to retake them.
export const media = {
  hero: {
    enabled: false,
    mp4: './media/hero-demo.mp4',
    webm: './media/hero-demo.webm',
    poster: './media/hero-poster.webp'
  },
  // Hero: wide overview with the sidebar visible.
  terminal: {
    enabled: true,
    src: './media/terminal-overview.jpg',
    alt: 'Tidefetch terminal UI with four active downloads, a completed download, speed graphs, piece map and disk usage',
    label: 'Terminal interface',
    dimensions: '1715 × 1008 JPEG · 211×50 cells',
    tone: 'terminal'
  } satisfies ImageMedia,
  // Interfaces section: a different view so the page doesn't repeat the hero.
  terminalAlt: {
    enabled: false,
    src: './media/terminal-detail.jpg',
    alt: 'Tidefetch download details with file list, piece map and per-task options',
    label: 'Download details',
    dimensions: '1695 × 895 JPEG · details view',
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
