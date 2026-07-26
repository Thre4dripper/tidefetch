<script lang="ts">
  let { pieces }: { pieces: number[] } = $props();

  let canvas: HTMLCanvasElement;

  $effect(() => {
    const data = pieces;
    if (!canvas || !data?.length) return;
    const dpr = window.devicePixelRatio || 1;
    const w = canvas.clientWidth;
    const cell = 10;
    const gap = 2;
    const cols = Math.max(1, Math.floor((w + gap) / (cell + gap)));
    const rows = Math.ceil(data.length / cols);
    const h = rows * (cell + gap) - gap;

    canvas.width = w * dpr;
    canvas.height = h * dpr;
    canvas.style.height = `${h}px`;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    ctx.scale(dpr, dpr);
    ctx.clearRect(0, 0, w, h);

    data.forEach((level, i) => {
      const x = (i % cols) * (cell + gap);
      const y = Math.floor(i / cols) * (cell + gap);
      if (level <= 0) {
        ctx.fillStyle = 'rgba(255,255,255,0.07)';
      } else {
        const a = 0.25 + (level / 8) * 0.75;
        ctx.fillStyle = `rgba(52, 216, 195, ${a.toFixed(2)})`;
      }
      ctx.beginPath();
      ctx.roundRect(x, y, cell, cell, 2.5);
      ctx.fill();
    });
  });
</script>

<canvas bind:this={canvas} style="width:100%"></canvas>
