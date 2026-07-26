<script lang="ts">
  let { points, color = '#34d8c3', height = 34 }: { points: number[]; color?: string; height?: number } = $props();

  let canvas: HTMLCanvasElement;

  $effect(() => {
    const pts = points;
    if (!canvas) return;
    const dpr = window.devicePixelRatio || 1;
    const w = canvas.clientWidth;
    const h = height;
    canvas.width = w * dpr;
    canvas.height = h * dpr;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    ctx.scale(dpr, dpr);
    ctx.clearRect(0, 0, w, h);
    if (pts.length < 2) return;

    const max = Math.max(...pts, 1);
    const step = w / (pts.length - 1);

    ctx.beginPath();
    pts.forEach((p, i) => {
      const x = i * step;
      const y = h - 2 - (p / max) * (h - 6);
      i === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y);
    });
    ctx.strokeStyle = color;
    ctx.lineWidth = 1.6;
    ctx.lineJoin = 'round';
    ctx.stroke();

    // area fill
    ctx.lineTo(w, h);
    ctx.lineTo(0, h);
    ctx.closePath();
    const grad = ctx.createLinearGradient(0, 0, 0, h);
    grad.addColorStop(0, color + '3d');
    grad.addColorStop(1, color + '00');
    ctx.fillStyle = grad;
    ctx.fill();
  });
</script>

<canvas bind:this={canvas} style="width:100%;height:{height}px"></canvas>
