export type ImageMedia = {
  enabled: boolean;
  src: string;
  alt: string;
  label: string;
  dimensions: string;
  tone: 'terminal' | 'dashboard' | 'detail' | 'mobile';
};

export const media = {
  hero: {
    enabled: false,
    mp4: '/media/hero-demo.mp4',
    webm: '/media/hero-demo.webm',
    poster: '/media/hero-poster.webp'
  },
  web: {
    enabled: false,
    src: '/media/web-dashboard.webp',
    alt: 'Tidefetch web dashboard showing active downloads and live transfer speeds',
    label: 'Web dashboard',
    dimensions: '2400 × 1500 WebP',
    tone: 'dashboard'
  } satisfies ImageMedia,
  terminal: {
    enabled: false,
    src: '/media/terminal-overview.webp',
    alt: 'Tidefetch terminal interface showing the download queue and telemetry sidebar',
    label: 'Terminal interface',
    dimensions: '2400 × 1500 WebP',
    tone: 'terminal'
  } satisfies ImageMedia,
  detail: {
    enabled: false,
    src: '/media/task-detail.webp',
    alt: 'Tidefetch task detail view with speed history, peers, files, and piece map',
    label: 'Task intelligence',
    dimensions: '1800 × 1350 WebP',
    tone: 'detail'
  } satisfies ImageMedia,
  mobile: {
    enabled: false,
    src: '/media/mobile-dashboard.webp',
    alt: 'Tidefetch web interface on a mobile phone',
    label: 'Mobile dashboard',
    dimensions: '1170 × 2532 WebP',
    tone: 'mobile'
  } satisfies ImageMedia
};