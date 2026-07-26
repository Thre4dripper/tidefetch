// Formatting helpers shared across components.

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];

export function fmtBytes(n: number): string {
  if (!n || n < 0) return '0 B';
  let i = 0;
  let v = n;
  while (v >= 1024 && i < UNITS.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v >= 100 ? v.toFixed(0) : v >= 10 ? v.toFixed(1) : v.toFixed(2)} ${UNITS[i]}`;
}

export function fmtSpeed(n: number): string {
  return n > 0 ? `${fmtBytes(n)}/s` : '—';
}

export function fmtEta(task: { total: number; done: number; downSpeed: number }): string {
  if (task.downSpeed <= 0 || task.total <= task.done) return '';
  const s = Math.round((task.total - task.done) / task.downSpeed);
  if (s < 60) return `${s}s`;
  if (s < 3600) return `${Math.floor(s / 60)}m ${s % 60}s`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ${Math.floor((s % 3600) / 60)}m`;
  return `${Math.floor(s / 86400)}d ${Math.floor((s % 86400) / 3600)}h`;
}

export function fmtPct(p: number): string {
  return `${(p * 100).toFixed(p >= 0.995 ? 0 : 1)}%`;
}

export function fmtDate(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '';
  return d.toLocaleString(undefined, {
    month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
  });
}

export function basename(p: string): string {
  const i = Math.max(p.lastIndexOf('/'), p.lastIndexOf('\\'));
  return i >= 0 ? p.slice(i + 1) : p;
}
