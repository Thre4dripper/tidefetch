<script lang="ts">
  // Animated tide mark: a traveling wave inside a rounded badge, with a
  // fetch (download) arrow riding above the waterline.
  let {
    size = 26,
    animated = true,
    id = 'tf'
  }: { size?: number; animated?: boolean; id?: string } = $props();

  // Unique gradient/clip ids so multiple marks can coexist on one page.
  const gid = $derived(`${id}-grad`);
  const cid = $derived(`${id}-clip`);
</script>

<svg
  class="tidemark"
  class:animated
  width={size}
  height={size}
  viewBox="0 0 32 32"
  fill="none"
  role="img"
  aria-label="Tidefetch"
>
  <defs>
    <linearGradient id={gid} x1="0" y1="4" x2="30" y2="30" gradientUnits="userSpaceOnUse">
      <stop offset="0%" stop-color="#5ed8e7" />
      <stop offset="55%" stop-color="#7ee9c4" />
      <stop offset="100%" stop-color="#b8ff3d" />
    </linearGradient>
    <clipPath id={cid}>
      <rect width="32" height="32" rx="9.5" />
    </clipPath>
  </defs>

  <rect width="32" height="32" rx="9.5" fill="#0d1014" />
  <rect x="0.6" y="0.6" width="30.8" height="30.8" rx="9" stroke={`url(#${gid})`} stroke-width="1.2" opacity="0.55" />

  <g clip-path={`url(#${cid})`}>
    <!-- back swell -->
    <path
      class="wave back"
      d="M-16 21 q4 -3.4 8 0 t8 0 t8 0 t8 0 t8 0 t8 0 t8 0 V34 H-16 Z"
      fill={`url(#${gid})`}
      opacity="0.32"
    />
    <!-- front swell -->
    <path
      class="wave front"
      d="M-16 23 q4 -3.4 8 0 t8 0 t8 0 t8 0 t8 0 t8 0 t8 0 V34 H-16 Z"
      fill={`url(#${gid})`}
      opacity="0.92"
    />
  </g>

  <!-- fetch arrow -->
  <path
    d="M16 7.5 V17"
    stroke="#eef3ef"
    stroke-width="2.1"
    stroke-linecap="round"
  />
  <path
    d="M11.9 13.1 16 17.3 20.1 13.1"
    stroke="#eef3ef"
    stroke-width="2.1"
    stroke-linecap="round"
    stroke-linejoin="round"
    fill="none"
  />
</svg>

<style>
  .tidemark { display: block; flex: 0 0 auto; }
  .wave { transform: translateX(0); }

  .tidemark.animated .front { animation: tide-front 3.1s linear infinite; }
  .tidemark.animated .back { animation: tide-back 4.7s linear infinite; }

  /* One period of the wave is 16 user units — translating by 16 loops seamlessly. */
  @keyframes tide-front {
    from { transform: translateX(0); }
    to { transform: translateX(16px); }
  }
  @keyframes tide-back {
    from { transform: translateX(16px); }
    to { transform: translateX(0); }
  }

  @media (prefers-reduced-motion: reduce) {
    .tidemark.animated .front,
    .tidemark.animated .back { animation: none; }
  }
</style>
