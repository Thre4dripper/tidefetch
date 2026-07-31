<script lang="ts">
  let { points, color = '#5ed8e7', height = 34 }: { points: number[]; color?: string; height?: number } = $props();

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

    // Headroom keeps a steady series off the ceiling so it reads as a chart.
    const max = Math.max(...pts, 1) * 1.18;
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

    // Area fill. The stroke colour may be hex or rgb()/rgba(), so build the
    // gradient stops through canvas itself rather than string concatenation.
    ctx.lineTo(w, h);
    ctx.lineTo(0, h);
    ctx.closePath();
    const grad = ctx.createLinearGradient(0, 0, 0, h);
    grad.addColorStop(0, withAlpha(color, 0.14));
    grad.addColorStop(1, withAlpha(color, 0));
    ctx.fillStyle = grad;
    ctx.fill();
  });

  /** Return `color` at the given alpha, for hex, rgb() and rgba() inputs. */
  function withAlpha(input: string, alpha: number): string {
    const hex = input.trim().match(/^#([0-9a-f]{6})$/i);
    if (hex) {
      const n = parseInt(hex[1], 16);
      return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`;
    }
    const rgb = input.trim().match(/^rgba?\(\s*([\d.]+)[\s,]+([\d.]+)[\s,]+([\d.]+)/i);
    if (rgb) return `rgba(${rgb[1]}, ${rgb[2]}, ${rgb[3]}, ${alpha})`;
    return input;
  }
</script>

<canvas bind:this={canvas} style="width:100%;height:{height}px"></canvas>
